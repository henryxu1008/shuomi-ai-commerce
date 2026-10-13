package tiktokshop

import (
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
	input.SKUs[0].Inventory[0].Quantity = 6
	if BuildDraft(input, rules, inventory, images).ReadyForUpload {
		t.Fatal("US inventory exceeds approved global stock")
	}
	input.SKUs[0].Inventory[0].Quantity = 5
	if BuildDraft(input, rules, asset.ApprovedAssetInventory{}, images).ReadyForUpload {
		t.Fatal("unapproved image accepted")
	}
}
