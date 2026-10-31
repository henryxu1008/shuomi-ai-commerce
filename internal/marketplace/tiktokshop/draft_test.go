package tiktokshop

import (
	"fmt"
	"strings"
	"task-processor/internal/product/asset"
	"testing"
)

func TestDraftRequiresConditionalUSAttributesApprovedImagesAndExplicitStockBudget(t *testing.T) {
	yes, no := true, false
	rules := RuleSnapshot{Selection: RuleSelection{Method: GlobalPublishing, CategoryID: "cat"}, Listing: ListingRules{Methods: []ListingMethod{GlobalPublishing}, Fingerprint: "rules"}, Category: Category{ID: "cat", Leaf: &yes, Permissions: []string{"AVAILABLE"}}, Rules: CategoryRules{SizeChart: &SizeChartRule{Required: &no, Supported: &no}}, Attributes: []AttributeRule{{ID: "battery", Type: "PRODUCT_PROPERTY", Required: &no, Multiple: &no, Customizable: &no, Values: []AttributeValue{{ID: "yes", Name: "Yes"}}}, {ID: "battery-type", Type: "PRODUCT_PROPERTY", Required: &no, Multiple: &no, Customizable: &no, Values: []AttributeValue{{ID: "lithium", Name: "Lithium"}}, Conditions: []Condition{{Region: "US", Type: "VALUE_ID_MATCH", AttributeID: "battery", ValueID: "yes"}}}}, Warehouses: []Warehouse{{ID: "us-wh", Type: "SALES_WAREHOUSE", EffectStatus: "ENABLED", SubType: "CB_DIRECT_SHIPPING_WAREHOUSE"}}, GlobalWarehouses: []GlobalWarehouse{{ID: "global-wh", Ownership: "SELLER"}}}
	input := DraftInput{Selection: rules.Selection, Title: "Battery pack", Description: "A rechargeable battery pack", SaveMode: "LISTING", MainImages: []string{"approved-image"}, PackageWeight: Weight{Value: "0.500", Unit: "KILOGRAM"}, SKUs: []SKUInput{{SellerSKU: "BATTERY1", SalePrice: "10.00", GlobalBasePrice: "8.00", Inventory: []InventoryInput{{WarehouseID: "us-wh", Quantity: 5}}, GlobalInventory: []GlobalInventoryInput{{WarehouseID: "global-wh", Quantity: 5}}}}, Attributes: []ProductAttributeInput{{ID: "battery", Values: []AttributeValue{{ID: "yes"}}}}}
	inventory := asset.ApprovedAssetInventory{Scope: asset.InventoryScope{TargetPlatform: "tiktokshop"}, Assets: []asset.ApprovedAsset{{ID: "approved-image", Role: asset.RoleMain, URL: "https://example.org/image.png"}}}
	images := []ImageObservation{{AssetID: "approved-image", UseCase: "MAIN_IMAGE", SourceURL: "https://example.org/image.png", ContentHash: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", Width: 600, Height: 600, Bytes: 1000, MediaType: "image/png"}}
	result := BuildDraft(input, rules, inventory, images)
	if result.ReadyForUpload {
		t.Fatal("conditional US battery attribute bypassed")
	}
	input.Attributes = append(input.Attributes, ProductAttributeInput{ID: "battery-type", Values: []AttributeValue{{ID: "lithium"}}})
	result = BuildDraft(input, rules, inventory, images)
	if !result.ReadyForUpload {
		t.Fatalf("ordinary validated draft not ready: %v", result.Issues)
	}
	input.Description = strings.Repeat("&", 2000)
	if BuildDraft(input, rules, inventory, images).ReadyForUpload {
		t.Fatal("escaped HTML exceeds the official description length")
	}
	input.Description = "A rechargeable battery pack"
	input.SKUs[0].Inventory[0].Quantity = 6
	if BuildDraft(input, rules, inventory, images).ReadyForUpload {
		t.Fatal("US inventory exceeds approved global stock")
	}
	input.SKUs[0].Inventory[0].Quantity = 5
	if BuildDraft(input, rules, asset.ApprovedAssetInventory{}, images).ReadyForUpload {
		t.Fatal("unapproved image accepted")
	}
}

func TestCustomNumericAttributePreservesOfficialPrecision(t *testing.T) {
	no, yes := false, true
	input := DraftInput{Attributes: []ProductAttributeInput{{ID: "capacity", Values: []AttributeValue{{Name: "0.00001"}}}}}
	rules := RuleSnapshot{Attributes: []AttributeRule{{ID: "capacity", Type: "PRODUCT_PROPERTY", Required: &no, Multiple: &no, Customizable: &yes, ValueDataFormat: "POSITIVE_INT_OR_DECIMAL"}}}
	var issues []string
	validateAttributes(input, rules, func(_, code string) { issues = append(issues, code) })
	if len(issues) > 0 {
		t.Fatalf("weight precision must not restrict numeric product attributes: %v", issues)
	}
}

func TestRequiredSalesPropertiesApplyToEverySKU(t *testing.T) {
	yes, no := true, false
	for _, tc := range []struct {
		name       string
		required   *bool
		regions    []string
		conditions []Condition
		want       string
	}{
		{name: "required", required: &yes, want: "required"},
		{name: "US region", required: &no, regions: []string{"US"}, want: "required"},
		{name: "US condition", required: &no, conditions: []Condition{{Region: "US", Type: "VALUE_ID_MATCH", AttributeID: "battery", ValueID: "yes"}}, want: "required"},
		{name: "other region", required: &no, regions: []string{"GB"}},
		{name: "unknown required", want: "unknown_requirement"},
		{name: "unknown condition", required: &no, conditions: []Condition{{Region: "US", Type: "UNSUPPORTED"}}, want: "unknown_requirement"},
	} {
		for _, count := range []int{1, 2} {
			t.Run(fmt.Sprintf("%s/%d", tc.name, count), func(t *testing.T) {
				rules := RuleSnapshot{Attributes: []AttributeRule{{ID: "color", Type: "SALES_PROPERTY", Required: tc.required, RequiredRegions: tc.regions, Conditions: tc.conditions, Values: []AttributeValue{{ID: "red"}, {ID: "blue"}}}}}
				input := DraftInput{Selection: RuleSelection{Method: GlobalPublishing}, Attributes: []ProductAttributeInput{{ID: "battery", Values: []AttributeValue{{ID: "yes"}}}}}
				for n := 0; n < count; n++ {
					input.SKUs = append(input.SKUs, SKUInput{SellerSKU: fmt.Sprintf("sku%d", n)})
				}
				check := func() []string {
					var found []string
					validateSKUs(input, rules, func(field, code string) {
						if field == "salesAttributes" && (code == "required" || code == "unknown_requirement") {
							found = append(found, code)
						}
					})
					return found
				}
				found := check()
				if tc.want == "" {
					if len(found) != 0 {
						t.Fatalf("optional property required: %v", found)
					}
				} else if len(found) != count || found[0] != tc.want {
					t.Fatalf("missing sales property was not rejected for every SKU: %v", found)
				}
				if tc.want == "required" {
					for n := range input.SKUs {
						input.SKUs[n].SalesAttributes = []SalesAttributeInput{{ID: "color", ValueID: []string{"red", "blue"}[n]}}
					}
					if found = check(); len(found) != 0 {
						t.Fatalf("supplied required properties rejected: %v", found)
					}
					if count == 2 {
						input.SKUs[1].SalesAttributes = nil
						if found = check(); len(found) != 1 {
							t.Fatalf("one missing SKU must fail: %v", found)
						}
					}
				}
			})
		}
	}
}
