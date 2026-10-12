package supplychainapp

import (
	"context"
	record "task-processor/internal/listing/record/target"
	temugoods "task-processor/internal/marketplace/temu/goods"
	"task-processor/internal/product/asset"
	"task-processor/internal/product/collection"
)

// The current AssetDB owner is read before claim and again before each send.
// Product UoW validates only its own immutable record and approved references.
func (s *UploadService) temuFacts(ctx context.Context, scope collection.Scope, saved record.TargetRecord) ([]temugoods.ImageObservation, error) {
	if s == nil || ctx == nil || s.dependencies.TemuRules == nil || s.dependencies.TemuImages == nil || !record.PublicationReady(saved) || saved.Input.Temu == nil {
		return nil, record.ErrNotReady
	}
	if err := s.authorize(ctx, scope); err != nil {
		return nil, err
	}
	if s.dependencies.ReviewGate != nil {
		if err := s.dependencies.ReviewGate(ctx, scope, saved); err != nil {
			return nil, err
		}
	}
	head, err := s.dependencies.Records.ReadTargetHead(ctx, scope, saved.TargetID)
	if err != nil || head.ID != saved.ID || collection.Digest(head) != collection.Digest(saved) {
		return nil, record.ErrConflict
	}
	proof, err := s.source(ctx, scope, saved)
	if err != nil {
		return nil, err
	}
	product, err := s.dependencies.Products.ReadEffectiveTargetProduct(ctx, proof, saved.EffectiveVersion, saved.ApplyReceiptID)
	if err != nil || product.Identity.TenantID != scope.OrganizationID || product.Identity.ProductKey != saved.Source.Source.ProductKey || product.Version != saved.EffectiveVersion || collection.Digest(product.Snapshot) != saved.ProductHash {
		return nil, record.ErrConflict
	}
	assetScope := asset.InventoryScope{TenantID: scope.OrganizationID, ProductKey: product.Identity.ProductKey, TargetPlatform: "temu", SourceSnapshotVersion: product.Version}
	inventory, err := s.dependencies.Assets.GetApprovedInventory(ctx, assetScope)
	if err != nil || inventory.Scope != assetScope || collection.Digest(inventory) != saved.InventoryHash {
		return nil, record.ErrNotReady
	}
	binding, snapshot, err := s.dependencies.TemuRules.ReadTemuTargetRules(ctx, scope, saved.Merchant.StoreID, saved.Input.Temu.Selection)
	if err != nil {
		return nil, err
	}
	if binding != saved.Merchant || temugoods.RuleHash(snapshot) != saved.RulesHash {
		return nil, record.ErrConflict
	}
	observed := record.ProbeTemuTargetImages(ctx, s.dependencies.TemuImages, *saved.Input.Temu, inventory)
	if len(observed) < 1 || collection.Digest(observed) != collection.Digest(saved.TemuImageObservations) {
		return nil, record.ErrConflict
	}
	current := temugoods.Build(*saved.Input.Temu, snapshot, inventory, observed)
	if !current.ReadyForUpload || collection.Digest(current) != collection.Digest(*saved.Temu) {
		return nil, record.ErrNotReady
	}
	// Rule and byte observations perform I/O. Recheck the original current heads
	// after that work, so changes during a probe cannot authorize the next send.
	head, err = s.dependencies.Records.ReadTargetHead(ctx, scope, saved.TargetID)
	if err != nil || head.ID != saved.ID || collection.Digest(head) != collection.Digest(saved) {
		return nil, record.ErrConflict
	}
	inventory, err = s.dependencies.Assets.GetApprovedInventory(ctx, assetScope)
	if err != nil || inventory.Scope != assetScope || collection.Digest(inventory) != saved.InventoryHash {
		return nil, record.ErrNotReady
	}
	if _, err = s.source(ctx, scope, saved); err != nil {
		return nil, err
	}
	if err := s.authorize(ctx, scope); err != nil {
		return nil, err
	}
	return observed, nil
}
