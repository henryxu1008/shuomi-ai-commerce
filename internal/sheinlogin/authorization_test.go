package sheinlogin

import (
	"context"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v4"
	"task-processor/internal/authidentity"
	"task-processor/internal/authruntime/zitadel"
	"task-processor/internal/authz"
	"task-processor/internal/storecenter"
)

type verifiedSubject struct {
	denied bool
	calls  int
}

func (v *verifiedSubject) Verify(context.Context, string) (authidentity.AuthenticatedIdentity, error) {
	v.calls++
	if v.denied {
		return authidentity.AuthenticatedIdentity{}, ErrInvalidInput
	}
	return authidentity.AuthenticatedIdentity{UserID: "actor", TenantID: "home"}, nil
}

type exactGrant struct {
	member, state, organization string
	calls                       int
	unavailable                 bool
}

func (g *exactGrant) ReadExactServiceProjectAuthorization(_ context.Context, token, actor, project, org string) (zitadel.ExactServiceProjectAuthorization, error) {
	g.calls++
	if g.unavailable {
		return zitadel.ExactServiceProjectAuthorization{}, ErrUnavailable
	}
	if token != "synthetic-service-token" || actor != "actor" || project != "project" || org != g.organization {
		return zitadel.ExactServiceProjectAuthorization{}, ErrInvalidInput
	}
	return zitadel.ExactServiceProjectAuthorization{Found: g.member != "", AuthorizationID: g.member, State: g.state, Roles: []string{"listingkit_admin"}}, nil
}

func TestReadinessRequiresLiveNativeVerificationAndAuthorizationAPI(t *testing.T) {
	v := &verifiedSubject{}
	g := &exactGrant{organization: "123"}
	missingToken := false
	a := &Authorization{Verifier: v, Grants: g, ProjectID: "project", ServiceToken: func(context.Context) (string, error) {
		if missingToken {
			return "", ErrUnavailable
		}
		return "synthetic-service-token", nil
	}}
	if !a.Ready(context.Background(), "123") || v.calls != 1 || g.calls != 1 {
		t.Fatal("healthy native dependencies were not checked")
	}
	// Readiness observes the service identity and provider availability; it does
	// not require or fabricate a business member/operator grant for that identity.
	v.denied = true
	if a.Ready(context.Background(), "123") {
		t.Fatal("revoked service token remained ready")
	}
	v.denied = false
	g.unavailable = true
	if a.Ready(context.Background(), "123") {
		t.Fatal("authorization API outage remained ready")
	}
	g.unavailable = false
	missingToken = true
	if a.Ready(context.Background(), "123") {
		t.Fatal("missing private service token remained ready")
	}
	missingToken = false
	if !a.Ready(context.Background(), "123") {
		t.Fatal("recovered dependencies remained unavailable")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if a.Ready(ctx, "123") {
		t.Fatal("cancelled readiness request succeeded")
	}
}
func TestCLIRequiresVerifiedJWTAndFreshExactMember(t *testing.T) {
	policy, e := authz.NewListingKitAuthorizer(nil, nil)
	if e != nil {
		t.Fatal(e)
	}
	v := &verifiedSubject{}
	g := &exactGrant{member: "member", state: "STATE_ACTIVE", organization: "123"}
	a := &Authorization{Verifier: v, Grants: g, ServiceToken: func(context.Context) (string, error) { return "synthetic-service-token", nil }, ProjectID: "project", IssuerURL: "https://identity.example", ClientID: "client", Permissions: policy}
	claims := jwt.RegisteredClaims{Issuer: a.IssuerURL, Subject: "actor", Audience: jwt.ClaimStrings{"project"}, ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour))}
	sign := func(c jwt.RegisteredClaims) string {
		token, e := jwt.NewWithClaims(jwt.SigningMethodHS256, c).SignedString([]byte("synthetic-provider-fixture"))
		if e != nil {
			t.Fatal(e)
		}
		return token
	}
	ctx, actor, member, e := a.Authenticate(context.Background(), sign(claims), "123")
	if e != nil || actor != "actor" || member != "member" {
		t.Fatal(actor, member, e)
	}
	if _, e = a.AuthorizeStoreMember(ctx, "123"); e != nil {
		t.Fatal(e)
	}
	g.member = "replacement"
	if _, e = a.AuthorizeStoreMember(ctx, "123"); e == nil {
		t.Fatal("stale member accepted")
	}
	g.member = "member"
	g.state = "STATE_INACTIVE"
	if _, e = a.AuthorizeStoreMember(ctx, "123"); e == nil {
		t.Fatal("revoked grant accepted")
	}
	g.state = "STATE_ACTIVE"
	for _, edit := range []func(*jwt.RegisteredClaims){func(c *jwt.RegisteredClaims) { c.Issuer = "https://wrong.example" }, func(c *jwt.RegisteredClaims) { c.Audience = jwt.ClaimStrings{"wrong"} }, func(c *jwt.RegisteredClaims) { c.ExpiresAt = jwt.NewNumericDate(time.Now().Add(-time.Minute)) }, func(c *jwt.RegisteredClaims) { c.ExpiresAt = nil }, func(c *jwt.RegisteredClaims) { c.Subject = "other" }} {
		changed := claims
		edit(&changed)
		if _, _, _, e = a.Authenticate(context.Background(), sign(changed), "123"); e == nil {
			t.Fatal("JWT boundary accepted")
		}
	}
	v.denied = true
	if _, _, _, e = a.Authenticate(context.Background(), sign(claims), "123"); e == nil {
		t.Fatal("native verification bypass")
	}
	if v.calls < 7 || g.calls < 4 {
		t.Fatal("native/live authority was not consulted")
	}
}

