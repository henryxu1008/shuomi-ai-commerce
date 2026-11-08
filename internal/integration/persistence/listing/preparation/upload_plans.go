package preparationpersistence

import (
	"context"
	"encoding/json"
	"errors"
	"sort"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"task-processor/internal/listing/preparation"
	"task-processor/internal/product/collection"
)

type UploadPlanRepository struct{ db *gorm.DB }

func NewUploadPlanRepository(ctx context.Context, db *gorm.DB) (*UploadPlanRepository, error) {
	if VerifyUploadPlanSchema(ctx, db) != nil {
		return nil, preparation.ErrUnavailable
	}
	return &UploadPlanRepository{db}, nil
}

type uploadPlanRow struct {
	Cancelled                                                    bool
	OrganizationID, ActorID, MemberID, ID, CommandKey, InputHash string
	InputJSON                                                    []byte
	PreparationID, StoreID, StoreBindingHash                     string
	ItemCount, Revision                                          int64
	CreatedAt                                                    time.Time
}

func (uploadPlanRow) TableName() string { return "listing_preparation_upload_plans" }
func (r uploadPlanRow) value() (preparation.UploadPlan, error) {
	var in preparation.UploadPlanInput
	scope := collection.Scope{OrganizationID: r.OrganizationID, ActorID: r.ActorID, MemberID: r.MemberID}
	if scope.Validate() != nil || json.Unmarshal(r.InputJSON, &in) != nil || in.Validate() != nil || collection.Digest(in) != r.InputHash || r.ID != preparation.UploadPlanID(scope, r.CommandKey) || r.PreparationID != in.PreparationID || r.StoreID != in.StoreID || len(r.StoreBindingHash) != 64 || r.ItemCount < 1 || r.Revision < 1 {
		return preparation.UploadPlan{}, preparation.ErrUnavailable
	}
	return preparation.UploadPlan{Cancelled: r.Cancelled, ID: r.ID, Owner: scope, Input: in, StoreBindingHash: r.StoreBindingHash, Count: r.ItemCount, Revision: r.Revision, CreatedAt: r.CreatedAt.UTC()}, nil
}

type uploadPlanItemRow struct {
	OrganizationID, ActorID, MemberID, PlanID, SourceID      string
	SourceVersion                                            uint64
	Revision                                                 int64
	AdaptOperationID, OptimizeOperationID, UploadOperationID *string
}

func (uploadPlanItemRow) TableName() string { return "listing_preparation_upload_plan_items" }
func (r uploadPlanItemRow) value() preparation.UploadPlanItem {
	v := preparation.UploadPlanItem{SourceID: r.SourceID, SourceVersion: r.SourceVersion, Revision: r.Revision}
	if r.AdaptOperationID != nil {
		v.AdaptOperationID = *r.AdaptOperationID
	}
	if r.OptimizeOperationID != nil {
		v.OptimizeOperationID = *r.OptimizeOperationID
	}
	if r.UploadOperationID != nil {
		v.UploadOperationID = *r.UploadOperationID
	}
	return v
}

