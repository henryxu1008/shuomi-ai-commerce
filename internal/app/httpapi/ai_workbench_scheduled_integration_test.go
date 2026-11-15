package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
	"task-processor/internal/agent"
	"task-processor/internal/agentconfig"
	"task-processor/internal/aicapability"
	"task-processor/internal/aiworkbench"
	"task-processor/internal/app/productsourcing"
	"task-processor/internal/authidentity"
	"task-processor/internal/authz"
	"task-processor/internal/commercetool"
	configstore "task-processor/internal/integration/persistence/agentconfig"
	workstore "task-processor/internal/integration/persistence/aiworkbench"
	collectionstore "task-processor/internal/integration/persistence/product/collection"
	"task-processor/internal/product/collection"
	"task-processor/internal/product/review"
	reviewproposal "task-processor/internal/product/review/proposal"
)

func TestScheduledContentUsesOriginalOwnersWithoutBrowserToken(t *testing.T) {
	for _, mode := range []string{"default", "read_only", "auto_approve", "full_access", "revoked-before-send", "unknown"} {
		t.Run(mode, func(t *testing.T) { testProductAgentOwners(t, "own content scheduled "+mode) })
	}
}

func TestScheduledChatContentKeepsScopeAndOriginalSourceOwners(t *testing.T) {
	for _, mode := range []string{"own content chat scheduled", "content flow chat scheduled"} {
		t.Run(mode, func(t *testing.T) { testProductAgentOwners(t, mode) })
	}
}

