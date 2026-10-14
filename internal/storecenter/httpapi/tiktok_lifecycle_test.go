package httpapi

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"task-processor/internal/storecenter"
)

type lifecycleFixture struct {
	OfficialConnectionService
	calls   int
	failure error
}

func (*lifecycleFixture) TikTokLifecycleEnabled() bool { return true }
func (s *lifecycleFixture) ReceiveTikTokLifecycle(context.Context, []byte, string) error {
	s.calls++
	return s.failure
}
func TestTikTokLifecycleRejectsInvalidEnvelopeAndPreservesSignatureFailure(t *testing.T) {
	gin.SetMode(gin.TestMode)
	s := &lifecycleFixture{}
	h := &Handler{officialConnections: s}
	engine := gin.New()
	route := h.tikTokLifecycleRoute()
	require.NotNil(t, route)
	engine.POST(route.Path, route.Handler)
	for _, test := range []struct {
		body, contentType string
		duplicate         bool
		failure           error
		status, calls     int
	}{
		{body: `{}`, contentType: "text/plain", status: 400},
		{body: strings.Repeat("x", 65537), contentType: "application/json", status: 400},
		{body: `{}`, contentType: "application/json", duplicate: true, status: 400},
		{body: `{}`, contentType: "application/json", failure: storecenter.ErrOfficialAuthorizationRejected, status: 401, calls: 1},
		{body: `{}`, contentType: "application/json", status: 200, calls: 1},
	} {
		s.calls = 0
		s.failure = test.failure
		r := httptest.NewRequest(http.MethodPost, route.Path, strings.NewReader(test.body))
		r.Header.Set("Content-Type", test.contentType)
		r.Header.Add("Authorization", "signed")
		if test.duplicate {
			r.Header.Add("Authorization", "duplicate")
		}
		response := httptest.NewRecorder()
		engine.ServeHTTP(response, r)
		require.Equal(t, test.status, response.Code)
		require.Equal(t, test.calls, s.calls)
		require.Empty(t, response.Body.String())
	}
	require.Nil(t, (&Handler{}).tikTokLifecycleRoute())
}