func readUploadPlan(db *gorm.DB, scope collection.Scope, id string) (uploadPlanRow, error) {
	var row uploadPlanRow
	if scope.Validate() != nil || !collection.ValidID(id) {
		return row, preparation.ErrInvalid
	}
	if err := operationBase(db, scope).Where("id=?", id).Take(&row).Error; err != nil {
		return row, operationError(err)
	}
	_, err := row.value()
	return row, err
}
func (r *UploadPlanRepository) ReadUploadPlan(ctx context.Context, scope collection.Scope, id string) (preparation.UploadPlan, error) {
	row, err := readUploadPlan(r.db.WithContext(ctx), scope, id)
	if err != nil {
		return preparation.UploadPlan{}, err
	}
	return row.value()
}
func (r *UploadPlanRepository) FindUploadPlan(ctx context.Context, scope collection.Scope, key, digest string) (preparation.UploadPlanReceipt, error) {
	row, err := readUploadPlan(r.db.WithContext(ctx), scope, preparation.UploadPlanID(scope, key))
	if err != nil {
		return preparation.UploadPlanReceipt{}, err
	}
	if digest != "" && row.InputHash != digest {
		return preparation.UploadPlanReceipt{}, preparation.ErrConflict
	}
	value, err := row.value()
	return preparation.UploadPlanReceipt{Plan: value, Replayed: true}, err
}
func (r *UploadPlanRepository) CreateUploadPlan(ctx context.Context, proof preparation.UploadPlanCommit) (preparation.UploadPlanReceipt, error) {
	scope, key, in, binding, err := proof.Read(ctx)
	if err != nil {
		return preparation.UploadPlanReceipt{}, err
	}
	tx := r.db.WithContext(ctx).Begin()
	if tx.Error != nil {
		return preparation.UploadPlanReceipt{}, preparation.ErrUnavailable
	}
	defer tx.Rollback()
	if tx.Exec("SET LOCAL synchronous_commit=on").Error != nil || tx.Exec("SELECT pg_advisory_xact_lock(hashtextextended(?,0))", collection.Digest([]string{scope.OrganizationID, scope.ActorID, "upload-plan", key})).Error != nil {
		return preparation.UploadPlanReceipt{}, preparation.ErrUnavailable
	}
	prior, err := readUploadPlan(tx, scope, preparation.UploadPlanID(scope, key))
	if err == nil {
		if prior.InputHash != collection.Digest(in) {
			return preparation.UploadPlanReceipt{}, preparation.ErrConflict
		}
		p, e := prior.value()
		return preparation.UploadPlanReceipt{Plan: p, Replayed: true}, e
	}
	if !errors.Is(err, preparation.ErrNotFound) {
		return preparation.UploadPlanReceipt{}, err
	}
	var parent preparationRow
	if e := operationBase(tx, scope).Clauses(clause.Locking{Strength: "SHARE"}).Where("id=?", in.PreparationID).Take(&parent).Error; e != nil {
		return preparation.UploadPlanReceipt{}, operationError(e)
	}
	if parent.Revision != in.ExpectedRevision {
		return preparation.UploadPlanReceipt{}, preparation.ErrConflict
	}
	raw, err := json.Marshal(in)
	if err != nil {
		return preparation.UploadPlanReceipt{}, preparation.ErrInvalid
	}
	count := parent.ItemCount
	if len(in.SourceIDs) > 0 {
		count = int64(len(in.SourceIDs))
	}
	if count < 1 {
		return preparation.UploadPlanReceipt{}, preparation.ErrConflict
	}
	row := uploadPlanRow{OrganizationID: scope.OrganizationID, ActorID: scope.ActorID, MemberID: scope.MemberID, ID: preparation.UploadPlanID(scope, key), CommandKey: key, InputHash: collection.Digest(in), InputJSON: raw, PreparationID: in.PreparationID, StoreID: in.StoreID, StoreBindingHash: binding, ItemCount: count, Revision: 1, CreatedAt: time.Now().UTC().Truncate(time.Microsecond)}
	if tx.Create(&row).Error != nil {
		return preparation.UploadPlanReceipt{}, preparation.ErrUnavailable
	}
	query := operationBase(tx.Model(&sourceRow{}), scope).Where("preparation_id=?", in.PreparationID)
	// Transfer fixes source membership and versions. The serving role cannot
	// update/delete sources; row locking would incorrectly require UPDATE.
	// The parent preparation lock above still binds its checked revision.
	if len(in.SourceIDs) > 0 {
		query = query.Where("id IN ?", in.SourceIDs)
	}
	after := ""
	var copied int64
	for {
		var sources []sourceRow
		page := query.Session(&gorm.Session{})
		if after != "" {
			page = page.Where("id>?::uuid", after)
		}
		if page.Order("id").Limit(100).Find(&sources).Error != nil {
			return preparation.UploadPlanReceipt{}, preparation.ErrUnavailable
		}
		if len(sources) == 0 {
			break
		}
		items := make([]uploadPlanItemRow, 0, len(sources))
		for _, s := range sources {
			items = append(items, uploadPlanItemRow{OrganizationID: scope.OrganizationID, ActorID: scope.ActorID, MemberID: scope.MemberID, PlanID: row.ID, SourceID: s.ID, SourceVersion: s.OriginalVersion, Revision: 1})
		}
		if tx.Create(&items).Error != nil {
			return preparation.UploadPlanReceipt{}, preparation.ErrUnavailable
		}
		copied += int64(len(items))
		after = sources[len(sources)-1].ID
	}
	if copied != count {
		return preparation.UploadPlanReceipt{}, preparation.ErrConflict
	}
	if _, _, _, _, err = proof.Read(ctx); err != nil {
		return preparation.UploadPlanReceipt{}, err
	}
	if tx.Commit().Error != nil {
		return preparation.UploadPlanReceipt{}, preparation.ErrUnknown
	}
	p, err := row.value()
	return preparation.UploadPlanReceipt{Plan: p}, err
}

