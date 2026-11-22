package asset_test

import (
	"testing"

	"github.com/henryxu1008/shuomi-ai-commerce/internal/product/asset"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/product/asset/assettest"
)

func TestMemoryRepositoryContract(t *testing.T) {
	assettest.ExerciseRepositoryContract(t, func(t *testing.T) asset.Repository {
		t.Helper()
		return assettest.NewMemoryRepository()
	})
}
