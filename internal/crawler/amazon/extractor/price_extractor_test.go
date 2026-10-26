package extractor

import (
	amazonresult "task-processor/internal/crawler/amazon/capturedata"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBuildScopedSelectors(t *testing.T) {
	result := buildScopedSelectors(
		[]string{"#corePriceDisplay_desktop_feature_div", "#apex_desktop"},
		[]string{".a-price .a-offscreen", ".a-price-whole"},
	)

	assert.Equal(t, []string{
		"#corePriceDisplay_desktop_feature_div .a-price .a-offscreen",
		"#corePriceDisplay_desktop_feature_div .a-price-whole",
		"#apex_desktop .a-price .a-offscreen",
		"#apex_desktop .a-price-whole",
	}, result)
}

func TestBuildScopedSelectorsWithoutScopes(t *testing.T) {
	result := buildScopedSelectors(nil, []string{".a-price .a-offscreen"})
	assert.Equal(t, []string{".a-price .a-offscreen"}, result)
}

func TestSyncInitialPriceWithListPrice(t *testing.T) {
	extractor := NewPriceExtractor("UK", nil)
	listPrice := 29.99
	product := &amazonresult.Product{
		FinalPrice: 19.93,
		PricesBreakdown: amazonresult.PriceBreakdown{
			ListPrice: &listPrice,
		},
	}

	extractor.syncInitialPriceWithListPrice(product)

	assert.Equal(t, 29.99, product.InitialPrice)
}

func TestSyncInitialPriceWithListPriceClearsMissingOriginalPrice(t *testing.T) {
	extractor := NewPriceExtractor("UK", nil)
	product := &amazonresult.Product{
		FinalPrice:   19.93,
		InitialPrice: 19.93,
	}

	extractor.syncInitialPriceWithListPrice(product)

	assert.Zero(t, product.InitialPrice)
}