func (r *UploadPlanRepository) ListUploadPlans(ctx context.Context, scope collection.Scope, q collection.Query) (collection.Page[preparation.UploadPlan], error) {
	out := collection.Page[preparation.UploadPlan]{Items: []preparation.UploadPlan{}}
	if scope.Validate() != nil || q.Validate() != nil || q.Keyword != "" {
		return out, preparation.ErrInvalid
	}
	limit := q.Limit
	if limit == 0 {
		limit = 20
	}
	base := operationBase(r.db.WithContext(ctx).Model(&uploadPlanRow{}), scope)
	if base.Count(&out.Total).Error != nil {
		return out, preparation.ErrUnavailable
	}
	if q.After != "" {
		base = base.Where("id>?::uuid", q.After)
	}
	var rows []uploadPlanRow
	if base.Order("id").Limit(limit+1).Find(&rows).Error != nil {
		return out, preparation.ErrUnavailable
	}
	if len(rows) > limit {
		rows = rows[:limit]
		out.NextCursor = rows[len(rows)-1].ID
	}
	for _, row := range rows {
		v, e := row.value()
		if e != nil {
			return out, e
		}
		out.Items = append(out.Items, v)
	}
	return out, nil
}
func (r *UploadPlanRepository) ListUploadPlanItems(ctx context.Context, scope collection.Scope, id string, q collection.Query) (collection.Page[preparation.UploadPlanItem], error) {
	out := collection.Page[preparation.UploadPlanItem]{Items: []preparation.UploadPlanItem{}}
	if q.Validate() != nil || q.Keyword != "" {
		return out, preparation.ErrInvalid
	}
	if _, err := r.ReadUploadPlan(ctx, scope, id); err != nil {
		return out, err
	}
	limit := q.Limit
	if limit == 0 {
		limit = 20
	}
	base := operationBase(r.db.WithContext(ctx).Model(&uploadPlanItemRow{}), scope).Where("plan_id=?", id)
	if base.Count(&out.Total).Error != nil {
		return out, preparation.ErrUnavailable
	}
	if q.After != "" {
		base = base.Where("source_id>?::uuid", q.After)
	}
	var rows []uploadPlanItemRow
	if base.Order("source_id").Limit(limit+1).Find(&rows).Error != nil {
		return out, preparation.ErrUnavailable
	}
	if len(rows) > limit {
		rows = rows[:limit]
		out.NextCursor = rows[len(rows)-1].SourceID
	}
	for _, row := range rows {
		out.Items = append(out.Items, row.value())
	}
	return out, nil
}

func findUploadPlanAdvance(db *gorm.DB, scope collection.Scope, id, key, digest string) (preparation.UploadPlanAdvanceReceipt, error) {
	out := preparation.UploadPlanAdvanceReceipt{PlanID: id, OperationIDs: []string{}, Replayed: true}
	var rows []operationRow
	if e := operationBase(db, scope).Where("input_json->>'planId'=? AND input_json->>'planCommandKey'=?", id, key).Order("id").Find(&rows).Error; e != nil {
		return out, preparation.ErrUnavailable
	}
	if len(rows) == 0 {
		return out, preparation.ErrNotFound
	}
	for _, row := range rows {
		op, e := row.value()
		if e != nil {
			return out, e
		}
		if digest != "" && op.Input.PlanRequestHash != digest {
			return out, preparation.ErrConflict
		}
		out.OperationIDs = append(out.OperationIDs, op.ID)
	}
	return out, nil
}
func (r *UploadPlanRepository) FindUploadPlanAdvance(ctx context.Context, scope collection.Scope, id, key, digest string) (preparation.UploadPlanAdvanceReceipt, error) {
	return findUploadPlanAdvance(r.db.WithContext(ctx), scope, id, key, digest)
}

