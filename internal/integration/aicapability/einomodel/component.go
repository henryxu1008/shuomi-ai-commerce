// Package einomodel is the bounded Eino component and single-send transport
// seam shared by Chat planning and Product title execution.
package einomodel

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptrace"
	"net/url"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	claude "github.com/cloudwego/eino-ext/components/model/claude"
	gemini "github.com/cloudwego/eino-ext/components/model/gemini"
	openaimodel "github.com/cloudwego/eino-ext/components/model/openai"
	qwen "github.com/cloudwego/eino-ext/components/model/qwen"
	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
	"github.com/sirupsen/logrus"
	"google.golang.org/genai"
	"task-processor/internal/aicapability"
	"task-processor/internal/integration/googleinteractions"
)

type AdapterKind string

const (
	AdapterOpenAICompatible      AdapterKind = "openai-compatible"
	AdapterClaudeNative          AdapterKind = "claude-native"
	AdapterGoogleInteractions    AdapterKind = "google-interactions"
	AdapterGoogleGenerateContent AdapterKind = "google-generate-content"
	AdapterQwenCompatible        AdapterKind = "qwen-compatible"
)

var (
	ErrInvalid = errors.New("invalid bounded model configuration or output")
	// ErrObservedInvalidSettled is emitted only after the terminal recorder
	// accepted observed usage and its commercial settlement returned success.
	ErrObservedInvalidSettled = errors.New("observed invalid model output settled")
	// ErrNoDispatchReleased proves this call persisted the known-zero terminal
	// invocation and its ResourceAIPoint release returned success.
	ErrNoDispatchReleased = errors.New("model no-dispatch reservation released")
	ErrNotDispatched      = errors.New("model request not dispatched")
	ErrOutcomeUnknown     = errors.New("model outcome unknown; do not redispatch")
	ErrUsageUnknown       = errors.New("provider usage unknown")
)

type ComponentConfig struct {
	ThinkingMode         string
	Adapter              AdapterKind
	ProviderID           string
	AdapterPolicyVersion string
	Endpoint             string
	APIKey               string
	ModelID              string
	MaximumOutputTokens  int
}

