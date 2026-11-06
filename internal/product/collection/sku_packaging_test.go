package collection

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"task-processor/internal/product/catalog"
	"task-processor/internal/product/sourcing"
)

func TestOwnSKUPackagingKeepsExactVariantAndPartialUnknown(t *testing.T) {
	var product OwnProduct
	require.NoError(t, json.Unmarshal([]byte(`{"title":"Measured","images":[],"variants":[{"sourceId":"red","sku":"RED","currency":"CNY","price":0,"stock":0,"packaging":{"weightGrams":125.5,"lengthCm":10,"widthCm":8,"heightCm":2}},{"sourceId":"blue","sku":"BLUE","currency":"CNY","price":0,"stock":7,"packaging":{"lengthCm":20,"widthCm":null}}]}`), &product))
	envelope, err := OwnEnvelope(uuid.NewString(), product)
	require.NoError(t, err)
	snapshot, err := sourcing.ToSnapshot(envelope)
	require.NoError(t, err)
	raw, err := json.Marshal(snapshot)
	require.NoError(t, err)
	require.Contains(t, string(raw), `"weightGrams":125.5`)
	canonical := catalog.ProjectCanonical(snapshot)
	require.Equal(t, "RED", canonical.Variants[0].SKU)
	require.NotNil(t, canonical.Variants[0].Weight)
	require.Equal(t, 125.5, canonical.Variants[0].Weight.Value)
	require.Equal(t, "g", canonical.Variants[0].Weight.Unit)
	require.Equal(t, "cm", canonical.Variants[0].Dimensions.Unit)
	require.Nil(t, canonical.Variants[1].Weight)
	require.Equal(t, float64(20), canonical.Variants[1].Dimensions.Length)
	require.Zero(t, canonical.Variants[1].Dimensions.Width)
	canonical.Variants[0].Weight.Value = 999
	after, err := json.Marshal(snapshot)
	require.NoError(t, err)
	require.Equal(t, string(raw), string(after), "projection must not mutate the source version")
}

func TestOwnSKUPackagingRejectsInvalidMeasurementsBeforeWrite(t *testing.T) {
	for _, number := range []string{"0", "-1", "9007199254740992"} {
		for _, field := range []string{"weightGrams", "lengthCm", "widthCm", "heightCm"} {
			t.Run(field+number, func(t *testing.T) {
				var product OwnProduct
				data := `{"title":"Measured","images":[],"variants":[{"sourceId":"sku","sku":"SKU","currency":"CNY","price":0,"stock":0,"packaging":{"` + field + `":` + number + `}}]}`
				require.NoError(t, json.Unmarshal([]byte(data), &product))
				_, err := OwnEnvelope(uuid.NewString(), product)
				require.ErrorIs(t, err, ErrInvalid)
			})
		}
	}
	for _, value := range []string{"null", "{}", `{"weightGrams":null}`, `{"heightCm":0.001}`} {
		var product OwnProduct
		data := `{"title":"Measured","images":[],"variants":[{"sourceId":"sku","sku":"SKU","currency":"CNY","price":0,"stock":0,"packaging":` + value + `}]}`
		require.NoError(t, json.Unmarshal([]byte(data), &product))
		_, err := OwnEnvelope(uuid.NewString(), product)
		require.NoError(t, err, value)
	}
}

func TestOwnSKUPackagingChangesOnlyTheNewSourceEvidence(t *testing.T) {
	const original = `{"title":"Original","images":[],"variants":[{"sourceId":"stable","sku":"SKU","currency":"CNY","price":0,"stock":0}]}`
	var before, after OwnProduct
	require.NoError(t, json.Unmarshal([]byte(original), &before))
	require.NoError(t, json.Unmarshal([]byte(strings.Replace(original, `"stock":0`, `"stock":0,"packaging":{"weightGrams":12}`, 1)), &after))
	id := uuid.NewString()
	oldEnvelope, err := OwnEnvelope(id, before)
	require.NoError(t, err)
	newEnvelope, err := OwnEnvelope(id, after)
	require.NoError(t, err)
	require.NotEqual(t, oldEnvelope.RawReference.Checksum, newEnvelope.RawReference.Checksum)
	oldSnapshot, err := sourcing.ToSnapshot(oldEnvelope)
	require.NoError(t, err)
	newSnapshot, err := sourcing.ToSnapshot(newEnvelope)
	require.NoError(t, err)
	require.NotEqual(t, Digest(oldSnapshot), Digest(newSnapshot))
	oldRaw, err := json.Marshal(oldSnapshot)
	require.NoError(t, err)
	require.NotContains(t, string(oldRaw), "packaging")
}
