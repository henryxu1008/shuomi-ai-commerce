package skillsandbox

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	osb "github.com/alibaba/OpenSandbox/sdks/sandbox/go"
)

func TestOfficialSDKUsesCombinedEventAndBodyGuards(t *testing.T) {
	init := `{"type":"init","text":"command-1"}` + "\n\n"
	done := `{"type":"execution_complete","exit_code":0}` + "\n\n"
	for _, tc := range []struct{ name, body, want string }{
		{"explicit-zero", init + done, "succeeded"},
		{"eof", init, "outcome_unknown"},
		{"missing-code", init + `{"type":"execution_complete"}` + "\n\n", "outcome_unknown"},
		{"conflicting", init + done + `{"type":"error","error":{"exit_code":7}}` + "\n\n", "outcome_unknown"},
		{"nonzero", init + `{"type":"error","error":{"exit_code":7}}` + "\n\n", "failed"},
		{"events", init + strings.Repeat("{\"type\":\"unknown\"}\n\n", 256) + done, "outcome_unknown"},
		{"line", init + `{"type":"stdout","text":"` + strings.Repeat("x", 256<<10) + `"}` + "\n\n", "outcome_unknown"},
		{"diagnostics", init + `{"type":"stdout","text":"` + strings.Repeat("x", 8193) + `"}` + "\n\n" + done, "outcome_unknown"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			posts := 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				posts++
				w.Header().Set("Content-Type", "text/event-stream")
				w.(http.Flusher).Flush()
				_, _ = io.WriteString(w, tc.body)
			}))
			defer server.Close()
			client := &http.Client{Transport: boundedTransport{base: http.DefaultTransport, origin: server.URL}}
			sdk := osb.NewExecdClient(server.URL, "", osb.WithHTTPClient(client))
			var o observation
			err := sdk.RunCommand(context.Background(), osb.RunCommandRequest{Argv: []string{"python3", "test.py"}, Timeout: 1000}, o.observe)
			if got := o.outcome(err); got != tc.want {
				t.Fatalf("outcome %s want %s error=%v", got, tc.want, err)
			}
			if posts != 1 {
				t.Fatal("command was retried")
			}
			if tc.name == "events" && !errors.Is(err, errBudget) {
				t.Fatal("full event stream not counted", err)
			}
		})
	}
}

func TestBodyBudgetExactBoundaryAndFileBudget(t *testing.T) {
	for _, n := range []int{8192, 8193} {
		t.Run(string(rune(n)), func(t *testing.T) {
			b := &limitedBody{ReadCloser: io.NopCloser(strings.NewReader(strings.Repeat("x", n))), left: 8192}
			raw, err := io.ReadAll(b)
			if n == 8192 && (err != nil || len(raw) != 8192) {
				t.Fatal("exact budget rejected", err)
			}
			if n == 8193 && !errors.Is(err, errBudget) {
				t.Fatal("oversize became EOF", err)
			}
		})
	}
}
