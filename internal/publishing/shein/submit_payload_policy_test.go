package shein

import (
	"testing"

	sheinproduct "task-processor/internal/shein/api/product"
)

type ProductAttributeLike struct {
	AttributeID         int
	AttributeValueID    *int
	AttributeExtraValue string
}

func (a ProductAttributeLike) toShein() sheinproduct.ProductAttribute {
	return sheinproduct.ProductAttribute{
		AttributeID:         a.AttributeID,
		AttributeValueID:    a.AttributeValueID,
		AttributeExtraValue: a.AttributeExtraValue,
	}
}

func TestDeriveSubmitProductSupplierCode(t *testing.T) {
	t.Parallel()

	product := &sheinproduct.Product{
		SupplierCode: "MG8089003001",
		SKCList: []sheinproduct.SKC{
			{
				SKUS: []sheinproduct.SKU{
					{SupplierSKU: "MG8089003001-V295977-TEEC9CE8E-RCODEX-4F2669C9"},
				},
			},
		},
	}

	if got := DeriveSubmitProductSupplierCode(product); got != "MG8089003001-4F2669C9" {
		t.Fatalf("DeriveSubmitProductSupplierCode() = %q, want derived supplier code", got)
	}

	product.SupplierCode = "SUPPLIER-CODE"
	if got := DeriveSubmitProductSupplierCode(product); got != "SUPPLIER-CODE" {
		t.Fatalf("DeriveSubmitProductSupplierCode(existing) = %q, want existing supplier code", got)
	}

	if got := DeriveSubmitProductSupplierCode(nil); got != "" {
		t.Fatalf("DeriveSubmitProductSupplierCode(nil) = %q, want empty", got)
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

func TestValidatePreparedProductPublishPayloadRequiresTemplateSizeChartAttributes(t *testing.T) {
	t.Parallel()

	product := preparedPublishProduct()
	requiredSizeAttrs := []PendingAttributeCandidate{
		{AttributeID: 55, AttributeName: "长度 (cm)", AttributeNameEn: "Length (cm)", Required: true},
		{AttributeID: 20, AttributeName: "胸围 (cm)", AttributeNameEn: "Bust (cm)", Required: true},
	}

	err := ValidatePreparedProductPublishPayloadWithSizeChartAttributes(product, requiredSizeAttrs)
	if err == nil || err.Error() != "SHEIN publish blocked: missing required size chart attributes: 长度 (cm), 胸围 (cm)" {
		t.Fatalf("ValidatePreparedProductPublishPayloadWithSizeChartAttributes(missing required size attrs) error = %v", err)
	}

	product.SizeAttributeList = []sheinproduct.SizeAttribute{
		{AttributeID: 55, AttributeExtraValue: "88", RelateSaleAttributeID: 87, RelateSaleAttributeValueID: 568},
		{AttributeID: 20, AttributeExtraValue: "96", RelateSaleAttributeID: 87, RelateSaleAttributeValueID: 568},
	}
	if err := ValidatePreparedProductPublishPayloadWithSizeChartAttributes(product, requiredSizeAttrs); err != nil {
		t.Fatalf("ValidatePreparedProductPublishPayloadWithSizeChartAttributes(with required size attrs) error = %v", err)
	}
}

func preparedPublishProduct() *sheinproduct.Product {
	quantity := 1
	quantityType := 1
	quantityUnit := 1
	return &sheinproduct.Product{
		SKCList: []sheinproduct.SKC{{
			ImageInfo: sheinproduct.ImageInfo{
				ImageInfoList: []sheinproduct.ImageDetail{{ImageType: 5}, {ImageType: 6}},
			},
			SKUS: []sheinproduct.SKU{{
				QuantityInfo: &sheinproduct.QuantityInfo{
					Quantity:     &quantity,
					QuantityType: &quantityType,
					QuantityUnit: &quantityUnit,
				},
				PackageType: 3,
				StockInfoList: []sheinproduct.StockInfo{{
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
