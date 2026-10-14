// Package skillauth borrows current identity, live Organization resolution and
// existing action permissions. It stores no identities, tokens or policy facts.
package skillauth

import (
	"context"
	"strings"
	"task-processor/internal/agent"
	"task-processor/internal/authidentity"
	"task-processor/internal/authz"
	"task-processor/internal/httproute"
	d "task-processor/internal/skill"
	"task-processor/internal/workbenchcontext"
	"time"
)

type OrganizationResolver interface {
	Resolve(context.Context, httproute.OrganizationAccessPolicy, workbenchcontext.ResolveInput) (authidentity.AuthenticatedIdentity, error)
}
type Authorizer struct {
	resolver OrganizationResolver
	policy   *authz.ListingKitAuthorizer
}

func New(resolver OrganizationResolver, policy *authz.ListingKitAuthorizer) (*Authorizer, error) {
	if policy == nil {
		return nil, d.ErrUnavailable
	}
	return &Authorizer{resolver: resolver, policy: policy}, nil
}

type requestKey struct{}
type personalRequestKey struct{}
type requestCapability struct {
	identity             authidentity.AuthenticatedIdentity
	bearer, organization string
}

func (a *Authorizer) Bind(ctx context.Context, header string, organization bool) (context.Context, error) {
	identity, ok := authidentity.AuthenticatedIdentityFromContext(ctx)
	if !ok || !agent.ValidID(identity.UserID) || !identity.TokenExpiresAt.After(time.Now()) {
		return ctx, d.ErrForbidden
	}
	if !organization {
		return context.WithValue(ctx, personalRequestKey{}, true), nil
	}
	parts := strings.Split(header, " ")
	if len(parts) != 2 || parts[0] != "Bearer" || parts[1] == "" || len(parts[1]) > 16384 || !agent.ValidID(identity.EffectiveOrganizationID) || identity.TenantID != identity.EffectiveOrganizationID || !agent.ValidID(identity.EffectiveMemberID) {
		return ctx, d.ErrForbidden
	}
	return context.WithValue(ctx, requestKey{}, requestCapability{identity: identity, bearer: parts[1], organization: identity.EffectiveOrganizationID}), nil
}
func (a *Authorizer) Resolve(ctx context.Context) (d.Principal, error) {
	identity, ok := authidentity.AuthenticatedIdentityFromContext(ctx)
	if a == nil || ctx.Err() != nil || !ok || !agent.ValidID(identity.UserID) || !identity.TokenExpiresAt.After(time.Now()) {
		return d.Principal{}, d.ErrForbidden
	}
	p := d.Principal{UserID: identity.UserID}
	personal, _ := ctx.Value(personalRequestKey{}).(bool)
	if personal || identity.EffectiveOrganizationID == "" {
		p.PlatformMaintainer = a.policy.Authorize(identity.UserID, identity.Roles, authz.PermissionListingKitPlatformAdm)
		return p, nil
	}
	capability, bound := ctx.Value(requestKey{}).(requestCapability)
	if !bound || a.resolver == nil || capability.identity.UserID != identity.UserID || capability.organization != identity.EffectiveOrganizationID || !capability.identity.TokenExpiresAt.Equal(identity.TokenExpiresAt) {
		return d.Principal{}, d.ErrForbidden
	}
	fresh, e := a.resolver.Resolve(ctx, httproute.OrganizationAccessPolicyLiveWrite, workbenchcontext.ResolveInput{Identity: capability.identity, BearerToken: capability.bearer, RequestedOrganizationID: capability.organization})
	if e != nil || fresh.UserID != identity.UserID || fresh.EffectiveOrganizationID != capability.organization || fresh.TenantID != capability.organization || !fresh.TokenExpiresAt.Equal(identity.TokenExpiresAt) || !agent.ValidID(fresh.EffectiveMemberID) {
		return d.Principal{}, d.ErrForbidden
	}
	p.OrganizationID, p.MemberID = fresh.EffectiveOrganizationID, fresh.EffectiveMemberID
	p.ReadOrganization = authz.AllowedOrganization(ctx, a.policy, p.UserID, p.OrganizationID, fresh.Roles, authz.PermissionWorkbenchAgentRead)
	p.UseOrganization = authz.AllowedOrganization(ctx, a.policy, p.UserID, p.OrganizationID, fresh.Roles, authz.PermissionWorkbenchAgentUse)
	p.ConfigureOrganization = authz.AllowedOrganization(ctx, a.policy, p.UserID, p.OrganizationID, fresh.Roles, authz.PermissionWorkbenchAgentConfigure)
	return p, nil
}
