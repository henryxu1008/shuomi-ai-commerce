package sheinlogin

import (
	"context"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v4"
	"task-processor/internal/authidentity"
	"task-processor/internal/authruntime/zitadel"
	"task-processor/internal/authz"
	"task-processor/internal/storecenter"
	"task-processor/internal/workbenchcontext"
)

type ExactGrantReader interface {
	ReadExactServiceProjectAuthorization(context.Context, string, string, string, string) (zitadel.ExactServiceProjectAuthorization, error)
}
type Authorization struct {
	Verifier                       zitadel.Verifier
	Grants                         ExactGrantReader
	ServiceToken                   func(context.Context) (string, error)
	ProjectID, IssuerURL, ClientID string
	Permissions                    *authz.ListingKitAuthorizer
	OrganizationStatus             workbenchcontext.OrganizationBusinessStatusChecker
}

// Authenticate reuses the native verifier's provider-backed verification.
// ParseUnverified is used only AFTER that succeeds, to narrow the accepted JWT
// issuer/audience/expiry contract; it cannot authorize a signature on its own.
func (a *Authorization) Authenticate(ctx context.Context, token, organization string) (context.Context, string, string, error) {
	if a == nil || a.Verifier == nil || !validOpaque(organization) || len(token) > 65536 {
		return nil, "", "", ErrInvalidInput
	}
	identity, e := a.Verifier.Verify(ctx, token)
	if e != nil {
		return nil, "", "", ErrInvalidInput
	}
	claims := &jwt.RegisteredClaims{}
	parsed, _, e := new(jwt.Parser).ParseUnverified(token, claims)
	if e != nil || parsed == nil || parsed.Method.Alg() == "none" || claims.Issuer != strings.TrimRight(a.IssuerURL, "/") || claims.Subject != identity.UserID || !claims.VerifyAudience(a.ProjectID, true) || claims.ExpiresAt == nil || !claims.VerifyExpiresAt(time.Now(), true) || claims.Valid() != nil {
		return nil, "", "", ErrInvalidInput
	}
	grant, e := a.grant(ctx, identity.UserID, organization)
	if e != nil {
		return nil, "", "", e
	}
	identity.EffectiveOrganizationID = organization
	identity.EffectiveMemberID = grant.AuthorizationID
	identity.Roles = grant.Roles
	return authidentity.WithAuthenticatedIdentity(ctx, identity), identity.UserID, grant.AuthorizationID, nil
}
func (a *Authorization) grant(ctx context.Context, actor, org string) (zitadel.ExactServiceProjectAuthorization, error) {
	if a == nil || a.Grants == nil || a.ServiceToken == nil || a.Permissions == nil || !validOpaque(actor) || !validOpaque(org) {
		return zitadel.ExactServiceProjectAuthorization{}, ErrInvalidInput
	}
	token, e := a.ServiceToken(ctx)
	if e != nil {
		return zitadel.ExactServiceProjectAuthorization{}, ErrUnavailable
	}
	g, e := a.Grants.ReadExactServiceProjectAuthorization(ctx, token, actor, a.ProjectID, org)
	if e != nil || !g.Found || g.State != "STATE_ACTIVE" || !validOpaque(g.AuthorizationID) || !authz.AllowedOrganization(ctx, a.Permissions, actor, org, g.Roles, authz.PermissionWorkbenchStoreLifecycle) {
		return g, ErrInvalidInput
	}
	if a.OrganizationStatus != nil {
		suspended, e := a.OrganizationStatus.IsOrganizationSuspended(ctx, org)
		if e != nil || suspended {
			return g, ErrInvalidInput
		}
	}
	return g, nil
}
func (a *Authorization) AuthorizeStoreMember(ctx context.Context, org string) (storecenter.StoreMemberAccess, error) {
	id, ok := authidentity.AuthenticatedIdentityFromContext(ctx)
	if !ok || id.EffectiveOrganizationID != org || !validOpaque(id.EffectiveMemberID) {
		return storecenter.StoreMemberAccess{}, ErrInvalidInput
	}
	grant, e := a.grant(ctx, id.UserID, org)
	if e != nil || grant.AuthorizationID != id.EffectiveMemberID {
		return storecenter.StoreMemberAccess{}, ErrInvalidInput
	}
	return storecenter.StoreMemberAccess{OrganizationID: org, ActorID: id.UserID, MemberID: grant.AuthorizationID, Administrator: a.Permissions.IsTenantAdmin(id.UserID, grant.Roles), CanWrite: true}, nil
}
func executionContext(ctx context.Context, a LoginAttempt) context.Context {
	return authidentity.WithAuthenticatedIdentity(ctx, authidentity.AuthenticatedIdentity{TenantID: a.Scope.OrganizationID, UserID: a.ActorID, EffectiveOrganizationID: a.Scope.OrganizationID, EffectiveMemberID: a.MemberID})
}
