package target

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"task-processor/internal/listing/completiondraft"
	"task-processor/internal/listing/preparation"
	"task-processor/internal/marketplace/shein/goods"
	model "task-processor/internal/marketplace/shein/model"
	"task-processor/internal/product/asset"
	"task-processor/internal/product/catalog"
	"task-processor/internal/product/collection"
)

func (s *TargetService) readSalesBase(ctx context.Context, scope collection.Scope, in SheinSalesRequest, source preparation.SourceItem) (TargetRecord, completiondraft.Record, error) {
	base, candidate, err := s.readImport(ctx, scope, SheinImportRequest{SourceID: in.SourceID, BaseRecordID: in.BaseRecordID, CompletionDraft: in.CompletionDraft}, source)
	if err != nil {
		return base, candidate, err
	}
	if base.Input.ColorPartition != nil || base.Input.Walmart != nil || base.Input.WalmartRules != nil || base.Merchant.ApplicationType != "self_operated" {
		return base, candidate, ErrNotReady
	}
	return base, candidate, nil
}
func (s *TargetService) salesInventory(ctx context.Context, scope collection.Scope, source preparation.SourceItem, version uint64) (asset.ApprovedAssetInventory, error) {
	wanted := asset.InventoryScope{TenantID: scope.OrganizationID, ProductKey: source.Source.ProductKey, TargetPlatform: "shein", SourceSnapshotVersion: version}
	inv, err := s.dependencies.Assets.GetApprovedInventory(ctx, wanted)
	if errors.Is(err, asset.ErrApprovedAssetsNotReady) {
		return asset.ApprovedAssetInventory{Scope: wanted, Assets: []asset.ApprovedAsset{}}, nil
	}
	if err != nil {
		return inv, ErrUnavailable
	}
	if inv.Scope != wanted {
		return inv, ErrConflict
	}
	return inv, nil
}
func salesProductMatches(scope collection.Scope, source preparation.SourceItem, base TargetRecord, product catalog.PublishedSnapshot) bool {
	return product.Identity.TenantID == scope.OrganizationID && product.Identity.ProductKey == source.Source.ProductKey && product.Version == base.EffectiveVersion && product.PublicationID != "" && base.ProductHash == collection.Digest(product.Snapshot)
}

