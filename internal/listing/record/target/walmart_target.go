package target

import (
	"context"
	"encoding/json"
	"sort"
	"strings"
	"task-processor/internal/marketplace/walmart"
	"task-processor/internal/product/asset"
	"task-processor/internal/product/catalog"
	"task-processor/internal/product/collection"
	"task-processor/internal/storecenter"
)

type TargetWalmartRuleReader interface {
	ReadWalmartTargetRules(context.Context, collection.Scope, string, walmart.RuleSelection) (storecenter.ProductMerchantBinding, walmart.RuleSnapshot, error)
}
type TargetWalmartImageReader interface {
	ProbeWalmart(context.Context, asset.ApprovedAsset) (walmart.ImageObservation, error)
}

func WalmartSourceSKUKeys(product catalog.ProductSnapshot) []string {
	out := make([]string, 0, len(product.Variants))
	for _, v := range product.Variants {
		key := v.SourceID
		if key == "" {
			key = v.SKU
		}
		out = append(out, key)
	}
	return out
}
func sortWalmartImages(images []walmart.ImageObservation) {
	sort.Slice(images, func(i, j int) bool { return images[i].AssetID < images[j].AssetID })
}
func ProbeWalmartTargetImages(ctx context.Context, reader TargetWalmartImageReader, in walmart.DraftInput, inventory asset.ApprovedAssetInventory) ([]walmart.ImageObservation, error) {
	urls := map[string]bool{}
	var visit func(any, string) error
	visit = func(v any, key string) error {
		switch n := v.(type) {
		case map[string]any:
			for k, value := range n {
				if e := visit(value, k); e != nil {
					return e
				}
			}
		case []any:
			for _, value := range n {
				if e := visit(value, key); e != nil {
					return e
				}
			}
		case string:
			lower := strings.ToLower(key)
			if strings.Contains(lower, "image") && strings.Contains(lower, "url") {
				urls[n] = true
			}
		}
		return nil
	}
	raw, _ := json.Marshal(in)
	var value any
	if json.Unmarshal(raw, &value) != nil {
		return nil, ErrInvalid
	}
	if e := visit(value, ""); e != nil {
		return nil, e
	}
	if len(urls) > 200 {
		return nil, ErrTooLarge
	}
	if in.Selection.FeedType == walmart.FullFeed && len(urls) == 0 {
		return nil, ErrNotReady
	}
	out := []walmart.ImageObservation{}
	for u := range urls {
		if reader == nil {
			return nil, ErrUnavailable
		}
		var approved *asset.ApprovedAsset
		for _, a := range inventory.Assets {
			if a.URL == u {
				copy := a
				approved = &copy
				break
			}
		}
		if approved == nil {
			return nil, ErrNotReady
		}
		observed, e := reader.ProbeWalmart(ctx, *approved)
		if e != nil || observed.AssetID != approved.ID || observed.URL != u || len(observed.ContentHash) != 64 || observed.Width < 1 || observed.Height < 1 {
			return nil, ErrNotReady
		}
		out = append(out, observed)
	}
	// Stable order is part of the immutable image evidence, irrespective of map order.
	sortWalmartImages(out)
	return out, nil
}
