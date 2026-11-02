package materials

import (
	"fmt"
	"strconv"
	"strings"

	"task-processor/internal/marketplace/shein/goods"
	"task-processor/internal/product/asset"
	"task-processor/internal/product/collection"
)

func (i Inspection) imageAdvice() (asset.ImageAdvice, error) {
	binding := asset.ImageAdviceBinding{ProductKey: i.Binding.ProductKey, CatalogVersion: i.Binding.CatalogVersion, PublicationID: i.Binding.PublicationID, TargetPlatform: i.Binding.TargetPlatform, InventoryDigest: collection.Digest(i.Inventory),
		Target: &asset.ImageAdviceTarget{RecordID: i.Base.ID, Revision: strconv.FormatInt(i.Base.Revision, 10), StoreID: i.Base.Input.StoreID, Site: i.Base.Merchant.Site, CategoryID: i.Base.Input.Draft.Product.CategoryID, RuleDigest: collection.Digest(i.Rules), SourceID: i.Base.Source.ID, OriginalPublicationID: i.Base.Source.Source.PublicationID, OriginalVersion: strconv.FormatUint(i.Base.Source.Source.Version, 10), ApplyReceiptID: i.Base.ApplyReceiptID}}
	// Only the original saved record's verified observation of this exact
	// approval URL/type can supply real dimensions. Plan placeholders cannot.
	var observations []asset.ImageDimensionFact
	approved := map[string]string{}
	selected := map[string]map[int]bool{}
	for _, image := range i.Inventory.Assets {
		approved[image.ID] = image.URL
	}
	for _, slot := range i.Base.Input.Draft.Images {
		if selected[slot.AssetID] == nil {
			selected[slot.AssetID] = map[int]bool{}
		}
		selected[slot.AssetID][slot.Type] = true
	}
	dimensions, conflicts := map[string]asset.ImageDimensionFact{}, map[string]bool{}
	for _, o := range i.Base.ImageObservations {
		if approved[o.AssetID] != o.SourceURL || !selected[o.AssetID][o.Type] || !goods.ValidImageObservation(o) {
			continue
		}
		if previous, ok := dimensions[o.AssetID]; ok && (previous.Width != o.Width || previous.Height != o.Height) {
			conflicts[o.AssetID] = true
		}
		dimensions[o.AssetID] = asset.ImageDimensionFact{AssetID: o.AssetID, Width: o.Width, Height: o.Height}
	}
	for _, image := range i.Inventory.Assets {
		if observed, ok := dimensions[image.ID]; ok && !conflicts[image.ID] {
			observations = append(observations, observed)
		}
	}
	a, err := asset.AnalyzeImageFacts(binding, i.Inventory, i.InventoryReady, observations...)
	if err != nil {
		return a, err
	}
	// The native owner remains the only official evaluator. Unavailable asset
	// facts cannot establish a missing required image even if its old builder
	// represents an unread inventory with the same empty slice.
	if i.InventoryReady {
		native := goods.AssessOfficial(i.Base.Input.Draft, i.Rules, i.Inventory, i.Base.ImageObservations, i.Base.ProofObservations).Native()
		facts := map[string]asset.ImageAdviceFact{}
		for _, fact := range a.Facts {
			facts[fact.AssetID] = fact
		}
		groups := map[string][]goods.OfficialImageSlot{}
		for _, slot := range i.Base.Input.Draft.Images {
			path := "image_info"
			switch slot.Group {
			case "skc":
				path = fmt.Sprintf("skc_list.%d.image_info", slot.SKC)
			case "sku":
				path = fmt.Sprintf("skc_list.%d.sku_list.%d.image_info", slot.SKC, slot.SKU)
			case "detail":
				path = fmt.Sprintf("skc_list.%d.site_detail_image_info_list", slot.SKC)
			}
			groups[path] = append(groups[path], slot)
		}
		seen := map[string]bool{}
		for _, issue := range native.Issues {
			if !strings.Contains(issue.Field, "image_info") {
				continue
			}
			code, status, kind := "owner_image_issue", "unknown", "verify_fact"
			switch issue.Code {
			case "image_dimensions":
				// The owner reports a group path. Project each selected image
				// independently with its original size rule; another image's
				// missing fact cannot downgrade an established conflict.
				if seen["dimensions/"+issue.Field] {
					continue
				}
				seen["dimensions/"+issue.Field] = true
				for _, slot := range groups[issue.Field] {
					fact, ok := facts[slot.AssetID]
					if !ok {
						continue
					}
					code, status, reason := "dimensions_conflict", "known_conflict", "已知真实尺寸不符合所选图片位置规范。"
					if fact.DimensionsStatus == "unknown" {
						code, status, reason = "dimensions_unknown", "unknown", "所选图片的真实尺寸尚未核实，请先在原图片入口核对。"
					} else if goods.OfficialImageSizeAllowed(slot.Group, slot.Type, fact.Width, fact.Height) {
						continue
					}
					rule := fmt.Sprintf("%s/%s/%d/%d/%d/%d", binding.Target.RuleDigest, slot.Group, slot.SKC, slot.SKU, slot.Sort, slot.Type)
					if seen[code+"/"+rule+"/"+fact.AssetID] {
						continue
					}
					seen[code+"/"+rule+"/"+fact.AssetID] = true
					a.AddCheck(code, status, []string{fact.AssetID}, rule, reason, "verify_fact")
					if status == "unknown" {
						a.Unresolved = append(a.Unresolved, reason)
					}
				}
				continue
			case "missing_images":
				code, status, kind = "required_image_missing", "known_conflict", "open_existing_images"
			case "invalid":
				code, status, kind = "placement_conflict", "known_conflict", "check_placement"
			case "rule_unavailable":
				code, status = "rule_unavailable", "unknown"
			}
			reason := issue.Message
			if seen[code+"|"+issue.Field+"|"+reason] {
				continue
			}
			seen[code+"|"+issue.Field+"|"+reason] = true
			a.AddCheck(code, status, nil, binding.Target.RuleDigest+"/"+issue.Field, reason, kind)
			if status == "unknown" {
				a.Unresolved = append(a.Unresolved, reason)
			}
		}
	}
	if err = a.Seal(InspectToolRef().ID, InspectToolRef().Version, ""); err != nil {
		return asset.ImageAdvice{}, err
	}
	return a, nil
}
