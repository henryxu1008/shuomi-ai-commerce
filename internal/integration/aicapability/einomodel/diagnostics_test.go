package einomodel

import (
	"bytes"
	"context"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptrace"
	"os"
	"strings"
	"sync"
	"syscall"
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

func diagnosticLogEntries(t *testing.T, output *bytes.Buffer) []map[string]any {
	t.Helper()
	decoder := json.NewDecoder(bytes.NewReader(output.Bytes()))
	var entries []map[string]any
	for {
		var entry map[string]any
		err := decoder.Decode(&entry)
		if errors.Is(err, io.EOF) {
			return entries
		}
		require.NoError(t, err)
		entries = append(entries, entry)
	}
}

func TestComponentTransportDiagnosticsClassifyStandardErrors(t *testing.T) {
	for _, tc := range []struct {
		category string
		err      error
	}{
		{"eof", io.EOF},
		{"unexpected_eof", io.ErrUnexpectedEOF},
		{"connection_reset", syscall.ECONNRESET},
		{"connection_refused", syscall.ECONNREFUSED},
		{"broken_pipe", syscall.EPIPE},
		{"connection_closed", net.ErrClosed},
		{"connection_closed", io.ErrClosedPipe},
	} {
		for _, wrapped := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/wrapped=%t/%T", tc.category, wrapped, tc.err), func(t *testing.T) {
				ctx, logger, output := diagnosticContext()
				transportErr := tc.err
				if wrapped {
					transportErr = fmt.Errorf("https://synthetic-secret-url/?key=synthetic-secret-key: %w", &net.OpError{
						Op: "synthetic-secret-operation", Net: "synthetic-secret-network", Err: transportErr,
					})
				}
				sends := 0
				result, err := runDiagnosticComponent(t, ctx, logger, diagnosticTransport(func(*http.Request) (*http.Response, error) {
					sends++
					return nil, transportErr
				}), "synthetic-secret-key")
				require.ErrorIs(t, err, ErrOutcomeUnknown)
				require.False(t, result.Usage.Known)
				require.Equal(t, 1, sends)
				found := false
				for _, entry := range diagnosticLogEntries(t, output) {
					if entry["stage"] == "transport" {
						found = true
						require.Equal(t, tc.category, entry["category"])
						require.EqualValues(t, 1, entry["handoffs"])
					}
				}
				require.True(t, found)
				require.NotContains(t, output.String(), "synthetic-secret")
				require.NotContains(t, output.String(), "https://")
			})
		}
	}
}

func TestDiagnosticErrorCategoriesPreservePriorityAndRejectFreeText(t *testing.T) {
	for _, tc := range []struct {
		name, category string
		err            error
	}{
		{"deadline", "deadline_exceeded", errors.Join(io.EOF, context.DeadlineExceeded)},
		{"canceled", "context_canceled", errors.Join(syscall.ECONNRESET, context.Canceled)},
		{"DNS timeout", "dns_error", &net.DNSError{Err: "synthetic-secret-error", Name: "synthetic-secret-host", IsTimeout: true}},
		{"certificate", "tls_certificate_error", errors.Join(io.EOF, x509.UnknownAuthorityError{})},
		{"timeout", "network_timeout", fmt.Errorf("synthetic-secret: %w", os.ErrDeadlineExceeded)},
		{"free text", "transport_error", errors.New("EOF: connection reset, refused, broken pipe, closed synthetic-secret")},
	} {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.category, diagnosticErrorCategory(tc.err))
		})
	}
}