func testScheduleExistingChatProposal(t *testing.T, ctx context.Context, f *acquisitionHTTPFixture, w *aiWorkbenchApplication, proposalID string, calls *atomic.Int32, projectRevoked *atomic.Bool, deps routeAuthDependencies) {
	require.NoError(t, w.connectScheduledExecution(ctx, w.agent.config.ReviewDB, f.db))
	owner := aiworkbench.Scope{OrganizationID: "B", ActorID: "admin"}
	p, err := w.store.GetProposal(ctx, owner, proposalID)
	require.NoError(t, err)
	require.True(t, p.WorkbenchContextRef.Valid())
	var profile aicapability.ModelProfile
	require.NoError(t, json.Unmarshal(p.ExecutionModelProfile, &profile))
	budget, err := aiworkbench.BudgetForScheduledExecution(profile, p.ExecutionLimits, 1)
	require.NoError(t, err)
	spec, err := json.Marshal(p)
	require.NoError(t, err)
	now := time.Now().UTC().Truncate(time.Second)
	plan := aiworkbench.ScheduledPlanDraft{ID: uuid.NewString(), Scope: owner, MemberID: "member-B-admin", Selection: *p.Selection(), GoalSummary: p.GoalSummary, ExecutionSpec: spec, Rule: aiworkbench.ScheduleRule{Frequency: aiworkbench.ScheduleOnce, TimeZone: "UTC", AtUTC: now.Add(time.Second)}, Budget: budget, CreatedAt: now}
	require.NoError(t, w.store.CreateScheduledPlan(ctx, plan))
	require.NoError(t, w.store.MarkScheduleRegistered(ctx, owner, plan.ID, 1))
	worker, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	require.NoError(t, w.DispatchScheduled(worker, plan.ID, plan.Rule.AtUTC))
	occurrence, err := w.store.ReadScheduledOccurrence(worker, owner, plan.ID, plan.Rule.AtUTC)
	require.NoError(t, err)
	task, err := w.store.GetTask(worker, owner, occurrence.BusinessTaskID)
	require.NoError(t, err)
	require.Equal(t, p.WorkbenchContextRef, task.WorkbenchContextRef)
	require.Empty(t, task.ConversationID)
	testScheduledTaskListing(t, w, deps, task.ID, "all", true)
	frozen, err := w.store.ReadFrozenScope(ctx, owner, task.WorkbenchContextRef)
	require.NoError(t, err)
	testScheduledTaskListing(t, w, deps, task.ID, "project&projectId="+frozen.Scope.ProjectID, true)
	var request agent.Request
	require.NoError(t, json.Unmarshal(task.ExecutionRequest, &request))
	run, found, err := w.agent.store.Lookup(worker, agent.Scope{OrganizationID: "B", ActorID: "admin"}, request.Binding, request.Key)
	require.NoError(t, err)
	require.True(t, found)
	require.Equal(t, agent.HumanReviewRequired, run.State.Phase, "stop=%s", run.State.StopReason)
	before := calls.Load()
	require.Positive(t, before)
	previousBackend := w.schedules
	w.schedules = &scheduledBackendFixture{}
	selected := *p.Selection()
	quoteInput := scheduledInput{ScheduleID: plan.ID, ExpectedRevision: "2", Selection: &selected, GoalSummary: "Updated scoped content", Rule: aiworkbench.ScheduleRule{Frequency: aiworkbench.ScheduleDaily, TimeZone: "UTC", Hour: 9}, MaxOccurrences: "2"}
	quoted, replanned, _, quoteErr := w.quoteScheduled(ctx, owner, quoteInput)
	require.NoError(t, quoteErr)
	require.Equal(t, p.WorkbenchContextRef, replanned.WorkbenchContextRef)
	require.Equal(t, "1", quoted.ConsumedOccurrences)
	quoteInput.ScheduleID = ""
	quoteInput.ExpectedRevision = ""
	quoteInput.SourceScheduleID = plan.ID
	quoteInput.MaxOccurrences = "1"
	_, copied, _, quoteErr := w.quoteScheduled(ctx, owner, quoteInput)
	require.NoError(t, quoteErr)
	require.Equal(t, p.WorkbenchContextRef, copied.WorkbenchContextRef)
	require.Equal(t, before, calls.Load(), "editing and copying quote must not call the model")
	w.schedules = previousBackend
	testScheduledHistoryAfterGoalEdit(t, w, deps, task, request)
	projectRevoked.Store(true)
	testScheduledTaskListing(t, w, deps, task.ID, "all", false)
	plan.ID = uuid.NewString()
	require.NoError(t, w.store.CreateScheduledPlan(ctx, plan))
	require.NoError(t, w.store.MarkScheduleRegistered(ctx, owner, plan.ID, 1))
	require.NoError(t, w.DispatchScheduled(worker, plan.ID, plan.Rule.AtUTC))
	blocked, err := w.store.ReadScheduledOccurrence(worker, owner, plan.ID, plan.Rule.AtUTC)
	require.NoError(t, err)
	require.Equal(t, "BLOCKED", blocked.State)
	require.Equal(t, before, calls.Load(), "revoked project scope must not dispatch another model call")
}

type scheduledStatusFixture struct{ suspended, failed bool }

func (s *scheduledStatusFixture) IsOrganizationSuspended(context.Context, string) (bool, error) {
	if s.failed {
		return false, errors.New("isolated organization status unavailable")
	}
	return s.suspended, nil
}

