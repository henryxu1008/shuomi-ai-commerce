package toolmarketpersistence

import (
	"context"
	"crypto/sha256"
	_ "embed"
	"encoding/hex"
	"encoding/json"
	"errors"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"math"
	"strconv"
	tm "task-processor/internal/toolmarket"
	"time"
)

//go:embed schema.sql
var schemaSQL string

type Store struct{ db *gorm.DB }

func New(db *gorm.DB) (*Store, error) {
	if db == nil || db.Dialector.Name() != "postgres" {
		return nil, tm.ErrUnavailable
	}
	return &Store{db}, nil
}

// InstallSchema is explicit fresh installation with schema-owner credentials.
// Serving startup never calls it, and existing schemas are not migrated.
func InstallSchema(db *gorm.DB) error {
	if db == nil || db.Dialector.Name() != "postgres" {
		return tm.ErrUnavailable
	}
	return db.Transaction(func(tx *gorm.DB) error {
		if e := tx.Exec(`DO $$ BEGIN IF NOT EXISTS(SELECT 1 FROM pg_roles WHERE rolname='tool_market_owner') THEN CREATE ROLE tool_market_owner NOLOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE NOBYPASSRLS; END IF; END $$;`).Error; e != nil {
			return e
		}
		var unsafe bool
		if e := tx.Raw(`SELECT rolcanlogin OR rolsuper OR rolcreaterole OR rolcreatedb OR rolbypassrls FROM pg_roles WHERE rolname='tool_market_owner'`).Scan(&unsafe).Error; e != nil {
			return e
		}
		if unsafe {
			return tm.ErrInvalid
		}
		if e := tx.Exec(`CREATE SCHEMA tool_market AUTHORIZATION tool_market_owner`).Error; e != nil {
			return e
		}
		if e := tx.Exec("SET LOCAL ROLE tool_market_owner").Error; e != nil {
			return e
		}
		if e := tx.Exec(schemaSQL).Error; e != nil {
			return e
		}
		return tx.Exec("RESET ROLE").Error
	})
}
func VerifySchema(ctx context.Context, db *gorm.DB) error {
	if db == nil {
		return tm.ErrUnavailable
	}
	for _, q := range []string{"SELECT organization_id,tool_id,enabled,revision,updated_by,updated_at FROM tool_market.activations LIMIT 0", "SELECT id,organization_id,created_by,kind,title,description,scenario,contact_name,contact_method,consent,consent_version,stage,revision,created_at,updated_at FROM tool_market.requests LIMIT 0", "SELECT id,request_id,name,content_type,sha256,size,data FROM tool_market.attachments LIMIT 0", "SELECT request_id,revision,stage,note,actor_id,occurred_at FROM tool_market.events LIMIT 0", "SELECT organization_id,actor_id,idempotency_key,fingerprint,receipt FROM tool_market.commands LIMIT 0"} {
		if e := db.WithContext(ctx).Exec(q).Error; e != nil {
			return tm.ErrUnavailable
		}
	}
	return nil
}

type activationRow struct {
	OrganizationID, ToolID string
	Enabled                bool
	Revision               int64
	UpdatedBy              string
	UpdatedAt              time.Time
}

func (activationRow) TableName() string { return "tool_market.activations" }

type requestRow struct {
	ID, OrganizationID, CreatedBy, Kind, Title, Description, Stage string
	Revision                                                       int64
	CreatedAt, UpdatedAt                                           time.Time
	Scenario, ContactName, ContactMethod, ConsentVersion           string
	Consent                                                        bool
}

func (requestRow) TableName() string { return "tool_market.requests" }
func (r requestRow) view() tm.Request {
	return tm.Request{ID: r.ID, OrganizationID: r.OrganizationID, Demand: tm.Demand{Kind: r.Kind, Title: r.Title, Description: r.Description, Scenario: r.Scenario, ContactName: r.ContactName, ContactMethod: r.ContactMethod, Consent: r.Consent}, CreatedBy: r.CreatedBy, ConsentVersion: r.ConsentVersion, Attachments: []tm.Attachment{}, Stage: r.Stage, Revision: strconv.FormatInt(r.Revision, 10), CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt}
}

