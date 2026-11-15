package skillsandbox

import (
	"context"
	"encoding/json"
	osb "github.com/alibaba/OpenSandbox/sdks/sandbox/go"
	"net/http"
	"net/http/httptest"
	"strings"
	"task-processor/internal/agent"
	"task-processor/internal/skill"
	"testing"
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
			_ = json.NewEncoder(w).Encode(map[string]any{"id": id, "image": map[string]string{"uri": descriptor.ImageDigest}, "metadata": map[string]string{"skill_profile": descriptor.Digest()}})
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
	defer server.Close()
	client := &http.Client{Transport: boundedTransport{base: http.DefaultTransport, origin: server.URL}}
	executor := &Executor{descriptor: descriptor, lifecycle: osb.NewLifecycleClient(server.URL+"/v1", "", osb.WithHTTPClient(client)), client: client, origin: server.URL, gate: make(chan struct{}, 1)}
	result, err := executor.Execute(context.Background(), task)
	expectedCommands := 2
	if unprotected {
		expectedCommands = 0
	}
	if err != nil || result.Receipt.Outcome != outcome || result.Receipt.Cleanup != "completed" || created != 1 || commands != expectedCommands || denied != 1 || (outcome == "succeeded" && result.Title != "after") || (outcome != "succeeded" && result.Title != "") {
		t.Fatalf("lifecycle outcome=%+v created=%d commands=%d denied=%d err=%v", result, created, commands, denied, err)
	}
}
