package walmart

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	auth "task-processor/internal/storecenter/walmart"
)

type transportFunc func(*http.Request) (*http.Response, error)

func (f transportFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }
func TestOAuthUSHeadersAndRefreshRetainsAbsoluteExpiry(t *testing.T) {
	calls := 0
	client, e := NewOfficialClient(Options{Application: auth.Application{ClientID: "client", Revision: "r1", Environment: "production", CallbackURL: "https://app.example/workbench/stores/walmart/callback"}, ClientSecret: "secret", HTTPClient: &http.Client{Transport: transportFunc(func(r *http.Request) (*http.Response, error) {
		calls++
		user, pass, ok := r.BasicAuth()
		if !ok || user != "client" || pass != "secret" || r.Header.Get("WM_PARTNER.ID") != "seller" || r.Header.Get("WM_MARKET") != "us" || r.URL.String() != "https://marketplace.walmartapis.com/v3/token" {
			t.Fatal("wrong token contract")
		}
		if e := r.ParseForm(); e != nil || r.Form.Get("grant_type") != "refresh_token" {
			t.Fatal("wrong grant")
		}
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{"access_token":"new","expires_in":37,"token_type":"Bearer"}`)), Header: make(http.Header)}, nil
	})}})
	if e != nil {
		t.Fatal(e)
	}
	link, e := client.AuthorizationURL("state", "nonce")
	if e != nil || !strings.Contains(link, "clientType=seller") || !strings.Contains(link, "responseType=code") || strings.Contains(link, "response_type") {
		t.Fatal(link, e)
	}
	expiry := time.Now().Add(24 * time.Hour)
	old := auth.TokenSet{SellerID: "seller", RefreshToken: "refresh", RefreshExpiresAt: expiry}
	next, e := client.Refresh(context.Background(), old)
	if e != nil || next.RefreshToken != "refresh" || !next.RefreshExpiresAt.Equal(expiry) || time.Until(next.AccessExpiresAt) > 38*time.Second || calls != 1 {
		t.Fatal("wrong refresh semantics", e)
	}
	b, _ := json.Marshal(next)
	if strings.Contains(string(b), "new") || strings.Contains(string(b), "refresh") {
		t.Fatal("credential leak")
	}
}
func TestLostMutationResponseIsNotRetriedOrRedirected(t *testing.T) {
	calls := 0
	c, e := NewOfficialClient(Options{Application: auth.Application{ClientID: "client", Revision: "r1", Environment: "production", CallbackURL: "https://app.example/callback"}, ClientSecret: "secret", HTTPClient: &http.Client{Transport: transportFunc(func(r *http.Request) (*http.Response, error) {
		calls++
		return &http.Response{StatusCode: 307, Header: http.Header{"Location": []string{"https://evil.example"}}, Body: io.NopCloser(strings.NewReader(""))}, nil
	})}})
	if e != nil {
		t.Fatal(e)
	}
	if _, e = c.Exchange(context.Background(), "code", "seller"); e == nil || calls != 1 {
		t.Fatal("unsafe token retry", e, calls)
	}
}
