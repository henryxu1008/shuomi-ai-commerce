package einomodel

import (
	"context"
	"crypto/x509"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"sync/atomic"
	"testing"
)

func TestRegisteredTransportPinsOnlyFullyPublicResolution(t *testing.T) {
	resolved := []netip.Addr{netip.MustParseAddr("8.8.8.8"), netip.MustParseAddr("::1")}
	var dest string
	dial := publicChannelDial(
		func(context.Context, string, string) ([]netip.Addr, error) { return resolved, nil },
		func(_ context.Context, _, address string) (net.Conn, error) {
			dest = address
			return nil, errors.New("fixture")
		},
	)
	ctx := context.Background()
	_, _ = dial(ctx, "tcp", "models.example.com:443")
	if dest != "" {
		t.Fatal("mixed response reached the dialer")
	}
	resolved = []netip.Addr{netip.MustParseAddr("::ffff:8.8.8.8")}
	_, _ = dial(ctx, "tcp", "models.example.com:443")
	if dest != "" {
		t.Fatal("mapped address reached the dialer")
	}
	resolved = []netip.Addr{netip.MustParseAddr("8.8.8.8")}
	_, _ = dial(ctx, "tcp", "models.example.com:443")
	if dest != "8.8.8.8:443" {
		t.Fatalf("dialed an unpinned endpoint: %q", dest)
	}
	transport := NewRegisteredChannelTransport()
	if transport.Proxy != nil || transport.DialTLSContext != nil || !transport.DisableKeepAlives || transport.ForceAttemptHTTP2 {
		t.Fatal("transport can bypass the guarded fresh dial")
	}
}

func TestRegisteredTransportAuthenticatesOriginalHostAndRechecksNewDial(t *testing.T) {
	var requests atomic.Int32
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		requests.Add(1)
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	roots := x509.NewCertPool()
	roots.AddCert(server.Certificate())
	transport := NewRegisteredChannelTransport()
	defer transport.CloseIdleConnections()
	transport.TLSClientConfig.RootCAs = roots
	var changed atomic.Bool
	var dialer net.Dialer
	transport.DialContext = publicChannelDial(func(context.Context, string, string) ([]netip.Addr, error) {
		if changed.Load() {
			return []netip.Addr{netip.MustParseAddr("127.0.0.1")}, nil
		}
		return []netip.Addr{netip.MustParseAddr("8.8.8.8")}, nil
	}, func(ctx context.Context, network, pinned string) (net.Conn, error) {
		if pinned != "8.8.8.8:443" {
			return nil, errors.New("unapproved fixture dial")
		}
		return dialer.DialContext(ctx, network, server.Listener.Addr().String())
	})
	client := &http.Client{Transport: transport}
	if _, err := client.Get("https://wrong.example.test/v1"); err == nil {
		t.Fatal("TLS identity did not bind the URL hostname")
	}
	host := server.Certificate().DNSNames[0]
	response, err := client.Get("https://" + host + "/v1")
	if err != nil {
		t.Fatal(err)
	}
	response.Body.Close()
	changed.Store(true)
	if _, err := client.Get("https://" + host + "/v1"); err == nil || requests.Load() != 1 {
		t.Fatal("changed DNS was accepted or a prior connection was reused")
	}
}
