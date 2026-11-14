package httpapi

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"time"

	"task-processor/internal/agent"
	"task-processor/internal/aicapability"
	"task-processor/internal/aiworkbench"
	"task-processor/internal/authidentity"
	"task-processor/internal/authz"
	"task-processor/internal/product/collection"
	"task-processor/internal/product/review"
)

// This capability is only constructed by the local application dispatch port.
// Temporal supplies a plan/slot reference, never an identity or permission.
type scheduledExecutionKey struct{}
type scheduledExecutionCommand struct {
	workbench *aiWorkbenchApplication
	planID    string
	slot      time.Time
}

func scheduledExecutionContext(ctx context.Context) bool {
	if ctx == nil {
		return false
	}
	_, ok := ctx.Value(scheduledExecutionKey{}).(scheduledExecutionCommand)
	return ok
}

func (a *productAgentApplication) scheduledPrincipal(ctx context.Context) (agent.ExecutionIdentity, []string, error) {
	command, ok := ctx.Value(scheduledExecutionKey{}).(scheduledExecutionCommand)
	if ok && a != nil && a.content != nil && command.workbench != nil && command.workbench.agent == a.content {
		return a.content.scheduledPrincipal(ctx)
	}
	if !ok || a == nil || a.originRoles == nil || command.workbench == nil || command.workbench.agent != a || command.planID == "" || command.slot.IsZero() {
		return agent.ExecutionIdentity{}, nil, review.ErrForbidden
	}
	if _, present := authidentity.AuthenticatedIdentityFromContext(ctx); present {
		return agent.ExecutionIdentity{}, nil, review.ErrForbidden
	}
	deadline, bounded := ctx.Deadline()
	if !bounded || !time.Now().Before(deadline) {
		return agent.ExecutionIdentity{}, nil, review.ErrForbidden
	}
	plan, err := command.workbench.store.ReadScheduledExecutionPlan(ctx, command.planID)
	if err != nil {
		return agent.ExecutionIdentity{}, nil, err
	}
	if !command.workbench.admittedOrganization(plan.Scope.OrganizationID) || !a.config.Enabled {
		return agent.ExecutionIdentity{}, nil, review.ErrUnavailable
	}
	if a.organizationStatus != nil {
		suspended, err := a.organizationStatus.IsOrganizationSuspended(ctx, plan.Scope.OrganizationID)
		if err != nil {
			return agent.ExecutionIdentity{}, nil, review.ErrUnavailable
		}
		if suspended {
			return agent.ExecutionIdentity{}, nil, review.ErrForbidden
		}
	}
	// New execution may only use an ACTIVE plan. A bound occurrence retains
	// its original task; pausing a plan never cancels that existing task.
	if plan.State != "ACTIVE" {
		occurrence, e := command.workbench.store.ReadScheduledOccurrence(ctx, plan.Scope, plan.ID, command.slot)
		if e != nil || occurrence.BusinessTaskID == "" {
			return agent.ExecutionIdentity{}, nil, review.ErrForbidden
		}
	}
	identity := agent.ExecutionIdentity{OrganizationID: plan.Scope.OrganizationID, ActorID: plan.Scope.ActorID, MemberID: plan.MemberID}
	roles, err := a.originRoles(ctx, identity)
	return identity, roles, err
}

// currentAgentIdentity is a shared application data adapter, not an HTTP
// identity injector. A browser request always uses its original fresh check;
// it can never fall through to durable execution after that check fails.
func (a *productAgentApplication) currentAgentIdentity(ctx context.Context) (authidentity.AuthenticatedIdentity, error) {
	if !scheduledExecutionContext(ctx) {
		return a.freshIdentity(ctx)
	}
	i, roles, err := a.scheduledPrincipal(ctx)
	if err != nil {
		return authidentity.AuthenticatedIdentity{}, err
	}
	return authidentity.AuthenticatedIdentity{TenantID: i.OrganizationID, EffectiveOrganizationID: i.OrganizationID, UserID: i.ActorID, EffectiveMemberID: i.MemberID, Roles: roles}, nil
}