type attachmentRow struct {
	ID, RequestID, Name, ContentType, SHA256 string
	Size                                     int
	Data                                     []byte
}

func (attachmentRow) TableName() string { return "tool_market.attachments" }
func (r attachmentRow) view() tm.Attachment {
	return tm.Attachment{ID: r.ID, Name: r.Name, ContentType: r.ContentType, Size: r.Size, SHA256: r.SHA256}
}

func (s *Store) Download(ctx context.Context, scope tm.Scope, platform bool, id, file string) (tm.Attachment, []byte, error) {
	if !scope.Valid(platform) {
		return tm.Attachment{}, nil, tm.ErrForbidden
	}
	if !tm.UUID(id) || !tm.UUID(file) {
		return tm.Attachment{}, nil, tm.ErrInvalid
	}
	q := s.db.WithContext(ctx).Table("tool_market.attachments AS a").Select("a.*").Joins("JOIN tool_market.requests AS r ON r.id=a.request_id").Where("a.id=? AND a.request_id=?", file, id)
	if !platform {
		q = q.Where("r.organization_id=?", scope.OrganizationID)
	}
	var row attachmentRow
	if err := q.Take(&row).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return tm.Attachment{}, nil, tm.ErrNotFound
		}
		return tm.Attachment{}, nil, err
	}
	hash := sha256.Sum256(row.Data)
	typ, err := tm.FileType(row.Data)
	if err != nil || len(row.Data) != row.Size || hex.EncodeToString(hash[:]) != row.SHA256 || typ != row.ContentType {
		return tm.Attachment{}, nil, tm.ErrUnavailable
	}
	return row.view(), row.Data, nil
}

type eventRow struct {
	RequestID            string
	Revision             int64
	Stage, Note, ActorID string
	OccurredAt           time.Time
}

func (eventRow) TableName() string { return "tool_market.events" }

type commandRow struct {
	OrganizationID, ActorID, IdempotencyKey, Fingerprint string
	Receipt                                              []byte `gorm:"type:jsonb"`
}

