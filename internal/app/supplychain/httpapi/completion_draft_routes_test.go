package supplychainhttp

import (
	"context"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"net/http"
	"net/http/httptest"
	"strings"
	"task-processor/internal/listing/completiondraft"
	"task-processor/internal/listing/preparation"
	"testing"
)

type completionHTTP struct {
	CompletionDraftApplication
	writes int
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
	require.Len(t, routes, 4)
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
