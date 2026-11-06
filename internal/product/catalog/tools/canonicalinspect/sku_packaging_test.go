package canonicalinspect

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestInspectionRegistryPreservesExactSKUPackaging(t *testing.T) {
	published := validPublished(1)
	if err := json.Unmarshal([]byte(`{"title":"Measured","variants":[{"source_id":"red","sku":"RED","stock":0,"own_stock_declared":true,"packaging":{"weightGrams":125.5,"lengthCm":10}}]}`), &published.Snapshot); err != nil { t.Fatal(err) }
	raw, err := Project(published)
	if err != nil { t.Fatal(err) }
	if !strings.Contains(string(raw), `"weightGrams":125.5`) { t.Fatalf("inspection lost exact package facts: %s", raw) }
	validateOutputAgainstRegistry(t, raw)
}
