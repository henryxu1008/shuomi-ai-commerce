package client

import (
	"context"
	"crypto/tls"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

type currentReadRoundTrip func(*http.Request) (*http.Response, error)

func (f currentReadRoundTrip) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }
func TestExplicitCurrentSessionTransportKeepsCookieDomainAndTLSBoundary(t *testing.T) {
	cookies, e := CookiesFromBrowserState(map[string]any{"cookies": []any{
		map[string]any{"name": "current", "value": "synthetic-private", "domain": ".shein.com", "path": "/", "expires": float64(time.Now().Add(time.Hour).Unix()), "sameSite": "Lax"},
		map[string]any{"name": "other-domain", "value": "must-not-leak", "domain": ".geiwohuo.com", "path": "/"},
		map[string]any{"name": "expired", "value": "must-not-send", "domain": ".shein.com", "path": "/", "expires": float64(time.Now().Add(-time.Hour).Unix())},
	}})
	if e != nil {
		t.Fatal(e)
	}
	transport, e := NewSessionTransport("https://sellerhub.shein.com", "", cookies)
	if e != nil {
		t.Fatal(e)
	}
	defer transport.Close()
	if transport.transport.TLSClientConfig.InsecureSkipVerify || transport.transport.TLSClientConfig.MinVersion < tls.VersionTLS12 || transport.client.CheckRedirect == nil {
		t.Fatal("session weakened TLS/redirect policy")
	}
	calls := 0
	transport.client.Transport = currentReadRoundTrip(func(r *http.Request) (*http.Response, error) {
		calls++
		if r.Header.Get("Cookie") != "current=synthetic-private" || r.Method != "POST" || r.Header.Get("referer") != "https://sellerhub.shein.com/" {
			t.Fatal("session/header boundary", r.Header.Get("Cookie"))
		}
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{"code":"0"}`)), Header: make(http.Header)}, nil
	})
	var out map[string]any
	if e = transport.Read(context.Background(), GetSupplierOperateInfoEndpoint(), nil, &out); e != nil || calls != 1 {
		t.Fatal(out, e)
	}
	if _, e = NewSessionTransport("https://other.example", "", cookies); e == nil {
		t.Fatal("untrusted endpoint accepted")
	}
}
