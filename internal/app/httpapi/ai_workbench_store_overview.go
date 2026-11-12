package httpapi

import (
	"context"

	"task-processor/internal/authidentity"
	"task-processor/internal/authz"
	"task-processor/internal/storecenter"
)

type storeOverviewLiveAuthorizer struct {
	app      *aiWorkbenchApplication
	expected storecenter.OverviewIdentity
}

func newStoreOverviewLiveAuthorizer(app *aiWorkbenchApplication, expected storecenter.OverviewIdentity) storecenter.LiveOverviewAuthorizer {
	return &storeOverviewLiveAuthorizer{app: app, expected: expected}
}

func (a *storeOverviewLiveAuthorizer) AuthorizeLiveOverview(ctx context.Context, org string) (storecenter.StoreMemberAccess, error) {
	if a == nil || a.app == nil || org != a.expected.OrganizationID {
		return storecenter.StoreMemberAccess{}, storecenter.ErrNotFound
	}
	identity, err := a.app.freshWorkbenchIdentity(ctx, authz.PermissionWorkbenchStoreRead)
	if err != nil || identity.UserID != a.expected.ActorID || identity.EffectiveMemberID != a.expected.MemberID || identity.EffectiveOrganizationID != org || identity.TenantID != org {
		return storecenter.StoreMemberAccess{}, storecenter.ErrNotFound
	}
	authorizer := a.app.authorizer
	if a.app.agent != nil {
		authorizer = a.app.agent.authorizer
	}
	allowed, err := authz.AuthorizeOrganization(ctx, authorizer, identity.UserID, org, identity.Roles, authz.PermissionWorkbenchChatRead)
	if err != nil || !allowed {
		return storecenter.StoreMemberAccess{}, storecenter.ErrNotFound
	}
	return (currentStoreMemberAuthorizer{authorizer: authorizer}).AuthorizeStoreMember(authidentity.WithAuthenticatedIdentity(ctx, identity), org)
}
