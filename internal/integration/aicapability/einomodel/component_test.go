package einomodel

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/http/httptrace"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestGuardedDefaultTransportUsesHTTP1OverTLS(t *testing.T) {
	// Clone initializes the standard transport's HTTP/2 defaults. The guarded
	// client must remain HTTP/1.1 even when it inherits that initialized config.
	defaults := http.DefaultTransport.(*http.Transport).Clone()
	require.NotNil(t, defaults.TLSClientConfig)
	require.Contains(t, defaults.TLSClientConfig.NextProtos, "h2")

	for _, enableHTTP2 := range []bool{true, false} {
		t.Run(fmt.Sprintf("server_http2=%t", enableHTTP2), func(t *testing.T) {
			var sends, gates atomic.Int32
			requests := make(chan string, 1)
			server := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				sends.Add(1)
				select {
				case requests <- r.Proto:
				default:
				}
				w.Header().Set("Content-Type", "application/json")
				_, _ = fmt.Fprint(w, `{"id":"chatcmpl-test","object":"chat.completion","created":1,"model":"synthetic-openai","choices":[{"index":0,"message":{"role":"assistant","content":"hello"},"finish_reason":"stop"}],"usage":{"prompt_tokens":10,"completion_tokens":5,"total_tokens":15}}`)
			}))
			server.EnableHTTP2 = enableHTTP2
			server.StartTLS()
			defer server.Close()

			guard, err := NewGuardedClient(GuardConfig{
				Adapter: AdapterOpenAICompatible, Endpoint: server.URL, ModelID: "synthetic-openai",
				MaximumOutputTokens: 32, MaximumRequestBytes: 128 << 10,
				MaximumResponseBytes: 256 << 10, Timeout: 2 * time.Second,
			}, nil, func(context.Context) error { gates.Add(1); return nil })
			require.NoError(t, err)
			transport := guard.base.(*http.Transport)
			defer transport.CloseIdleConnections()
			// Only add trust for this loopback server. Keep the production ALPN,
			// protocol handlers and certificate/hostname verification intact.
			roots := x509.NewCertPool()
			roots.AddCert(server.Certificate())
			if transport.TLSClientConfig == nil {
				transport.TLSClientConfig = &tls.Config{}
			}
			transport.TLSClientConfig.RootCAs = roots
			require.False(t, transport.TLSClientConfig.InsecureSkipVerify)
			component, err := NewComponent(context.Background(), ComponentConfig{
				Adapter: AdapterOpenAICompatible, Endpoint: server.URL, APIKey: "synthetic-only",
				ModelID: "synthetic-openai", MaximumOutputTokens: 32,
			}, guard)
			require.NoError(t, err)
			handshakes := make(chan tls.ConnectionState, 1)
			ctx := httptrace.WithClientTrace(context.Background(), &httptrace.ClientTrace{
				TLSHandshakeDone: func(state tls.ConnectionState, err error) {
					if err == nil {
						select {
						case handshakes <- state:
						default:
						}
					}
				},
			})
			output, err := GenerateText(ctx, component, "system", "user", guard)
			select {
			case state := <-handshakes:
				t.Logf("ALPN=%q handlers=%d handoffs=%d requests=%d error=%v", state.NegotiatedProtocol, len(transport.TLSNextProto), guard.NetworkSends(), sends.Load(), err)
				require.NoError(t, err)
				// httptest's HTTP/2 server advertises only h2; Go also accepts
				// HTTP/1.1 there with no ALPN selection. Neither may select h2.
				require.Contains(t, []string{"", "http/1.1"}, state.NegotiatedProtocol)
				require.NotEmpty(t, state.VerifiedChains)
			default:
				t.Fatalf("no successful TLS handshake: %v", err)
			}
			require.Equal(t, "hello", output.Content)
			require.True(t, output.Usage.Known)
			require.EqualValues(t, 1, guard.NetworkSends())
			require.EqualValues(t, 1, gates.Load())
			require.EqualValues(t, 1, sends.Load())
			require.Equal(t, "HTTP/1.1", <-requests)
			_, err = GenerateText(ctx, component, "system", "user", guard)
			require.ErrorIs(t, err, ErrOutcomeUnknown)
			require.EqualValues(t, 1, sends.Load(), "a second invocation must not reach the server")
			require.EqualValues(t, 1, gates.Load())
		})
	}
	currentDefaults := http.DefaultTransport.(*http.Transport).Clone()
	require.Equal(t, defaults.TLSClientConfig.NextProtos, currentDefaults.TLSClientConfig.NextProtos)
	require.Same(t, defaults.TLSClientConfig.RootCAs, currentDefaults.TLSClientConfig.RootCAs)
}

