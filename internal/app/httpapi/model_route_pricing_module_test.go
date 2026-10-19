package httpapi

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	d "task-processor/internal/aicapability/routeconfig"
	"task-processor/internal/authidentity"
	"task-processor/internal/authz"
	"task-processor/internal/httproute"
	qualified "task-processor/internal/integration/aicapability/routeconfig"
	"task-processor/internal/integration/openai"
	store "task-processor/internal/integration/persistence/routeconfig"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func TestModelRoutePricingHTTPPublicationUsesOriginalOwnersAndScope(t *testing.T) {
	dsn := os.Getenv("ISSUE830_TEST_DSN")
	if dsn == "" {
		t.Skip("requires task-owned isolated PG")
	}
	u, e := url.Parse(dsn)
	require.NoError(t, e)
	require.Equal(t, "127.0.0.1", u.Hostname())
	root, e := gorm.Open(postgres.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	require.NoError(t, e)
	name := "route830_" + strings.ReplaceAll(uuid.NewString(), "-", "")
	require.NoError(t, root.Exec("CREATE DATABASE "+name).Error)
	u.Path = "/" + name
	db, e := gorm.Open(postgres.Open(u.String()), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	require.NoError(t, e)
	t.Cleanup(func() {
		r, _ := db.DB()
		_ = r.Close()
		_ = root.Exec("DROP DATABASE " + name + " WITH (FORCE)").Error
		r, _ = root.DB()
		_ = r.Close()
	})
	require.NoError(t, db.Migrator().CreateTable(&openai.AIClientCredential{}))
	require.NoError(t, store.InstallSchema(context.Background(), db))
	owner := openai.NewGormCredentialResolver(db)
	require.NoError(t, owner.SaveCredential(context.Background(), openai.AIClientCredential{TenantID: "org-a", ClientName: "image-a", APIKey: "fixture-secret-never-public", BaseURL: "https://provider.example.test/v1", Model: "gpt-image-2.5", APIStyle: "grsai", TimeoutSecond: 30, Enabled: true}))
	row, e := owner.GetCredential(context.Background(), "org-a", "", "image-a")
	require.NoError(t, e)
	scope := d.Scope{OrganizationID: "org-a", Capability: d.ImageCapability}
	catalog, e := qualified.NewCatalog([]qualified.Definition{{Scope: scope, Image: &qualified.ImageAdmission{ClientName: row.ClientName, Provider: "grsai", Model: row.Model, Endpoint: row.BaseURL, CredentialVersion: openai.CredentialVersion(*row), MaximumPointsPerImage: 10}}})
	require.NoError(t, e)
	repo, e := store.New(db)
	require.NoError(t, e)
	service := &d.Service{Repository: repo, Catalog: catalog}
	auth, e := authz.NewListingKitAuthorizer([]string{"maintainer"}, nil)
	require.NoError(t, e)
	id := authidentity.AuthenticatedIdentity{UserID: "maintainer", TenantID: "org-a", EffectiveOrganizationID: "org-a", EffectiveMemberID: "member", TokenExpiresAt: time.Now().Add(time.Hour)}
	module := modelRoutePricingModule{services: ModelRoutePricingDependencies{Image: service}, authorizer: auth}
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Request = c.Request.WithContext(authidentity.WithAuthenticatedIdentity(c.Request.Context(), id))
	})
	for _, r := range module.routes() {
		router.Handle(r.Method, r.Path, r.Handler)
	}
	request := func(method, path, body string) *httptest.ResponseRecorder {
		r := httptest.NewRequest(method, modelRouteBase+path, strings.NewReader(body))
		r.Header.Set("Content-Type", "application/json")
		r.Header.Set("Idempotency-Key", "original")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, r)
		require.NotContains(t, w.Body.String(), row.APIKey)
		require.NotContains(t, w.Body.String(), row.BaseURL)
		return w
	}
	path := "/image/" + d.ImageCapability
	raw, _ := json.Marshal(d.Input{ExpectedRevision: d.NoRevision, BindingID: row.ClientName, CredentialVersion: openai.CredentialVersion(*row), Rates: d.Rates{PointsPerImage: "5"}})
	require.Equal(t, 200, request("GET", path+"/bindings", "").Code)
	require.Equal(t, 200, request("POST", path+"/validate", string(raw)).Code)
	_, e = repo.Current(context.Background(), scope)
	require.ErrorIs(t, e, d.ErrNotFound, "preview must not publish")
	first := request("POST", path+"/publish", string(raw))
	require.Equal(t, 200, first.Code)
	replay := request("POST", path+"/publish", string(raw))
	require.JSONEq(t, first.Body.String(), replay.Body.String())
	require.Equal(t, 409, request("POST", path+"/publish", strings.Replace(string(raw), `"5"`, `"6"`, 1)).Code)
	require.Equal(t, 400, request("POST", path+"/publish", strings.TrimSuffix(string(raw), "}")+`,"apiKey":"injected"}`).Code)
	require.Equal(t, 400, request("POST", path+"/publish", strings.TrimSuffix(string(raw), "}")+`,"bindingId":"duplicate"}`).Code)
	require.Equal(t, 400, request("POST", path+"/publish", strings.Repeat(" ", 16<<10)+string(raw)).Code)
	require.Equal(t, 400, request("GET", path+"/history?limit=51", "").Code)
	id.UserID = "ordinary-admin"
	id.Roles = []string{"admin"}
	require.Equal(t, 403, request("GET", path, "").Code)
	id.UserID = "maintainer"
	id.TenantID = "org-b"
	id.EffectiveOrganizationID = "org-b"
	require.Equal(t, 503, request("GET", path+"/receipts/original", "").Code)
	rows, e := repo.History(context.Background(), scope, "", 50)
	require.NoError(t, e)
	require.Len(t, rows, 1)
}