func (commandRow) TableName() string { return "tool_market.commands" }
func (s *Store) Activations(ctx context.Context, scope tm.Scope) ([]tm.Activation, error) {
	if !scope.Valid(false) {
		return nil, tm.ErrForbidden
	}
	rows := []activationRow{}
	e := s.db.WithContext(ctx).Where("organization_id=?", scope.OrganizationID).Order("tool_id").Limit(50).Find(&rows).Error
	result := []tm.Activation{}
	for _, r := range rows {
		result = append(result, tm.Activation{ToolID: r.ToolID, Enabled: r.Enabled, Revision: strconv.FormatInt(r.Revision, 10), UpdatedAt: r.UpdatedAt})
	}
	return result, e
}
func (s *Store) Requests(ctx context.Context, scope tm.Scope, platform bool, cursor string, limit int) (tm.RequestPage, error) {
	result := tm.RequestPage{Items: []tm.RequestSummary{}}
	if !scope.Valid(platform) {
		return result, tm.ErrForbidden
	}
	if limit < 1 || limit > 50 || cursor != "" && !tm.UUID(cursor) {
		return result, tm.ErrInvalid
	}
	q := s.db.WithContext(ctx)
	if !platform {
		q = q.Where("organization_id=?", scope.OrganizationID)
	}
	if cursor != "" {
		q = q.Where("id>?", cursor)
	}
	rows := []requestRow{}
	if e := q.Order("id").Limit(limit + 1).Find(&rows).Error; e != nil {
		return result, e
	}
	if len(rows) > limit {
		rows = rows[:limit]
		result.NextCursor = rows[len(rows)-1].ID
	}
	for _, r := range rows {
		result.Items = append(result.Items, tm.RequestSummary{ID: r.ID, OrganizationID: r.OrganizationID, Kind: r.Kind, Title: r.Title, Stage: r.Stage, Revision: strconv.FormatInt(r.Revision, 10), CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt})
	}
	return result, nil
}
func (s *Store) Detail(ctx context.Context, scope tm.Scope, platform bool, id, eventsBefore string) (tm.Detail, error) {
	result := tm.Detail{Events: []tm.Event{}}
	if !scope.Valid(platform) {
		return result, tm.ErrForbidden
	}
	if !tm.UUID(id) {
		return result, tm.ErrInvalid
	}
	var before int64
	if eventsBefore != "" {
		var err error
		before, err = strconv.ParseInt(eventsBefore, 10, 64)
		if err != nil || before <= 0 || strconv.FormatInt(before, 10) != eventsBefore {
			return result, tm.ErrInvalid
		}
	}
	q := s.db.WithContext(ctx).Where("id=?", id)
	if !platform {
		q = q.Where("organization_id=?", scope.OrganizationID)
	}
	var r requestRow
	if e := q.Take(&r).Error; e != nil {
		if errors.Is(e, gorm.ErrRecordNotFound) {
			return result, tm.ErrNotFound
		}
		return result, e
	}
	result.Request = r.view()
	files := []attachmentRow{}
	if e := s.db.WithContext(ctx).Select("id,request_id,name,content_type,sha256,size").Where("request_id=?", id).Order("id").Limit(3).Find(&files).Error; e != nil {
		return result, e
	}
	for _, f := range files {
		result.Request.Attachments = append(result.Request.Attachments, f.view())
	}
	rows := []eventRow{}
	events := s.db.WithContext(ctx).Where("request_id=? AND revision<=?", id, r.Revision)
	if eventsBefore != "" {
		events = events.Where("revision<?", before)
	}
	if e := events.Order("revision DESC").Limit(tm.EventPageSize + 1).Find(&rows).Error; e != nil {
		return result, e
	}
	if len(rows) > tm.EventPageSize {
		rows = rows[:tm.EventPageSize]
		result.NextEventsBefore = strconv.FormatInt(rows[len(rows)-1].Revision, 10)
	}
	// Each page displays chronologically; the initial page contains latest
	// progress and an explicit cursor for all earlier immutable events.
	for i := len(rows) - 1; i >= 0; i-- {
		r := rows[i]
		result.Events = append(result.Events, tm.Event{Revision: strconv.FormatInt(r.Revision, 10), Stage: r.Stage, Note: r.Note, OccurredAt: r.OccurredAt})
	}
	return result, nil
}
func (s *Store) Execute(ctx context.Context, c tm.Command, guard func(context.Context) error, beforeApply ...func(context.Context) error) (tm.Receipt, error) {
	var result tm.Receipt
	if !c.Valid() {
		return result, tm.ErrInvalid
	}
	if guard == nil {
		return result, tm.ErrForbidden
	}
	if len(beforeApply) > 1 || len(beforeApply) == 1 && beforeApply[0] == nil {
		return result, tm.ErrInvalid
	}
	raw, _ := json.Marshal(c)
	digest := sha256.Sum256(raw)
	fingerprint := hex.EncodeToString(digest[:])
	e := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if e := guard(ctx); e != nil {
			return e
		}
		org := c.Scope.OrganizationID
		var demand requestRow
		if c.Platform {
			if e := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id=?", c.ID).Take(&demand).Error; e != nil {
				if errors.Is(e, gorm.ErrRecordNotFound) {
					return tm.ErrNotFound
				}
				return e
			}
			org = demand.OrganizationID
		}
		row := commandRow{OrganizationID: org, ActorID: c.Scope.ActorID, IdempotencyKey: c.Key, Fingerprint: fingerprint}
		if e := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&row).Error; e != nil {
			return e
		}
		var saved commandRow
		if e := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("organization_id=? AND actor_id=? AND idempotency_key=?", org, c.Scope.ActorID, c.Key).Take(&saved).Error; e != nil {
			return e
		}
		if saved.Fingerprint != fingerprint {
			return tm.ErrConflict
		}
		if len(saved.Receipt) > 0 {
			if e := json.Unmarshal(saved.Receipt, &result); e != nil {
				return tm.ErrUnavailable
			}
			return guard(ctx)
		}
		if len(beforeApply) == 1 {
			if e := beforeApply[0](ctx); e != nil {
				return e
			}
		}
		now := time.Now().UTC().Truncate(time.Microsecond)
		result = tm.Receipt{CommandID: c.Key, Operation: c.Operation, ID: c.ID, CommittedAt: now}
		switch c.Operation {
		case "activation":
			if c.Absent {
				a := activationRow{OrganizationID: org, ToolID: c.ID, Enabled: c.Enabled, Revision: 1, UpdatedBy: c.Scope.ActorID, UpdatedAt: now}
				res := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&a)
				if res.Error != nil {
					return res.Error
				}
				if res.RowsAffected != 1 {
					return tm.ErrRevision
				}
				result.Revision = "1"
			} else {
				var a activationRow
				if e := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("organization_id=? AND tool_id=?", org, c.ID).Take(&a).Error; e != nil {
					if errors.Is(e, gorm.ErrRecordNotFound) {
						return tm.ErrRevision
					}
					return e
				}
				if a.Revision != c.Expected || a.Revision == math.MaxInt64 {
					return tm.ErrRevision
				}
				if e := tx.Model(&activationRow{}).Where("organization_id=? AND tool_id=? AND revision=?", org, c.ID, c.Expected).Updates(map[string]any{"enabled": c.Enabled, "revision": a.Revision + 1, "updated_by": c.Scope.ActorID, "updated_at": now}).Error; e != nil {
					return e
				}
				result.Revision = strconv.FormatInt(a.Revision+1, 10)
			}
		case "create":
			demand = requestRow{ID: uuid.NewString(), OrganizationID: org, CreatedBy: c.Scope.ActorID, Kind: c.Demand.Kind, Title: c.Demand.Title, Description: c.Demand.Description, Scenario: c.Demand.Scenario, ContactName: c.Demand.ContactName, ContactMethod: c.Demand.ContactMethod, Consent: c.Demand.Consent, ConsentVersion: tm.ConsentVersion, Stage: "SUBMITTED", Revision: 1, CreatedAt: now, UpdatedAt: now}
			if e := tx.Create(&demand).Error; e != nil {
				return e
			}
			for _, f := range c.Demand.Files {
				typ, err := tm.FileType(f.Data)
				if err != nil {
					return err
				}
				hash := sha256.Sum256(f.Data)
				if e := tx.Create(&attachmentRow{ID: uuid.NewString(), RequestID: demand.ID, Name: f.Name, ContentType: typ, SHA256: hex.EncodeToString(hash[:]), Size: len(f.Data), Data: f.Data}).Error; e != nil {
					return e
				}
			}
			result.ID = demand.ID
			result.Revision = "1"
		case "progress":
			if demand.Revision != c.Expected || demand.Revision == math.MaxInt64 {
				return tm.ErrRevision
			}
			if !tm.Transition(demand.Stage, c.Progress.Stage) {
				return tm.ErrInvalid
			}
			demand.Revision++
			demand.Stage = c.Progress.Stage
			demand.UpdatedAt = now
			if e := tx.Model(&requestRow{}).Where("id=? AND organization_id=? AND revision=?", c.ID, org, c.Expected).Updates(map[string]any{"stage": demand.Stage, "revision": demand.Revision, "updated_at": now}).Error; e != nil {
				return e
			}
			result.Revision = strconv.FormatInt(demand.Revision, 10)
		}
		if c.Operation != "activation" {
			note := c.Progress.Note
			if c.Operation == "create" {
				note = "需求已保存，待硕米专员评估"
			}
			if e := tx.Create(&eventRow{RequestID: demand.ID, Revision: demand.Revision, Stage: demand.Stage, Note: note, ActorID: c.Scope.ActorID, OccurredAt: now}).Error; e != nil {
				return e
			}
		}
		if e := guard(ctx); e != nil {
			return e
		}
		receipt, e := json.Marshal(result)
		if e != nil {
			return e
		}
		return tx.Model(&commandRow{}).Where("organization_id=? AND actor_id=? AND idempotency_key=?", org, c.Scope.ActorID, c.Key).Update("receipt", receipt).Error
	})
	return result, e
}
