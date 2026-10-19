package listingadmin

import (
	"testing"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	_ "modernc.org/sqlite"
)

func TestSharedDTOValueHelpersPreserveExplicitValues(t *testing.T) {
	for _, value := range []float64{0, -1.25, 2.5} {
		filter := FilterRuleToRespDTO(FilterRule{PriceMin: value, RatingMin: value})
		mapping := ProductImportMappingToRespDTO(&ProductImportMapping{SalePriceMultiplier: value, DiscountPriceMultiplier: value})
		for _, got := range []*float64{filter.PriceMin, filter.RatingMin, mapping.SalePriceMultiplier, mapping.DiscountPriceMultiplier} {
			if got == nil || *got != value {
				t.Fatalf("explicit value %v lost in DTO projection: %v", value, got)
			}
		}
	}
	for _, value := range []string{"", "SKU-123"} {
		got := ptrString(value)
		if got == nil || *got != value {
			t.Fatalf("explicit string %q lost: %v", value, got)
		}
	}
	when := time.Date(2026, 10, 19, 11, 0, 0, 123, time.FixedZone("store", 8*60*60))
	for _, value := range []*time.Time{nil, &when} {
		got := FilterRuleToRespDTO(FilterRule{CreateTime: value}).CreateTime
		if value == nil && !got.Time.IsZero() || value != nil && got.Time != *value {
			t.Fatalf("time projection = %#v, want %#v", got.Time, value)
		}
	}
	mapping := ProductImportMappingToRespDTO(&ProductImportMapping{SKU: "SKU-123"})
	if mapping.Sku == nil || *mapping.Sku != "SKU-123" || mapping.ParentProductId != nil {
		t.Fatalf("optional mapping strings changed: %+v", mapping)
	}
}

func TestGormProductImportMappingAPIKeepsExplicitZeroLookupScope(t *testing.T) {
	db, err := gorm.Open(sqlite.Dialector{DriverName: "sqlite", DSN: ":memory:"}, &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&listingProductImportMapping{}); err != nil {
		t.Fatalf("create product import mapping table: %v", err)
	}
	if err := AutoMigrateProductImportMappingRepository(db); err != nil {
		t.Fatalf("migrate product import mapping: %v", err)
	}
	for _, mapping := range []listingProductImportMapping{
		{ID: 1, StoreID: 0, ImportTaskID: 0, SKU: "ZERO-SCOPE", PlatformProductID: "ZERO-PLATFORM", Deleted: 0},
		{ID: 2, StoreID: 88, ImportTaskID: 99, SKU: "ZERO-SCOPE", PlatformProductID: "ZERO-PLATFORM", Deleted: 0},
	} {
		if err := db.Table("listing_product_import_mapping").Create(&mapping).Error; err != nil {
			t.Fatalf("seed mapping %#v: %v", mapping, err)
		}
	}

	api := NewGormProductImportMappingAPI(NewGormProductImportMappingRepository(db))

	bySKU, err := api.GetProductImportMappingBySku(&ProductImportMappingGetBySkuReqDTO{StoreId: 0, Sku: "ZERO-SCOPE"})
	if err != nil || bySKU == nil || bySKU.ID != 1 {
		t.Fatalf("GetProductImportMappingBySku() = %#v, %v; want zero-store mapping ID 1", bySKU, err)
	}

	byTask, err := api.GetProductImportMappingByTaskAndSku(0, "ZERO-SCOPE")
	if err != nil || byTask == nil || byTask.ID != 1 {
		t.Fatalf("GetProductImportMappingByTaskAndSku() = %#v, %v; want zero-task mapping ID 1", byTask, err)
	}

	byPlatform, err := api.GetProductImportMappingByPlatformProductIdAndStore(&ProductImportMappingGetByPlatformProductIdAndStoreReqDTO{StoreId: 0, PlatformProductId: "ZERO-PLATFORM"})
	if err != nil || byPlatform == nil || byPlatform.ID != 1 {
		t.Fatalf("GetProductImportMappingByPlatformProductIdAndStore() = %#v, %v; want zero-store mapping ID 1", byPlatform, err)
	}
}
