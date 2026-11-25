package listingapp

import (
	"fmt"
	"strings"
	"testing"

	"github.com/henryxu1008/shuomi-ai-commerce/internal/imageagent"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/product/catalog"
	"github.com/stretchr/testify/require"
)

func TestListingImageColorOptionsIgnoresUnsupportedAttributes(t *testing.T) {
	const black = "https://example.test/black.png"
	const white = "https://example.test/white.png"
	longName := strings.Repeat("N", 193)
	longValue := strings.Repeat("V", 193)
	multibyteValue := strings.Repeat("色", 65)
	snapshot := catalog.PublishedSnapshot{Snapshot: catalog.ProductSnapshot{Variants: []catalog.Variant{
		{Attributes: []catalog.Attribute{
			{Name: "Color", Value: "黑色"},
			{Name: longName, Value: "A"},
			{Name: "Long value", Value: longValue + "A"},
			{Name: "Multi-byte value", Value: multibyteValue + "A"},
			{Name: "Web address", Value: "https://example.test/a"},
		}, Images: []catalog.Image{{URL: black}}},
		{Attributes: []catalog.Attribute{
			{Name: "Color", Value: "白色"},
			{Name: longName, Value: "B"},
			{Name: "Long value", Value: longValue + "B"},
			{Name: "Multi-byte value", Value: multibyteValue + "B"},
			{Name: "Web address", Value: "https://example.test/b"},
		}, Images: []catalog.Image{{URL: white}}},
	}}}
	assets := []imageagent.AuthorizedAsset{{ID: "black-original", URL: black}, {ID: "white-original", URL: white}}
	require.Equal(t, []imageagent.ImageColorOption{{Name: "Color", Groups: []imageagent.ImageColorGroup{
		{Value: "黑色", OriginalIDs: []string{"black-original"}}, {Value: "白色", OriginalIDs: []string{"white-original"}},
	}}}, listingImageColorOptions(snapshot, assets))
}

func TestListingImageColorOptionsRespectsSourceResponseLimit(t *testing.T) {
	var first, second catalog.Variant
	for i := range 32 {
		name := fmt.Sprintf("Attribute %d", i)
		first.Attributes = append(first.Attributes, catalog.Attribute{Name: name, Value: "A"})
		second.Attributes = append(second.Attributes, catalog.Attribute{Name: name, Value: "B"})
	}
	snapshot := catalog.PublishedSnapshot{Snapshot: catalog.ProductSnapshot{Variants: []catalog.Variant{first, second}}}
	tests := listingImageColorOptions(snapshot, nil)
	require.Len(t, tests, 30)
	require.Equal(t, "Attribute 0", tests[0].Name)
	require.Equal(t, "Attribute 29", tests[29].Name)
}
