package httpapi

import (
	"context"
	"errors"

	"task-processor/internal/agent"
	"task-processor/internal/agentconfig"
	confighttp "task-processor/internal/agentconfig/httpapi"
	"task-processor/internal/agentcustomization"
	"task-processor/internal/authidentity"
	"task-processor/internal/authz"
	"task-processor/internal/commercetool"
)

type privateModeDeliveryReader interface {
	Delivery(context.Context, agentcustomization.Scope, string) (agentcustomization.Delivery, error)
}

func agentModeIdentity(ctx context.Context, h *confighttp.Handler, write bool) (agent.ExecutionIdentity, error) {
	if h == nil || h.Authorize == nil {
		return agent.ExecutionIdentity{}, agentconfig.ErrUnavailable
	}
	permissions := []string{authz.PermissionWorkbenchAgentRead, authz.PermissionWorkbenchAgentConfigure}
	if write {
		permissions = []string{authz.PermissionWorkbenchAgentConfigure}
	}
	scope, err := h.Authorize(ctx, permissions...)
	if err != nil {
		return agent.ExecutionIdentity{}, err
	}
	current, ok := authidentity.AuthenticatedIdentityFromContext(ctx)
	if !ok || current.UserID != scope.ActorID || current.EffectiveOrganizationID != scope.OrganizationID || current.TenantID != scope.OrganizationID || !agent.ValidID(current.EffectiveMemberID) {
		return agent.ExecutionIdentity{}, agentconfig.ErrForbidden
	}
	return agent.ExecutionIdentity{OrganizationID: scope.OrganizationID, ActorID: scope.ActorID, MemberID: current.EffectiveMemberID}, nil
}

func agentModeSubjectResolver(h *confighttp.Handler, private privateModeDeliveryReader) func(context.Context, agentconfig.ModeSubject, bool) (agent.ExecutionIdentity, agentconfig.ResolvedModeSubject, error) {
	return func(ctx context.Context, subject agentconfig.ModeSubject, write bool) (agent.ExecutionIdentity, agentconfig.ResolvedModeSubject, error) {
		var identity agent.ExecutionIdentity
		var resolved agentconfig.ResolvedModeSubject
		if h == nil || h.Authorize == nil || h.Catalog == nil || h.Repository == nil || !subject.Valid() {
			return identity, resolved, agentconfig.ErrUnavailable
		}
		identity, err := agentModeIdentity(ctx, h, write)
		if err != nil {
			return identity, resolved, err
		}
		scope := agent.Scope{OrganizationID: identity.OrganizationID, ActorID: identity.ActorID}
		resolved.Subject = subject
		if subject.Kind == agentconfig.RegisteredAgentSubject {
			entries, err := h.Catalog.ReadCatalog(ctx)
			if err != nil {
				return identity, resolved, err
			}
			for _, entry := range entries {
				if entry.Definition.ID == subject.ID {
					resolved.Definition = commercetool.AgentRef{ID: entry.Definition.ID, Version: entry.Definition.Version}
					break
				}
			}
			if resolved.Definition.ID == "" {
				return identity, resolved, agentconfig.ErrNotFound
			}
			activation, err := h.Repository.ReadAgent(ctx, scope, subject.ID)
			if errors.Is(err, agentconfig.ErrNotFound) {
				return identity, resolved, agentconfig.ErrNotEnabled
			}
			if err != nil {
				return identity, resolved, err
			}
			if activation.Activation != "ENABLED" && activation.Activation != "DISABLED" {
				return identity, resolved, agentconfig.ErrNotEnabled
			}
			return identity, resolved, nil
		}
		if private == nil {
			return identity, resolved, agentconfig.ErrUnavailable
		}
		delivery, err := private.Delivery(ctx, agentcustomization.Scope{OrganizationID: scope.OrganizationID, ActorID: scope.ActorID}, subject.ID)
		if errors.Is(err, agentcustomization.ErrNotFound) {
			return identity, resolved, agentconfig.ErrNotFound
		}
		if errors.Is(err, agentcustomization.ErrForbidden) {
			return identity, resolved, agentconfig.ErrForbidden
		}
		if err != nil {
			return identity, resolved, agentconfig.ErrUnavailable
		}
		if delivery.OrganizationID != scope.OrganizationID || delivery.ID != subject.ID {
			return identity, resolved, agentconfig.ErrForbidden
		}
		if delivery.Definition != agentcustomization.QualityDefinition || delivery.Version != agentcustomization.QualityVersion {
			return identity, resolved, agentconfig.ErrDefinition
		}
		resolved.Definition = commercetool.AgentRef{ID: delivery.Definition, Version: delivery.Version}
		return identity, resolved, nil
	}
}
