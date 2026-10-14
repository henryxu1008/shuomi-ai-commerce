package materials

import (
	"encoding/json"
	"reflect"
	"testing"

	record "task-processor/internal/listing/record/target"
	"task-processor/internal/marketplace/shein/goods"
	model "task-processor/internal/marketplace/shein/model"
)

func TestMaterialProposalStrictlyExcludesCommercialAndContentFields(t *testing.T) {
	for _, raw := range []string{
		`{"categoryId":1,"price":1,"evidenceIds":["observed"]}`,
		`{"categoryId":1,"title":"new title","evidenceIds":["observed"]}`,
		`{"categoryId":1,"categoryId":2,"evidenceIds":["observed"]}`,
		`{"skcs":[{"index":0,"skus":[{"index":0,"stock":0}]}],"evidenceIds":["observed"]}`,
		`{"skcs":[{"skus":[{"index":0,"attributes":[]}]}],"evidenceIds":["observed"]}`,
		`{"categoryId":1,"evidenceIds":["observed","observed"]}`,
		`{"categoryId":1,"evidenceIds":[]}`,
		`null`,
	} {
		if _, err := ParseProposal(json.RawMessage(raw)); err == nil {
			t.Fatalf("invalid or unauthorized proposal admitted: %s", raw)
		}
	}
}

func TestMaterialProposalPreservesSKUIdentityPricesStockAndContent(t *testing.T) {
	base := record.TargetRecord{Revision: 4, Input: record.TargetInput{SourceID: "source", StoreID: "store", EffectiveVersion: 2, ApplyReceiptID: "approved-content", Draft: goods.OfficialDraftInput{Product: model.PublishProduct{
		CategoryID: 12, BrandCode: "old-brand", Names: []model.LanguageContent{{Language: "en", Name: "Approved title"}}, Descriptions: []model.LanguageContent{{Language: "en", Name: "Approved description"}},
		SKCs: []model.ProductSKC{{SupplierCode: "merchant-skc", SKUs: []model.ProductSKU{{SupplierSKU: "merchant-sku", Prices: []model.ProductPrice{{BasePrice: 12.5, Currency: "USD", SubSite: "shein-us"}}, Stock: []model.ProductStock{{Quantity: 17, WarehouseID: "warehouse"}}, Length: "10", Weight: floatPointer(45)}}}},
	}}}}
	before, _ := json.Marshal(base)
	proposal, err := ParseProposal(json.RawMessage(`{"brandCode":"known-brand","skcs":[{"index":0,"skus":[{"index":0,"attributes":[{"attribute_id":5,"attribute_value_id":8}]}]}],"evidenceIds":["observed"]}`))
	if err != nil {
		t.Fatal(err)
	}
	input, err := ApplyProposal(base, proposal)
	if err != nil {
		t.Fatal(err)
	}
	if input.ExpectedRevision != 4 || input.SourceID != base.Input.SourceID || input.StoreID != base.Input.StoreID || input.ApplyReceiptID != base.Input.ApplyReceiptID || input.EffectiveVersion != base.Input.EffectiveVersion {
		t.Fatal("proposal replaced owner-qualified source/version/CAS")
	}
	sku := input.Draft.Product.SKCs[0].SKUs[0]
	original := base.Input.Draft.Product.SKCs[0].SKUs[0]
	if sku.SupplierSKU != original.SupplierSKU || !reflect.DeepEqual(sku.Prices, original.Prices) || !reflect.DeepEqual(sku.Stock, original.Stock) || sku.Length != original.Length || *sku.Weight != *original.Weight ||
		!reflect.DeepEqual(input.Draft.Product.Names, base.Input.Draft.Product.Names) || !reflect.DeepEqual(input.Draft.Product.Descriptions, base.Input.Draft.Product.Descriptions) {
		t.Fatal("platform proposal changed protected SKU/commercial/content facts")
	}
	if input.Draft.Product.BrandCode != "known-brand" || len(sku.Attributes) != 1 {
		t.Fatal("permitted platform fields were not applied")
	}
	input.Draft.Product.SKCs[0].SKUs[0].Stock[0].Quantity = 999
	after, _ := json.Marshal(base)
	if string(before) != string(after) {
		t.Fatal("proposal application mutated its original owner record")
	}
}

func TestMaterialProposalCannotInventSKUOrGuessNewCategoryAttributes(t *testing.T) {
	base := record.TargetRecord{Revision: 1, Input: record.TargetInput{Draft: goods.OfficialDraftInput{Product: model.PublishProduct{CategoryID: 12, SKCs: []model.ProductSKC{{SKUs: []model.ProductSKU{{SupplierSKU: "sku"}}}}}}}}
	for _, raw := range []string{
		`{"skcs":[{"index":1,"saleAttribute":{"attribute_id":1}}],"evidenceIds":["observed"]}`,
		`{"skcs":[{"index":0,"skus":[{"index":1,"attributes":[]}]}],"evidenceIds":["observed"]}`,
		`{"categoryId":99,"attributes":[{"attribute_id":5}],"evidenceIds":["observed"]}`,
		`{"categoryId":99,"skcs":[{"index":0,"saleAttribute":{"attribute_id":1}}],"evidenceIds":["observed"]}`,
		`{"categoryId":99,"brandCode":"known-brand","evidenceIds":["observed"]}`,
		`{"categoryId":99,"images":[],"evidenceIds":["observed"]}`,
	} {
		proposal, err := ParseProposal(json.RawMessage(raw))
		if err != nil {
			t.Fatal(err)
		}
		if _, err = ApplyProposal(base, proposal); err == nil {
			t.Fatalf("unobserved SKU/category fields admitted: %s", raw)
		}
	}
	proposal, err := ParseProposal(json.RawMessage(`{"categoryId":99,"evidenceIds":["observed"]}`))
	if err != nil {
		t.Fatal(err)
	}
	input, err := ApplyProposal(base, proposal)
	if err != nil {
		t.Fatal(err)
	}
	expected := base.Input
	expected.ExpectedRevision = base.Revision
	expected.Draft.Product.CategoryID = 99
	if !reflect.DeepEqual(expected, input) {
		t.Fatal("category-only proposal changed companion fields")
	}
}

func floatPointer(v float64) *float64 { return &v }
