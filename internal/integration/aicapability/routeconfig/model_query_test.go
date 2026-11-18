package routeconfiguration

import (
	"context"
	"crypto/tls"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	credential "task-processor/internal/aicapability/channelcredential"
	d "task-processor/internal/aicapability/routeconfig"
	governed "task-processor/internal/integration/aicapability/einomodel"
)

type queryCredentials struct {
	row     credential.Metadata
	resolve func() credential.Secret
}

func (s *queryCredentials) Metadata(context.Context, string, string) (credential.Metadata, error) {
	return s.row, nil
}
func (s *queryCredentials) Resolve(context.Context, string, string, string) (credential.Secret, error) {
	return s.resolve(), nil
}

func queryFixture(t *testing.T, handler http.Handler) (*ModelQuery, *queryCredentials, d.ModelBindingReference) {
	t.Helper()
	ref := d.ModelBindingReference{ID: "official-mini", Version: "exact-v1"}
	row := (credential.Metadata{OrganizationID: "org-a", ConnectionID: "openai-1", Provider: "openai", Channel: "openai-direct", Region: "global", BaseURL: "https://api.openai.com/v1", Revision: 2, Enabled: true, KeyPresent: true}).WithIdentity()
	secrets := &queryCredentials{row: row}
	secrets.resolve = func() credential.Secret { return credential.Secret{Metadata: secrets.row, APIKey: "test-fixture-key"} }
	scope := d.Scope{OrganizationID: "org-a", Capability: "product_agent_decision"}
	policy := &governed.RoutePolicy{}
	policy.Profile.ClientName = ref.ID
	policy.Profile.ProviderID = "openai"
	policy.Profile.ModelID = OpenAIQueryModel
	policy.AdmittedEndpointIdentityDigest = row.EndpointDigest
	catalog := &Catalog{definitions: map[d.Scope]map[string]Definition{scope: {ref.ID: {
		Scope: scope, Text: policy, ConnectionID: row.ConnectionID, ConnectionProvider: row.Provider,
		Manufacturer: "openai", Channel: "openai-direct", Region: row.Region, ReferenceVersion: ref.Version, ConnectionEndpoint: row.BaseURL,
	}}}}
	query := NewModelQuery(catalog, secrets, secrets)
	server := httptest.NewTLSServer(handler)
	t.Cleanup(server.Close)
	transport := query.transport.(*http.Transport)
	require.False(t, transport.ForceAttemptHTTP2)
	require.True(t, transport.DisableKeepAlives)
	require.Nil(t, transport.Proxy)
	transport.TLSClientConfig = &tls.Config{RootCAs: server.Client().Transport.(*http.Transport).TLSClientConfig.RootCAs, ServerName: "example.com", NextProtos: []string{"http/1.1"}}
	transport.DialContext = func(ctx context.Context, _, addr string) (net.Conn, error) {
		require.Equal(t, "api.openai.com:443", addr)
		return (&net.Dialer{}).DialContext(ctx, "tcp", server.Listener.Addr().String())
	}
	return query, secrets, ref
}