// NewComponent always constructs an unbound model instance. Neither callers
// nor model output may supply Eino model.Option or a tool schema.
func NewComponent(ctx context.Context, cfg ComponentConfig, guard *GuardedClient) (model.BaseChatModel, error) {
	if ctx == nil || guard == nil || guard.Client == nil || cfg.APIKey == "" || cfg.ModelID == "" ||
		cfg.MaximumOutputTokens < 1 || cfg.MaximumOutputTokens > 65536 ||
		!ValidAdapterEndpoint(cfg.Adapter, cfg.Endpoint) || cfg.Adapter != guard.cfg.Adapter ||
		cfg.ProviderID != guard.cfg.ProviderID || cfg.AdapterPolicyVersion != guard.cfg.AdapterPolicyVersion ||
		cfg.Endpoint != guard.cfg.Endpoint || cfg.ModelID != guard.cfg.ModelID ||
		cfg.MaximumOutputTokens != guard.cfg.MaximumOutputTokens || cfg.ThinkingMode != guard.cfg.ThinkingMode {
		return nil, ErrInvalid
	}
	maxTokens := cfg.MaximumOutputTokens
	switch cfg.Adapter {
	case AdapterGoogleGenerateContent:
		thinking := geminiThinkingConfig(cfg.ThinkingMode)
		if thinking == nil {
			return nil, ErrInvalid
		}
		attempts := int32(1)
		client, err := genai.NewClient(ctx, &genai.ClientConfig{APIKey: cfg.APIKey, Backend: genai.BackendGeminiAPI, HTTPClient: guard.Client,
			HTTPOptions: genai.HTTPOptions{BaseURL: cfg.Endpoint, APIVersion: "v1beta", RetryOptions: &genai.HTTPRetryOptions{Attempts: &attempts}}})
		if err != nil {
			return nil, ErrInvalid
		}
		return gemini.NewChatModel(ctx, &gemini.Config{Client: client, Model: cfg.ModelID, MaxTokens: &maxTokens,
			ThinkingConfig:     thinking,
			ResponseModalities: []gemini.GeminiResponseModality{gemini.GeminiResponseModalityText}})
	case AdapterQwenCompatible:
		thinking := false
		return qwen.NewChatModel(ctx, &qwen.ChatModelConfig{APIKey: cfg.APIKey, BaseURL: cfg.Endpoint, Model: cfg.ModelID,
			MaxTokens: &maxTokens, HTTPClient: guard.Client, EnableThinking: &thinking})
	case AdapterGoogleInteractions:
		if cfg.ModelID != googleinteractions.Model {
			return nil, ErrInvalid
		}
		return &googleComponent{config: cfg, guard: guard}, nil
	case AdapterOpenAICompatible:
		if cfg.AdapterPolicyVersion == aicapability.EinoOpenAIImagePolicy {
			return openaimodel.NewChatModel(ctx, &openaimodel.ChatModelConfig{APIKey: cfg.APIKey, BaseURL: cfg.Endpoint, Model: cfg.ModelID,
				MaxTokens: &maxTokens, HTTPClient: guard.Client, ExtraFields: map[string]any{"n": 1, "store": false}})
		}
		if cfg.AdapterPolicyVersion == aicapability.SingleImageMiniAdapterPolicy || cfg.AdapterPolicyVersion == TitlePilotOpenAIAdapterPolicy {
			return openaimodel.NewChatModel(ctx, &openaimodel.ChatModelConfig{
				APIKey: cfg.APIKey, BaseURL: cfg.Endpoint, Model: cfg.ModelID,
				MaxCompletionTokens: &maxTokens, ReasoningEffort: openaimodel.ReasoningEffortLevel("none"),
				HTTPClient: guard.Client, ExtraFields: map[string]any{"n": 1, "store": false, "service_tier": "default"},
			})
		}
		var extra map[string]any
		if cfg.AdapterPolicyVersion == DashScopeDeepSeekV41FlashNonThinkingPolicy {
			extra = map[string]any{"enable_thinking": false}
		}
		return openaimodel.NewChatModel(ctx, &openaimodel.ChatModelConfig{
			APIKey: cfg.APIKey, BaseURL: cfg.Endpoint, Model: cfg.ModelID,
			MaxTokens: &maxTokens, HTTPClient: guard.Client, ExtraFields: extra,
		})
	case AdapterClaudeNative:
		endpoint := cfg.Endpoint
		return claude.NewChatModel(ctx, &claude.Config{
			APIKey: cfg.APIKey, BaseURL: &endpoint, Model: cfg.ModelID,
			MaxTokens: maxTokens, HTTPClient: guard.Client,
		})
	default:
		return nil, ErrInvalid
	}
}

type GuardConfig struct {
	ThinkingMode string
	// Exact canonical identity is supplied only by the qualified frozen
	// manufacturer/wire binding; a response cannot introduce a new alias.
	ResponseModelID      string
	SingleImage          *SingleImageRequest `json:"-"`
	Logger               *logrus.Entry       `json:"-"`
	Adapter              AdapterKind
	ProviderID           string
	AdapterPolicyVersion string
	BillingUsageVersion  string
	Endpoint             string
	ModelID              string
	MaximumOutputTokens  int
	MaximumRequestBytes  int64
	MaximumResponseBytes int64
	Timeout              time.Duration
}

type ObservedUsage struct {
	BillableUsage    *aicapability.BillableModelUsage
	PromptTokens     int
	CompletionTokens int
	TotalTokens      int
	Known            bool
}

type GuardedClient struct {
	Client            *http.Client
	cfg               GuardConfig
	endpoint          *url.URL
	base              http.RoundTripper
	finalGate         func(context.Context) error
	sent              atomic.Bool
	sendMu            sync.Mutex
	mu                sync.Mutex
	usage             ObservedUsage
	providerRequestID string
	diagnostic        *textDiagnostics
}

