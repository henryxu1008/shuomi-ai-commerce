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
	"task-processor/internal/authidentity"
	"task-processor/internal/authz"
	"task-processor/internal/httproute"
)

func TestChannelCredentialRoutesRequireLiveIndependentEnterprisePermission(t *testing.T) {
	wanted := map[string]bool{
		"GET " + modelRouteBase + "/connections":                   false,
		"POST " + modelRouteBase + "/connections/:id/commands":     false,
		"GET " + modelRouteBase + "/connections/:id/receipts/:key": false,
	}
	for _, route := range (modelRoutePricingModule{}).routes() {
		key := route.Method + " " + route.Path
		if _, ok := wanted[key]; !ok {
			continue
		}
		wanted[key] = true
		require.Equal(t, "workbench.ai.credentials.manage", route.Permission)
		require.Equal(t, httproute.AuthPolicyCurrentIdentity, route.AuthPolicy)
		require.Equal(t, httproute.OrganizationAccessPolicyLiveWrite, route.OrganizationAccessPolicy)
		require.NoError(t, validateModelRouteDescriptor(route))
	}
	for route, registered := range wanted {
		require.True(t, registered, "missing the approved management path: %s", route)
	}
}

type credentialHTTPFixture struct {
	commands []credential.Command
	receipt  credential.Metadata
}

func (f *credentialHTTPFixture) Metadata(context.Context, string, string) (credential.Metadata, error) {
	return credential.Metadata{}, credential.ErrNotFound
}
func (f *credentialHTTPFixture) Receipt(_ context.Context, org, actor, key string) (credential.Metadata, error) {
	if org != "org-a" || actor != "manager" || key != "original" {
		return credential.Metadata{}, credential.ErrNotFound
	}
	return f.receipt, nil
}
func (f *credentialHTTPFixture) Publish(_ context.Context, c credential.Command) (credential.Metadata, error) {
	f.commands = append(f.commands, c)
	f.receipt = credential.Metadata{OrganizationID: c.OrganizationID, ConnectionID: c.ConnectionID, Provider: c.Provider, Channel: c.Channel, Region: c.Region, Account: c.Account, Revision: c.ExpectedRevision + 1, Enabled: c.Enabled, KeyPresent: true}
	return f.receipt, nil
}

type credentialRoleFixture struct{ enabled bool }

func (f *credentialRoleFixture) RoleModules(_ context.Context, _ string, keys []string) (map[string][]string, error) {
	out := map[string][]string{}
	if f.enabled {
		for _, key := range keys {
			out[key] = []string{"ai-credentials"}
		}
	}
	return out, nil
}

func TestChannelCredentialHTTPOnlyAcceptsScopedSecretCommandsAndSanitizedReceipts(t *testing.T) {
	store := &credentialHTTPFixture{}
	manager, e := credential.NewManagement(store, store, []credential.ConnectionDefinition{{OrganizationID: "org-a", ConnectionID: "shared", Provider: "openai", Channel: "openai-direct", Region: "global", BaseURL: "https://api.openai.com/v1"}})
	require.NoError(t, e)
	authorizer, e := authz.NewListingKitAuthorizer(nil, nil)
	require.NoError(t, e)
	policy := &credentialRoleFixture{enabled: true}
	authorizer.SetRolePolicyReader(policy)
	id := authidentity.AuthenticatedIdentity{UserID: "manager", TenantID: "org-a", EffectiveOrganizationID: "org-a", EffectiveMemberID: "member", Roles: []string{authz.EnterpriseRoleKey("org-a", 1)}, TokenExpiresAt: time.Now().Add(time.Hour)}
	module := modelRoutePricingModule{authorizer: authorizer, services: ModelRoutePricingDependencies{Credentials: manager}}
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Request = c.Request.WithContext(authidentity.WithAuthenticatedIdentity(c.Request.Context(), id))
	})
	for _, route := range module.routes() {
		router.Handle(route.Method, route.Path, route.Handler)
	}
	request := func(method, path, body string) *httptest.ResponseRecorder {
		r := httptest.NewRequest(method, modelRouteBase+path, strings.NewReader(body))
		r.Header.Set("Content-Type", "application/json")
		r.Header.Set("Idempotency-Key", "original")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, r)
		require.NotContains(t, w.Body.String(), "synthetic-secret")
		require.NotContains(t, w.Body.String(), "writerDatabase")
		require.NotContains(t, w.Body.String(), "api.openai.com")
		return w
	}
	require.Equal(t, 200, request("GET", "/connections", "").Code)
	body := `{"action":"set-key","expectedRevision":"0","account":"account","apiKey":"synthetic-secret"}`
	for _, bad := range []string{strings.TrimSuffix(body, "}") + `,"organizationId":"org-b"}`, strings.TrimSuffix(body, "}") + `,"writerDatabase":{}}`, strings.TrimSuffix(body, "}") + `,"apiKey":"different"}`, strings.Repeat(" ", 16<<10) + body} {
		require.Equal(t, 400, request("POST", "/connections/shared/commands", bad).Code)
	}
	require.Empty(t, store.commands)
	require.Equal(t, 200, request("POST", "/connections/shared/commands", body).Code)
	require.Len(t, store.commands, 1)
	require.Equal(t, "org-a", store.commands[0].OrganizationID)
	require.Equal(t, "manager", store.commands[0].ActorID)
	require.Equal(t, 200, request("GET", "/connections/shared/receipts/original", "").Code)
	require.Equal(t, 404, request("GET", "/connections/other/receipts/original", "").Code)
	id.UserID = "other"
	require.Equal(t, 404, request("GET", "/connections/shared/receipts/original", "").Code)
	id.UserID = "manager"
	policy.enabled = false
	require.Equal(t, 403, request("GET", "/connections", "").Code)
	id.Roles = []string{"listingkit_admin"}
	require.Equal(t, 403, request("POST", "/connections/shared/commands", body).Code)
	id.EffectiveOrganizationID = "org-b"
	require.Equal(t, 403, request("GET", "/connections", "").Code)
	require.Len(t, store.commands, 1)
}