func (a *productAgentApplication) authorizeScheduledSend(ctx context.Context, input aicapability.TextInputIdentity) error {
	i, _, err := a.scheduledPrincipal(ctx)
	if err != nil {
		return err
	}
	if i.OrganizationID != input.OrganizationID || i.ActorID != input.ActorID || i.MemberID != input.MemberID {
		return review.ErrForbidden
	}
	command := ctx.Value(scheduledExecutionKey{}).(scheduledExecutionCommand)
	owner := aiworkbench.Scope{OrganizationID: i.OrganizationID, ActorID: i.ActorID}
	key := aiworkbench.ScheduledOccurrenceID(owner, command.planID, command.slot)
	task, err := command.workbench.store.TaskForExecution(ctx, owner, key)
	if err != nil || task.ScheduledOccurrenceID != key {
		return review.ErrForbidden
	}
	var request agent.Request
	if json.Unmarshal(task.ExecutionRequest, &request) != nil || request.Key != key || !request.ManualReviewOnly {
		return review.ErrForbidden
	}
	identity, err := a.currentAgentIdentity(ctx)
	if err != nil {
		return err
	}
	binding, err := a.workbenchCurrentBinding(ctx, identity, request.Binding)
	if err != nil {
		return err
	}
	if binding != request.Binding {
		return aiworkbench.ErrProposalStale
	}
	if _, err = a.configuration.CheckRegisteredTextExecutionMode(ctx, i, request.ConfigurationSnapshotRef, request.ModeSnapshotRef); err != nil {
		return err
	}
	if err = a.checkManualReviewMode(ctx, i, request); err != nil {
		return err
	}
	if !request.WorkbenchContextRef.Absent() {
		_, err = command.workbench.authorizedFrozenScope(ctx, owner, request.WorkbenchContextRef, false)
	}
	return err
}

type scheduledCollectionAuthorization struct{ agent *productAgentApplication }

func (r scheduledCollectionAuthorization) AuthorizeExecution(ctx context.Context, scope collection.Scope, permission string) error {
	i, roles, err := r.agent.scheduledPrincipal(ctx)
	if err != nil {
		return err
	}
	if permission != collection.PermissionRead || scope != (collection.Scope{OrganizationID: i.OrganizationID, ActorID: i.ActorID, MemberID: i.MemberID}) {
		return review.ErrForbidden
	}
	allowed, err := authz.AuthorizeOrganization(ctx, r.agent.authorizer, i.ActorID, i.OrganizationID, roles, permission)
	if err != nil {
		return review.ErrUnavailable
	}
	if !allowed {
		return review.ErrForbidden
	}
	return nil
}

func (a *aiWorkbenchApplication) prepareScheduledExecution(ctx context.Context, plan aiworkbench.ScheduledPlan, key string) (aiworkbench.PreparedTask, error) {
	var p aiworkbench.ExecutionProposal
	if a == nil || a.agent == nil || json.Unmarshal(plan.ExecutionSpec, &p) != nil || p.Scope != plan.Scope || p.Kind != aiworkbench.ContentTaskKind || p.Selection() == nil || *p.Selection() != plan.Selection || p.GoalSummary != plan.GoalSummary {
		return aiworkbench.PreparedTask{}, aiworkbench.ErrInvalid
	}
	var profile aicapability.ModelProfile
	if json.Unmarshal(p.ExecutionModelProfile, &profile) != nil {
		return aiworkbench.PreparedTask{}, aiworkbench.ErrInvalidSchedule
	}
	budget, err := aiworkbench.BudgetForScheduledExecution(profile, p.ExecutionLimits, plan.Budget.MaxOccurrences)
	if err != nil || budget != plan.Budget {
		return aiworkbench.PreparedTask{}, aiworkbench.ErrInvalidSchedule
	}
	return (workbenchExecution{agent: a.agent, workbench: a}).prepareTask(ctx, p, key, true)
}

