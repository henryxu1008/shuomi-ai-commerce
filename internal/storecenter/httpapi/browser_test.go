package httpapi

import (
	"context"
	"github.com/stretchr/testify/require"
	"net/http"
	"strings"
	"task-processor/internal/authidentity"
	"task-processor/internal/authz"
	"task-processor/internal/httproute"
	"task-processor/internal/storecenter"
	"testing"
	"time"
)

type browserHTTPStub struct {
	calls   int
	org     string
	receipt storecenter.BrowserReceipt
}

func (s *browserHTTPStub) Admit(_ context.Context, org, store, key string, c storecenter.BrowserCommand) (storecenter.BrowserReceipt, error) {
	s.calls++
	s.org = org
	return s.receipt, nil
}
func (s *browserHTTPStub) Receipt(ctx context.Context, store, key string) (storecenter.BrowserReceipt, error) {
	s.calls++
	id, _ := authidentity.AuthenticatedIdentityFromContext(ctx)
	s.org = id.EffectiveOrganizationID
	return s.receipt, nil
}
func (s *browserHTTPStub) Report(ctx context.Context, store, key string, r storecenter.BrowserReport) (storecenter.BrowserReceipt, error) {
	return s.Receipt(ctx, store, key)
}
func (s *browserHTTPStub) Audit(_ context.Context, org, store string, q storecenter.BrowserAuditQuery) (storecenter.BrowserAuditPage, error) {
	s.calls++
	s.org = org
	return storecenter.BrowserAuditPage{}, nil
}
func TestBrowserHTTPContractsAndIdentityOnlyReceipt(t *testing.T) {
	h := mustHandler(t, &storeServiceStub{})
	s := &browserHTTPStub{}
	require.NoError(t, h.SetBrowserOperations(s))
	require.Error(t, h.SetBrowserOperations(s))
	for _, r := range h.browserRoutes() {
		require.Equal(t, httproute.AuthPolicyCurrentIdentity, r.AuthPolicy)
		require.Equal(t, 15*time.Second, r.RequestTimeout)
		if strings.HasSuffix(r.Path, "/:operation_id") || strings.HasSuffix(r.Path, "/reports") {
			require.Empty(t, r.Permission)
			require.Equal(t, httproute.OrganizationAccessPolicyNone, r.OrganizationAccessPolicy)
		} else {
			require.Equal(t, httproute.OrganizationAccessPolicyLiveWrite, r.OrganizationAccessPolicy)
			if r.Method == http.MethodPost {
				require.Equal(t, authz.PermissionWorkbenchStoreBrowserUse, r.Permission)
			}
		}
	}
	router := mountedRouter(t, h)
	base := "/api/v1/workbench/stores/" + testStoreID + "/browser-operations"
	response := serve(t, router, authidentity.AuthenticatedIdentity{UserID: "original-subject"}, "GET", base+"/"+testCreateKey, "", nil)
	require.Equal(t, 200, response.Code)
	require.Empty(t, s.org)
	require.Equal(t, "private, no-store", response.Header().Get("Cache-Control"))
	for _, body := range []string{`{"action":"open_environment","action":"configure_environment"}`, `{"schemaVersion":"store-browser-v1","extra":"private"}`, strings.Repeat("x", 16385)} {
		before := s.calls
		r := serve(t, router, validIdentity(), "POST", base, body, map[string][]string{"Idempotency-Key": {testCreateKey}})
		require.Equal(t, 400, r.Code)
		require.Equal(t, before, s.calls)
	}
	before := s.calls
	require.Equal(t, 400, serve(t, router, validIdentity(), "GET", "/api/v1/workbench/stores/"+testStoreID+"/audit?clear=true", "", nil).Code)
	require.Equal(t, before, s.calls)
}
