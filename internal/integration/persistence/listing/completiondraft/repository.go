package completiondraftpersistence

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"gorm.io/gorm"
	"task-processor/internal/listing/completiondraft"
	"task-processor/internal/listing/preparation"
	"task-processor/internal/product/collection"
)

type Repository struct{ db *gorm.DB }

func NewRepository(ctx context.Context, db *gorm.DB) (*Repository, error) {
	if VerifySchema(ctx, db) != nil {
		return nil, preparation.ErrUnavailable
	}
	return &Repository{db}, nil
}

type recordRow struct {
	OrganizationID, ActorID, MemberID string
	ID, SourceID, DraftID             string
	Revision                          int64
	SourceJSON, ContentJSON           []byte
	ContentHash                       string
	CreatedAt                         time.Time
}

func (recordRow) TableName() string { return Tables[0] }

type headRow struct {
	OrganizationID, ActorID, MemberID, SourceID, RecordID, DraftID string
	Revision                                                       int64
}

func (headRow) TableName() string { return Tables[1] }

type commandRow struct {
	OrganizationID, ActorID, MemberID, CommandKey, InputHash string
	InputJSON                                                []byte
	RecordID, SourceID, DraftID                              string
	Revision                                                 int64
}

func (commandRow) TableName() string { return Tables[2] }

type sourceRow struct {
	OrganizationID, ActorID, MemberID, ID, PreparationID, CollectionItemID string
	CollectionRevision                                                     int64
	ProductKey, PublicationID                                              string
	OriginalVersion                                                        uint64
	SourceKind, SourceOperationID                                          string
}

func (r sourceRow) value() preparation.SourceItem {
	return preparation.SourceItem{ID: r.ID, PreparationID: r.PreparationID, CollectionItemID: r.CollectionItemID, CollectionRevision: r.CollectionRevision, Source: collection.Source{ProductKey: r.ProductKey, PublicationID: r.PublicationID, Version: r.OriginalVersion, Kind: r.SourceKind, OperationID: r.SourceOperationID}}
}
func filter(db *gorm.DB, scope collection.Scope) *gorm.DB {
	return db.Where("organization_id=? AND actor_id=? AND member_id=?", scope.OrganizationID, scope.ActorID, scope.MemberID)
}
func dbError(err error) error {
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return preparation.ErrNotFound
	}
	if err != nil {
		return preparation.ErrUnavailable
	}
	return nil
}
func (r recordRow) value() (completiondraft.Record, error) {
	var source preparation.SourceItem
	var content completiondraft.Content
	if json.Unmarshal(r.SourceJSON, &source) != nil || json.Unmarshal(r.ContentJSON, &content) != nil || source.ID != r.SourceID || (completiondraft.SaveInput{SourceID: source.ID, Content: content}).Validate() != nil || r.Revision < 1 || r.ContentHash != collection.Digest(content) || r.DraftID != collection.StableID(r.OrganizationID, r.ActorID, r.SourceID, "completion-draft") {
		return completiondraft.Record{}, preparation.ErrUnavailable
	}
	return completiondraft.Record{ID: r.ID, DraftID: r.DraftID, Revision: r.Revision, Source: source, Content: content, Status: completiondraft.StatusPending, CreatedAt: r.CreatedAt.UTC()}, nil
}
func readRecord(db *gorm.DB, scope collection.Scope, id string) (completiondraft.Record, error) {
	var row recordRow
	if err := dbError(filter(db, scope).Where("id=?", id).Take(&row).Error); err != nil {
		return completiondraft.Record{}, err
	}
	return row.value()
}
func readCommand(db *gorm.DB, scope collection.Scope, key string) (completiondraft.Receipt, string, error) {
	var cmd commandRow
	if err := dbError(filter(db, scope).Where("command_key=?", key).Take(&cmd).Error); err != nil {
		return completiondraft.Receipt{}, "", err
	}
	var in completiondraft.SaveInput
	if json.Unmarshal(cmd.InputJSON, &in) != nil || in.Validate() != nil || cmd.InputHash != completiondraft.InputHash(in) || cmd.SourceID != in.SourceID || cmd.Revision != in.ExpectedRevision+1 || cmd.RecordID != collection.StableID(scope.OrganizationID, scope.ActorID, key, "completion-record") {
		return completiondraft.Receipt{}, "", preparation.ErrUnavailable
	}
	r, err := readRecord(db, scope, cmd.RecordID)
	if err != nil {
		return completiondraft.Receipt{}, "", err
	}
	if r.Source.ID != cmd.SourceID || r.DraftID != cmd.DraftID || r.Revision != cmd.Revision || r.Content != in.Content {
		return completiondraft.Receipt{}, "", preparation.ErrUnavailable
	}
	return completiondraft.Receipt{Record: r, Replayed: true}, cmd.InputHash, nil
}
func (r *Repository) ReadCommand(ctx context.Context, scope collection.Scope, key string) (completiondraft.Receipt, error) {
	if scope.Validate() != nil || !collection.ValidID(key) {
		return completiondraft.Receipt{}, preparation.ErrInvalid
	}
	v, _, err := readCommand(r.db.WithContext(ctx), scope, key)
	return v, err
}
func (r *Repository) Read(ctx context.Context, scope collection.Scope, sourceID string) (completiondraft.Record, error) {
	if scope.Validate() != nil || !collection.ValidID(sourceID) {
		return completiondraft.Record{}, preparation.ErrInvalid
	}
	var head headRow
	if err := dbError(filter(r.db.WithContext(ctx), scope).Where("source_id=?", sourceID).Take(&head).Error); err != nil {
		return completiondraft.Record{}, err
	}
	v, err := readRecord(r.db.WithContext(ctx), scope, head.RecordID)
	if err == nil && (v.Revision != head.Revision || v.Source.ID != sourceID || v.DraftID != head.DraftID) {
		return completiondraft.Record{}, preparation.ErrUnavailable
	}
	return v, err
}

