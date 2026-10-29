//go:build integration

package knowledge_test

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"os"
	"os/exec"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"task-processor/internal/agent"
	"task-processor/internal/authidentity"
	"task-processor/internal/authz"
	"task-processor/internal/commercetool"
	einoruntime "task-processor/internal/integration/agent/eino"
	"task-processor/internal/integration/commercetoolauth"
	"task-processor/internal/integration/knowledgeauth"
	agentstore "task-processor/internal/integration/persistence/agent"
	store "task-processor/internal/integration/persistence/knowledge"
	k "task-processor/internal/knowledge"
	"task-processor/internal/product/catalog/tools/canonicalinspect"
	"task-processor/internal/workbenchcontext"
)

// This is an OS-killed test process, not a current-application crash test. Both
// databases belong to this test's existing disposable PostgreSQL fixture.
func TestKnowledgeContextPostgresPreclaimProcessCrashRecovery(t *testing.T) {
	f := newContextFixture(t)
	source := f.source("frozen before process interruption", "INCOMPLETE_EXTRACTION")
	request := f.request()
	require.NoError(t, f.db.Exec("CREATE DATABASE agent558").Error)
	endpoint, err := url.Parse(f.dsn)
	require.NoError(t, err)
	endpoint.Path = "/agent558"
	agentDB := preclaimDatabase(t, endpoint.String())
	require.NoError(t, agentstore.InstallSchema(agentDB))
	input := preclaimInput{KnowledgeDSN: f.dsn, AgentDSN: endpoint.String(), Request: request}

	crashed := startPreclaimProcess(t, input)
	var committed preclaimReady
	crashed.read(t, &committed)
	require.Equal(t, crashed.cmd.Process.Pid, committed.PID)
	require.True(t, committed.Ref.Valid())
	frozen, err := f.repo.ReadContext(context.Background(), f.scope, committed.Ref)
	require.NoError(t, err)
	require.Len(t, frozen.Entries, 1)
	require.Equal(t, source.CurrentReadableRevisionID, frozen.Entries[0].RevisionID)
	preclaimRows(t, f.db, "knowledge_context_bundles", 1)
	preclaimRows(t, agentDB, "product_agent_runs", 0)
	preclaimRows(t, f.db, "knowledge_dispatch_permits", 0)

	// The child has acknowledged Materialize's committed return and is blocked
	// before constructing/calling Runtime.Start. Kill only this exec-owned PID.
	require.NoError(t, crashed.cmd.Process.Kill())
	var exit *exec.ExitError
	require.ErrorAs(t, crashed.cmd.Wait(), &exit)
	require.False(t, exit.Success())
	t.Logf("killed owned test PID %d after bundle commit, before Runtime.Start/Claim", committed.PID)
	preclaimRows(t, agentDB, "product_agent_runs", 0)

	newRevision := f.apply(k.Command{Kind: "revision_create", SourceID: source.ID, Version: source.Version, Name: source.Name,
		Upload: &k.Revision{Filename: "next.txt", ContentType: "text/plain", SizeBytes: 5, SHA256: k.Digest([]byte("newer"))}})
	f.promote(*newRevision.Revision, "new current text must not replace frozen context", "")
	current, err := f.repo.GetSource(context.Background(), f.scope.OrganizationID, source.ID)
	require.NoError(t, err)
	require.Equal(t, newRevision.Revision.ID, current.CurrentReadableRevisionID)
	require.NotEqual(t, frozen.Entries[0].RevisionID, current.CurrentReadableRevisionID)

	// A new process receives exactly the same pre-Start command, not the old ref.
	// It reopens both native repositories and reauthorizes before adoption.
	retried := startPreclaimProcess(t, input)
	var adopted preclaimReady
	retried.read(t, &adopted)
	require.Equal(t, retried.cmd.Process.Pid, adopted.PID)
	require.Equal(t, committed.Ref, adopted.Ref)
	require.Equal(t, committed.Citations, adopted.Citations)
	require.NoError(t, json.NewEncoder(retried.input).Encode(true))
	var result preclaimResult
	retried.read(t, &result)
	err = retried.cmd.Wait()
	require.NoError(t, err, retried.stderr.String())
	require.EqualValues(t, 1, result.Acquired)
	require.EqualValues(t, 1, result.Quotes)
	require.Zero(t, result.Decisions)
	require.GreaterOrEqual(t, result.KnowledgeChecks, 2)
	require.GreaterOrEqual(t, result.AgentChecks, int32(8))
	require.Len(t, result.RunIDs, 8)
	for _, id := range result.RunIDs {
		require.Equal(t, result.Record.State.RunID, id)
	}
	require.Equal(t, agent.ContextSnapshotRef{Kind: committed.Ref.Kind, ID: committed.Ref.ID, Digest: committed.Ref.Digest}, result.Record.State.Request.ContextSnapshotRef)
	require.Equal(t, agent.Stopped, result.Record.State.Phase)
	require.Equal(t, agent.StopDependency, result.Record.State.StopReason)
	require.Zero(t, result.Record.State.Usage.ModelCalls)
	require.Zero(t, result.Record.State.Usage.Tokens)
	require.Zero(t, result.Record.State.Usage.CostMicros)
	preclaimRows(t, agentDB, "product_agent_runs", 1)
	preclaimRows(t, f.db, "knowledge_context_bundles", 1)
	preclaimRows(t, f.db, "knowledge_context_bundle_entries", 1)
	preclaimRows(t, f.db, "knowledge_dispatch_permits", 0)
	got, err := f.repo.ReadContext(context.Background(), f.scope, adopted.Ref)
	require.NoError(t, err)
	require.Equal(t, frozen, got)
	raw, err := json.Marshal(result.Record)
	require.NoError(t, err)
	require.NotContains(t, string(raw), frozen.Entries[0].Text)
	require.NotContains(t, string(raw), "new current text")
}

