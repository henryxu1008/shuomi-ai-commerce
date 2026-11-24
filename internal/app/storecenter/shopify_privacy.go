package storecenterapp

import (
	"context"
	"errors"
	"time"

	wire "github.com/henryxu1008/shuomi-ai-commerce/internal/integration/shopify"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/storecenter"
)

// IsolatedShopifyPrivacyReceiver is composed only in the product-only synthetic
// trial. Its dependencies cannot dispatch a Shopify request or a purge workflow.
func IsolatedShopifyPrivacyReceiver(store *storecenter.MemberScopedStoreRepository, appID string, oauth *wire.OAuthClient, now func() time.Time) func(context.Context, string, string, string, []byte) error {
	return func(ctx context.Context, topic, eventID, signature string, body []byte) error {
		if store == nil || oauth == nil || now == nil {
			return storecenter.ErrOfficialConnectionUnavailable
		}
		shop, err := oauth.VerifyPrivacyNotice(topic, signature, body)
		if errors.Is(err, wire.ErrNoticeSignature) {
			return storecenter.ErrOfficialAuthorizationRejected
		}
		if err != nil {
			return storecenter.ErrNotFound
		}
		return store.ReceiveIsolatedShopifyPrivacy(ctx, appID, topic, eventID, shop, now())
	}
}
