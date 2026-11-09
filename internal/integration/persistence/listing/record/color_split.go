package recordpersistence

import (
	"context"
	"encoding/json"
	"errors"
	"gorm.io/gorm"
	"reflect"
	"sort"
	record "task-processor/internal/listing/record/target"
	"task-processor/internal/product/collection"
	"time"
)

type splitCommandRow struct {
	OrganizationID, ActorID, MemberID                                 string
	CommandKey, InputHash, FamilyID, SourceID, StoreID, Site, Purpose string
	BaseRecordID, CompletionRecordID                                  string
	FirstTargetID, FirstRecordID                                      string
	FirstRevision                                                     int64
	SecondTargetID, SecondRecordID                                    string
	SecondRevision                                                    int64
	CreatedAt                                                         time.Time
}

func (splitCommandRow) TableName() string { return "listing_target_split_commands" }
func readSplit(db *gorm.DB, scope collection.Scope, field, id string, current bool) (record.ColorSplitReceipt, string, error) {
	if scope.Validate() != nil || !collection.ValidID(id) || (field != "command_key" && field != "family_id") {
		return record.ColorSplitReceipt{}, "", record.ErrInvalid
	}
	var row splitCommandRow
	err := db.Where("organization_id=? AND actor_id=? AND member_id=?", scope.OrganizationID, scope.ActorID, scope.MemberID).Where(field+"=?", id).Take(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return record.ColorSplitReceipt{}, "", record.ErrNotFound
	}
	if err != nil {
		return record.ColorSplitReceipt{}, "", record.ErrUnavailable
	}
	receipt := record.ColorSplitReceipt{FamilyID: row.FamilyID, Records: []record.TargetRecord{}}
	for i, ref := range []struct {
		target, id string
		revision   int64
	}{{row.FirstTargetID, row.FirstRecordID, row.FirstRevision}, {row.SecondTargetID, row.SecondRecordID, row.SecondRevision}} {
		_, v, e := readRecord(db, scope, ref.id)
		if e != nil {
			return receipt, "", e
		}
		if v.TargetID != ref.target || v.Revision != ref.revision || v.Source.ID != row.SourceID || v.Input.StoreID != row.StoreID || v.Merchant.Site != row.Site || v.Input.ColorPartition == nil || v.Input.ColorPartition.FamilyID != row.FamilyID || v.Input.ColorPartition.BaseRecordID != row.BaseRecordID || v.Input.ColorPartition.CompletionDraft.RecordID != row.CompletionRecordID || i == 1 && receipt.Records[0].Input.ColorPartition.PartitionID == v.Input.ColorPartition.PartitionID {
			return receipt, "", record.ErrUnavailable
		}
		if current {
			var h headRow
			if db.Where("organization_id=? AND actor_id=? AND member_id=? AND id=?", scope.OrganizationID, scope.ActorID, scope.MemberID, ref.target).Take(&h).Error != nil {
				return receipt, "", record.ErrUnavailable
			}
			_, head, e := readRecord(db, scope, h.CurrentRecordID)
			if e != nil {
				return receipt, "", e
			}
			if head.TargetID != v.TargetID || !reflect.DeepEqual(head.Input.ColorPartition, v.Input.ColorPartition) || h.Revision != head.Revision || h.FamilyID != row.FamilyID || h.Purpose != "shein_color_local" || h.PartitionID != head.Input.ColorPartition.PartitionID {
				return receipt, "", record.ErrUnavailable
			}
			v = head
		}
		receipt.Records = append(receipt.Records, v)
	}
	return receipt, row.InputHash, nil
}
func (r *Repository) FindColorSplitCommand(ctx context.Context, scope collection.Scope, key, hash string) (record.ColorSplitReceipt, error) {
	v, h, e := readSplit(r.db.WithContext(ctx), scope, "command_key", key, false)
	if e != nil {
		return v, e
	}
	if h != hash {
		return record.ColorSplitReceipt{}, record.ErrConflict
	}
	v.Replayed = true
	return v, nil
}
func (r *Repository) ReadColorSplitCommand(ctx context.Context, scope collection.Scope, key string) (record.ColorSplitReceipt, error) {
	v, _, e := readSplit(r.db.WithContext(ctx), scope, "command_key", key, false)
	return v, e
}
func (r *Repository) ReadColorSplitFamily(ctx context.Context, scope collection.Scope, family string) (record.ColorSplitReceipt, error) {
	v, _, e := readSplit(r.db.WithContext(ctx), scope, "family_id", family, true)
	return v, e
}

