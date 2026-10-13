package goods

import (
	"sort"
	"strconv"
	"task-processor/internal/marketplace/temu/model"
	"task-processor/internal/product/asset"
)

// These are named E13 image positions. No category integer is interpreted as
// an apparel enum, and source URLs never become official upload evidence.
func resolveSupplierImages(input DraftInput, inventory asset.ApprovedAssetInventory, observations []ImageObservation) (DraftInput, []Issue) {
	issues := []Issue{}
	add := func(code string) {
		issues = append(issues, Issue{"images", code, "选择符合目标位置要求的已批准图片"})
	}
	if input.Supplier == nil {
		add("image_input_invalid")
		return input, issues
	}
	// Empty required LIST fields have one wire representation. Attribute and
	// combination checks still decide whether an empty selection is admissible.
	if input.Supplier.SpecProperties == nil {
		input.Supplier.SpecProperties = []model.SupplierSpecProperty{}
	}
	if input.Supplier.Properties == nil {
		input.Supplier.Properties = []model.SupplierBasicProperty{}
	}
	for ci := range input.Supplier.SKCs {
		for si := range input.Supplier.SKCs[ci].SKUs {
			if input.Supplier.SKCs[ci].SKUs[si].Specs == nil {
				input.Supplier.SKCs[ci].SKUs[si].Specs = []model.SupplierSpecification{}
			}
		}
	}
	slots := append([]ImageSlot(nil), input.Images...)
	sort.SliceStable(slots, func(i, j int) bool { return slots[i].Sort < slots[j].Sort })
	next := map[string]int{}
	for _, slot := range slots {
		usage := map[string]int64{"gallery": 2, "material": 5, "outer": 6, "skc": 2, "thumb": 3}[slot.Field]
		key := slot.Field + ":" + strconv.Itoa(slot.SKC) + ":" + strconv.Itoa(slot.SKU)
		group := slot.Field == "skc" || slot.Field == "thumb"
		if usage == 0 || slot.Usage != usage || slot.Sort != next[key] || !group && (slot.SKC != 0 || slot.SKU != 0) || group && (slot.SKC < 0 || slot.SKC >= len(input.Supplier.SKCs)) || slot.Field != "thumb" && slot.SKU != 0 || slot.Field == "thumb" && (slot.SKU < 0 || slot.SKU >= len(input.Supplier.SKCs[slot.SKC].SKUs)) || (slot.Field == "material" || slot.Field == "thumb") && slot.Sort != 0 {
			add("image_slot_invalid")
			continue
		}
		next[key]++
		source := ""
		for _, a := range inventory.Assets {
			if a.ID == slot.AssetID {
				source = a.URL
				break
			}
		}
		var observed *ImageObservation
		for index := range observations {
			o := &observations[index]
			if o.AssetID == slot.AssetID && o.SourceURL == source && o.CatID == input.Selection.CatID && o.Usage == usage && ValidImageObservation(*o) {
				if observed != nil {
					add("image_observation_ambiguous")
				}
				observed = o
			}
		}
		if source == "" {
			add("asset_unapproved")
			continue
		}
		if observed == nil {
			add("image_unverified")
			continue
		}
		if observed.Bytes > 2<<20 {
			add("image_geometry_invalid")
			continue
		}
		if slot.Field == "skc" {
			if observed.Width < 1340 || observed.Height < 1785 || int64(observed.Width)*4 != int64(observed.Height)*3 {
				add("image_geometry_invalid")
				continue
			}
		} else if slot.Field == "material" || slot.Field == "gallery" || slot.Field == "thumb" {
			if observed.Width < 800 || observed.Height < 800 || observed.Width != observed.Height {
				add("image_geometry_invalid")
				continue
			}
		}
		urls := observed.RemoteURLs
		if len(urls) == 0 {
			urls = []string{source}
		}
		if len(urls) != 1 {
			add("supplier_image_result_ambiguous")
			continue
		}
		switch slot.Field {
		case "material":
			input.Supplier.MaterialImage = urls[0]
		case "gallery":
			input.Supplier.CarouselImages = append(input.Supplier.CarouselImages, urls[0])
		case "outer":
			input.Supplier.OuterImages = append(input.Supplier.OuterImages, model.SupplierImage{URL: urls[0]})
		case "skc":
			input.Supplier.SKCs[slot.SKC].PreviewURLs = append(input.Supplier.SKCs[slot.SKC].PreviewURLs, urls[0])
		case "thumb":
			input.Supplier.SKCs[slot.SKC].SKUs[slot.SKU].ThumbURL = urls[0]
		}
	}
	if input.Supplier.MaterialImage == "" {
		add("supplier_material_image_missing")
	}
	allPreview := len(input.Supplier.SKCs) > 0
	for _, skc := range input.Supplier.SKCs {
		allPreview = allPreview && len(skc.PreviewURLs) > 0
		for _, sku := range skc.SKUs {
			if sku.ThumbURL == "" {
				add("sku_image_missing")
			}
		}
	}
	if len(input.Supplier.CarouselImages) == 0 && !allPreview {
		add("supplier_display_image_missing")
	}
	return input, issues
}