// This is the existing optional overlay contract, not a production policy owner.
type denyOnlyOrganizationFixture struct {
	suspended   bool
	unavailable bool
	calls       int
}

func (f *denyOnlyOrganizationFixture) IsOrganizationSuspended(_ context.Context, org string) (bool, error) {
	f.calls++
	if org != "123" || f.unavailable {
		return false, ErrUnavailable
	}
	return f.suspended, nil
}

func TestOptionalOrganizationPolicyIsDenyOnlyAndRechecked(t *testing.T) {
	policy, err := authz.NewListingKitAuthorizer(nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	g := &exactGrant{member: "member", state: "STATE_ACTIVE", organization: "123"}
	a := &Authorization{Grants: g, ServiceToken: func(context.Context) (string, error) { return "synthetic-service-token", nil }, ProjectID: "project", Permissions: policy}
	ctx := executionContext(context.Background(), LoginAttempt{ActorID: "actor", MemberID: "member", Scope: storecenter.SheinLoginScope{OrganizationID: "123"}})
	if _, err = a.AuthorizeStoreMember(ctx, "123"); err != nil {
		t.Fatal("nil overlay changed native authority", err)
	}
	f := &denyOnlyOrganizationFixture{}
	a.OrganizationStatus = f
	if _, err = a.AuthorizeStoreMember(ctx, "123"); err != nil {
		t.Fatal(err)
	}
	f.suspended = true
	if _, err = a.AuthorizeStoreMember(ctx, "123"); err == nil {
		t.Fatal("configured suspension was ignored")
	}
	f.suspended = false
	f.unavailable = true
	if _, err = a.AuthorizeStoreMember(ctx, "123"); err == nil {
		t.Fatal("configured policy outage was ignored")
	}
	f.unavailable = false
	g.state = "STATE_INACTIVE"
	if _, err = a.AuthorizeStoreMember(ctx, "123"); err == nil {
		t.Fatal("clear overlay granted revoked native access")
	}
	if f.calls != 3 {
		t.Fatal("optional overlay was cached or used as authority", f.calls)
	}
}
