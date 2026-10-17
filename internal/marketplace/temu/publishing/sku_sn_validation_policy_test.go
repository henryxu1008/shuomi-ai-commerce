package publishing

import (
	"reflect"
	"slices"
	"strings"
	"testing"
)

func TestBuildUniqueSKUSNs(t *testing.T) {
	tests := []struct {
		name     string
		mainASIN string
		variants []string
		prefix   string
		suffix   string
		want     []string
	}{
		{name: "no source identifiers"},
		{name: "empty identifiers do not become prefix-only codes", variants: []string{"", ""}, prefix: "P_", suffix: "_S"},
		{name: "main product only", mainASIN: "B001", want: []string{"B001"}},
		{name: "variants without main product", variants: []string{"B002", "B001"}, want: []string{"B002", "B001"}},
		{name: "main first and duplicates retain first occurrence", mainASIN: "B002", variants: []string{"B001", "B002", "B003", "B001", "", "B003"}, want: []string{"B002", "B001", "B003"}},
		{name: "ASIN-only strategy retains configured prefix and suffix", mainASIN: "B002", variants: []string{"B001", "B002"}, prefix: "P_", suffix: "_S", want: []string{"P_B002_S", "P_B001_S"}},
		{name: "opaque identifiers are not trimmed or case folded", mainASIN: " B001 ", variants: []string{"B001", "b001", " ", ""}, prefix: "[", suffix: "]", want: []string{"[ B001 ]", "[B001]", "[b001]", "[ ]"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			before := slices.Clone(tt.variants)
			got := buildUniqueSKUSNs(tt.mainASIN, tt.variants, tt.prefix, tt.suffix)
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("SKU codes = %v, want %v", got, tt.want)
			}
			if !reflect.DeepEqual(tt.variants, before) {
				t.Fatal("source identifiers changed")
			}
		})
	}
}

func TestBuildUniqueSKUSNsKeepsSourceStorageIndependent(t *testing.T) {
	storage := []string{"B002", "B001", "B002", "", "sentinel"}
	before := slices.Clone(storage)
	got := buildUniqueSKUSNs("B003", storage[:4], "", "")
	if !slices.Equal(got, []string{"B003", "B002", "B001"}) {
		t.Fatalf("SKU codes = %v", got)
	}
	got[0] = "changed"
	if !slices.Equal(storage, before) {
		t.Fatal("building or changing result modified source storage")
	}
}

func TestValidateSKUSNFailures(t *testing.T) {
	tests := []struct {
		name        string
		failures    []skuSNFailure
		wantFailure bool
		wantDetails []string
	}{
		{name: "nil failures pass"},
		{name: "empty failures pass", failures: []skuSNFailure{}},
		{name: "any reported failure rejects even with no details", failures: []skuSNFailure{{}}, wantFailure: true},
		{name: "all duplicate diagnostics survive", failures: []skuSNFailure{
			{outSKUSN: "SKU-A", usedGoodsID: "goods-1", usedSKUID: "sku-10", failReason: "already used"},
			{outSKUSN: "SKU-B", usedGoodsID: "goods-2", usedSKUID: "sku-20", failReason: "conflicting item"},
		}, wantFailure: true, wantDetails: []string{"SKU-A", "goods-1", "sku-10", "already used", "SKU-B", "goods-2", "sku-20", "conflicting item"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			before := slices.Clone(tt.failures)
			err := validateSKUSNFailures(tt.failures)
			if (err != nil) != tt.wantFailure {
				t.Fatalf("error = %v, want failure %v", err, tt.wantFailure)
			}
			for _, detail := range tt.wantDetails {
				if !strings.Contains(err.Error(), detail) {
					t.Errorf("error %q lost diagnostic %q", err, detail)
				}
			}
			if !reflect.DeepEqual(tt.failures, before) {
				t.Fatal("reported failures changed")
			}
		})
	}
}
