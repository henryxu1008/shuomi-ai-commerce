package draft

import (
	"task-processor/internal/marketplace/shein/offlinepackage"
	"task-processor/internal/marketplace/shein/publishing"
	"task-processor/internal/product/catalog/canonical"
)

func buildSourceVariants(product *canonical.Product) []offlinepackage.Variant {
	if product == nil || len(product.Variants) == 0 {
		return nil
	}
	result := make([]offlinepackage.Variant, 0, len(product.Variants))
	for _, variant := range product.Variants {
		attributes := make(map[string]string, len(variant.Attributes))
		for key, value := range variant.Attributes {
			attributes[key] = value.Value
		}
		item := offlinepackage.Variant{
			SKU:        variant.SKU,
			Attributes: attributes,
			Stock:      variant.Stock,
			Barcode:    variant.Barcode,
			IsDefault:  variant.IsDefault,
			Dimensions: variant.Dimensions,
			Weight:     variant.Weight,
		}
		if variant.Price != nil {
			item.Price = &publishing.Price{
				Currency:  variant.Price.Currency,
				Amount:    variant.Price.Amount,
				CostPrice: variant.Price.CostPrice,
			}
		}
		if len(variant.Images) > 0 {
			item.Image = variant.Images[0].URL
		}
		result = append(result, item)
	}
	return result
}
