package supplychainhttp

import (
	"context"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"task-processor/internal/listing/completiondraft"
	"task-processor/internal/listing/preparation"
)

func (a *completionHTTP) ListHistory(_ context.Context, id string, q completiondraft.HistoryQuery) (completiondraft.HistoryPage, error) {
	a.reads++
	a.limit = q.Limit
	return completiondraft.HistoryPage{Source: preparation.SourceItem{ID: id}, Items: []completiondraft.Record{}}, nil
}
func (a *completionHTTP) ReadRevision(_ context.Context, id string, revision int64) (completiondraft.Record, error) {
	a.reads++
	return completiondraft.Record{Source: preparation.SourceItem{ID: id}, Revision: revision}, nil
}

func TestCompletionHistoryHTTPIsReadOnlyAndStrict(t *testing.T) {
	gin.SetMode(gin.TestMode)
	app := &completionHTTP{}
	router := gin.New()
	for _, r := range CompletionDraftRoutes(app, func(ctx context.Context, _ string) (context.Context, error) { return ctx, nil }) {
		router.Handle(r.Method, r.Path, r.Handler)
	}
	base := SupplyBasePath + "/sources/" + uuid.NewString() + "/completion-draft/history"
	for _, suffix := range []string{"", "?before=3&limit=2", "/1"} {
		w := httptest.NewRecorder()
		router.ServeHTTP(w, httptest.NewRequest("GET", base+suffix, nil))
		require.Equal(t, 200, w.Code)
		require.Equal(t, "no-store", w.Header().Get("Cache-Control"))
	}
	require.Equal(t, 3, app.reads)
	for _, suffix := range []string{"?before=0", "?before=01", "?before=9007199254740992", "?before=2&before=2", "?limit=101", "?limit=0", "?limit=01", "?actor=other", "?after=" + uuid.NewString(), "?before=%ZZ", "/1?before=2"} {
		w := httptest.NewRecorder()
		router.ServeHTTP(w, httptest.NewRequest("GET", base+suffix, nil))
		require.Equal(t, 400, w.Code, suffix)
	}
	for _, revision := range []string{"0", "01", "-1", "1e2", "9007199254740992"} {
		w := httptest.NewRecorder()
		router.ServeHTTP(w, httptest.NewRequest("GET", base+"/"+revision, nil))
		require.Equal(t, 400, w.Code)
	}
	for _, suffix := range []string{"", "/1"} {
		for _, header := range []string{"Idempotency-Key", "Content-Encoding", "body"} {
			req := httptest.NewRequest("GET", base+suffix, nil)
			if header == "body" {
				req = httptest.NewRequest("GET", base+suffix, strings.NewReader("{}"))
			} else {
				req.Header.Set(header, "invalid")
			}
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)
			require.Equal(t, 400, w.Code)
		}
	}
	require.Equal(t, 3, app.reads)
	require.Zero(t, app.writes)
}
