package storecenter

import (
	"context"
	"encoding/json"
	"errors"
	"strconv"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var ErrBrowserInvalid = errors.New("invalid store browser request")
var ErrBrowserConflict = errors.New("store browser operation conflicts with its original request")
var ErrBrowserDenied = errors.New("store browser permission denied")

// BrowserUseAuthorizer consumes current application authentication/permissions.
// ReceiptSubject deliberately requires identity only, not a current Store grant.
type BrowserUseAuthorizer interface {
	AuthorizeStoreBrowser(context.Context, string) error
	ReceiptSubject(context.Context) (string, error)
}

type BrowserCommand struct {
	SchemaVersion  string `json:"schemaVersion"`
	Action         string `json:"action"`
	BindingID      string `json:"bindingId"`
	ProfileID      string `json:"profileId"`
	BindingVersion string `json:"bindingVersion"`
}

type BrowserReport struct {
	SchemaVersion string       `json:"schemaVersion"`
	Kind          string       `json:"kind"`
	Outcome       AuditOutcome `json:"outcome"`
	ReasonCode    string       `json:"reasonCode"`
	ObservedAt    time.Time    `json:"observedAt"`
}

type BrowserReceipt struct {
	SchemaVersion       string          `json:"schemaVersion"`
	UserID              string          `json:"userId"`
	OrganizationID      string          `json:"organizationId"`
	StoreID             string          `json:"storeId"`
	OperationID         string          `json:"operationId"`
	Action              string          `json:"action"`
	BindingID           string          `json:"bindingId"`
	ProfileID           string          `json:"profileId"`
	BindingVersion      string          `json:"bindingVersion"`
	AdmittedAt          time.Time       `json:"admittedAt"`
	ExecutionWindowMS   int             `json:"executionWindowMs"`
	ExecutionAuthorized bool            `json:"executionAuthorized"`
	Reports             []BrowserReport `json:"reports"`
}

type browserOperation struct {
	OrganizationID string    `gorm:"primaryKey;size:200;not null"`
	OperationID    string    `gorm:"primaryKey;type:char(36);not null"`
	StoreID        string    `gorm:"type:char(36);not null;index"`
	Issuer         string    `gorm:"not null"`
	ActorID        string    `gorm:"size:200;not null"`
	MemberID       string    `gorm:"size:200;not null"`
	Action         string    `gorm:"size:32;not null"`
	BindingID      string    `gorm:"type:char(36);not null"`
	ProfileID      string    `gorm:"type:char(36);not null"`
	BindingVersion string    `gorm:"size:19;not null"`
	Fingerprint    string    `gorm:"size:64;not null"`
	StoreVersion   int64     `gorm:"not null"`
	AdmittedAt     time.Time `gorm:"not null"`
}

func (browserOperation) TableName() string { return "workbench_store_browser_operations" }

type browserReportRecord struct {
	OrganizationID string    `gorm:"primaryKey;size:200;not null"`
	OperationID    string    `gorm:"primaryKey;type:char(36);not null"`
	Kind           string    `gorm:"primaryKey;size:32;not null"`
	Outcome        string    `gorm:"size:32;not null"`
	ReasonCode     string    `gorm:"size:40;not null"`
	ObservedAt     time.Time `gorm:"not null"`
	ReceivedAt     time.Time `gorm:"not null"`
	Fingerprint    string    `gorm:"size:64;not null"`
}

func (browserReportRecord) TableName() string { return "workbench_store_browser_reports" }

// BrowserOperations borrows the existing Store database and authorization ports.
// Construction never installs schema, starts services, or creates identities.
type BrowserOperations struct {
	db     *gorm.DB
	stores StoreMemberAuthorizer
	use    BrowserUseAuthorizer
	issuer string
	now    func() time.Time
}

func NewBrowserOperations(db *gorm.DB, stores StoreMemberAuthorizer, use BrowserUseAuthorizer, issuer string, now func() time.Time) (*BrowserOperations, error) {
	if db == nil || isNilDependency(stores) || isNilDependency(use) || issuer == "" || now == nil {
		return nil, ErrDependencyUnavailable
	}
	return &BrowserOperations{db: db, stores: stores, use: use, issuer: issuer, now: now}, nil
}

func (c BrowserCommand) validate() error {
	if c.SchemaVersion != "store-browser-v1" || (c.Action != "configure_environment" && c.Action != "open_environment") {
		return ErrBrowserInvalid
	}
	for _, id := range []string{c.BindingID, c.ProfileID} {
		if _, err := canonicalUUID(id); err != nil {
			return ErrBrowserInvalid
		}
	}
	v, err := strconv.ParseInt(c.BindingVersion, 10, 64)
	if err != nil || v <= 0 || strconv.FormatInt(v, 10) != c.BindingVersion {
		return ErrBrowserInvalid
	}
	return nil
}
func (r BrowserReport) validate(action string) error {
	if r.SchemaVersion != "store-browser-v1" || r.ObservedAt.IsZero() {
		return ErrBrowserInvalid
	}
	if r.Kind == "configure_result" && action != "configure_environment" || (r.Kind == "open_result" || r.Kind == "close_result") && action != "open_environment" {
		return ErrBrowserInvalid
	}
	var allowed []string
	switch r.Kind {
	case "configure_result":
		if r.Outcome == AuditOutcomeSucceeded {
			allowed = []string{"configuration_saved"}
		} else if r.Outcome == AuditOutcomeFailed {
			allowed = []string{"configuration_failed", "context_changed", "execution_window_expired"}
		}
	case "open_result":
		if r.Outcome == AuditOutcomeSucceeded {
			allowed = []string{"process_started", "already_running"}
		} else if r.Outcome == AuditOutcomeFailed {
			allowed = []string{"launch_failed", "context_changed", "execution_window_expired"}
		}
	case "close_result":
		if r.Outcome == AuditOutcomeSucceeded {
			allowed = []string{"owned_close_confirmed"}
		} else if r.Outcome == AuditOutcomeFailed {
			allowed = []string{"owned_close_failed"}
		}
	default:
		return ErrBrowserInvalid
	}
	if r.Outcome == AuditOutcomeUnknown && r.ReasonCode == "result_unknown" {
		return nil
	}
	for _, s := range allowed {
		if r.ReasonCode == s {
			return nil
		}
	}
	return ErrBrowserInvalid
}
func browserFingerprint(v any) string {
	data, _ := json.Marshal(v)
	return hashTuple("store-browser-v1", string(data))
}
func (o browserOperation) receipt(execute bool) BrowserReceipt {
	return BrowserReceipt{SchemaVersion: "store-browser-v1", UserID: o.ActorID, OrganizationID: o.OrganizationID, StoreID: o.StoreID, OperationID: o.OperationID, Action: o.Action, BindingID: o.BindingID, ProfileID: o.ProfileID, BindingVersion: o.BindingVersion, AdmittedAt: o.AdmittedAt, ExecutionWindowMS: 15000, ExecutionAuthorized: execute, Reports: []BrowserReport{}}
}
func (s *BrowserOperations) Admit(ctx context.Context, org, store, key string, c BrowserCommand) (BrowserReceipt, error) {
	if c.validate() != nil {
		return BrowserReceipt{}, ErrBrowserInvalid
	}
	for _, id := range []string{store, key} {
		if _, err := canonicalUUID(id); err != nil {
			return BrowserReceipt{}, ErrBrowserInvalid
		}
	}
	if err := s.use.AuthorizeStoreBrowser(ctx, org); err != nil {
		return BrowserReceipt{}, err
	}
	access, err := s.stores.AuthorizeStoreMember(ctx, org)
	if err != nil {
		return BrowserReceipt{}, err
	}
	if access.OrganizationID != org {
		return BrowserReceipt{}, ErrNotFound
	}
	var result BrowserReceipt
	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		scoped, err := NewMemberScopedStoreRepository(tx, s.stores)
		if err != nil {
			return err
		}
		if err = scoped.LockRead(ctx, org, []string{store}); err != nil {
			return err
		}
		base, _ := NewGormStoreRepository(tx)
		current, err := base.Get(ctx, org, store)
		if err != nil {
			return err
		}
		if current.RecordStatus() != RecordStatusActive {
			return ErrInvalidTransition
		}
		candidate := browserOperation{OrganizationID: org, OperationID: key, StoreID: store, Issuer: s.issuer, ActorID: access.ActorID, MemberID: access.MemberID, Action: c.Action, BindingID: c.BindingID, ProfileID: c.ProfileID, BindingVersion: c.BindingVersion, Fingerprint: browserFingerprint(c), StoreVersion: current.Version(), AdmittedAt: s.now().UTC()}
		saved := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&candidate)
		if saved.Error != nil {
			return saved.Error
		}
		if saved.RowsAffected == 0 {
			var original browserOperation
			if err = tx.Where("organization_id=? AND operation_id=?", org, key).Take(&original).Error; err != nil {
				return err
			}
			if original.Issuer != s.issuer || original.ActorID != access.ActorID {
				return ErrNotFound
			}
			if original.StoreID != store || original.Fingerprint != candidate.Fingerprint || original.MemberID != access.MemberID {
				return ErrBrowserConflict
			}
			result = original.receipt(false)
			return nil
		}
		if err = recordBrowserAudit(tx, candidate, AuditActionBrowserAdmitted, AuditOutcomeUnknown, candidate.AdmittedAt); err != nil {
			return err
		}
		result = candidate.receipt(true)
		return nil
	})
	return result, err
}
func (s *BrowserOperations) original(ctx context.Context, db *gorm.DB, store, key string) (browserOperation, error) {
	for _, id := range []string{store, key} {
		if _, err := canonicalUUID(id); err != nil {
			return browserOperation{}, ErrBrowserInvalid
		}
	}
	subject, err := s.use.ReceiptSubject(ctx)
	if err != nil {
		return browserOperation{}, err
	}
	var row browserOperation
	err = db.WithContext(ctx).Where("store_id=? AND operation_id=? AND issuer=? AND actor_id=?", store, key, s.issuer, subject).Take(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		err = ErrNotFound
	}
	return row, err
}
func (s *BrowserOperations) Receipt(ctx context.Context, store, key string) (BrowserReceipt, error) {
	row, err := s.original(ctx, s.db, store, key)
	if err != nil {
		return BrowserReceipt{}, err
	}
	receipt := row.receipt(false)
	var reports []browserReportRecord
	if err = s.db.WithContext(ctx).Where("organization_id=? AND operation_id=?", row.OrganizationID, row.OperationID).Order("received_at,kind").Find(&reports).Error; err != nil {
		return BrowserReceipt{}, err
	}
	for _, r := range reports {
		receipt.Reports = append(receipt.Reports, BrowserReport{SchemaVersion: "store-browser-v1", Kind: r.Kind, Outcome: AuditOutcome(r.Outcome), ReasonCode: r.ReasonCode, ObservedAt: r.ObservedAt})
	}
	return receipt, nil
}
func (s *BrowserOperations) Report(ctx context.Context, store, key string, r BrowserReport) (BrowserReceipt, error) {
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		row, err := s.original(ctx, tx, store, key)
		if err != nil {
			return err
		}
		r.ObservedAt = r.ObservedAt.UTC()
		if err = r.validate(row.Action); err != nil {
			return err
		}
		candidate := browserReportRecord{OrganizationID: row.OrganizationID, OperationID: key, Kind: r.Kind, Outcome: string(r.Outcome), ReasonCode: r.ReasonCode, ObservedAt: r.ObservedAt, ReceivedAt: s.now().UTC(), Fingerprint: browserFingerprint(r)}
		saved := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&candidate)
		if saved.Error != nil {
			return saved.Error
		}
		if saved.RowsAffected == 0 {
			var original browserReportRecord
			if err = tx.Where("organization_id=? AND operation_id=? AND kind=?", row.OrganizationID, key, r.Kind).Take(&original).Error; err != nil {
				return err
			}
			if original.Fingerprint != candidate.Fingerprint {
				return ErrBrowserConflict
			}
			return nil
		}
		action := AuditActionBrowserOpenReported
		if r.Kind == "configure_result" {
			action = AuditActionBrowserConfigured
		} else if r.Kind == "close_result" {
			action = AuditActionBrowserClosed
		}
		return recordBrowserAudit(tx, row, action, r.Outcome, candidate.ReceivedAt)
	})
	if err != nil {
		return BrowserReceipt{}, err
	}
	return s.Receipt(ctx, store, key)
}
func recordBrowserAudit(tx *gorm.DB, row browserOperation, action AuditAction, outcome AuditOutcome, at time.Time) error {
	audit, err := NewGormAuditRepository(tx)
	if err != nil {
		return err
	}
	event := newAuditEvent(row.OrganizationID, row.StoreID, row.OperationID, action, outcome, row.ActorID, nil, RecordStatusActive, RecordStatusActive, AuditFailureNone, at)
	event.EventID = uuid.NewString()
	event.StoreVersion = row.StoreVersion
	_, _, err = audit.Record(tx.Statement.Context, event)
	return err
}

// MigrateBrowserOperationsFixture is exclusively for isolated repository tests.
// Production installs the schema through the existing Store schema owner.
func MigrateBrowserOperationsFixture(db *gorm.DB) error {
	if db == nil {
		return ErrDependencyUnavailable
	}
	return db.AutoMigrate(&browserOperation{}, &browserReportRecord{})
}
