package storeobservationshttp

import (
	"context"
	"encoding/json"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	app "task-processor/internal/app/storeobservations"
	"task-processor/internal/authidentity"
	o "task-processor/internal/storecenter/observations"
	"testing"
	"time"
)

func TestListQueryBoundedAndUnique(t *testing.T) {
	store := uuid.NewString()
	r := httptest.NewRequest("GET", "/orders?storeId="+store+"&limit=20&status=exceptional&keyword=shoe", nil)
	q, e := listQuery(r, o.Orders)
	require.NoError(t, e)
	require.Equal(t, []string{store}, q.Stores)
	require.Equal(t, 20, q.Limit)
	for _, raw := range []string{"limit=51", "storeId=" + store + "&storeId=" + store, "url=https://evil.test", "limit=01", "keyword=bad%0A", "limit=10;keyword=x"} {
		_, e = listQuery(httptest.NewRequest("GET", "/orders?"+raw, nil), o.Orders)
		require.ErrorIs(t, e, o.ErrInvalid)
	}
}

type closedPreparationPorts struct {
	o.Repository
	o.Directory
	o.Starter
	o.Access
}

func (*closedPreparationPorts) Authorize(context.Context, o.Scope, o.Kind, bool) error { return nil }
func TestPreparationClosesAllBusinessRoutesBeforePersistenceOrDispatch(t *testing.T) {
	gin.SetMode(gin.TestMode)
	p := &closedPreparationPorts{}
	a := &app.Application{Service: &o.Service{Repository: p, Directory: p, Starter: p, Access: p}, Ready: func() bool { return true }, PreparationOnly: true}
	engine := gin.New()
	bind := func(ctx context.Context, _ string) (context.Context, error) {
		return authidentity.WithAuthenticatedIdentity(ctx, authidentity.AuthenticatedIdentity{TenantID: "org-a", EffectiveOrganizationID: "org-a", UserID: "actor-a", EffectiveMemberID: "member-a", TokenExpiresAt: time.Now().Add(time.Minute)}), nil
	}
	for _, route := range Routes(a, bind) {
		engine.Handle(route.Method, route.Path, route.Handler)
	}
	for _, route := range Routes(a, bind) {
		path := strings.NewReplacer(":key", uuid.NewString(), ":sync_id", uuid.NewString(), ":store_id", uuid.NewString(), ":record_id", "record-a", ":package_id", "package-a").Replace(route.Path)
		var body io.Reader
		if strings.HasSuffix(path, "/syncs") {
			kind := "products"
			if strings.Contains(path, "/orders/") {
				kind = "orders"
			}
			body = strings.NewReader(`{"kind":"` + kind + `","stores":[]}`)
		}
		r := httptest.NewRequest(route.Method, path, body)
		if body != nil {
			r.Header.Set("Content-Type", "application/json")
			r.Header.Set("Idempotency-Key", uuid.NewString())
		}
		w := httptest.NewRecorder()
		engine.ServeHTTP(w, r)
		if strings.HasSuffix(path, "/capabilities") {
			require.Equal(t, 200, w.Code, path)
			var out struct {
				Data app.Capabilities `json:"data"`
			}
			require.NoError(t, json.Unmarshal(w.Body.Bytes(), &out))
			require.True(t, out.Data.RuntimeReady)
			require.False(t, out.Data.Available)
			require.False(t, out.Data.CanSync)
			require.False(t, out.Data.ProviderConfigured)
		} else {
			require.Equal(t, 503, w.Code, path)
		}
	}
}

type unreadBody struct{}

func (unreadBody) Read([]byte) (int, error) { panic("read-only request body must not be consumed") }
func (unreadBody) Close() error             { return nil }
func TestReadRejectsBodyWithoutBlockingOrBindingIdentity(t *testing.T) {
	gin.SetMode(gin.TestMode)
	routes := Routes(nil, nil)
	engine := gin.New()
	for _, route := range routes {
		engine.Handle(route.Method, route.Path, route.Handler)
	}
	request := httptest.NewRequest("GET", BasePath+"/orders", nil)
	request.Body = unreadBody{}
	request.ContentLength = 1
	response := httptest.NewRecorder()
	engine.ServeHTTP(response, request)
	require.Equal(t, 400, response.Code)
	for _, body := range []string{`{"kind":"orders","stores":[],"url":"https://evil.test"}`, `{"kind":"orders","kind":"orders","stores":[]}`} {
		r := httptest.NewRequest(http.MethodPost, BasePath+"/orders/syncs", io.NopCloser(strings.NewReader(body)))
		r.Header.Set("Content-Type", "application/json")
		r.Header.Set("Idempotency-Key", uuid.NewString())
		w := httptest.NewRecorder()
		engine.ServeHTTP(w, r)
		require.Equal(t, 400, w.Code)
	}
}
