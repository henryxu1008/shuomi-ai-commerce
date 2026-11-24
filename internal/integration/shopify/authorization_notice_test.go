package shopify

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAuthorizationNoticeAuthenticatesOriginalBodyAndTopic(t *testing.T) {
	const secret = "fixture-shopify-secret"
	oauth, err := NewOAuthClient(nil, "fixture", secret, "https://fixture.invalid/callback")
	require.NoError(t, err)
	sign := func(body []byte) string {
		mac := hmac.New(sha256.New, []byte(secret))
		mac.Write(body)
		return base64.StdEncoding.EncodeToString(mac.Sum(nil))
	}
	uninstall := []byte("{\"id\":548380009,\"name\":\"Fixture\",\"email\":\"discard@example.invalid\",\"myshopify_domain\":null}")
	scopes := []byte("{\"id\":1,\"shop_id\":\"gid://shopify/Shop/548380009\",\"previous\":[\"read_products\",\"write_products\"],\"current\":[\"read_products\"],\"updated_at\":\"2024-06-25T00:00:00.000Z\"}")
	notice, err := oauth.VerifyAuthorizationNotice(TopicAppUninstalled, sign(uninstall), uninstall)
	require.NoError(t, err)
	require.Equal(t, "gid://shopify/Shop/548380009", notice.ShopID)
	require.True(t, notice.RequiresCheck)
	require.Len(t, notice.BodyHash, 64)

	notice, err = oauth.VerifyAuthorizationNotice(TopicAppScopesUpdate, sign(scopes), scopes)
	require.NoError(t, err)
	require.Equal(t, "gid://shopify/Shop/548380009", notice.ShopID)
	require.True(t, notice.RequiresCheck)
	_, err = oauth.VerifyAuthorizationNotice(TopicAppUninstalled, sign(scopes), scopes)
	require.Error(t, err)
	_, err = oauth.VerifyAuthorizationNotice(TopicAppUninstalled, sign(uninstall), append(uninstall, ' '))
	require.ErrorIs(t, err, ErrNoticeSignature)

	restored := []byte("{\"id\":1,\"shop_id\":\"gid://shopify/Shop/548380009\",\"previous\":[],\"current\":[\"write_products\"]}")
	notice, err = oauth.VerifyAuthorizationNotice(TopicAppScopesUpdate, sign(restored), restored)
	require.NoError(t, err)
	require.False(t, notice.RequiresCheck)
	for _, bad := range []string{
		"{\"id\":1,\"id\":2}",
		"{\"id\":\"548380009\"}",
		"{\"id\":1,\"ID\":2}",
		"{\"id\":1,\"admin_graphql_api_id\":\"gid://shopify/Shop/2\"}",
	} {
		raw := []byte(bad)
		_, err = oauth.VerifyAuthorizationNotice(TopicAppUninstalled, sign(raw), raw)
		require.ErrorIs(t, err, ErrInvalidRequest, bad)
	}
}
