package httpapi

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"task-processor/internal/aicapability"
	d "task-processor/internal/aicapability/routeconfig"
	"task-processor/internal/authidentity"
	"task-processor/internal/authz"
	"task-processor/internal/httproute"
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
	scope := d.Scope{OrganizationID: "org-a", Capability: d.SharedModelCapability}
	repo, e := store.New(db)
	require.NoError(t, e)
	service := &d.SharedModelService{Repository: repo, Catalog: httpSharedCatalogFixture{}}
	auth, e := authz.NewListingKitAuthorizer([]string{"maintainer"}, nil)
	require.NoError(t, e)
	id := authidentity.AuthenticatedIdentity{UserID: "maintainer", Roles: []string{"listingkit_admin"}, TenantID: "org-a", EffectiveOrganizationID: "org-a", EffectiveMemberID: "member", TokenExpiresAt: time.Now().Add(time.Hour)}
	now := time.Now().UTC().Truncate(time.Second)
	prices := &d.PlatformPricesService{Repository: repo, Now: func() time.Time { return now }}
	module := modelRoutePricingModule{services: ModelRoutePricingDependencies{Shared: service, Prices: prices}, authorizer: auth}
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
	path := "/enterprise"
	input := d.SharedModelInput{ExpectedRevision: d.NoRevision, Selection: d.SharedModelSelection{SchemaVersion: 1, UnitVersion: aicapability.AIPointMicrounitV1, DefaultChannel: "synthetic-channel", Text: d.ModelBindingReference{ID: "synthetic-text", Version: "v1"}, Image: d.ModelBindingReference{ID: "synthetic-image", Version: "v1"}, Multimodal: d.ModelBindingReference{ID: "synthetic-text", Version: "v1"}}}
	raw, _ := json.Marshal(input)
	require.Equal(t, 200, request("POST", path+"/validate", string(raw)).Code)
	_, e = repo.Current(context.Background(), scope)
	require.ErrorIs(t, e, d.ErrNotFound)
	first := request("POST", path+"/publish", string(raw))
	require.Equal(t, 200, first.Code, first.Body.String())
	replay := request("POST", path+"/publish", string(raw))
	require.JSONEq(t, first.Body.String(), replay.Body.String())
	require.Equal(t, 409, request("POST", path+"/publish", strings.Replace(string(raw), `"v1"`, `"v2"`, 1)).Code)
	require.Equal(t, 400, request("POST", path+"/publish", strings.TrimSuffix(string(raw), "}")+`,"pointsPerImage":"5"}`).Code)
	require.Equal(t, 400, request("POST", path+"/publish", strings.TrimSuffix(string(raw), "}")+`,"expectedRevision":"duplicate"}`).Code)
	require.Equal(t, 400, request("POST", path+"/publish", strings.Repeat(" ", 16<<10)+string(raw)).Code)
	require.Equal(t, 400, request("GET", path+"/history?limit=51", "").Code)
	id.UserID = "ordinary-admin"
	id.Roles = []string{"admin"}
	require.Equal(t, 403, request("GET", path, "").Code)
	id.UserID = "maintainer"
	id.Roles = []string{"listingkit_admin"}
	id.TenantID = "org-b"
	id.EffectiveOrganizationID = "org-b"
	require.Equal(t, 404, request("GET", path+"/receipts/original", "").Code, "the same key must not find another enterprise's receipt")
	rows, e := repo.History(context.Background(), scope, "", 50)
	require.NoError(t, e)
	require.Len(t, rows, 1)

	id.TenantID = "org-a"
	id.EffectiveOrganizationID = "org-a"
	priceInput := d.PlatformPricesInput{ExpectedRevision: d.NoRevision, Prices: []d.PublishedModelPrice{{ID: "synthetic-price", Region: "global", ServiceType: "standard-sync", Provider: "openai", Channel: "openai-direct", Model: "synthetic-mini", Currency: "USD", UsageVersion: "openai-text-v1", Rates: aicapability.OfficialModelRates{InputText: 750000, OutputText: 4500000}, SourceURL: "https://developers.openai.com/api/docs/pricing", VerifiedAt: now, EffectiveFrom: now, EffectiveUntil: now.Add(time.Hour)}}, FX: []d.PublishedFXRate{{Currency: "USD", CNYMicrosPerCurrencyUnit: 7000000, SourceURL: "https://www.chinamoney.com.cn/", RateDate: now.In(time.FixedZone("CST", 8*60*60)).Format("2006-01-02"), VerifiedAt: now, EffectiveFrom: now, EffectiveUntil: now.Add(time.Hour)}}}
	priceRaw, e := json.Marshal(priceInput)
	require.NoError(t, e)
	require.Equal(t, 200, request("POST", "/platform-prices/validate", string(priceRaw)).Code)
	_, e = repo.Current(context.Background(), d.PlatformPricesScope())
	require.ErrorIs(t, e, d.ErrNotFound, "price preview must not publish")
	priceReceipt := request("POST", "/platform-prices/publish", string(priceRaw))
	require.Equal(t, 200, priceReceipt.Code, priceReceipt.Body.String())
	require.JSONEq(t, priceReceipt.Body.String(), request("POST", "/platform-prices/publish", string(priceRaw)).Body.String())
	current, e := repo.Current(context.Background(), scope)
	require.NoError(t, e)
	require.NotNil(t, current.Payload.SharedModels, "platform price publication must not replace enterprise selections")
	require.Equal(t, 400, request("POST", "/platform-prices/publish", strings.TrimSuffix(string(priceRaw), "}")+`,"pointsPerImage":"5"}`).Code)
	id.UserID = "ordinary-admin"
	require.Equal(t, 403, request("POST", "/platform-prices/publish", string(priceRaw)).Code, "enterprise configuration permission must not confer platform price authority")
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
		permission := authz.PermissionWorkbenchAgentConfigure
		if strings.Contains(r.Path, "/platform-prices") {
			permission = authz.PermissionListingKitPlatformAdm
		}
		require.Equal(t, permission, r.Permission)
		require.Equal(t, 30*time.Second, r.RequestTimeout)
		require.NoError(t, validateModelRouteDescriptor(r))
		bad := r
		bad.OrganizationAccessPolicy = httproute.OrganizationAccessPolicyCachedRead
		require.Error(t, validateModelRouteDescriptor(bad))
	}
}

