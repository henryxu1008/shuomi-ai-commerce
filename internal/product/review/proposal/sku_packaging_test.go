package proposal

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"task-processor/internal/product/catalog"
	"task-processor/internal/product/sourcing"
)

func TestSKUPackagingProposalCopiesCannotChangeOriginalFacts(t *testing.T) {
	var variants []catalog.Variant
	if err := json.Unmarshal([]byte(`[{"source_id":"red","sku":"RED","packaging":{"weightGrams":10}},{"source_id":"blue","sku":"BLUE","packaging":{"weightGrams":20}}]`), &variants); err != nil { t.Fatal(err) }
	var source sourcing.SourceEnvelope
	if err := json.Unmarshal([]byte(`{"ProductCandidate":{"Variants":[{"SourceID":"red","Packaging":{"weightGrams":10}},{"SourceID":"blue","Packaging":{"weightGrams":20}}]}}`), &source); err != nil { t.Fatal(err) }
	assertCopy := func(name string, original, copied any, row reflect.Value) {
		t.Helper()
		before, err := json.Marshal(original)
		if err != nil { t.Fatal(err) }
		encoded, err := json.Marshal(copied)
		if err != nil || !strings.Contains(string(encoded), `"weightGrams":10`) { t.Fatalf("%s lost packaging: %s (%v)", name, encoded, err) }
		row.FieldByName("Packaging").Elem().FieldByName("WeightGrams").Elem().SetFloat(999)
		after, err := json.Marshal(original)
		if err != nil || string(before) != string(after) { t.Fatalf("%s shares packaging with original: %s", name, after) }
	}
	copyVariants := cloneVariants(variants)
	assertCopy("cloneVariants", variants, copyVariants, reflect.ValueOf(&copyVariants[0]).Elem())
	copySource := cloneSource(source)
	assertCopy("cloneSource", source, copySource, reflect.ValueOf(&copySource.ProductCandidate.Variants[0]).Elem())
	normalized := source.Normalize()
	assertCopy("Normalize", source, normalized, reflect.ValueOf(&normalized.ProductCandidate.Variants[0]).Elem())
}
