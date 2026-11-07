package materialsinspect

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"task-processor/internal/listing/materials"
	"task-processor/internal/product/collection"
)

func TestMaterialsInspectionRegistryPreservesSKUPackaging(t *testing.T) {
	for _, kind := range []string{"own", "acquisition"} {
		t.Run(kind, func(t *testing.T) {
			i, r, _, _ := inspector(t)
			product := r.value.Product
			require.NoError(t, json.Unmarshal([]byte(`{"title":"Measured","variants":[{"source_id":"red","sku":"RED","stock":0,"own_stock_declared":true,"packaging":{"weightGrams":125.5}},{"source_id":"blue","sku":"BLUE","packaging":{"lengthCm":20}}]}`), &product.Snapshot))
			base := r.value.Base
			base.Source.Source.Kind = kind
			base.ProductHash = collection.Digest(product.Snapshot)
			var err error
			r.value, err = materials.NewInspection(base, product, r.value.Rules, r.value.Inventory, time.Now().UTC(), time.Now().UTC())
			require.NoError(t, err)
			result, err := invoke(i, r.value.Binding)
			require.NoError(t, err)
			require.Contains(t, string(result.Output), `"weightGrams":125.5`)
			require.Contains(t, string(result.Output), `"lengthCm":20`)
			require.Contains(t, string(result.Output), `"stock":0`)
			require.NotContains(t, string(result.Output), "own_stock_declared")
		})
	}
}