type httpSharedCatalogFixture struct{}

func (httpSharedCatalogFixture) Options(context.Context, string) ([]d.ModelOption, error) {
	return []d.ModelOption{
		{Category: d.ModelCategoryText, Reference: d.ModelBindingReference{ID: "synthetic-text", Version: "v1"}},
		{Category: d.ModelCategoryImage, Reference: d.ModelBindingReference{ID: "synthetic-image", Version: "v1"}},
		{Category: d.ModelCategoryMultimodal, Reference: d.ModelBindingReference{ID: "synthetic-text", Version: "v1"}},
	}, nil
}

func (httpSharedCatalogFixture) ValidateDefaultChannel(context.Context, string, string) error {
	return nil
}
func (httpSharedCatalogFixture) QualifyModel(_ context.Context, org string, category d.ModelCategory, ref d.ModelBindingReference) (d.ModelQualification, error) {
	tariff := aicapability.OfficialModelTariff{UnitVersion: aicapability.AIPointMicrounitV1, PriceVersion: "synthetic-price", FXVersion: "synthetic-fx", Currency: "CNY", CNYMicrosPerCurrencyUnit: 1000000, Model: ref.ID, Channel: "synthetic-channel", UsageVersion: "synthetic", Rates: aicapability.OfficialModelRates{InputText: 1000000, OutputText: 1000000}}
	q := d.ModelQualification{OrganizationID: org, Category: category, Reference: ref, Channel: tariff.Channel, Model: ref.ID, Ready: true, BoundEvidenceVersion: "synthetic-bound", ProfileVersion: "synthetic-profile", Price: tariff}
	if category == d.ModelCategoryImage {
		billing := aicapability.OfficialModelBilling{OfficialTariff: tariff, BoundEvidenceVersion: q.BoundEvidenceVersion, UsageBound: aicapability.BillableModelUsage{MappingVersion: tariff.UsageVersion, Complete: true, InputText: 100, OutputText: 100}}
		n, _ := billing.ReservationMicropoints(ref.ID)
		q.Image = &d.ImageProfile{Model: ref.ID, Points: n, PriceVersion: tariff.PriceVersion, Billing: billing}
	} else {
		q.Text = &aicapability.ModelProfile{ClientName: ref.ID, ProviderID: "synthetic", AdapterKind: "openai-compatible", ModelID: ref.ID, EndpointIdentityDigest: "endpoint", CredentialVersion: "v1", RoutingPolicyVersion: "route", AdapterPolicyVersion: "adapter", PromptVersion: "prompt", OutputSchemaVersion: "schema", UsageMappingVersion: tariff.UsageVersion, CostPricingVersion: "cost", OfficialTariff: tariff, Currency: "CNY", InputMicrosPerMillion: 1, OutputMicrosPerMillion: 1, MaximumPromptTokens: 100, MaximumCompletionTokens: 100, MaximumInputBytes: 1024, MaximumOutputBytes: 1024, DeadlineBound: time.Second}
	}
	return q, nil
}
