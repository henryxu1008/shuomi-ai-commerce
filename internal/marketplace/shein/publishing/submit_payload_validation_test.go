package publishing

import (
	"task-processor/internal/marketplace/shein/offlinepackage/previewdata"
	"testing"
)

func TestValidateProductPublishPayloadRequiresSKCImages(t *testing.T) {
	t.Parallel()

	if err := ValidateProductPublishPayload(nil); err == nil || err.Error() != "SHEIN publish payload is empty" {
		t.Fatalf("ValidateProductPublishPayload(nil) error = %v, want empty payload error", err)
	}

	product := &previewdata.Product{
		SKCList: []previewdata.SKC{{}},
	}
	if err := ValidateProductPublishPayload(product); err == nil || err.Error() != "SHEIN publish blocked: SKC[0] has no images" {
		t.Fatalf("ValidateProductPublishPayload(no images) error = %v, want image error", err)
	}

	product.SKCList[0].ImageInfo.ImageInfoList = []previewdata.ImageDetail{{ImageType: 5}}
	if err := ValidateProductPublishPayload(product); err == nil || err.Error() != "SHEIN publish blocked: SKC[0] is missing required color block image" {
		t.Fatalf("ValidateProductPublishPayload(no color block) error = %v, want color block error", err)
	}

	product.SKCList[0].ImageInfo.ImageInfoList = []previewdata.ImageDetail{{ImageType: 5}, {ImageType: 6}}
	if err := ValidateProductPublishPayload(product); err != nil {
		t.Fatalf("ValidateProductPublishPayload(valid) error = %v", err)
	}
}

func TestValidatePreparedProductPublishPayloadRequiresNormalizedSKUFields(t *testing.T) {
	t.Parallel()

	product := preparedPublishProduct()
	product.SKCList[0].SKUS[0].QuantityInfo = nil
	if err := ValidatePreparedProductPublishPayload(product); err == nil || err.Error() != "SHEIN publish blocked: SKC[0] SKU[0] is missing quantity_info" {
		t.Fatalf("ValidatePreparedProductPublishPayload(no quantity) error = %v, want quantity error", err)
	}

	product = preparedPublishProduct()
	product.SKCList[0].SKUS[0].PackageType = 0
	if err := ValidatePreparedProductPublishPayload(product); err == nil || err.Error() != "SHEIN publish blocked: SKC[0] SKU[0] is missing package_type" {
		t.Fatalf("ValidatePreparedProductPublishPayload(no package type) error = %v, want package type error", err)
	}

	product = preparedPublishProduct()
	product.SKCList[0].SKUS[0].StockInfoList = nil
	if err := ValidatePreparedProductPublishPayload(product); err == nil || err.Error() != "SHEIN publish blocked: SKC[0] SKU[0] is missing stock_info_list" {
		t.Fatalf("ValidatePreparedProductPublishPayload(no stock info) error = %v, want stock info error", err)
	}

	product = preparedPublishProduct()
	product.SKCList[0].SKUS[0].Length = ""
	if err := ValidatePreparedProductPublishPayload(product); err == nil || err.Error() != "SHEIN publish blocked: SKC[0] SKU[0] is missing package dimensions" {
		t.Fatalf("ValidatePreparedProductPublishPayload(no dimensions) error = %v, want dimensions error", err)
	}

	product = preparedPublishProduct()
	if err := ValidatePreparedProductPublishPayload(product); err != nil {
		t.Fatalf("ValidatePreparedProductPublishPayload(valid) error = %v", err)
	}
}

func preparedPublishProduct() *previewdata.Product {
	quantity := 1
	quantityType := 1
	quantityUnit := 1
	return &previewdata.Product{
		SKCList: []previewdata.SKC{{
			ImageInfo: previewdata.ImageInfo{
				ImageInfoList: []previewdata.ImageDetail{{ImageType: 5}, {ImageType: 6}},
			},
			SKUS: []previewdata.SKU{{
				QuantityInfo: &previewdata.QuantityInfo{
					Quantity:     &quantity,
					QuantityType: &quantityType,
					QuantityUnit: &quantityUnit,
				},
				PackageType: 3,
				StockInfoList: []previewdata.StockInfo{{
					MerchantWarehouseCode: "DEFAULT",
					InventoryNum:          1,
				}},
				Length:     "1",
				Width:      "1",
				Height:     "1",
				LengthUnit: "Inch",
			}},
		}},
	}
}