func testScheduledContentThroughOwners(t *testing.T, f *acquisitionHTTPFixture, module productAgentModule, deps routeAuthDependencies, permissions *authz.ListingKitAuthorizer, configuration *configstore.Store, calls *atomic.Int32, scenario string) {
	ctx := context.Background()
	live := &productReviewLiveOrganizationAccess{resolver: deps.organizationResolver, now: time.Now}
	authority, err := collection.NewContextAuthorizer(live, permissions)
	require.NoError(t, err)
	require.NoError(t, collectionstore.InstallSchema(f.owner))
	repository, err := collectionstore.NewRepository(ctx, f.owner, func(tx *gorm.DB) (collectionstore.OwnPublisher, error) {
		return productsourcing.NewOwnProductWriter(tx, authority)
	})
	require.NoError(t, err)
	sources, err := productsourcing.NewPublishedAcquisitionReader(ctx, f.owner, live, permissions)
	require.NoError(t, err)
	collections, err := collection.NewService(repository, authority, sources)
	require.NoError(t, err)
	a := module.application.content
	collections.WithSnapshots(a.snapshots)
	a.ownProducts = collections
	actor := authidentity.WithAuthenticatedIdentity(ctx, authidentity.AuthenticatedIdentity{TenantID: "B", EffectiveOrganizationID: "B", UserID: "admin", EffectiveMemberID: "member-B-admin", Roles: []string{"listingkit_admin"}, TokenExpiresAt: time.Now().Add(time.Hour)})
	actor, err = (productReviewCapabilityBinder{}).Bind(actor, "Bearer admin")
	require.NoError(t, err)
	saved, err := collections.Mutate(actor, uuid.NewString(), collection.Mutation{Action: "create_product", Product: &collection.OwnProduct{Title: "Saved own product", Images: []string{"https://images.example.org/own.png"}}})
	require.NoError(t, err)
	_, err = configuration.Execute(ctx, agentconfig.Command{Scope: agent.Scope{OrganizationID: "B", ActorID: "admin"}, Key: uuid.NewString(), AgentID: agentconfig.ContentAgentID, Operation: "enable", Absent: true})
	require.NoError(t, err)
	if scenario != "default" && scenario != "revoked-before-send" && scenario != "unknown" {
		_, err = configuration.ExecuteMode(ctx, agentconfig.ModeCommand{Identity: agent.ExecutionIdentity{OrganizationID: "B", ActorID: "admin", MemberID: "member-B-admin"}, Subject: a.modeSubject(), Key: uuid.NewString(), Mode: agent.ExecutionMode(scenario)})
		require.NoError(t, err)
	}
	identity, _ := authidentity.AuthenticatedIdentityFromContext(actor)
	item, err := a.ownSource(actor, identity, saved.ItemID, nil)
	require.NoError(t, err)
	require.NoError(t, workstore.InstallSchema(f.owner))
	store, err := workstore.New(f.owner)
	require.NoError(t, err)
	w := &aiWorkbenchApplication{store: store, agent: a, allowedOrganizationIDs: []string{"B"}}
	a.workbenchScopes = w
	var revoked atomic.Bool
	a.originRoles = func(_ context.Context, i agent.ExecutionIdentity) ([]string, error) {
		if revoked.Load() || i != (agent.ExecutionIdentity{OrganizationID: "B", ActorID: "admin", MemberID: "member-B-admin"}) {
			return nil, review.ErrForbidden
		}
		if f.grants.revoked.Load() {
			return nil, nil
		} // collection read is revoked after original reservation
		return []string{"listingkit_admin"}, nil
	}
	module.application.originRoles = a.originRoles
	require.NoError(t, w.connectScheduledExecution(ctx, a.config.ReviewDB, f.db))
	selection := aiworkbench.TaskSelection{Subject: aiworkbench.Subject{Kind: item.Binding.ContextKind, ID: item.Binding.ContextID, CatalogVersion: item.Binding.CatalogVersion, PublicationID: item.Binding.PublicationID}, Capability: aiworkbench.ContentTaskKind, Parameters: reviewproposal.ContentSelection{TargetLanguage: "en-US", Title: true}}
	if scenario == "default" {
		testScheduledHTTPManagement(t, w, deps, selection, f.owner)
	}
	p, err := (&workbenchPlanner{agent: a, workbench: w}).buildProposal(actor, aiworkbench.PlanningCommand{Scope: aiworkbench.Scope{OrganizationID: "B", ActorID: "admin"}, WorkScope: aiworkbench.WorkScope{TaskSelection: &selection}}, "Localize product")
	require.NoError(t, err)
	spec, err := json.Marshal(p)
	require.NoError(t, err)
	now := time.Now().UTC().Truncate(time.Second)
	var profile aicapability.ModelProfile
	require.NoError(t, json.Unmarshal(p.ExecutionModelProfile, &profile))
	budget, err := aiworkbench.BudgetForScheduledExecution(profile, p.ExecutionLimits, 1)
	require.NoError(t, err)
	plan := aiworkbench.ScheduledPlanDraft{ID: uuid.NewString(), Scope: p.Scope, MemberID: "member-B-admin", Selection: selection, GoalSummary: p.GoalSummary, ExecutionSpec: spec, Rule: aiworkbench.ScheduleRule{Frequency: aiworkbench.ScheduleOnce, TimeZone: "UTC", AtUTC: now.Add(time.Second)}, Budget: budget, CreatedAt: now}
	require.NoError(t, store.CreateScheduledPlan(ctx, plan))
	require.NoError(t, store.MarkScheduleRegistered(ctx, plan.Scope, plan.ID, 1))
	worker, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()
	status := &scheduledStatusFixture{suspended: true}
	a.organizationStatus = status
	require.NoError(t, w.DispatchScheduled(worker, plan.ID, plan.Rule.AtUTC))
	blocked, err := store.ReadScheduledOccurrence(worker, plan.Scope, plan.ID, plan.Rule.AtUTC)
	require.NoError(t, err)
	require.Equal(t, "BLOCKED", blocked.State)
	require.Zero(t, calls.Load(), "a suspended organization must not dispatch")
	plan.ID = uuid.NewString()
	require.NoError(t, store.CreateScheduledPlan(ctx, plan))
	require.NoError(t, store.MarkScheduleRegistered(ctx, plan.Scope, plan.ID, 1))
	status.suspended, status.failed = false, true
	require.NoError(t, w.DispatchScheduled(worker, plan.ID, plan.Rule.AtUTC))
	blocked, err = store.ReadScheduledOccurrence(worker, plan.Scope, plan.ID, plan.Rule.AtUTC)
	require.NoError(t, err)
	require.Equal(t, "BLOCKED", blocked.State)
	require.Zero(t, calls.Load(), "status dependency failure must fail closed")
	status.failed = false
	plan.ID = uuid.NewString()
	require.NoError(t, store.CreateScheduledPlan(ctx, plan))
	require.NoError(t, store.MarkScheduleRegistered(ctx, plan.Scope, plan.ID, 1))
	executionCtx := context.WithValue(worker, scheduledExecutionKey{}, scheduledExecutionCommand{workbench: w, planID: plan.ID, slot: plan.Rule.AtUTC})
	// A selected store uses the scheduled principal rather than a browser token.
	w.attachStoreConnectionScope(nil, nil)
	w.scopes.scheduledConnectionRead = func(_ context.Context, org, id string) (aiworkbench.StoreConnectionFacts, error) {
		if org != "B" || id != "store-b" {
			return aiworkbench.StoreConnectionFacts{}, aiworkbench.ErrNotFound
		}
		return aiworkbench.StoreConnectionFacts{Version: 1, ObservedAt: "2026-11-14T00:00:00Z", Connected: true}, nil
	}
	_, err = w.scopes.reader.Connection(executionCtx, plan.Scope, "store-b")
	require.NoError(t, err)
	_, err = (scheduledStoreMemberAuthorizer{app: w}).AuthorizeStoreMember(executionCtx, "B")
	require.NoError(t, err)
	revoked.Store(true)
	_, err = w.scopes.reader.Connection(executionCtx, plan.Scope, "store-b")
	require.Error(t, err)
	_, err = (scheduledStoreMemberAuthorizer{app: w}).AuthorizeStoreMember(executionCtx, "B")
	require.Error(t, err)
	revoked.Store(false)
	_, err = (productAgentSourceGateway{application: a}).ReadEffectiveSource(context.WithValue(executionCtx, productAgentSourceBindingKey{}, item.Binding), commercetool.Principal{TenantID: "B", UserID: "admin", Roles: []string{"listingkit_admin"}}, item.Effective)
	require.NoError(t, err, "source evidence must authorize the same tokenless original owner")
	err = w.DispatchScheduled(worker, plan.ID, plan.Rule.AtUTC)
	if scenario == "auto_approve" || scenario == "full_access" {
		require.NoError(t, err)
		blocked, err = store.ReadScheduledOccurrence(worker, plan.Scope, plan.ID, plan.Rule.AtUTC)
		require.NoError(t, err)
		require.Equal(t, "BLOCKED", blocked.State)
		require.Zero(t, calls.Load(), "automatic modes are never scheduled")
		return
	}
	require.NoError(t, err)
	occurrence, err := store.ReadScheduledOccurrence(worker, plan.Scope, plan.ID, plan.Rule.AtUTC)
	require.NoError(t, err)
	require.NotEmpty(t, occurrence.BusinessTaskID)
	task, err := store.GetTask(worker, plan.Scope, occurrence.BusinessTaskID)
	require.NoError(t, err)
	var request agent.Request
	require.NoError(t, json.Unmarshal(task.ExecutionRequest, &request))
	require.True(t, request.ManualReviewOnly)
	run, found, err := a.store.Lookup(worker, agent.Scope{OrganizationID: "B", ActorID: "admin"}, request.Binding, request.Key)
	require.NoError(t, err)
	require.True(t, found)
	if scenario == "revoked-before-send" {
		require.Equal(t, agent.Stopped, run.State.Phase)
		require.Zero(t, calls.Load(), "revocation after reservation must stop the final provider send")
		return
	}
	if scenario == "default" {
		testScheduledHistoryAfterGoalEdit(t, w, deps, task, request)
		testScheduledTaskListing(t, w, deps, task.ID, "all", true)
		testScheduledTaskListing(t, w, deps, task.ID, "unrecorded", true)
	}
	complete, err := w.ReconcileScheduled(worker, plan.ID, plan.Rule.AtUTC)
	require.NoError(t, err)
	if scenario == "unknown" {
		require.False(t, complete)
		current, err := store.ReadScheduledPlan(worker, plan.Scope, plan.ID)
		require.NoError(t, err)
		require.Equal(t, occurrence.ID, current.ActiveOccurrenceID)
		require.EqualValues(t, 1, current.ConsumedOccurrences)
		before := calls.Load()
		require.Positive(t, before)
		require.NoError(t, w.DispatchScheduled(worker, plan.ID, plan.Rule.AtUTC))
		require.Equal(t, before, calls.Load(), "unknown cannot repeat the paid send")
		return
	}
	require.True(t, complete)
	currentPlan, err := store.ReadScheduledPlan(worker, plan.Scope, plan.ID)
	require.NoError(t, err)
	require.Empty(t, currentPlan.ActiveOccurrenceID)
	require.Equal(t, agent.HumanReviewRequired, run.State.Phase, "stop=%s history=%+v unresolved=%v calls=%d", run.State.StopReason, run.State.History, run.State.Unresolved, calls.Load())
	before := calls.Load()
	require.Positive(t, before)
	require.NoError(t, w.DispatchScheduled(worker, plan.ID, plan.Rule.AtUTC))
	require.Equal(t, before, calls.Load(), "same occurrence cannot redispatch a paid model call")
	revoked.Store(true)
	require.Error(t, w.DispatchScheduled(worker, plan.ID, plan.Rule.AtUTC))
	require.Equal(t, before, calls.Load())
}

