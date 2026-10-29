package httpapi

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"task-processor/internal/authz"
	tm "task-processor/internal/toolmarket"
)

type pluginDownloadRepo struct {
	repo
	enabled bool
}

func (r *pluginDownloadRepo) Activations(context.Context, tm.Scope) ([]tm.Activation, error) {
	return []tm.Activation{{ToolID: tm.AcquisitionID, Enabled: r.enabled}}, nil
}

func TestDeploymentBuildPackageDownload(t *testing.T) {
	directory := os.Getenv("TOOLMARKET_PACKAGE_TEST_DIR")
	if directory == "" {
		t.Skip("requires actual existing extension release build")
	}
	raw, err := os.ReadFile(filepath.Join(directory, "release.json"))
	require.NoError(t, err)
	var record struct{ CaptureAppURL, SHA256, Filename string }
	require.NoError(t, json.Unmarshal(raw, &record))
	cfg := tm.PackageConfig{Path: filepath.Join(directory, record.Filename), SHA256: record.SHA256, CaptureAppURL: record.CaptureAppURL}
	archive, err := os.ReadFile(cfg.Path)
	require.NoError(t, err)
	allowed := true
	db := &pluginDownloadRepo{enabled: true}
	h := &Handler{Repository: db, Readiness: tm.Readiness{LocalCapture: true},
		Authorize: func(context.Context, string, bool) (tm.Scope, error) {
			t.Fatal("download must use the existing read authorization")
			return tm.Scope{}, tm.ErrForbidden
		},
		ReadAuthorize: func(_ context.Context, permission string, platform bool) (tm.Scope, error) {
			require.Equal(t, authz.PermissionWorkbenchToolsRead, permission)
			require.False(t, platform)
			if !allowed {
				return tm.Scope{}, tm.ErrForbidden
			}
			return tm.Scope{ActorID: "member", OrganizationID: "org-a"}, nil
		},
	}
	require.NoError(t, h.ConfigurePackage(cfg))
	gin.SetMode(gin.TestMode)
	router := gin.New()
	for _, route := range Routes(h) {
		router.Handle(route.Method, route.Path, route.Handler)
	}
	download := func() *httptest.ResponseRecorder {
		response := httptest.NewRecorder()
		router.ServeHTTP(response, httptest.NewRequest("GET", Base+"/plugin", nil))
		return response
	}
	response := download()
	require.Equal(t, 200, response.Code)
	require.Equal(t, archive, response.Body.Bytes())
	require.Equal(t, "application/zip", response.Header().Get("Content-Type"))
	require.Equal(t, "private, no-store", response.Header().Get("Cache-Control"))
	allowed = false
	require.Equal(t, 403, download().Code, "having a valid package grants no read permission")
	allowed = true
	db.enabled = false
	require.Equal(t, 503, download().Code, "download still requires enterprise activation")
	db.enabled = true
	cfg.SHA256 = "invalid"
	require.ErrorIs(t, h.ConfigurePackage(cfg), tm.ErrUnavailable)
	require.Equal(t, 503, download().Code, "bad configuration must clear previously accepted bytes")
	require.False(t, db.called, "downloading never writes a business fact")
}