func TestPinnedEinoComponentsUseOneGuardedSyntheticTransport(t *testing.T) {
	for _, tc := range []struct {
		name     string
		kind     AdapterKind
		model    string
		response string
	}{
		{
			name: "openai-compatible", kind: AdapterOpenAICompatible, model: "synthetic-openai",
			response: "{\"id\":\"chatcmpl-test\",\"object\":\"chat.completion\",\"created\":1,\"model\":\"synthetic-openai\",\"choices\":[{\"index\":0,\"message\":{\"role\":\"assistant\",\"content\":\"hello\"},\"finish_reason\":\"stop\"}],\"usage\":{\"prompt_tokens\":10,\"completion_tokens\":5,\"total_tokens\":15}}",
		},
		{
			name: "claude-native", kind: AdapterClaudeNative, model: "synthetic-claude",
			response: "{\"id\":\"msg_test\",\"type\":\"message\",\"role\":\"assistant\",\"content\":[{\"type\":\"text\",\"text\":\"hello\"}],\"model\":\"synthetic-claude\",\"stop_reason\":\"end_turn\",\"stop_sequence\":null,\"usage\":{\"input_tokens\":10,\"output_tokens\":5}}",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var sends atomic.Int32
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				sends.Add(1)
				require.Equal(t, "POST", r.Method)
				require.Equal(t, "application/json", r.Header.Get("Content-Type"))
				var payload map[string]any
				require.NoError(t, json.NewDecoder(r.Body).Decode(&payload))
				require.NotContains(t, payload, "enable_thinking", "other model protocols are unchanged")
				w.Header().Set("Content-Type", "application/json")
				_, _ = fmt.Fprint(w, tc.response)
			}))
			defer server.Close()
			guard, err := NewGuardedClient(GuardConfig{
				Adapter: tc.kind, Endpoint: server.URL, ModelID: tc.model,
				MaximumOutputTokens: 32, MaximumRequestBytes: 128 << 10,
				MaximumResponseBytes: 256 << 10, Timeout: 5 * time.Second,
			}, nil, func(context.Context) error { return nil })
			require.NoError(t, err)
			component, err := NewComponent(context.Background(), ComponentConfig{
				Adapter: tc.kind, Endpoint: server.URL, APIKey: "synthetic-only",
				ModelID: tc.model, MaximumOutputTokens: 32,
			}, guard)
			require.NoError(t, err)
			output, err := GenerateText(context.Background(), component, "system", "user", guard)
			if err != nil {
				t.Logf("synthetic output=%+v usage=%+v sends=%d", output, guard.Usage(), guard.NetworkSends())
			}
			require.NoError(t, err)
			require.Equal(t, "hello", output.Content)
			require.True(t, output.Usage.Known)
			require.Equal(t, 10, output.Usage.PromptTokens)
			require.Equal(t, 5, output.Usage.CompletionTokens)
			require.EqualValues(t, 1, sends.Load())
			require.EqualValues(t, 1, guard.NetworkSends())
		})
	}
}

func TestPinnedComponentsCannotRetryOrRedirectAfterHandoff(t *testing.T) {
	for _, tc := range []struct {
		kind  AdapterKind
		model string
	}{
		{AdapterOpenAICompatible, "synthetic-openai"},
		{AdapterClaudeNative, "synthetic-claude"},
	} {
		t.Run(string(tc.kind), func(t *testing.T) {
			var sends atomic.Int32
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				sends.Add(1)
				w.Header().Set("Retry-After", "0")
				http.Error(w, "try again", http.StatusTooManyRequests)
			}))
			defer server.Close()
			guard, err := NewGuardedClient(GuardConfig{
				Adapter: tc.kind, Endpoint: server.URL, ModelID: tc.model,
				MaximumOutputTokens: 32, MaximumRequestBytes: 128 << 10,
				MaximumResponseBytes: 256 << 10, Timeout: 2 * time.Second,
			}, nil, func(context.Context) error { return nil })
			require.NoError(t, err)
			component, err := NewComponent(context.Background(), ComponentConfig{
				Adapter: tc.kind, Endpoint: server.URL, APIKey: "synthetic-only",
				ModelID: tc.model, MaximumOutputTokens: 32,
			}, guard)
			require.NoError(t, err)
			_, err = GenerateText(context.Background(), component, "system", "user", guard)
			require.Error(t, err)
			require.EqualValues(t, 1, sends.Load())
			require.EqualValues(t, 1, guard.NetworkSends())
		})
	}
}
