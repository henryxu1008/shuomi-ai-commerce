package httpapi

import (
	"context"
	"encoding/json"

	"task-processor/internal/agent"
	"task-processor/internal/aiworkbench"
	"task-processor/internal/authidentity"
	"task-processor/internal/product/review"
)

func (a *productAgentApplication) authorizeWorkbenchReview(ctx context.Context, execution review.ExecutionProvenance) error {
	if execution.WorkbenchContextRef.Absent() {
		return nil
	}
	if a == nil || a.store == nil {
		return review.ErrUnavailable
	}
	i, ok := authidentity.AuthenticatedIdentityFromContext(ctx)
	if !ok || i.TenantID != execution.Scope.OrganizationID || i.UserID != execution.Scope.ActorID {
		return review.ErrNotFound
	}
	run, found, err := a.store.Lookup(ctx, execution.Scope, execution.Binding, execution.RequestKey)
	if err != nil || !found || run.State.RunID != execution.RunID ||
		run.State.Request.ConfigurationSnapshotRef != execution.Configuration ||
		run.State.Request.WorkbenchContextRef != execution.WorkbenchContextRef {
		return review.ErrNotFound
	}
	_, err = a.readContentWorkbenchContext(ctx, agent.ExecutionIdentity{OrganizationID: i.TenantID, ActorID: i.UserID, MemberID: i.EffectiveMemberID},
		execution.WorkbenchContextRef, execution.RequestKey, execution.Binding, true)
	if err != nil {
		return review.ErrNotFound
	}
	return nil
}

type workbenchQuoteKey struct{}
type workbenchQuote struct {
	ref     agent.WorkbenchContextRef
	key     string
	binding agent.Binding
}

func (a *productAgentApplication) readContentWorkbenchContext(ctx context.Context, identity agent.ExecutionIdentity, ref agent.WorkbenchContextRef, key string, binding agent.Binding, send bool) (any, error) {
	if a == nil || a.workbenchScopes == nil {
		return nil, agent.ErrUnavailable
	}
	w := a.workbenchScopes
	owner := aiworkbench.Scope{OrganizationID: identity.OrganizationID, ActorID: identity.ActorID}
	// T1 quote has not created a Task. Its exact, server-only authorization
	// is never accepted for a send or for subsequent runtime work.
	preview, ok := ctx.Value(workbenchQuoteKey{}).(workbenchQuote)
	if !ok || send || preview != (workbenchQuote{ref, key, binding}) {
		task, err := w.store.TaskForExecution(ctx, owner, key)
		if err != nil || task.ScheduledOccurrenceID == "" && task.ConversationID != ref.ConversationID || task.WorkbenchContextRef != ref {
			return nil, agent.ErrUnavailable
		}
		var request agent.Request
		if json.Unmarshal(task.ExecutionRequest, &request) != nil || request.Binding != binding {
			return nil, agent.ErrUnavailable
		}
	}
	f, err := w.authorizedFrozenScope(ctx, owner, ref, false)
	if err != nil || f.MemberID != identity.MemberID {
		return nil, agent.ErrUnavailable
	}
	command, err := w.store.GetCommand(ctx, owner, ref.CommandKey)
	if err != nil || command.ScopeRef() != ref || command.AssistantMessageID == "" {
		return nil, agent.ErrUnavailable
	}
	seq, err := w.store.AssistantSequence(ctx, owner, ref.ConversationID, command.AssistantMessageID)
	if err != nil {
		return nil, agent.ErrUnavailable
	}
	var current authidentity.AuthenticatedIdentity
	if scheduledExecutionContext(ctx) {
		current, err = a.currentAgentIdentity(ctx)
	} else {
		var ok bool
		current, ok = authidentity.AuthenticatedIdentityFromContext(ctx)
		if !ok {
			return nil, agent.ErrUnavailable
		}
	}
	if err != nil || current.TenantID != owner.OrganizationID || current.UserID != owner.ActorID || current.EffectiveMemberID != identity.MemberID || w.authorizeProductHistory(ctx, owner, ref.ConversationID, seq, current) != nil {
		return nil, agent.ErrUnavailable
	}
	return f.EffectiveModelInput(), nil
}
