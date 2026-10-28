//go:build integration

package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"

	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"task-processor/internal/app/runtime/currentapplication"
	registrySchema "task-processor/internal/app/schema/sourceaccountregistry"
	"task-processor/internal/authidentity"
	"task-processor/internal/authz"
	resourceadapter "task-processor/internal/integration/orgresource"
	kernelmodule "task-processor/internal/kernel/module"
	"task-processor/internal/ledger/orgresource"
	"task-processor/internal/storecenter"

	platformdatabase "task-processor/internal/platform/database"
)

func TestCommercialReadVerifiedPublicSchemaPostgres(t *testing.T) {
	gin.SetMode(gin.TestMode)
	owner, connection := commercialPostgres(t) // Never accepts an external DSN.
	ctx := context.Background()
	require.NoError(t, resourceadapter.AutoMigrate(owner))
	require.NoError(t, storecenter.AutoMigrateStoreRepository(owner))
	require.NoError(t, registrySchema.Migrate(ctx, owner))
	require.NoError(t, owner.Exec(`REVOKE CREATE ON SCHEMA public FROM PUBLIC`).Error)
	require.NoError(t, owner.Exec(`REVOKE ALL ON ALL TABLES IN SCHEMA public FROM PUBLIC`).Error)
	for _, role := range []string{"commercial_reader", "source_account_runtime"} {
		require.NoError(t, owner.Exec(`CREATE ROLE `+role+` LOGIN PASSWORD 'synthetic-run1-password'`).Error)
		require.NoError(t, owner.Exec(`GRANT CONNECT ON DATABASE issue347 TO `+role).Error)
		require.NoError(t, owner.Exec(`GRANT USAGE ON SCHEMA public TO `+role).Error)
		privilegeSet := run1AllowedPrivileges[role]
		if role == "commercial_reader" {
			privilegeSet = map[string][]string{"saas_organization_resource_buckets": {"SELECT"}, "saas_organization_resource_debts": {"SELECT"}, "workbench_stores": {"SELECT"}}
		}
		for table, privileges := range privilegeSet {
			require.NoError(t, owner.Exec(`GRANT `+strings.Join(privileges, ",")+` ON `+pgx.Identifier{"public", table}.Sanitize()+` TO `+role).Error)
		}
		if role == "source_account_runtime" {
			require.NoError(t, owner.Exec(`GRANT USAGE, SELECT ON SEQUENCE public.account_business_profile_audit_events_id_seq TO `+role).Error)
		}
	}
	require.NoError(t, owner.Exec(`ALTER ROLE commercial_reader SET default_transaction_read_only=on`).Error)
	now := time.Now().UTC()
	for _, seed := range []struct {
		org      string
		quantity int64
	}{{"org-B", 17}, {"org-A", 31}} {
		require.NoError(t, owner.Exec(`INSERT INTO public.saas_organization_resource_buckets(organization_id,resource_type,available,allocated,reserved,consumed,created_at,updated_at) VALUES (?, 'ai_point', ?,0,2,7,?,?)`, seed.org, seed.quantity, now, now).Error)
	}
	require.NoError(t, owner.Exec(`INSERT INTO public.saas_organization_resource_debts(organization_id,resource_type,amount,created_at,updated_at) VALUES ('org-B','ai_point',0,?,?)`, now, now).Error)
	repo, err := storecenter.NewGormStoreRepository(owner)
	require.NoError(t, err)
	for _, org := range []string{"org-B", "org-A"} {
		id := uuid.NewString()
		value, err := storecenter.NewStore(storecenter.CreateStoreInput{ID: id, OrganizationID: org, ActorSubject: "fixture", Name: "Synthetic", Platform: "shein", Region: "SG", ExternalStoreID: id, CreateIdempotencyKey: uuid.NewString(), OccurredAt: now})
		require.NoError(t, err)
		_, _, err = repo.CreateOrReplay(ctx, org, value)
		require.NoError(t, err)
	}
	tables := []string{"saas_organization_resource_buckets", "saas_organization_resource_debts", "workbench_stores"}
	var publicTables []string
	require.NoError(t, owner.Raw(`SELECT tablename FROM pg_tables WHERE schemaname='public' ORDER BY tablename`).Scan(&publicTables).Error)
	beforePublic := run1PermissionFacts(t, owner, publicTables)
	open := func() *gorm.DB {
		db, err := platformdatabase.OpenExistingReadOnlyContext(ctx, &platformdatabase.Config{Host: connection.Host, Port: connection.Port, User: "commercial_reader", Password: "synthetic-run1-password", Database: connection.Database, MaxConnections: 4, MaxIdleConnections: 4})
		require.True(t, err == nil, "runtime pool open failed")
		t.Cleanup(func() { require.NoError(t, platformdatabase.Close(db)) })
		return db
	}
	db := open()
	build := func(t *testing.T, pool *gorm.DB) http.Handler {
		t.Helper()
		module, err := buildUnifiedCommercialRead(ctx, pool, pool, currentRoleTestAuthorizer())
		require.NoError(t, err, "current read composition must not write or repair")
		registry := kernelmodule.NewRegistry()
		require.NoError(t, module.Register(registry))
		require.Len(t, registry.Routes(), 1)
		router := gin.New()
		router.GET(registry.Routes()[0].Path, registry.Routes()[0].Handler)
		return router
	}
	read := func(handler http.Handler, requestContext context.Context) *httptest.ResponseRecorder {
		identity := authidentity.AuthenticatedIdentity{UserID: "fixture", HomeOrganizationID: "home-A", TenantID: "org-B", EffectiveOrganizationID: "org-B", Roles: []string{authz.EnterpriseRoleKey("org-B", 1)}, TokenExpiresAt: time.Now().Add(time.Hour)}
		request := httptest.NewRequest(http.MethodGet, "/api/v1/workbench/commercial/overview", nil).WithContext(authidentity.WithAuthenticatedIdentity(requestContext, identity))
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		return response
	}
	assertPublic := func(t *testing.T, response *httptest.ResponseRecorder) {
		t.Helper()
		require.Equal(t, 200, response.Code, response.Body.String())
		var result struct {
			SchemaVersion string `json:"schema_version"`
			Resources     struct {
				State string               `json:"state"`
				Value orgresource.Balances `json:"value"`
			} `json:"resources"`
			Stores struct {
				State string                     `json:"state"`
				Value storecenter.ServiceSummary `json:"value"`
			} `json:"store_services"`
		}
		require.NoError(t, json.Unmarshal(response.Body.Bytes(), &result))
		require.Equal(t, "unified-base-prepaid-v1", result.SchemaVersion)
		require.Equal(t, "available", result.Resources.State)
		require.Equal(t, "available", result.Stores.State)
		require.Equal(t, "org-B", result.Resources.Value.OrganizationID)
		require.Len(t, result.Resources.Value.Resources, 3)
		require.Equal(t, "17", *result.Resources.Value.Resources[1].Available)
		require.Equal(t, "2", *result.Resources.Value.Resources[1].Reserved)
		require.Equal(t, "7", *result.Resources.Value.Resources[1].Consumed)
		require.Equal(t, int64(1), result.Stores.Value.Records)
	}
	t.Run("default_public", func(t *testing.T) { assertPublic(t, read(build(t, db), ctx)) })
	// The test owner alone installs accessible shadows. The application has no
	// DDL, grants or repair authority, and retains the default "$user", public path.
	require.NoError(t, owner.Exec(`CREATE SCHEMA commercial_reader; GRANT USAGE ON SCHEMA commercial_reader TO commercial_reader`).Error)
	changes := []string{`UPDATE commercial_reader.saas_organization_resource_buckets SET available=999,reserved=888`, `UPDATE commercial_reader.saas_organization_resource_debts SET amount=999`, `DELETE FROM commercial_reader.workbench_stores`}
	for index, table := range tables {
		t.Run("shadow/"+table, func(t *testing.T) {
			target := pgx.Identifier{"commercial_reader", table}.Sanitize()
			require.NoError(t, owner.Exec(`CREATE TABLE `+target+` (LIKE public.`+table+` INCLUDING ALL); INSERT INTO `+target+` SELECT * FROM public.`+table).Error)
			require.NoError(t, owner.Exec(changes[index]).Error)
			require.NoError(t, owner.Exec(`GRANT SELECT ON `+target+` TO commercial_reader`).Error)
			// Keep only this shadow during its subtest, so every query is tested
			// independently even when an earlier query would return wrong facts.
			defer func() { require.NoError(t, owner.Exec(`ALTER TABLE `+target+` RENAME TO `+table+`_saved`).Error) }()
			// A fresh connection must resolve the newly installed shadow; reuse
			// of a prepared query bound before CREATE would mask this regression.
			assertPublic(t, read(build(t, open()), ctx))
		})
	}
	// Rename restoration also runs on RED: failed assertions must not conceal
	// subsequent independent table probes or cleanup.
	for _, table := range tables {
		if owner.Migrator().HasTable("commercial_reader." + table + "_saved") {
			require.NoError(t, owner.Exec(`ALTER TABLE commercial_reader.`+table+`_saved RENAME TO `+table).Error)
		}
	}
	snapshot := func() string {
		var values []string
		for _, table := range tables {
			var rows []string
			require.NoError(t, owner.Raw(`SELECT row_to_json(r)::text || ':' || xmin::text FROM `+pgx.Identifier{"commercial_reader", table}.Sanitize()+` r ORDER BY 1`).Scan(&rows).Error)
			values = append(values, strings.Join(rows, "|"))
		}
		return strings.Join(values, "\n")
	}
	beforeShadow := snapshot()
	db = open()
	handler := build(t, db)
	t.Run("all_shadows", func(t *testing.T) { assertPublic(t, read(handler, ctx)) })
	t.Run("four_actual_connections", func(t *testing.T) {
		lock := owner.Begin()
		require.NoError(t, lock.Error)
		defer lock.Rollback()
		// Lock both possible resolutions so RED also proves distinct backends.
		require.NoError(t, lock.Exec(`LOCK public.saas_organization_resource_buckets, commercial_reader.saas_organization_resource_buckets IN ACCESS EXCLUSIVE MODE`).Error)
		responses := make(chan *httptest.ResponseRecorder, 4)
		for range 4 {
			go func() { responses <- read(handler, ctx) }()
		}
		require.Eventually(t, func() bool {
			var count int64
			return owner.Raw(`SELECT count(DISTINCT pid) FROM pg_stat_activity WHERE usename='commercial_reader' AND wait_event_type='Lock'`).Scan(&count).Error == nil && count == 4
		}, 5*time.Second, 20*time.Millisecond)
		require.NoError(t, lock.Rollback().Error)
		for range 4 {
			assertPublic(t, <-responses)
		}
	})
	t.Run("new_pool_and_module", func(t *testing.T) { assertPublic(t, read(build(t, open()), ctx)) })
	t.Run("recreated_connections_existing_module", func(t *testing.T) {
		pool, err := db.DB()
		require.NoError(t, err)
		closed := pool.Stats().MaxLifetimeClosed
		pool.SetConnMaxLifetime(time.Nanosecond)
		defer pool.SetConnMaxLifetime(0)
		assertPublic(t, read(handler, ctx))
		require.Greater(t, pool.Stats().MaxLifetimeClosed, closed)
	})
	for _, table := range tables {
		for _, failure := range []string{"permission", "missing"} {
			t.Run(failure+"/"+table, func(t *testing.T) {
				if failure == "permission" {
					require.NoError(t, owner.Exec(`REVOKE SELECT ON public.`+table+` FROM commercial_reader`).Error)
					defer func() { require.NoError(t, owner.Exec(`GRANT SELECT ON public.`+table+` TO commercial_reader`).Error) }()
				} else {
					require.NoError(t, owner.Exec(`ALTER TABLE public.`+table+` RENAME TO `+table+`_unavailable`).Error)
					defer func() {
						require.NoError(t, owner.Exec(`ALTER TABLE public.`+table+`_unavailable RENAME TO `+table).Error)
					}()
				}
				module, err := buildUnifiedCommercialRead(ctx, db, db, currentRoleTestAuthorizer())
				require.NoError(t, err)
				require.NotNil(t, module)
				response := read(handler, ctx)
				require.Equal(t, 200, response.Code)
				var result map[string]json.RawMessage
				require.NoError(t, json.Unmarshal(response.Body.Bytes(), &result))
				failed, healthy := "resources", "store_services"
				if table == "workbench_stores" {
					failed, healthy = healthy, failed
				}
				var unavailable struct {
					State string `json:"state"`
					Value any    `json:"value"`
				}
				require.NoError(t, json.Unmarshal(result[failed], &unavailable))
				require.Equal(t, "unavailable", unavailable.State)
				require.Nil(t, unavailable.Value)
				require.Contains(t, string(result[healthy]), `"state":"available"`)
				for _, secret := range []string{"synthetic-run1-password", "Shadow", table, "permission denied", "does not exist"} {
					require.NotContains(t, response.Body.String(), secret)
				}
			})
		}
	}
	t.Run("organization_and_auth_scope", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/workbench/commercial/overview", nil)
		out := httptest.NewRecorder()
		handler.ServeHTTP(out, req)
		require.Equal(t, 401, out.Code)
		bad := authidentity.AuthenticatedIdentity{UserID: "fixture", TenantID: "org-A", EffectiveOrganizationID: "org-B"}
		out = httptest.NewRecorder()
		handler.ServeHTTP(out, req.WithContext(authidentity.WithAuthenticatedIdentity(ctx, bad)))
		require.Equal(t, 403, out.Code)
		other := authidentity.AuthenticatedIdentity{UserID: "fixture", TenantID: "org-A", EffectiveOrganizationID: "org-A"}
		out = httptest.NewRecorder()
		handler.ServeHTTP(out, req.WithContext(authidentity.WithAuthenticatedIdentity(ctx, other)))
		require.Equal(t, 200, out.Code)
		require.Contains(t, out.Body.String(), `"available":"31"`)
		require.NotContains(t, out.Body.String(), `"available":"17"`)
	})
	t.Run("cancellation_and_timeout", func(t *testing.T) {
		assertUnavailable := func(response *httptest.ResponseRecorder) {
			require.Equal(t, 200, response.Code)
			require.NotContains(t, response.Body.String(), `"state":"available"`)
			require.Contains(t, response.Body.String(), `"value":null`)
		}
		canceled, cancel := context.WithCancel(ctx)
		cancel()
		assertUnavailable(read(handler, canceled))
		lock := owner.Begin()
		require.NoError(t, lock.Error)
		defer lock.Rollback()
		require.NoError(t, lock.Exec(`LOCK public.saas_organization_resource_buckets IN ACCESS EXCLUSIVE MODE`).Error)
		deadline, cancel := context.WithTimeout(ctx, 100*time.Millisecond)
		defer cancel()
		started := time.Now()
		response := read(handler, deadline)
		require.Equal(t, 200, response.Code)
		require.Contains(t, response.Body.String(), `"resources":{"state":"unavailable","value":null}`)
		require.Less(t, time.Since(started), 2*time.Second)
	})
	t.Run("normal_binary_restarts_with_shadows", func(t *testing.T) {
		config := currentapplication.DatabaseConfig{Host: connection.Host, Port: connection.Port, User: "source_account_runtime", Password: "synthetic-run1-password", Database: connection.Database, MaxConnections: 4}
		source := config
		source.User = "source_account_runtime"
		run1PermissionBinary(t, owner, &currentapplication.Config{SchemaVersion: 1, Listen: currentapplication.ListenConfig{Host: "127.0.0.1", Port: 18443}, Identity: currentapplication.IdentityConfig{IssuerURL: "http://127.0.0.1:18080", AuthorizationAPIURL: "http://127.0.0.1:18080", ClientID: "run1", ClientSecret: "synthetic-identity", ProjectID: "run1"}, SourceAccountDatabase: source})
	})
	require.Equal(t, beforePublic, run1PermissionFacts(t, owner, publicTables))
	require.Equal(t, beforeShadow, snapshot(), "reads or preflight changed shadow facts")
	if !t.Failed() {
		t.Log("PASS current unified handler -> Resource/Store readers: all three public tables, real pooled backends, recreated pool/module, binary restarts, fail-closed public absence, read-only/timeouts/cancel, public/shadow rows+xmin unchanged")
	}
}