type preclaimInput struct {
	KnowledgeDSN, AgentDSN string
	Request                k.ContextRequest
}

type preclaimReady struct {
	PID       int
	Ref       k.ContextSnapshotRef
	Citations []k.Citation
}

type preclaimResult struct {
	Record                      agent.Record
	RunIDs                      []string
	Acquired, Quotes, Decisions int32
	KnowledgeChecks             int
	AgentChecks                 int32
}

type preclaimProcess struct {
	cmd    *exec.Cmd
	input  io.WriteCloser
	output *bufio.Scanner
	stderr bytes.Buffer
}

func startPreclaimProcess(t *testing.T, input preclaimInput) *preclaimProcess {
	t.Helper()
	executable, err := os.Executable()
	require.NoError(t, err)
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	p := &preclaimProcess{cmd: exec.CommandContext(ctx, executable, "-test.run=^TestKnowledgeContextPreclaimProcess$", "-test.timeout=50s")}
	p.cmd.Env = append(os.Environ(), "KNOWLEDGE558_TEST_PROCESS=1")
	p.cmd.Stderr = &p.stderr
	p.input, err = p.cmd.StdinPipe()
	require.NoError(t, err)
	output, err := p.cmd.StdoutPipe()
	require.NoError(t, err)
	p.output = bufio.NewScanner(output)
	p.output.Buffer(nil, agent.MaxStateBytes)
	require.NoError(t, p.cmd.Start())
	t.Cleanup(func() {
		cancel() // CommandContext owns only the process started above.
		_ = p.input.Close()
		if p.cmd.ProcessState == nil {
			_ = p.cmd.Wait()
		}
	})
	// Connection settings stay on private stdin, never argv, environment or logs.
	require.NoError(t, json.NewEncoder(p.input).Encode(input))
	return p
}

func (p *preclaimProcess) read(t *testing.T, result any) {
	t.Helper()
	var diagnostics []string
	// Dependency initialization may log to stdout. Only the test's framed
	// messages acknowledge the barrier; a log line cannot release/claim a run.
	for p.output.Scan() {
		if data, ok := strings.CutPrefix(p.output.Text(), "KNOWLEDGE558:"); ok {
			require.NoError(t, json.Unmarshal([]byte(data), result))
			return
		}
		diagnostics = append(diagnostics, p.output.Text())
	}
	require.NoError(t, p.output.Err())
	t.Fatalf("test child exited before its preclaim protocol message: %s", strings.Join(diagnostics, "\n"))
}

func writePreclaimMessage(t *testing.T, message any) {
	t.Helper()
	data, err := json.Marshal(message)
	require.NoError(t, err)
	_, err = fmt.Fprintln(os.Stdout, "KNOWLEDGE558:"+string(data))
	require.NoError(t, err)
}

func preclaimDatabase(t *testing.T, dsn string) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	require.NoError(t, err)
	pool, err := db.DB()
	require.NoError(t, err)
	pool.SetMaxOpenConns(8)
	t.Cleanup(func() { _ = pool.Close() })
	return db
}

func preclaimRows(t *testing.T, db *gorm.DB, table string, expected int64) {
	t.Helper()
	var count int64
	require.NoError(t, db.Table(table).Count(&count).Error)
	require.Equal(t, expected, count, table)
}

