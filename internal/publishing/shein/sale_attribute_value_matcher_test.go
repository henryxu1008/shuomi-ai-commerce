package shein

import (
	"encoding/json"
	"testing"

	common "task-processor/internal/publishing/common"
)

func TestApplySaleAttributeResolutionDoesNotFallbackToAttributeOnlyPlaceholder(t *testing.T) {
	pkg := &Package{
		SkcList: []SKCPackage{
			{
				SupplierCode: "SCRAPED-SET",
				Attributes:   map[string]string{"颜色": "一桌四椅套装"},
				SKUs:         []common.Variant{{SKU: "SCRAPED-SET"}},
			},
		},
		RequestDraft: &RequestDraft{
			SKCList: []SKCRequestDraft{{
				SupplierCode: "SCRAPED-SET",
				SKUList:      []SKUDraft{{SupplierSKU: "SCRAPED-SET"}},
			}},
		},
	}
	resolution := &SaleAttributeResolution{
		PrimarySourceDimension: "颜色",
		SKCAttributes: []ResolvedSaleAttribute{{
			Scope:       "skc",
			Name:        "Color",
			AttributeID: 27,
			MatchedBy:   "attribute_name",
		}},
	}

	ApplySaleAttributeResolution(pkg, resolution)

	if pkg.RequestDraft.SKCList[0].SaleAttribute != nil {
		t.Fatalf("expected no placeholder SKC sale attribute, got %+v", pkg.RequestDraft.SKCList[0].SaleAttribute)
	}
}

func TestApplySaleAttributeResolutionUsesSerializedValueAssignments(t *testing.T) {
	valueID := 9001
	resolution := &SaleAttributeResolution{
		PrimarySourceDimension: "ai_style",
		SKCAttributes: []ResolvedSaleAttribute{{
			Scope:       "skc",
			Name:        "Style Type",
			Value:       "Style A",
			AttributeID: 1001184,
			MatchedBy:   "custom_attribute_value",
		}},
		SKCValueAssignments: map[string]ResolvedSaleAttribute{
			normalizeText("Style A"): {
				Scope:            "skc",
				Name:             "Style Type",
				Value:            "Style A",
				AttributeID:      1001184,
				AttributeValueID: &valueID,
				MatchedBy:        "custom_attribute_value",
			},
		},
	}
	data, err := json.Marshal(resolution)
	if err != nil {
		t.Fatalf("marshal resolution: %v", err)
	}
	var decoded SaleAttributeResolution
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal resolution: %v", err)
	}
	pkg := &Package{
		SkcList: []SKCPackage{{
			SupplierCode: "SKC-1",
			Attributes:   map[string]string{"ai_style": "Style A"},
		}},
		RequestDraft: &RequestDraft{SKCList: []SKCRequestDraft{{
			SupplierCode: "SKC-1",
			SKUList:      []SKUDraft{{SupplierSKU: "SKU-1"}},
		}}},
	}

	ApplySaleAttributeResolution(pkg, &decoded)

	got := pkg.RequestDraft.SKCList[0].SaleAttribute
	if got == nil || got.AttributeValueID == nil || *got.AttributeValueID != 9001 {
		t.Fatalf("skc sale attribute = %+v, want serialized value id 9001", got)
	}
}