func (s *TargetService) PreviewSheinSalesMapping(ctx context.Context, in SheinSalesRequest) (SheinSalesPreview, error) {
	if s == nil || ctx == nil || !validImportRequest(SheinImportRequest{SourceID: in.SourceID, BaseRecordID: in.BaseRecordID, CompletionDraft: in.CompletionDraft}) {
		return SheinSalesPreview{}, ErrInvalid
	}
	raw, err := json.Marshal(in)
	if err != nil || len(raw) > MaxPayloadBytes {
		return SheinSalesPreview{}, ErrTooLarge
	}
	var copied SheinSalesRequest
	if json.Unmarshal(raw, &copied) != nil {
		return SheinSalesPreview{}, ErrInvalid
	}
	in = copied
	normalizeSalesSelection(&in.Selection)
	result := SheinSalesPreview{Request: in, Issues: []goods.OfficialIssue{}}
	diagnostic := func(err error) (SheinSalesPreview, error) {
		var field *SheinSalesFieldError
		if errors.As(err, &field) {
			result.Issues = []goods.OfficialIssue{{Field: field.Field, Code: "mapping_conflict", Message: field.Message}}
			return result, nil
		}
		return SheinSalesPreview{}, err
	}
	ctx, cancel := context.WithTimeout(ctx, Timeout)
	defer cancel()
	scope, err := s.authorizeTarget(ctx, preparation.PermissionManage)
	if err != nil {
		return result, err
	}
	selected, err := s.dependencies.Sources.Select(ctx, in.SourceID)
	if err != nil {
		return result, err
	}
	owner, source, _, err := selected.Read(ctx)
	if err != nil || owner != scope {
		return result, ErrForbidden
	}
	base, candidate, err := s.readSalesBase(ctx, scope, in, source)
	if err != nil {
		return result, err
	}
	head, err := s.dependencies.Records.ReadTargetHead(ctx, scope, base.TargetID)
	if err != nil {
		return result, err
	}
	if head.ID != base.ID || head.Revision != base.Revision {
		return result, ErrConflict
	}
	product, err := s.dependencies.Products.ReadEffectiveTargetProduct(ctx, selected, base.EffectiveVersion, base.ApplyReceiptID)
	if err != nil {
		return result, err
	}
	if !salesProductMatches(scope, source, base, product) {
		return result, ErrConflict
	}
	inventory, err := s.salesInventory(ctx, scope, source, base.EffectiveVersion)
	if err != nil {
		return result, err
	}
	_, rules, err := s.dependencies.Rules.ReadTargetRules(ctx, scope, base.Input.StoreID, base.Input.Draft)
	if err != nil {
		return result, err
	}
	draft, err := deriveSheinSales(base.Input.Draft, product.Snapshot, candidate.Content, in.Selection, rules, inventory)
	if err != nil {
		return diagnostic(err)
	}
	merchant, rules, err := s.dependencies.Rules.ReadTargetRules(ctx, scope, base.Input.StoreID, draft)
	if err != nil {
		return result, err
	}
	if !validImportMerchant(scope, base.Input.StoreID, merchant, rules) || rules.ApplicationMode != model.ModeSelfOperated {
		return result, ErrNotReady
	}
	recomputed, err := deriveSheinSales(base.Input.Draft, product.Snapshot, candidate.Content, in.Selection, rules, inventory)
	if err != nil {
		return diagnostic(err)
	}
	if !reflect.DeepEqual(draft, recomputed) {
		return result, ErrConflict
	}
	result.Attributes = &rules.Attributes
	result.Input = &TargetInput{SourceID: source.ID, StoreID: base.Input.StoreID, ExpectedRevision: base.Revision, EffectiveVersion: base.EffectiveVersion, ApplyReceiptID: base.ApplyReceiptID, Draft: draft, SheinSalesMapping: &SheinSalesMapping{Version: SheinSalesMappingVersion, TargetID: base.TargetID, BaseRecordID: base.ID, CompletionDraft: in.CompletionDraft, Selection: in.Selection, RulesHash: collection.Digest(rules), MerchantHash: collection.Digest(merchant), InventoryHash: collection.Digest(inventory)}}
	if _, _, _, err = selected.Read(ctx); err != nil {
		return SheinSalesPreview{}, err
	}
	if _, err = s.authorizeTarget(ctx, preparation.PermissionManage); err != nil {
		return SheinSalesPreview{}, err
	}
	return result, nil
}
func (s *TargetService) validateSheinSalesMapping(ctx context.Context, scope collection.Scope, input TargetInput, source preparation.SourceItem, platform preparedPlatform, product catalog.PublishedSnapshot, inventory asset.ApprovedAssetInventory) error {
	ref := input.SheinSalesMapping
	if ref == nil {
		return nil
	}
	base, candidate, err := s.readSalesBase(ctx, scope, SheinSalesRequest{SourceID: input.SourceID, BaseRecordID: ref.BaseRecordID, CompletionDraft: ref.CompletionDraft, Selection: ref.Selection}, source)
	if err != nil {
		return err
	}
	if ref.Version != SheinSalesMappingVersion || ref.TargetID != base.TargetID || input.TargetID != "" && input.TargetID != base.TargetID || input.StoreID != base.Input.StoreID || input.ExpectedRevision != base.Revision || input.EffectiveVersion != base.EffectiveVersion || input.ApplyReceiptID != base.ApplyReceiptID || ref.RulesHash != platform.rulesHash || ref.MerchantHash != collection.Digest(platform.merchant) || ref.InventoryHash != collection.Digest(inventory) || !salesProductMatches(scope, source, base, product) {
		return ErrConflict
	}
	derived, err := deriveSheinSales(base.Input.Draft, product.Snapshot, candidate.Content, ref.Selection, platform.sheinRules, inventory)
	if err != nil {
		return err
	}
	if !equalSheinDraft(derived, input.Draft) {
		return ErrConflict
	}
	return nil
}
