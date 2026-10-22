package supplychainapp

import (
	"context"
	"encoding/json"
	"errors"

	"task-processor/internal/listing/preparation"
	"task-processor/internal/listing/readiness/tools/readinessinspect"
	"task-processor/internal/product/asset"
)

// ProductInputs is an ordinary read. The server selects the original saved
// source and its exact Asset inventory; no Agent or write port participates.
func (a *Application) ProductInputs(ctx context.Context, id string) (json.RawMessage, error) {
	if a == nil || a.Sources == nil || a.Authorization == nil || a.Assets == nil {
		return nil, preparation.ErrUnavailable
	}
	if _, err := a.Authorization.Authorize(ctx, preparation.PermissionManage); err != nil {
		return nil, err
	}
	selected, err := a.Sources.Select(ctx, id)
	if err != nil {
		return nil, err
	}
	_, source, original, err := selected.Read(ctx)
	if err != nil {
		return nil, err
	}
	selection := asset.SourceSelectionRequest{ItemID: source.ID, OriginalPublicationID: original.PublicationID, OriginalSnapshotVersion: original.Version, EffectiveCatalogVersion: original.Version, TargetPlatform: "product"}
	reader := SourceImageReader{Sources: a.Sources, Products: EffectiveProductReader{}, Authorization: a.Authorization, OriginalOnly: true}
	if _, err = reader.ReadSourceSelection(ctx, selection); err != nil {
		return nil, err
	}
	scope := asset.InventoryScope{TenantID: original.Identity.TenantID, ProductKey: original.Identity.ProductKey, TargetPlatform: "product", SourceSnapshotVersion: original.Version}
	inventory, err := a.Assets.GetApprovedInventory(ctx, scope)
	if errors.Is(err, asset.ErrApprovedAssetsNotReady) {
		return readinessinspect.Project(original, nil, "product")
	}
	if err != nil {
		return nil, err
	}
	return readinessinspect.Project(original, &inventory, "product")
}
