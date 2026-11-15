package httpapi

import (
	"encoding/json"
	"encoding/pem"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/require"
	"task-processor/internal/agent"
	"task-processor/internal/integration/skillsandbox"
)

// Real HTTP/PG/Registry/Graph/official SDK composition; only the external
// model and sandbox control servers are bounded protocol fixtures.
func TestScriptHTTPThroughNativeOwners(t *testing.T) {
	for _, mode := range []string{"default", "read_only", "auto_approve", "full_access"} {
		t.Run(mode, func(t *testing.T) { testProductAgentOwners(t, "content flow skill script "+mode) })
	}
}

func configureScriptHTTPFixture(t *testing.T, settings *ProductAgentDependencies) *atomic.Int32 {
	t.Helper()
	var creations atomic.Int32
	const id = "d68f289c-2e45-4d0a-9a13-d53db4425739"
	var token, image string
	var metadata map[string]string
	deleted := false
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.URL.Path == "/v1/sandboxes" && r.Method == http.MethodGet:
			_, _ = w.Write([]byte(`{"items":[],"pagination":{"totalItems":0}}`))
		case r.URL.Path == "/v1/sandboxes" && r.Method == http.MethodPost:
			creations.Add(1)
			deleted = false
			var request struct {
				Env   map[string]string `json:"env"`
				Image struct {
					URI string `json:"uri"`
				} `json:"image"`
				Metadata map[string]string `json:"metadata"`
			}
			require.NoError(t, json.NewDecoder(r.Body).Decode(&request))
			token = request.Env["EXECD_ACCESS_TOKEN"]
			require.Len(t, token, 64)
			image = request.Image.URI
			metadata = request.Metadata
			_ = json.NewEncoder(w).Encode(map[string]any{"id": id, "image": map[string]string{"uri": image}, "metadata": metadata})
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
			_ = json.NewEncoder(w).Encode(map[string]any{"id": id, "image": map[string]string{"uri": image}, "metadata": metadata})
		case strings.HasSuffix(r.URL.Path, "/ping"):
			_, _ = w.Write([]byte(`{}`))
		default:
			if r.Header.Get("X-EXECD-ACCESS-TOKEN") != token || token == "" {
				http.Error(w, `{"message":"unauthorized"}`, 401)
				return
			}
			if strings.HasSuffix(r.URL.Path, "/command") && r.Method == http.MethodPost {
				var command struct {
					Envs map[string]string `json:"envs"`
					Argv []string          `json:"argv"`
				}
				require.NoError(t, json.NewDecoder(r.Body).Decode(&command))
				value, present := command.Envs["EXECD_ACCESS_TOKEN"]
				require.True(t, present)
				require.Empty(t, value)
				require.NotEmpty(t, command.Argv)
				w.Header().Set("Content-Type", "text/event-stream")
				_, _ = w.Write([]byte("{\"type\":\"init\",\"text\":\"script-command\"}\n\n{\"type\":\"execution_complete\",\"exit_code\":0}\n\n"))
				return
			}
			if strings.HasSuffix(r.URL.Path, "/files/download") {
				_, _ = w.Write([]byte(`{"schemaVersion":"title-result-v1","title":"Formatted bottle title","diagnostics":[]}`))
				return
			}
			_, _ = w.Write([]byte(`{}`))
		}
	}))
	t.Cleanup(server.Close)
	directory := t.TempDir()
	ca := filepath.Join(directory, "ca.pem")
	credential := filepath.Join(directory, "client.json")
	require.NoError(t, os.WriteFile(ca, pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: server.Certificate().Raw}), 0600))
	require.NoError(t, os.WriteFile(credential, []byte(`{"user":"fixture","password":"synthetic-service-password","lifecycle":"synthetic-lifecycle-key"}`), 0600))
	settings.ScriptSandbox = &skillsandbox.Config{Domain: strings.TrimPrefix(server.URL, "https://"), CAFile: ca, CredentialFile: credential, TrialEvidence: "sha256:" + strings.Repeat("c", 64), Descriptor: agent.ScriptDescriptor{Profile: "python-title-json-v1", Revision: "1", NetworkRevision: "1", ToolVersion: "v1.0.0", ImageDigest: "sha256:" + strings.Repeat("a", 64), ExecdDigest: "sha256:" + strings.Repeat("b", 64)}}
	return &creations
}