func (g *GuardedClient) diagnosticsFor(ctx context.Context) *textDiagnostics {
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.diagnostic == nil {
		g.diagnostic, _ = ctx.Value(textDiagnosticKey{}).(*textDiagnostics)
		if g.diagnostic == nil {
			g.diagnostic = newTextDiagnostics(ctx, "", g.cfg.Logger)
		}
	}
	return g.diagnostic
}

func validEndpoint(raw string) bool {
	parsed, err := url.Parse(raw)
	if err != nil || parsed == nil || parsed.Hostname() == "" || parsed.User != nil ||
		parsed.RawQuery != "" || parsed.Fragment != "" {
		return false
	}
	if parsed.Scheme == "https" {
		return true
	}
	if parsed.Scheme != "http" {
		return false
	}
	host := parsed.Hostname()
	return strings.EqualFold(host, "localhost") || net.ParseIP(host) != nil && net.ParseIP(host).IsLoopback()
}

// ValidEndpoint uses the same endpoint admission rule as the final transport.
func ValidEndpoint(raw string) bool { return validEndpoint(raw) }

func ValidAdapterEndpoint(adapter AdapterKind, raw string) bool {
	if adapter == AdapterGoogleGenerateContent {
		u, err := url.Parse(raw)
		return err == nil && validEndpoint(raw) && (u.Path == "" || u.Path == "/")
	}
	if adapter == AdapterGoogleInteractions {
		return googleinteractions.ValidEndpoint(raw)
	}
	return validEndpoint(raw)
}

