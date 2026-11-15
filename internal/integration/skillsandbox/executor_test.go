package skillsandbox

import (
	"context"
	"encoding/base32"
	"encoding/hex"
	"encoding/json"
	osb "github.com/alibaba/OpenSandbox/sdks/sandbox/go"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"task-processor/internal/agent"
	"task-processor/internal/skill"
	"testing"
	"time"
)

func TestOfficialSDKLifecycleKeepsExecdCredentialOutOfWorkload(t *testing.T) {
	for _, tc := range []struct {
		name, output, outcome string
		unprotected           bool
	}{
		{name: "valid", output: `{"schemaVersion":"title-result-v1","title":"after","diagnostics":[]}`, outcome: "succeeded"},
		{name: "missing diagnostics", output: `{"schemaVersion":"title-result-v1","title":"after"}`, outcome: "failed"},
		{name: "null diagnostics", output: `{"schemaVersion":"title-result-v1","title":"after","diagnostics":null}`, outcome: "failed"},
		{name: "invalid UTF8", output: "{\"schemaVersion\":\"title-result-v1\",\"title\":\"\xff\",\"diagnostics\":[]}", outcome: "failed"},
		{name: "duplicate key", output: `{"schemaVersion":"title-result-v1","title":"after","title":"other","diagnostics":[]}`, outcome: "failed"},
		{name: "unprotected daemon", outcome: "outcome_unknown", unprotected: true},
	} {
		t.Run(tc.name, func(t *testing.T) { testLifecycle(t, tc.output, tc.outcome, tc.unprotected) })
	}
}

func testLifecycle(t *testing.T, output, outcome string, unprotected bool) {
	t.Helper()
	executor, task, counts := lifecycleFixture(t, output, unprotected, lifecycleOptions{})
	result, err := executor.Execute(context.Background(), task)
	created, commands, denied := counts()
	expectedCommands := 2
	if unprotected {
		expectedCommands = 0
	}
	if err != nil || result.Receipt.Outcome != outcome || result.Receipt.Cleanup != "completed" || created != 1 || commands != expectedCommands || denied != 1 || (outcome == "succeeded" && result.Title != "after") || (outcome != "succeeded" && result.Title != "") {
		t.Fatalf("lifecycle outcome=%+v created=%d commands=%d denied=%d err=%v", result, created, commands, denied, err)
	}
}

type lifecycleOptions struct {
	collectorEvents       string
	nearDeadlineCollector bool
	cleanupDelay          time.Duration
}

type lifecycleTransport func(*http.Request) (*http.Response, error)

