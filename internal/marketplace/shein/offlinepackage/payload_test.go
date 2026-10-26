package offlinepackage

import (
	"strings"
	"testing"

	sheinproduct "task-processor/internal/shein/api/product"
)

func TestNormalizeSubmitImagesRepairsMissingSKUImageFromSKC(t *testing.T) {
	t.Parallel()

	product := &sheinproduct.Product{
		SKCList: []sheinproduct.SKC{{
			ImageInfo: sheinproduct.ImageInfo{
				ImageInfoList: []sheinproduct.ImageDetail{
					{ImageType: 1, ImageSort: 1, ImageURL: "https://img.example.com/skc-main.jpg"},
					{ImageType: 2, ImageSort: 2, ImageURL: "https://img.example.com/skc-gallery.jpg"},
				},
			},
			SKUS: []sheinproduct.SKU{{SupplierSKU: "SKU-1"}},
		}},
	}

	normalizeSubmitImages(product)

	got := product.SKCList[0].SKUS[0].ImageInfo
	if got == nil || len(got.ImageInfoList) != 1 {
		t.Fatalf("sku image info = %+v, want single fallback image", got)
	}
	if got.ImageInfoList[0].ImageURL != "https://img.example.com/skc-main.jpg" {
		t.Fatalf("sku image URL = %q, want SKC main image", got.ImageInfoList[0].ImageURL)
	}
	if got.ImageInfoList[0].ImageType != 1 || got.ImageInfoList[0].ImageSort != 1 {
		t.Fatalf("sku image detail = %+v, want type=1 sort=1", got.ImageInfoList[0])
	}
	if got.OriginalImageInfoList == nil {
		t.Fatal("sku original image info list is nil, want empty list pointer")
	}
}

func TestNormalizeSubmitGalleryImagesAddsSquareAndOptionalColorBlock(t *testing.T) {
	t.Parallel()

	images := []sheinproduct.ImageDetail{
		{ImageType: 1, ImageSort: 9, ImageURL: "https://img.example.com/main.jpg", MarketingMainImage: true, SizeImgFlag: true, TransformCVSizeImage: true},
		{ImageType: 2, ImageSort: 8, ImageURL: "https://img.example.com/detail.jpg"},
		{ImageType: 6, ImageSort: 7, ImageURL: "https://img.example.com/color.jpg"},
		{ImageType: 2, ImageSort: 6, ImageURL: "https://img.example.com/detail.jpg"},
	}

	normalized := normalizeSubmitGalleryImages(images, true)

	if len(normalized) != 4 {
		t.Fatalf("normalized images len = %d, want 4", len(normalized))
	}
	wantTypes := []int{1, 2, 5, 6}
	for index, wantType := range wantTypes {
		if normalized[index].ImageType != wantType {
			t.Fatalf("image %d type = %d, want %d in %+v", index, normalized[index].ImageType, wantType, normalized)
		}
		if normalized[index].ImageSort != index+1 {
			t.Fatalf("image %d sort = %d, want %d", index, normalized[index].ImageSort, index+1)
		}
		if normalized[index].MarketingMainImage || normalized[index].SizeImgFlag || normalized[index].TransformCVSizeImage {
			t.Fatalf("image %d flags = %+v, want submit flags disabled", index, normalized[index])
		}
		if normalized[index].PSTypes == nil {
			t.Fatalf("image %d ps types nil, want empty slice", index)
		}
	}
	if normalized[3].ImageURL != "https://img.example.com/color.jpg" {
		t.Fatalf("color block URL = %q, want explicit color source", normalized[3].ImageURL)
	}
}

