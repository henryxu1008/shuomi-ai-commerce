package einomodel

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"task-processor/internal/aicapability"
)

func TestEinoAdmittedComponentsUseOriginalSingleSendAndReferenceUsage(t *testing.T) {
	for _, tc := range []struct{ name, adapter, provider, policy, model, endpoint, response, path string }{
		{"gemini", "google-generate-content", "google", "eino-gemini-v0.1.36-text-v1", "synthetic-gemini", "https://generativelanguage.googleapis.com",
			`{"candidates":[{"index":0,"content":{"parts":[{"text":"hello"}],"role":"model"},"finishReason":"STOP"}],"modelVersion":"synthetic-gemini","responseId":"fixture","usageMetadata":{"promptTokenCount":10,"candidatesTokenCount":5,"totalTokenCount":15}}`, "/v1beta/models/synthetic-gemini:generateContent"},
		{"qwen", "qwen-compatible", "DASHSCOPE", "eino-qwen-v0.1.9-text-v1", "synthetic-qwen", "https://dashscope.aliyuncs.com/compatible-mode/v1",
			`{"id":"fixture","object":"chat.completion","created":1,"model":"synthetic-qwen","choices":[{"index":0,"message":{"role":"assistant","content":"hello"},"finish_reason":"stop"}],"usage":{"prompt_tokens":10,"completion_tokens":5,"total_tokens":15}}`, "/compatible-mode/v1/chat/completions"},
		{"claude", "claude-native", "anthropic", "eino-claude-v0.1.26-text-v1", "synthetic-claude", "https://api.anthropic.com",
			`{"id":"fixture","type":"message","role":"assistant","content":[{"type":"text","text":"hello"}],"model":"synthetic-claude","stop_reason":"end_turn","usage":{"input_tokens":10,"cache_read_input_tokens":0,"cache_creation_input_tokens":0,"output_tokens":5}}`, "/v1/messages"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			sends, gates := 0, 0
			cfg := GuardConfig{Adapter: AdapterKind(tc.adapter), ProviderID: tc.provider, AdapterPolicyVersion: tc.policy,
				BillingUsageVersion: aicapability.ReferenceTextTotalsV1, Endpoint: tc.endpoint, ModelID: tc.model,
				MaximumOutputTokens: 32, MaximumRequestBytes: 128 << 10, MaximumResponseBytes: 256 << 10, Timeout: time.Second}
			if tc.name == "gemini" {
				cfg.ThinkingMode = "low"
			}
			guard, err := NewGuardedClient(cfg, imageFixtureTransport(func(r *http.Request) (*http.Response, error) {
				sends++
				require.Equal(t, tc.path, r.URL.EscapedPath())
				wire, err := io.ReadAll(r.Body)
				require.NoError(t, err)
				require.NotContains(t, string(wire), "tools")
				require.Empty(t, r.URL.RawQuery, "a secret must not be put in URL query")
				if tc.name == "qwen" {
					require.Contains(t, string(wire), `"enable_thinking":false`)
				}
				return &http.Response{StatusCode: 200, Header: http.Header{"Content-Type": []string{"application/json"}}, Body: io.NopCloser(strings.NewReader(tc.response))}, nil
			}), func(context.Context) error { gates++; return nil })
			require.NoError(t, err)
			component, err := NewComponent(context.Background(), ComponentConfig{ThinkingMode: cfg.ThinkingMode, Adapter: cfg.Adapter, ProviderID: cfg.ProviderID,
				AdapterPolicyVersion: cfg.AdapterPolicyVersion, Endpoint: cfg.Endpoint, APIKey: "synthetic-only", ModelID: cfg.ModelID, MaximumOutputTokens: cfg.MaximumOutputTokens}, guard)
			require.NoError(t, err)
			out, err := GenerateText(context.Background(), component, "system", "prompt", guard)
			require.NoError(t, err)
			require.Equal(t, "hello", out.Content)
			require.NotNil(t, out.Usage.BillableUsage)
			require.Equal(t, aicapability.ReferenceTextTotalsV1, out.Usage.BillableUsage.MappingVersion)
			require.EqualValues(t, 10, out.Usage.BillableUsage.InputText)
			require.EqualValues(t, 5, out.Usage.BillableUsage.OutputText)
			_, err = GenerateText(context.Background(), component, "system", "prompt", guard)
			require.Error(t, err)
			require.Equal(t, 1, sends)
			require.Equal(t, 1, gates)
		})
	}
}
