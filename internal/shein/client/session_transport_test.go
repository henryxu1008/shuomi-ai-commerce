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

func TestCurrentSessionRetainsProviderEndpointHeaders(t *testing.T) {
	for _, origin := range []string{"https://sellerhub.shein.com", "https://sso.geiwohuo.com"} {
		for _, path := range []string{GetSupplierOperateInfoEndpoint(), GetWarehousesEndpoint(), GetStoreAddressListEndpoint()} {
			t.Run(origin+path, func(t *testing.T) {
				transport, err := NewSessionTransport(origin, "", nil)
				if err != nil {
					t.Fatal(err)
				}
				defer transport.Close()
				transport.client.Transport = currentReadRoundTrip(func(r *http.Request) (*http.Response, error) {
					want := map[string]string{"method": "POST", "scheme": "https", "content-type": "application/json", "referer": origin + "/"}
					if path == GetSupplierOperateInfoEndpoint() {
						for k, v := range map[string]string{"path": "/sso/public/account/supplier/getSupplierOperateInfo", "gmpsso-language": "CN", "sso-frontend-version": "1.0.0", "time-zone": "America/Los_Angeles", "x-sso-scene": "sw"} {
							want[k] = v
						}
					}
					for k, v := range want {
						if r.Header.Get(k) != v {
							t.Errorf("header %s=%q, want %q", k, r.Header.Get(k), v)
						}
					}
					return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{"code":"0"}`)), Header: make(http.Header)}, nil
				})
				var out map[string]any
				if err := transport.Read(context.Background(), path, nil, &out); err != nil {
					t.Fatal(err)
				}
			})
		}
	}
	// The retained platform client's self-operated distinction must stay intact.
	b := &BaseAPIClient{shopType: "self_operated"}
	h := b.createHeaders("https://sellerhub.shein.com" + GetSupplierOperateInfoEndpoint())
	if h["method"] != "POST" || h["scheme"] != "https" || h["x-sso-scene"] != "" || h["gmpsso-language"] != "" {
		t.Fatal("self-operated platform contract changed")
	}
}
