package httpapi

import (
	"context"
	"encoding/json"
	"errors"
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

func testScheduleExistingChatProposal(t *testing.T, ctx context.Context, f *acquisitionHTTPFixture, w *aiWorkbenchApplication, proposalID string, calls *atomic.Int32, projectRevoked *atomic.Bool) {
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
	var request agent.Request
	require.NoError(t, json.Unmarshal(task.ExecutionRequest, &request))
	run, found, err := w.agent.store.Lookup(worker, agent.Scope{OrganizationID: "B", ActorID: "admin"}, request.Binding, request.Key)
	require.NoError(t, err)
	require.True(t, found)
	require.Equal(t, agent.HumanReviewRequired, run.State.Phase, "stop=%s", run.State.StopReason)
	before := calls.Load()
	require.Positive(t, before)
	projectRevoked.Store(true)
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
