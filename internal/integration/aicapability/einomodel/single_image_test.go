package einomodel

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"hash/crc32"
	"image"
	"image/png"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"task-processor/internal/aicapability"
)

type imageFixtureTransport func(*http.Request) (*http.Response, error)

func (f imageFixtureTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func singleImageExecutionInput(t *testing.T) aicapability.TextInputIdentity {
	t.Helper()
	var b bytes.Buffer
	require.NoError(t, png.Encode(&b, image.NewRGBA(image.Rect(0, 0, 2, 2))))
	content := b.Bytes()
	h := sha256.Sum256(content)
	p := testTextProfile("https://api.openai.com/v1")
	p.ProviderID, p.ModelID = "openai", aicapability.SingleImageMiniModel
	p.AdapterPolicyVersion, p.UsageMappingVersion = aicapability.SingleImageMiniAdapterPolicy, "openai-text-v1"
	p.OfficialTariff.Model, p.OfficialTariff.Channel = p.ModelID, "openai-direct"
	p.MaximumPromptTokens, p.MaximumInputBytes = 10000, 8192
	return aicapability.TextInputIdentity{OrganizationID: "org", ActorID: "actor", MemberID: "member", AgentRunID: "run", InvocationID: "image-invocation", Operation: aicapability.OperationProductImageAnalysis, System: "fixed readonly observations", Prompt: "inspect only this frozen image", Profile: p, Image: &aicapability.SingleImageInput{SelectionDigest: strings.Repeat("a", 64), ContentSHA256: hex.EncodeToString(h[:]), MediaType: "image/png", ByteLength: len(content), Width: 2, Height: 2}, ImageContent: content}
}

func singleImageExecutor(in aicapability.TextInputIdentity, l *recordingLedger, transport http.RoundTripper) *Executor {
	return &Executor{Ledger: l, Admission: NewBoundedAdmission(), BaseTransport: transport,
		Resolve: func(context.Context, aicapability.TextInputIdentity) (QualifiedRoute, error) {
			return QualifiedRoute{Profile: in.Profile, Endpoint: "https://api.openai.com/v1", APIKey: "synthetic-only"}, nil
		}, Authorize: func(context.Context, aicapability.TextInputIdentity) error { return nil }}
}

func singleImageResponse() *http.Response {
	return &http.Response{StatusCode: 200, Header: http.Header{"Content-Type": []string{"application/json"}}, Body: io.NopCloser(strings.NewReader(fmt.Sprintf(`{"id":"synthetic-image","object":"chat.completion","created":1,"model":%q,"choices":[{"index":0,"message":{"role":"assistant","content":"readonly observation"},"finish_reason":"stop"}],"usage":{"prompt_tokens":30,"completion_tokens":5,"total_tokens":35,"prompt_tokens_details":{"cached_tokens":10}}}`, aicapability.SingleImageMiniModel)))}
}

func TestSingleImageExecutorUsesOriginalLedgerAndActualTypedSDKWire(t *testing.T) {
	in := singleImageExecutionInput(t)
	q, err := aicapability.QuoteText(in)
	require.NoError(t, err)
	ledger := &recordingLedger{}
	var body []byte
	sends := 0
	e := singleImageExecutor(in, ledger, imageFixtureTransport(func(r *http.Request) (*http.Response, error) {
		sends++
		body, _ = io.ReadAll(r.Body)
		return singleImageResponse(), nil
	}))
	out, err := e.Generate(context.Background(), in, q, nil)
	require.NoError(t, err)
	require.Equal(t, "readonly observation", out.Content)
	require.Equal(t, 1, sends)
	var wire map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(body, &wire))
	require.NotContains(t, wire, "max_tokens")
	require.JSONEq(t, `"none"`, string(wire["reasoning_effort"]))
	require.JSONEq(t, `"default"`, string(wire["service_tier"]))
	require.JSONEq(t, `false`, string(wire["store"]))
	require.JSONEq(t, `1`, string(wire["n"]))
	require.JSONEq(t, fmt.Sprint(in.Profile.MaximumCompletionTokens), string(wire["max_completion_tokens"]))
	var messages []struct {
		Role    string
		Content json.RawMessage
	}
	require.NoError(t, json.Unmarshal(wire["messages"], &messages))
	require.Len(t, messages, 2)
	require.Equal(t, "system", messages[0].Role)
	require.JSONEq(t, fmt.Sprintf("%q", in.System), string(messages[0].Content))
	var parts []struct {
		Type     string
		Text     string
		ImageURL *struct{ URL, Detail string } `json:"image_url"`
	}
	require.NoError(t, json.Unmarshal(messages[1].Content, &parts))
	require.Len(t, parts, 2)
	require.Equal(t, in.Prompt, parts[0].Text)
	require.Equal(t, "image_url", parts[1].Type)
	require.Equal(t, "high", parts[1].ImageURL.Detail)
	require.Equal(t, "data:image/png;base64,"+base64.StdEncoding.EncodeToString(in.ImageContent), parts[1].ImageURL.URL)
	require.Len(t, ledger.records, 2)
	require.Equal(t, aicapability.OperationProductImageAnalysis, ledger.records[0].Operation)
	require.Equal(t, q.InputHash, ledger.records[0].InputHash)
	require.EqualValues(t, q.MaximumTokens-in.Profile.MaximumCompletionTokens, ledger.records[0].MaximumPromptTokens)
	require.Equal(t, aicapability.InvocationSucceeded, ledger.records[1].Outcome)
	require.NotNil(t, ledger.records[1].BillableUsage)
	require.EqualValues(t, 30, ledger.records[1].BillableUsage.InputText, "prompt already includes the image, billed once")
	require.EqualValues(t, 10, ledger.records[1].BillableUsage.CachedInputText)
	_, err = e.Generate(context.Background(), in, q, nil)
	require.ErrorIs(t, err, ErrOutcomeUnknown)
	require.Equal(t, 1, sends, "the original ledger rejects another provider attempt")
}

