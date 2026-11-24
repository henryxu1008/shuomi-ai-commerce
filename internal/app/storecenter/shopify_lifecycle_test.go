package storecenterapp

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/google/uuid"
	wire "github.com/henryxu1008/shuomi-ai-commerce/internal/integration/shopify"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/ledger/orgresource"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/storecenter"
	"github.com/stretchr/testify/require"
)

func TestShopifyLifecycleSignedNoticeSuspendsNewServiceChargesBeforeProviderCheck(t *testing.T) {
	f := newServiceChargeFixtureForPlatform(t, storecenter.PlatformShopify)
	const secret = "fixture-lifecycle-secret"
	ctx := context.Background()
	var calls int
	transport := &http.Client{Transport: startedShopifyTransport(func(req *http.Request) (*http.Response, error) {
		calls++
		require.Equal(t, f.shopify.Domain, req.URL.Host)
		var body string
		switch req.URL.Path {
		case "/admin/oauth/access_token":
			body = `{"access_token":"scope-probe-token","scope":"read_products","expires_in":300}`
		case "/admin/api/2026-10/graphql.json":
			data, err := io.ReadAll(req.Body)
			require.NoError(t, err)
			require.Contains(t, string(data), "currentAppInstallation")
			require.NotContains(t, string(data), "mutation")
			body = `{"data":{"shop":{"id":"gid://shopify/Shop/19","myshopifyDomain":"fixture.myshopify.com","currencyCode":"USD","plan":{"partnerDevelopment":true}},"currentAppInstallation":{"id":"gid://shopify/AppInstallation/27","app":{"id":"gid://shopify/App/812"},"accessScopes":[{"handle":"read_products"}]}}}`
		default:
			t.Fatalf("unexpected Shopify endpoint: %s", req.URL.Path)
		}
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(body))}, nil
	})}
	oauth, err := wire.NewOAuthClient(transport, "fixture-client", secret, "https://fixture.invalid/callback")
	require.NoError(t, err)
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)
	issuer, err := wire.NewAppTokenIssuer(transport, "fixture-client", "fixture-key", key)
	require.NoError(t, err)
	protection, err := wire.NewPairingProtection("fixture-key", []byte(strings.Repeat("k", 32)))
	require.NoError(t, err)
	connections, err := NewShopifyConnections(f.repo, ShopifyApplication{AppID: f.shopify.AppID, InstallURL: "https://apps.shopify.com/fixture",
		OAuth: oauth, Admin: wire.NewDevelopmentAdminClient(transport), Issuer: issuer, Protection: protection, LifecycleNotifications: true})
	require.NoError(t, err)

	body := []byte(`{"id":1,"shop_id":"gid://shopify/Shop/19","previous":["write_products"],"current":["read_products"]}`)
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	signature := base64.StdEncoding.EncodeToString(mac.Sum(nil))
	topic := wire.TopicAppScopesUpdate
	require.ErrorIs(t, connections.ReceiveShopifyLifecycle(ctx, topic, "fixture-event", "bad", body), storecenter.ErrOfficialAuthorizationRejected)
	require.NoError(t, connections.ReceiveShopifyLifecycle(ctx, topic, "fixture-event", signature, body))
	require.NoError(t, connections.ReceiveShopifyLifecycle(ctx, topic, "fixture-event", signature, body))
	require.Zero(t, calls, "notification acknowledgement is independent of Shopify API")
	before := func() storecenter.ConnectionStatus {
		status, err := connections.Status(f.member, f.shopify.OrganizationID, f.store.ID())
		require.NoError(t, err)
		return status
	}
	require.Equal(t, storecenter.ConnectionStatusUnavailable, before())
	official := &OfficialConnections{shopify: connections}
	application := serviceApplicationWithConnections(t, f, f.ledger, official)
	req := storecenter.ServiceLifecycleApplicationRequest{StoreID: f.store.ID(), OperationID: uuid.NewString(), ExpectedStoreVersion: f.store.Version()}
	_, err = application.Activate(f.member, req)
	require.ErrorIs(t, err, storecenter.ErrConnectionNotFresh)
	_, err = f.ledger.Lookup(f.member, storeServiceChargeIdentity(req.OperationID))
	require.ErrorIs(t, err, orgresource.ErrReservationNotFound)
	require.Zero(t, calls)
	// Stopping HTTP acceptance must not reopen or strand an existing pause.
	connections.app.LifecycleNotifications = false
	require.False(t, connections.LifecycleNotificationsEnabled())
	require.Equal(t, storecenter.ConnectionStatusUnavailable, before())
	require.NoError(t, connections.ReconcileShopifyLifecycle(ctx))
	require.Equal(t, 2, calls, "one token exchange and one read-only installation probe")
	require.Equal(t, storecenter.ConnectionStatusDisconnected, before())
	require.NoError(t, connections.ReconcileShopifyLifecycle(ctx))
	require.Equal(t, 2, calls, "terminal receipt is not redispatched")
}
