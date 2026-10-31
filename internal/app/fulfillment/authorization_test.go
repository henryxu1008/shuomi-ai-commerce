package fulfillmentapp

import (
	"context"
	"task-processor/internal/authidentity"
	"task-processor/internal/authruntime/zitadel"
	"task-processor/internal/authz"
	f "task-processor/internal/fulfillment"
	"task-processor/internal/storecenter"
	"testing"
	"time"
)

type exactGrant struct {
	grant zitadel.ExactServiceProjectAuthorization
	calls int
}

func (g *exactGrant) ReadExactServiceProjectAuthorization(_ context.Context, token, actor, project, org string) (zitadel.ExactServiceProjectAuthorization, error) {
	g.calls++
	if token != "test-token" || actor != "actor" || project != "project" || org != "org" {
		return zitadel.ExactServiceProjectAuthorization{}, f.ErrForbidden
	}
	return g.grant, nil
}

type organizationStatus bool
type fulfillmentRolePolicy struct{ modules []string }

func (p *fulfillmentRolePolicy) RoleModules(_ context.Context, _ string, roles []string) (map[string][]string, error) {
	return map[string][]string{roles[0]: p.modules}, nil
}

func (s organizationStatus) IsOrganizationSuspended(context.Context, string) (bool, error) {
	return bool(s), nil
}
func TestAuthorizationUsesCurrentExactMembershipAndSeparateAddressPurpose(t *testing.T) {
	permissions, _ := authz.NewListingKitAuthorizer(nil, nil)
	policy := &fulfillmentRolePolicy{modules: []string{"store-orders"}}
	permissions.SetRolePolicyReader(policy)
	grant := &exactGrant{grant: zitadel.ExactServiceProjectAuthorization{Found: true, AuthorizationID: "member", State: "STATE_ACTIVE", Roles: []string{authz.EnterpriseRoleKey("org", 1)}}}
	a := Authorization{Client: grant, ServiceToken: func(context.Context) (string, error) { return "test-token", nil }, ProjectID: "project", Permissions: permissions, OrganizationStatus: organizationStatus(false)}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	subject := storecenter.FulfillmentSubject{OrganizationID: "org", ActorID: "actor", MemberID: "member", Purpose: "fulfillment.read"}
	if _, err := a.AuthorizeFulfillment(ctx, subject); err != nil {
		t.Fatal(err)
	}
	subject.Purpose = "fulfillment.address"
	if _, err := a.AuthorizeFulfillment(ctx, subject); err != f.ErrForbidden {
		t.Fatal("viewer obtained address", err)
	}
	policy.modules = append(policy.modules, "fulfillment-address")
	if _, err := a.AuthorizeFulfillment(ctx, subject); err != nil {
		t.Fatal(err)
	}
	grant.grant.AuthorizationID = "replacement-member"
	if _, err := a.AuthorizeFulfillment(ctx, subject); err != f.ErrForbidden {
		t.Fatal("stale member retained authority", err)
	}
	if grant.calls != 4 {
		t.Fatal("exact grant was cached", grant.calls)
	}
	grant.grant.AuthorizationID = "member"
	ctx = authidentity.WithAuthenticatedIdentity(ctx, authidentity.AuthenticatedIdentity{TenantID: "other", EffectiveOrganizationID: "other", UserID: "actor", EffectiveMemberID: "member", TokenExpiresAt: time.Now().Add(time.Hour)})
	if _, err := a.AuthorizeFulfillment(ctx, subject); err != f.ErrForbidden {
		t.Fatal("foreign request context accepted", err)
	}
}
