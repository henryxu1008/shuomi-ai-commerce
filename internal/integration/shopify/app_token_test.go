package shopify

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"net/http"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v4"
)

func TestApplicationTokenUsesFreshShopBoundAssertion(t *testing.T) {
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	seen := make(map[string]bool)
	admin := &http.Client{Transport: transportFunc(func(req *http.Request) (*http.Response, error) {
		if req.ParseForm() != nil || req.Form.Get("client_secret") != "" || req.Form.Get("scope") != "" ||
			req.Form.Get("grant_type") != "client_credentials" || req.Form.Get("client_id") != "fixture-client" ||
			req.Form.Get("client_assertion_type") != "urn:ietf:params:oauth:client-assertion-type:jwt-bearer" {
			t.Fatal("incorrect assertion token request")
		}
		claims := &assertionClaims{}
		token, e := jwt.ParseWithClaims(req.Form.Get("client_assertion"), claims, func(token *jwt.Token) (any, error) {
			return &key.PublicKey, nil
		}, jwt.WithValidMethods([]string{"ES256"}))
		if e != nil || !token.Valid || token.Header["kid"] != "fixture-key" || claims.Issuer != "fixture-client" ||
			claims.Subject != claims.Issuer || claims.Audience != req.URL.String() || claims.ID == "" || seen[claims.ID] ||
			claims.ExpiresAt == nil || claims.IssuedAt == nil || claims.ExpiresAt.Sub(claims.IssuedAt.Time) != time.Minute {
			t.Fatalf("malformed or replayed assertion: %v", e)
		}
		seen[claims.ID] = true
		return fixtureResponse(http.StatusOK, `{"access_token":"fixture-token","scope":"write_products","expires_in":3599}`), nil
	})}
	issuer, err := NewAppTokenIssuer(admin, "fixture-client", "fixture-key", key)
	if err != nil {
		t.Fatal(err)
	}
	for range 2 {
		token, err := issuer.Issue(context.Background(), fixtureShop)
		if err != nil || token.Value != "fixture-token" || !time.Now().Before(token.ExpiresAt) {
			t.Fatalf("application token unavailable: %v", err)
		}
	}
}
