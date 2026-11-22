package tests

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestProductReviewCurrentOwnerBoundaries(t *testing.T) {
	assertNoBannedImportPrefixes(t, filepath.Join("..", "internal", "product", "review"), []string{"gorm.io", "github.com/henryxu1008/shuomi-ai-commerce/internal/integration", "github.com/henryxu1008/shuomi-ai-commerce/internal/app", "github.com/henryxu1008/shuomi-ai-commerce/internal/platform", "github.com/henryxu1008/shuomi-ai-commerce/internal/listingkit", "github.com/henryxu1008/shuomi-ai-commerce/internal/compatibility", "github.com/henryxu1008/shuomi-ai-commerce/internal/tenantbridge", "github.com/henryxu1008/shuomi-ai-commerce/internal/product/image", "github.com/henryxu1008/shuomi-ai-commerce/internal/product/asset"}, nil)
	adapter := filepath.Join("..", "internal", "integration", "persistence", "product", "review")
	assertNoBannedImportPrefixes(t, adapter, []string{"github.com/henryxu1008/shuomi-ai-commerce/internal/product/sourcing", "github.com/henryxu1008/shuomi-ai-commerce/internal/product/enrichment", "github.com/henryxu1008/shuomi-ai-commerce/internal/product/review/proposal", "github.com/henryxu1008/shuomi-ai-commerce/internal/product/image", "github.com/henryxu1008/shuomi-ai-commerce/internal/product/asset", "github.com/henryxu1008/shuomi-ai-commerce/internal/marketplace", "github.com/henryxu1008/shuomi-ai-commerce/internal/listing"}, nil)
	raw, e := os.ReadFile(filepath.Join(adapter, "repository.go"))
	require.NoError(t, e)
	for _, table := range []string{"product_snapshot_versions", "product_snapshot_heads"} {
		require.NotContains(t, string(raw), table, "review must use Catalog writer, not Catalog SQL")
	}
	root := filepath.Join("..", "internal", "app", "httpapi")
	files, e := os.ReadDir(root)
	require.NoError(t, e)
	for _, file := range files {
		if file.IsDir() || !strings.HasSuffix(file.Name(), ".go") || strings.HasSuffix(file.Name(), "_test.go") || strings.HasPrefix(file.Name(), "product_review_") {
			continue
		}
		raw, e = os.ReadFile(filepath.Join(root, file.Name()))
		require.NoError(t, e)
		require.NotContains(t, string(raw), "NewProductReviewApplication(", "no default production registration")
	}
}