// Runs only when explicitly launched by startPreclaimProcess. No production
// client/transport, provider credentials, usage ledger or recovery loop exists.
func TestKnowledgeContextPreclaimProcess(t *testing.T) {
	if os.Getenv("KNOWLEDGE558_TEST_PROCESS") != "1" {
		t.Skip("helper for the owned preclaim crash regression")
	}
	input := preclaimInput{}
	decoder := json.NewDecoder(os.Stdin)
	require.NoError(t, decoder.Decode(&input))
	ctx := context.Background()
	repo, err := store.NewRepository(ctx, preclaimDatabase(t, input.KnowledgeDSN))
	require.NoError(t, err)
	client := &contextGrantClient{roles: []string{authz.EnterpriseRoleKey(input.Request.Scope.OrganizationID, 1)}}
	resolver := workbenchcontext.NewResolver(workbenchcontext.NewGrantResolver(client, nil), "project", "v1", nil)
	policy, err := authz.NewListingKitAuthorizer(nil, nil)
	require.NoError(t, err)
	policy.SetRolePolicyReader(contextRolePolicy{})
	auth, err := knowledgeauth.NewAuthorizer(resolver, policy)
	require.NoError(t, err)
	service, err := k.NewContextService(repo, auth)
	require.NoError(t, err)
	ctx = commercetoolauth.WithOrganizationRequest(ctx, commercetoolauth.OrganizationRequest{
		Identity:    authidentity.AuthenticatedIdentity{UserID: input.Request.Scope.ActorID, HomeOrganizationID: input.Request.Scope.OrganizationID, TokenExpiresAt: time.Now().Add(time.Hour)},
		BearerToken: "synthetic-test-token", RequestedOrganizationID: input.Request.Scope.OrganizationID})
	ref, err := service.Materialize(ctx, input.Request)
	require.NoError(t, err)
	bundle, err := service.ReadContext(ctx, input.Request.Scope, ref)
	require.NoError(t, err)
	ready := preclaimReady{PID: os.Getpid(), Ref: ref}
	for _, entry := range bundle.Entries {
		ready.Citations = append(ready.Citations, entry.Citation)
	}
	writePreclaimMessage(t, ready)
	var proceed bool
	require.NoError(t, decoder.Decode(&proceed)) // Parent kills here, or releases retry.
	require.True(t, proceed)

	native, err := agentstore.New(preclaimDatabase(t, input.AgentDSN))
	require.NoError(t, err)
	claims := &preclaimStore{Store: native}
	model := &preclaimModel{}
	agentAuth := &preclaimAgentAuthorizer{scope: agent.Scope{OrganizationID: input.Request.Scope.OrganizationID, ActorID: input.Request.Scope.ActorID}, binding: input.Request.Binding}
	tool := canonicalinspect.Definition()
	runtime, err := einoruntime.New(einoruntime.Config{
		Definition: commercetool.AgentDefinition{ID: "product.diagnose", Version: "v1.0.0", AllowedTools: []commercetool.ToolRef{tool.Ref}}, Tools: []commercetool.Definition{tool},
		Store: claims, Model: model, Gateway: preclaimUnusedPorts{}, Validator: preclaimUnusedPorts{}, Authorizer: agentAuth})
	require.NoError(t, err)
	request := agent.Request{Key: input.Request.Key, Binding: input.Request.Binding,
		ContextSnapshotRef: agent.ContextSnapshotRef{Kind: ref.Kind, ID: ref.ID, Digest: ref.Digest}, PolicyVersion: "title-v1", PromptVersion: "prompt-v1",
		Limits: agent.Limits{Steps: 8, ModelCalls: 3, Tokens: 100, CostMicros: 100, Currency: "USD", Runtime: time.Minute}}
	var wg sync.WaitGroup
	records := make(chan agent.Record, 8)
	errors := make(chan error, 8)
	for range 8 {
		wg.Go(func() { r, err := runtime.Start(ctx, request); records <- r; errors <- err })
	}
	wg.Wait()
	close(records)
	close(errors)
	for err := range errors {
		require.NoError(t, err)
	}
	result := preclaimResult{Acquired: claims.acquired.Load(), Quotes: model.quotes.Load(), Decisions: model.decisions.Load(), KnowledgeChecks: client.calls, AgentChecks: agentAuth.calls.Load()}
	for r := range records {
		result.RunIDs = append(result.RunIDs, r.State.RunID)
	}
	result.Record, err = native.Read(ctx, agentAuth.scope, request.Binding.ContextID, request.Key)
	require.NoError(t, err)
	writePreclaimMessage(t, result)
}

type preclaimStore struct {
	agent.Store
	acquired atomic.Int32
}

func (s *preclaimStore) Claim(ctx context.Context, record agent.Record, expected uint64) (agent.Record, bool, error) {
	r, acquired, err := s.Store.Claim(ctx, record, expected)
	if acquired {
		s.acquired.Add(1)
	}
	return r, acquired, err
}

type preclaimAgentAuthorizer struct {
	scope   agent.Scope
	binding agent.Binding
	calls   atomic.Int32
}

func (a *preclaimAgentAuthorizer) Authorize(_ context.Context, binding agent.Binding) (agent.Scope, error) {
	a.calls.Add(1)
	if binding != a.binding {
		return agent.Scope{}, agent.ErrInvalid
	}
	return a.scope, nil
}

type preclaimUnusedPorts struct {
	agent.ToolGateway
	agent.Validator
}

type preclaimModel struct{ quotes, decisions atomic.Int32 }

func (m *preclaimModel) Quote(context.Context, agent.ModelInput) (agent.Quote, error) {
	m.quotes.Add(1)
	return agent.Quote{}, agent.ErrUnavailable // Stop before any dispatch/accounting.
}

func (m *preclaimModel) Decide(context.Context, agent.ModelInput) (agent.ModelResult, error) {
	m.decisions.Add(1)
	return agent.ModelResult{}, agent.ErrUnavailable
}
