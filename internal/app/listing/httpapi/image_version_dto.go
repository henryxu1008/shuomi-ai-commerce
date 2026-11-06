package listinghttp

import (
	"strconv"
	"task-processor/internal/listing/preparation"
	"task-processor/internal/product/asset"
)

// Catalog versions remain integers inside the owners and decimal strings on
// browser boundaries, including reads of the existing approved inventory.
type listingImageSourceBody struct {
	asset.SourceSelectionRequest
	OriginalSnapshotVersion string `json:"originalSnapshotVersion"`
	EffectiveCatalogVersion string `json:"effectiveCatalogVersion"`
}

func (b listingImageSourceBody) domain() (asset.SourceSelectionRequest, error) {
	result := b.SourceSelectionRequest
	for _, field := range []struct {
		text  string
		value *uint64
	}{{b.OriginalSnapshotVersion, &result.OriginalSnapshotVersion}, {b.EffectiveCatalogVersion, &result.EffectiveCatalogVersion}} {
		version, err := strconv.ParseUint(field.text, 10, 63)
		if err != nil || version == 0 || strconv.FormatUint(version, 10) != field.text {
			return asset.SourceSelectionRequest{}, preparation.ErrInvalid
		}
		*field.value = version
	}
	return result, nil
}

type listingImageApprovalBody struct {
	asset.SourceApprovalCommand
	Selection listingImageSourceBody `json:"selection"`
}

type listingImageInventoryScope struct {
	asset.InventoryScope
	SourceSnapshotVersion string `json:"source_snapshot_version"`
}

type listingImageInventoryResponse struct {
	asset.ApprovedAssetInventory
	Scope listingImageInventoryScope `json:"scope"`
}

func listingImageInventory(inventory asset.ApprovedAssetInventory) listingImageInventoryResponse {
	return listingImageInventoryResponse{ApprovedAssetInventory: inventory, Scope: listingImageInventoryScope{InventoryScope: inventory.Scope, SourceSnapshotVersion: strconv.FormatUint(inventory.Scope.SourceSnapshotVersion, 10)}}
}