// A future edit cannot reinterpret a task's already frozen intent or hide its
// review/action entry. This runs for both direct and Chat-originated schedules.
func testScheduledHistoryAfterGoalEdit(t *testing.T, w *aiWorkbenchApplication, deps routeAuthDependencies, task aiworkbench.BusinessTask, request agent.Request) {
	t.Helper()
	previousBackend := w.schedules
	w.schedules = &scheduledBackendFixture{}
	defer func() { w.schedules = previousBackend }()
	server := httptest.NewServer(buildCurrentApplicationHTTPServer(aiWorkbenchRoutes(w), deps).Handler)
	defer server.Close()
	call := func(method, path, key string, input any, want int) []byte {
		var body []byte
		var err error
		if input != nil {
			body, err = json.Marshal(input)
			require.NoError(t, err)
		}
		code, raw, err := acquisitionHTTPRequest(server, method, path, "admin", "B", key, string(body))
		require.NoError(t, err)
		require.Equal(t, want, code, string(raw))
		return raw
	}
	current, err := w.store.ReadScheduledPlan(context.Background(), task.Scope, request.ScheduledSource.PlanID)
	require.NoError(t, err)
	selected := current.Selection
	selected.Parameters.TargetLanguage = "de-DE"
	input := scheduledInput{ScheduleID: current.ID, ExpectedRevision: strconv.FormatInt(current.Revision, 10), Selection: &selected, GoalSummary: "Different future objective", Rule: aiworkbench.ScheduleRule{Frequency: aiworkbench.ScheduleDaily, TimeZone: "UTC", Hour: 9}, MaxOccurrences: "2"}
	var quote scheduledQuote
	require.NoError(t, json.Unmarshal(call("POST", workbenchScheduleBase+"/quote", "", input, 200), &quote))
	require.NotEmpty(t, quote.Confirmation)
	call("PATCH", workbenchScheduleBase+"/"+current.ID, uuid.NewString(), map[string]any{"expectedRevision": input.ExpectedRevision, "rule": input.Rule, "confirmation": quote.Confirmation, "changes": scheduledChanges{Selection: input.Selection, GoalSummary: input.GoalSummary, MaxOccurrences: input.MaxOccurrences}}, 200)
	edited, err := w.store.ReadScheduledPlan(context.Background(), task.Scope, current.ID)
	require.NoError(t, err)
	require.Equal(t, input.GoalSummary, edited.GoalSummary)
	require.NotEqual(t, task.GoalSummary, edited.GoalSummary)
	stored, err := w.store.GetTask(context.Background(), task.Scope, task.ID)
	require.NoError(t, err)
	require.Equal(t, task, stored, "editing future settings must not rewrite the historical task")
	var detail struct{ Task workbenchTaskView }
	require.NoError(t, json.Unmarshal(call("GET", workbenchTaskBase+"/"+task.ID, "", nil, 200), &detail))
	require.Equal(t, task.GoalSummary, detail.Task.GoalSummary)
	require.True(t, detail.Task.ProductDetailsAvailable)
	require.NotNil(t, detail.Task.TaskSelection)
	require.Equal(t, request.ContentSelection, detail.Task.TaskSelection.Parameters)
	identity := authidentity.AuthenticatedIdentity{TenantID: task.Scope.OrganizationID, UserID: task.Scope.ActorID, EffectiveMemberID: current.MemberID}
	wrongOccurrence := task
	wrongOccurrence.ScheduledOccurrenceID = uuid.NewString()
	require.True(t, w.taskHistoryUnavailable(context.Background(), task.Scope, wrongOccurrence, identity), "frozen occurrence must match the task")
	wrongOrigin := task
	wrongOrigin.WorkbenchContextRef = agent.WorkbenchContextRef{Contract: agent.WorkbenchContextContract, ConversationID: uuid.NewString(), CommandKey: uuid.NewString(), Digest: strings.Repeat("a", 64)}
	require.True(t, w.taskHistoryUnavailable(context.Background(), task.Scope, wrongOrigin, identity), "frozen origin must match the task")
	identity.TenantID = "other-tenant"
	require.True(t, w.taskHistoryUnavailable(context.Background(), task.Scope, task, identity), "history cannot cross tenants")
	// An invalid resume stops at ordinary body validation, after authorization,
	// without dispatching a model, creating a review, or changing task state.
	call("POST", workbenchTaskBase+"/"+task.ID+"/resume", uuid.NewString(), map[string]any{}, 400)
}
