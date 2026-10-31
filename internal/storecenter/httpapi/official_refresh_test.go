package httpapi

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"task-processor/internal/authidentity"
	"task-processor/internal/storecenter"
)

type refreshConnectionFixture struct {
	OfficialConnectionService
	calls                 int
	org, store, operation string
	generation            int64
}

func (s *refreshConnectionFixture) Refresh(_ context.Context, org, store, operation string, generation int64) (storecenter.OfficialConnectionView, error) {
	s.calls++
	s.org, s.store, s.operation, s.generation = org, store, operation, generation
	return storecenter.OfficialConnectionView{Platform: storecenter.PlatformTikTokShop, Status: storecenter.ConnectionStatusConnected}, nil
}

func TestOfficialRefreshUsesVerifiedScopeAndCanonicalOperation(t *testing.T) {
	gin.SetMode(gin.TestMode)
	s := &refreshConnectionFixture{}
	h := &Handler{officialConnections: s}
	router := gin.New()
	router.POST("/stores/:store_id/refresh", h.RefreshOfficialConnection)
	path := "/stores/" + testStoreID + "/refresh"
	body := fmt.Sprintf(`{"operationId":%q,"generation":"7"}`, testCreateKey)
	for _, tc := range []struct {
		name, body string
		identity   authidentity.AuthenticatedIdentity
		status     int
	}{
		{"missing identity", body, authidentity.AuthenticatedIdentity{}, http.StatusUnauthorized},
		{"bad generation", fmt.Sprintf(`{"operationId":%q,"generation":"07"}`, testCreateKey), validIdentity(), http.StatusBadRequest},
		{"injected organization", fmt.Sprintf(`{"operationId":%q,"generation":"7","organizationId":"other"}`, testCreateKey), validIdentity(), http.StatusBadRequest},
		{"valid explicit refresh", body, validIdentity(), http.StatusOK},
	} {
		t.Run(tc.name, func(t *testing.T) {
			before := s.calls
			response := serve(t, router, tc.identity, http.MethodPost, path, tc.body, nil)
			require.Equal(t, tc.status, response.Code, response.Body.String())
			if tc.status != http.StatusOK {
				require.Equal(t, before, s.calls)
				return
			}
			require.Equal(t, before+1, s.calls)
			require.Equal(t, "org-effective", s.org)
			require.Equal(t, testStoreID, s.store)
			require.Equal(t, testCreateKey, s.operation)
			require.EqualValues(t, 7, s.generation)
			require.Equal(t, "no-store", response.Header().Get("Cache-Control"))
		})
	}
}
