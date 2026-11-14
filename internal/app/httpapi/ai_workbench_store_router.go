package httpapi

import (
	"context"
	"encoding/json"

	"task-processor/internal/aicapability"
	"task-processor/internal/aiworkbench"
	"task-processor/internal/authz"
	governed "task-processor/internal/integration/aicapability/einomodel"
	"task-processor/internal/integration/aiworkbench/einorouter"
)

// workbenchStoreRouter is the trusted bridge between Chat and the existing
// governed invocation and resource owners. It never receives model-selected
// identities or resource parameters.
type workbenchStoreRouter struct {
	app      *aiWorkbenchApplication
	routes   *governed.OrganizationRouteResolver
	executor *governed.Executor
	allowed  map[string]bool
}

type gatedStoreRouteText struct {
	router  *workbenchStoreRouter
	command aiworkbench.StoreRouteCommand
}

func (g gatedStoreRouteText) Generate(ctx context.Context, input aicapability.TextInputIdentity, quote aicapability.TextQuote, validate func(string) error) (governed.TextOutput, error) {
	if g.router == nil || g.router.executor == nil {
		return governed.TextOutput{}, governed.ErrNotDispatched
	}
	gate := func(gateCtx context.Context) error {
		if err := g.router.Authorize(gateCtx, g.command.Scope, g.command.ConversationID, g.command.MemberID); err != nil {
			return err
		}
		return g.router.app.store.VerifyStoreRouteDispatch(gateCtx, g.command)
	}
	return g.router.executor.GenerateWithGates(ctx, input, quote, validate, gate, func(gateCtx context.Context) (func(), error) { return nil, gate(gateCtx) })
}

func (p *workbenchStoreRouter) Authorize(ctx context.Context, scope aiworkbench.Scope, conversation, member string) error {
	if p == nil || p.app == nil || p.app.store == nil {
		return aiworkbench.ErrUnavailable
	}
	identity, err := p.app.freshWorkbenchIdentity(ctx, authz.PermissionWorkbenchChatUse)
	if err != nil || identity.TenantID != scope.OrganizationID || identity.UserID != scope.ActorID || identity.EffectiveMemberID != member {
		return aiworkbench.ErrNotFound
	}
	if err = p.app.store.ConversationMember(ctx, scope, conversation, member); err != nil {
		return err
	}
	current, err := p.app.store.Get(ctx, scope, conversation)
	if err != nil || current.BusinessScope == nil || current.BusinessScope.Kind != "general" {
		return aiworkbench.ErrNotFound
	}
	return nil
}

func (p *workbenchStoreRouter) Admit(ctx context.Context, scope aiworkbench.Scope, conversation, member, content string) (aiworkbench.StoreRouteRequest, error) {
	if err := p.Authorize(ctx, scope, conversation, member); err != nil {
		return aiworkbench.StoreRouteRequest{}, err
	}
	if p.routes == nil || !p.allowed[scope.OrganizationID] {
		return aiworkbench.StoreRouteRequest{}, aiworkbench.ErrUnavailable
	}
	route, err := p.routes.Resolve(ctx, aicapability.TextInputIdentity{OrganizationID: scope.OrganizationID, Operation: aicapability.OperationAIWorkbenchStoreRoute})
	if err != nil {
		return aiworkbench.StoreRouteRequest{}, aiworkbench.ErrUnavailable
	}
	profile := route.Profile
	// Fail before T0 if the current route cannot form the approved envelope;
	// the final calculation repeats with the owner-assigned invocation ID.
	if _, err = einorouter.Prepare(scope, member, "preflight", content, profile); err != nil {
		return aiworkbench.StoreRouteRequest{}, aiworkbench.ErrUnavailable
	}
	return aiworkbench.StoreRouteRequest{Content: content, MemberID: member, Prepare: func(invocation string) (aiworkbench.StoreRoutePrepared, error) {
		prepared, err := einorouter.Prepare(scope, member, invocation, content, profile)
		if err != nil {
			return aiworkbench.StoreRoutePrepared{}, err
		}
		wire, err := json.Marshal(profile)
		if err != nil {
			return aiworkbench.StoreRoutePrepared{}, aiworkbench.ErrUnavailable
		}
		return aiworkbench.StoreRoutePrepared{InputHash: prepared.Quote.InputHash, ModelProfile: wire, ModelTimeout: profile.DeadlineBound}, nil
	}}, nil
}