func (r *Repository) Save(ctx context.Context, proof completiondraft.Prepared) (completiondraft.Receipt, error) {
	scope, key, hash, in, source, err := proof.Read(ctx)
	if err != nil {
		return completiondraft.Receipt{}, err
	}
	tx := r.db.WithContext(ctx).Begin()
	if tx.Error != nil {
		return completiondraft.Receipt{}, preparation.ErrUnavailable
	}
	defer tx.Rollback()
	for _, lock := range []string{collection.Digest([]string{scope.OrganizationID, scope.ActorID, "completion-command", key}), collection.Digest([]string{scope.OrganizationID, scope.ActorID, "completion-source", in.SourceID})} {
		if tx.Exec("SELECT pg_advisory_xact_lock(hashtextextended(?,0))", lock).Error != nil {
			return completiondraft.Receipt{}, preparation.ErrUnavailable
		}
	}
	old, oldHash, err := readCommand(tx, scope, key)
	if err == nil {
		if oldHash != hash {
			return completiondraft.Receipt{}, preparation.ErrConflict
		}
		if _, _, _, _, _, err = proof.Read(ctx); err != nil {
			return completiondraft.Receipt{}, err
		}
		return old, nil
	}
	if !errors.Is(err, preparation.ErrNotFound) {
		return completiondraft.Receipt{}, err
	}
	var retained sourceRow
	if err = dbError(filter(tx.Table("listing_preparation_sources"), scope).Where("id=?", source.ID).Take(&retained).Error); err != nil {
		return completiondraft.Receipt{}, err
	}
	if retained.value() != source {
		return completiondraft.Receipt{}, preparation.ErrForbidden
	}
	var head headRow
	err = filter(tx, scope).Where("source_id=?", source.ID).Take(&head).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		if in.ExpectedRevision != 0 {
			return completiondraft.Receipt{}, preparation.ErrConflict
		}
	} else if err != nil {
		return completiondraft.Receipt{}, preparation.ErrUnavailable
	} else if head.Revision != in.ExpectedRevision {
		return completiondraft.Receipt{}, preparation.ErrConflict
	}
	sourceJSON, _ := json.Marshal(source)
	contentJSON, _ := json.Marshal(in.Content)
	inputJSON, _ := json.Marshal(in)
	row := recordRow{OrganizationID: scope.OrganizationID, ActorID: scope.ActorID, MemberID: scope.MemberID, ID: collection.StableID(scope.OrganizationID, scope.ActorID, key, "completion-record"), SourceID: source.ID, DraftID: collection.StableID(scope.OrganizationID, scope.ActorID, source.ID, "completion-draft"), Revision: in.ExpectedRevision + 1, SourceJSON: sourceJSON, ContentJSON: contentJSON, ContentHash: collection.Digest(in.Content), CreatedAt: time.Now().UTC().Truncate(time.Microsecond)}
	if tx.Create(&row).Error != nil {
		return completiondraft.Receipt{}, preparation.ErrUnavailable
	}
	if in.ExpectedRevision == 0 {
		head = headRow{OrganizationID: scope.OrganizationID, ActorID: scope.ActorID, MemberID: scope.MemberID, SourceID: source.ID, RecordID: row.ID, DraftID: row.DraftID, Revision: row.Revision}
		if tx.Create(&head).Error != nil {
			return completiondraft.Receipt{}, preparation.ErrConflict
		}
	} else {
		updated := filter(tx.Model(&headRow{}), scope).Where("source_id=? AND revision=?", source.ID, in.ExpectedRevision).Updates(map[string]any{"record_id": row.ID, "revision": row.Revision})
		if updated.Error != nil {
			return completiondraft.Receipt{}, preparation.ErrUnavailable
		}
		if updated.RowsAffected != 1 {
			return completiondraft.Receipt{}, preparation.ErrConflict
		}
	}
	cmd := commandRow{OrganizationID: scope.OrganizationID, ActorID: scope.ActorID, MemberID: scope.MemberID, CommandKey: key, InputHash: hash, InputJSON: inputJSON, RecordID: row.ID, SourceID: source.ID, DraftID: row.DraftID, Revision: row.Revision}
	if tx.Create(&cmd).Error != nil {
		return completiondraft.Receipt{}, preparation.ErrUnavailable
	}
	if _, _, _, _, _, err = proof.Read(ctx); err != nil {
		return completiondraft.Receipt{}, err
	}
	if tx.Commit().Error != nil {
		return completiondraft.Receipt{}, preparation.ErrUnknown
	}
	v, err := row.value()
	return completiondraft.Receipt{Record: v}, err
}
func (r *Repository) List(ctx context.Context, scope collection.Scope, preparationID string, q collection.Query) (collection.Page[completiondraft.Record], error) {
	page := collection.Page[completiondraft.Record]{Items: []completiondraft.Record{}}
	if scope.Validate() != nil || !collection.ValidID(preparationID) || q.Validate() != nil {
		return page, preparation.ErrInvalid
	}
	base := r.db.WithContext(ctx).Table("listing_completion_draft_heads h").Joins("JOIN listing_completion_draft_records r ON (r.organization_id,r.actor_id,r.member_id,r.id)=(h.organization_id,h.actor_id,h.member_id,h.record_id)").Joins("JOIN listing_preparation_sources s ON (s.organization_id,s.actor_id,s.member_id,s.id)=(h.organization_id,h.actor_id,h.member_id,h.source_id)").Where("h.organization_id=? AND h.actor_id=? AND h.member_id=? AND s.preparation_id=?", scope.OrganizationID, scope.ActorID, scope.MemberID, preparationID)
	if q.Keyword != "" {
		keyword := strings.NewReplacer("\\", "\\\\", "%", "\\%", "_", "\\_").Replace(q.Keyword)
		base = base.Where("r.content_json->>'title' ILIKE ? ESCAPE '\\' OR s.product_key ILIKE ? ESCAPE '\\'", "%"+keyword+"%", "%"+keyword+"%")
	}
	if base.Count(&page.Total).Error != nil {
		return page, preparation.ErrUnavailable
	}
	if q.After != "" {
		base = base.Where("h.source_id>?", q.After)
	}
	var rows []recordRow
	if base.Select("r.*").Order("h.source_id").Limit(q.Limit+1).Scan(&rows).Error != nil {
		return page, preparation.ErrUnavailable
	}
	if len(rows) > q.Limit {
		rows = rows[:q.Limit]
		page.NextCursor = rows[len(rows)-1].SourceID
	}
	encodedBytes := 0
	for _, row := range rows {
		v, err := row.value()
		if err != nil {
			return page, err
		}
		raw, err := json.Marshal(v)
		if err != nil {
			return page, preparation.ErrUnavailable
		}
		// Keep the existing Supply response budget even when 100 valid, large
		// descriptions are requested. Keyset pagination retains the omitted rows.
		if encodedBytes+len(raw)+1 > (2<<20)-4096 {
			if len(page.Items) == 0 {
				return page, preparation.ErrUnavailable
			}
			page.NextCursor = page.Items[len(page.Items)-1].Source.ID
			break
		}
		encodedBytes += len(raw) + 1
		page.Items = append(page.Items, v)
	}
	return page, nil
}
