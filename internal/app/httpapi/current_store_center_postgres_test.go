//go:build integration

package httpapi

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	storeschema "task-processor/internal/app/schema/storecenter"
	storeapp "task-processor/internal/app/storecenter"
	"task-processor/internal/authz"
	"task-processor/internal/core/config"
	"task-processor/internal/integration/shein"
	"task-processor/internal/integration/temu"
	kernelmodule "task-processor/internal/kernel/module"
	"task-processor/internal/ledger/orgresource"
	temumodel "task-processor/internal/marketplace/temu/model"
	"task-processor/internal/storecenter"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// Synthetic development checks use a newly allocated container, never an
// environment DSN or an existing account/application database.
func TestCurrentStorePostgresDelivery(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	c, err := tcpostgres.Run(ctx, "postgres:16-alpine", tcpostgres.WithDatabase("commercial"), tcpostgres.WithUsername("postgres"), tcpostgres.WithPassword("synthetic-store-test"), tcpostgres.BasicWaitStrategies())
	require.NoError(t, err)
	t.Cleanup(func() { _ = c.Terminate(context.Background()) })
	dsn, err := c.ConnectionString(ctx, "sslmode=disable")
	require.NoError(t, err)
	open := func(database, role string) *gorm.DB {
		t.Helper()
		u, err := url.Parse(dsn)
		require.NoError(t, err)
		u.Path = "/" + database
		u.User = url.UserPassword(role, "synthetic-store-test")
		db, err := gorm.Open(postgres.Open(u.String()), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
		require.NoError(t, err)
		pool, err := db.DB()
		require.NoError(t, err)
		pool.SetMaxOpenConns(1)
		t.Cleanup(func() { _ = pool.Close() })
		return db
	}
	commercial := open("commercial", "postgres")
	for _, q := range []string{
		`CREATE ROLE store_center_owner LOGIN PASSWORD 'synthetic-store-test'`,
		`CREATE ROLE store_center_runtime LOGIN PASSWORD 'synthetic-store-test'`,
		`CREATE DATABASE store_center OWNER store_center_owner`,
		`REVOKE CREATE ON SCHEMA public FROM PUBLIC`,
		`REVOKE ALL ON DATABASE commercial FROM PUBLIC`,
	} {
		require.NoError(t, commercial.Exec(q).Error)
	}
	recordsOwner := open("store_center", "store_center_owner")
	require.NoError(t, storeschema.Migrate(ctx, recordsOwner))
	require.NoError(t, storeschema.Migrate(ctx, recordsOwner))
	for _, q := range []string{`REVOKE CREATE ON SCHEMA public FROM PUBLIC`, `REVOKE ALL ON DATABASE store_center FROM PUBLIC`, `GRANT CONNECT ON DATABASE store_center TO store_center_runtime`, `GRANT USAGE ON SCHEMA public TO store_center_runtime`, `GRANT SELECT,INSERT,UPDATE ON workbench_stores TO store_center_runtime`, `GRANT SELECT,INSERT ON workbench_store_audit_logs TO store_center_runtime`, `GRANT SELECT,INSERT,UPDATE ON workbench_store_member_grants TO store_center_runtime`, `GRANT SELECT,INSERT ON workbench_store_member_grant_operations TO store_center_runtime`, `GRANT SELECT,INSERT,UPDATE ON workbench_store_service_operations TO store_center_runtime`} {
		require.NoError(t, recordsOwner.Exec(q).Error)
	}
	for _, statement := range []string{`GRANT SELECT,INSERT,UPDATE ON workbench_store_connections,workbench_store_connection_attempts TO store_center_runtime`, `GRANT SELECT,INSERT ON workbench_store_merchant_bindings TO store_center_runtime`, `GRANT SELECT,INSERT,UPDATE ON workbench_store_groups,workbench_store_profiles TO store_center_runtime`, `GRANT SELECT,INSERT ON workbench_store_metadata_operations TO store_center_runtime`} {
		require.NoError(t, recordsOwner.Exec(statement).Error)
	}
	records := open("store_center", "store_center_runtime")
	t.Run("optional TikTok persistence does not admit extra runtime access", func(t *testing.T) {
		require.NoError(t, storecenter.VerifyCurrentSchemaForCapabilities(ctx, recordsOwner, storecenter.RuntimeCapabilities{TikTokShop: true}))
		require.Error(t, storecenter.VerifyRuntimePermissionsForCapabilities(ctx, records, storecenter.RuntimeCapabilities{TikTokShop: true}), "configured TikTok requires its exact grants")
		require.NoError(t, recordsOwner.Exec(`GRANT SELECT ON workbench_store_tiktok_oauth_grants TO store_center_runtime`).Error)
		require.Error(t, storecenter.VerifyRuntimePermissions(ctx, records), "disabled TikTok still rejects extra access")
		require.NoError(t, recordsOwner.Exec(`REVOKE SELECT ON workbench_store_tiktok_oauth_grants FROM store_center_runtime`).Error)
	})
	// This fixture's Store consumer has no configured TikTok application. The
	// full installer verified every table above; now exercise an absent extension.
	require.NoError(t, recordsOwner.Exec(`DROP TABLE public.workbench_store_tiktok_oauth_attempts,public.workbench_store_tiktok_oauth_grants,public.workbench_store_tiktok_shop_bindings,public.workbench_store_tiktok_oauth_refresh_operations,public.workbench_store_tiktok_authorization_events RESTRICT`).Error)
	require.NoError(t, storecenter.VerifyCurrentSchemaForCapabilities(ctx, records, storecenter.RuntimeCapabilities{}))
	require.ErrorContains(t, storecenter.VerifyCurrentSchemaForCapabilities(ctx, recordsOwner, storecenter.RuntimeCapabilities{TikTokShop: true}), "store schema columns mismatch")
	require.ErrorContains(t, storecenter.VerifyCurrentSchema(ctx, recordsOwner), "store schema columns mismatch", "full installation verification remains strict")
	charges, err := orgresource.NewConsumerChargeService(currentStoreChargeFixture{}, map[orgresource.ResourceConsumer]orgresource.ConsumerChargeOwner{orgresource.ConsumerStoreService: currentStoreChargeFixture{}})
	require.NoError(t, err)
	module, err := buildCurrentStoreCenterModule(ctx, records, authz.DefaultListingKitAuthorizer(), charges, nil, storecenter.RuntimeCapabilities{})
	require.NoError(t, err)
	f := newAccountFixture(t)
	f.role.Store("listingkit_admin")
	cfg := &config.Config{Workbench: config.WorkbenchConfig{Enabled: true}, ListingKit: config.ListingKitConfig{Zitadel: config.ListingKitZitadelConfig{IssuerURL: f.provider.URL, ClientID: "fixture-client", ClientSecret: "fixture-secret", ProjectID: "project", AuthorizationAPIURL: f.provider.URL}}}
	auth, err := buildWorkbenchContextModule(cfg, logrus.New(), defaultWorkbenchContextFactories())
	require.NoError(t, err)
	reg := kernelmodule.NewRegistry()
	require.NoError(t, module.Register(reg))
	server := buildCurrentApplicationHTTPServer(reg.Routes(), *auth.authDependencies)
	request := func(method, path, org, body, key string, version int64) (int, map[string]any) {
		t.Helper()
		r := httptest.NewRequest(method, path, strings.NewReader(body))
		r.Header.Set("Authorization", "Bearer u1")
		r.Header.Set("X-Requested-Organization-ID", org)
		if body != "" {
			r.Header.Set("Content-Type", "application/json")
		}
		if key != "" {
			r.Header.Set("Idempotency-Key", key)
		}
		if version > 0 {
			r.Header.Set("If-Match", fmt.Sprintf("\"%d\"", version))
		}
		w := httptest.NewRecorder()
		server.Handler.ServeHTTP(w, r)
		out := map[string]any{}
		_ = json.Unmarshal(w.Body.Bytes(), &out)
		return w.Code, out
	}
	root := "/api/v1/workbench/stores"
	status, out := request("GET", root, "B", "", "", 0)
	require.Equal(t, 200, status, out)
	f.role.Store("unmapped")
	status, out = request("POST", root, "B", `{"name":"Fixture","platform":"shein","region":"US"}`, uuid.NewString(), 0)
	require.Equal(t, 403, status, out)
	f.role.Store("listingkit_admin")
	key := uuid.NewString()
	body := `{"name":"Fixture","platform":"shein","region":"US"}`
	status, out = request("POST", root, "B", body, key, 0)
	require.Equal(t, 201, status, out)
	id := out["id"].(string)
	version := int64(out["version"].(float64))
	require.Equal(t, "active", out["recordStatus"])
	version = int64(out["version"].(float64))
	require.Equal(t, "pending_activation", out["serviceStatus"])
	require.Nil(t, out["serviceExpiresAt"])
	require.Equal(t, "unavailable", out["connectionStatus"])
	require.NotContains(t, out, "lifecycleStatus")
	status, connection := request("GET", root+"/"+id+"/connection", "B", "", "", 0)
	require.Equal(t, 200, status, connection)
	require.Equal(t, "unavailable", connection["connectionStatus"])
	status, connection = request("POST", root+"/"+id+"/connection/begin", "B", `{"appId":"unconfigured-app"}`, uuid.NewString(), version)
	require.Equal(t, 503, status, connection)
	require.Equal(t, "STORE_OFFICIAL_SETUP_UNAVAILABLE", connection["code"])
	status, connection = request("POST", root+"/"+id+"/activate", "B", `{}`, uuid.NewString(), version)
	require.Equal(t, 503, status, connection)
	require.Equal(t, "STORE_CONNECTION_UNAVAILABLE", connection["code"])
	metadataKey := uuid.NewString()
	metadataBody := fmt.Sprintf(`{"action":"set_profile","name":"","groupId":null,"groupVersion":0,"stores":[{"storeId":%q,"profileVersion":0}],"responsibleMemberId":null,"lowStockThreshold":5}`, id)
	status, profile := request("POST", "/api/v1/workbench/store-metadata", "B", metadataBody, metadataKey, 0)
	require.Equal(t, 200, status, profile)
	require.Equal(t, "B", profile["organizationId"])
	require.Equal(t, "u1", profile["userId"])
	status, receipt := request("GET", "/api/v1/workbench/store-metadata/"+metadataKey, "B", "", "", 0)
	require.Equal(t, 200, status, receipt)
	require.Equal(t, true, receipt["data"].(map[string]any)["replayed"])
	status, catalog := request("GET", "/api/v1/workbench/store-catalog", "B", "", "", 0)
	require.Equal(t, 200, status, catalog)
	require.Equal(t, float64(1), catalog["data"].(map[string]any)["metrics"].(map[string]any)["unknown"])
	status, replay := request("POST", root, "B", body, key, 0)
	require.Equal(t, 201, status, replay)
	require.Equal(t, id, replay["id"])
	status, out = request("PUT", root+"/"+id, "B", `{"name":"Edited","region":"EU"}`, "", version)
	require.Equal(t, 200, status, out)
	require.Equal(t, "Edited", out["name"])
	version = int64(out["version"].(float64))

	start := time.Now().UTC().Truncate(time.Microsecond)
	expiry := start.Add(30 * 24 * time.Hour)
	require.NoError(t, recordsOwner.Table("workbench_stores").Where("id = ?", id).Updates(map[string]any{"service_status": "active", "service_started_at": start, "service_expires_at": expiry}).Error)
	status, out = request("POST", root+"/"+id+"/disable", "B", "", "", version)
	require.Equal(t, 200, status, out)
	require.Equal(t, "disabled", out["recordStatus"])
	version = int64(out["version"].(float64))
	require.Equal(t, "active", out["serviceStatus"])
	require.Equal(t, expiry.Format(time.RFC3339Nano), out["serviceExpiresAt"])
	status, out = request("POST", root+"/"+id+"/enable", "B", "", "", version)
	require.Equal(t, 200, status, out)
	require.Equal(t, "active", out["serviceStatus"])
	require.Equal(t, expiry.Format(time.RFC3339Nano), out["serviceExpiresAt"])
	status, out = request("GET", root+"/"+id, "C", "", "", 0)
	require.Equal(t, 404, status, out)
	require.Greater(t, f.grantReads.Load(), int32(5))
	// A fresh module/pool reads the durable results after reconstructing runtime.
	fresh, err := buildCurrentStoreCenterModule(ctx, open("store_center", "store_center_runtime"), authz.DefaultListingKitAuthorizer(), charges, nil, storecenter.RuntimeCapabilities{})
	require.NoError(t, err)
	reg = kernelmodule.NewRegistry()
	require.NoError(t, fresh.Register(reg))
	server = buildCurrentApplicationHTTPServer(reg.Routes(), *auth.authDependencies)
	status, out = request("GET", root+"/"+id, "B", "", "", 0)
	require.Equal(t, 200, status, out)
	require.Equal(t, "Edited", out["name"])
	version = int64(out["version"].(float64))
	require.Equal(t, "active", out["serviceStatus"])
	require.Equal(t, expiry.Format(time.RFC3339Nano), out["serviceExpiresAt"])
	f.revoked.Store(true)
	status, out = request("GET", root+"/"+id, "B", "", "", 0)
	require.Equal(t, 403, status, out)
	f.revoked.Store(false)
	f.role.Store("unmapped")
	status, out = request("DELETE", root+"/"+id, "B", "", uuid.NewString(), version)
	require.Equal(t, 403, status, out)
	f.role.Store("listingkit_admin")
	deleteKey := uuid.NewString()
	status, out = request("DELETE", root+"/"+id, "B", "", deleteKey, version)
	require.Equal(t, 200, status, out)
	status, out = request("DELETE", root+"/"+id, "B", "", deleteKey, version)
	require.Equal(t, 200, status, out)
	status, out = request("GET", root, "B", "", "", 0)
	require.Equal(t, 200, status, out)
	require.NotContains(t, out, "quota", "Native Store directory does not consume retired Store-count entitlement semantics")
	require.EqualValues(t, 0, out["pagination"].(map[string]any)["total"])
	t.Run("temu-create-replay-catalog-and-manual-connection", func(t *testing.T) {
		protection, err := temu.NewCredentialProtection("synthetic-temu-key", make([]byte, 32))
		require.NoError(t, err)
		metadataCalls := 0
		credentialExpiry := time.Now().UTC().Truncate(time.Second).Add(time.Hour)
		registry, err := storeapp.NewOfficialApplicationRegistry([]storeapp.OfficialApplicationRegistration{
			{Provider: postgresTemuConnectionProvider{partition: storecenter.PartitionCN, expiry: credentialExpiry, metadataCalls: &metadataCalls}, Protection: protection},
			{Provider: postgresTemuConnectionProvider{partition: storecenter.PartitionUS, expiry: credentialExpiry, metadataCalls: &metadataCalls}, Protection: protection},
		})
		require.NoError(t, err)
		configured, err := buildCurrentStoreCenterModule(ctx, records, authz.DefaultListingKitAuthorizer(), charges, registry, storecenter.RuntimeCapabilities{})
		require.NoError(t, err)
		routes := kernelmodule.NewRegistry()
		require.NoError(t, configured.Register(routes))
		server = buildCurrentApplicationHTTPServer(routes.Routes(), *auth.authDependencies)
		status, control := request("POST", root, "B", `{"name":"SHEIN filter control","platform":"shein","region":"US"}`, uuid.NewString(), 0)
		require.Equal(t, 201, status, control)
		for _, fixture := range []struct {
			mode      storecenter.OfficialMerchantMode
			partition storecenter.OfficialIdentityPartition
		}{
			{storecenter.MerchantFullyManaged, storecenter.PartitionCN},
			{storecenter.MerchantSemiManaged, storecenter.PartitionCN},
			{storecenter.MerchantLocal, storecenter.PartitionUS},
		} {
			t.Run(string(fixture.mode), func(t *testing.T) {
				key := uuid.NewString()
				body := fmt.Sprintf(`{"name":"TEMU %s","platform":"temu","region":"%s"}`, fixture.mode, fixture.partition)
				status, created := request("POST", root, "B", body, key, 0)
				require.Equal(t, 201, status, created)
				require.Equal(t, "temu", created["platform"])
				id := created["id"].(string)
				status, replay := request("POST", root, "B", body, key, 0)
				require.Equal(t, 201, status, replay)
				require.Equal(t, created, replay)
				status, conflict := request("POST", root, "B", strings.Replace(body, `"temu"`, `"shein"`, 1), key, 0)
				require.Equal(t, 409, status, conflict)
				status, read := request("GET", root+"/"+id, "B", "", "", 0)
				require.Equal(t, 200, status, read)
				require.Equal(t, "temu", read["platform"])
				for _, path := range []string{root + "?platform=temu", "/api/v1/workbench/store-catalog?platform=temu"} {
					status, filtered := request("GET", path, "B", "", "", 0)
					require.Equal(t, 200, status, filtered)
					if strings.Contains(path, "/store-catalog") {
						filtered = filtered["data"].(map[string]any)
					}
					items := filtered["items"].([]any)
					require.NotEmpty(t, items)
					found := false
					for _, raw := range items {
						item := raw.(map[string]any)
						require.Equal(t, "temu", item["platform"])
						require.NotEqual(t, control["id"], item["id"])
						found = found || item["id"] == id
					}
					require.True(t, found, "filtered consumer must include the newly created TEMU Store")
				}
				status, begin := request("POST", root+"/"+id+"/connection/begin", "B", fmt.Sprintf(`{"appId":"synthetic-temu-app","identityPartition":"%s"}`, fixture.partition), uuid.NewString(), int64(read["version"].(float64)))
				require.Equal(t, 200, status, begin)
				require.Equal(t, "manual_token", begin["authorizationMethod"])
				require.Equal(t, "", begin["authorizationUrl"])
				token := "synthetic-" + string(fixture.mode)
				status, connected := request("POST", root+"/"+id+"/connection/token", "B", fmt.Sprintf(`{"attemptId":"%s","accessToken":"%s"}`, begin["attemptId"], token), "", 0)
				require.Equal(t, 200, status, connected)
				require.Equal(t, "connected", connected["connectionStatus"])
				require.Equal(t, "temu", connected["platform"])
				require.Equal(t, string(fixture.mode), connected["mode"])
				require.NotEmpty(t, connected["merchantIdentityHash"])
				public, err := json.Marshal(connected)
				require.NoError(t, err)
				require.NotContains(t, string(public), token)
				status, current := request("GET", root+"/"+id, "B", "", "", 0)
				require.Equal(t, 200, status, current)
				require.Equal(t, "connected", current["connectionStatus"])
				// Exercise the actual PostgreSQL serving role and original grant,
				// with synthetic service facts in this disposable test only.
				// The HTTP creator is an administrator, so it uses the existing
				// administrator policy. A distinct synthetic member exercises the
				// explicit durable Store grant and its revocation below.
				grantMemberID := uuid.NewString()
				require.NoError(t, recordsOwner.Table("workbench_store_member_grants").Create(map[string]any{"organization_id": "B", "store_id": id, "member_id": grantMemberID, "active": true, "version": 1, "updated_by": "u1", "updated_at": time.Now().UTC()}).Error)
				repository, err := storecenter.NewMemberScopedStoreRepository(records, postgresConnectionAccess{})
				require.NoError(t, err)
				subject := storecenter.ProductExecutionSubject{OrganizationID: "B", ActorID: "u1", MemberID: grantMemberID, Purpose: storecenter.ProductPurposeRules}
				ruleAccess, err := storeapp.NewOfficialProductAccess(repository, postgresProductRuleAccess{subject}, registry)
				require.NoError(t, err)
				_, err = ruleAccess.AuthorizeTemuRules(ctx, subject, id, nil)
				require.ErrorIs(t, err, storeapp.ErrProductAccessChanged, "connection does not activate paid service")
				require.NoError(t, recordsOwner.Table("workbench_stores").Where("organization_id = ? AND id = ?", "B", id).Updates(map[string]any{"service_status": "active", "service_started_at": time.Now().UTC().Add(-time.Minute), "service_expires_at": time.Now().UTC().Add(time.Hour)}).Error)
				handle, err := ruleAccess.AuthorizeTemuRules(ctx, subject, id, nil)
				require.NoError(t, err)
				callsBefore := metadataCalls
				if fixture.partition == storecenter.PartitionCN {
					_, err = handle.SupplierCategories(ctx, temumodel.SupplierCategoryQuery{})
				} else {
					_, err = handle.LocalCategories(ctx, temumodel.LocalCategoryQuery{})
				}
				require.NoError(t, err)
				require.Equal(t, callsBefore+1, metadataCalls)
				require.NoError(t, recordsOwner.Table("workbench_store_member_grants").Where("organization_id = ? AND store_id = ? AND member_id = ?", "B", id, grantMemberID).Update("active", false).Error)
				if fixture.partition == storecenter.PartitionCN {
					_, err = handle.SupplierCategories(ctx, temumodel.SupplierCategoryQuery{})
				} else {
					_, err = handle.LocalCategories(ctx, temumodel.LocalCategoryQuery{})
				}
				require.ErrorIs(t, err, storeapp.ErrProductAccessChanged)
				require.Equal(t, callsBefore+1, metadataCalls, "revoked grant cannot read metadata with a still-live handle")
			})
		}
	})
	// Current serving credentials cannot delete audit or install persistent/temporary DDL.
	require.Error(t, records.Exec(`DELETE FROM workbench_store_audit_logs`).Error)
	require.Error(t, records.Exec(`CREATE TABLE forbidden(id int)`).Error)
	require.Error(t, records.Exec(`CREATE TEMP TABLE workbench_stores(id text)`).Error)
	require.NoError(t, recordsOwner.Exec(`GRANT UPDATE ON workbench_store_audit_logs TO store_center_runtime`).Error)
	require.Error(t, storecenter.VerifyRuntimePermissions(ctx, records))
	require.NoError(t, recordsOwner.Exec(`REVOKE UPDATE ON workbench_store_audit_logs FROM store_center_runtime`).Error)

	for _, target := range []struct {
		name, role     string
		owner, runtime *gorm.DB
		verify         func(context.Context, *gorm.DB) error
	}{
		{"records", "store_center_runtime", recordsOwner, records, storecenter.VerifyRuntimePermissions},
	} {
		t.Run("extra-schema-permissions-"+target.name, func(t *testing.T) {
			require.NoError(t, target.owner.Exec(`CREATE SCHEMA pgx_store_shadow`).Error)
			require.NoError(t, target.owner.Exec(`CREATE TABLE pgx_store_shadow.workbench_stores(id text)`).Error)
			require.NoError(t, target.owner.Exec(`GRANT USAGE ON SCHEMA pgx_store_shadow TO `+target.role).Error)
			require.NoError(t, target.owner.Exec(`GRANT SELECT ON pgx_store_shadow.workbench_stores TO `+target.role).Error)
			require.Error(t, target.verify(ctx, target.runtime), "additional schema grants must not bypass the narrow runtime contract")
			require.NoError(t, target.owner.Exec(`REVOKE SELECT ON pgx_store_shadow.workbench_stores FROM `+target.role).Error)
			require.NoError(t, target.runtime.Exec(`SET search_path TO pgx_store_shadow,public`).Error)
			require.Error(t, target.verify(ctx, target.runtime), "unqualified tables must resolve the inspected canonical schema")
			require.NoError(t, target.runtime.Exec(`RESET search_path`).Error)
			require.NoError(t, target.owner.Exec(`DROP TABLE pgx_store_shadow.workbench_stores`).Error)
			require.NoError(t, target.owner.Exec(`DROP SCHEMA pgx_store_shadow`).Error)
		})
	}
	t.Run("wrong-record-search-path", func(t *testing.T) {
		require.NoError(t, records.Exec(`SET search_path TO pg_catalog`).Error)
		defer func() { require.NoError(t, records.Exec(`RESET search_path`).Error) }()
		require.Error(t, storecenter.VerifyRuntimePermissions(ctx, records), "catalog preflight must match the unqualified repository target")
	})
	t.Run("missing-store-primary-identity", func(t *testing.T) {
		require.NoError(t, recordsOwner.Exec(`ALTER TABLE workbench_stores DROP CONSTRAINT workbench_stores_pkey`).Error)
		require.Error(t, storecenter.VerifyCurrentSchemaForCapabilities(ctx, records, storecenter.RuntimeCapabilities{}))
		require.NoError(t, recordsOwner.Exec(`ALTER TABLE workbench_stores ADD PRIMARY KEY(id)`).Error)
	})
	require.NoError(t, storecenter.VerifyCurrentSchemaForCapabilities(ctx, records, storecenter.RuntimeCapabilities{}))
	t.Run("official-credential-shape-on-narrow-postgres", func(t *testing.T) {
		repo, err := storecenter.NewMemberScopedStoreRepository(records, postgresConnectionAccess{})
		require.NoError(t, err)
		candidate, err := storecenter.NewStore(storecenter.CreateStoreInput{ID: uuid.NewString(), OrganizationID: "official-fixture", ActorSubject: "synthetic-operator", Name: "Official fixture", Platform: "shein", Region: "SG", ExternalStoreID: "metadata-only", CreateIdempotencyKey: uuid.NewString(), OccurredAt: time.Now().UTC().Add(-time.Minute)})
		require.NoError(t, err)
		candidate, _, err = repo.CreateOrReplay(ctx, "official-fixture", candidate)
		require.NoError(t, err)
		require.Equal(t, storecenter.RecordStatusActive, candidate.RecordStatus())
		protection, err := shein.NewCredentialProtection("synthetic-key", make([]byte, 32))
		require.NoError(t, err)
		registry, err := storeapp.NewOfficialApplicationRegistry([]storeapp.OfficialApplicationRegistration{{Provider: postgresConnectionProvider{}, Protection: protection, Type: storecenter.ApplicationSelfOperated}})
		require.NoError(t, err)
		app, err := storeapp.NewOfficialConnections(repo, registry)
		require.NoError(t, err)
		begin, err := app.Begin(ctx, storecenter.OfficialConnectionCommand{OrganizationID: "official-fixture", StoreID: candidate.ID(), AttemptID: uuid.NewString(), ExpectedStoreVersion: candidate.Version(), ApplicationID: "synthetic-app"})
		require.NoError(t, err)
		authorized, err := url.Parse(begin.AuthorizationURL)
		require.NoError(t, err)
		_, query, _ := strings.Cut(authorized.Fragment, "?")
		params, err := url.ParseQuery(query)
		require.NoError(t, err)
		complete := storecenter.CompleteOfficialConnection{OrganizationID: "official-fixture", StoreID: candidate.ID(), AttemptID: begin.AttemptID, AppID: "synthetic-app", State: params.Get("state"), TempToken: "synthetic-temp"}
		view, err := app.Complete(ctx, complete)
		require.NoError(t, err)
		require.Equal(t, storecenter.ConnectionStatusConnected, view.Status)
		current, err := repo.Get(ctx, "official-fixture", candidate.ID())
		require.NoError(t, err)
		status, err := app.Status(ctx, storecenter.ConnectionStatusInput{OrganizationID: "official-fixture", StoreID: current.ID(), Platform: storecenter.PlatformShein, ConnectionRef: current.ConnectionRef()})
		require.NoError(t, err)
		require.Equal(t, storecenter.ConnectionStatusConnected, status)
		_, err = app.Disconnect(ctx, storecenter.OfficialConnectionCommand{OrganizationID: "official-fixture", StoreID: current.ID(), AttemptID: uuid.NewString(), ExpectedStoreVersion: current.Version()})
		require.NoError(t, err)
		_, err = app.Complete(ctx, complete)
		require.ErrorIs(t, err, storecenter.ErrNotFound)
		require.Error(t, records.Exec(`UPDATE workbench_store_merchant_bindings SET open_key_id='other'`).Error)
		require.Error(t, records.Exec(`DELETE FROM workbench_store_connection_attempts`).Error)
	})
	t.Run("weakened-store-constraint", func(t *testing.T) {
		require.NoError(t, recordsOwner.Exec(`ALTER TABLE workbench_stores DROP CONSTRAINT store_service_period`).Error)
		require.NoError(t, recordsOwner.Exec(`ALTER TABLE workbench_stores ADD CONSTRAINT store_service_period CHECK (TRUE)`).Error)
		require.Error(t, storecenter.VerifyCurrentSchemaForCapabilities(ctx, records, storecenter.RuntimeCapabilities{}), "a constraint name alone is not a service contract")
	})

}

type postgresConnectionAccess struct{}

func (postgresConnectionAccess) AuthorizeStoreMember(_ context.Context, org string) (storecenter.StoreMemberAccess, error) {
	return storecenter.StoreMemberAccess{OrganizationID: org, ActorID: "synthetic-operator", MemberID: "synthetic-membership", CanWrite: true}, nil
}

type postgresConnectionProvider struct{}

func (postgresConnectionProvider) Application() storecenter.OfficialApplication {
	return storecenter.OfficialApplication{Platform: storecenter.PlatformShein, AuthorizationMethod: storecenter.AuthorizationCallback, AppID: "synthetic-app", Version: storeapp.BoundOfficialRevision("config-v1", storecenter.ApplicationSelfOperated), CallbackURL: "https://localhost/callback"}
}
func (postgresConnectionProvider) AuthorizationURL(state string) (string, error) {
	return "https://openapi-sem.sheincorp.com/#/empower?state=" + state, nil
}
func (postgresConnectionProvider) Exchange(context.Context, string, string) (storecenter.OfficialMerchantCredential, error) {
	return storecenter.OfficialMerchantCredential{AppID: "synthetic-app", OpenKeyID: "synthetic-open-key", SecretKey: "synthetic-merchant-secret", SupplierID: "123"}, nil
}
func (postgresConnectionProvider) QueryStore(context.Context, storecenter.OfficialMerchantCredential) (storecenter.OfficialStoreInformation, error) {
	return storecenter.OfficialStoreInformation{}, nil
}

type postgresTemuConnectionProvider struct {
	storeapp.TemuRulesProvider
	partition     storecenter.OfficialIdentityPartition
	expiry        time.Time
	metadataCalls *int
}

func (p postgresTemuConnectionProvider) Application() storecenter.OfficialApplication {
	modes := storecenter.ModesLocal
	if p.partition == storecenter.PartitionCN {
		modes = storecenter.ModesFullyManaged | storecenter.ModesSemiManaged
	}
	return storecenter.OfficialApplication{AppID: "synthetic-temu-app", Version: storeapp.BoundTemuOfficialRevision("v1", p.partition), Platform: storecenter.PlatformTemu, IdentityPartition: p.partition, AuthorizationMethod: storecenter.AuthorizationManualToken, SupportedModes: modes}
}
func (postgresTemuConnectionProvider) AuthorizationURL(string) (string, error) { return "", nil }
func (postgresTemuConnectionProvider) Exchange(context.Context, string, string) (storecenter.OfficialMerchantCredential, error) {
	panic("manual token install must not exchange a code")
}
func (p postgresTemuConnectionProvider) QueryStore(_ context.Context, credential storecenter.OfficialMerchantCredential) (storecenter.OfficialStoreInformation, error) {
	mode, namespace, region, scope := storecenter.MerchantLocal, storecenter.MerchantLocalNamespace, int64(211), "temu.local.goods.v2.add"
	if p.partition == storecenter.PartitionCN {
		mode, namespace, region, scope = storecenter.MerchantFullyManaged, storecenter.MerchantSupplierNamespace, 0, "bg.glo.goods.add"
		if credential.Temu.AccessToken == "synthetic-semi_managed" {
			mode = storecenter.MerchantSemiManaged
		}
	}
	scopes := []string{scope, "bg.local.goods.cats.get", "temu.local.goods.cats.rules.get"}
	if p.partition == storecenter.PartitionCN {
		scopes = []string{scope, "bg.goods.cats.get", "bg.goods.attrs.get"}
	}
	return storecenter.OfficialStoreInformation{Verification: &storecenter.OfficialVerificationFacts{Platform: storecenter.PlatformTemu, IdentityPartition: p.partition, Namespace: namespace, MerchantID: "synthetic-merchant-" + credential.Temu.AccessToken, Mode: mode, RegionID: region, ExpiresAt: p.expiry, Scopes: scopes, ContractRevision: "synthetic-identity-v1", VerifiedAt: time.Now().UTC()}}, nil
}

func (p postgresTemuConnectionProvider) QuerySupplierCategories(context.Context, storecenter.OfficialMerchantCredential, temumodel.SupplierCategoryQuery) ([]temumodel.SupplierCategory, error) {
	*p.metadataCalls++
	return []temumodel.SupplierCategory{{CatID: 34, Name: "Synthetic category"}}, nil
}
func (p postgresTemuConnectionProvider) QueryLocalCategories(context.Context, storecenter.OfficialMerchantCredential, temumodel.LocalCategoryQuery) ([]temumodel.LocalCategory, error) {
	*p.metadataCalls++
	return []temumodel.LocalCategory{{CatID: 34, Name: "Synthetic category"}}, nil
}

type postgresProductRuleAccess struct {
	subject storecenter.ProductExecutionSubject
}

func (a postgresProductRuleAccess) AuthorizeProductExecution(_ context.Context, s storecenter.ProductExecutionSubject) (storecenter.ProductExecutionAuthorization, error) {
	return storecenter.ProductExecutionAuthorization{Access: storecenter.StoreMemberAccess{OrganizationID: s.OrganizationID, ActorID: s.ActorID, MemberID: s.MemberID}, Allowed: s == a.subject}, nil
}
