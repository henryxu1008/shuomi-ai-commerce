package target

import (
	"context"
	"strconv"
	temugoods "task-processor/internal/marketplace/temu/goods"
	"task-processor/internal/product/asset"
)

// Only selected current approved assets are downloaded. Failed observations
// keep the editable record incomplete rather than manufacturing content proof.
func ProbeTemuTargetImages(ctx context.Context, reader TargetTemuImageReader, input temugoods.DraftInput, inventory asset.ApprovedAssetInventory) []temugoods.ImageObservation {
	out := []temugoods.ImageObservation{}
	if ctx == nil || reader == nil || inventory.Scope.TargetPlatform != "temu" {
		return out
	}
	seen := map[string]bool{}
	for _, slot := range input.Images {
		key := slot.AssetID + ":" + strconv.FormatInt(slot.Usage, 10)
		if seen[key] || ctx.Err() != nil {
			continue
		}
		seen[key] = true
		for _, approved := range inventory.Assets {
			if approved.ID == slot.AssetID {
				value, err := reader.ProbeTemu(ctx, approved, input.Selection.CatID, slot.Usage, input.Mode)
				if err == nil && value.AssetID == approved.ID && value.SourceURL == approved.URL && value.CatID == input.Selection.CatID && value.Usage == slot.Usage && temugoods.ValidImageObservation(value) && len(value.RemoteURLs) == 0 {
					out = append(out, value)
				}
				break
			}
		}
	}
	return out
}