func (r *UploadPlanRepository) AdvanceUploadPlan(ctx context.Context, proof preparation.UploadPlanAdvanceCommit) (preparation.UploadPlanAdvanceReceipt, error) {
	plan, key, request, err := proof.Read(ctx)
	if err != nil {
		return preparation.UploadPlanAdvanceReceipt{}, err
	}
	out := preparation.UploadPlanAdvanceReceipt{PlanID: plan.ID, OperationIDs: []string{}}
	tx := r.db.WithContext(ctx).Begin()
	if tx.Error != nil {
		return out, preparation.ErrUnavailable
	}
	defer tx.Rollback()
	if tx.Exec("SET LOCAL synchronous_commit=on").Error != nil {
		return out, preparation.ErrUnavailable
	}
	var locked uploadPlanRow
	if e := operationBase(tx, plan.Owner).Clauses(clause.Locking{Strength: "UPDATE"}).Where("id=?", plan.ID).Take(&locked).Error; e != nil {
		return out, operationError(e)
	}
	prior, err := findUploadPlanAdvance(tx, plan.Owner, plan.ID, key, collection.Digest(request))
	if err == nil {
		return prior, nil
	}
	if !errors.Is(err, preparation.ErrNotFound) {
		return out, err
	}
	if locked.Cancelled || locked.Revision != request.ExpectedRevision {
		return out, preparation.ErrConflict
	}
	digest := preparation.NewUploadPlanPreviewDigest(plan)
	query := operationBase(tx.Model(&uploadPlanItemRow{}), plan.Owner).Where("plan_id=?", plan.ID)
	if len(request.SourceIDs) > 0 {
		query = query.Where("source_id IN ?", request.SourceIDs)
	}
	after := ""
	var selected int64
	chunk := 0
	for {
		page := query.Session(&gorm.Session{})
		if after != "" {
			page = page.Where("source_id>?::uuid", after)
		}
		var rows []uploadPlanItemRow
		if page.Clauses(clause.Locking{Strength: "UPDATE"}).Order("source_id").Limit(100).Find(&rows).Error != nil {
			return out, preparation.ErrUnavailable
		}
		if len(rows) == 0 {
			break
		}
		groups := map[string][]preparation.UploadPlanItemView{}
		for _, row := range rows {
			view, e := proof.Resolve(ctx, row.value())
			if e != nil {
				return out, e
			}
			digest.Add(view)
			selected++
			if preparation.IsUploadPlanAction(view.Step) {
				groups[view.Step] = append(groups[view.Step], view)
			}
		}
		for _, step := range []string{preparation.OperationAdapt, preparation.OperationOptimize, preparation.OperationUpload} {
			views := groups[step]
			if len(views) == 0 {
				continue
			}
			input := plan.Input.Operation(step)
			input.PlanID, input.PlanCommandKey, input.PlanRequestHash = plan.ID, key, collection.Digest(request)
			for _, view := range views {
				input.SourceIDs = append(input.SourceIDs, view.Item.SourceID)
				if step != preparation.OperationAdapt {
					input.TargetApprovals = append(input.TargetApprovals, view.Facts.Target)
				}
				if step == preparation.OperationUpload && view.Facts.WalmartApproval != nil {
					input.WalmartApprovals = append(input.WalmartApprovals, *view.Facts.WalmartApproval)
				}
			}
			if input.Validate() != nil {
				return out, preparation.ErrInvalid
			}
			if e := proof.AuthorizeStep(ctx, input); e != nil {
				return out, e
			}
			commandKey := collection.StableID(plan.ID, key, step, collection.Digest(chunk))
			child, e := prepareOperationTx(ctx, tx, plan.Owner, commandKey, input)
			if e != nil {
				return out, e
			}
			out.OperationIDs = append(out.OperationIDs, child.Operation.ID)
			chunk++
			column := map[string]string{preparation.OperationAdapt: "adapt_operation_id", preparation.OperationOptimize: "optimize_operation_id", preparation.OperationUpload: "upload_operation_id"}[step]
			for _, view := range views {
				result := operationBase(tx.Model(&uploadPlanItemRow{}), plan.Owner).Where("plan_id=? AND source_id=? AND revision=? AND "+column+" IS NULL", plan.ID, view.Item.SourceID, view.Item.Revision).Updates(map[string]any{column: child.Operation.ID, "revision": gorm.Expr("revision+1")})
				if result.Error != nil {
					return out, preparation.ErrUnavailable
				}
				if result.RowsAffected != 1 {
					return out, preparation.ErrConflict
				}
			}
		}
		after = rows[len(rows)-1].SourceID
	}
	expected := plan.Count
	if len(request.SourceIDs) > 0 {
		expected = int64(len(request.SourceIDs))
	}
	if selected != expected || digest.Sum() != request.PreviewHash || len(out.OperationIDs) == 0 {
		return out, preparation.ErrConflict
	}
	result := operationBase(tx.Model(&uploadPlanRow{}), plan.Owner).Where("id=? AND revision=?", plan.ID, request.ExpectedRevision).UpdateColumn("revision", gorm.Expr("revision+1"))
	if result.Error != nil {
		return out, preparation.ErrUnavailable
	}
	if result.RowsAffected != 1 {
		return out, preparation.ErrConflict
	}
	if _, _, _, err = proof.Read(ctx); err != nil {
		return out, err
	}
	if tx.Commit().Error != nil {
		return out, preparation.ErrUnknown
	}
	sort.Strings(out.OperationIDs)
	return out, nil
}