// DispatchScheduled is injected into the local Temporal activity only. It
// reloads the identity and frozen specification from Workbench PostgreSQL,
// persists one legal BusinessTask and calls the original guarded Start kernel.
func (a *aiWorkbenchApplication) DispatchScheduled(ctx context.Context, planID string, slot time.Time) error {
	if a == nil || a.store == nil || ctx == nil {
		return aiworkbench.ErrUnavailable
	}
	ctx, cancel := context.WithTimeout(ctx, 3*time.Minute)
	defer cancel()
	if _, present := authidentity.AuthenticatedIdentityFromContext(ctx); present {
		return review.ErrForbidden
	}
	plan, err := a.store.ReadScheduledExecutionPlan(ctx, planID)
	if err != nil {
		return err
	}
	if plan.Report != nil {
		return a.dispatchDailyReport(ctx, plan, slot)
	}
	if !a.contentSchedulesEnabled() {
		return aiworkbench.ErrUnavailable
	}
	ctx = context.WithValue(ctx, scheduledExecutionKey{}, scheduledExecutionCommand{workbench: a, planID: planID, slot: slot.UTC()})
	occurrence, err := a.store.ReadScheduledOccurrence(ctx, plan.Scope, plan.ID, slot)
	var task aiworkbench.BusinessTask
	if err == nil {
		if occurrence.BusinessTaskID == "" {
			return nil
		}
		task, err = a.store.GetTask(ctx, plan.Scope, occurrence.BusinessTaskID)
		if err != nil {
			return err
		}
	} else if !errors.Is(err, aiworkbench.ErrNotFound) {
		return err
	} else {
		if plan.State == "PAUSED" || plan.State == "CLOSED" {
			return a.store.RecordScheduledPreflightReceipt(ctx, plan.Scope, plan.ID, slot, time.Now().UTC(), "SKIPPED", "PLAN_NOT_ACTIVE")
		}
		if plan.State != "ACTIVE" || plan.RegistrationState != "MATCHED" {
			return aiworkbench.ErrUnavailable
		}
		if _, _, err = a.agent.scheduledPrincipal(ctx); err != nil {
			return a.store.RecordScheduledPreflightReceipt(ctx, plan.Scope, plan.ID, slot, time.Now().UTC(), "BLOCKED", "CURRENT_AUTHORIZATION_UNAVAILABLE")
		}
		key := aiworkbench.ScheduledOccurrenceID(plan.Scope, plan.ID, slot)
		prepared, err := a.prepareScheduledExecution(ctx, plan, key)
		if err != nil {
			return a.store.RecordScheduledPreflightReceipt(ctx, plan.Scope, plan.ID, slot, time.Now().UTC(), "BLOCKED", "FROZEN_EXECUTION_UNAVAILABLE")
		}
		_, task, _, err = a.store.ClaimScheduledTask(ctx, plan.Scope, plan.ID, plan.Revision, slot, time.Now().UTC(), prepared)
		if err != nil {
			return err
		}
		if task.ID == "" {
			return nil
		}
	}
	_, err = (workbenchExecution{agent: a.agent, workbench: a}).startTask(ctx, task)
	if err != nil {
		_ = a.store.NoteScheduledOccurrence(ctx, plan.Scope, task.ScheduledOccurrenceID, "UNKNOWN", "EXECUTION_HANDOFF_UNRESOLVED")
	}
	return err
}

