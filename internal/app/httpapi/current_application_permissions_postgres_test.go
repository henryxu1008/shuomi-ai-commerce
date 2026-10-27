//go:build integration

package httpapi

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"task-processor/internal/app/runtime/currentapplication"
	accountallocationSchema "task-processor/internal/app/schema/accountallocation"
	registrySchema "task-processor/internal/app/schema/sourceaccountregistry"
	zitadelruntime "task-processor/internal/authruntime/zitadel"
	"task-processor/internal/core/config"
	"task-processor/internal/listingsubscription"
	platformdatabase "task-processor/internal/platform/database"
	"task-processor/internal/workbenchcontext"
)

var run1AllowedPrivileges = map[string]map[string][]string{
	"source_account_runtime": {
		"source_account_resources":              {"SELECT", "INSERT", "UPDATE"},
		"source_account_operations":             {"SELECT", "INSERT"},
		"account_user_preferences":              {"SELECT", "INSERT", "UPDATE"},
		"account_business_profiles":             {"SELECT", "INSERT", "UPDATE"},
		"account_business_profile_audit_events": {"SELECT", "INSERT"},
		"subject_verification_applications":     {"SELECT", "INSERT", "UPDATE"},
		"subject_verification_messages":         {"SELECT", "INSERT", "UPDATE"},
		"personal_verification_applications":    {"SELECT", "INSERT", "UPDATE"},
	},
	"commercial_runtime": {
		"saas_tenant_subscriptions": {"SELECT"}, "saas_plans": {"SELECT"},
		"saas_tenant_entitlements":          {"SELECT"},
		"saas_usage_buckets":                {"SELECT", "INSERT", "UPDATE"},
		"saas_usage_events":                 {"SELECT", "INSERT", "UPDATE"},
		"saas_usage_event_outbox":           {"SELECT", "INSERT", "UPDATE"},
		"saas_subscription_audit_logs":      {"SELECT", "INSERT", "UPDATE"},
		"account_member_token_locks":        {"SELECT", "INSERT", "UPDATE"},
		"account_member_token_allocations":  {"SELECT", "INSERT", "UPDATE"},
		"account_member_token_operations":   {"SELECT", "INSERT", "UPDATE"},
		"account_member_token_audit_events": {"SELECT", "INSERT", "UPDATE"},
	},
	"commercial_reader": {
		"saas_tenant_subscriptions": {"SELECT"}, "saas_plans": {"SELECT"},
		"saas_tenant_entitlements": {"SELECT"}, "saas_usage_buckets": {"SELECT"},
	},
}

