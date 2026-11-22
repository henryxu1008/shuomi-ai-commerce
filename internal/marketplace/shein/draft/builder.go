// Package draft builds local, incomplete SHEIN drafts without remote resolvers.
package draft

import (
	"context"
	"encoding/json"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/listing/record"
	shein "github.com/henryxu1008/shuomi-ai-commerce/internal/marketplace/shein/offlinepackage"
	productasset "github.com/henryxu1008/shuomi-ai-commerce/internal/product/asset"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/product/catalog"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/product/catalog/canonical"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/product/description"
	"github.com/sirupsen/logrus"
)

type Builder struct{ LogFor func(string) *logrus.Entry }

func (b Builder) Build(ctx context.Context, snapshot catalog.ProductSnapshot, inventory productasset.ApprovedAssetInventory, input record.Input) ([]byte, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	product := catalog.ProjectCanonical(snapshot)
	product.Description = description.PlainText(product.Description, product.DescriptionFormat)
	product.DescriptionFormat = ""
	// Source imagery is not ApprovedAsset. Replace it completely with the exact
	// version-bound approved inventory supplied by Product/Asset.
	product.Images = make([]canonical.Image, 0, len(inventory.Assets))
	for _, approved := range inventory.Assets {
		product.Images = append(product.Images, canonical.Image{URL: approved.URL, Role: string(approved.Role)})
	}
	for i := range product.Variants {
		product.Variants[i].Images = nil
	}
	// Background intentionally contains no brand authorization, resolver, or AI
	// context values. Computation is synchronous/bounded; cancellation surrounds it.
	pkg := buildLocalPackage(product, input.Country, input.Language, b.LogFor)
	raw, err := json.Marshal(pkg)
	if err != nil {
		return nil, err
	}
	if len(raw) > shein.MaxPersistedPackageBytes {
		return nil, record.ErrTooLarge
	}
	if _, err = shein.DecodePersistedPackageStrict(raw); err != nil {
		return nil, err
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return raw, nil
}
