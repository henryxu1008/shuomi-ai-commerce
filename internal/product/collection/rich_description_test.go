package collection

import (
	"encoding/json"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"task-processor/internal/product/sourcing"
	"testing"
)

func TestOwnDescriptionFormatSurvivesSourceAndSnapshot(t *testing.T) {
	var in OwnProduct
	require.NoError(t, json.Unmarshal([]byte(`{"title":"rich","description":"<p><strong>text</strong><img src=\"https://images.example.org/a.png\"></p>","descriptionFormat":"html-v1","images":["https://images.example.org/a.png"]}`), &in))
	envelope, err := OwnEnvelope(uuid.NewString(), in)
	require.NoError(t, err)
	snapshot, err := sourcing.ToSnapshot(envelope)
	require.NoError(t, err)
	raw, err := json.Marshal(snapshot)
	require.NoError(t, err)
	require.Contains(t, string(raw), `"description_format":"html-v1"`)
	in.Images = []string{}
	_, err = OwnEnvelope(uuid.NewString(), in)
	require.Error(t, err)
}
func TestOwnDescriptionRejectsUnsafeDirectInput(t *testing.T) {
	for _, body := range []string{`{"title":"x","description":"<script>x</script>","descriptionFormat":"html-v1","images":[]}`, `{"title":"x","description":"x","descriptionFormat":"unknown","images":[]}`} {
		var p OwnProduct
		require.NoError(t, json.Unmarshal([]byte(body), &p))
		_, err := OwnEnvelope(uuid.NewString(), p)
		require.Error(t, err)
	}
}

func TestDescriptionMayUseExactVariantImagePool(t *testing.T) {
	image := "https://images.example.org/sku.png"
	p := OwnProduct{Title: "variant", Description: `<p><img src="` + image + `"></p>`, DescriptionFormat: "html-v1", Images: []string{}, Variants: []OwnVariant{{SourceID: "sku-1", Currency: "CNY", Images: []string{image}}}}
	e, err := OwnEnvelope(uuid.NewString(), p)
	require.NoError(t, err)
	snapshot, err := sourcing.ToSnapshot(e)
	require.NoError(t, err)
	require.Equal(t, "html-v1", snapshot.DescriptionFormat)
}