func TestComponentTraceDiagnosticsKeepCallbackResultsWithoutClaimingSuccess(t *testing.T) {
	secretEOF := fmt.Errorf("synthetic-secret-key https://synthetic-secret-host: %w", io.EOF)
	for _, tc := range []struct {
		name                             string
		tlsErrors, writeErrors           []error
		tlsSuccess, tlsFailure           int
		writeSuccess, writeFailure       int
		tlsLastFailure, writeLastFailure string
	}{
		{name: "not observed"},
		{name: "callbacks succeeded but outcome unknown", tlsErrors: []error{nil}, writeErrors: []error{nil}, tlsSuccess: 1, writeSuccess: 1},
		{name: "failed callbacks", tlsErrors: []error{secretEOF}, writeErrors: []error{syscall.EPIPE}, tlsFailure: 1, writeFailure: 1, tlsLastFailure: "eof", writeLastFailure: "broken_pipe"},
		{name: "later success preserves failures", tlsErrors: []error{secretEOF, syscall.ECONNRESET, nil}, writeErrors: []error{syscall.EPIPE, net.ErrClosed, nil}, tlsSuccess: 1, tlsFailure: 2, writeSuccess: 1, writeFailure: 2, tlsLastFailure: "connection_reset", writeLastFailure: "connection_closed"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx, logger, output := diagnosticContext()
			sends := 0
			result, err := runDiagnosticComponent(t, ctx, logger, diagnosticTransport(func(r *http.Request) (*http.Response, error) {
				sends++
				trace := httptrace.ContextClientTrace(r.Context())
				require.NotNil(t, trace)
				for _, callbackErr := range tc.tlsErrors {
					trace.TLSHandshakeDone(tls.ConnectionState{}, callbackErr)
				}
				for _, callbackErr := range tc.writeErrors {
					trace.WroteRequest(httptrace.WroteRequestInfo{Err: callbackErr})
				}
				return nil, io.EOF
			}), "synthetic-secret-key")
			require.ErrorIs(t, err, ErrOutcomeUnknown)
			require.False(t, result.Usage.Known)
			require.Equal(t, 1, sends)
			for _, entry := range diagnosticLogEntries(t, output) {
				require.EqualValues(t, tc.tlsSuccess, entry["tls_handshake_success_count"])
				require.EqualValues(t, tc.tlsFailure, entry["tls_handshake_failure_count"])
				require.EqualValues(t, tc.writeSuccess, entry["wrote_request_success_count"])
				require.EqualValues(t, tc.writeFailure, entry["wrote_request_failure_count"])
				for key, expected := range map[string]string{
					"tls_handshake_last_failure_category": tc.tlsLastFailure,
					"wrote_request_last_failure_category": tc.writeLastFailure,
				} {
					if expected == "" {
						require.NotContains(t, entry, key)
					} else {
						require.Equal(t, expected, entry[key])
					}
				}
			}
			require.NotContains(t, output.String(), "synthetic-secret")
			require.NotContains(t, output.String(), "https://")
		})
	}
}

func TestTraceDiagnosticCallbackSnapshotsRemainConsistentUnderConcurrency(t *testing.T) {
	ctx, logger, output := diagnosticContext()
	diagnostic := newTextDiagnostics(ctx, "synthetic-secret-invocation", logger)
	trace := diagnostic.trace()
	const callbacks = 32
	var workers sync.WaitGroup
	for range callbacks {
		workers.Go(func() {
			trace.TLSHandshakeDone(tls.ConnectionState{}, io.EOF)
			trace.TLSHandshakeDone(tls.ConnectionState{}, nil)
			trace.WroteRequest(httptrace.WroteRequestInfo{Err: syscall.EPIPE})
			trace.WroteRequest(httptrace.WroteRequestInfo{})
			diagnostic.emit(ctx, "offline_snapshot", "transport_error")
		})
	}
	workers.Wait()
	diagnostic.emit(ctx, "offline_final", "transport_error")
	entries := diagnosticLogEntries(t, output)
	require.Len(t, entries, callbacks+1)
	for _, entry := range entries {
		for prefix, category := range map[string]string{"tls_handshake": "eof", "wrote_request": "broken_pipe"} {
			failure, ok := entry[prefix+"_failure_count"].(float64)
			require.True(t, ok)
			if failure == 0 {
				require.NotContains(t, entry, prefix+"_last_failure_category")
			} else {
				require.Equal(t, category, entry[prefix+"_last_failure_category"])
			}
		}
	}
	final := entries[len(entries)-1]
	for _, prefix := range []string{"tls_handshake", "wrote_request"} {
		require.EqualValues(t, callbacks, final[prefix+"_success_count"])
		require.EqualValues(t, callbacks, final[prefix+"_failure_count"])
	}
	require.EqualValues(t, 0, final["handoffs"], "callback counts do not create transport handoffs")
	require.NotContains(t, output.String(), "synthetic-secret")
}
