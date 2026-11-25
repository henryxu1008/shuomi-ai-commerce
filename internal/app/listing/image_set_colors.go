package listingapp

import (
	"slices"

	"github.com/henryxu1008/shuomi-ai-commerce/internal/imageagent"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/product/catalog"
)

func listingImageColorOptions(snapshot catalog.PublishedSnapshot, assets []imageagent.AuthorizedAsset) []imageagent.ImageColorOption {
	byURL := make(map[string]string, len(assets))
	for _, entry := range assets {
		byURL[entry.URL] = entry.ID
	}
	var out []imageagent.ImageColorOption
	if len(snapshot.Snapshot.Variants) == 0 {
		return nil
	}
	for _, attribute := range snapshot.Snapshot.Variants[0].Attributes {
		if option, ok := listingColorOption(attribute.Name, snapshot.Snapshot.Variants, byURL); ok {
			out = append(out, option)
			if len(out) == imageagent.MaxImageColorOptions {
				break
			}
		}
	}
	return out
}

func listingColorOption(name string, variants []catalog.Variant, byURL map[string]string) (imageagent.ImageColorOption, bool) {
	option := imageagent.ImageColorOption{Name: name}
	imageOwner := make(map[string]string)
	shared := make(map[string]bool)
	for _, variant := range variants {
		var value string
		for _, attribute := range variant.Attributes {
			if attribute.Name == name {
				if value != "" {
					return option, false
				}
				value = attribute.Value
			}
		}
		if value == "" {
			return option, false
		}
		index := slices.IndexFunc(option.Groups, func(group imageagent.ImageColorGroup) bool { return group.Value == value })
		if index == -1 {
			index = len(option.Groups)
			option.Groups = append(option.Groups, imageagent.ImageColorGroup{Value: value, OriginalIDs: []string{}})
		}
		group := &option.Groups[index]
		for _, img := range variant.Images {
			id := byURL[img.URL]
			if id == "" || slices.Contains(group.OriginalIDs, id) {
				continue
			}
			group.OriginalIDs = append(group.OriginalIDs, id)
			if prior, ok := imageOwner[id]; ok && prior != value {
				shared[id] = true
			}
			imageOwner[id] = value
		}
	}
	if !option.Valid() {
		return option, false
	}
	for i := range option.Groups {
		option.Groups[i].OriginalIDs = slices.DeleteFunc(option.Groups[i].OriginalIDs, func(id string) bool { return shared[id] })
	}
	return option, true
}
