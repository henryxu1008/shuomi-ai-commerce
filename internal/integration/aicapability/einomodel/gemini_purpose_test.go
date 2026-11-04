package einomodel

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"task-processor/internal/aicapability"
)

func TestEinoGeminiThinkingParametersAreBoundToTheQualifiedPurpose(t *testing.T) {
	for _, mode := range []string{"low", "minimal", "budget-zero"} {
		t.Run(mode, func(t *testing.T) {
			cfg := GuardConfig{Adapter: AdapterGoogleGenerateContent, ProviderID: "google", AdapterPolicyVersion: aicapability.EinoGeminiTextPolicy,
				ModelID: "m", Endpoint: "https://generativelanguage.googleapis.com", BillingUsageVersion: aicapability.ReferenceTextTotalsV1,
				MaximumOutputTokens: 32, MaximumRequestBytes: 8192, MaximumResponseBytes: 8192, Timeout: time.Second}
			require.NoError(t, json.Unmarshal([]byte(`{"ThinkingMode":"`+mode+`"}`), &cfg))
			sends := 0
			g, err := NewGuardedClient(cfg, imageFixtureTransport(func(r *http.Request) (*http.Response, error) {
				sends++
				raw, e := io.ReadAll(r.Body)
				require.NoError(t, e)
				var body struct {
					Generation struct {
						Thinking map[string]any `json:"thinkingConfig"`
					} `json:"generationConfig"`
				}
				require.NoError(t, json.Unmarshal(raw, &body))
				if mode == "budget-zero" {
					require.EqualValues(t, 0, body.Generation.Thinking["thinkingBudget"])
					require.NotContains(t, body.Generation.Thinking, "thinkingLevel")
				} else {
					require.Equal(t, strings.ToUpper(mode), body.Generation.Thinking["thinkingLevel"])
					require.NotContains(t, body.Generation.Thinking, "thinkingBudget", "different model generations must not receive both controls")
				}
				return &http.Response{StatusCode: 200, Header: http.Header{"Content-Type": []string{"application/json"}}, Body: io.NopCloser(strings.NewReader(`{"candidates":[{"content":{"role":"model","parts":[{"text":"hello"}]},"finishReason":"STOP"}],"modelVersion":"m","usageMetadata":{"promptTokenCount":10,"candidatesTokenCount":5,"totalTokenCount":15}}`))}, nil
			}), func(context.Context) error { return nil })
			require.NoError(t, err)
			cc := ComponentConfig{Adapter: cfg.Adapter, ProviderID: cfg.ProviderID, AdapterPolicyVersion: cfg.AdapterPolicyVersion,
				Endpoint: cfg.Endpoint, ModelID: cfg.ModelID, APIKey: "synthetic-only", MaximumOutputTokens: cfg.MaximumOutputTokens}
			require.NoError(t, json.Unmarshal([]byte(`{"ThinkingMode":"`+mode+`"}`), &cc))
			c, err := NewComponent(context.Background(), cc, g)
			require.NoError(t, err)
			changed := cc
			changed.ThinkingMode = "auto"
			_, err = NewComponent(context.Background(), changed, g)
			require.ErrorIs(t, err, ErrInvalid, "component options cannot drift from the frozen guarded purpose")
			_, err = GenerateText(context.Background(), c, "system", "prompt", g)
			require.NoError(t, err)
			require.Equal(t, 1, sends)
		})
	}
}

func TestEinoGeminiUnspecifiedOrUnadmittedThinkingNeverConstructsATransport(t *testing.T) {
	for _, mode := range []string{"", "auto", "medium", "high"} {
		_, err := NewGuardedClient(GuardConfig{ThinkingMode: mode, Adapter: AdapterGoogleGenerateContent, ProviderID: "google",
			AdapterPolicyVersion: aicapability.EinoGeminiTextPolicy, Endpoint: "https://generativelanguage.googleapis.com", ModelID: "m",
			MaximumOutputTokens: 32, MaximumRequestBytes: 8192, MaximumResponseBytes: 8192, Timeout: time.Second},
			imageFixtureTransport(func(*http.Request) (*http.Response, error) {
				t.Fatal("unadmitted parameter reached transport")
				return nil, nil
			}),
			func(context.Context) error { t.Fatal("unadmitted parameter reached final gate"); return nil })
		require.ErrorIs(t, err, ErrInvalid)
	}
}