func TestBuildSubmitSiteDetailImageInfoListPrefersDetailImages(t *testing.T) {
	t.Parallel()

	items := buildSubmitSiteDetailImageInfoList([]sheinproduct.ImageDetail{
		{ImageType: 1, ImageURL: "https://img.example.com/main.jpg"},
		{ImageType: 2, ImageURL: "https://img.example.com/detail-1.jpg"},
		{ImageType: 2, ImageURL: "https://img.example.com/detail-2.jpg"},
		{ImageType: 5, ImageURL: "https://img.example.com/square.jpg"},
	})

	if len(items) != 1 || len(items[0].ImageInfoList) != 2 {
		t.Fatalf("site detail image info = %+v, want two detail images", items)
	}
	if items[0].ImageInfoList[0].ImageURL != "https://img.example.com/detail-1.jpg" || items[0].ImageInfoList[0].ImageSort != 1 {
		t.Fatalf("first detail image = %+v, want detail-1 sort 1", items[0].ImageInfoList[0])
	}
	if items[0].ImageInfoList[1].ImageURL != "https://img.example.com/detail-2.jpg" || items[0].ImageInfoList[1].ImageSort != 2 {
		t.Fatalf("second detail image = %+v, want detail-2 sort 2", items[0].ImageInfoList[1])
	}
}

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

	if got := deriveSubmitProductSupplierCode(product); got != "MG8089003001-4F2669C9" {
		t.Fatalf("DeriveSubmitProductSupplierCode() = %q, want derived supplier code", got)
	}

	product.SupplierCode = "SUPPLIER-CODE"
	if got := deriveSubmitProductSupplierCode(product); got != "SUPPLIER-CODE" {
		t.Fatalf("DeriveSubmitProductSupplierCode(existing) = %q, want existing supplier code", got)
	}

	if got := deriveSubmitProductSupplierCode(nil); got != "" {
		t.Fatalf("DeriveSubmitProductSupplierCode(nil) = %q, want empty", got)
	}
}

