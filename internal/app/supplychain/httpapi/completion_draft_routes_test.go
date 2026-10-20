package supplychainhttp

import (
	"context"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"net/http"
	"net/http/httptest"
	"strings"
	"task-processor/internal/listing/completiondraft"
	"task-processor/internal/listing/preparation"
	"task-processor/internal/product/collection"
	"testing"
)

type completionHTTP struct {
	CompletionDraftApplication
	writes int
	reads  int
	limit  int
	err    error
}

func (a *completionHTTP) ListForCollectionItem(_ context.Context, id string, q collection.Query) (completiondraft.ItemPage, error) {
	a.reads++
	a.limit = q.Limit
	if a.err != nil {
		return completiondraft.ItemPage{}, a.err
	}
	return completiondraft.ItemPage{CollectionItemID: id, Page: collection.Page[completiondraft.Record]{Items: []completiondraft.Record{}, Total: 0}}, nil
}
func TestItemDraftHTTPPreservesCollectionReadFailureStatuses(t *testing.T) {
	gin.SetMode(gin.TestMode)
	for _, failure := range []struct {
		err    error
		status int
		code   string
	}{{collection.ErrNotFound, 404, "NOT_FOUND"}, {collection.ErrForbidden, 403, "PERMISSION_DENIED"}, {collection.ErrInvalid, 400, "INVALID_REQUEST"}, {collection.ErrUnavailable, 503, "DEPENDENCY_UNAVAILABLE"}} {
		t.Run(failure.code, func(t *testing.T) {
			app := &completionHTTP{err: fmt.Errorf("current item read: %w", failure.err)}
			router := gin.New()
			for _, r := range CompletionDraftRoutes(app, func(ctx context.Context, _ string) (context.Context, error) { return ctx, nil }) {
				router.Handle(r.Method, r.Path, r.Handler)
			}
			w := httptest.NewRecorder()
			router.ServeHTTP(w, httptest.NewRequest("GET", SupplyBasePath+"/collection-items/"+uuid.NewString()+"/completion-drafts", nil))
			require.Equal(t, failure.status, w.Code)
			require.JSONEq(t, `{"code":"`+failure.code+`"}`, w.Body.String())
		})
	}
}
func TestItemDraftHTTPRejectsUnboundedOrAuthorityQueriesAndGETBodies(t *testing.T) {
	gin.SetMode(gin.TestMode)
	app := &completionHTTP{}
	router := gin.New()
	for _, r := range CompletionDraftRoutes(app, func(ctx context.Context, _ string) (context.Context, error) { return ctx, nil }) {
		router.Handle(r.Method, r.Path, r.Handler)
	}
	id := uuid.NewString()
	path := SupplyBasePath + "/collection-items/" + id + "/completion-drafts"
	for _, query := range []string{"?limit=20", "?limit=101", "?itemId=" + id, "?memberId=other", "?limit=20&limit=20", "?version=2"} {
		w := httptest.NewRecorder()
		router.ServeHTTP(w, httptest.NewRequest("GET", path+query, nil))
		if query == "?limit=20" {
			require.Equal(t, 200, w.Code)
			require.Contains(t, w.Body.String(), id)
			require.Equal(t, "no-store", w.Header().Get("Cache-Control"))
		} else {
			require.Equal(t, 400, w.Code)
		}
	}
	wDefault := httptest.NewRecorder()
	router.ServeHTTP(wDefault, httptest.NewRequest("GET", path, nil))
	require.Equal(t, 200, wDefault.Code)
	require.Equal(t, 20, app.limit)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, httptest.NewRequest("GET", path, strings.NewReader(`{}`)))
	require.Equal(t, 400, w.Code)
	require.Equal(t, 2, app.reads)
}