func TestSingleImageExecutorRejectsMissingOrChangedBytesBeforeClaim(t *testing.T) {
	for _, mutate := range []func(*aicapability.TextInputIdentity){
		func(i *aicapability.TextInputIdentity) { i.ImageContent = nil },
		func(i *aicapability.TextInputIdentity) { i.ImageContent[0] ^= 1 },
		func(i *aicapability.TextInputIdentity) { i.Image.Width = 3 },
		func(i *aicapability.TextInputIdentity) { i.Image.MediaType = "image/jpeg" },
	} {
		in := singleImageExecutionInput(t)
		mutate(&in)
		q, err := aicapability.QuoteText(in)
		require.NoError(t, err)
		ledger := &recordingLedger{}
		sends := 0
		e := singleImageExecutor(in, ledger, imageFixtureTransport(func(*http.Request) (*http.Response, error) { sends++; return singleImageResponse(), nil }))
		_, err = e.Generate(context.Background(), in, q, nil)
		require.ErrorIs(t, err, ErrNotDispatched)
		require.False(t, ledger.claimed)
		require.Zero(t, sends)
	}
}

func TestSingleImageFinalWireRejectsDriftBeforeAnyHandoff(t *testing.T) {
	in := singleImageExecutionInput(t)
	url := "data:image/png;base64," + base64.StdEncoding.EncodeToString(in.ImageContent)
	base := fmt.Sprintf(`{"model":%q,"reasoning_effort":"none","service_tier":"default","n":1,"store":false,"max_completion_tokens":64,"messages":[{"role":"system","content":%q},{"role":"user","content":[{"type":"text","text":%q},{"type":"image_url","image_url":{"url":%q,"detail":"high"}}]}]}`, in.Profile.ModelID, in.System, in.Prompt, url)
	for name, mutate := range map[string]func(string) string{
		"different-model": func(s string) string { return strings.Replace(s, in.Profile.ModelID, "other-model", 1) },
		"auto-detail":     func(s string) string { return strings.Replace(s, `"detail":"high"`, `"detail":"auto"`, 1) },
		"arbitrary-url":   func(s string) string { return strings.Replace(s, url, "https://example.invalid/image.png", 1) },
		"different-content": func(s string) string {
			return strings.Replace(s, base64.StdEncoding.EncodeToString(in.ImageContent), base64.StdEncoding.EncodeToString(make([]byte, len(in.ImageContent))), 1)
		},
		"different-prompt": func(s string) string { return strings.Replace(s, in.Prompt, "changed intent", 1) },
		"old-output-limit": func(s string) string { return strings.Replace(s, "max_completion_tokens", "max_tokens", 1) },
		"higher-output-limit": func(s string) string {
			return strings.Replace(s, `"max_completion_tokens":64`, `"max_completion_tokens":65`, 1)
		},
		"extra-tools": func(s string) string { return strings.Replace(s, `"n":1`, `"n":1,"tools":[]`, 1) },
		"duplicate-detail": func(s string) string {
			return strings.Replace(s, `"detail":"high"`, `"detail":"high","detail":"high"`, 1)
		},
		"extra-part-field": func(s string) string { return strings.Replace(s, `"detail":"high"`, `"detail":"high","extra":true`, 1) },
		"storage-enabled":  func(s string) string { return strings.Replace(s, `"store":false`, `"store":true`, 1) },
	} {
		t.Run(name, func(t *testing.T) {
			sends, gates := 0, 0
			cfg := GuardConfig{Adapter: AdapterOpenAICompatible, ProviderID: "openai", AdapterPolicyVersion: aicapability.SingleImageMiniAdapterPolicy, ModelID: in.Profile.ModelID, Endpoint: "https://api.openai.com/v1", MaximumOutputTokens: 64, MaximumRequestBytes: 8192, MaximumResponseBytes: 16384, Timeout: in.Profile.DeadlineBound, SingleImage: &SingleImageRequest{Image: *in.Image, System: in.System, Prompt: in.Prompt}}
			g, err := NewGuardedClient(cfg, imageFixtureTransport(func(*http.Request) (*http.Response, error) { sends++; return singleImageResponse(), nil }), func(context.Context) error { gates++; return nil })
			require.NoError(t, err)
			req, err := http.NewRequest(http.MethodPost, "https://api.openai.com/v1/chat/completions", strings.NewReader(mutate(base)))
			require.NoError(t, err)
			_, err = g.Client.Do(req)
			require.ErrorIs(t, err, ErrNotDispatched)
			require.Zero(t, sends)
			require.Zero(t, gates)
		})
	}
}

