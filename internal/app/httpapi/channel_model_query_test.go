package httpapi

import (
	"context"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	credential "task-processor/internal/aicapability/channelcredential"
	d "task-processor/internal/aicapability/routeconfig"
	"task-processor/internal/authidentity"
	"task-processor/internal/authz"
	"task-processor/internal/httproute"
	qualified "task-processor/internal/integration/aicapability/routeconfig"
	"task-processor/internal/workbenchcontext"
)

type modelQueryRoleFixture []string

func (f modelQueryRoleFixture) RoleModules(_ context.Context, _ string, keys []string) (map[string][]string, error) {
	result := map[string][]string{}
	for _, key := range keys {
		result[key] = f
	}
	return result, nil
}

func TestModelQueryRouteRequiresBothPermissionsAndAnExactInput(t *testing.T) {
	a, err := authz.NewListingKitAuthorizer(nil, nil)
	require.NoError(t, err)
	m := modelRoutePricingModule{authorizer: a}
	const path = modelRouteBase + "/connections/:id/model-query"
	router := gin.New()
	id := authidentity.AuthenticatedIdentity{UserID: "user", TenantID: "org-a", EffectiveOrganizationID: "org-a", EffectiveMemberID: "member", Roles: []string{authz.EnterpriseRoleKey("org-a", 1)}, TokenExpiresAt: time.Now().Add(time.Minute)}
	router.Use(func(c *gin.Context) {
		c.Request = c.Request.WithContext(authidentity.WithAuthenticatedIdentity(c.Request.Context(), id))
	})
	found := false
	for _, r := range m.routes() {
		if r.Path != path {
			continue
		}
		found = true
		require.Equal(t, "POST", r.Method)
		require.Equal(t, httproute.OrganizationAccessPolicyLiveWrite, r.OrganizationAccessPolicy)
		require.Equal(t, "workbench.ai.connection.test", r.Permission)
		require.NoError(t, validateModelRouteDescriptor(r))
		router.Handle(r.Method, r.Path, r.Handler)
	}
	require.True(t, found)
	request := func(method, body string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(method, modelRouteBase+"/connections/openai/model-query", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		return w
	}
	body := `{"expectedMemberId":"member","expectedRevision":"3","reference":{"id":"mini","version":"v1"}}`
	for _, modules := range [][]string{nil, {"ai-credentials"}, {"ai-connection-test"}} {
		a.SetRolePolicyReader(modelQueryRoleFixture(modules))
		require.Equal(t, 403, request("POST", body).Code)
	}
	a.SetRolePolicyReader(modelQueryRoleFixture{"ai-credentials", "ai-connection-test"})
	require.Equal(t, 404, request("GET", "").Code)
	require.Equal(t, 404, request("HEAD", "").Code)
	bad := strings.Replace(body, "member\"", "other-member\"", 1)
	require.Equal(t, 409, request("POST", bad).Code)
	for _, field := range []string{"project", "apiKey", "model", "url", "tenantId"} {
		require.Equal(t, 400, request("POST", strings.TrimSuffix(body, "}")+`,"`+field+`":"forged"}`).Code)
	}
	require.Equal(t, 503, request("POST", body).Code)
}

type modelQueryHTTPFixture struct {
	before func(int)
	sends  int
}

func (f *modelQueryHTTPFixture) Target(credential.ManagedConnection) *d.ModelBindingReference {
	return nil
}
func (f *modelQueryHTTPFixture) Check(ctx context.Context, _ string, _ string, _ int64, _ d.ModelBindingReference, allow func(context.Context) bool) (qualified.ModelQueryResult, error) {
	for gate := 1; gate <= 3; gate++ {
		f.before(gate)
		if !allow(ctx) {
			return qualified.ModelQueryResult{}, qualified.ErrModelQueryForbidden
		}
		if gate == 2 {
			f.sends++
		}
	}
	return qualified.ModelQueryResult{Model: qualified.OpenAIQueryModel, Status: "found", CheckedAt: time.Now().UTC()}, nil
}

func TestMountedModelQueryRechecksGrantedRoleAndMemberBeforeSendAndReply(t *testing.T) {
	for _, tc := range []struct {
		name                     string
		changedAt                int
		dropModule, switchMember bool
		status, sends            int
	}{
		{"live success", 0, false, false, 200, 1},
		{"role revoked before send", 2, true, false, 403, 0},
		{"role revoked before reply", 3, true, false, 403, 1},
		{"member replaced before send", 2, false, true, 403, 0},
		{"member replaced before reply", 3, false, true, 403, 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			a, err := authz.NewListingKitAuthorizer(nil, nil)
			require.NoError(t, err)
			a.SetRolePolicyReader(modelQueryRoleFixture{"ai-credentials", "ai-connection-test"})
			grants := []authidentity.OrganizationGrant{{OrganizationID: "org-a", ProjectID: "project", AuthorizationID: "original", Roles: []string{authz.EnterpriseRoleKey("org-a", 1)}}}
			resolver := workbenchcontext.NewResolver(mountedGrantLoader{grants: grants}, "project", "v1", nil)
			f := &modelQueryHTTPFixture{before: func(at int) {
				if at != tc.changedAt {
					return
				}
				if tc.dropModule {
					a.SetRolePolicyReader(modelQueryRoleFixture{"ai-credentials"})
				}
				if tc.switchMember {
					grants[0].AuthorizationID = "replacement"
				}
			}}
			m := modelRoutePricingModule{authorizer: a, organizationResolver: resolver, services: ModelRoutePricingDependencies{ModelQuery: f}}
			r := gin.New()
			id := authidentity.AuthenticatedIdentity{UserID: "user", HomeOrganizationID: "org-a", TokenExpiresAt: time.Now().Add(time.Minute)}
			mountRoutesWithAuthDependencies(r, []httproute.Descriptor{m.modelQueryRoute()}, routeAuthDependencies{
				authorizer: a, workbenchVerifier: mountedVerifierStub{identity: id}, organizationResolver: resolver,
			})
			req := httptest.NewRequest("POST", modelRouteBase+"/connections/openai/model-query", strings.NewReader(`{"expectedMemberId":"original","expectedRevision":"3","reference":{"id":"mini","version":"v1"}}`))
			req.Header.Set("Authorization", "Bearer current-request-token")
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)
			require.Equal(t, tc.status, w.Code, w.Body.String())
			require.Equal(t, tc.sends, f.sends)
			require.Equal(t, tc.status == 200, strings.Contains(w.Body.String(), qualified.OpenAIQueryModel))
		})
	}
}
