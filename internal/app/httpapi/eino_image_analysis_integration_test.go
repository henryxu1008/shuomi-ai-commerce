package httpapi

import (
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"

	"task-processor/internal/aicapability"
	governed "task-processor/internal/integration/aicapability/einomodel"
)

// Extend the original HTTP/PG owner fixture, not a parallel model runner.
func TestEinoImageAnalysisHTTPUsesOriginalOwnersAndReferenceLedger(t *testing.T) {
	for _, provider := range []string{"gemini", "qwen"} {
		t.Run(provider, func(t *testing.T) { testProductAgentOwners(t, "image analysis", provider) })
	}
}

func configureEinoImageAnalysisFixture(s *ProductAgentDependencies, row httpModelFixture, calls *atomic.Int32, provider string) {
	p := s.ImageAnalysisPolicies["B"]
	// The connection reader deliberately returns no model or protocol. Those
	// identities come from this trusted purpose definition, never from a Key.
	model := "synthetic-eino-" + provider
	p.Profile.ModelID = model
	p.Profile.EndpointIdentityDigest = governed.EndpointIdentityDigest(row.BaseURL)
	p.Profile.ImagePromptTokenEstimate = 3001
	p.Profile.OfficialTariff.Model = model
	p.Profile.OfficialTariff.Channel = "fixture-" + provider
	p.Profile.OfficialTariff.UsageVersion = aicapability.ReferenceTextTotalsV1
	p.Profile.OfficialTariff.Rates.CachedInputText = 0
	if provider == "gemini" {
		p.Profile.AdapterKind = string(governed.AdapterGoogleGenerateContent)
		p.Profile.ProviderID = "google"
		p.Profile.ThinkingMode = "low"
		p.Profile.AdapterPolicyVersion = aicapability.EinoGeminiImagePolicy
		p.Profile.UsageMappingVersion = aicapability.GeminiUsageMapping
	} else {
		p.Profile.AdapterKind = string(governed.AdapterQwenCompatible)
		p.Profile.ProviderID = "DASHSCOPE"
		p.Profile.AdapterPolicyVersion = aicapability.EinoQwenImagePolicy
		p.Profile.UsageMappingVersion = aicapability.QwenUsageMapping
	}
	s.ImageAnalysisPolicies["B"] = p
	s.TextTransport = analysisFixtureTransport(func(r *http.Request) (*http.Response, error) {
		calls.Add(1)
		raw, err := io.ReadAll(r.Body)
		if err != nil {
			return nil, err
		}
		if provider == "gemini" && !strings.Contains(string(raw), `"inlineData"`) || provider == "qwen" && !strings.Contains(string(raw), "data:image/png;base64,") {
			return nil, fmt.Errorf("missing frozen inline image")
		}
		opinion := `{"observations":[],"suggestions":[],"unknowns":["Synthetic fixture only; visual quality not evaluated."]}`
		reply := fmt.Sprintf(`{"id":"fixture","object":"chat.completion","created":1,"model":%q,"choices":[{"index":0,"message":{"role":"assistant","content":%q},"finish_reason":"stop"}],"usage":{"prompt_tokens":30,"completion_tokens":5,"total_tokens":35}}`, model, opinion)
		if provider == "gemini" {
			reply = fmt.Sprintf(`{"candidates":[{"content":{"role":"model","parts":[{"text":%q}]},"finishReason":"STOP"}],"modelVersion":%q,"usageMetadata":{"promptTokenCount":30,"candidatesTokenCount":5,"totalTokenCount":35}}`, opinion, model)
		}
		return &http.Response{StatusCode: 200, Header: http.Header{"Content-Type": []string{"application/json"}}, Body: io.NopCloser(strings.NewReader(reply))}, nil
	})
}