// ReadColorSplitCreation returns the immutable initial pair for ownership checks.
// The separate family reader intentionally resolves current editable heads.
func (r *Repository) ReadColorSplitCreation(ctx context.Context, scope collection.Scope, family string) (record.ColorSplitReceipt, error) {
	v, _, e := readSplit(r.db.WithContext(ctx), scope, "family_id", family, false)
	return v, e
}
func validateColorEdit(db *gorm.DB, scope collection.Scope, v record.TargetRecord) error {
	if v.Input.ColorPartition == nil {
		return nil
	}
	if v.Revision <= 1 {
		return record.ErrForbidden
	}
	saved, _, err := readSplit(db, scope, "family_id", v.Input.ColorPartition.FamilyID, false)
	if err != nil {
		return err
	}
	_, base, err := readRecord(db, scope, v.Input.ColorPartition.BaseRecordID)
	if err != nil {
		return err
	}
	return record.ValidateColorEdit(saved, v.Input, base)
}
func (r *Repository) SaveColorSplit(ctx context.Context, proof record.ColorSplitPrepared) (record.ColorSplitReceipt, error) {
	scope, key, hash, plan, values, err := proof.Read(ctx)
	if err != nil {
		return record.ColorSplitReceipt{}, err
	}
	if len(values) != 2 || values[0].TargetID == values[1].TargetID {
		return record.ColorSplitReceipt{}, record.ErrInvalid
	}
	ctx, cancel := context.WithTimeout(ctx, record.Timeout)
	defer cancel()
	tx := r.db.WithContext(ctx).Begin()
	if tx.Error != nil {
		return record.ColorSplitReceipt{}, record.ErrUnavailable
	}
	defer tx.Rollback()
	lock := func(value string) error {
		return tx.Exec("SELECT pg_advisory_xact_lock(hashtextextended(?,0))", value).Error
	}
	for _, v := range []string{collection.Digest([]string{scope.OrganizationID, scope.ActorID, "shein-color-split-command", key}), collection.Digest([]string{scope.OrganizationID, scope.ActorID, "supply-target", record.TargetIdentity(scope, plan.Request.SourceID, values[0].Input.StoreID)}), collection.Digest([]string{scope.OrganizationID, scope.ActorID, "shein-color-family", plan.FamilyID})} {
		if lock(v) != nil {
			return record.ColorSplitReceipt{}, record.ErrUnavailable
		}
	}
	ids := []string{values[0].TargetID, values[1].TargetID}
	sort.Strings(ids)
	for _, id := range ids {
		if lock(collection.Digest([]string{scope.OrganizationID, scope.ActorID, "supply-target", id})) != nil {
			return record.ColorSplitReceipt{}, record.ErrUnavailable
		}
	}
	if prior, h, e := readSplit(tx, scope, "command_key", key, false); e == nil {
		if h != hash {
			return record.ColorSplitReceipt{}, record.ErrConflict
		}
		prior.Replayed = true
		return prior, nil
	} else if !errors.Is(e, record.ErrNotFound) {
		return record.ColorSplitReceipt{}, e
	}
	var count int64
	if tx.Model(&splitCommandRow{}).Where("organization_id=? AND actor_id=? AND family_id=?", scope.OrganizationID, scope.ActorID, plan.FamilyID).Count(&count).Error != nil {
		return record.ColorSplitReceipt{}, record.ErrUnavailable
	}
	if count > 0 {
		return record.ColorSplitReceipt{}, record.ErrSplitExists
	}
	var base headRow
	if tx.Where("organization_id=? AND actor_id=? AND member_id=? AND id=?", scope.OrganizationID, scope.ActorID, scope.MemberID, record.TargetIdentity(scope, plan.Request.SourceID, values[0].Input.StoreID)).Take(&base).Error != nil || base.CurrentRecordID != plan.Request.BaseRecordID {
		return record.ColorSplitReceipt{}, record.ErrConflict
	}
	for i := range values {
		v := &values[i]
		v.CreatedAt = v.CreatedAt.UTC().Truncate(time.Microsecond)
		if !validateRecord(scope, *v) || v.Revision != 1 || v.Input.ColorPartition == nil || v.Input.ColorPartition.FamilyID != plan.FamilyID || v.ID != collection.StableID(scope.OrganizationID, scope.ActorID, "shein-color-record-v1", key, v.Input.ColorPartition.PartitionID) {
			return record.ColorSplitReceipt{}, record.ErrInvalid
		}
		raw, e := json.Marshal(v)
		if e != nil {
			return record.ColorSplitReceipt{}, record.ErrInvalid
		}
		purpose, partition, family := record.RecordPurpose(v.Input)
		row := recordRow{OrganizationID: scope.OrganizationID, ActorID: scope.ActorID, MemberID: scope.MemberID, ID: v.ID, TargetID: v.TargetID, SourceID: v.Source.ID, StoreID: v.Input.StoreID, Site: v.Merchant.Site, Revision: 1, InputHash: collection.Digest(v.Input), BodyHash: collection.Digest(v), BodyJSON: raw, CreatedAt: v.CreatedAt, Purpose: purpose, PartitionID: partition, FamilyID: family}
		if tx.Create(&row).Error != nil {
			return record.ColorSplitReceipt{}, record.ErrUnavailable
		}
		head := headRow{OrganizationID: scope.OrganizationID, ActorID: scope.ActorID, MemberID: scope.MemberID, ID: v.TargetID, SourceID: v.Source.ID, StoreID: v.Input.StoreID, Site: v.Merchant.Site, CurrentRecordID: v.ID, Revision: 1, Purpose: purpose, PartitionID: partition, FamilyID: family}
		if tx.Create(&head).Error != nil {
			return record.ColorSplitReceipt{}, record.ErrUnavailable
		}
	}
	row := splitCommandRow{OrganizationID: scope.OrganizationID, ActorID: scope.ActorID, MemberID: scope.MemberID, CommandKey: key, InputHash: hash, FamilyID: plan.FamilyID, SourceID: plan.Request.SourceID, StoreID: values[0].Input.StoreID, Site: "shein-us", Purpose: "shein_color_local", BaseRecordID: plan.Request.BaseRecordID, CompletionRecordID: plan.Request.CompletionDraft.RecordID, FirstTargetID: values[0].TargetID, FirstRecordID: values[0].ID, FirstRevision: 1, SecondTargetID: values[1].TargetID, SecondRecordID: values[1].ID, SecondRevision: 1, CreatedAt: values[0].CreatedAt}
	if tx.Create(&row).Error != nil {
		return record.ColorSplitReceipt{}, record.ErrUnavailable
	}
	if _, _, _, _, _, err = proof.Read(ctx); err != nil {
		return record.ColorSplitReceipt{}, err
	}
	if tx.Commit().Error != nil {
		return record.ColorSplitReceipt{}, record.ErrTargetUnknown
	}
	return record.ColorSplitReceipt{FamilyID: plan.FamilyID, Records: values}, nil
}

var _ record.ColorSplitRepository = (*Repository)(nil)

func (r *Repository) ListColorSplitFamilies(ctx context.Context, scope collection.Scope, source, store string) ([]record.ColorSplitReceipt, error) {
	if scope.Validate() != nil || !collection.ValidID(source) || !collection.ValidID(store) {
		return nil, record.ErrInvalid
	}
	var rows []splitCommandRow
	if r.db.WithContext(ctx).Where("organization_id=? AND actor_id=? AND member_id=? AND source_id=? AND store_id=?", scope.OrganizationID, scope.ActorID, scope.MemberID, source, store).Order("family_id").Limit(41).Find(&rows).Error != nil {
		return nil, record.ErrUnavailable
	}
	if len(rows) > 40 {
		return nil, record.ErrTooLarge
	}
	out := []record.ColorSplitReceipt{}
	for _, row := range rows {
		v, _, err := readSplit(r.db.WithContext(ctx), scope, "family_id", row.FamilyID, true)
		if err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, nil
}
