package httpapi

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"net/http"
	"net/http/httptest"
	"strings"
	"task-processor/internal/product/collection"
	"task-processor/internal/product/sourcing"
	"testing"
)

type historyDetailFixture struct {
	serviceFixture
	detail collection.OwnVersionDetail
}

func (f *historyDetailFixture) ReadOwnVersion(context.Context, string, uint64) (collection.OwnVersionDetail, error) {
	return f.detail, nil
}

func TestOwnHistoryDetailFitsLegalSnapshotAndRejectsOversizedEnvelope(t *testing.T) {
	id := uuid.NewString()
	input := collection.OwnProduct{Title: "Near-limit own product", Attributes: map[string]string{}}
	for i := 0; i < 256; i++ {
		input.Attributes[fmt.Sprint(i)] = "x"
	}
	envelope, err := collection.OwnEnvelope(id, input)
	require.NoError(t, err)
	snapshot, err := sourcing.ToSnapshot(envelope)
	require.NoError(t, err)
	raw, err := json.Marshal(snapshot)
	require.NoError(t, err)
	remaining := sourcing.MaxEncodedSnapshotBytes - 32 - len(raw)
	for i := 0; i < 256; i++ {
		added := min(remaining, sourcing.MaxSourceEnvelopeStringBytes-1)
		input.Attributes[fmt.Sprint(i)] += strings.Repeat("x", added)
		remaining -= added
	}
	require.Zero(t, remaining)
	envelope, err = collection.OwnEnvelope(id, input)
	require.NoError(t, err)
	snapshot, err = sourcing.ToSnapshot(envelope)
	require.NoError(t, err)
	raw, err = json.Marshal(snapshot)
	require.NoError(t, err)
	require.Len(t, raw, sourcing.MaxEncodedSnapshotBytes-32)
	encodedSource, err := json.Marshal(envelope)
	require.NoError(t, err)
	require.LessOrEqual(t, len(encodedSource), sourcing.MaxEncodedEnvelopeBytes)
	f := &historyDetailFixture{detail: collection.OwnVersionDetail{Version: collection.OwnVersion{Version: 1, PublicationID: id, Title: input.Title}, Product: snapshot}}
	gin.SetMode(gin.TestMode)
	router := gin.New()
	for _, route := range Routes(f, func(ctx context.Context, _ string) (context.Context, error) { return ctx, nil }) {
		router.Handle(route.Method, route.Path, route.Handler)
	}
	response := httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, BasePath+"/items/"+id+"/history/1", nil))
	require.Equal(t, http.StatusOK, response.Code)
	require.Greater(t, response.Body.Len(), collection.MaxPayloadBytes)
	var actual collection.OwnVersionDetail
	require.NoError(t, json.Unmarshal(response.Body.Bytes(), &actual))
	require.Equal(t, snapshot, actual.Product)
	f.detail.Product.Description = strings.Repeat("x", collection.MaxDetailResponseBytes)
	response = httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, BasePath+"/items/"+id+"/history/1", nil))
	require.Equal(t, http.StatusServiceUnavailable, response.Code)
}

func TestOwnHistoryRoutesRejectInvalidQueryAndBodies(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	for _, route := range Routes(&serviceFixture{}, func(ctx context.Context, _ string) (context.Context, error) { return ctx, nil }) {
		router.Handle(route.Method, route.Path, route.Handler)
	}
	base := BasePath + "/items/" + uuid.NewString() + "/history"
	for _, suffix := range []string{"?limit=21", "?limit=01", "?before=0", "?before=01", "?before=9223372036854775808", "?before=1&before=2", "?productKey=foreign", "/01", "/0", "/2?publicationId=foreign"} {
		result := httptest.NewRecorder()
		router.ServeHTTP(result, httptest.NewRequest(http.MethodGet, base+suffix, nil))
		require.Equal(t, http.StatusBadRequest, result.Code, suffix)
	}
	for _, suffix := range []string{"", "/1"} {
		req := httptest.NewRequest(http.MethodGet, base+suffix, strings.NewReader("{}"))
		out := httptest.NewRecorder()
		router.ServeHTTP(out, req)
		require.Equal(t, http.StatusBadRequest, out.Code)
		req = httptest.NewRequest(http.MethodGet, base+suffix, nil)
		req.Header.Set("Idempotency-Key", uuid.NewString())
		out = httptest.NewRecorder()
		router.ServeHTTP(out, req)
		require.Equal(t, http.StatusBadRequest, out.Code)
	}
}
