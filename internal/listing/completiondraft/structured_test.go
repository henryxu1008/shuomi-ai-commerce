package completiondraft

import (
	"encoding/json"
	"math"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"task-processor/internal/listing/preparation"
	"task-processor/internal/product/catalog"
	"task-processor/internal/product/collection"
)

func TestStructuredContentRoundTripKeepsZeroNullAndExplicitEmptyArrays(t *testing.T) {
	for _, raw := range []string{
		`{"title":"候选","description":"","structured":{"attributes":[],"variants":[]}}`,
		`{"title":"候选","description":"","structured":{"attributes":[{"name":"材质","value":""}],"variants":[{"sourceIndex":0,"price":{"currency":"CNY","amount":0},"stock":0},{"sourceIndex":1,"price":null,"stock":null}]}}`,
	} {
		var content Content
		require.NoError(t, json.Unmarshal([]byte(raw), &content))
		require.NoError(t, (SaveInput{SourceID: uuid.NewString(), Content: content}).Validate())
		encoded, err := json.Marshal(content)
		require.NoError(t, err)
		require.JSONEq(t, raw, string(encoded))
	}
}

func TestStructuredInputRejectsInvalidFieldsBeforePersistence(t *testing.T) {
	for _, raw := range []string{
		`{"attributes":null,"variants":[]}`,
		`{"attributes":[],"variants":null}`,
		`{"attributes":[{"name":" ","value":"x"}],"variants":[]}`,
		`{"attributes":[{"name":"材质","value":"x"},{"name":" 材质 ","value":"y"}],"variants":[]}`,
		`{"attributes":[],"variants":[{"sourceIndex":1,"price":null,"stock":null}]}`,
		`{"attributes":[],"variants":[{"sourceIndex":0,"price":{"currency":"cny","amount":1},"stock":0}]}`,
		`{"attributes":[],"variants":[{"sourceIndex":0,"price":{"currency":"CNY"},"stock":0}]}`,
		`{"attributes":[],"variants":[{"price":null,"stock":0}]}`,
		`{"attributes":[],"variants":[{"sourceIndex":0,"price":{"currency":"CNY","amount":-1},"stock":0}]}`,
		`{"attributes":[],"variants":[{"sourceIndex":0,"price":null,"stock":-1}]}`,
		`{"attributes":[],"variants":[{"sourceIndex":0,"price":null,"stock":9007199254740992}]}`,
	} {
		t.Run(raw, func(t *testing.T) {
			in := SaveInput{SourceID: uuid.NewString()}
			err := json.Unmarshal([]byte(`{"title":"","description":"","structured":`+raw+`}`), &in.Content)
			if err == nil {
				err = in.Validate()
			}
			require.Error(t, err)
		})
	}
}

func TestStructuredSaveRejectsRowsOutsideExactFixedSnapshot(t *testing.T) {
	s, _, r, ctx, source := fixture(t) // this fixed source has no variants
	in := SaveInput{SourceID: source.ID}
	require.NoError(t, json.Unmarshal([]byte(`{"structured":{"attributes":[],"variants":[{"sourceIndex":0,"price":null,"stock":0}]}}`), &in.Content))
	_, err := s.Save(ctx, uuid.NewString(), in)
	require.ErrorIs(t, err, preparation.ErrInvalid)
	require.Zero(t, r.writes)
}

func TestStructuredSaveCopiesTheCompleteInputAndReturnsIndependentPreparedValues(t *testing.T) {
	s, a, r, ctx, source := fixture(t)
	var err error
	s.sources, err = preparation.NewSourceSelector(s.preparations, s.owners, testSources{source: source}, testCatalog{catalog.PublishedSnapshot{Identity: catalog.SnapshotIdentity{TenantID: a.scope.OrganizationID, ProductKey: source.Source.ProductKey}, PublicationID: source.Source.PublicationID, Version: 1, Snapshot: catalog.ProductSnapshot{Variants: []catalog.Variant{{SKU: ""}, {SKU: ""}}}}})
	require.NoError(t, err)
	in := SaveInput{SourceID: source.ID}
	original := `{"title":"new","description":"","structured":{"attributes":[{"name":"材质","value":"棉"}],"variants":[{"sourceIndex":0,"price":{"currency":"CNY","amount":0},"stock":0},{"sourceIndex":1,"price":null,"stock":null}]}}`
	require.NoError(t, json.Unmarshal([]byte(original), &in.Content))
	hash := InputHash(in)
	_, err = s.Save(ctx, uuid.NewString(), in)
	require.NoError(t, err)
	in.Content.Structured.Attributes[0].Value = "changed"
	*in.Content.Structured.Variants[0].Stock = 99
	*in.Content.Structured.Variants[0].Price.Amount = 99
	_, _, gotHash, copied, _, err := r.saved.Read(ctx)
	require.NoError(t, err)
	require.Equal(t, hash, gotHash)
	raw, err := json.Marshal(copied.Content)
	require.NoError(t, err)
	require.JSONEq(t, original, string(raw))
	*copied.Content.Structured.Variants[0].Stock = 7
	_, _, _, again, _, err := r.saved.Read(ctx)
	require.NoError(t, err)
	require.EqualValues(t, 0, *again.Content.Structured.Variants[0].Stock)
	require.False(t, EqualContent(copied.Content, again.Content))
}

func TestTextOnlyHashesRemainUnchangedAndUnpreparedStructuredDoesNotBecomeSourceData(t *testing.T) {
	source := uuid.NewString()
	in := SaveInput{SourceID: source, Content: Content{Title: "old"}}
	oldWire := `{"sourceId":"` + source + `","expectedRevision":0,"content":{"title":"old","description":""}}`
	require.Equal(t, collection.Digest(oldWire), InputHash(in))
	var old Content
	require.NoError(t, json.Unmarshal([]byte(`{"title":"old","description":"","structured":null}`), &old))
	require.True(t, EqualContent(in.Content, old))
	raw, err := json.Marshal(old)
	require.NoError(t, err)
	require.JSONEq(t, `{"title":"old","description":""}`, string(raw))
}

func TestStructuredSignedZeroHasTheSameCanonicalHashAsJSONBZero(t *testing.T) {
	negative := SaveInput{SourceID: uuid.NewString()}
	require.NoError(t, json.Unmarshal([]byte(`{"title":"","description":"","structured":{"attributes":[],"variants":[{"sourceIndex":0,"price":{"currency":"CNY","amount":-0},"stock":0}]}}`), &negative.Content))
	copy, err := cloneInput(negative)
	require.NoError(t, err)
	require.False(t, math.Signbit(*copy.Content.Structured.Variants[0].Price.Amount))
	positive := SaveInput{SourceID: negative.SourceID}
	require.NoError(t, json.Unmarshal([]byte(`{"title":"","description":"","structured":{"attributes":[],"variants":[{"sourceIndex":0,"price":{"currency":"CNY","amount":0},"stock":0}]}}`), &positive.Content))
	require.Equal(t, InputHash(positive), InputHash(negative))
}
