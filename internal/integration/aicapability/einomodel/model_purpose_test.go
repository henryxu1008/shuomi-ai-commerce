package einomodel

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"task-processor/internal/aicapability"
)

func TestEinoSingleImageUsesActualComponentsOriginalClaimAndOneBillingMapping(t *testing.T) {
	for _, tc := range []struct{ name, adapter, provider, policy, usage, endpoint string }{
		{"gemini", "google-generate-content", "google", aicapability.EinoGeminiImagePolicy, aicapability.GeminiUsageMapping, "https://generativelanguage.googleapis.com"},
		{"qwen", "qwen-compatible", "DASHSCOPE", aicapability.EinoQwenImagePolicy, aicapability.QwenUsageMapping, "https://dashscope.aliyuncs.com/compatible-mode/v1"},
		{"openai", "openai-compatible", "openai", aicapability.EinoOpenAIImagePolicy, "openai-text-v1", "https://api.openai.com/v1"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			in := singleImageExecutionInput(t)
			p := &in.Profile
			p.ProviderID, p.ModelID, p.AdapterKind = tc.provider, "synthetic-"+tc.name, tc.adapter
			p.AdapterPolicyVersion, p.UsageMappingVersion = tc.policy, tc.usage
			p.ImagePromptTokenEstimate, p.EndpointIdentityDigest = 3001, EndpointIdentityDigest(tc.endpoint)
			if tc.name == "gemini" {
				p.ThinkingMode = "low"
			}
			p.OfficialTariff.Model, p.OfficialTariff.Channel = p.ModelID, "fixture-"+tc.name
			p.OfficialTariff.UsageVersion, p.OfficialTariff.Rates.CachedInputText = aicapability.ReferenceTextTotalsV1, 0
			q, err := aicapability.QuoteText(in)
			require.NoError(t, err)
			ledger, sends := &recordingLedger{}, 0
			e := &Executor{Ledger: ledger, Admission: NewBoundedAdmission(),
				Resolve: func(context.Context, aicapability.TextInputIdentity) (QualifiedRoute, error) {
					return QualifiedRoute{Profile: *p, Endpoint: tc.endpoint, APIKey: "synthetic-only"}, nil
				}, Authorize: func(context.Context, aicapability.TextInputIdentity) error { return nil },
				BaseTransport: imageFixtureTransport(func(r *http.Request) (*http.Response, error) {
					sends++
					raw, err := io.ReadAll(r.Body)
					require.NoError(t, err)
					if tc.name == "gemini" {
						require.Contains(t, string(raw), `"inlineData"`)
						require.NotContains(t, string(raw), "data:image")
					} else {
						require.Contains(t, string(raw), "data:image/png;base64,")
					}
					body := fmt.Sprintf(`{"id":"fixture","object":"chat.completion","created":1,"model":%q,"choices":[{"index":0,"message":{"role":"assistant","content":"readonly observation"},"finish_reason":"stop"}],"usage":{"prompt_tokens":30,"completion_tokens":5,"total_tokens":35}}`, p.ModelID)
					if tc.name == "gemini" {
						body = fmt.Sprintf(`{"candidates":[{"content":{"role":"model","parts":[{"text":"readonly observation"}]},"finishReason":"STOP"}],"modelVersion":%q,"usageMetadata":{"promptTokenCount":30,"candidatesTokenCount":5,"totalTokenCount":35}}`, p.ModelID)
					}
					return &http.Response{StatusCode: 200, Header: http.Header{"Content-Type": []string{"application/json"}}, Body: io.NopCloser(strings.NewReader(body))}, nil
				})}
			out, err := e.Generate(context.Background(), in, q, nil)
			require.NoError(t, err)
			require.Equal(t, "readonly observation", out.Content)
			require.Len(t, ledger.records, 2)
			require.Equal(t, aicapability.InvocationSucceeded, ledger.records[1].Outcome)
			require.Equal(t, p.OfficialTariff, ledger.records[1].OfficialTariff)
			require.NotNil(t, ledger.records[1].BillableUsage)
			require.Equal(t, aicapability.ReferenceTextTotalsV1, ledger.records[1].BillableUsage.MappingVersion)
			require.EqualValues(t, 30, ledger.records[1].BillableUsage.InputText, "image input is already in the provider input total")
			_, err = e.Generate(context.Background(), in, q, nil)
			require.ErrorIs(t, err, ErrOutcomeUnknown)
			require.Equal(t, 1, sends)
		})
	}
}

