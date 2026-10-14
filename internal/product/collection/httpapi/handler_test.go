package httpapi

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"task-processor/internal/httproute"
	"task-processor/internal/product/collection"
)

type serviceFixture struct {
	mutations int
	key       string
}

func (s *serviceFixture) Mutate(_ context.Context, key string, _ collection.Mutation) (collection.Receipt, error) {
	s.mutations++
	s.key = key
	return collection.Receipt{OperationID: key, Revision: 1}, nil
}
func (*serviceFixture) ListBatches(context.Context, collection.Query) (collection.Page[collection.Batch], error) {
	return collection.Page[collection.Batch]{Items: []collection.Batch{}}, nil
}
func (*serviceFixture) ListItems(context.Context, string, collection.Query) (collection.Page[collection.Item], error) {
	return collection.Page[collection.Item]{Items: []collection.Item{}}, nil
}
func (*serviceFixture) ReadItem(context.Context, string) (collection.ItemDetail, error) {
	return collection.ItemDetail{}, collection.ErrNotFound
}
func (*serviceFixture) ReadBatch(_ context.Context, id string) (collection.Batch, error) {
	if !collection.ValidID(id) {
		return collection.Batch{}, collection.ErrInvalid
	}
	return collection.Batch{ID: id, Name: "自有商品库", Kind: "own", Revision: 3, Count: 2, SupplyTransferSupported: true, CreatedAt: time.Date(2026, 10, 14, 0, 0, 0, 0, time.UTC)}, nil
}

func TestCollectionBatchReadExposesExistingOwnerRevisionAndRejectsMutationInputs(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	for _, route := range Routes(&serviceFixture{}, func(ctx context.Context, _ string) (context.Context, error) { return ctx, nil }) {
		router.Handle(route.Method, route.Path, route.Handler)
	}
	id := uuid.NewString()
	response := httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, BasePath+"/batches/"+id, nil))
	require.Equal(t, http.StatusOK, response.Code)
	require.Equal(t, "no-store", response.Header().Get("Cache-Control"))
	require.Contains(t, response.Body.String(), `"revision":3`)
	require.Contains(t, response.Body.String(), id)
	for _, suffix := range []string{"?expectedRevision=3", "?organizationId=foreign"} {
		response = httptest.NewRecorder()
		router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, BasePath+"/batches/"+id+suffix, nil))
		require.Equal(t, http.StatusBadRequest, response.Code)
	}
	response = httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, BasePath+"/batches/"+id, strings.NewReader(`{}`)))
	require.Equal(t, http.StatusBadRequest, response.Code)
}
func (*serviceFixture) ReadOperation(context.Context, string) (collection.Receipt, error) {
	return collection.Receipt{}, collection.ErrNotFound
}
func (*serviceFixture) ListOwnHistory(context.Context, string, uint64, int) (collection.OwnHistoryPage, error) {
	return collection.OwnHistoryPage{}, collection.ErrNotFound
}
func (*serviceFixture) ReadOwnVersion(context.Context, string, uint64) (collection.OwnVersionDetail, error) {
	return collection.OwnVersionDetail{}, collection.ErrNotFound
}
func TestCollectionRoutesKeepLiveActorScopeAndStrictCommands(t *testing.T) {
	gin.SetMode(gin.TestMode)
	service := &serviceFixture{}
	routes := Routes(service, func(ctx context.Context, bearer string) (context.Context, error) { return ctx, nil })
	router := gin.New()
	for _, r := range routes {
		require.Equal(t, httproute.AuthPolicyVerifiedIdentity, r.AuthPolicy)
		require.Equal(t, httproute.OrganizationAccessPolicyLiveWrite, r.OrganizationAccessPolicy)
		require.Equal(t, collection.Timeout, r.RequestTimeout)
		router.Handle(r.Method, r.Path, r.Handler)
	}
	key := uuid.NewString()
	for _, body := range []string{`{"action":"create_batch","name":"a","organizationId":"foreign"}`, `{"action":"create_batch","name":"a","name":"b"}`} {
		req := httptest.NewRequest(http.MethodPost, BasePath+"/commands", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Idempotency-Key", key)
		response := httptest.NewRecorder()
		router.ServeHTTP(response, req)
		require.Equal(t, http.StatusBadRequest, response.Code)
	}
	require.Zero(t, service.mutations)
	req := httptest.NewRequest(http.MethodPost, BasePath+"/commands", strings.NewReader(`{"action":"create_batch","name":"a"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Idempotency-Key", key)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, req)
	require.Equal(t, http.StatusOK, response.Code)
	require.Equal(t, key, service.key)
	req = httptest.NewRequest(http.MethodGet, BasePath+"/batches?organizationId=foreign", nil)
	response = httptest.NewRecorder()
	router.ServeHTTP(response, req)
	require.Equal(t, http.StatusBadRequest, response.Code)
}