func (a *completionHTTP) Save(_ context.Context, key string, in completiondraft.SaveInput) (completiondraft.Receipt, error) {
	a.writes++
	return completiondraft.Receipt{Record: completiondraft.Record{ID: key, Content: in.Content, Status: completiondraft.StatusPending}}, nil
}
func TestCompletionDraftHTTPIsBoundedStrictAndHasNoOfficialExecutionRoutes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	app := &completionHTTP{}
	router := gin.New()
	routes := CompletionDraftRoutes(app, func(ctx context.Context, _ string) (context.Context, error) { return ctx, nil })
	require.Len(t, routes, 7)
	for _, r := range routes {
		require.NoError(t, ValidateCompletionDraftDescriptor(r))
		router.Handle(r.Method, r.Path, r.Handler)
	}
	id := uuid.NewString()
	valid := `{"sourceId":"` + id + `","expectedRevision":0,"content":{"title":"","description":""}}`
	for _, body := range []string{valid, strings.Replace(valid, `"title":""`, `"title":"","title":"other"`, 1), strings.Replace(valid, `"expectedRevision":0`, `"expectedRevision":0,"storeId":"`+id+`"`, 1), strings.Replace(valid, `"description":""`, `"description":"`+strings.Repeat("a", 70<<10)+`"`, 1)} {
		r := httptest.NewRequest(http.MethodPost, SupplyBasePath+"/completion-drafts", strings.NewReader(body))
		r.Header.Set("Content-Type", "application/json")
		r.Header.Set("Idempotency-Key", id)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, r)
		if body == valid {
			require.Equal(t, http.StatusOK, w.Code)
			require.Contains(t, w.Body.String(), "pending_completion")
		} else {
			require.Equal(t, http.StatusBadRequest, w.Code)
		}
	}
	require.Equal(t, 1, app.writes)
	for _, path := range []string{"/targets", "/operations", "/uploads/resolve"} {
		w := httptest.NewRecorder()
		router.ServeHTTP(w, httptest.NewRequest("POST", SupplyBasePath+path, nil))
		require.Equal(t, 404, w.Code)
	}
	_, err := supplyQuery(httptest.NewRequest("GET", SupplyBasePath+"/"+id+"/completion-drafts?limit=101", nil))
	require.ErrorIs(t, err, preparation.ErrInvalid)
}

func TestCompletionDraftStructuredHTTPKeepsStrictNestedFieldsAndZeroValues(t *testing.T) {
	gin.SetMode(gin.TestMode)
	app := &completionHTTP{}
	router := gin.New()
	for _, route := range CompletionDraftRoutes(app, func(ctx context.Context, _ string) (context.Context, error) { return ctx, nil }) {
		router.Handle(route.Method, route.Path, route.Handler)
	}
	id := uuid.NewString()
	valid := `{"sourceId":"` + id + `","expectedRevision":0,"content":{"title":"","description":"","structured":{"attributes":[],"variants":[{"sourceIndex":0,"price":{"currency":"CNY","amount":0},"stock":0}]}}}`
	for _, body := range []string{valid, strings.Replace(valid, `"attributes":[]`, `"attributes":[],"storeId":"forbidden"`, 1), strings.Replace(valid, `"amount":0`, `"amount":0,"amount":1`, 1), strings.Replace(valid, `"sourceIndex":0`, `"sourceIndex":0,"sku":"changed"`, 1), strings.Replace(valid, `"stock":0`, `"stock":-1`, 1)} {
		request := httptest.NewRequest(http.MethodPost, SupplyBasePath+"/completion-drafts", strings.NewReader(body))
		request.Header.Set("Content-Type", "application/json")
		request.Header.Set("Idempotency-Key", id)
		response := httptest.NewRecorder()
		router.ServeHTTP(response, request)
		if body == valid {
			require.Equal(t, 200, response.Code)
			require.Contains(t, response.Body.String(), `"stock":0`)
			require.Contains(t, response.Body.String(), `"amount":0`)
		} else {
			require.Equal(t, 400, response.Code)
		}
	}
	require.Equal(t, 1, app.writes)
}
