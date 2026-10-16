package einomodel

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
	"task-processor/internal/aicapability"
)

type diagnosticTransport func(*http.Request) (*http.Response, error)

func (f diagnosticTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

type diagnosticBrokenBody struct{}

func (diagnosticBrokenBody) Read([]byte) (int, error) {
	return 0, errors.New("synthetic-secret-body")
}
func (diagnosticBrokenBody) Close() error { return nil }

const diagnosticResponse = `{"id":"chatcmpl-synthetic-secret-completion","object":"chat.completion","created":1,"model":"synthetic-model","choices":[{"index":0,"message":{"role":"assistant","content":"READY"},"finish_reason":"stop"}],"usage":{"prompt_tokens":10,"completion_tokens":5,"total_tokens":15}}`

func diagnosticContext() (context.Context, *logrus.Entry, *bytes.Buffer) {
	var output bytes.Buffer
	logger := logrus.New()
	logger.SetOutput(&output)
	logger.SetFormatter(&logrus.JSONFormatter{DisableTimestamp: true})
	return context.Background(), logrus.NewEntry(logger), &output
}

func runDiagnosticComponent(t *testing.T, ctx context.Context, logger *logrus.Entry, transport http.RoundTripper, key string) (TextOutput, error) {
	t.Helper()
	guard, err := NewGuardedClient(GuardConfig{Logger: logger, Adapter: AdapterOpenAICompatible, Endpoint: "http://localhost/v1", ModelID: "synthetic-model",
		MaximumOutputTokens: 32, MaximumRequestBytes: 65536, MaximumResponseBytes: 16384, Timeout: time.Second}, transport, func(context.Context) error { return nil })
	require.NoError(t, err)
	component, err := NewComponent(ctx, ComponentConfig{Adapter: AdapterOpenAICompatible, Endpoint: "http://localhost/v1", ModelID: "synthetic-model", APIKey: key, MaximumOutputTokens: 32}, guard)
	require.NoError(t, err)
	return GenerateText(ctx, component, "synthetic-secret-system", "synthetic-secret-prompt", guard)
}

func TestComponentFailureDiagnosticsDoNotExposeRawErrors(t *testing.T) {
	for _, tc := range []struct {
		name, stage, category string
		status                int
		body                  io.ReadCloser
		transportErr          error
	}{
		{name: "transport deadline", stage: "transport", category: "deadline_exceeded", transportErr: errors.Join(context.DeadlineExceeded, errors.New("https://synthetic-secret-url/?api_key=synthetic-secret-key"))},
		{name: "body failure", stage: "response_body", category: "read_failed", status: 200, body: diagnosticBrokenBody{}},
		{name: "HTTP rejection", stage: "sdk", category: "http_error", status: 401, body: io.NopCloser(strings.NewReader(`{"error":{"message":"synthetic-secret-provider-body","type":"invalid_request_error","code":"synthetic-secret-code"}}`))},
		{name: "invalid JSON", stage: "sdk", category: "response_json_invalid", status: 200, body: io.NopCloser(strings.NewReader(`synthetic-secret-invalid-json`))},
		{name: "invalid JSON field type", stage: "sdk", category: "response_json_invalid", status: 200, body: io.NopCloser(strings.NewReader(strings.Replace(diagnosticResponse, `"prompt_tokens":10`, `"prompt_tokens":"synthetic-secret-token"`, 1)))},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx, logger, output := diagnosticContext()
			sends := 0
			_, err := runDiagnosticComponent(t, ctx, logger, diagnosticTransport(func(r *http.Request) (*http.Response, error) {
				sends++
				require.Equal(t, "localhost", r.URL.Hostname(), "all protocol fixtures stay offline")
				if tc.transportErr != nil {
					return nil, tc.transportErr
				}
				return &http.Response{StatusCode: tc.status, Header: http.Header{"Content-Type": {"application/json"}}, Body: tc.body, Request: r}, nil
			}), "synthetic-secret-key")
			require.ErrorIs(t, err, ErrOutcomeUnknown)
			require.Equal(t, 1, sends)
			require.Contains(t, output.String(), `"stage":"`+tc.stage+`"`)
			require.Contains(t, output.String(), `"category":"`+tc.category+`"`)
			require.NotContains(t, output.String(), "synthetic-secret")
			require.NotContains(t, output.String(), "http://")
			require.NotContains(t, output.String(), "https://")
			require.NotContains(t, output.String(), `"caller_context_state":"canceled"`, "SDK cleanup must not be mistaken for caller cancellation")
		})
	}
}

