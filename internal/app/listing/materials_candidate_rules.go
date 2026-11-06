package listingapp

import (
	"context"
	"strconv"
	"time"

	"task-processor/internal/agent"
	"task-processor/internal/listing/materials"
	"task-processor/internal/listing/preparation"
	record "task-processor/internal/listing/record/target"
	"task-processor/internal/marketplace/shein/goods"
	"task-processor/internal/product/collection"
	"task-processor/internal/storecenter"
)

type candidateCategoryRulesReader interface {
	ReadCandidateCategoryRules(context.Context, collection.Scope, string, int64) (storecenter.ProductMerchantBinding, goods.OfficialRuleSnapshot, error)
}

// The selection stays outside the editable Target. Only the existing facts
// and rules owners are read, and current authorization is checked on both ends.
func (a *Application) InspectCandidateRulesRecord(ctx context.Context, id, category string) (materials.CandidateRulesDisclosure, error) {
	if ctx == nil || a == nil || a.Authorization == nil || a.Sources == nil || a.Products == nil || a.Records == nil {
		return materials.CandidateRulesDisclosure{}, record.ErrUnavailable
	}
	n, err := strconv.ParseInt(category, 10, 64)
	if err != nil || n <= 0 || strconv.FormatInt(n, 10) != category {
		return materials.CandidateRulesDisclosure{}, record.ErrInvalid
	}
	rulesReader, ok := a.Rules.(candidateCategoryRulesReader)
	if !ok {
		return materials.CandidateRulesDisclosure{}, record.ErrUnavailable
	}
	ctx, cancel := context.WithTimeout(ctx, record.Timeout)
	defer cancel()
	readAt := time.Now().UTC()
	scope, err := a.Authorization.Authorize(ctx, preparation.PermissionRead)
	if err != nil {
		return materials.CandidateRulesDisclosure{}, err
	}
	base, err := a.ReadRecord(ctx, id)
	if err != nil {
		return materials.CandidateRulesDisclosure{}, err
	}
	selected, err := a.Sources.Select(ctx, base.Source.ID)
	if err != nil {
		return materials.CandidateRulesDisclosure{}, err
	}
	owner, source, _, err := selected.Read(ctx)
	if err != nil || owner != scope || source != base.Source || base.TargetID != record.TargetIdentity(scope, source.ID, base.Input.StoreID) {
		return materials.CandidateRulesDisclosure{}, preparation.ErrForbidden
	}
	checkHead := func() error {
		head, e := a.Records.ReadTargetHead(ctx, scope, base.TargetID)
		if e != nil {
			return e
		}
		if collection.Digest(head) != collection.Digest(base) {
			return materials.ErrInspectionChanged
		}
		return nil
	}
	if err = checkHead(); err != nil {
		return materials.CandidateRulesDisclosure{}, err
	}
	product, err := a.Products.ReadEffectiveTargetProduct(ctx, selected, base.EffectiveVersion, base.ApplyReceiptID)
	if err != nil {
		return materials.CandidateRulesDisclosure{}, err
	}
	merchant, rules, err := rulesReader.ReadCandidateCategoryRules(ctx, scope, base.Input.StoreID, n)
	if err != nil {
		return materials.CandidateRulesDisclosure{}, err
	}
	if collection.Digest(merchant) != collection.Digest(base.Merchant) {
		return materials.CandidateRulesDisclosure{}, materials.ErrInspectionChanged
	}
	result, err := materials.NewCandidateRulesDisclosure(base, product, category, rules, readAt)
	if err != nil {
		return materials.CandidateRulesDisclosure{}, err
	}
	if err = checkHead(); err != nil {
		return materials.CandidateRulesDisclosure{}, err
	}
	current, currentSource, _, err := selected.Read(ctx)
	if err != nil || current != scope || currentSource != source {
		return materials.CandidateRulesDisclosure{}, preparation.ErrForbidden
	}
	current, err = a.Authorization.Authorize(ctx, preparation.PermissionRead)
	if err != nil || current != scope {
		return materials.CandidateRulesDisclosure{}, preparation.ErrForbidden
	}
	if err = ctx.Err(); err != nil {
		return materials.CandidateRulesDisclosure{}, err
	}
	return result, nil
}

func (a *Application) ReadCandidateRules(ctx context.Context, b agent.Binding) (materials.CandidateRulesDisclosure, error) {
	if !b.Valid() || b.ContextKind != agent.CandidateRulesContextKind {
		return materials.CandidateRulesDisclosure{}, materials.ErrInspectionChanged
	}
	value, err := a.InspectCandidateRulesRecord(ctx, b.ContextID, b.CandidateRules.CandidateCategoryID)
	if err != nil {
		return materials.CandidateRulesDisclosure{}, err
	}
	if value.Binding != b {
		return materials.CandidateRulesDisclosure{}, materials.ErrInspectionChanged
	}
	return value, nil
}

var _ materials.CandidateRulesReader = (*Application)(nil)