func TestOpenAIModelQueryUsesSingleFixedGetAndSafeOutcome(t *testing.T) {
	for _, tc := range []struct {
		name       string
		code       int
		body, want string
	}{
		{"match", 200, `{"object":"model","id":"gpt-5.4-mini-2026-03-17","owned_by":"vendor"}`, "found"},
		{"wrong model", 200, `{"object":"model","id":"different"}`, "unconfirmed"},
		{"malformed", 200, `{"object":"model","object":"model","id":"gpt-5.4-mini-2026-03-17"}`, "unconfirmed"},
		{"oversized", 200, strings.Repeat("x", 17<<10), "unconfirmed"},
		{"invalid key", 401, `{"error":{"code":"invalid_api_key","message":"test-fixture-key"}}`, "invalid-key"},
		{"ambiguous 401", 401, `{"error":{"message":"test-fixture-key"}}`, "authentication-rejected"},
		{"permission", 403, `{"error":{"code":"insufficient_permissions"}}`, "read-forbidden"},
		{"region", 403, `{"error":{"code":"unsupported_country_region_territory"}}`, "region-restricted"},
		{"not found", 404, "", "not-found"},
		{"rate limit", 429, `{"error":{"code":"rate_limit_exceeded"}}`, "rate-limited"},
		{"ambiguous 429", 429, "", "unconfirmed"},
		{"exhausted credits", 429, `{"error":{"code":"credit_balance_exhausted"}}`, "unavailable"},
		{"quota", 429, `{"error":{"code":"insufficient_quota"}}`, "unavailable"},
		{"redirect", 307, "", "unavailable"},
		{"server error", 500, "", "unavailable"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var sends atomic.Int32
			q, _, ref := queryFixture(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				sends.Add(1)
				if r.Method != "GET" || r.URL.RequestURI() != "/v1/models/gpt-5.4-mini-2026-03-17" ||
					r.Host != "api.openai.com" || r.Header.Get("Authorization") != "Bearer test-fixture-key" ||
					r.Header.Get("OpenAI-Project") != "" || r.Header.Get("OpenAI-Organization") != "" || r.ContentLength != 0 {
					t.Error("outbound request crossed the fixed query boundary")
				}
				w.Header().Set("Location", "https://example.com/redirect")
				if tc.name == "rate limit" {
					w.Header().Set("Retry-After", "30")
				}
				w.WriteHeader(tc.code)
				_, _ = io.WriteString(w, tc.body)
			}))
			result, err := q.Check(context.Background(), "org-a", "openai-1", 2, ref, func(context.Context) bool { return true })
			require.NoError(t, err)
			require.Equal(t, tc.want, result.Status)
			require.Equal(t, OpenAIQueryModel, result.Model)
			require.False(t, result.CheckedAt.IsZero())
			if tc.name == "rate limit" {
				require.NotNil(t, result.RetryAfter)
				require.Equal(t, 30*time.Second, result.RetryAfter.Sub(result.CheckedAt))
			}
			require.EqualValues(t, 1, sends.Load())
		})
	}
}

func TestOpenAIModelQueryRejectsChangedAuthorityAndCredential(t *testing.T) {
	for _, tc := range []struct {
		name              string
		adjust            func(*queryCredentials)
		org               string
		revision          int64
		denyAt, wantSends int
	}{
		{"no initial authority", nil, "org-a", 2, 1, 0},
		{"revoked before send", nil, "org-a", 2, 2, 0},
		{"revoked before result", nil, "org-a", 2, 3, 1},
		{"different organization", nil, "org-b", 2, 0, 0},
		{"old revision", nil, "org-a", 1, 0, 0},
		{"disabled", func(s *queryCredentials) { s.row.Enabled = false }, "org-a", 2, 0, 0},
		{"proxy URL", func(s *queryCredentials) { s.row.BaseURL = "https://proxy.invalid/v1"; s.row = s.row.WithIdentity() }, "org-a", 2, 0, 0},
		{"resolution mismatch", func(s *queryCredentials) {
			s.resolve = func() credential.Secret {
				changed := s.row
				changed.Revision++
				return credential.Secret{Metadata: changed.WithIdentity(), APIKey: "stale"}
			}
		}, "org-a", 2, 0, 0},
		{"rotation after resolve", func(s *queryCredentials) {
			s.resolve = func() credential.Secret {
				secret := credential.Secret{Metadata: s.row, APIKey: "stale"}
				s.row.Revision++
				s.row = s.row.WithIdentity()
				return secret
			}
		}, "org-a", 2, 0, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var sends atomic.Int32
			q, s, ref := queryFixture(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { sends.Add(1); w.WriteHeader(404) }))
			if tc.adjust != nil {
				tc.adjust(s)
			}
			gates := 0
			result, err := q.Check(context.Background(), tc.org, "openai-1", tc.revision, ref, func(context.Context) bool { gates++; return gates != tc.denyAt })
			require.Error(t, err)
			require.Empty(t, result.Status)
			require.EqualValues(t, tc.wantSends, sends.Load())
		})
	}
}

func TestOpenAIModelQueryDoesNotRetryTimedOutTransport(t *testing.T) {
	var sends atomic.Int32
	q, _, ref := queryFixture(t, http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) { sends.Add(1); <-r.Context().Done() }))
	ctx, cancel := context.WithTimeout(context.Background(), 750*time.Millisecond)
	defer cancel()
	_, err := q.Check(ctx, "org-a", "openai-1", 2, ref, func(context.Context) bool { return true })
	require.Error(t, err)
	require.EqualValues(t, 1, sends.Load())
}
