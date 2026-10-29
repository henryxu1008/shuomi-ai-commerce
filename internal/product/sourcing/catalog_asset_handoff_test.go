package sourcing

import (
	"encoding/json"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestToSnapshotDoesNotInferADeclaredPriceFromSupplierCost(t *testing.T) {
	for _, own := range []bool{false, true} {
		t.Run(map[bool]string{false: "unknown non-own price", true: "declared own zero"}[own], func(t *testing.T) {
			identity := SourceIdentity{SourceType: SourceTypeManualImport, SourcePlatform: "controlled", SourceID: "source"}
			if own {
				identity.SourceType, identity.SourcePlatform = "user_input", "own_product"
			}
			envelope := SourceEnvelope{Identity: identity, ProductCandidate: ProductCandidate{Title: "Product", Variants: []ProductVariantCandidate{{SourceID: "sku", Currency: "CNY"}}}, SupplierOrCostFacts: SupplierOrCostFacts{Currency: "CNY", Cost: "12.50"}}
			snapshot, err := ToSnapshot(envelope)
			require.NoError(t, err)
			raw, err := json.Marshal(snapshot)
			require.NoError(t, err)
			if own {
				require.NotNil(t, snapshot.Variants[0].Price)
				require.Zero(t, snapshot.Variants[0].Price.Amount)
				require.Equal(t, 12.5, snapshot.Variants[0].Price.CostPrice)
				require.Contains(t, string(raw), `"amount":0`)
			} else {
				require.Nil(t, snapshot.Variants[0].Price)
				require.NotContains(t, string(raw), `"amount"`)
			}
			require.Equal(t, "12.50", envelope.SupplierOrCostFacts.Cost)
		})
	}
}

func TestToSnapshotPersistsImageAssetCandidates(t *testing.T) {
	snapshot, err := ToSnapshot(SourceEnvelope{
		Identity:         SourceIdentity{SourceType: SourceTypeCrawler, SourcePlatform: "1688", SourceID: "777"},
		ProductCandidate: ProductCandidate{Title: "Lunch Bag"},
		AssetCandidates: []AssetCandidate{
			{SourceID: "main", URL: "https://cdn.example.test/main.jpg", MediaType: "image", Role: "primary", Checksum: "sha-main", Width: 1600, Height: 1200},
			{SourceID: "video", URL: "https://cdn.example.test/demo.mp4", MediaType: "video", Role: "video"},
		},
	})
	if err != nil {
		t.Fatalf("ToSnapshot() error = %v", err)
	}
	if len(snapshot.Images) != 1 {
		t.Fatalf("snapshot.Images = %#v, want one image candidate", snapshot.Images)
	}
	if got := snapshot.Images[0]; got.URL != "https://cdn.example.test/main.jpg" || got.Role != "primary" || got.Width != 1600 || got.Height != 1200 {
		t.Fatalf("snapshot.Images[0] = %#v, want persisted image identity", got)
	}
}

func TestToSnapshotPreservesSupplierCostSeparatelyFromAmount(t *testing.T) {
	snapshot, err := ToSnapshot(SourceEnvelope{
		Identity: SourceIdentity{SourceType: SourceTypeCrawler, SourcePlatform: "1688", SourceID: "777"},
		ProductCandidate: ProductCandidate{
			Title: "Lunch Bag",
			Variants: []ProductVariantCandidate{{
				SourceID: "SKU-1", SKU: "SKU-1", Price: 19.9, Stock: 20,
			}},
		},
		SupplierOrCostFacts: SupplierOrCostFacts{Currency: "CNY", Cost: "12.5"},
	})
	if err != nil {
		t.Fatalf("ToSnapshot() error = %v", err)
	}
	if len(snapshot.Variants) != 1 || snapshot.Variants[0].Price == nil {
		t.Fatalf("snapshot variant = %+v, want typed price", snapshot.Variants)
	}
	price := snapshot.Variants[0].Price
	if price.Currency != "CNY" || price.Amount != 19.9 || price.CostPrice != 12.5 {
		t.Fatalf("variant price = %+v, want amount 19.9 with supplier cost 12.5", price)
	}
}

func TestToSnapshotPreservesTypedVariantCommerceFacts(t *testing.T) {
	snapshot, err := ToSnapshot(SourceEnvelope{
		Identity: SourceIdentity{SourceType: SourceTypeCrawler, SourcePlatform: "1688", SourceID: "777"},
		ProductCandidate: ProductCandidate{
			Title: "Lunch Bag",
			Variants: []ProductVariantCandidate{{
				SourceID: "SKU-1", SKU: "SKU-1", Currency: "CNY", Price: 19.9, Stock: 20,
			}},
		},
	})
	if err != nil {
		t.Fatalf("ToSnapshot() error = %v", err)
	}
	if len(snapshot.Variants) != 1 || snapshot.Variants[0].Price == nil || snapshot.Variants[0].Price.Currency != "CNY" || snapshot.Variants[0].Price.Amount != 19.9 || snapshot.Variants[0].Stock != 20 {
		t.Fatalf("snapshot variant = %+v, want typed price and stock", snapshot.Variants)
	}
}
