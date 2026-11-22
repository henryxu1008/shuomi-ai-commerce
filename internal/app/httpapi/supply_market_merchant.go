package httpapi

import (
	"context"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/product/supplymarket"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/subjectverification"
)

type verifiedMarketMerchant struct{ service *subjectverification.Service }

func (r verifiedMarketMerchant) ReadVerifiedMerchant(ctx context.Context, organizationID string) (supplymarket.Merchant, error) {
	name, err := r.service.VerifiedCompanyName(ctx, organizationID)
	if err != nil {
		return supplymarket.Merchant{}, err
	}
	if name == "" {
		return supplymarket.Merchant{State: "unverified"}, nil
	}
	return supplymarket.Merchant{State: "verified", Name: name}, nil
}