func TestModelRoutePricingRequiresInjectedPlatformAuthorityAndOrganization(t *testing.T) {
	auth, e := authz.NewListingKitAuthorizer([]string{"maintainer"}, []string{"custom_platform"})
	require.NoError(t, e)
	id := authidentity.AuthenticatedIdentity{UserID: "maintainer", TenantID: "org-a", EffectiveOrganizationID: "org-a", EffectiveMemberID: "member-a", TokenExpiresAt: time.Now().Add(time.Hour)}
	actor, org, e := modelRouteIdentity(authidentity.WithAuthenticatedIdentity(context.Background(), id), auth, true)
	require.NoError(t, e)
	require.Equal(t, "maintainer", actor)
	require.Equal(t, "org-a", org)
	for _, change := range []func(*authidentity.AuthenticatedIdentity){func(i *authidentity.AuthenticatedIdentity) { i.UserID = "org-admin"; i.Roles = []string{"admin"} }, func(i *authidentity.AuthenticatedIdentity) { i.EffectiveOrganizationID = "org-b" }, func(i *authidentity.AuthenticatedIdentity) { i.EffectiveMemberID = "" }, func(i *authidentity.AuthenticatedIdentity) { i.TokenExpiresAt = time.Now().Add(-time.Second) }} {
		bad := id
		change(&bad)
		_, _, e = modelRouteIdentity(authidentity.WithAuthenticatedIdentity(context.Background(), bad), auth, true)
		require.Error(t, e)
	}
}
func TestModelRoutePricingDescriptorsKeepLiveAuthorizationAndBounds(t *testing.T) {
	for _, r := range (modelRoutePricingModule{}).routes() {
		require.Equal(t, httproute.AuthPolicyCurrentIdentity, r.AuthPolicy)
		require.Equal(t, httproute.OrganizationAccessPolicyLiveWrite, r.OrganizationAccessPolicy)
		require.Equal(t, authz.PermissionListingKitPlatformAdm, r.Permission)
		require.Equal(t, 30*time.Second, r.RequestTimeout)
		require.NoError(t, validateModelRouteDescriptor(r))
		bad := r
		bad.OrganizationAccessPolicy = httproute.OrganizationAccessPolicyCachedRead
		require.Error(t, validateModelRouteDescriptor(bad))
	}
}
