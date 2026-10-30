//go:build integration

package httpapi

import (
	"context"
	"encoding/json"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"net/http"
	"net/http/httptest"
	"strings"
	"task-processor/internal/authidentity"
	"task-processor/internal/authz"
	k "task-processor/internal/browserkernel"
	kh "task-processor/internal/browserkernel/httpapi"
	"task-processor/internal/core/config"
	kp "task-processor/internal/integration/persistence/browserkernel"
	kernelmodule "task-processor/internal/kernel/module"
	"testing"
	"time"
)

func TestCurrentApplicationKernelCatalogPostgres(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()
	container, e := tcpostgres.Run(ctx, "postgres:17-alpine", tcpostgres.WithDatabase("browser_kernel"), tcpostgres.WithUsername("fixture"), tcpostgres.WithPassword("fixture"), tcpostgres.BasicWaitStrategies())
	require.NoError(t, e)
	t.Cleanup(func() { _ = container.Terminate(context.Background()) })
	dsn, e := container.ConnectionString(ctx, "sslmode=disable")
	require.NoError(t, e)
	open := func(d string) *gorm.DB {
		db, e := gorm.Open(postgres.Open(d), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
		require.NoError(t, e)
		raw, e := db.DB()
		require.NoError(t, e)
		t.Cleanup(func() { _ = raw.Close() })
		return db
	}
	admin := open(dsn)
	require.NoError(t, admin.Exec("CREATE ROLE browser_kernel_runtime LOGIN PASSWORD 'fixture-runtime'").Error)
	runtime := open(strings.Replace(dsn, "fixture:fixture@", "browser_kernel_runtime:fixture-runtime@", 1))
	source := &gorm.DB{}
	deps := newRouteAuthDependencies()
	deps.workbenchVerifier = mountedVerifierStub{identity: authidentity.AuthenticatedIdentity{UserID: "publisher", Roles: []string{"platform_admin"}, TokenExpiresAt: time.Now().Add(time.Hour)}}
	factories := currentApplicationFactories{buildWorkbench: func(*config.Config, *logrus.Logger) (workbenchContextBuildResult, error) {
		return workbenchContextBuildResult{module: currentApplicationTestModule{name: "base", routes: currentWorkbenchApplicationRoutes}, authDependencies: &deps}, nil
	}, buildSourceAccount: func(*gorm.DB, *authz.ListingKitAuthorizer) (kernelmodule.Module, error) { return nil, nil }, buildCommercial: func(*gorm.DB, *gorm.DB, *authz.ListingKitAuthorizer) (kernelmodule.Module, error) { return nil, nil }}
	build := func(options ...CurrentApplicationOption) (*http.Server, error) {
		return buildCurrentApplication(ctx, source, currentApplicationTestConfig(), logrus.New(), factories, options...)
	}
	option := WithBrowserKernelCatalog(runtime, []string{"https://assets.example.test", "https://cdn.example.test"})
	server, e := build()
	require.NoError(t, e)
	w := httptest.NewRecorder()
	server.Handler.ServeHTTP(w, httptest.NewRequest("GET", kh.ReadBase, nil))
	require.Equal(t, 404, w.Code)
	_, e = build(option)
	require.Error(t, e, "runtime must not install missing schema")
	var present bool
	require.NoError(t, admin.Raw("SELECT to_regnamespace('browser_kernel') IS NOT NULL").Scan(&present).Error)
	require.False(t, present)
	require.NoError(t, kp.InstallSchema(admin))
	require.NoError(t, kp.GrantRuntime(admin, "browser_kernel_runtime"))
	_, e = build(option, option)
	require.Error(t, e)
	_, e = build(WithBrowserKernelCatalog(source, []string{"https://assets.example.test"}))
	require.Error(t, e)
	server, e = build(option)
	require.NoError(t, e)
	request := func(method, path, body string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(method, path, strings.NewReader(body))
		req.Header.Set("Authorization", "Bearer fixture")
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		server.Handler.ServeHTTP(w, req)
		return w
	}
	id := uuid.NewString()
	p := k.Publish{Version: "132.0.0.1", OS: "windows", Arch: "amd64", MinClientVersion: "1.0.0", ArchiveFormat: "zip", Size: 42, SHA256: strings.Repeat("a", 64), DownloadURL: "https://assets.example.test/132.zip"}
	raw, _ := json.Marshal(p)
	w = request("PUT", kh.WriteBase+"/"+id, string(raw))
	require.Equal(t, 200, w.Code, w.Body.String())
	q := "?os=windows&arch=amd64&clientVersion=1.0.0"
	w = request("GET", kh.ReadBase+q, "")
	require.Equal(t, 200, w.Code, w.Body.String())
	require.NotContains(t, w.Body.String(), "download")
	var page k.Page
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &page))
	require.Len(t, page.Releases, 1)
	require.Equal(t, id, page.Releases[0].ReleaseID)
	w = request("GET", kh.ReadBase+"/"+id+"/artifact"+q, "")
	require.Equal(t, 200, w.Code, w.Body.String())
	var artifact k.Artifact
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &artifact))
	require.Equal(t, p.Size, artifact.Size)
	require.Equal(t, p.SHA256, artifact.SHA256)
	require.Equal(t, p.DownloadURL, artifact.DownloadURL)
	require.Len(t, artifact.DownloadOrigins, 2)
	require.NotContains(t, w.Body.String(), `"data"`)
	w = request("PUT", kh.WriteBase+"/"+id+"/recommendation", `{"recommended":true}`)
	require.Equal(t, 200, w.Code, w.Body.String())
	w = request("POST", kh.WriteBase+"/"+id+"/withdraw", "")
	require.Equal(t, 200, w.Code, w.Body.String())
	w = request("GET", kh.ReadBase+"/"+id+"/artifact"+q, "")
	require.Equal(t, 404, w.Code, w.Body.String())
	w = request("PUT", kh.WriteBase+"/"+id, string(raw))
	require.Equal(t, 200, w.Code)
	require.Contains(t, w.Body.String(), `"state":"withdrawn"`)
	w = request("PUT", kh.WriteBase+"/"+id+"/recommendation", `{"recommended":true}`)
	require.Equal(t, 409, w.Code)
	require.JSONEq(t, `{"code":"KERNEL_RELEASE_WITHDRAWN"}`, w.Body.String())
}
