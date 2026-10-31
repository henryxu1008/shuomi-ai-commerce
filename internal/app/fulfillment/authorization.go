package fulfillmentapp

import (
	"context"
	"task-processor/internal/authidentity"
	"task-processor/internal/authruntime/zitadel"
	"task-processor/internal/authz"
	f "task-processor/internal/fulfillment"
	"task-processor/internal/storecenter"
	"task-processor/internal/workbenchcontext"
	"time"
)

type ExactIAM interface {
	ReadExactServiceProjectAuthorization(context.Context, string, string, string, string) (zitadel.ExactServiceProjectAuthorization, error)
}
type Authorization struct {
	Client             ExactIAM
	ServiceToken       func(context.Context) (string, error)
	ProjectID          string
	Permissions        *authz.ListingKitAuthorizer
	OrganizationStatus workbenchcontext.OrganizationBusinessStatusChecker
}

func (a Authorization) AuthorizeFulfillment(ctx context.Context, s storecenter.FulfillmentSubject) (storecenter.ProductExecutionAuthorization, error) {
	empty := storecenter.ProductExecutionAuthorization{}
	if ctx == nil || ctx.Err() != nil || !storecenter.ValidFulfillmentPurpose(s.Purpose) || !authidentity.IsBoundedIdentifier(s.OrganizationID) || !authidentity.IsBoundedIdentifier(s.ActorID) || !authidentity.IsBoundedIdentifier(s.MemberID) {
		return empty, f.ErrForbidden
	}
	if _, ok := ctx.Deadline(); !ok {
		return empty, f.ErrForbidden
	}
	if a.Client == nil || a.ServiceToken == nil || a.Permissions == nil || a.ProjectID == "" || a.OrganizationStatus == nil {
		return empty, f.ErrUnavailable
	}
	if identity, ok := authidentity.AuthenticatedIdentityFromContext(ctx); ok && (identity.TenantID != s.OrganizationID || identity.EffectiveOrganizationID != s.OrganizationID || identity.UserID != s.ActorID || identity.EffectiveMemberID != s.MemberID || !time.Now().Before(identity.TokenExpiresAt)) {
		return empty, f.ErrForbidden
	}
	token, err := a.ServiceToken(ctx)
	if err != nil {
		return empty, f.ErrUnavailable
	}
	grant, err := a.Client.ReadExactServiceProjectAuthorization(ctx, token, s.ActorID, a.ProjectID, s.OrganizationID)
	if err != nil {
		return empty, f.ErrUnavailable
	}
	if !grant.Found || grant.State != "STATE_ACTIVE" || grant.AuthorizationID != s.MemberID {
		return empty, f.ErrForbidden
	}
	suspended, err := a.OrganizationStatus.IsOrganizationSuspended(ctx, s.OrganizationID)
	if err != nil {
		return empty, f.ErrUnavailable
	}
	if suspended {
		return empty, f.ErrForbidden
	}
	for _, permission := range []string{authz.PermissionWorkbenchStoreRead, "workbench." + s.Purpose} {
		allowed, err := authz.AuthorizeOrganization(ctx, a.Permissions, s.ActorID, s.OrganizationID, grant.Roles, permission)
		if err != nil {
			return empty, f.ErrUnavailable
		}
		if !allowed {
			return empty, f.ErrForbidden
		}
	}
	return storecenter.ProductExecutionAuthorization{Allowed: true, Access: storecenter.StoreMemberAccess{OrganizationID: s.OrganizationID, ActorID: s.ActorID, MemberID: s.MemberID, Administrator: a.Permissions.IsTenantAdmin("", grant.Roles)}}, nil
}
