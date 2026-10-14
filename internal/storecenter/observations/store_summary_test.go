package observations

import "testing"

func TestLowInventoryThresholdUnknownAndConflictingDuplicates(t *testing.T) {
	threshold := int64(5)
	five, zero, negative := int64(5), int64(0), int64(-1)
	rows := []InventoryFact{{SKU: "s1", Warehouse: "a", Quantity: &five}, {SKU: "s1", Warehouse: "a", Quantity: &five}, {SKU: "s1", Warehouse: "b", Quantity: &zero}, {SKU: "s2", Warehouse: "a"}, {SKU: "s3", Warehouse: "a", Quantity: &negative}}
	result := LowInventory(rows, &threshold)
	if result.Value == nil || *result.Value != 2 || result.Unknown != 2 || result.Complete {
		t.Fatalf("unknown or duplicates: %+v", result)
	}
	result = LowInventory(rows, nil)
	if result.Value != nil || result.Reason != "threshold_not_set" {
		t.Fatal(result)
	}
	rows = append(rows, InventoryFact{SKU: "s1", Warehouse: "a", Quantity: &zero})
	result = LowInventory(rows, &threshold)
	if result.Value == nil || *result.Value != 1 || result.Unknown != 3 {
		t.Fatalf("conflicting evidence: %+v", result)
	}
	result = LowInventory([]InventoryFact{}, &threshold)
	if result.Value == nil || *result.Value != 0 || !result.Complete {
		t.Fatalf("complete empty: %+v", result)
	}
}
