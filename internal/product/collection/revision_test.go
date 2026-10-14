package collection

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestRevisionRequiresCompleteBaseAndKeepsUserEvidence(t *testing.T) {
	store := &testStore{}
	service, err := NewService(store, &testAuth{scope: Scope{"org", "actor", "member"}}, testSources{})
	require.NoError(t, err)
	// Decode the new wire contract so this test also captures its absence.
	var input Mutation
	require.NoError(t, json.Unmarshal([]byte(`{"action":"revise_product","itemId":"`+uuid.NewString()+`","expectedRevision":1,"expectedSourceVersion":"2","expectedPublicationId":"`+uuid.NewString()+`","product":{"title":"新版","images":[]}}`), &input))
	_, err = service.Mutate(context.Background(), uuid.NewString(), input)
	require.NoError(t, err)
	require.Len(t, store.commands, 1)
	require.Equal(t, "新版", store.commands[0].Envelope.ProductCandidate.Title)
	require.Equal(t, "2", store.commands[0].Envelope.RawReference.Metadata["previous_catalog_version"])
	for _, field := range []string{"expectedRevision", "expectedSourceVersion", "expectedPublicationId", "itemId", "product"} {
		raw, _ := json.Marshal(input)
		var fields map[string]json.RawMessage
		require.NoError(t, json.Unmarshal(raw, &fields))
		delete(fields, field)
		raw, _ = json.Marshal(fields)
		var invalid Mutation
		require.NoError(t, json.Unmarshal(raw, &invalid))
		_, err = service.Mutate(context.Background(), uuid.NewString(), invalid)
		require.ErrorIs(t, err, ErrInvalid, field)
	}
	input.TargetBatchID = uuid.NewString()
	_, err = service.Mutate(context.Background(), uuid.NewString(), input)
	require.ErrorIs(t, err, ErrInvalid)
	require.Len(t, store.commands, 1)
}

func TestRevisionLineageFitsExistingSourceBudget(t *testing.T) {
	product := OwnProduct{Title: "边界商品", Attributes: map[string]string{}}
	for i := range 254 {
		product.Attributes[fmt.Sprintf("a%d", i)] = "v"
	}
	for i := range 256 {
		product.Variants = append(product.Variants, OwnVariant{SourceID: fmt.Sprintf("sku%d", i), Currency: "CNY", Attributes: map[string]string{"颜色": "红", "尺码": "S"}})
	}
	// 1022 user items + 2 lineage entries is the unchanged Source limit.
	_, err := OwnRevisionEnvelope(uuid.NewString(), product, uuid.NewString(), 1)
	require.NoError(t, err)
	product.Attributes["extra1"] = "v"
	product.Attributes["extra2"] = "v"
	_, err = OwnEnvelope(uuid.NewString(), product)
	require.NoError(t, err)
	_, err = OwnRevisionEnvelope(uuid.NewString(), product, uuid.NewString(), 1)
	require.ErrorIs(t, err, ErrInvalid)
}