func TestEinoNativeReferenceUsageRequiresRawEvidence(t *testing.T) {
	for _, tc := range []struct {
		name          string
		adapter       AdapterKind
		model, raw    string
		input, output int
		known         bool
	}{
		{"gemini-basic", AdapterGoogleGenerateContent, "m", `{"modelVersion":"m","usageMetadata":{"promptTokenCount":10,"candidatesTokenCount":5,"totalTokenCount":15}}`, 10, 5, true},
		{"gemini-thoughts", AdapterGoogleGenerateContent, "m", `{"modelVersion":"m","usageMetadata":{"promptTokenCount":10,"candidatesTokenCount":5,"thoughtsTokenCount":3,"totalTokenCount":18}}`, 10, 8, true},
		{"gemini-derived-thoughts", AdapterGoogleGenerateContent, "m", `{"modelVersion":"m","usageMetadata":{"promptTokenCount":10,"candidatesTokenCount":5,"totalTokenCount":18}}`, 10, 8, true},
		{"gemini-missing-input", AdapterGoogleGenerateContent, "m", `{"modelVersion":"m","usageMetadata":{"candidatesTokenCount":5,"totalTokenCount":15}}`, 0, 0, false},
		{"gemini-missing-total", AdapterGoogleGenerateContent, "m", `{"modelVersion":"m","usageMetadata":{"promptTokenCount":10,"candidatesTokenCount":5}}`, 0, 0, false},
		{"gemini-contradiction", AdapterGoogleGenerateContent, "m", `{"modelVersion":"m","usageMetadata":{"promptTokenCount":10,"candidatesTokenCount":5,"thoughtsTokenCount":4,"totalTokenCount":18}}`, 0, 0, false},
		{"gemini-wrong-model", AdapterGoogleGenerateContent, "different", `{"modelVersion":"m","usageMetadata":{"promptTokenCount":10,"candidatesTokenCount":5,"totalTokenCount":15}}`, 0, 0, false},
		{"claude-basic", AdapterClaudeNative, "m", `{"model":"m","usage":{"input_tokens":10,"cache_read_input_tokens":0,"cache_creation_input_tokens":0,"output_tokens":5}}`, 10, 5, true},
		{"claude-cache", AdapterClaudeNative, "m", `{"model":"m","usage":{"input_tokens":10,"cache_read_input_tokens":3,"cache_creation_input_tokens":2,"output_tokens":5}}`, 15, 5, true},
		{"claude-missing", AdapterClaudeNative, "m", `{"model":"m","usage":{"output_tokens":5}}`, 0, 0, false},
		{"claude-negative", AdapterClaudeNative, "m", `{"model":"m","usage":{"input_tokens":-1,"output_tokens":5}}`, 0, 0, false},
		{"qwen-zero", AdapterQwenCompatible, "m", `{"model":"m","usage":{"prompt_tokens":0,"completion_tokens":0,"total_tokens":0}}`, 0, 0, true},
		{"qwen-missing-zero", AdapterQwenCompatible, "m", `{"model":"m","usage":{"total_tokens":0}}`, 0, 0, false},
		{"qwen-duplicate", AdapterQwenCompatible, "m", `{"model":"m","usage":{"prompt_tokens":1,"prompt_tokens":2,"completion_tokens":1,"total_tokens":2}}`, 0, 0, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			u := observeNativeReferenceUsage([]byte(tc.raw), GuardConfig{Adapter: tc.adapter, ModelID: tc.model})
			require.Equal(t, tc.known, u.Known)
			if !tc.known {
				require.Nil(t, u.BillableUsage)
				return
			}
			require.Equal(t, tc.input, u.PromptTokens)
			require.Equal(t, tc.output, u.CompletionTokens)
			require.NotNil(t, u.BillableUsage)
			require.Equal(t, aicapability.ReferenceTextTotalsV1, u.BillableUsage.MappingVersion)
		})
	}
}

