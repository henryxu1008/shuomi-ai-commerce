package httpx

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"task-processor/internal/crawler/shared"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

// Use the retained Amazon handler to protect the shared scoped-result boundary;
// the retired 1688 submission protocol is not needed for these assertions.
func TestCrawlerHandlerKeepsTrustedScopeForTaskResults(t *testing.T) {
	trustedKey := struct{}{}
	for _, scope := range []struct {
		name     string
		resolver TenantResolver
		status   int
	}{
		{"missing resolver", nil, http.StatusForbidden},
		{"unresolved", func(context.Context) (int64, bool) { return 101, false }, http.StatusForbidden},
		{"invalid tenant", func(context.Context) (int64, bool) { return 0, true }, http.StatusForbidden},
		{"trusted context", func(ctx context.Context) (int64, bool) {
			id, ok := ctx.Value(trustedKey).(int64)
			return id, ok
		}, http.StatusOK},
	} {
		for _, route := range []struct{ method, path string }{
			{http.MethodGet, "/api/v1/tasks/known-task"},
			{http.MethodDelete, "/api/v1/tasks/known-task"},
			{http.MethodGet, "/api/v1/tasks"},
		} {
			t.Run(scope.name+"/"+route.method+route.path, func(t *testing.T) {
				service := &scopedCrawlerResults{}
				handler := NewCrawlerHandler(service, logrus.New())
				handler.tenantResolver = scope.resolver
				req := httptest.NewRequest(route.method, route.path+"?tenant_id=999", nil)
				req.Header.Set("X-Tenant-ID", "999")
				req = req.WithContext(context.WithValue(req.Context(), trustedKey, int64(101)))
				rec := httptest.NewRecorder()

				handler.RegisterRoutes().ServeHTTP(rec, req)

				assert.Equal(t, scope.status, rec.Code)
				if scope.status == http.StatusOK {
					assert.Equal(t, int64(101), service.tenantID)
					assert.Equal(t, 1, service.calls)
				} else {
					assert.Zero(t, service.calls)
				}
			})
		}
	}
}

type scopedCrawlerResults struct {
	stubAmazonCrawlerService
	tenantID int64
	calls    int
}

func (s *scopedCrawlerResults) GetTaskForTenant(tenantID int64, taskID string) (*shared.CrawlerResult, error) {
	s.tenantID, s.calls = tenantID, s.calls+1
	return &shared.CrawlerResult{}, nil
}

func (s *scopedCrawlerResults) DeleteTaskForTenant(tenantID int64, taskID string) {
	s.tenantID, s.calls = tenantID, s.calls+1
}

func (s *scopedCrawlerResults) GetAllTasksForTenant(tenantID int64) []*shared.CrawlerResult {
	s.tenantID, s.calls = tenantID, s.calls+1
	return []*shared.CrawlerResult{}
}
