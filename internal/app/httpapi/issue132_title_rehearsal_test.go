package httpapi

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"task-processor/internal/agent"
	"task-processor/internal/app/productsourcing"
	"task-processor/internal/authidentity"
	"task-processor/internal/commercetool"
	"task-processor/internal/product/catalog"
	"task-processor/internal/product/catalog/tools/canonicalinspect"
	reviewproposal "task-processor/internal/product/review/proposal"
	"task-processor/internal/product/sourcing"
	"task-processor/internal/workbenchcontext"
)

func TestIssue132OfflineTitleRehearsal(t *testing.T) {
	dsn := os.Getenv("ISSUE132_TEST_DSN")
	if dsn == "" {
		t.Skip("requires a task-owned ISSUE132_TEST_DSN")
	}
	require.Contains(t, dsn, "dbname=issue132_rehearsal")
	t.Setenv("ISSUE398_TEST_DSN", dsn)
	_, cases := loadSharedRegression(t, "postgres")
	samples := make(map[string]sharedCase)
	for _, sample := range cases {
		samples[sample.CaseID] = sample
	}
	testProductAgentOwnersWithTitleRehearsal(t, "", &titleRehearsal{samples: samples})
}

type titleRehearsal struct {
	samples   map[string]sharedCase
	pilot     bool
	unknownAt int
	mu        sync.Mutex
	actions   []agent.Action
	calls     []titleRehearsalCall
}

type titleRehearsalCall struct {
	options  map[string]json.RawMessage
	system   string
	input    agent.ModelInput
	evidence json.RawMessage
}

func titleRehearsalLoopback(t *testing.T, endpoint string) http.RoundTripper {
	t.Helper()
	target, err := url.Parse(endpoint)
	require.NoError(t, err)
	require.Equal(t, "http", target.Scheme)
	require.Equal(t, "127.0.0.1", target.Hostname())
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.Proxy = nil
	t.Cleanup(transport.CloseIdleConnections)
	return acquisitionHTTPTransport(func(req *http.Request) (*http.Response, error) {
		if req.URL.Scheme != target.Scheme || req.URL.Host != target.Host {
			return nil, errors.New("title rehearsal allows only its loopback model")
		}
		return transport.RoundTrip(req)
	})
}

func (r *titleRehearsal) reply(w http.ResponseWriter, req *http.Request) {
	var options map[string]json.RawMessage
	var messages []struct{ Role, Content string }
	if req.Method != http.MethodPost || json.NewDecoder(req.Body).Decode(&options) != nil ||
		json.Unmarshal(options["messages"], &messages) != nil || len(messages) != 2 ||
		messages[0].Role != "system" || messages[1].Role != "user" {
		http.Error(w, "unexpected title model request", http.StatusBadRequest)
		return
	}
	var payload struct{ Input agent.ModelInput }
	if json.Unmarshal([]byte(messages[1].Content), &payload) != nil {
		http.Error(w, "invalid title model input", http.StatusBadRequest)
		return
	}
	delete(options, "messages")
	call := titleRehearsalCall{options: options, system: messages[0].Content, input: payload.Input}
	for _, observation := range call.input.History {
		if observation.Tool == canonicalinspect.Definition().Ref {
			call.evidence = observation.Output
		}
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if len(r.calls) >= len(r.actions) {
		http.Error(w, "unexpected extra model call", http.StatusBadRequest)
		return
	}
	content, err := json.Marshal(r.actions[len(r.calls)])
	if err != nil {
		http.Error(w, "invalid scripted action", http.StatusInternalServerError)
		return
	}
	r.calls = append(r.calls, call)
	w.Header().Set("Content-Type", "application/json")
	response := map[string]any{
		"id":      "issue132-loopback",
		"choices": []any{map[string]any{"message": map[string]string{"content": string(content)}, "finish_reason": "stop"}},
	}
	if r.pilot {
		response["model"] = "gpt-5.4-mini-2026-03-17"
	}
	if len(r.calls) != r.unknownAt {
		response["usage"] = map[string]any{"prompt_tokens": 20, "completion_tokens": 10, "total_tokens": 30, "prompt_tokens_details": map[string]int{"cached_tokens": 0}}
	}
	_ = json.NewEncoder(w).Encode(response)
}

func (r *titleRehearsal) script(actions ...agent.Action) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.actions, r.calls = actions, nil
}

func (r *titleRehearsal) captured(t *testing.T) []titleRehearsalCall {
	t.Helper()
	r.mu.Lock()
	defer r.mu.Unlock()
	require.Len(t, r.calls, len(r.actions))
	return append([]titleRehearsalCall(nil), r.calls...)
}