func TestEinoProtocolDriftIsRejectedBeforeFinalGate(t *testing.T) {
	for _, tc := range []struct {
		adapter                               AdapterKind
		provider, policy, endpoint, path, raw string
		mutate                                []func(map[string]any)
	}{
		{AdapterGoogleGenerateContent, "google", aicapability.EinoGeminiTextPolicy, "https://generativelanguage.googleapis.com", "/v1beta/models/m:generateContent",
			`{"systemInstruction":{"parts":[{"text":"system"}]},"contents":[{"role":"user","parts":[{"text":"prompt"}]}],"generationConfig":{"maxOutputTokens":32,"responseModalities":["TEXT"],"thinkingConfig":{"thinkingBudget":0}}}`,
			[]func(map[string]any){
				func(e map[string]any) { e["tools"] = []any{} },
				func(e map[string]any) { e["cachedContent"] = "foreign" },
				func(e map[string]any) { e["generationConfig"].(map[string]any)["maxOutputTokens"] = 33 },
				func(e map[string]any) {
					e["generationConfig"].(map[string]any)["thinkingConfig"].(map[string]any)["thinkingBudget"] = -1
				},
				func(e map[string]any) {
					e["generationConfig"].(map[string]any)["responseModalities"] = []string{"IMAGE"}
				},
			}},
		{AdapterQwenCompatible, "DASHSCOPE", aicapability.EinoQwenTextPolicy, "https://dashscope.aliyuncs.com/compatible-mode/v1", "/compatible-mode/v1/chat/completions",
			`{"model":"m","max_tokens":32,"enable_thinking":false,"chat_template_kwargs":{"enable_thinking":false},"messages":[{"role":"system","content":"system"},{"role":"user","content":"prompt"}]}`,
			[]func(map[string]any){
				func(e map[string]any) { e["tools"] = []any{} },
				func(e map[string]any) { e["models"] = []string{"fallback"} },
				func(e map[string]any) { e["stream"] = true },
				func(e map[string]any) { e["max_tokens"] = 33 },
				func(e map[string]any) { e["enable_thinking"] = true },
				func(e map[string]any) { e["chat_template_kwargs"].(map[string]any)["enable_thinking"] = true },
				func(e map[string]any) { e["model"] = "other" },
			}},
	} {
		t.Run(string(tc.adapter), func(t *testing.T) {
			cfg := GuardConfig{Adapter: tc.adapter, ProviderID: tc.provider, AdapterPolicyVersion: tc.policy, ModelID: "m",
				BillingUsageVersion: aicapability.ReferenceTextTotalsV1, Endpoint: tc.endpoint, MaximumOutputTokens: 32, MaximumRequestBytes: 8192, MaximumResponseBytes: 8192, Timeout: time.Second}
			if tc.adapter == AdapterGoogleGenerateContent {
				cfg.ThinkingMode = "budget-zero"
			}
			require.True(t, validWire([]byte(tc.raw), cfg), "valid pinned SDK shape")
			for _, mutate := range tc.mutate {
				var e map[string]any
				require.NoError(t, json.Unmarshal([]byte(tc.raw), &e))
				mutate(e)
				raw, err := json.Marshal(e)
				require.NoError(t, err)
				sends, gates := 0, 0
				g, err := NewGuardedClient(cfg, imageFixtureTransport(func(*http.Request) (*http.Response, error) { sends++; return nil, nil }), func(context.Context) error { gates++; return nil })
				require.NoError(t, err)
				target, err := url.Parse(tc.endpoint)
				require.NoError(t, err)
				target.Path = tc.path
				r, err := http.NewRequest(http.MethodPost, target.String(), strings.NewReader(string(raw)))
				require.NoError(t, err)
				_, err = g.RoundTrip(r)
				require.ErrorIs(t, err, ErrNotDispatched)
				require.Zero(t, sends)
				require.Zero(t, gates)
			}
		})
	}
}

func TestEinoResponseAliasRequiresTheFrozenManufacturerBinding(t *testing.T) {
	raw := []byte(`{"model":"manufacturer-model","usage":{"prompt_tokens":10,"completion_tokens":5,"total_tokens":15}}`)
	cfg := GuardConfig{Adapter: AdapterQwenCompatible, ModelID: "vendor/manufacturer-model"}
	require.False(t, observeNativeReferenceUsage(raw, cfg).Known)
	cfg.ResponseModelID = "manufacturer-model"
	require.True(t, observeNativeReferenceUsage(raw, cfg).Known)
	cfg.ResponseModelID = "other-model"
	require.False(t, observeNativeReferenceUsage(raw, cfg).Known)
}

func TestEinoClaudeMissingAggregateInputPartsCannotProveZeroOrRelease(t *testing.T) {
	for _, raw := range []string{
		`{"model":"m","usage":{"input_tokens":0,"output_tokens":0}}`,
		`{"model":"m","usage":{"input_tokens":10,"output_tokens":5,"cache_creation_input_tokens":0}}`,
	} {
		u := observeNativeReferenceUsage([]byte(raw), GuardConfig{Adapter: AdapterClaudeNative, ModelID: "m"})
		require.False(t, u.Known, "Claude input_tokens excludes cache; absent cache counters cannot be normalized to zero")
		require.Nil(t, u.BillableUsage)
	}
}