func (f lifecycleTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

// The same official-SDK fixture is used directly and through the real Commerce
// registry. No test-only timeout fields are added to the production executor.
func lifecycleFixture(t *testing.T, output string, unprotected bool, options lifecycleOptions) (*Executor, agent.ScriptTask, func() (int, int, int)) {
	t.Helper()
	descriptor := agent.ScriptDescriptor{Profile: "python-title-json-v1", Revision: "1", NetworkRevision: "1", ToolVersion: "v1.0.0", ImageDigest: "sha256:" + strings.Repeat("a", 64), ExecdDigest: "sha256:" + strings.Repeat("b", 64)}
	files := []skill.File{{Path: "SKILL.md", Text: "---\nname: synthetic-title\ndescription: Saved formatting\nmetadata:\n  listingkit-method-v1: product-title\n---\nFormat the candidate.\n"}, {Path: "execution.json", Text: `{"schemaVersion":"skill-script-v1","profile":"python-title-json-v1","entrypoint":"scripts/main.py","phase":"title-candidate-postprocess","inputContract":"title-candidate-v1","outputContract":"title-result-v1"}`}, {Path: "scripts/main.py", Text: "print('synthetic')\n"}}
	pkg, err := skill.ValidatePackage(files)
	if err != nil {
		t.Fatal(err)
	}
	task := agent.ScriptTask{Descriptor: descriptor, Entrypoint: "scripts/main.py", PackageDigest: pkg.Digest, Input: []byte(`{"schemaVersion":"title-candidate-v1","title":"before","targetLanguage":"en","evidenceRefs":[]}`), RunID: "run", CallID: "call"}
	for _, f := range files {
		task.Files = append(task.Files, agent.ScriptFile{Path: f.Path, Text: f.Text})
	}
	const id = "d68f289c-2e45-4d0a-9a13-d53db4425739"
	token := ""
	profile := ""
	created, commands, denied := 0, 0, 0
	deleted := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.URL.Path == "/v1/sandboxes" && r.Method == http.MethodGet:
			_, _ = w.Write([]byte(`{"items":[],"pagination":{"totalItems":0}}`))
		case r.URL.Path == "/v1/sandboxes" && r.Method == http.MethodPost:
			created++
			var request osb.CreateSandboxRequest
			_ = json.NewDecoder(r.Body).Decode(&request)
			profile = request.Metadata["skill_profile"]
			if len(profile) > 63 {
				http.Error(w, `{"message":"metadata value exceeds 63 characters"}`, http.StatusBadRequest)
				return
			}
			decoded, decodeErr := base32.StdEncoding.WithPadding(base32.NoPadding).DecodeString(profile)
			if decodeErr != nil || hex.EncodeToString(decoded) != descriptor.Digest() {
				t.Error("profile label did not preserve the complete descriptor digest")
			}
			token = request.Env["EXECD_ACCESS_TOKEN"]
			if len(token) < 32 {
				t.Error("execd starts without private per-instance credential")
				http.Error(w, "missing", 500)
				return
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"id": id, "image": request.Image, "metadata": request.Metadata})
		case r.URL.Path == "/v1/sandboxes/"+id:
			if r.Method == http.MethodDelete {
				deleted = true
				w.WriteHeader(204)
				return
			}
			if deleted {
				http.Error(w, `{"message":"absent"}`, 404)
				return
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"id": id, "image": map[string]string{"uri": descriptor.ImageDigest}, "metadata": map[string]string{"skill_profile": profile}})
		case strings.HasSuffix(r.URL.Path, "/ping"):
			_, _ = w.Write([]byte(`{}`))
		default:
			if r.Header.Get("X-EXECD-ACCESS-TOKEN") != token || token == "" {
				denied++
				if unprotected {
					_, _ = w.Write([]byte(`{}`))
					return
				}
				http.Error(w, `{"message":"unauthorized"}`, 401)
				return
			}
			if strings.HasSuffix(r.URL.Path, "/command") && r.Method == http.MethodPost {
				commands++
				var command osb.RunCommandRequest
				_ = json.NewDecoder(r.Body).Decode(&command)
				value, present := command.Envs["EXECD_ACCESS_TOKEN"]
				if !present || value != "" {
					t.Error("daemon credential inherited by workload")
				}
				if command.Command != "" || command.Background || command.Timeout <= 0 || command.UID == nil {
					t.Error("unbounded/free shell execution")
				}
				if commands == 1 && *command.UID != 65534 {
					t.Error("script is privileged")
				}
				w.Header().Set("Content-Type", "text/event-stream")
				if commands == 2 && options.collectorEvents != "" {
					_, _ = io.WriteString(w, options.collectorEvents)
					return
				}
				_, _ = w.Write([]byte("{\"type\":\"init\",\"text\":\"command-id\"}\n\n{\"type\":\"execution_complete\",\"exit_code\":0}\n\n"))
				return
			}
			if strings.HasSuffix(r.URL.Path, "/files/download") {
				_, _ = w.Write([]byte(output))
				return
			}
			_, _ = w.Write([]byte(`{}`))
		}
	}))
	t.Cleanup(server.Close)
	transport := lifecycleTransport(func(r *http.Request) (*http.Response, error) {
		var delay time.Duration
		if options.nearDeadlineCollector && r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/command") {
			raw, err := io.ReadAll(r.Body)
			if err != nil {
				return nil, err
			}
			r.Body = io.NopCloser(strings.NewReader(string(raw)))
			var command osb.RunCommandRequest
			_ = json.Unmarshal(raw, &command)
			if len(command.Argv) > 3 && command.Argv[3] == "-c" {
				deadline, _ := r.Context().Deadline()
				delay = time.Until(deadline) - 500*time.Millisecond
			}
		}
		if r.Method == http.MethodDelete && r.URL.Path == "/v1/sandboxes/"+id {
			delay = options.cleanupDelay
		}
		if delay > 0 {
			timer := time.NewTimer(delay)
			defer timer.Stop()
			select {
			case <-timer.C:
			case <-r.Context().Done():
				return nil, r.Context().Err()
			}
		}
		return http.DefaultTransport.RoundTrip(r)
	})
	client := &http.Client{Transport: boundedTransport{base: transport, origin: server.URL}}
	executor := &Executor{descriptor: descriptor, lifecycle: osb.NewLifecycleClient(server.URL+"/v1", "", osb.WithHTTPClient(client)), client: client, origin: server.URL, gate: make(chan struct{}, 1)}
	return executor, task, func() (int, int, int) { return created, commands, denied }
}
