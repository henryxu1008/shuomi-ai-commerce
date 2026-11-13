package httpapi

import (
	"context"
	"reflect"

	"task-processor/internal/aiworkbench"
	"task-processor/internal/authidentity"
	"task-processor/internal/authz"
	"task-processor/internal/storecenter"
)

type storeOverviewLiveAuthorizer struct {
	app      *aiWorkbenchApplication
	expected storecenter.OverviewIdentity
}

type projectedWorkbenchMessage struct {
	aiworkbench.Message
	StoreOverview            *storecenter.VisibleStoreOverview `json:"currentStoreOverview,omitempty"`
	StoreOverviewUnavailable bool                              `json:"storeOverviewUnavailable,omitempty"`
}

func (a *aiWorkbenchApplication) projectStoreMessages(ctx context.Context, scope aiworkbench.Scope, messages []aiworkbench.Message) []projectedWorkbenchMessage {
	var snapshot storecenter.VisibleStoreOverview
	hasReference := false
	for _, message := range messages {
		hasReference = hasReference || message.Kind == "STORE_OVERVIEW_REF"
	}
	available := false
	if hasReference && a.scopes.storeOverview != nil && a.scopes.storeOverviewAuthorize != nil {
		var err error
		snapshot, err = a.scopes.storeOverview(ctx, scope, storecenter.OverviewAll)
		available = err == nil && a.scopes.storeOverviewAuthorize(ctx, scope, snapshot) == nil
	}
	out := make([]projectedWorkbenchMessage, 0, len(messages))
	for _, message := range messages {
		projected := projectedWorkbenchMessage{Message: message}
		if message.Kind == "STORE_OVERVIEW_REF" {
			status := storecenter.OverviewScope(message.StoreQueryStatus)
			if !available || (status != storecenter.OverviewAll && status != storecenter.OverviewActive) {
				projected.StoreOverviewUnavailable = true
			} else {
				result := snapshot
				if status == storecenter.OverviewActive {
					result.Total = snapshot.Statuses[string(storecenter.RecordStatusActive)]
					result.Platforms = snapshot.ActivePlatforms
					result.Statuses = map[string]int64{string(storecenter.RecordStatusActive): result.Total}
					result.OperatingTypeUnknown = result.Total
				}
				projected.StoreOverview = &result
			}
		}
		out = append(out, projected)
	}
	return out
}

func (a *aiWorkbenchApplication) attachStoreOverview(catalog *storecenter.Catalog) {
	a.scopes.storeOverview = func(ctx context.Context, scope aiworkbench.Scope, status storecenter.OverviewScope) (storecenter.VisibleStoreOverview, error) {
		identity, ok := authidentity.AuthenticatedIdentityFromContext(ctx)
		if !ok || identity.TenantID != scope.OrganizationID || identity.UserID != scope.ActorID || identity.EffectiveMemberID == "" {
			return storecenter.VisibleStoreOverview{}, storecenter.ErrNotFound
		}
		expected := storecenter.OverviewIdentity{OrganizationID: scope.OrganizationID, ActorID: scope.ActorID, MemberID: identity.EffectiveMemberID}
		return catalog.VisibleOverview(ctx, expected, status, newStoreOverviewLiveAuthorizer(a, expected))
	}
	a.scopes.storeOverviewAuthorize = func(ctx context.Context, scope aiworkbench.Scope, first storecenter.VisibleStoreOverview) error {
		latest, err := a.scopes.storeOverview(ctx, scope, storecenter.OverviewAll)
		if err != nil || latest.Total != first.Total || latest.OperatingTypeUnknown != first.OperatingTypeUnknown || !reflect.DeepEqual(latest.Platforms, first.Platforms) || !reflect.DeepEqual(latest.Statuses, first.Statuses) || !reflect.DeepEqual(latest.ActivePlatforms, first.ActivePlatforms) {
			return storecenter.ErrNotFound
		}
		return nil
	}
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