// These are test acquisition receipts for the frozen shared snapshots, whose
// authored source IDs cannot enter the numeric 1688 acquisition endpoint.
type titleRehearsalReceipts map[string]sourcing.PublishedAcquisition

func (r titleRehearsalReceipts) ReadPublished(ctx context.Context, id string) (sourcing.PublishedAcquisition, error) {
	identity, ok := authidentity.AuthenticatedIdentityFromContext(ctx)
	item, found := r[id]
	if !ok || !found || identity.TenantID != item.Result.Operation.Scope.OrganizationID ||
		identity.UserID != item.Result.Operation.Scope.ActorID {
		return sourcing.PublishedAcquisition{}, sourcing.ErrAcquisitionNotFound
	}
	return item, nil
}

func (r *titleRehearsal) run(t *testing.T, db *gorm.DB, app *productAgentApplication) {
	t.Helper()
	resolver, ok := app.resolver.(*workbenchcontext.Resolver)
	require.True(t, ok)
	ctx, err := candidateReviewFixtureContext(context.Background(), titleVerifier{}, resolver, "admin", "B")
	require.NoError(t, err)
	producer, err := productsourcing.NewInternalProducer(db, titleSetupLiveAccess{}, app.authorizer)
	require.NoError(t, err)
	receipts := titleRehearsalReceipts{}
	app.receipts = receipts
	profile, err := app.selectTitleProfile(ctx, "B")
	require.NoError(t, err)
	t.Logf("synthetic_model=%s pricing=%s prompt_version=%s", profile.ModelID, profile.CostPricingVersion, profile.PromptVersion)
	var fixedTotals, agentTotals titleRehearsalTotals
	for _, id := range []string{"PS-001", "PS-003", "PS-002"} {
		t.Run(id, func(t *testing.T) {
			sample := r.samples[id]
			require.NotNil(t, sample.Source)
			require.NotNil(t, sample.Snapshot)
			published, err := producer.Publish(ctx, sourcing.PublicationCommand{
				PublicationID: uuid.NewString(), ProductKey: id, Envelope: *sample.Source,
				Producer: sourcing.ProducerDescriptor{Kind: sourcing.ControlledSnapshotProducerKind, Version: sourcing.ControlledSnapshotProducerVersion},
			})
			require.NoError(t, err)
			source, err := producer.Read(ctx, published.PublicationID)
			require.NoError(t, err)
			require.Equal(t, *sample.Snapshot, source.Snapshot)
			base, err := app.snapshots.GetCurrentSnapshot(ctx, catalog.SnapshotIdentity{TenantID: published.OrganizationID, ProductKey: published.ProductKey})
			require.NoError(t, err)
			require.Equal(t, source.Snapshot, base.Snapshot)
			operation := uuid.NewString()
			receipts[operation] = sourcing.PublishedAcquisition{
				Snapshot: base,
				Result: sourcing.AcquisitionResult{Publication: &source, Operation: sourcing.AcquisitionOperation{
					ID: operation, State: sourcing.AcquisitionPublished,
					Scope: sourcing.PublicationScope{OrganizationID: "B", ActorID: "admin"},
				}},
			}
			identity, err := app.freshIdentity(ctx)
			require.NoError(t, err)
			binding, err := app.currentAcquisitionBinding(ctx, identity, operation, "shein")
			require.NoError(t, err)
			request, err := app.startRequest(ctx, binding, uuid.NewString(), "", nil, "0")
			require.NoError(t, err)
			evidenceID, err := reviewproposal.CanonicalEvidenceID(source.Envelope)
			require.NoError(t, err)

			require.Contains(t, strings.ToLower(source.Snapshot.Description), "steel bottle")
			required := []string{"steel", "bottle"}
			if source.Snapshot.Brand != "" {
				required = append(required, strings.ToLower(source.Snapshot.Brand))
			}
			fixedAction := rehearsalTitle("Example Steel Bottle", evidenceID)
			agentActions := []agent.Action{{Kind: "tool", Tool: canonicalinspect.Definition().Ref}}
			if id == "PS-003" {
				// Exercise a rejected proposal and one repair independently of
				// the reviewer rule that catches the unsupported brand.
				agentActions = append(agentActions, rehearsalTitle("Example Steel Bottle", "unobserved"))
			}
			title := "Steel Bottle"
			if source.Snapshot.Brand != "" {
				title = "Example " + title
			}
			if id == "PS-002" {
				require.Empty(t, source.Snapshot.Title)
				require.Empty(t, source.Snapshot.Images)
				fixedAction = agent.Action{Kind: "interrupt", Unresolved: []string{"missing original title and images"}}
			}
			agentActions = append(agentActions, rehearsalTitle(title, evidenceID))

			fixedRequest, err := app.startRequest(ctx, binding, uuid.NewString(), "", nil, "0")
			require.NoError(t, err)
			r.script(fixedAction)
			fixed := rehearseFixedTitle(t, ctx, app, fixedRequest)
			fixedCalls := r.captured(t)
			require.Len(t, fixedCalls, 1)

			r.script(agentActions...)
			started := time.Now()
			record, err := app.runtime.Start(ctx, request)
			elapsed := time.Since(started)
			require.NoError(t, err)
			state := record.State
			agentCalls := r.captured(t)
			if id == "PS-003" {
				require.EqualValues(t, 1, state.Repairs)
				require.NotNil(t, agentCalls[2].input.Validation)
				require.False(t, agentCalls[2].input.Validation.Valid)
				t.Log("agent intermediate proposal: unsupported brand with unobserved evidence, rejected before repair")
			}
			require.Equal(t, agent.HumanReviewRequired, state.Phase, "%+v", state.Validation)
			require.NotNil(t, state.Validation)
			dynamic := titleRehearsalResult{run: state.RunID, candidate: state.Candidate,
				valid: state.Validation.Valid, outcome: string(state.Phase), elapsed: elapsed,
				unresolved: state.Unresolved}

			left, right := fixedCalls[0], agentCalls[len(agentCalls)-1]
			require.NotEmpty(t, left.evidence)
			require.JSONEq(t, string(left.evidence), string(right.evidence))
			require.Equal(t, left.options, right.options)
			require.Equal(t, left.system, right.system)
			require.Equal(t, left.input.Binding, right.input.Binding)
			require.Equal(t, left.input.GoalSummary, right.input.GoalSummary)
			require.Equal(t, left.input.PolicyVersion, right.input.PolicyVersion)
			require.Equal(t, left.input.PromptVersion, right.input.PromptVersion)

			fixedScore := reportTitleRehearsal(t, db, "fixed", fixed, required, len(fixedCalls))
			agentScore := reportTitleRehearsal(t, db, "agent", dynamic, required, len(agentCalls))
			if id == "PS-002" {
				require.Equal(t, "interrupt", fixed.outcome)
				require.Zero(t, agentScore.unsupported)
				t.Log("missing-input case excluded from paired totals")
				return
			}
			fixedTotals.add(fixedScore)
			agentTotals.add(agentScore)
		})
	}
	require.Equal(t, titleRehearsalTotals{covered: 5, required: 5, unsupported: 1, claims: 6, valid: 2, admitted: 2, edits: 1, reviewable: 2}, fixedTotals)
	require.Equal(t, titleRehearsalTotals{covered: 5, required: 5, unsupported: 0, claims: 5, valid: 2, admitted: 2, edits: 0, reviewable: 2}, agentTotals)
	t.Logf("paired fixed=%+v agent=%+v", fixedTotals, agentTotals)
	var proposals int64
	require.NoError(t, db.Table("product_title_proposals").Count(&proposals).Error)
	require.Zero(t, proposals, "rehearsal ends before saving or applying a proposal")
}

