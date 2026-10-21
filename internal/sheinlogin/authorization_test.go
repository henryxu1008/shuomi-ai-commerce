package sheinlogin

import (
	"context"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v4"
	"task-processor/internal/authidentity"
	"task-processor/internal/authruntime/zitadel"
	"task-processor/internal/authz"
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
}

func (g *exactGrant) ReadExactServiceProjectAuthorization(_ context.Context, token, actor, project, org string) (zitadel.ExactServiceProjectAuthorization, error) {
	g.calls++
	if token != "synthetic-service-token" || actor != "actor" || project != "project" || org != g.organization {
		return zitadel.ExactServiceProjectAuthorization{}, ErrInvalidInput
	}
	return zitadel.ExactServiceProjectAuthorization{Found: true, AuthorizationID: g.member, State: g.state, Roles: []string{"listingkit_admin"}}, nil
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
