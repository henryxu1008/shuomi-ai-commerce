package draft

import (
	"strings"
	"task-processor/internal/marketplace/shein/offlinepackage"
)

type variantGroup struct {
	skcName      string
	saleName     string
	supplierCode string
	mainImageURL string
	attributes   map[string]string
	skus         []offlinepackage.Variant
}

// buildLocalVariantGroups preserves the default local path: one stable group,
// with the first source SKU as its name and no remote resolution/assignment state.
func buildLocalVariantGroups(variants []offlinepackage.Variant, images *offlinepackage.ImageSet) []variantGroup {
	if len(variants) == 0 {
		return nil
	}
	mainImage := ""
	if images != nil {
		mainImage = images.MainImage
	}
	first := variants[0]
	name := firstNonEmpty(first.SKU, "DEFAULT-001")
	return []variantGroup{{skcName: name, saleName: name, supplierCode: first.SKU,
		mainImageURL: firstNonEmpty(first.Image, mainImage), attributes: commonAttributes(variants),
		skus: append([]offlinepackage.Variant(nil), variants...)}}
}

func commonAttributes(variants []offlinepackage.Variant) map[string]string {
	if len(variants) == 0 {
		return nil
	}
	result := cloneSourceMap(variants[0].Attributes)
	for key, value := range result {
		for _, variant := range variants[1:] {
			if normalizeText(variant.Attributes[key]) != normalizeText(value) {
				delete(result, key)
				break
			}
		}
	}
	if len(result) == 0 {
		return nil
	}
	return result
}

func normalizeText(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	replacer := strings.NewReplacer("_", " ", "-", " ", "/", " ")
	return strings.Join(strings.Fields(replacer.Replace(value)), " ")
}