// NewGuardedClient creates a per-invocation transport. Even if an SDK retries,
// follows a redirect, or Generate is called twice, only one underlying model
// RoundTrip can occur. finalGate runs at the actual HTTP handoff.
func NewGuardedClient(cfg GuardConfig, base http.RoundTripper, finalGate func(context.Context) error) (*GuardedClient, error) {
	if (cfg.Adapter != AdapterOpenAICompatible && cfg.Adapter != AdapterClaudeNative && cfg.Adapter != AdapterGoogleInteractions &&
		cfg.Adapter != AdapterGoogleGenerateContent && cfg.Adapter != AdapterQwenCompatible) ||
		!ValidAdapterEndpoint(cfg.Adapter, cfg.Endpoint) || cfg.ModelID == "" || cfg.MaximumOutputTokens < 1 ||
		cfg.MaximumOutputTokens > 65536 || cfg.MaximumRequestBytes < 1 ||
		cfg.MaximumRequestBytes > 1<<20 || cfg.MaximumResponseBytes < 1 ||
		cfg.MaximumResponseBytes > 1<<20 || cfg.Timeout <= 0 || cfg.Timeout > 2*time.Minute ||
		finalGate == nil {
		return nil, ErrInvalid
	}
	if !validFixedAdapterPolicy(cfg.Adapter, cfg.ProviderID, cfg.ModelID, cfg.AdapterPolicyVersion) {
		return nil, ErrInvalid
	}
	if cfg.AdapterPolicyVersion == TitlePilotOpenAIAdapterPolicy &&
		(cfg.Endpoint != TitlePilotOpenAIEndpoint || cfg.MaximumOutputTokens != TitlePilotCompletionTokens) {
		return nil, ErrInvalid
	}
	if cfg.Adapter == AdapterGoogleGenerateContent {
		if !aicapability.ValidGeminiThinkingMode(cfg.ThinkingMode) {
			return nil, ErrInvalid
		}
	} else if cfg.ThinkingMode != "" {
		return nil, ErrInvalid
	}
	if aicapability.SingleImagePurpose(cfg.AdapterPolicyVersion) {
		if !validSingleImageRequest(cfg) {
			return nil, ErrInvalid
		}
		frozen := *cfg.SingleImage
		cfg.SingleImage = &frozen
	} else if cfg.SingleImage != nil {
		return nil, ErrInvalid
	}
	if cfg.Adapter == AdapterGoogleInteractions && (cfg.ModelID != googleinteractions.Model || cfg.MaximumRequestBytes > 128<<10 || cfg.MaximumResponseBytes > 256<<10) {
		return nil, ErrInvalid
	}
	endpoint, _ := url.Parse(cfg.Endpoint)
	if base == nil {
		transport := http.DefaultTransport.(*http.Transport).Clone()
		transport.DisableKeepAlives = true
		transport.TLSNextProto = map[string]func(string, *tls.Conn) http.RoundTripper{}
		// Clone can inherit h2 ALPN from the initialized default transport.
		// Advertise only the protocol handled by this single-send transport.
		if transport.TLSClientConfig == nil {
			transport.TLSClientConfig = &tls.Config{}
		}
		transport.TLSClientConfig.NextProtos = []string{"http/1.1"}
		base = transport
	}
	guard := &GuardedClient{cfg: cfg, endpoint: endpoint, base: base, finalGate: finalGate}
	guard.Client = &http.Client{Transport: guard, Timeout: cfg.Timeout,
		CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	return guard, nil
}

func (g *GuardedClient) NetworkSends() int {
	if g == nil || !g.sent.Load() {
		return 0
	}
	return 1
}

func (g *GuardedClient) Usage() ObservedUsage {
	if g == nil {
		return ObservedUsage{}
	}
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.usage
}

func (g *GuardedClient) ProviderRequestID() string {
	if g == nil {
		return ""
	}
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.providerRequestID
}

func (g *GuardedClient) RoundTrip(req *http.Request) (*http.Response, error) {
	if g == nil || req == nil || req.URL == nil || req.Method != http.MethodPost ||
		req.URL.Scheme != g.endpoint.Scheme || !strings.EqualFold(req.URL.Host, g.endpoint.Host) ||
		!strings.HasPrefix(req.URL.EscapedPath(), strings.TrimSuffix(g.endpoint.EscapedPath(), "/")+"/") ||
		req.URL.RawQuery != "" || req.URL.Fragment != "" || req.Body == nil ||
		g.cfg.AdapterPolicyVersion == TitlePilotOpenAIAdapterPolicy && req.URL.EscapedPath() != "/v1/chat/completions" ||
		g.cfg.Adapter == AdapterGoogleInteractions && req.URL.EscapedPath() != "/v1beta/interactions" ||
		g.cfg.Adapter == AdapterGoogleGenerateContent && req.URL.EscapedPath() != "/v1beta/models/"+g.cfg.ModelID+":generateContent" ||
		g.cfg.SingleImage != nil && g.cfg.Adapter != AdapterGoogleGenerateContent &&
			req.URL.EscapedPath() != strings.TrimSuffix(g.endpoint.EscapedPath(), "/")+"/chat/completions" {
		return nil, ErrNotDispatched
	}
	diagnostic := g.diagnosticsFor(req.Context())
	body, err := io.ReadAll(io.LimitReader(req.Body, g.cfg.MaximumRequestBytes+1))
	_ = req.Body.Close()
	if err != nil || int64(len(body)) > g.cfg.MaximumRequestBytes || !validWire(body, g.cfg) {
		diagnostic.emit(req.Context(), "wire_guard", "rejected_before_handoff")
		return nil, ErrNotDispatched
	}
	g.sendMu.Lock()
	if g.sent.Load() {
		g.sendMu.Unlock()
		diagnostic.emit(req.Context(), "handoff", "duplicate_blocked")
		return nil, ErrOutcomeUnknown
	}
	if err := g.finalGate(req.Context()); err != nil {
		g.sendMu.Unlock()
		diagnostic.emit(req.Context(), "final_gate", "rejected_before_handoff")
		return nil, errors.Join(ErrNotDispatched, err)
	}
	if !g.sent.CompareAndSwap(false, true) {
		g.sendMu.Unlock()
		return nil, ErrOutcomeUnknown
	}
	g.sendMu.Unlock()
	diagnostic.set("handoffs", 1)
	req.Body = io.NopCloser(bytes.NewReader(body))
	req.ContentLength = int64(len(body))
	req.GetBody = nil
	req = req.WithContext(httptrace.WithClientTrace(req.Context(), diagnostic.trace()))
	response, err := g.base.RoundTrip(req)
	if err != nil {
		diagnostic.emit(req.Context(), "transport", diagnosticErrorCategory(err))
		return nil, ErrOutcomeUnknown
	}
	if response == nil || response.Body == nil {
		diagnostic.emit(req.Context(), "transport", "response_missing")
		return nil, ErrOutcomeUnknown
	}
	diagnostic.response(response.StatusCode, response.Header.Get("X-Request-Id"), nil, g.cfg.Adapter)
	if g.cfg.Adapter == AdapterGoogleInteractions && (response.StatusCode != http.StatusOK || !strings.HasPrefix(response.Header.Get("Content-Type"), "application/json")) {
		_ = response.Body.Close()
		diagnostic.emit(req.Context(), "response", "http_or_content_type_rejected")
		return nil, ErrOutcomeUnknown
	}
	raw, err := io.ReadAll(io.LimitReader(response.Body, g.cfg.MaximumResponseBytes+1))
	_ = response.Body.Close()
	if err != nil || int64(len(raw)) > g.cfg.MaximumResponseBytes {
		category := "read_failed"
		if int64(len(raw)) > g.cfg.MaximumResponseBytes {
			category = "response_too_large"
		}
		diagnostic.emit(req.Context(), "response_body", category)
		return nil, ErrOutcomeUnknown
	}
	diagnostic.response(response.StatusCode, response.Header.Get("X-Request-Id"), raw, g.cfg.Adapter)
	diagnostic.set("response_bytes", len(raw))
	diagnostic.emit(req.Context(), "response", "response_received")
	if response.StatusCode >= 200 && response.StatusCode < 300 {
		usage := observeUsage(raw, g.cfg.Adapter)
		if g.cfg.Adapter == AdapterOpenAICompatible && g.cfg.BillingUsageVersion == aicapability.ReferenceTextTotalsV1 {
			usage = observeReferenceTextUsage(raw)
		}
		if g.cfg.BillingUsageVersion == aicapability.ReferenceTextTotalsV1 &&
			(g.cfg.Adapter == AdapterQwenCompatible || g.cfg.Adapter == AdapterGoogleGenerateContent || g.cfg.Adapter == AdapterClaudeNative ||
				g.cfg.AdapterPolicyVersion == aicapability.EinoOpenAIImagePolicy) {
			usage = observeNativeReferenceUsage(raw, g.cfg)
		}
		if g.cfg.AdapterPolicyVersion == DashScopeDeepSeekV41FlashNonThinkingPolicy {
			usage = observeDeepSeekUsage(raw)
		}
		if (g.cfg.SingleImage != nil || g.cfg.AdapterPolicyVersion == TitlePilotOpenAIAdapterPolicy) &&
			(usage.BillableUsage == nil || g.cfg.BillingUsageVersion != aicapability.ReferenceTextTotalsV1 && !validStrictOpenAIResponseUsage(raw, g.cfg.ModelID)) {
			usage = ObservedUsage{}
		}
		var providerID string
		if g.cfg.Adapter == AdapterGoogleInteractions {
			parsed := googleinteractions.Observe(raw, g.cfg.MaximumOutputTokens)
			usage = ObservedUsage{PromptTokens: parsed.PromptTokens, CompletionTokens: parsed.CompletionTokens, TotalTokens: parsed.TotalTokens, Known: parsed.UsageKnown}
			providerID = parsed.ID
		}
		g.mu.Lock()
		g.usage = usage
		g.providerRequestID = providerID
		g.mu.Unlock()
	}
	response.Body = io.NopCloser(bytes.NewReader(raw))
	response.ContentLength = int64(len(raw))
	return response, nil
}

func validWire(raw []byte, cfg GuardConfig) bool {
	if cfg.AdapterPolicyVersion == TitlePilotOpenAIAdapterPolicy {
		return validTitlePilotWire(raw, cfg.ModelID)
	}
	if cfg.Adapter == AdapterGoogleGenerateContent {
		return validGeminiWire(raw, cfg)
	}
	if cfg.Adapter == AdapterQwenCompatible || cfg.AdapterPolicyVersion == aicapability.EinoOpenAIImagePolicy {
		return validPurposeChatWire(raw, cfg)
	}
	if cfg.AdapterPolicyVersion == aicapability.SingleImageMiniAdapterPolicy {
		return validSingleImageWire(raw, cfg)
	}
	var envelope map[string]json.RawMessage
	if json.Unmarshal(raw, &envelope) != nil || envelope == nil {
		return false
	}
	var modelID string
	if json.Unmarshal(envelope["model"], &modelID) != nil || modelID != cfg.ModelID {
		return false
	}
	if cfg.AdapterPolicyVersion == DashScopeDeepSeekV41FlashNonThinkingPolicy &&
		!bytes.Equal(bytes.TrimSpace(envelope["enable_thinking"]), []byte("false")) {
		return false
	}
	if cfg.Adapter == AdapterGoogleInteractions {
		for _, key := range []string{"store", "background", "stream"} {
			if !bytes.Equal(bytes.TrimSpace(envelope[key]), []byte("false")) {
				return false
			}
		}
		for _, key := range []string{"tools", "tool_choice", "previous_interaction_id", "agent", "environment", "cached_content"} {
			if _, found := envelope[key]; found {
				return false
			}
		}
		var input, system string
		var generation struct {
			Max      int    `json:"max_output_tokens"`
			Thinking string `json:"thinking_level"`
		}
		return json.Unmarshal(envelope["input"], &input) == nil && input != "" &&
			json.Unmarshal(envelope["system_instruction"], &system) == nil && system != "" &&
			json.Unmarshal(envelope["generation_config"], &generation) == nil && generation.Thinking == "low" && generation.Max == cfg.MaximumOutputTokens
	}
	if _, found := envelope["tools"]; found {
		return false
	}
	if _, found := envelope["tool_choice"]; found {
		return false
	}
	if stream, found := envelope["stream"]; found && string(stream) != "false" {
		return false
	}
	var max int
	if cfg.Adapter == AdapterClaudeNative {
		if json.Unmarshal(envelope["max_tokens"], &max) != nil {
			return false
		}
	} else if value, found := envelope["max_completion_tokens"]; found {
		if json.Unmarshal(value, &max) != nil {
			return false
		}
	} else if json.Unmarshal(envelope["max_tokens"], &max) != nil {
		return false
	}
	return max > 0 && max <= cfg.MaximumOutputTokens
}

func observeUsage(raw []byte, adapter AdapterKind) ObservedUsage {
	if adapter == AdapterOpenAICompatible {
		var result struct {
			Usage *struct {
				Prompt     *int `json:"prompt_tokens"`
				Completion *int `json:"completion_tokens"`
				Total      *int `json:"total_tokens"`
			} `json:"usage"`
		}
		if json.Unmarshal(raw, &result) != nil || result.Usage == nil ||
			result.Usage.Prompt == nil || result.Usage.Completion == nil || result.Usage.Total == nil {
			return ObservedUsage{}
		}
		p, c, t := *result.Usage.Prompt, *result.Usage.Completion, *result.Usage.Total
		if p < 0 || c < 0 || t <= 0 || p+c != t {
			return ObservedUsage{}
		}
		observed := ObservedUsage{PromptTokens: p, CompletionTokens: c, TotalTokens: t, Known: true}
		if billable, err := ObserveOfficialOpenAITextUsage(raw); err == nil {
			observed.BillableUsage = &billable
		}
		return observed
	}
	var result struct {
		Usage *struct {
			Input      *int `json:"input_tokens"`
			Output     *int `json:"output_tokens"`
			CacheRead  int  `json:"cache_read_input_tokens"`
			CacheWrite int  `json:"cache_creation_input_tokens"`
		} `json:"usage"`
	}
	if json.Unmarshal(raw, &result) != nil || result.Usage == nil ||
		result.Usage.Input == nil || result.Usage.Output == nil {
		return ObservedUsage{}
	}
	// V1's two-category tariff has no cache rates. A cache-bearing native
	// response remains unknown until an admitted cost policy can map it.
	u := result.Usage
	if *u.Input < 0 || *u.Output < 0 || u.CacheRead != 0 || u.CacheWrite != 0 ||
		*u.Input+*u.Output <= 0 {
		return ObservedUsage{}
	}
	return ObservedUsage{PromptTokens: *u.Input, CompletionTokens: *u.Output,
		TotalTokens: *u.Input + *u.Output, Known: true}
}

type TextOutput struct {
	Content           string
	FinishReason      string
	Usage             ObservedUsage
	ProviderRequestID string
}

// GenerateText makes one non-streaming, no-tool call. Raw counter presence is
// supplied by the guarded transport, not inferred from Eino's optional ints.
func GenerateText(ctx context.Context, component model.BaseChatModel, system, prompt string, guard *GuardedClient) (TextOutput, error) {
	if ctx == nil || component == nil || guard == nil || system == "" || prompt == "" {
		return TextOutput{}, ErrInvalid
	}
	return generateMessages(ctx, component, []*schema.Message{schema.SystemMessage(system), schema.UserMessage(prompt)}, guard)
}

func generateMessages(ctx context.Context, component model.BaseChatModel, messages []*schema.Message, guard *GuardedClient) (TextOutput, error) {
	diagnostic := guard.diagnosticsFor(ctx)
	message, err := component.Generate(ctx, messages)
	usage := guard.Usage()
	diagnostic.set("usage_known", usage.Known)
	if err != nil {
		diagnostic.emit(ctx, "sdk", diagnostic.sdkCategory(err))
		if guard.NetworkSends() == 0 {
			return TextOutput{}, errors.Join(ErrNotDispatched, err)
		}
		return TextOutput{Usage: usage, ProviderRequestID: guard.ProviderRequestID()}, ErrOutcomeUnknown
	}
	output := TextOutput{Usage: usage, ProviderRequestID: guard.ProviderRequestID()}
	if message != nil && message.ResponseMeta != nil {
		output.Content = message.Content
		output.FinishReason = message.ResponseMeta.FinishReason
	}
	if !usage.Known {
		diagnostic.emit(ctx, "usage", "usage_missing_or_invalid")
		return output, ErrUsageUnknown
	}
	if message == nil || message.Role != schema.Assistant || message.ResponseMeta == nil ||
		message.ResponseMeta.Usage == nil || message.ResponseMeta.Usage.PromptTokens != usage.PromptTokens ||
		message.ResponseMeta.Usage.CompletionTokens != usage.CompletionTokens ||
		message.ResponseMeta.Usage.TotalTokens != usage.TotalTokens ||
		len(message.ToolCalls) != 0 || message.ToolCallID != "" ||
		!validComponentTextParts(message, guard.cfg.Adapter) ||
		message.ReasoningContent != "" || !safeComponentMetadata(message.Extra) ||
		message.Content == "" || len(message.Content) > 16384 ||
		(output.FinishReason != "stop" && output.FinishReason != "end_turn" && !(guard.cfg.Adapter == AdapterGoogleGenerateContent && output.FinishReason == "STOP")) {
		diagnostic.emit(ctx, "output_validation", "rejected")
		return output, ErrInvalid
	}
	return output, nil
}

func safeComponentMetadata(extra map[string]any) bool {
	for key, value := range extra {
		// The pinned OpenAI component records its request ID in Extra. It is
		// metadata, never text for the planner or title consumer.
		if key != "openai-request-id" || len(fmt.Sprint(value)) > 128 {
			return false
		}
	}
	return true
}
