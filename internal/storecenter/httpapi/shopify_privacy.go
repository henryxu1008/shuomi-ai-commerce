package httpapi

import "github.com/henryxu1008/shuomi-ai-commerce/internal/httproute"

const (
	ShopifyCustomerDataRequestPath = "/api/v1/platform/shopify/customers-data-request"
	ShopifyCustomerRedactPath      = "/api/v1/platform/shopify/customers-redact"
	ShopifyShopRedactPath          = "/api/v1/platform/shopify/shop-redact"
)

// IsolatedShopifyPrivacyReceiptRoutes is not part of the Store route module.
// It is mounted only by the synthetic trial with synthetic data and a test secret.
func IsolatedShopifyPrivacyReceiptRoutes(receive shopifyNoticeReceiver) []httproute.Descriptor {
	return []httproute.Descriptor{
		shopifyNoticeRoute(ShopifyCustomerDataRequestPath, "customers/data_request", receive),
		shopifyNoticeRoute(ShopifyCustomerRedactPath, "customers/redact", receive),
		shopifyNoticeRoute(ShopifyShopRedactPath, "shop/redact", receive),
	}
}
