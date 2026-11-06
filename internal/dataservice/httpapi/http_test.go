package dataservicehttpapi

import (
	"context"
	"crypto/tls"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"net/http/httptest"
	"strings"
	"task-processor/internal/dataservice"
	"task-processor/internal/product/collection"
	"testing"
)

type closedExecutionAccess struct{ dataservice.Access }

func (closedExecutionAccess) Resolve(context.Context, string) (collection.Scope, error) {
	return collection.Scope{OrganizationID: "org", ActorID: "actor", MemberID: "member"}, nil
}

func TestDisabledExecutionRejectsCaptureAndCancelBeforeAnyExecutionPort(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	for _, route := range BuildRoutes(Dependencies{Access: closedExecutionAccess{}}) {
		router.Handle(route.Method, route.Path, route.Handler)
	}
	for _, path := range []string{ConsoleBase + "/amazon/jobs", ConsoleBase + "/amazon/jobs/" + uuid.NewString() + "/cancel"} {
		request := httptest.NewRequest("POST", path, strings.NewReader(`{}`))
		request.Header.Set("Content-Type", "application/json")
		request.Header.Set("Idempotency-Key", uuid.NewString())
		response := httptest.NewRecorder()
		router.ServeHTTP(response, request)
		require.Equal(t, 503, response.Code, path)
		require.Contains(t, response.Body.String(), "DATA_UNAVAILABLE")
	}
}

func TestTransportRejectsAmbiguousBodyAndUnauthorizedProxy(t *testing.T) {
	for _, body := range []string{`{"maximumRows":1,"maximumRows":2}`, `{"Query":{}}`, `{"actorId":"other"}`, `{"query":{},"maximumRows":1,"maximumCostFen":5,"extra":true}`} {
		r := httptest.NewRequest("POST", "/", strings.NewReader(body))
		r.Header.Set("Content-Type", "application/json")
		var input createJob
		require.ErrorIs(t, readJSON(r, &input), dataservice.ErrInvalid)
	}
	r := httptest.NewRequest("GET", "/", nil)
	r.RemoteAddr = "192.0.2.1:4000"
	r.Header.Set("X-Forwarded-Proto", "https")
	r.Header.Set("X-Forwarded-For", "198.51.100.2")
	_, err := externalPeer(r, nil)
	require.ErrorIs(t, err, dataservice.ErrForbidden)
	r.TLS = &tls.ConnectionState{}
	peer, err := externalPeer(r, nil)
	require.NoError(t, err)
	require.Equal(t, "192.0.2.1", peer)
	peer, err = externalPeer(r, []string{"192.0.2.0/24"})
	require.NoError(t, err)
	require.Equal(t, "198.51.100.2", peer)
	r.Header.Set("X-Forwarded-For", "198.51.100.2,198.51.100.3")
	_, err = externalPeer(r, []string{"192.0.2.0/24"})
	require.ErrorIs(t, err, dataservice.ErrForbidden)
}