func TestSingleImageMissingCacheUsageStaysUnknownWithoutRedispatch(t *testing.T) {
	in := singleImageExecutionInput(t)
	q, err := aicapability.QuoteText(in)
	require.NoError(t, err)
	ledger := &recordingLedger{}
	sends := 0
	e := singleImageExecutor(in, ledger, imageFixtureTransport(func(*http.Request) (*http.Response, error) {
		sends++
		r := singleImageResponse()
		raw, _ := io.ReadAll(r.Body)
		r.Body = io.NopCloser(strings.NewReader(strings.Replace(string(raw), `,"prompt_tokens_details":{"cached_tokens":10}`, "", 1)))
		return r, nil
	}))
	_, err = e.Generate(context.Background(), in, q, nil)
	require.ErrorIs(t, err, ErrOutcomeUnknown)
	require.True(t, ledger.reserved)
	require.Len(t, ledger.records, 1, "no terminal or zero-usage proof is invented")
	_, err = e.Generate(context.Background(), in, q, nil)
	require.ErrorIs(t, err, ErrOutcomeUnknown)
	require.Equal(t, 1, sends)
}

func TestSingleImageUnexpectedResponseModelKeepsOriginalUnknownReservation(t *testing.T) {
	in := singleImageExecutionInput(t)
	q, err := aicapability.QuoteText(in)
	require.NoError(t, err)
	ledger := &recordingLedger{}
	e := singleImageExecutor(in, ledger, imageFixtureTransport(func(*http.Request) (*http.Response, error) {
		r := singleImageResponse()
		raw, _ := io.ReadAll(r.Body)
		r.Body = io.NopCloser(strings.NewReader(strings.Replace(string(raw), in.Profile.ModelID, "unexpected-model", 1)))
		return r, nil
	}))
	_, err = e.Generate(context.Background(), in, q, nil)
	require.ErrorIs(t, err, ErrOutcomeUnknown)
	require.True(t, ledger.reserved)
	require.Len(t, ledger.records, 1)
}

func TestSingleImageUnmappedBillingDimensionRemainsUnknown(t *testing.T) {
	in := singleImageExecutionInput(t)
	q, err := aicapability.QuoteText(in)
	require.NoError(t, err)
	ledger := &recordingLedger{}
	e := singleImageExecutor(in, ledger, imageFixtureTransport(func(*http.Request) (*http.Response, error) {
		r := singleImageResponse()
		raw, _ := io.ReadAll(r.Body)
		r.Body = io.NopCloser(strings.NewReader(strings.Replace(string(raw), `"total_tokens":35`, `"total_tokens":35,"future_billable_units":1`, 1)))
		return r, nil
	}))
	_, err = e.Generate(context.Background(), in, q, nil)
	require.ErrorIs(t, err, ErrOutcomeUnknown)
	require.True(t, ledger.reserved)
	require.Len(t, ledger.records, 1, "unknown pricing dimensions cannot receive a guessed terminal amount")
}

func TestSingleImageRejectsAnimationAndTruncatedPixelsBeforeClaim(t *testing.T) {
	for _, animated := range []bool{false, true} {
		in := singleImageExecutionInput(t)
		if animated {
			chunk := make([]byte, 20)
			binary.BigEndian.PutUint32(chunk[:4], 8)
			copy(chunk[4:8], "acTL")
			binary.BigEndian.PutUint32(chunk[8:12], 2)
			binary.BigEndian.PutUint32(chunk[16:20], crc32.ChecksumIEEE(chunk[4:16]))
			in.ImageContent = append(append(append([]byte(nil), in.ImageContent[:33]...), chunk...), in.ImageContent[33:]...)
			_, _, err := image.Decode(bytes.NewReader(in.ImageContent))
			require.NoError(t, err, "static PNG decode alone would accept this APNG declaration")
		} else {
			in.ImageContent = in.ImageContent[:len(in.ImageContent)-15]
		}
		h := sha256.Sum256(in.ImageContent)
		in.Image.ByteLength, in.Image.ContentSHA256 = len(in.ImageContent), hex.EncodeToString(h[:])
		q, err := aicapability.QuoteText(in)
		require.NoError(t, err)
		ledger := &recordingLedger{}
		e := singleImageExecutor(in, ledger, imageFixtureTransport(func(*http.Request) (*http.Response, error) {
			t.Fatal("invalid image reached transport")
			return nil, nil
		}))
		_, err = e.Generate(context.Background(), in, q, nil)
		require.ErrorIs(t, err, ErrNotDispatched)
		require.False(t, ledger.claimed)
	}
}
