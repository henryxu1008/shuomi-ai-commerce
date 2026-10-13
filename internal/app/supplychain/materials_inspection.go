package supplychainapp

import (
	"context"
	"errors"

	"task-processor/internal/agent"
	"task-processor/internal/listing/materials"
	"task-processor/internal/listing/preparation"
	record "task-processor/internal/listing/record/target"
	"task-processor/internal/product/asset"
	"task-processor/internal/product/collection"
)

// InspectMaterialsRecord only reads existing owner facts. Preparing Agent
// context cannot create a draft, approve an asset or submit to a platform.
func (a *Application) InspectMaterialsRecord(ctx context.Context, id string) (materials.Inspection, error) {
	if ctx == nil || a == nil || a.Authorization == nil || a.Sources == nil || a.Records == nil || a.Products == nil || a.Rules == nil || a.Assets == nil {
		return materials.Inspection{}, record.ErrUnavailable
	}
	ctx, cancel := context.WithTimeout(ctx, record.Timeout)
	defer cancel()
	scope, err := a.Authorization.Authorize(ctx, preparation.PermissionRead)
	if err != nil {
		return materials.Inspection{}, err
	}
	base, err := a.ReadRecord(ctx, id)
	if err != nil {
		return materials.Inspection{}, err
	}
	selected, err := a.Sources.Select(ctx, base.Source.ID)
	if err != nil {
		return materials.Inspection{}, err
	}
	owner, source, _, err := selected.Read(ctx)
	if err != nil || owner != scope || source != base.Source || base.TargetID != record.TargetIdentity(scope, source.ID, base.Input.StoreID) {
		return materials.Inspection{}, preparation.ErrForbidden
	}
	checkHead := func() error {
		head, err := a.Records.ReadTargetHead(ctx, scope, base.TargetID)
		if err != nil {
			return err
		}
		if head.ID != base.ID || head.Revision != base.Revision || collection.Digest(head) != collection.Digest(base) {
			return materials.ErrInspectionChanged
		}
		return nil
	}
	if err = checkHead(); err != nil {
		return materials.Inspection{}, err
	}
	product, err := a.Products.ReadEffectiveTargetProduct(ctx, selected, base.EffectiveVersion, base.ApplyReceiptID)
	if err != nil {
		return materials.Inspection{}, err
	}
	merchant, rules, err := a.Rules.ReadTargetRules(ctx, scope, base.Input.StoreID, base.Input.Draft)
	if err != nil {
		return materials.Inspection{}, err
	}
	if collection.Digest(merchant) != collection.Digest(base.Merchant) {
		return materials.Inspection{}, materials.ErrInspectionChanged
	}
	inventoryScope := asset.InventoryScope{TenantID: scope.OrganizationID, ProductKey: product.Identity.ProductKey, TargetPlatform: "shein", SourceSnapshotVersion: product.Version}
	inventory, err := a.Assets.GetApprovedInventory(ctx, inventoryScope)
	if errors.Is(err, asset.ErrApprovedAssetsNotReady) {
		inventory, err = asset.ApprovedAssetInventory{Scope: inventoryScope, Assets: []asset.ApprovedAsset{}}, nil
	}
	if err != nil {
		return materials.Inspection{}, err
	}
	value, err := materials.NewInspection(base, product, rules, inventory)
	if err != nil {
		return materials.Inspection{}, err
	}
	if err = checkHead(); err != nil {
		return materials.Inspection{}, err
	}
	current, err := a.Authorization.Authorize(ctx, preparation.PermissionRead)
	if err != nil || current != scope {
		return materials.Inspection{}, preparation.ErrForbidden
	}
	if err = ctx.Err(); err != nil {
		return materials.Inspection{}, err
	}
	return value, nil
}

func (a *Application) ReadMaterialsInspection(ctx context.Context, binding agent.Binding) (materials.Inspection, error) {
	if !binding.Valid() || binding.ContextKind != "listing-target-record" || binding.TargetPlatform != "shein" {
		return materials.Inspection{}, materials.ErrInspectionChanged
	}
	value, err := a.InspectMaterialsRecord(ctx, binding.ContextID)
	if err != nil {
		return materials.Inspection{}, err
	}
	if value.Binding != binding {
		return materials.Inspection{}, materials.ErrInspectionChanged
	}
	return value, nil
}
