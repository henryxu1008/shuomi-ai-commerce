package servicepayments

import (
	"github.com/henryxu1008/shuomi-ai-commerce/internal/commercial/billing"
	e "github.com/henryxu1008/shuomi-ai-commerce/internal/ecoservices"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/integration/paymentsecurity"
)

func NewPayloadProtection(key []byte) (billing.ServicePayloadProtection, error) {
	return paymentsecurity.NewPayloadProtection(key, "ecoservices-payment:v1:")
}
func NewMerchantPayloadProtection(key []byte) (e.MerchantProtection, error) {
	return paymentsecurity.NewPayloadProtection(key, "ecoservices-merchant:v1:")
}
