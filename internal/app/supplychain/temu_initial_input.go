package supplychainapp

import (
	"strconv"
	temugoods "task-processor/internal/marketplace/temu/goods"
	"task-processor/internal/marketplace/temu/model"
	"task-processor/internal/product/catalog"
	"task-processor/internal/product/collection"
	"task-processor/internal/storecenter"
)

func initialTemuInput(source catalog.PublishedSnapshot, merchant storecenter.ProductMerchantBinding) *temugoods.DraftInput {
	input := &temugoods.DraftInput{Mode: string(merchant.MerchantMode), Images: []temugoods.ImageSlot{}}
	if merchant.MerchantMode == storecenter.MerchantLocal {
		input.Local = &model.LocalGoodsV2{Basic: model.LocalGoodsBasic{Name: source.Snapshot.Title, Description: source.Snapshot.Description, ExternalGoodsID: collection.StableID(source.Identity.ProductKey, merchant.SupplierIdentityHash, "temu-goods")}, SKUs: []model.LocalSKU{}}
		for i, variant := range source.Snapshot.Variants {
			input.Local.SKUs = append(input.Local.SKUs, model.LocalSKU{ExternalSKUID: collection.StableID(source.Identity.ProductKey, merchant.SupplierIdentityHash, "temu-sku", variant.SourceID, variant.SKU, strconv.Itoa(i))})
		}
	} else {
		input.Supplier = &model.SupplierGoods{Name: source.Snapshot.Title, SKCs: []model.SupplierSKC{{ExternalCode: collection.StableID(source.Identity.ProductKey, merchant.SupplierIdentityHash, "temu-skc"), SKUs: []model.SupplierSKU{}}}}
		for i, variant := range source.Snapshot.Variants {
			input.Supplier.SKCs[0].SKUs = append(input.Supplier.SKCs[0].SKUs, model.SupplierSKU{ExternalCode: collection.StableID(source.Identity.ProductKey, merchant.SupplierIdentityHash, "temu-sku", variant.SourceID, variant.SKU, strconv.Itoa(i))})
		}
	}
	return input
}
