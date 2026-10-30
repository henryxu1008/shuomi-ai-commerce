package kernelhttp

import (
	"context"
	"encoding/json"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"net/http/httptest"
	"strings"
	"task-processor/internal/authidentity"
	k "task-processor/internal/browserkernel"
	"testing"
	"time"
)

type repository struct {
	calls int
	err   error
}

func (r *repository) Candidates(context.Context, string, string, string, int) ([]k.Release, error) {
	r.calls++
	return []k.Release{}, r.err
}
func (r *repository) Find(context.Context, string) (k.Release, error) {
	r.calls++
	return k.Release{}, k.ErrNotFound
}
func (r *repository) Publish(_ context.Context, id string, p k.Publish, _ string) (k.Release, error) {
	r.calls++
	return k.Release{ReleaseID: id, Publish: p}, r.err
}
func (r *repository) Withdraw(context.Context, string, string) (k.Release, error) {
	r.calls++
	return k.Release{}, r.err
}
func (r *repository) Recommend(context.Context, string, bool, string) (k.Release, error) {
	r.calls++
	return k.Release{}, r.err
}

const id = "00000000-0000-4000-8000-000000000001"
const query = "?os=windows&arch=amd64&clientVersion=1.0.0"

func inputJSON() string {
	p := k.Publish{Version: "132.0.0.1", OS: "windows", Arch: "amd64", MinClientVersion: "1.0.0", ArchiveFormat: "zip", Size: 12, SHA256: strings.Repeat("a", 64), DownloadURL: "https://assets.example.test/132.zip"}
	b, _ := json.Marshal(p)
	return string(b)
}
func TestStrictTransportBeforeOwner(t *testing.T) {
	for _, tc := range []struct {
		method, path, body string
		want               int
	}{
		{"GET", "/api/v1/browser/kernel-releases" + query, "", 200},
		{"GET", "/api/v1/browser/kernel-releases" + query + "&os=linux", "", 400},
		{"GET", "/api/v1/browser/kernel-releases" + query + "&downloadUrl=https://evil.test", "", 400},
		{"GET", "/api/v1/browser/kernel-releases" + query + "&bad=%zz", "", 400},
		{"GET", "/api/v1/browser/kernel-releases" + query + "&limit=01", "", 400},
		{"GET", "/api/v1/browser/kernel-releases" + query, "{}", 400},
		{"GET", "/api/v1/browser/kernel-releases/" + id + "/artifact" + query + "&limit=2", "", 400},
		{"PUT", "/api/v1/platform/browser/kernel-releases/" + id, inputJSON(), 200},
		{"PUT", "/api/v1/platform/browser/kernel-releases/" + id, strings.TrimSuffix(inputJSON(), "}") + `,"size":4}`, 400},
		{"PUT", "/api/v1/platform/browser/kernel-releases/" + id, strings.Replace(inputJSON(), `"maxClientVersionExclusive":"",`, "", 1), 400},
		{"PUT", "/api/v1/platform/browser/kernel-releases/" + id, strings.TrimSuffix(inputJSON(), "}") + `,"recommended":true}`, 400},
		{"PUT", "/api/v1/platform/browser/kernel-releases/" + id, "null", 400},
		{"PUT", "/api/v1/platform/browser/kernel-releases/" + id + "/recommendation", "{}", 400},
		{"POST", "/api/v1/platform/browser/kernel-releases/" + id + "/withdraw", "{}", 400},
	} {
		t.Run(tc.method+tc.path+tc.body, func(t *testing.T) {
			r := &repository{}
			s, e := k.New(r, []string{"https://assets.example.test"})
			require.NoError(t, e)
			router := gin.New()
			router.Use(func(c *gin.Context) {
				c.Request = c.Request.WithContext(authidentity.WithAuthenticatedIdentity(c.Request.Context(), authidentity.AuthenticatedIdentity{UserID: "publisher", TokenExpiresAt: time.Now().Add(time.Minute)}))
			})
			for _, route := range Routes(&Handler{Service: s}) {
				router.Handle(route.Method, route.Path, route.Handler)
			}
			req := httptest.NewRequest(tc.method, tc.path, strings.NewReader(tc.body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)
			require.Equal(t, tc.want, w.Code, w.Body.String())
			require.Equal(t, "no-store", w.Header().Get("Cache-Control"))
			if tc.want == 400 {
				require.Zero(t, r.calls)
				require.JSONEq(t, `{"code":"INVALID_REQUEST"}`, w.Body.String())
			}
		})
	}
}
