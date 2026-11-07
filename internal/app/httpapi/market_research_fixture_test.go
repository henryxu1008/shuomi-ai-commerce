package httpapi

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	pc "task-processor/internal/aiworkbench/projectcenter"
	"task-processor/internal/authidentity"
	"task-processor/internal/authz"
	"task-processor/internal/httproute"
	"task-processor/internal/product/collection"
	"task-processor/internal/workbenchcontext"
)

type fixtureProjectStore struct {
	pc.Repository
	project pc.Project
}

func (r *fixtureProjectStore) Get(context.Context, pc.Scope, string) (pc.Project, []pc.Reference, error) {
	return r.project, nil, nil
}

type fixtureRoleModules struct {
	key     string
	modules []string
}

func (p *fixtureRoleModules) RoleModules(context.Context, string, []string) (map[string][]string, error) {
	return map[string][]string{p.key: p.modules}, nil
}

type fixtureLiveResolver struct {
	productReviewResolverStub
	calls int
}

func (r *fixtureLiveResolver) Resolve(ctx context.Context, policy httproute.OrganizationAccessPolicy, input workbenchcontext.ResolveInput) (authidentity.AuthenticatedIdentity, error) {
	r.calls++
	return r.productReviewResolverStub.Resolve(ctx, policy, input)
}

func TestResearchFixtureOriginalReadRechecksCurrentPermissions(t *testing.T) {
	scope := collection.Scope{OrganizationID: "200", ActorID: "fixture-actor", MemberID: "fixture-member"}
	projectScope := pc.Scope{OrganizationID: scope.OrganizationID, ActorID: scope.ActorID}
	projectID := uuid.NewString()
	key := authz.EnterpriseRoleKey(scope.OrganizationID, 3)
	full := []string{"projects", "supply-mine", "acquisition"}
	modules := &fixtureRoleModules{key: key, modules: full}
	policy, err := authz.NewListingKitAuthorizer(nil, nil)
	require.NoError(t, err)
	policy.SetRolePolicyReader(modules)
	id := authidentity.AuthenticatedIdentity{
		TenantID: scope.OrganizationID, EffectiveOrganizationID: scope.OrganizationID, HomeOrganizationID: "home",
		UserID: scope.ActorID, EffectiveMemberID: scope.MemberID, Roles: []string{key}, TokenExpiresAt: time.Now().Add(time.Hour),
	}
	resolver := &fixtureLiveResolver{productReviewResolverStub: productReviewResolverStub{value: id}}
	auth := &projectSources{resolver: resolver, policy: policy}
	store := &fixtureProjectStore{project: pc.Project{ID: projectID, Scope: projectScope, Fields: pc.Fields{Kind: "PRODUCT_RESEARCH"}}}
	projects := researchProjects{auth: auth, service: &pc.Service{Store: store, Authorize: auth.authorize}}
	ctx, err := (productReviewCapabilityBinder{}).Bind(authidentity.WithAuthenticatedIdentity(context.Background(), id), "Bearer fixture-only")
	require.NoError(t, err)

	require.NoError(t, projects.authorizeFixture(ctx, scope, projectID, false))
	require.Equal(t, 2, resolver.calls, "one live ProjectCenter read and one current Collection/Sourcing check")
	for _, remaining := range [][]string{{"supply-mine", "acquisition"}, {"projects", "acquisition"}, {"projects", "supply-mine"}} {
		modules.modules = remaining
		require.ErrorIs(t, projects.authorizeFixture(ctx, scope, projectID, false), collection.ErrForbidden, remaining)
	}
	modules.modules = full
	resolver.value.EffectiveMemberID = "different-member"
	require.ErrorIs(t, projects.authorizeFixture(ctx, scope, projectID, false), collection.ErrForbidden)
	resolver.value = id
	store.project.Archived = true
	require.ErrorIs(t, projects.authorizeFixture(ctx, scope, projectID, false), collection.ErrForbidden)
}