func TestCurrentApplicationPermissionInventoryPostgres(t *testing.T) {
	gin.SetMode(gin.TestMode)
	owner, connection := commercialPostgres(t) // New disposable PG17, never a supplied DSN.
	ctx := context.Background()
	require.NoError(t, listingsubscription.AutoMigrateRepository(owner))
	require.NoError(t, accountallocationSchema.Migrate(ctx, owner))
	require.NoError(t, registrySchema.Migrate(ctx, owner))
	require.NoError(t, owner.Exec(`REVOKE CREATE ON SCHEMA public FROM PUBLIC`).Error)
	require.NoError(t, owner.Exec(`REVOKE ALL ON ALL TABLES IN SCHEMA public FROM PUBLIC`).Error)
	require.NoError(t, owner.Exec(`CREATE TABLE public.run1_additional_fact (id integer PRIMARY KEY, value text); INSERT INTO public.run1_additional_fact VALUES (1, 'unchanged')`).Error)
	require.NoError(t, owner.Exec(`INSERT INTO public.saas_plans (code, name, active, created_at, updated_at) VALUES ('run1-test', 'unchanged', true, now(), now())`).Error)
	require.NoError(t, owner.Exec(`INSERT INTO public.source_account_resources (organization_id,id,platform,display_name,management_status,connection_status,version,created_by,updated_by,created_at,updated_at) VALUES ('run1-org','01991e24-1009-7009-8009-000000000009','1688','unchanged','disabled','pending_connection',1,'fixture','fixture',now(),now())`).Error)
	cfg := &currentapplication.Config{SchemaVersion: 1, Listen: currentapplication.ListenConfig{Host: "127.0.0.1", Port: 18443}, Identity: currentapplication.IdentityConfig{IssuerURL: "http://127.0.0.1:18080", AuthorizationAPIURL: "http://127.0.0.1:18080", ClientID: "run1", ClientSecret: "synthetic-identity", ProjectID: "run1"}}
	for _, role := range []string{"source_account_runtime"} {
		require.NoError(t, owner.Exec(`CREATE ROLE `+role+` LOGIN PASSWORD 'synthetic-run1-password'`).Error)
		require.NoError(t, owner.Exec(`GRANT CONNECT ON DATABASE issue347 TO `+role).Error)
		require.NoError(t, owner.Exec(`GRANT USAGE ON SCHEMA public TO `+role).Error)
		for table, privileges := range run1AllowedPrivileges[role] {
			require.NoError(t, owner.Exec(`GRANT `+strings.Join(privileges, ",")+` ON TABLE `+pgx.Identifier{"public", table}.Sanitize()+` TO `+role).Error)
		}
		if role == "source_account_runtime" {
			require.NoError(t, owner.Exec(`GRANT USAGE, SELECT ON SEQUENCE public.account_business_profile_audit_events_id_seq TO `+role).Error)
		}
		dbConfig := currentapplication.DatabaseConfig{Host: "127.0.0.1", Port: connection.Port, User: role, Password: "synthetic-run1-password", Database: connection.Database, MaxConnections: 2}
		if role == "source_account_runtime" {
			cfg.SourceAccountDatabase = dbConfig
		}
	}
	open := func(c currentapplication.DatabaseConfig, readOnly bool) *gorm.DB {
		p := &platformdatabase.Config{Host: c.Host, Port: c.Port, User: c.User, Password: c.Password, Database: c.Database, MaxConnections: 2}
		var db *gorm.DB
		var err error
		if readOnly {
			db, err = platformdatabase.OpenExistingReadOnlyContext(ctx, p)
		} else {
			db, err = platformdatabase.OpenExistingWritableContext(ctx, p)
		}
		require.True(t, err == nil, "synthetic runtime pool open failed")
		t.Cleanup(func() { require.NoError(t, platformdatabase.Close(db)) })
		return db
	}
	source := open(cfg.SourceAccountDatabase, false)
	var tables, privileges []string
	require.NoError(t, owner.Raw(`SELECT c.relname FROM pg_catalog.pg_class c JOIN pg_catalog.pg_namespace n ON n.oid=c.relnamespace WHERE n.nspname='public' AND c.relkind IN ('r','p','v','m','f') ORDER BY c.relname`).Scan(&tables).Error)
	require.NoError(t, owner.Raw(`SELECT privilege_type FROM aclexplode(acldefault('r', (SELECT oid FROM pg_roles WHERE rolname=current_user))) ORDER BY privilege_type`).Scan(&privileges).Error)
	for _, must := range []string{"saas_usage_events", "saas_subscription_audit_logs", registrySchema.VersionTableName, "run1_additional_fact"} {
		require.Contains(t, tables, must)
	}
	require.Contains(t, privileges, "TRUNCATE")
	require.Contains(t, privileges, "MAINTAIN") // PG17's complete table privilege vocabulary.
	t.Logf("actual table inventory (%d): %s; privileges: %s", len(tables), strings.Join(tables, ","), strings.Join(privileges, ","))
	before := run1PermissionFacts(t, owner, tables)
	start := func(t *testing.T, rejected bool) {
		t.Helper()
		var logs bytes.Buffer
		logger := logrus.New()
		logger.SetOutput(&logs)
		listenCalls := 0
		listenReached := errors.New("test stops at listen boundary")
		err := currentapplication.Run(ctx, cfg, logger, currentapplication.Dependencies{
			IdentityPreflight: func(context.Context, currentapplication.IdentityConfig) error { return nil },
			OpenSourceAccount: func(context.Context, currentapplication.DatabaseConfig) (*gorm.DB, error) { return source, nil },
			CloseDatabase:     func(*gorm.DB) error { return nil },
			NewApplication:    NewCurrentApplication,
			Listen:            func(string, string) (net.Listener, error) { listenCalls++; return nil, listenReached },
		})
		require.NotNil(t, err)
		require.NotContains(t, err.Error()+logs.String(), "synthetic-run1-password")
		if rejected {
			require.Zero(t, listenCalls, "overprivileged/missing-privilege runtime reached listen")
			require.True(t, strings.Contains(err.Error(), "permissions do not match") || strings.Contains(err.Error(), "permission denied"), "expected permission rejection")
		} else {
			require.ErrorIs(t, err, listenReached)
			require.Equal(t, 1, listenCalls)
		}
	}
	start(t, false) // Extra table without grants and ordinary pg_catalog access are allowed.
	for _, role := range []string{"source_account_runtime"} {
		for _, table := range tables {
			for _, privilege := range privileges {
				if slices.Contains(run1AllowedPrivileges[role][table], privilege) {
					continue
				}
				t.Run(role+"/"+table+"/"+privilege, func(t *testing.T) {
					target := pgx.Identifier{"public", table}.Sanitize()
					require.NoError(t, owner.Exec("GRANT "+privilege+" ON TABLE "+target+" TO "+role).Error)
					defer func() {
						require.NoError(t, owner.Exec("REVOKE "+privilege+" ON TABLE "+target+" FROM "+role).Error)
						start(t, false)
					}()
					start(t, true)
				})
			}
		}
		for table, required := range run1AllowedPrivileges[role] {
			for _, privilege := range required {
				t.Run(role+"/missing/"+table+"/"+privilege, func(t *testing.T) {
					target := pgx.Identifier{"public", table}.Sanitize()
					require.NoError(t, owner.Exec("REVOKE "+privilege+" ON TABLE "+target+" FROM "+role).Error)
					defer func() {
						require.NoError(t, owner.Exec("GRANT "+privilege+" ON TABLE "+target+" TO "+role).Error)
						start(t, false)
					}()
					start(t, true)
				})
			}
		}
		if role == "source_account_runtime" {
			t.Run(role+"/missing/account_business_profile_audit_events_id_seq/USAGE", func(t *testing.T) {
				require.NoError(t, owner.Exec(`REVOKE USAGE ON SEQUENCE public.account_business_profile_audit_events_id_seq FROM `+role).Error)
				defer func() {
					require.NoError(t, owner.Exec(`GRANT USAGE ON SEQUENCE public.account_business_profile_audit_events_id_seq TO `+role).Error)
					start(t, false)
				}()
				start(t, true)
			})
			t.Run(role+"/forbidden/account_business_profile_audit_events_id_seq/UPDATE", func(t *testing.T) {
				require.NoError(t, owner.Exec(`GRANT UPDATE ON SEQUENCE public.account_business_profile_audit_events_id_seq TO `+role).Error)
				defer func() {
					require.NoError(t, owner.Exec(`REVOKE UPDATE ON SEQUENCE public.account_business_profile_audit_events_id_seq FROM `+role).Error)
					start(t, false)
				}()
				start(t, true)
			})
		}
		t.Run(role+"/inherited", func(t *testing.T) {
			require.NoError(t, owner.Exec(`CREATE ROLE run1_inherited; GRANT SELECT ON public.run1_additional_fact TO run1_inherited; GRANT run1_inherited TO `+role).Error)
			defer func() {
				require.NoError(t, owner.Exec(`REVOKE run1_inherited FROM `+role+`; REVOKE SELECT ON public.run1_additional_fact FROM run1_inherited; DROP ROLE run1_inherited`).Error)
				start(t, false)
			}()
			start(t, true)
		})
	}
	t.Run("PUBLIC/source_account_runtime", func(t *testing.T) {
		require.NoError(t, owner.Exec(`GRANT SELECT ON public.run1_additional_fact TO PUBLIC`).Error)
		defer func() {
			require.NoError(t, owner.Exec(`REVOKE SELECT ON public.run1_additional_fact FROM PUBLIC`).Error)
			start(t, false)
		}()
		// The current startup opens the source owner; PUBLIC grants must be rejected.
		start(t, true)
	})
	t.Run("column_acl", func(t *testing.T) {
		run1ColumnPermissionMatrix(t, owner, source, tables, start)
	})
	require.Equal(t, before, run1PermissionFacts(t, owner, tables), "preflight/GRANT/REVOKE changed business or migration facts")
	t.Run("normal_binary", func(t *testing.T) { run1PermissionBinary(t, owner, cfg) })
	require.Equal(t, before, run1PermissionFacts(t, owner, tables))
	t.Log("permission checks finished; fact snapshot unchanged; TRUNCATE was granted/revoked only")
	t.Run("authenticated_source_accounts", func(t *testing.T) {
		run1AuthenticatedSourceAccounts(t, owner, source, tables)
	})
}