func (a *aiWorkbenchApplication) ReconcileScheduled(ctx context.Context, planID string, slot time.Time) (bool, error) {
	if a == nil || a.store == nil || ctx == nil {
		return false, aiworkbench.ErrUnavailable
	}
	ctx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()
	plan, readErr := a.store.ReadScheduledExecutionPlan(ctx, planID)
	if readErr != nil {
		return false, readErr
	}
	if plan.Report != nil {
		if e := a.dispatchDailyReport(ctx, plan, slot); e != nil {
			return false, e
		}
		o, e := a.store.ReadScheduledOccurrence(ctx, plan.Scope, planID, slot)
		return e == nil && (o.State == "RECONCILED" || o.State == "BLOCKED" || o.State == "SKIPPED"), e
	}
	if !a.contentSchedulesEnabled() {
		return false, aiworkbench.ErrUnavailable
	}
	ctx = context.WithValue(ctx, scheduledExecutionKey{}, scheduledExecutionCommand{workbench: a, planID: planID, slot: slot.UTC()})
	i, _, err := a.agent.scheduledPrincipal(ctx)
	if err != nil {
		return false, err
	}
	owner := aiworkbench.Scope{OrganizationID: i.OrganizationID, ActorID: i.ActorID}
	occurrence, err := a.store.ReadScheduledOccurrence(ctx, owner, planID, slot)
	if err != nil {
		return false, err
	}
	if occurrence.BusinessTaskID == "" {
		return occurrence.State == "SKIPPED" || occurrence.State == "BLOCKED", nil
	}
	task, err := a.store.GetTask(ctx, owner, occurrence.BusinessTaskID)
	if err != nil {
		return false, err
	}
	var request agent.Request
	if json.Unmarshal(task.ExecutionRequest, &request) != nil || !request.ManualReviewOnly || request.ScheduledSource != (agent.ScheduledSourceRef{PlanID: planID, OccurrenceID: occurrence.ID}) {
		return false, aiworkbench.ErrUnavailable
	}
	run, found, err := a.agent.store.Lookup(ctx, agent.Scope{OrganizationID: i.OrganizationID, ActorID: i.ActorID}, request.Binding, request.Key)
	if err != nil || !found {
		return false, err
	}
	state := run.State
	if state.Request != request || state.Scope != (agent.Scope{OrganizationID: i.OrganizationID, ActorID: i.ActorID}) {
		return false, aiworkbench.ErrUnavailable
	}
	if state.Phase != agent.HumanReviewRequired && state.Phase != agent.Stopped || state.PendingInvocationID != "" || state.StopReason == agent.StopModelUnknown || state.StopReason == agent.StopUsageUnknown || state.StopReason == agent.StopExecutionOutcomeUnknown {
		_ = a.store.NoteScheduledOccurrence(ctx, owner, occurrence.ID, "UNKNOWN", "ORIGINAL_RUN_OUTCOME_UNRESOLVED")
		return false, nil
	}
	for _, observation := range state.History {
		if observation.InvocationID == "" {
			continue
		}
		if observation.ObservedUsage == nil || !observation.ObservedUsage.Known {
			return false, nil
		}
		fact, err := a.agent.config.Ledger.ReadModelInvocation(ctx, i.OrganizationID, observation.InvocationID)
		if err != nil {
			return false, err
		}
		if fact.InvocationID != observation.InvocationID || fact.AgentRunID != state.RunID || fact.TenantID != i.OrganizationID || fact.UserID != i.ActorID || fact.MemberID != i.MemberID || fact.Outcome == aicapability.InvocationDispatched || !fact.UsageKnown || !fact.EstimatedCostKnown || fact.FinishedAt.IsZero() || int64(fact.TotalTokens) != observation.ObservedUsage.Tokens || fact.EstimatedCostMicros != observation.ObservedUsage.CostMicros {
			return false, nil
		}
		// The existing resource owner proves or completes settlement under the
		// same invocation key. Unknown facts cannot take this path.
		if err = a.agent.points.SettleAIInvocationUsage(ctx, i.OrganizationID, i.MemberID, fact.InvocationID, int64(fact.TotalTokens), fact.FinishedAt); err != nil {
			return false, err
		}
	}
	wire, err := json.Marshal(struct {
		TaskID, RunID string
		Revision      uint64
		History       []agent.Observation
	}{task.ID, state.RunID, state.Revision, state.History})
	if err != nil {
		return false, aiworkbench.ErrUnavailable
	}
	hash := sha256.Sum256(wire)
	err = a.store.ReconcileScheduledTask(ctx, owner, planID, occurrence.ID, task.ID, hex.EncodeToString(hash[:]))
	return err == nil, err
}
