package collection

import (
	"encoding/json"
	"fmt"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"task-processor/internal/product/sourcing"
	"testing"
)

func TestOwnVariantsKeepSeparateImagesAndExplicitZeroPrice(t *testing.T) {
	var input OwnProduct
	require.NoError(t, json.Unmarshal([]byte(`{"title":"多规格上衣","images":["https://images.example.org/common.png"],"variants":[{"sourceId":"red-s","title":"红色 / S","sku":"RED-S","attributes":{"颜色":"红色","尺码":"S"},"currency":"CNY","price":0,"stock":0,"images":["https://images.example.org/red.png"]},{"sourceId":"blue-m","title":"蓝色 / M","sku":"BLUE-M","attributes":{"颜色":"蓝色","尺码":"M"},"currency":"CNY","price":12.5,"stock":8,"images":["https://images.example.org/blue.png"]}]}`), &input))
	envelope, err := OwnEnvelope(uuid.NewString(), input)
	require.NoError(t, err)
	require.Len(t, envelope.ProductCandidate.Variants[0].Images, 1)
	require.Equal(t, "https://images.example.org/red.png", envelope.ProductCandidate.Variants[0].Images[0].URL)
	require.Equal(t, "https://images.example.org/blue.png", envelope.ProductCandidate.Variants[1].Images[0].URL)
	require.NotEqual(t, envelope.ProductCandidate.Variants[0].Images[0].SourceID, envelope.ProductCandidate.Variants[1].Images[0].SourceID)
	snapshot, err := sourcing.ToSnapshot(envelope)
	require.NoError(t, err)
	require.NotNil(t, snapshot.Variants[0].Price)
	require.Equal(t, "CNY", snapshot.Variants[0].Price.Currency)
	require.Zero(t, snapshot.Variants[0].Price.Amount)
	encoded, err := json.Marshal(snapshot)
	require.NoError(t, err)
	require.Contains(t, string(encoded), `"amount":0`)
	require.Len(t, snapshot.Images, 1)
	require.Len(t, snapshot.Variants[0].Images, 1)
}

func TestOwnVariantRejectsInvalidFactsBeforeWrite(t *testing.T) {
	base := OwnVariant{SourceID: "red", Title: "red", SKU: "RED", Currency: "CNY", Price: 0, Stock: 0, Attributes: map[string]string{"color": "red"}}
	for _, test := range []struct {
		name   string
		change func(*OwnProduct)
	}{
		{"duplicate source", func(p *OwnProduct) { p.Variants = append(p.Variants, base) }},
		{"duplicate sku", func(p *OwnProduct) { v := base; v.SourceID = "blue"; p.Variants = append(p.Variants, v) }},
		{"unsafe image", func(p *OwnProduct) { p.Variants[0].Images = []string{"file:///private.jpg"} }},
		{"negative price", func(p *OwnProduct) { p.Variants[0].Price = -1 }},
		{"negative stock", func(p *OwnProduct) { p.Variants[0].Stock = -1 }},
		{"invalid currency", func(p *OwnProduct) { p.Variants[0].Currency = "cny" }},
		{"variant budget", func(p *OwnProduct) { p.Variants = make([]OwnVariant, 257) }},
		{"aggregate budget", func(p *OwnProduct) {
			p.Variants = make([]OwnVariant, 256)
			for i := range p.Variants {
				p.Variants[i] = base
				p.Variants[i].SourceID = fmt.Sprint(i)
				p.Variants[i].SKU = fmt.Sprint(i)
				p.Variants[i].Attributes = map[string]string{"a": "1", "b": "2", "c": "3", "d": "4"}
			}
		}},
	} {
		t.Run(test.name, func(t *testing.T) {
			input := OwnProduct{Title: "product", Variants: []OwnVariant{base}}
			test.change(&input)
			_, err := OwnEnvelope(uuid.NewString(), input)
			require.ErrorIs(t, err, ErrInvalid)
		})
	}
}
