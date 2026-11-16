package skillsandbox

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"io"
	"math/big"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	osb "github.com/alibaba/OpenSandbox/sdks/sandbox/go"
)

func TestControlFailureIsNotCapacityExhaustion(t *testing.T) {
	for _, tc := range []struct {
		name, body, reason string
		status             int
		err                error
	}{
		{name: "expired certificate", err: x509.CertificateInvalidError{Cert: &x509.Certificate{}, Reason: x509.Expired}, reason: "control_unavailable"},
		{name: "connection failure", err: errors.New("private control endpoint unreachable"), reason: "control_unavailable"},
		{name: "authentication", status: 401, body: `{}`, reason: "control_unavailable"},
		{name: "authorization", status: 403, body: `{}`, reason: "control_unavailable"},
		{name: "server unavailable", status: 503, body: `{}`, reason: "control_unavailable"},
		{name: "invalid response", status: 200, body: `{broken`, reason: "control_unavailable"},
		{name: "confirmed full", status: 200, body: `{"items":[],"pagination":{"totalItems":2}}`, reason: "capacity_unavailable"},
		{name: "more pages", status: 200, body: `{"items":[],"pagination":{"hasNextPage":true}}`, reason: "capacity_unavailable"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			executor, task, counts := lifecycleFixture(t, "", false, lifecycleOptions{})
			reads := 0
			executor.client.Transport = boundedTransport{origin: executor.origin, base: lifecycleTransport(func(r *http.Request) (*http.Response, error) {
				reads++
				if r.Method != http.MethodGet || r.URL.Path != "/v1/sandboxes" {
					t.Fatalf("unexpected mutation: %s %s", r.Method, r.URL.Path)
				}
				if tc.err != nil {
					return nil, tc.err
				}
				return &http.Response{StatusCode: tc.status, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(tc.body)), Request: r}, nil
			})}
			executor.lifecycle = osb.NewLifecycleClient(executor.origin+"/v1", "", osb.WithHTTPClient(executor.client))
			result := invokeCommerceReceipt(t, context.Background(), executor, task)
			created, commands, _ := counts()
			if result.Receipt.Outcome != "not_dispatched" || result.Receipt.Cleanup != "not_needed" || result.Receipt.Reason != tc.reason || reads != 1 || created != 0 || commands != 0 {
				t.Fatalf("wrong control receipt: %+v reads=%d creates=%d commands=%d", result.Receipt, reads, created, commands)
			}
		})
	}
}

func TestCheckControlUsesVerifiedTLSAndReadOnlySDK(t *testing.T) {
	for _, expired := range []bool{false, true} {
		t.Run(map[bool]string{false: "valid", true: "expired"}[expired], func(t *testing.T) {
			requests := 0
			server := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				requests++
				u, p, ok := r.BasicAuth()
				if r.Method != http.MethodGet || r.URL.Path != "/v1/sandboxes" || !ok || u != "fixture" || p != "synthetic-service-password" || r.Header.Get("OPEN-SANDBOX-API-KEY") != "synthetic-lifecycle-key" {
					t.Errorf("wrong read-only authenticated control request")
				}
				io.WriteString(w, `{"items":[],"pagination":{"totalItems":2}}`)
			}))
			key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
			if err != nil {
				t.Fatal(err)
			}
			now := time.Now()
			until := now.Add(time.Hour)
			if expired {
				until = now.Add(-time.Hour)
			}
			cert := &x509.Certificate{SerialNumber: big.NewInt(1), NotBefore: now.Add(-2 * time.Hour), NotAfter: until, IPAddresses: []net.IP{net.ParseIP("127.0.0.1")}, IsCA: true, BasicConstraintsValid: true, KeyUsage: x509.KeyUsageCertSign | x509.KeyUsageDigitalSignature, ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth}}
			der, err := x509.CreateCertificate(rand.Reader, cert, cert, &key.PublicKey, key)
			if err != nil {
				t.Fatal(err)
			}
			server.TLS = &tls.Config{Certificates: []tls.Certificate{{Certificate: [][]byte{der}, PrivateKey: key}}}
			server.StartTLS()
			defer server.Close()
			dir := t.TempDir()
			ca := filepath.Join(dir, "ca.pem")
			secret := filepath.Join(dir, "secret.json")
			if err = os.WriteFile(ca, pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}), 0600); err != nil {
				t.Fatal(err)
			}
			if err = os.WriteFile(secret, []byte(`{"user":"fixture","password":"synthetic-service-password","lifecycle":"synthetic-lifecycle-key"}`), 0600); err != nil {
				t.Fatal(err)
			}
			fixture, _, _ := lifecycleFixture(t, "", false, lifecycleOptions{})
			executor, err := New(Config{Domain: strings.TrimPrefix(server.URL, "https://"), CAFile: ca, CredentialFile: secret, TrialEvidence: "sha256:" + strings.Repeat("c", 64), Descriptor: fixture.Descriptor()})
			if err != nil {
				t.Fatal(err)
			}
			err = executor.CheckControl(context.Background())
			if expired {
				if err == nil || requests != 0 {
					t.Fatalf("expired TLS accepted: err=%v requests=%d", err, requests)
				}
			} else if err != nil || requests != 1 {
				t.Fatalf("valid control rejected: err=%v requests=%d", err, requests)
			}
		})
	}
}

func TestCheckControlHonorsCallerDeadline(t *testing.T) {
	executor, _, _ := lifecycleFixture(t, "", false, lifecycleOptions{})
	client := &http.Client{Transport: lifecycleTransport(func(r *http.Request) (*http.Response, error) { <-r.Context().Done(); return nil, r.Context().Err() })}
	executor.lifecycle = osb.NewLifecycleClient(executor.origin+"/v1", "", osb.WithHTTPClient(client))
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	start := time.Now()
	if executor.CheckControl(ctx) == nil {
		t.Fatal("cancellation accepted as readiness")
	}
	if time.Since(start) > time.Second {
		t.Fatal("preflight exceeded caller deadline")
	}
}