func (p *workbenchStoreRouter) Decide(ctx context.Context, command aiworkbench.StoreRouteCommand) (aiworkbench.StoreRoute, error) {
	if err := p.Authorize(ctx, command.Scope, command.ConversationID, command.MemberID); err != nil {
		return aiworkbench.StoreRoute{}, err
	}
	var profile aicapability.ModelProfile
	if json.Unmarshal(command.ModelProfile, &profile) != nil {
		return aiworkbench.StoreRoute{}, aiworkbench.ErrUnavailable
	}
	prepared, err := einorouter.Prepare(command.Scope, command.MemberID, command.InvocationID, command.Content, profile)
	if err != nil || prepared.Quote.InputHash != command.InputHash {
		return aiworkbench.StoreRoute{}, aiworkbench.ErrUnavailable
	}
	return (einorouter.Router{Text: gatedStoreRouteText{router: p, command: command}}).Decide(ctx, prepared)
}

func (p *workbenchStoreRouter) VerifySettled(ctx context.Context, command aiworkbench.StoreRouteCommand) error {
	if p == nil || p.app == nil || p.app.agent == nil || p.app.agent.config.Ledger == nil || p.app.agent.points == nil {
		return aiworkbench.ErrUnavailable
	}
	return verifyStoreRouteSettlement(ctx, command, p.app.agent.config.Ledger.ReadModelInvocation, p.app.agent.points.InspectModelInvocationReservation)
}

func (p *workbenchStoreRouter) InvocationExists(ctx context.Context, invocation string) (bool, error) {
	if p == nil || p.app == nil || p.app.agent == nil || p.app.agent.config.Ledger == nil {
		return false, aiworkbench.ErrUnavailable
	}
	reader, ok := p.app.agent.config.Ledger.(interface {
		HasInvocationID(context.Context, string) (bool, error)
	})
	if !ok {
		return false, aiworkbench.ErrUnavailable
	}
	return reader.HasInvocationID(ctx, invocation)
}

func (p *workbenchStoreRouter) VerifyDecision(ctx context.Context, command aiworkbench.StoreRouteCommand, decision aiworkbench.StoreRoute) error {
	if p == nil || p.app == nil || p.app.agent == nil || p.app.agent.config.Ledger == nil || p.app.agent.points == nil {
		return aiworkbench.ErrUnavailable
	}
	return verifyStoreRouteDecision(ctx, command, decision, p.app.agent.config.Ledger.ReadModelInvocation, p.app.agent.points.InspectModelInvocationReservation)
}

func (p *workbenchStoreRouter) FailureState(ctx context.Context, command aiworkbench.StoreRouteCommand) (aiworkbench.PlanningState, error) {
	if p == nil || p.app == nil || p.app.agent == nil || p.app.agent.config.Ledger == nil {
		return "", aiworkbench.ErrUnavailable
	}
	fact, err := p.app.agent.config.Ledger.ReadModelInvocation(ctx, command.Scope.OrganizationID, command.InvocationID)
	if err != nil || fact.TenantID != command.Scope.OrganizationID || fact.UserID != command.Scope.ActorID || fact.MemberID != command.MemberID || fact.InvocationID != command.InvocationID || fact.InputHash != command.InputHash || fact.Operation != aicapability.OperationAIWorkbenchStoreRoute || !fact.UsageKnown || !fact.EstimatedCostKnown {
		return "", aiworkbench.ErrUnavailable
	}
	// A no-dispatch invocation is written before the resource owner finishes
	// releasing its reservation. The fact alone cannot establish release.
	if fact.Outcome == aicapability.InvocationFailed && fact.PromptTokens == 0 && fact.CompletionTokens == 0 && fact.TotalTokens == 0 && (fact.ErrorCode == "reservation_failed_before_dispatch" || fact.ErrorCode == "rejected_before_dispatch") {
		if p.app.agent.points == nil || p.app.agent.points.ConfirmModelInvocationNoDispatchRelease(ctx, command.Scope.OrganizationID, command.InvocationID) != nil {
			return "", aiworkbench.ErrUnavailable
		}
		return aiworkbench.PlanningFailedBeforeDispatch, nil
	}
	if fact.Outcome != aicapability.InvocationUsageObservedFailed || fact.ErrorCategory != aicapability.ErrorStructuredOutputInvalid || p.app.agent.points == nil {
		return "", aiworkbench.ErrUnavailable
	}
	reservation, err := p.app.agent.points.InspectModelInvocationReservation(ctx, command.Scope.OrganizationID, command.InvocationID)
	if err != nil || reservation.OrganizationID != command.Scope.OrganizationID || reservation.InvocationID != command.InvocationID || reservation.MemberID != command.MemberID || reservation.State != "committed" || reservation.SettledAt == nil || reservation.SettledAt.IsZero() {
		return "", aiworkbench.ErrUnavailable
	}
	return aiworkbench.PlanningInvalidOutput, nil
}
