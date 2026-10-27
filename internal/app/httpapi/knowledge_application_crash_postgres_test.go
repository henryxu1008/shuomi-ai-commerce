//go:build integration

package httpapi

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"task-processor/internal/agent"
	agentstore "task-processor/internal/integration/persistence/agent"
	"task-processor/internal/knowledge"
)

// Runs the ordinary compiled entrypoint twice, with identical private config
// and persisted owners. Only synthetic, test-owned loopback peers are present.
func TestCurrentApplicationKnowledgePreclaimCrashRecovery(t *testing.T) {
	f := newKnowledgeApplicationFixture(t)
	key := uuid.NewString()
	ctx := context.Background()
	barrier := make(chan knowledge.ContextSnapshotRef, 1)
	barrierReleased := make(chan struct{})
	var release sync.Once
	releaseBarrier := func() { release.Do(func() { close(barrierReleased) }) }
	var armed atomic.Bool
	armed.Store(true)
	identity := newAccountProviderFixture(t, "", func(r *http.Request) bool {
		if !armed.Load() {
			return false
		}
		var row struct{ ID, Digest string }
		result := f.knowledgeDB.Table("knowledge_context_bundles").Select("id,digest").Where("organization_id = ? AND actor_id = ? AND context_id = ? AND request_key = ?", "B", "admin", f.operation.OperationID, key).Find(&row)
		if result.Error != nil || row.ID == "" {
			return false
		}
		var runs int64
		if err := f.agent.Table("product_agent_runs").Count(&runs).Error; err != nil || runs != 0 || !armed.CompareAndSwap(true, false) {
			return false
		}
		// A separate committed DB read is the barrier evidence. This live grant
		// lookup is inside Quote's Knowledge read, before Runtime.Start/Claim.
		barrier <- knowledge.ContextSnapshotRef{Kind: knowledge.ContextKind, ID: row.ID, Digest: row.Digest}
		select {
		case <-barrierReleased:
		case <-r.Context().Done():
		case <-time.After(20 * time.Second):
		}
		return true
	})
	t.Cleanup(releaseBarrier) // registered after provider cleanup; never strand it
	identity.role.Store("listingkit_admin")
	var calls, unexpectedIO atomic.Int32
	observed := make(chan knowledge.ContextBundle, 1)
	model := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/chat/completions" || calls.Add(1) != 1 {
			t.Errorf("unexpected synthetic dispatch: %s", r.URL.Path)
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		var wire struct {
			Messages []struct{ Content string } `json:"messages"`
		}
		if err := json.NewDecoder(r.Body).Decode(&wire); err != nil {
			t.Error(err)
			w.WriteHeader(400)
			return
		}
		for _, message := range wire.Messages {
			var prompt struct{ Knowledge *knowledge.ContextBundle }
			if json.Unmarshal([]byte(message.Content), &prompt) == nil && prompt.Knowledge != nil {
				observed <- *prompt.Knowledge
				break
			}
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"id":"isolated-synthetic","choices":[{"message":{"content":"{\"Kind\":\"interrupt\"}"},"finish_reason":"stop"}],"usage":{"prompt_tokens":20,"completion_tokens":10,"total_tokens":30}}`)
	}))
	t.Cleanup(model.Close)
	sink := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		unexpectedIO.Add(1)
		t.Errorf("unexpected object/parser I/O: %s", r.URL.Path)
		w.WriteHeader(503)
	}))
	t.Cleanup(sink.Close)
	f.configureHTTP(t, identity.provider.URL, model.URL, sink.URL)
	data, err := json.Marshal(f.config)
	require.NoError(t, err)
	manifest := writeSMSPrivateConfig(t, string(data))
	binary := buildCurrentApplicationTestBinary(t)
	image, err := os.ReadFile(binary)
	require.NoError(t, err)
	t.Logf("entrypoint=./cmd/current-application; binary SHA256=%x; same binary/manifest/databases for both processes", sha256.Sum256(image))
	origin := applicationCrashAddress(f.config)
	path := productAcquisitionBase + "/" + f.operation.OperationID + "/product-agent/runs"
	body := `{"expectedModeRevision":"0","targetPlatform":"shein","knowledgeSelection":{"knowledgeBaseId":"` + f.knowledge.base.ID + `"}}`
	type reply struct {
		code int
		raw  []byte
		err  error
	}
	client := &http.Client{Timeout: 40 * time.Second}
	t.Cleanup(client.CloseIdleConnections)
	start := func() reply {
		r, e := http.NewRequest("POST", origin+path, strings.NewReader(body))
		if e != nil {
			return reply{err: e}
		}
		r.Header.Set("Content-Type", "application/json")
		r.Header.Set("Authorization", "Bearer admin")
		r.Header.Set("X-Requested-Organization-ID", "B")
		r.Header.Set("Idempotency-Key", key)
		response, e := client.Do(r)
		if e != nil {
			return reply{err: e}
		}
		defer response.Body.Close()
		raw, e := io.ReadAll(response.Body)
		return reply{response.StatusCode, raw, e}
	}
	crashed := startKnowledgeApplicationBinary(t, binary, manifest, origin)
	first := make(chan reply, 1)
	go func() { first <- start() }()
	var ref knowledge.ContextSnapshotRef
	select {
	case ref = <-barrier:
	case result := <-first:
		t.Fatalf("Start finished before committed preclaim barrier: status=%d body=%s error=%v", result.code, result.raw, result.err)
	case <-time.After(25 * time.Second):
		t.Fatal("normal binary did not reach committed preclaim barrier")
	}
	require.True(t, ref.Valid())
	frozen, err := f.knowledge.repo.ReadContext(ctx, knowledge.Scope{OrganizationID: "B", ActorID: "admin"}, ref)
	require.NoError(t, err)
	require.Len(t, frozen.Entries, 1)
	require.Equal(t, f.knowledge.source.CurrentReadableRevisionID, frozen.Entries[0].RevisionID)
	require.Equal(t, "Frozen brand text", frozen.Entries[0].Text)
	require.Equal(t, "INCOMPLETE_EXTRACTION", frozen.Entries[0].Warning)
	for _, table := range []string{"product_agent_runs", "ai_invocations"} {
		require.Zero(t, applicationCrashCount(t, f.agent, table), table)
	}
	require.EqualValues(t, 1, applicationCrashCount(t, f.knowledgeDB, "knowledge_context_bundles"))
	require.Zero(t, applicationCrashCount(t, f.knowledgeDB, "knowledge_dispatch_permits"))
	require.Zero(t, calls.Load())
	require.NoError(t, crashed.command.Process.Kill()) // exact exec-owned OS handle
	<-crashed.done
	releaseBarrier()
	var exit *exec.ExitError
	require.ErrorAs(t, crashed.err, &exit)
	require.False(t, exit.Success())
	require.Error(t, (<-first).err, "crashed HTTP request must not report success")
	t.Logf("normal binary PID=%d killed after committed bundle=%s, before claim; runs/invocations/permits=0", crashed.command.Process.Pid, ref.ID)

	f.knowledge.replaceSource()
	current, err := f.knowledge.repo.GetSource(ctx, "B", f.knowledge.source.ID)
	require.NoError(t, err)
	require.NotEqual(t, frozen.Entries[0].RevisionID, current.CurrentReadableRevisionID)
	restarted := startKnowledgeApplicationBinary(t, binary, manifest, origin)
	replies := make(chan reply, 2)
	for range 2 {
		go func() { replies <- start() }()
	}
	var runID string
	for range 2 {
		result := <-replies
		require.NoError(t, result.err)
		require.Equal(t, 200, result.code, string(result.raw))
		var dto productAgentResultDTO
		require.NoError(t, json.Unmarshal(result.raw, &dto))
		if runID == "" {
			runID = dto.RunID
		}
		require.NotEmpty(t, runID)
		require.Equal(t, runID, dto.RunID)
	}
	result := start() // a later identical Start must also replay the same run
	require.NoError(t, result.err)
	require.Equal(t, 200, result.code, string(result.raw))
	var dto productAgentResultDTO
	require.NoError(t, json.Unmarshal(result.raw, &dto))
	require.Equal(t, runID, dto.RunID)
	require.Equal(t, agent.Interrupted, dto.Phase)
	require.NotNil(t, dto.Knowledge)
	require.Equal(t, "uncited", dto.Knowledge.Status)
	require.Empty(t, dto.Knowledge.Citations, "interrupt does not propose a citation")
	store, err := agentstore.New(f.agent)
	require.NoError(t, err)
	record, err := store.Read(ctx, agent.Scope{OrganizationID: "B", ActorID: "admin"}, f.operation.OperationID, key)
	require.NoError(t, err)
	require.Equal(t, agent.ContextSnapshotRef{Kind: ref.Kind, ID: ref.ID, Digest: ref.Digest}, record.State.Request.ContextSnapshotRef)
	require.Equal(t, agent.Interrupted, record.State.Phase)
	require.EqualValues(t, 1, record.State.Usage.ModelCalls)
	require.EqualValues(t, 30, record.State.Usage.Tokens)
	require.NotEmpty(t, record.Checkpoint)
	raw, err := json.Marshal(record)
	require.NoError(t, err)
	require.NotContains(t, string(raw), "Frozen brand text")
	require.NotContains(t, string(raw), "Newer brand text")
	select {
	case sent := <-observed:
		require.Equal(t, frozen, sent)
	default:
		t.Fatal("synthetic peer did not receive the original Knowledge bundle")
	}
	adopted, err := f.knowledge.repo.ReadContext(ctx, knowledge.Scope{OrganizationID: "B", ActorID: "admin"}, ref)
	require.NoError(t, err)
	require.Equal(t, frozen, adopted)
	require.EqualValues(t, 1, calls.Load())
	require.Zero(t, unexpectedIO.Load())
	for _, table := range []string{"product_agent_runs", "ai_invocations"} {
		require.EqualValues(t, 1, applicationCrashCount(t, f.agent, table), table)
	}
	require.Zero(t, applicationCrashCount(t, f.agent, "product_agent_tool_calls"))
	for _, table := range []string{"product_title_proposals", "product_title_operations"} {
		require.Zero(t, applicationCrashCount(t, f.product, table), table)
	}
	require.EqualValues(t, 1, applicationCrashCount(t, f.knowledgeDB, "knowledge_context_bundles"))
	require.EqualValues(t, 1, applicationCrashCount(t, f.knowledgeDB, "knowledge_context_bundle_entries"))
	var permits []struct{ State, BundleID string }
	require.NoError(t, f.knowledgeDB.Table("knowledge_dispatch_permits").Select("state,bundle_id").Find(&permits).Error)
	require.Equal(t, []struct{ State, BundleID string }{{"RELEASED", ref.ID}}, permits)
	require.NoError(t, os.WriteFile(restarted.shutdown, []byte("stop"), 0600))
	<-restarted.done
	require.NoError(t, restarted.err)
	t.Logf("restarted normal PID=%d; identical concurrent and later Start => run=%s; original ref/digest/revision/citation/text retained; one synthetic invocation (30 tokens), one RELEASED permit; product acceptance NOT_RUN", restarted.command.Process.Pid, runID)
}

type knowledgeApplicationBinary struct {
	command  *exec.Cmd
	done     chan struct{}
	err      error // published by closing done
	shutdown string
}

func startKnowledgeApplicationBinary(t *testing.T, binary, manifest, origin string) *knowledgeApplicationBinary {
	t.Helper()
	dir := t.TempDir()
	logs, err := os.CreateTemp(dir, "application-*.log")
	require.NoError(t, err)
	t.Cleanup(func() { _ = logs.Close() })
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	p := &knowledgeApplicationBinary{command: exec.CommandContext(ctx, binary, "-config", manifest, "-shutdown-file", filepath.Join(dir, "shutdown")), done: make(chan struct{}), shutdown: filepath.Join(dir, "shutdown")}
	p.command.Stdout, p.command.Stderr = logs, logs
	require.NoError(t, p.command.Start())
	go func() { p.err = p.command.Wait(); close(p.done) }()
	t.Cleanup(func() { cancel(); <-p.done })
	client := &http.Client{Timeout: time.Second}
	defer client.CloseIdleConnections()
	deadline := time.After(15 * time.Second)
	ticker := time.NewTicker(50 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-p.done:
			// Application diagnostics never include the private manifest itself.
			output, _ := os.ReadFile(logs.Name())
			t.Fatalf("normal binary exited before HTTP readiness: %v; %s", p.err, output)
		case <-deadline:
			t.Fatal("normal binary HTTP readiness timed out")
		case <-ticker.C:
			response, e := client.Get(origin + "/api/v1/workbench/source-accounts")
			if e == nil {
				_ = response.Body.Close()
				if response.StatusCode == 401 {
					return p
				}
			}
		}
	}
}
