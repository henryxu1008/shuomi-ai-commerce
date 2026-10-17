package review

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"task-processor/internal/product/catalog"
	"task-processor/internal/product/enrichment"
)

func TestContentPatchIsAtomicAndPreservesWholeOtherProduct(t *testing.T) {
	base := catalog.ProductSnapshot{Title: "old title", Description: "old description", Brand: "source brand",
		SellingPoints: []string{"old point"}, SEOKeywords: []string{"old keyword"},
		Attributes:     []catalog.Attribute{{Name: "material", Value: "steel"}},
		Specifications: &catalog.Specifications{Technical: map[string]string{"grade": "304"}},
		Variants:       []catalog.Variant{{SKU: "original-sku", Stock: 7, Price: &catalog.Price{Currency: "USD", Amount: 12.3}}},
		Images:         []catalog.Image{{URL: "https://source.invalid/original.jpg"}},
		Sources:        []catalog.SourceRecord{{SourceID: "source-1", Checksum: "source-checksum"}}}
	before, err := json.Marshal(base)
	require.NoError(t, err)
	selection, err := enrichment.NewContentSelection("en-US", []string{"title", "description", "selling_points", "seo_keywords"})
	require.NoError(t, err)
	changes := []enrichment.FieldChange{{Field: "title", Value: "new title"}, {Field: "description", Value: "New description.\nSecond paragraph."},
		{Field: "selling_points", Value: `["Steel body","Portable"]`}, {Field: "seo_keywords", Value: `["steel bottle"]`}}
	got, err := applyContentPatch(base, selection, changes)
	require.NoError(t, err)
	want, err := catalog.CloneProductSnapshot(base)
	require.NoError(t, err)
	want.Title, want.Description = "new title", "New description.\nSecond paragraph."
	want.SellingPoints, want.SEOKeywords = []string{"Steel body", "Portable"}, []string{"steel bottle"}
	require.Equal(t, want, got)
	got.Variants[0].Price.Amount = 99
	got.Specifications.Technical["grade"] = "tampered"
	got.Sources[0].Checksum = "tampered"
	got.SellingPoints[0] = "tampered"
	after, err := json.Marshal(base)
	require.NoError(t, err)
	require.Equal(t, before, after)

	// A later invalid field must leave the caller's complete base untouched.
	changes[3].Value = `{"price":99}`
	_, err = applyContentPatch(base, selection, changes)
	require.Error(t, err)
	after, err = json.Marshal(base)
	require.NoError(t, err)
	require.Equal(t, before, after)
}

func TestContentPatchRejectsIncompleteOrExpandedFieldSetAndErasure(t *testing.T) {
	selection, err := enrichment.NewContentSelection("en", []string{"title", "description"})
	require.NoError(t, err)
	base := catalog.ProductSnapshot{Title: "original title", Description: "original description"}
	for _, changes := range [][]enrichment.FieldChange{
		{{Field: "title", Value: "valid title"}},
		{{Field: "title", Value: "valid title"}, {Field: "price", Value: "1"}},
		{{Field: "title", Value: "valid title"}, {Field: "title", Value: "duplicate"}},
		{{Field: "title", Value: "valid title"}, {Field: "description", Value: ""}},
		{{Field: "title", Value: "valid title"}, {Field: "description", Value: " leading"}},
		{{Field: "title", Value: "valid title"}, {Field: "description", Value: "control\x00"}},
	} {
		_, err := applyContentPatch(base, selection, changes)
		require.Error(t, err)
	}
	_, err = applyContentPatch(base, enrichment.ContentSelection{}, []enrichment.FieldChange{{Field: "title", Value: "valid"}})
	require.Error(t, err)
}

func TestContentPatchArrayValuesHaveStrictBoundedStringSchema(t *testing.T) {
	selection, err := enrichment.NewContentSelection("en", []string{"selling_points"})
	require.NoError(t, err)
	base := catalog.ProductSnapshot{Title: "source title"}
	for _, value := range []string{`null`, `[]`, `[""]`, `[" leading"]`, `[1]`, `[["nested"]]`, `["good",null]`, `["good"] {}`, ` ["good"]`, `[ "good" ]`, `["good\u0000"]`} {
		_, err := applyContentPatch(base, selection, []enrichment.FieldChange{{Field: "selling_points", Value: value}})
		require.Error(t, err, value)
	}
	_, err = applyContentPatch(base, selection, []enrichment.FieldChange{{Field: "selling_points", Value: `["` + strings.Repeat("x", MaxRecordBytes) + `"]`}})
	require.ErrorIs(t, err, ErrTooLarge)
	got, err := applyContentPatch(base, selection, []enrichment.FieldChange{{Field: "selling_points", Value: `["原有事实","Supported fact"]`}})
	require.NoError(t, err)
	require.Equal(t, []string{"原有事实", "Supported fact"}, got.SellingPoints)
}

func TestContentPatchPreservesRichDescriptionUnlessExplicitlyReplaced(t *testing.T) {
	base := catalog.ProductSnapshot{Title: "title", Description: "<p><strong>cotton</strong></p>", DescriptionFormat: "html-v1"}
	selection, err := enrichment.NewContentSelection("en", []string{"title"})
	require.NoError(t, err)
	got, err := applyContentPatch(base, selection, []enrichment.FieldChange{{Field: "title", Value: "new title"}})
	require.NoError(t, err)
	require.Equal(t, base.Description, got.Description)
	require.Equal(t, "html-v1", got.DescriptionFormat)
	selection, err = enrichment.NewContentSelection("en", []string{"description"})
	require.NoError(t, err)
	got, err = applyContentPatch(base, selection, []enrichment.FieldChange{{Field: "description", Value: "<b>literal output</b>"}})
	require.NoError(t, err)
	require.Empty(t, got.DescriptionFormat)
	require.Equal(t, "<b>literal output</b>", got.Description)
	values, err := snapshotContentValues(base, selection)
	require.NoError(t, err)
	require.Equal(t, "cotton", values[0].Value)
}