type titleRehearsalResult struct {
	run        string
	candidate  reviewproposal.Candidate
	valid      bool
	outcome    string
	elapsed    time.Duration
	unresolved []string
}

func rehearsalTitle(title, evidence string) agent.Action {
	return agent.Action{Kind: "propose", Candidate: reviewproposal.Candidate{
		Changes: []reviewproposal.FieldChange{{Field: "title", Value: title, EvidenceIDs: []string{evidence}}},
	}}
}

func rehearseFixedTitle(t *testing.T, ctx context.Context, app *productAgentApplication, request agent.Request) titleRehearsalResult {
	t.Helper()
	started := time.Now()
	scope, err := app.Authorize(ctx, request.Binding)
	require.NoError(t, err)
	raw, err := json.Marshal(request)
	require.NoError(t, err)
	// Tool audits reference a run in PostgreSQL even in the fixed test group.
	record, claimed, err := app.store.Claim(ctx, agent.Record{State: agent.State{
		Request: request, RunID: uuid.NewString(), TraceID: uuid.NewString(),
		Scope: scope, Fingerprint: fmt.Sprintf("%x", sha256.Sum256(raw)),
		Phase: agent.Running, StartedAt: started, Deadline: started.Add(request.Limits.Runtime),
		HumanReviewRequired: true,
	}}, 0)
	require.NoError(t, err)
	require.True(t, claimed)
	state := record.State
	tool := canonicalinspect.Definition().Ref
	call := commercetool.CallMetadata{CallID: uuid.NewString(), AgentID: app.definition.ID, AgentVersion: app.definition.Version,
		AgentRunID: state.RunID, BusinessTaskID: request.Binding.ContextID, TraceID: state.TraceID}
	facts, err := app.Invoke(ctx, tool, call, request.Binding)
	require.NoError(t, err)
	require.Equal(t, commercetool.AuditStatusRecorded, facts.AuditStatus)
	state.History = []agent.Observation{{Step: 1, Tool: tool, CallID: call.CallID, Output: facts.Output, AuditStatus: facts.AuditStatus}}

	input := state.ModelInput(app.definition)
	quote, err := app.model.Quote(ctx, input)
	require.NoError(t, err)
	var usage agent.Usage
	require.Empty(t, usage.Reserve(request.Limits, quote))
	input.UpperBound, input.InvocationID = quote, uuid.NewString()
	generated, err := app.model.Decide(ctx, input)
	require.NoError(t, err)
	require.Equal(t, input.InvocationID, generated.InvocationID)
	require.Empty(t, usage.Settle(quote, generated.Usage))
	result := titleRehearsalResult{run: state.RunID, candidate: generated.Action.Candidate, outcome: generated.Action.Kind, unresolved: generated.Action.Unresolved}
	if generated.Action.Kind == "propose" {
		validated, err := app.Validate(ctx, request, result.candidate, state.History)
		require.NoError(t, err)
		result.valid = validated.Valid
		result.unresolved = validated.Unresolved
	}
	result.elapsed = time.Since(started)
	return result
}