var _ preparation.UploadPlanRepository = (*UploadPlanRepository)(nil)

func (r *UploadPlanRepository) ReadUploadPlanOperation(ctx context.Context, scope collection.Scope, id, sourceID string) (preparation.Operation, preparation.OperationItem, error) {
	op, err := readOperation(r.db.WithContext(ctx), scope, id)
	if err != nil {
		return preparation.Operation{}, preparation.OperationItem{}, err
	}
	var row operationItemRow
	if !collection.ValidID(sourceID) {
		return preparation.Operation{}, preparation.OperationItem{}, preparation.ErrInvalid
	}
	if e := operationBase(r.db.WithContext(ctx), scope).Where("operation_id=? AND source_id=?", id, sourceID).Take(&row).Error; e != nil {
		return preparation.Operation{}, preparation.OperationItem{}, operationError(e)
	}
	value, err := op.value()
	if err != nil || !preparation.ValidItemStatus(row.Status) {
		return preparation.Operation{}, preparation.OperationItem{}, preparation.ErrUnavailable
	}
	return value, row.value(), nil
}

func (r *UploadPlanRepository) CancelUploadPlan(ctx context.Context, proof preparation.UploadPlanCancelCommit) (preparation.UploadPlan, error) {
	plan, err := proof.Read(ctx)
	if err != nil {
		return preparation.UploadPlan{}, err
	}
	tx := r.db.WithContext(ctx).Begin()
	if tx.Error != nil {
		return preparation.UploadPlan{}, preparation.ErrUnavailable
	}
	defer tx.Rollback()
	if tx.Exec("SET LOCAL synchronous_commit=on").Error != nil {
		return preparation.UploadPlan{}, preparation.ErrUnavailable
	}
	var row uploadPlanRow
	if e := operationBase(tx, plan.Owner).Clauses(clause.Locking{Strength: "UPDATE"}).Where("id=?", plan.ID).Take(&row).Error; e != nil {
		return preparation.UploadPlan{}, operationError(e)
	}
	if !row.Cancelled {
		if row.Revision != plan.Revision {
			return preparation.UploadPlan{}, preparation.ErrConflict
		}
		result := operationBase(tx.Model(&uploadPlanRow{}), plan.Owner).Where("id=? AND revision=?", plan.ID, plan.Revision).Updates(map[string]any{"cancelled": true, "revision": gorm.Expr("revision+1")})
		if result.Error != nil || result.RowsAffected != 1 {
			return preparation.UploadPlan{}, preparation.ErrUnavailable
		}
		row.Cancelled = true
		row.Revision++
	}
	if _, err = proof.Read(ctx); err != nil {
		return preparation.UploadPlan{}, err
	}
	if tx.Commit().Error != nil {
		return preparation.UploadPlan{}, preparation.ErrUnknown
	}
	return row.value()
}
