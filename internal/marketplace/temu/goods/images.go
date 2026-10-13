package goods

import (
	"encoding/hex"
	"encoding/json"
	"net/url"
	"sort"
	"strconv"
	"task-processor/internal/marketplace/temu/model"
	"task-processor/internal/product/asset"
)

func ValidImageObservation(v ImageObservation) bool {
	hash, err := hex.DecodeString(v.ContentHash)
	u, urlErr := url.Parse(v.SourceURL)
	if err != nil || len(hash) != 32 || v.AssetID == "" || v.CatID < 1 || v.Usage < 1 || v.Usage > 7 || v.Width < 1 || v.Height < 1 || v.Width > 10000 || v.Height > 10000 || int64(v.Width)*int64(v.Height) > 20_000_000 || v.Bytes < 1 || v.Bytes > 3<<20 || v.MediaType != "image/png" && v.MediaType != "image/jpeg" || urlErr != nil || u.Scheme != "https" || u.Hostname() == "" || u.User != nil || u.Fragment != "" {
		return false
	}
	if len(v.RemoteURLs) == 0 {
		return v.ResponseHash == ""
	}
	hash, err = hex.DecodeString(v.ResponseHash)
	if err != nil || len(hash) != 32 {
		return false
	}
	for _, value := range v.RemoteURLs {
		u, err := url.Parse(value)
		if err != nil || u.Scheme != "https" || u.Hostname() == "" || u.User != nil || u.Fragment != "" {
			return false
		}
	}
	return true
}

func resolveImages(input DraftInput, inventory asset.ApprovedAssetInventory, observations []ImageObservation, categoryType *int64) (DraftInput, []Issue) {
	raw, _ := json.Marshal(input)
	var copied DraftInput
	if json.Unmarshal(raw, &copied) != nil {
		return DraftInput{}, []Issue{{"images", "image_input_invalid", "图片资料无效"}}
	}
	input = copied
	issues := []Issue{}
	add := func(code string) {
		issues = append(issues, Issue{"images", code, "核实已批准图片的内容、用途及目标位置"})
	}
	if input.Mode != "local" {
		return resolveSupplierImages(input, inventory, observations)
	}
	if input.Local == nil {
		add("image_input_invalid")
		return input, issues
	}
	slots := append([]ImageSlot(nil), input.Images...)
	sort.SliceStable(slots, func(i, j int) bool { return slots[i].Sort < slots[j].Sort })
	next := map[string]int{}
	for _, slot := range slots {
		path := slot.Field + ":" + strconv.Itoa(slot.SKU)
		usage := int64(0)
		switch slot.Field {
		case "detail":
			usage = 1
		case "gallery":
			usage = 2
		case "sku":
			usage = 3
		case "size":
			usage = 4
		}
		if usage == 0 || slot.Usage != usage || slot.SKC != 0 || slot.Sort != next[path] || slot.Field != "sku" && slot.SKU != 0 || slot.Field == "sku" && (slot.SKU < 0 || slot.SKU >= len(input.Local.SKUs)) {
			add("image_slot_invalid")
			continue
		}
		next[path]++
		source := ""
		for _, a := range inventory.Assets {
			if a.ID == slot.AssetID {
				source = a.URL
				break
			}
		}
		if source == "" {
			add("asset_unapproved")
			continue
		}
		var observed *ImageObservation
		for i := range observations {
			v := &observations[i]
			if v.AssetID == slot.AssetID && v.SourceURL == source && v.CatID == input.Selection.CatID && v.Usage == usage && ValidImageObservation(*v) {
				if observed != nil {
					add("image_observation_ambiguous")
				}
				observed = v
			}
		}
		if observed == nil {
			add("image_unverified")
			continue
		}
		if usage == 2 || usage == 3 {
			if categoryType == nil || *categoryType < 0 || *categoryType > 4 {
				add("image_geometry_unverified")
				continue
			}
			if *categoryType == 0 {
				if observed.Width < 1340 || observed.Height < 1785 || int64(observed.Width)*4 != int64(observed.Height)*3 {
					add("image_geometry_invalid")
					continue
				}
			} else if observed.Width < 800 || observed.Height < 800 || observed.Width != observed.Height {
				add("image_geometry_invalid")
				continue
			}
		}
		urls := observed.RemoteURLs
		if len(urls) == 0 {
			urls = []string{source}
		}
		switch slot.Field {
		case "sku":
			input.Local.SKUs[slot.SKU].Images = append(input.Local.SKUs[slot.SKU].Images, urls...)
		case "size":
			if input.Local.Size == nil {
				input.Local.Size = &model.LocalSize{}
			}
			input.Local.Size.Images = append(input.Local.Size.Images, urls...)
		case "detail", "gallery":
			if input.Local.Basic.Gallery == nil {
				input.Local.Basic.Gallery = &model.LocalGallery{}
			}
			if slot.Field == "detail" {
				input.Local.Basic.Gallery.DetailImages = append(input.Local.Basic.Gallery.DetailImages, urls...)
			} else {
				input.Local.Basic.Gallery.CarouselImages = append(input.Local.Basic.Gallery.CarouselImages, urls...)
			}
		}
	}
	for _, sku := range input.Local.SKUs {
		if len(sku.Images) == 0 {
			add("sku_image_missing")
		}
	}
	return input, issues
}
