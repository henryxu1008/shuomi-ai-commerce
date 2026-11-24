package shopify

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"
)

const (
	fixtureClientSecret = "dummy-shopify-secret"
	fixtureShop         = "demo-shop.myshopify.com"
	fixtureNonce        = "fixture-nonce"
)

func TestOnlineOAuthURLRequestsOnlyProductWrite(t *testing.T) {
	client := fixtureOAuthClient(t, nil)
	got, err := client.AuthorizationURL(fixtureShop, fixtureNonce)
	if err != nil {
		t.Fatal(err)
	}
	link, err := url.Parse(got)
	if err != nil || link.Host != fixtureShop || link.Path != "/admin/oauth/authorize" ||
		link.Query().Get("grant_options[]") != "per-user" || link.Query().Get("scope") != "write_products" ||
		link.Query().Get("state") != fixtureNonce || link.Query().Get("redirect_uri") != "https://example.com/shopify/callback" {
		t.Fatalf("unexpected authorization URL %q, %v", got, err)
	}
}

func TestOwnerCodeRequiresAValidCallbackAndOwnerScope(t *testing.T) {
	for _, tc := range []struct {
		name, body string
		want       error
	}{
		{"owner", `{"access_token":"fixture-token","scope":"read_products,write_products","expires_in":86399,"associated_user_scope":"write_products","associated_user":{"account_owner":true,"email":"omit@example.com"}}`, nil},
		{"non-owner", `{"access_token":"fixture-token","scope":"write_products","expires_in":86399,"associated_user_scope":"write_products","associated_user":{"account_owner":false}}`, ErrShopOwnerRequired},
		{"read-only owner", `{"access_token":"fixture-token","scope":"write_products","expires_in":86399,"associated_user_scope":"read_products","associated_user":{"account_owner":true}}`, ErrShopOwnerRequired},
		{"no online identity", `{"access_token":"fixture-token","scope":"write_products","expires_in":3599}`, ErrShopOwnerRequired},
	} {
		t.Run(tc.name, func(t *testing.T) {
			calls := 0
			client := fixtureOAuthClient(t, transportFunc(func(req *http.Request) (*http.Response, error) {
				calls++
				if req.URL.String() != "https://"+fixtureShop+"/admin/oauth/access_token" || req.Header.Get("X-Shopify-Access-Token") != "" ||
					req.ParseForm() != nil || req.Form.Get("code") != "fixture-code" || req.Form.Get("client_secret") != fixtureClientSecret {
					t.Fatal("unexpected authorization-code exchange")
				}
				return fixtureResponse(http.StatusOK, tc.body), nil
			}))
			now := time.Unix(1795450500, 0)
			query := fixtureCallbackQuery()
			proof, err := client.VerifyCallback(query, fixtureNonce, now)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := client.ExchangeOwnerCode(context.Background(), proof); !errors.Is(err, tc.want) || calls != 1 {
				t.Fatalf("exchange error %v, want %v (requests %d)", err, tc.want, calls)
			}
		})
	}
}

func TestCallbackRejectsUntrustworthyEvidence(t *testing.T) {
	client := fixtureOAuthClient(t, nil)
	now := time.Unix(1795450500, 0)
	valid := fixtureCallbackQuery()
	for _, tc := range []struct {
		query, state string
		clock        time.Time
	}{
		{valid, "other-session", now},
		{valid, fixtureNonce, now.Add(time.Hour)},
		{valid + "&shop=" + fixtureShop, fixtureNonce, now},
		{strings.Replace(valid, fixtureShop, "other-shop.myshopify.com", 1), fixtureNonce, now},
	} {
		if _, err := client.VerifyCallback(tc.query, tc.state, tc.clock); !errors.Is(err, ErrInvalidCallback) {
			t.Fatalf("untrustworthy callback was accepted: %v", err)
		}
	}
}

func fixtureCallbackQuery() string {
	message := "code=fixture-code&host=YWRtaW4vZXhhbXBsZQ==&shop=" + fixtureShop + "&state=" + fixtureNonce + "&timestamp=1795450500"
	signature := hmac.New(sha256.New, []byte(fixtureClientSecret))
	_, _ = signature.Write([]byte(message))
	return "shop=" + fixtureShop + "&state=" + fixtureNonce + "&timestamp=1795450500&host=YWRtaW4vZXhhbXBsZQ%3D%3D&code=fixture-code&hmac=" + hex.EncodeToString(signature.Sum(nil))
}

func fixtureOAuthClient(t *testing.T, transport http.RoundTripper) *OAuthClient {
	t.Helper()
	client, err := NewOAuthClient(&http.Client{Transport: transport}, "fixture-client", fixtureClientSecret, "https://example.com/shopify/callback")
	if err != nil {
		t.Fatal(err)
	}
	return client
}
