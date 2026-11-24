package googlemerchant

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	domain "github.com/henryxu1008/shuomi-ai-commerce/internal/storecenter/googlemerchant"
	merchant "google.golang.org/api/merchantapi/products/v1"
)

const oauthState = "7EI5DwRz8TTBOsjvB6QI9Eixrr16BV3NU5lFMvOrLkk"
const oauthVerifier = "tFKrtU_nU_Kjv26rbdrZQG6tHv5h9XtLkn5ZRzK6wMk"

func TestOAuthAuthorizationUsesMerchantScopeAndPKCE(t *testing.T) {
	client, err := NewOAuth(domain.Application{ID: "client", Revision: "r1", CallbackURL: "https://example.com/stores/oauth/google"}, "secret", nil)
	if err != nil {
		t.Fatal(err)
	}
	location, err := client.AuthorizationURL(oauthState, oauthVerifier)
	if err != nil {
		t.Fatal(err)
	}
	u, err := url.Parse(location)
	if err != nil || u.Host != "accounts.google.com" {
		t.Fatalf("invalid authorization URL %q %v", location, err)
	}
	q := u.Query()
	if q.Get("scope") != merchant.ContentScope || q.Get("state") != oauthState ||
		q.Get("access_type") != "offline" || q.Get("prompt") != "consent" || q.Get("code_challenge_method") != "S256" ||
		q.Get("code_challenge") == "" || strings.Contains(location, oauthVerifier) {
		t.Fatalf("invalid authorization parameters: %v", q)
	}
}

func TestOAuthExchangeAndRefreshRetainOnlyConfirmedCredentials(t *testing.T) {
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if err := r.ParseForm(); err != nil {
			t.Errorf("parse form: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		switch r.Form.Get("grant_type") {
		case "authorization_code":
			if r.Form.Get("code_verifier") != oauthVerifier || r.Form.Get("redirect_uri") != "https://example.com/stores/oauth/google" {
				t.Errorf("wrong code exchange: %v", r.Form)
			}
			_ = json.NewEncoder(w).Encode(map[string]any{
				"access_token": "test-access", "refresh_token": "test-refresh", "token_type": "Bearer",
				"expires_in": 3600, "scope": merchant.ContentScope,
			})
		case "refresh_token":
			if r.Form.Get("refresh_token") != "test-refresh" {
				t.Errorf("wrong refresh: %v", r.Form)
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"access_token": "new-access", "token_type": "Bearer", "expires_in": 3600})
		default:
			t.Errorf("unexpected grant type: %s", r.Form.Get("grant_type"))
		}
	}))
	defer server.Close()
	client, err := NewOAuth(domain.Application{ID: "client", Revision: "r1", CallbackURL: "https://example.com/stores/oauth/google"}, "secret", server.Client())
	if err != nil {
		t.Fatal(err)
	}
	client.config.Endpoint.TokenURL = server.URL
	tokens, err := client.Exchange(context.Background(), "authorization-code", oauthVerifier)
	if err != nil || tokens.AccessToken != "test-access" || tokens.RefreshToken != "test-refresh" || tokens.ExpiresAt.IsZero() {
		t.Fatalf("unexpected token exchange result: %v", err)
	}
	tokens, err = client.Refresh(context.Background(), tokens)
	if err != nil || tokens.AccessToken != "new-access" || tokens.RefreshToken != "test-refresh" || calls != 2 {
		t.Fatalf("unexpected token refresh result: %v, calls %d", err, calls)
	}
}

func TestOAuthRejectsConsentWithoutMerchantScope(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token":"test-access","refresh_token":"test-refresh","token_type":"Bearer","expires_in":3600,"scope":"openid"}`))
	}))
	defer server.Close()
	client, err := NewOAuth(domain.Application{ID: "client", Revision: "r1", CallbackURL: "https://example.com/stores/oauth/google"}, "secret", server.Client())
	if err != nil {
		t.Fatal(err)
	}
	client.config.Endpoint.TokenURL = server.URL
	_, err = client.Exchange(context.Background(), "authorization-code", oauthVerifier)
	if !errors.Is(err, domain.ErrGrantRejected) {
		t.Fatalf("unexpected exchange error: %v", err)
	}
}