func TestComponentDiagnosticIDsKeepTheirActualSources(t *testing.T) {
	for _, tc := range []struct{ name, requestID, key string }{
		{name: "HTTP UUID", requestID: "4adc046b-d907-4427-a090-b075a507c83d", key: "synthetic-secret-key"},
		{name: "secret disguised as ID", requestID: "sk-synthetic-secret-key", key: "sk-synthetic-secret-key"},
		{name: "UUID credential disguised as ID", requestID: "4adc046b-d907-4427-a090-b075a507c83d", key: "4adc046b-d907-4427-a090-b075a507c83d"},
		{name: "overlong ID", requestID: strings.Repeat("x", 129), key: "synthetic-secret-key"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx, logger, output := diagnosticContext()
			result, err := runDiagnosticComponent(t, ctx, logger, diagnosticTransport(func(r *http.Request) (*http.Response, error) {
				return &http.Response{StatusCode: 200, Header: http.Header{"Content-Type": {"application/json"}, "X-Request-Id": {tc.requestID}}, Body: io.NopCloser(strings.NewReader(diagnosticResponse)), Request: r}, nil
			}), tc.key)
			require.NoError(t, err)
			require.True(t, result.Usage.Known)
			require.Contains(t, output.String(), `"completion_id_hash":`)
			require.NotContains(t, output.String(), "chatcmpl-synthetic-secret-completion")
			if len(tc.requestID) <= 128 {
				sum := sha256.Sum256([]byte(tc.requestID))
				require.Contains(t, output.String(), `"http_request_id_hash":"`+hex.EncodeToString(sum[:])+`"`)
			} else {
				require.NotContains(t, output.String(), `"http_request_id_hash"`)
			}
			require.NotContains(t, output.String(), tc.requestID, "provider IDs may echo private input, even when UUID-shaped")
			require.NotContains(t, output.String(), tc.key)
			require.NotContains(t, output.String(), "synthetic-secret")
		})
	}
}

type diagnosticLedger struct {
	recordingLedger
	terminalErr error
}

func (l *diagnosticLedger) RecordInvocation(ctx context.Context, record aicapability.InvocationRecord) error {
	if l.terminalErr != nil {
		return l.terminalErr
	}
	return l.recordingLedger.RecordInvocation(ctx, record)
}

func TestExecutorDiagnosticsDistinguishTerminalFailure(t *testing.T) {
	for _, fails := range []bool{false, true} {
		t.Run(map[bool]string{false: "recorded", true: "record_failed"}[fails], func(t *testing.T) {
			ctx, logger, logOutput := diagnosticContext()
			ledger := &diagnosticLedger{}
			if fails {
				ledger.terminalErr = errors.New("synthetic-secret-sql-value")
			}
			profile := testTextProfile("http://localhost/v1")
			input := aicapability.TextInputIdentity{OrganizationID: "synthetic-secret-org", ActorID: "synthetic-secret-actor", MemberID: "member-1", Operation: aicapability.OperationAIWorkbenchChatPlan,
				InvocationID: "synthetic-secret-invocation", System: "system", Prompt: "prompt", Profile: profile}
			quote, err := aicapability.QuoteText(input)
			require.NoError(t, err)
			sends := 0
			executor := &Executor{Ledger: ledger, Logger: logger, Admission: NewBoundedAdmission(),
				Resolve: func(context.Context, aicapability.TextInputIdentity) (QualifiedRoute, error) {
					return QualifiedRoute{Profile: profile, Endpoint: "http://localhost/v1", APIKey: "synthetic-secret-key"}, nil
				},
				Authorize: func(context.Context, aicapability.TextInputIdentity) error { return nil },
				BaseTransport: diagnosticTransport(func(r *http.Request) (*http.Response, error) {
					sends++
					return &http.Response{StatusCode: 200, Header: http.Header{}, Body: io.NopCloser(strings.NewReader(diagnosticResponse)), Request: r}, nil
				})}
			result, err := executor.Generate(ctx, input, quote, nil)
			if fails {
				require.ErrorIs(t, err, ErrOutcomeUnknown)
				require.False(t, result.Usage.Known)
				require.Len(t, ledger.records, 1, "the durable dispatch is not overwritten")
			} else {
				require.NoError(t, err)
				require.True(t, result.Usage.Known)
				require.Len(t, ledger.records, 2)
			}
			require.Equal(t, 1, sends)
			require.Contains(t, logOutput.String(), `"stage":"terminal"`)
			require.Contains(t, logOutput.String(), `"category":"`+map[bool]string{false: "recorded", true: "record_failed"}[fails]+`"`)
			require.NotContains(t, logOutput.String(), "synthetic-secret")
		})
	}
}

func TestExecutorPanicDiagnosticsPreserveOriginalPanicAndDispatch(t *testing.T) {
	ctx, logger, output := diagnosticContext()
	profile := testTextProfile("http://localhost/v1")
	input := aicapability.TextInputIdentity{OrganizationID: "org-1", ActorID: "actor-1", MemberID: "member-1", Operation: aicapability.OperationAIWorkbenchChatPlan,
		InvocationID: "invocation-1", System: "system", Prompt: "prompt", Profile: profile}
	quote, err := aicapability.QuoteText(input)
	require.NoError(t, err)
	ledger := &recordingLedger{}
	sends := 0
	executor := &Executor{Ledger: ledger, Logger: logger, Admission: NewBoundedAdmission(),
		Resolve: func(context.Context, aicapability.TextInputIdentity) (QualifiedRoute, error) {
			return QualifiedRoute{Profile: profile, Endpoint: "http://localhost/v1", APIKey: "synthetic-secret-key"}, nil
		},
		Authorize:     func(context.Context, aicapability.TextInputIdentity) error { return nil },
		BaseTransport: diagnosticTransport(func(*http.Request) (*http.Response, error) { sends++; panic("synthetic-secret-panic") })}
	require.PanicsWithValue(t, "synthetic-secret-panic", func() { _, _ = executor.Generate(ctx, input, quote, nil) })
	require.Equal(t, 1, sends)
	require.Len(t, ledger.records, 1)
	require.Equal(t, aicapability.InvocationDispatched, ledger.records[0].Outcome)
	require.Contains(t, output.String(), `"category":"panic"`)
	require.NotContains(t, output.String(), "synthetic-secret")
}