func TestValidatePreparedProductPublishPayloadRequiresNormalizedSKUFields(t *testing.T) {
	t.Parallel()

	product := preparedPublishProduct()
	product.SKCList[0].SKUS[0].QuantityInfo = nil
	if err := validatePreparedProductPublishPayload(product); err == nil || err.Error() != "SHEIN publish blocked: SKC[0] SKU[0] is missing quantity_info" {
		t.Fatalf("ValidatePreparedProductPublishPayload(no quantity) error = %v, want quantity error", err)
	}

	product = preparedPublishProduct()
	product.SKCList[0].SKUS[0].PackageType = 0
	if err := validatePreparedProductPublishPayload(product); err == nil || err.Error() != "SHEIN publish blocked: SKC[0] SKU[0] is missing package_type" {
		t.Fatalf("ValidatePreparedProductPublishPayload(no package type) error = %v, want package type error", err)
	}

	product = preparedPublishProduct()
	product.SKCList[0].SKUS[0].StockInfoList = nil
	if err := validatePreparedProductPublishPayload(product); err == nil || err.Error() != "SHEIN publish blocked: SKC[0] SKU[0] is missing stock_info_list" {
		t.Fatalf("ValidatePreparedProductPublishPayload(no stock info) error = %v, want stock info error", err)
	}

	product = preparedPublishProduct()
	product.SKCList[0].SKUS[0].Length = ""
	if err := validatePreparedProductPublishPayload(product); err == nil || err.Error() != "SHEIN publish blocked: SKC[0] SKU[0] is missing package dimensions" {
		t.Fatalf("ValidatePreparedProductPublishPayload(no dimensions) error = %v, want dimensions error", err)
	}

	product = preparedPublishProduct()
	if err := validatePreparedProductPublishPayload(product); err != nil {
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

func TestPrepareProductForValidationPreservesIdentityAndNormalizesTransport(t *testing.T) {
	makeProduct := func() *sheinproduct.Product {
		return &sheinproduct.Product{SPUName: "title", SKCList: []sheinproduct.SKC{{SKUS: []sheinproduct.SKU{{SupplierSKU: "sku"}}}}}
	}
	for _, identity := range []string{"", "given-identity"} {
		product := makeProduct()
		product.PointKey = identity
		PrepareProductForValidation(product)
		if product.PointKey != identity {
			t.Fatal("validation changed submission identity")
		}
		if product.SPUName != "" || product.SourceSystem != "listingkit" || product.SupplierCode == "" {
			t.Fatalf("normalization = %+v", product)
		}
		sku := product.SKCList[0].SKUS[0]
		if sku.SupplierSKU != "sku" || sku.WeightUnit != "g" || sku.LengthUnit != "Inch" {
			t.Fatalf("SKU transport defaults = %+v", sku)
		}
		if strings.TrimSpace(product.SupplierCode) != product.SupplierCode {
			t.Fatal("supplier code is not normalized")
		}
	}
	PrepareProductForValidation(nil)
}

func TestEnsureSubmitSitesNormalizesUSDefaults(t *testing.T) {
	t.Parallel()

	product := &sheinproduct.Product{
		SiteList: []sheinproduct.SiteInfo{{
			MainSite:    "US",
			SubSiteList: []string{"US"},
		}},
	}

	ensureSubmitSites(product, payloadDefaults{Site: "US"})

	if len(product.SiteList) != 1 {
		t.Fatalf("site list len = %d, want 1", len(product.SiteList))
	}
	if product.SiteList[0].MainSite != "shein" {
		t.Fatalf("main site = %q, want shein", product.SiteList[0].MainSite)
	}
	if len(product.SiteList[0].SubSiteList) != 1 || product.SiteList[0].SubSiteList[0] != "shein-us" {
		t.Fatalf("sub sites = %#v, want shein-us", product.SiteList[0].SubSiteList)
	}
}

func TestEnsureSubmitSKUsNormalizesStockQuantityDimensionsAndPriceSite(t *testing.T) {
	t.Parallel()

	stockCount := 7
	supplierCode := "SKC-1"
	product := &sheinproduct.Product{
		SKCList: []sheinproduct.SKC{{
			SupplierCode: &supplierCode,
			SaleAttribute: sheinproduct.SaleAttribute{
				PreFillSpec:        true,
				IsSPPSaleAttribute: true,
			},
			SKUS: []sheinproduct.SKU{{
				StockCount:    &stockCount,
				Weight:        1,
				WeightUnit:    "lb",
				PriceInfoList: []sheinproduct.PriceInfo{{SubSite: "US"}},
			}},
		}},
	}

	ensureSubmitSKUs(product, payloadDefaults{WarehouseCode: "WH-CA-1,WH-US-1"})

	skc := product.SKCList[0]
	if skc.SupplierCode != nil {
		t.Fatalf("supplier code = %v, want nil", *skc.SupplierCode)
	}
	if skc.SaleAttribute.PreFillSpec || skc.SaleAttribute.IsSPPSaleAttribute {
		t.Fatalf("sale attribute flags = %+v, want disabled", skc.SaleAttribute)
	}
	if skc.ShelfWay != defaultSubmitSKCShelfWay {
		t.Fatalf("shelf way = %d, want %d", skc.ShelfWay, defaultSubmitSKCShelfWay)
	}

	sku := skc.SKUS[0]
	if sku.StockCount != nil {
		t.Fatalf("stock count = %v, want nil", *sku.StockCount)
	}
	if len(sku.StockInfoList) != 1 || sku.StockInfoList[0].InventoryNum != 7 || sku.StockInfoList[0].MerchantWarehouseCode != "WH-CA-1" {
		t.Fatalf("stock info = %+v, want inventory 7 in WH-CA-1", sku.StockInfoList)
	}
	if sku.QuantityInfo == nil || *sku.QuantityInfo.Quantity != 1 || *sku.QuantityInfo.QuantityType != 1 || *sku.QuantityInfo.QuantityUnit != 1 {
		t.Fatalf("quantity info = %+v, want all ones", sku.QuantityInfo)
	}
	if sku.PackageType != 3 || sku.StopPurchase != 1 {
		t.Fatalf("package/stop = %d/%d, want 3/1", sku.PackageType, sku.StopPurchase)
	}
	if sku.LengthUnit != "Inch" || sku.Length != "1" || sku.Width != "1" || sku.Height != "1" {
		t.Fatalf("dimensions = %q/%q/%q/%q, want Inch/1/1/1", sku.LengthUnit, sku.Length, sku.Width, sku.Height)
	}
	if sku.WeightUnit != "g" || sku.Weight != 453.59 {
		t.Fatalf("weight = %.2f %s, want 453.59 g", sku.Weight, sku.WeightUnit)
	}
	if len(sku.PriceInfoList) != 1 || sku.PriceInfoList[0].SubSite != "shein-us" {
		t.Fatalf("price site = %+v, want shein-us", sku.PriceInfoList)
	}
	if sku.CompetingCostPriceImages == nil || sku.SaleAttributeList == nil {
		t.Fatalf("normalized slices should be non-nil: %+v", sku)
	}
}

func TestNormalizeSubmitWeightClampsAndConvertsUnits(t *testing.T) {
	t.Parallel()

	sku := &sheinproduct.SKU{Weight: 2, WeightUnit: "kg"}
	normalizeSubmitWeight(sku)
	if sku.Weight != 2000 || sku.WeightUnit != "g" {
		t.Fatalf("weight = %.2f %s, want 2000 g", sku.Weight, sku.WeightUnit)
	}

	sku = &sheinproduct.SKU{Weight: 0, WeightUnit: "g"}
	normalizeSubmitWeight(sku)
	if sku.Weight != 0.01 || sku.WeightUnit != "g" {
		t.Fatalf("min weight = %.2f %s, want %.2f g", sku.Weight, sku.WeightUnit, 0.01)
	}
}
