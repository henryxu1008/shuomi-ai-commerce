package httpapi

import (
	"context"
	"task-processor/internal/authz"
	"task-processor/internal/product/sourcing"
	"task-processor/internal/storecenter"
)

type scheduledSourceAuthorization struct{ agent *productAgentApplication }

func (s scheduledSourceAuthorization) Authorize(ctx context.Context) (sourcing.PublicationScope, error) {
	i, roles, err := s.agent.scheduledPrincipal(ctx)
	if err != nil {
		return sourcing.PublicationScope{}, err
	}
	allowed, err := authz.AuthorizeOrganization(ctx, s.agent.authorizer, i.ActorID, i.OrganizationID, roles, authz.PermissionProductSourcingWrite)
	if err != nil {
		return sourcing.PublicationScope{}, sourcing.ErrSourcePublicationUnavailable
	}
	if !allowed {
		return sourcing.PublicationScope{}, sourcing.ErrPublicationForbidden
	}
	return sourcing.PublicationScope{OrganizationID: i.OrganizationID, ActorID: i.ActorID}, nil
}

type scheduledStoreMemberAuthorizer struct{ app *aiWorkbenchApplication }

func (s scheduledStoreMemberAuthorizer) AuthorizeStoreMember(ctx context.Context, org string) (storecenter.StoreMemberAccess, error) {
	i, roles, err := s.app.agent.scheduledPrincipal(ctx)
	if err != nil || i.OrganizationID != org {
		return storecenter.StoreMemberAccess{}, storecenter.ErrNotFound
	}
	allowed, err := authz.AuthorizeOrganization(ctx, s.app.agent.authorizer, i.ActorID, org, roles, authz.PermissionWorkbenchStoreRead)
	if err != nil {
		return storecenter.StoreMemberAccess{}, storecenter.ErrDependencyUnavailable
	}
	if !allowed {
		return storecenter.StoreMemberAccess{}, storecenter.ErrNotFound
	}
	return storecenter.StoreMemberAccess{OrganizationID: org, ActorID: i.ActorID, MemberID: i.MemberID, Administrator: s.app.agent.authorizer.IsTenantAdmin(i.ActorID, roles)}, nil
}
