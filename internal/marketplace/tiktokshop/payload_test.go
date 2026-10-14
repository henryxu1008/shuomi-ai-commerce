package tiktokshop

import (
	"encoding/json"
	"testing"
)

func TestGlobalPublishRejectsDuplicateInputSKUAndPreservesUSPrices(t *testing.T) {
	created := CreatedGlobalProduct{ID: "global", ResponseHash: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", SKUs: []CreatedSKU{{ID: "g1", SellerSKU: "A"}, {ID: "g2", SellerSKU: "B"}}}
	input := DraftInput{Selection: RuleSelection{Method: GlobalPublishing}, SKUs: []SKUInput{{SellerSKU: "A", SalePrice: "10.00", Inventory: []InventoryInput{{WarehouseID: "US", Quantity: 1}}}, {SellerSKU: "A", SalePrice: "20.00", Inventory: []InventoryInput{{WarehouseID: "US", Quantity: 1}}}}}
	if _, err := BuildGlobalPublish(input, created); err == nil {
		t.Fatal("duplicate SKU was published instead of exact global membership")
	}
	input.SKUs[1].SellerSKU = "B"
	result, err := BuildGlobalPublish(input, created)
	if err != nil || len(result.Targets) != 1 || result.Targets[0].Region != "US" || result.Targets[0].SKUs[1].Price.SalePrice != "20.00" {
		t.Fatalf("US publication mapping: %v %+v", err, result)
	}
}

func TestGlobalPublishInventoryIsOneExplicitOfficialObject(t *testing.T) {
	input := DraftInput{Selection: RuleSelection{Method: GlobalPublishing}, SKUs: []SKUInput{{SellerSKU: "A", SalePrice: "10.00", Inventory: []InventoryInput{{WarehouseID: "US", Quantity: 3}}}}}
	created := CreatedGlobalProduct{ID: "global", ResponseHash: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", SKUs: []CreatedSKU{{ID: "g", SellerSKU: "A"}}}
	result, err := BuildGlobalPublish(input, created)
	if err != nil {
		t.Fatal(err)
	}
	raw, _ := json.Marshal(result)
	var body struct {
		Targets []struct {
			SKUs []struct {
				Inventory map[string]any `json:"inventory"`
			} `json:"skus"`
		} `json:"publish_target"`
	}
	if json.Unmarshal(raw, &body) != nil || body.Targets[0].SKUs[0].Inventory["warehouse_id"] != "US" {
		t.Fatalf("official publish inventory is an object: %s", raw)
	}
	input.SKUs[0].Inventory = append(input.SKUs[0].Inventory, InventoryInput{WarehouseID: "US2", Quantity: 3})
	if _, err = BuildGlobalPublish(input, created); err == nil {
		t.Fatal("multiple local inventory objects silently rewritten")
	}
}