type titleRehearsalTotals struct {
	covered, required, unsupported, claims, valid, admitted, edits, reviewable int
}

func (s *titleRehearsalTotals) add(other titleRehearsalTotals) {
	s.covered += other.covered
	s.required += other.required
	s.unsupported += other.unsupported
	s.claims += other.claims
	s.valid += other.valid
	s.admitted += other.admitted
	s.edits += other.edits
	s.reviewable += other.reviewable
}

func reportTitleRehearsal(t *testing.T, db *gorm.DB, group string, result titleRehearsalResult, required []string, calls int) titleRehearsalTotals {
	t.Helper()
	score := titleRehearsalTotals{required: len(required), admitted: 1}
	known := make(map[string]bool)
	for _, fact := range required {
		known[fact] = true
	}
	title := ""
	if len(result.candidate.Changes) > 0 {
		require.Len(t, result.candidate.Changes, 1)
		require.Equal(t, "title", result.candidate.Changes[0].Field)
		title = result.candidate.Changes[0].Value
	}
	// A closed vocabulary for these scripted titles. This simulates scoring
	// and reviewer edits only, without claiming human review or model quality.
	seen := make(map[string]bool)
	for _, word := range strings.Fields(strings.ToLower(title)) {
		if seen[word] {
			continue
		}
		seen[word] = true
		score.claims++
		if !known[word] {
			score.unsupported++
		} else if result.valid {
			score.covered++
		}
	}
	edit := "N/A"
	if result.valid {
		score.valid, score.reviewable = 1, 1
		edit = "0"
		if score.covered < score.required || score.unsupported > 0 {
			score.edits, edit = 1, "1"
		}
	}
	var usage []struct {
		PromptTokens, CompletionTokens, EstimatedCostMicros int64
		UsageKnown, EstimatedCostKnown                      bool
	}
	require.NoError(t, db.Table("ai_invocations").
		Select("prompt_tokens, completion_tokens, estimated_cost_micros, usage_known, estimated_cost_known").
		Where("agent_run_id = ?", result.run).Find(&usage).Error)
	require.Len(t, usage, calls)
	var input, output, cost int64
	for _, call := range usage {
		require.True(t, call.UsageKnown && call.EstimatedCostKnown, "incomplete synthetic usage")
		input += call.PromptTokens
		output += call.CompletionTokens
		cost += call.EstimatedCostMicros
	}
	risk := "N/A"
	if score.claims > 0 {
		risk = fmt.Sprintf("%d/%d", score.unsupported, score.claims)
	}
	t.Logf("group=%s title=%q outcome=%s unresolved=%v completeness=%d/%d unsupported=%s validator=%d/%d simulated_edits=%s elapsed=%s calls=%d prompt_tokens=%d completion_tokens=%d synthetic_cost_micros=%d",
		group, title, result.outcome, result.unresolved, score.covered, score.required, risk, score.valid, score.admitted, edit, result.elapsed, calls, input, output, cost)
	return score
}