func run1AuthenticatedSourceAccounts(t *testing.T, owner, source *gorm.DB, tables []string) {
	t.Helper()
	ctx := context.Background()
	identity := &sourceAccountSubjectFixture{sourceAccountAuthFixture: newSourceAccountAuthFixture()}
	cfg := currentApplicationTestConfig()
	cfg.ListingKit.PlatformAdminUsers = []string{"actor-1"}
	// Even an unexpected identity request can only reach this package's synthetic
	// loopback provider. Token verification and grant loading use the existing SA1
	// fixture; current middleware, factories, runtime role and repositories are real.
	cfg.ListingKit.Zitadel.IssuerURL = appHTTPTestConfig.ListingKit.Zitadel.IssuerURL
	cfg.ListingKit.Zitadel.AuthorizationAPIURL = cfg.ListingKit.Zitadel.IssuerURL
	authFactories := defaultWorkbenchContextFactories()
	authFactories.newVerifier = func(zitadelruntime.Config) zitadelruntime.Verifier { return identity }
	authFactories.newResolver = func(_ workbenchcontext.GrantLoader, project, contract string, _ workbenchcontext.OrganizationBusinessStatusChecker) *workbenchcontext.Resolver {
		return workbenchcontext.NewResolver(identity, project, contract, identity)
	}
	factories := defaultCurrentApplicationFactories(ctx, cfg.ListingKit.Zitadel.ProjectID)
	factories.buildWorkbench = func(cfg *config.Config, logger *logrus.Logger) (workbenchContextBuildResult, error) {
		return buildWorkbenchContextModule(cfg, logger, authFactories)
	}
	schemaBefore := sourceAccountApplicationTables(t, owner)
	start := func() (string, func()) {
		var logs bytes.Buffer
		logger := logrus.New()
		logger.SetOutput(&logs)
		application, err := buildCurrentApplication(ctx, source, cfg, logger, factories)
		require.NoError(t, err)
		require.Equal(t, schemaBefore, sourceAccountApplicationTables(t, owner), "construction changed schema")
		return serveSourceAccountApplication(t, application)
	}
	baseURL, stop := start()
	const path = "/api/v1/workbench/source-accounts"
	const key = "0198c5c0-6666-7666-8666-666666666666"
	const input = `{"displayName":"Current application","platform":"1688"}`
	status, body := sourceAccountRequest(t, baseURL, "token-admin", "org-b", http.MethodPost, path, key, "", input)
	require.Equal(t, http.StatusCreated, status, string(body))
	var created struct {
		Account struct {
			ID               string `json:"id"`
			ConnectionStatus string `json:"connectionStatus"`
		} `json:"account"`
	}
	require.NoError(t, json.Unmarshal(body, &created))
	require.NotEmpty(t, created.Account.ID)
	require.Equal(t, "pending_connection", created.Account.ConnectionStatus)
	var persisted struct{ OrganizationID, CreatedBy string }
	require.NoError(t, owner.Raw(`SELECT organization_id, created_by FROM public.source_account_resources WHERE id = ?`, created.Account.ID).Scan(&persisted).Error)
	require.Equal(t, "org-b", persisted.OrganizationID, "owner came from an untrusted header or Home Organization")
	require.Equal(t, "actor-1", persisted.CreatedBy)
	item := path + "/" + created.Account.ID
	for _, readPath := range []string{path, item} {
		status, body = sourceAccountRequest(t, baseURL, "token-admin", "org-b", http.MethodGet, readPath, "", "", "")
		require.Equal(t, http.StatusOK, status, string(body))
		require.Contains(t, string(body), created.Account.ID)
	}
	stop()
	baseURL, stop = start()
	defer stop()
	before := run1PermissionFacts(t, owner, tables)
	status, body = sourceAccountRequest(t, baseURL, "token-admin", "org-b", http.MethodGet, item, "", "", "")
	require.Equal(t, http.StatusOK, status, string(body))
	require.Contains(t, string(body), created.Account.ID, "reconstructed current application lost the persisted record")
	for _, test := range []struct {
		name, token, org, method, target, code string
		status                                 int
	}{
		{"other admitted organization", "token-admin", "org-a", http.MethodGet, item, "SOURCE_ACCOUNT_NOT_FOUND", http.StatusNotFound},
		{"missing organization grant", "token-other", "org-b", http.MethodGet, item, "ORGANIZATION_ACCESS_DENIED", http.StatusForbidden},
		{"missing manage permission", "token-viewer", "org-b", http.MethodPost, path, "PERMISSION_DENIED", http.StatusForbidden},
	} {
		t.Run(test.name, func(t *testing.T) {
			requestKey, payload := "", ""
			if test.method == http.MethodPost {
				requestKey, payload = key, input
			}
			status, body := sourceAccountRequest(t, baseURL, test.token, test.org, test.method, test.target, requestKey, "", payload)
			assertHTTPCode(t, status, body, test.status, test.code)
			require.Equal(t, before, run1PermissionFacts(t, owner, tables), "denied request changed facts")
		})
	}
	identity.setRevoked(true)
	status, body = sourceAccountRequest(t, baseURL, "token-admin", "org-b", http.MethodPost, path, key, "", input)
	assertHTTPCode(t, status, body, http.StatusForbidden, "ORGANIZATION_ACCESS_DENIED")
	identity.setRevoked(false)
	identity.setUnavailable(true)
	status, body = sourceAccountRequest(t, baseURL, "token-admin", "org-b", http.MethodPost, path, key, "", input)
	assertHTTPCode(t, status, body, http.StatusServiceUnavailable, "DEPENDENCY_UNAVAILABLE")
	require.Equal(t, before, run1PermissionFacts(t, owner, tables), "reads, revoked replay or dependency failure changed facts")
	require.Equal(t, schemaBefore, sourceAccountApplicationTables(t, owner))
}

