package bailian

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"task-processor/internal/ai"
)

func TestUnknownDiagnosticIsBoundedAndNeverChangesDispatch(t *testing.T) {
	const requestID = "12345678-1234-1234-1234-123456789012"
	for _, tc := range []struct {
		name, body, code, stage string
		status                  int
	}{
		{"quota", `{"code":"AllocationQuota.FreeTierOnly","message":"secret-key private-prompt private-url","request_id":"` + requestID + `"}`, "AllocationQuota.FreeTierOnly", "http_status", 403},
		{"untrusted_fields", `{"code":"secret-key","message":"private-prompt","request_id":"private-url"}`, "unrecognized", "http_status", 400},
		{"invalid_json", `{`, "", "response_json", 200},
		{"contract", strings.Replace(successBody(), `"output_image_count":1`, `"output_image_count":2`, 1), "", "response_contract", 200},
	} {
		t.Run(tc.name, func(t *testing.T) {
			calls, observed := 0, 0
			var diagnostics []Diagnostic
			cfg := fixtureConfig(imageTransport(func(*http.Request) (*http.Response, error) {
				calls++
				return &http.Response{StatusCode: tc.status, Header: http.Header{"X-Request-Id": []string{requestID}}, Body: io.NopCloser(strings.NewReader(tc.body))}, nil
			}))
			cfg.APIKey = "secret-key"
			cfg.Diagnose = func(d Diagnostic) { diagnostics = append(diagnostics, d) }
			client, err := NewClient(cfg, func(context.Context, Observation) error { observed++; return nil })
			require.NoError(t, err)
			_, err = client.EditImageWithRoute(context.Background(), &ai.ImageEditRequest{Model: Model, Prompt: "private-prompt", Image: sourcePNG(t), ImageContentType: "image/png"}, fixtureRoute())
			require.ErrorIs(t, err, errUnknown)
			require.Equal(t, 1, calls)
			require.Zero(t, observed)
			require.Len(t, diagnostics, 1)
			require.Equal(t, tc.stage, diagnostics[0].Stage)
			require.Equal(t, tc.status, diagnostics[0].HTTPStatus)
			require.Equal(t, tc.code, diagnostics[0].ProviderCode)
			require.Equal(t, requestID, diagnostics[0].RequestID)
			encoded, err := json.Marshal(diagnostics)
			require.NoError(t, err)
			for _, private := range []string{"secret-key", "private-prompt", "private-url", "data:image", "Authorization"} {
				require.NotContains(t, string(encoded), private)
			}
		})
	}
}

func TestTransportDiagnosticDoesNotExposeWrappedError(t *testing.T) {
	var diagnostics []Diagnostic
	cfg := fixtureConfig(imageTransport(func(*http.Request) (*http.Response, error) {
		return nil, errors.New("sensitive transport proxy url and credential")
	}))
	cfg.Diagnose = func(d Diagnostic) { diagnostics = append(diagnostics, d) }
	client, err := NewClient(cfg, func(context.Context, Observation) error { t.Fatal("must not observe success"); return nil })
	require.NoError(t, err)
	_, err = client.GenerateImageWithRoute(context.Background(), &ai.ImageGenerateRequest{Model: Model, Prompt: "private-prompt"}, fixtureRoute())
	require.ErrorIs(t, err, errUnknown)
	require.Equal(t, []Diagnostic{{Stage: "transport", Reason: "network_other"}}, diagnostics)
}

func TestErrorBodyDiagnosticsCannotDelayRejectionUntilGenerationTimeout(t *testing.T) {
	const requestID = "12345678-1234-1234-1234-123456789012"
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Request-Id", requestID)
		w.WriteHeader(http.StatusForbidden)
		w.(http.Flusher).Flush()
		<-r.Context().Done() // A failed response may never complete its body.
	}))
	defer server.Close()
	var diagnostics []Diagnostic
	calls := 0
	cfg := fixtureConfig(imageTransport(func(r *http.Request) (*http.Response, error) {
		calls++
		forward, err := http.NewRequestWithContext(r.Context(), r.Method, server.URL, r.Body)
		require.NoError(t, err)
		return server.Client().Transport.RoundTrip(forward)
	}))
	cfg.Timeout = 3 * time.Second
	cfg.Diagnose = func(d Diagnostic) { diagnostics = append(diagnostics, d) }
	client, err := NewClient(cfg, func(context.Context, Observation) error { t.Fatal("must not observe success"); return nil })
	require.NoError(t, err)
	started := time.Now()
	_, err = client.GenerateImageWithRoute(context.Background(), &ai.ImageGenerateRequest{Model: Model, Prompt: "fixture-only"}, fixtureRoute())
	require.ErrorIs(t, err, errUnknown)
	require.Less(t, time.Since(started), 2*time.Second, "diagnostic reads must not wait for the generation timeout")
	require.Equal(t, 1, calls)
	require.Equal(t, []Diagnostic{{Stage: "response_read", HTTPStatus: http.StatusForbidden, RequestID: requestID}}, diagnostics)
}
