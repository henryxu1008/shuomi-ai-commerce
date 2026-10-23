package httpapi

import (
	"bytes"
	"context"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"task-processor/internal/app/productsourcing"
	"task-processor/internal/authidentity"
	"task-processor/internal/httproute"
	"testing"
)

func TestAmazonCaptureRoutesKeepReadAndWriteAuthoritySeparate(t *testing.T) {
	routes := amazonCaptureRoutes(nil, nil)
	require.Len(t, routes, 5)
	for _, route := range routes {
		require.Equal(t, httproute.AuthPolicyVerifiedIdentity, route.AuthPolicy)
		if route.Method == "POST" && route.Path == amazonCaptureBase {
			require.Equal(t, httproute.OrganizationAccessPolicyLiveWrite, route.OrganizationAccessPolicy)
			require.Equal(t, "product_sourcing.write", route.Permission)
		} else {
			require.Equal(t, httproute.OrganizationAccessPolicyCachedRead, route.OrganizationAccessPolicy)
			require.Equal(t, "workbench.supply.read", route.Permission)
		}
	}
}

type amazonCaptureHTTPStub struct{ calls int }

func (s *amazonCaptureHTTPStub) Capture(context.Context, string, []byte) (productsourcing.AmazonCaptureResult, error) {
	s.calls++
	return productsourcing.AmazonCaptureResult{}, nil
}
func (s *amazonCaptureHTTPStub) Verify(context.Context, string, []byte) (productsourcing.AmazonCaptureResult, error) {
	s.calls++
	return productsourcing.AmazonCaptureResult{}, nil
}
func (s *amazonCaptureHTTPStub) ByKey(context.Context, string) (productsourcing.AmazonCaptureResult, error) {
	s.calls++
	return productsourcing.AmazonCaptureResult{}, nil
}
func (s *amazonCaptureHTTPStub) Read(context.Context, string) (productsourcing.AmazonCaptureResult, error) {
	s.calls++
	return productsourcing.AmazonCaptureResult{}, nil
}
func (s *amazonCaptureHTTPStub) Published(context.Context, string) (productsourcing.AmazonCaptureResult, error) {
	s.calls++
	return productsourcing.AmazonCaptureResult{}, nil
}
func TestAmazonCaptureRejectsRecreatedMemberBeforeDispatch(t *testing.T) {
	service := &amazonCaptureHTTPStub{}
	router := gin.New()
	bind := func(ctx context.Context, _ string) (context.Context, error) { return ctx, nil }
	for _, route := range amazonCaptureRoutes(service, bind) {
		router.Handle(route.Method, route.Path, route.Handler)
	}
	for _, spec := range []struct{ method, suffix string }{{"POST", ""}, {"POST", "/verify"}, {"GET", "/by-key/22222222-2222-4222-8222-22222222222b"}, {"GET", "/22222222-2222-4222-8222-22222222222b"}, {"GET", "/22222222-2222-4222-8222-22222222222b/published"}} {
		request := httptest.NewRequest(spec.method, amazonCaptureBase+spec.suffix, nil)
		request.Header.Set("X-Expected-Member-ID", "member-original")
		request = request.WithContext(authidentity.WithAuthenticatedIdentity(request.Context(), authidentity.AuthenticatedIdentity{UserID: "actor", EffectiveOrganizationID: "org", TenantID: "org", EffectiveMemberID: "member-recreated"}))
		response := httptest.NewRecorder()
		router.ServeHTTP(response, request)
		require.Equal(t, 403, response.Code)
	}
	require.Zero(t, service.calls, "recreated member must not reach any capture business port")
}
func TestAmazonCaptureHTTPStrictRequestBoundary(t *testing.T) {
	body, err := os.ReadFile("../../product/sourcing/testdata/amazon-capture-v2.json")
	require.NoError(t, err)
	for _, value := range []string{string(body), strings.Replace(string(body), `"source":"amazon"`, `"source":"1688"`, 1), strings.Replace(string(body), `"captureVersion":2`, `"captureVersion":2,"captureVersion":2`, 1)} {
		request := httptest.NewRequest(http.MethodPost, amazonCaptureBase, strings.NewReader(value))
		request.Header.Set("Content-Type", "application/json")
		request.Header.Set("Idempotency-Key", "22222222-2222-4222-8222-22222222222b")
		_, _, err = readAmazonCaptureRequest(request)
		if value == string(body) {
			require.NoError(t, err)
		} else {
			require.Error(t, err)
		}
	}
}

func TestAmazonCapturePanicDiagnosticsNeverDumpRequest(t *testing.T) {
	var output bytes.Buffer
	previous := gin.DefaultErrorWriter
	gin.DefaultErrorWriter = &output
	t.Cleanup(func() { gin.DefaultErrorWriter = previous })
	router := gin.New()
	router.Use(browserCaptureRecovery())
	router.POST(amazonCaptureBase, func(c *gin.Context) { panic("private fixture payload") })
	request := httptest.NewRequest(http.MethodPost, amazonCaptureBase, strings.NewReader("private fixture payload"))
	request.Header.Set("Authorization", "Bearer fixture-credential")
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	require.Equal(t, 500, response.Code)
	require.NotContains(t, output.String(), "fixture-credential")
	require.NotContains(t, output.String(), "private fixture payload")
	require.Contains(t, output.String(), "event=browser_capture_recovery")
}