func run1PermissionFacts(t *testing.T, db *gorm.DB, tables []string) string {
	t.Helper()
	var facts []string
	for _, table := range tables {
		var rows []string
		require.NoError(t, db.Raw(`SELECT row_to_json(r)::text || ':' || xmin::text FROM `+pgx.Identifier{"public", table}.Sanitize()+` r`).Scan(&rows).Error)
		slices.Sort(rows)
		facts = append(facts, table+":"+strings.Join(rows, "|"))
	}
	return fmt.Sprintf("%x", sha256.Sum256([]byte(strings.Join(facts, "\n"))))
}

// This narrow binary check uses only a same-origin discovery fixture. It does
// not claim to repeat official login/session acceptance or access any provider.
func run1PermissionBinary(t *testing.T, owner *gorm.DB, cfg *currentapplication.Config) {
	t.Helper()
	var origin string
	provider := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/.well-known/openid-configuration" {
			http.NotFound(w, r)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]string{"issuer": origin, "authorization_endpoint": origin + "/authorize", "token_endpoint": origin + "/token", "userinfo_endpoint": origin + "/userinfo", "introspection_endpoint": origin + "/introspect"})
	}))
	defer provider.Close()
	origin = provider.URL
	copyConfig := *cfg
	copyConfig.Identity.IssuerURL = origin
	copyConfig.Identity.AuthorizationAPIURL = origin
	dir := t.TempDir()
	binary := buildCurrentApplicationTestBinary(t)
	run := func(rejected bool) {
		t.Helper()
		listener, err := net.Listen("tcp", "127.0.0.1:0")
		require.NoError(t, err)
		copyConfig.Listen.Port = listener.Addr().(*net.TCPAddr).Port
		require.NoError(t, listener.Close())
		data, err := json.Marshal(copyConfig)
		require.NoError(t, err)
		// Reuse the private fixture writer; Windows does not enforce os.FileMode.
		manifest := writeSMSPrivateConfig(t, string(data))
		shutdown := filepath.Join(dir, fmt.Sprintf("shutdown-%d", copyConfig.Listen.Port))
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()
		command := exec.CommandContext(ctx, binary, "-config", manifest, "-shutdown-file", shutdown)
		var logs bytes.Buffer
		command.Stdout = &logs
		command.Stderr = &logs
		require.NoError(t, command.Start())
		done := make(chan error, 1)
		go func() { done <- command.Wait() }()
		if rejected {
			err = <-done
			require.Error(t, err)
			require.NoError(t, ctx.Err(), "binary did not promptly reject")
			require.Contains(t, logs.String(), "permissions do not match")
		} else {
			client := &http.Client{Timeout: time.Second}
			require.Eventually(t, func() bool {
				response, e := client.Get("http://" + copyConfig.ListenAddress() + "/api/v1/workbench/source-accounts")
				if e != nil {
					return false
				}
				_ = response.Body.Close()
				return response.StatusCode == 401
			}, 10*time.Second, 50*time.Millisecond)
			require.NoError(t, os.WriteFile(shutdown, []byte("stop"), 0600))
			require.NoError(t, <-done)
		}
		require.NotContains(t, logs.String(), "synthetic-run1-password")
		connection, e := net.DialTimeout("tcp", copyConfig.ListenAddress(), 100*time.Millisecond)
		if connection != nil {
			_ = connection.Close()
		}
		require.Error(t, e, "application listener remains after exit")
	}
	run(false)
	for _, role := range []string{"source_account_runtime"} {
		require.NoError(t, owner.Exec(`GRANT UPDATE ON public.goose_source_account_registry_version TO `+role).Error)
		run(true)
		require.NoError(t, owner.Exec(`REVOKE UPDATE ON public.goose_source_account_registry_version FROM `+role).Error)
		run(false)
	}
	t.Log("PASS normal binary: minimal roles listen; each role with migration UPDATE rejected; REVOKE restores listen; all processes/listeners stopped")
}

func buildCurrentApplicationTestBinary(t *testing.T) string {
	t.Helper()
	binary := filepath.Join(t.TempDir(), "current-application.exe")
	build := exec.Command("go", "build", "-o", binary, "./cmd/current-application")
	build.Dir = filepath.Join("..", "..", "..")
	output, err := build.CombinedOutput()
	require.NoError(t, err, string(output))
	return binary
}
