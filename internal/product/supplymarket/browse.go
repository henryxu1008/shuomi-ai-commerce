package supplymarket

import (
	"context"
	"task-processor/internal/product/collection"
)

type PriceRange struct {
	Currency string  `json:"currency"`
	Minimum  float64 `json:"minimum"`
	Maximum  float64 `json:"maximum"`
}
type Merchant struct {
	State string `json:"state"`
	Name  string `json:"name,omitempty"`
}
type VerifiedMerchantReader interface {
	ReadVerifiedMerchant(context.Context, string) (Merchant, error)
}
type MarketData struct {
	Prices   []PriceRange
	Added    *bool
	Category *CategoryPath
}
type MarketView struct {
	Release
	Prices   []PriceRange  `json:"prices"`
	Added    *bool         `json:"added"`
	Merchant *Merchant     `json:"merchant,omitempty"`
	Category *CategoryPath `json:"category,omitempty"`
}

func (s *Service) views(ctx context.Context, scope collection.Scope, releases []Release) ([]MarketView, error) {
	ids := make([]string, 0, len(releases))
	for _, r := range releases {
		ids = append(ids, r.ID)
	}
	data, err := s.repository.ReadMarketData(ctx, scope, ids)
	if err != nil {
		return nil, err
	}
	views := make([]MarketView, 0, len(releases))
	merchants := map[string]Merchant{}
	for _, r := range releases {
		d, ok := data[r.ID]
		if !ok {
			return nil, ErrUnavailable
		}
		view := MarketView{Release: r, Prices: d.Prices, Added: d.Added, Category: d.Category}
		if view.Prices == nil {
			view.Prices = []PriceRange{}
		}
		if r.Channel == "selected" {
			m, exists := merchants[r.OriginalOwner.OrganizationID]
			if !exists {
				m = Merchant{State: "unavailable"}
				if s.merchants != nil {
					value, e := s.merchants.ReadVerifiedMerchant(ctx, r.OriginalOwner.OrganizationID)
					if e == nil && value.State == "unverified" {
						m = Merchant{State: "unverified"}
					}
					if e == nil && value.State == "verified" && text(value.Name, 1, 256) {
						m = value
					}
				}
				merchants[r.OriginalOwner.OrganizationID] = m
			}
			view.Merchant = &m
		}
		views = append(views, view)
	}
	return views, nil
}
