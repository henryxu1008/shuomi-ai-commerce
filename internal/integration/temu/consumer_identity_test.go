package temu

import (
	"context"
	"github.com/stretchr/testify/require"
	"net/http"
	"strings"
	"task-processor/internal/storecenter"
	"testing"
	"time"
)

func TestRegionalOrdersRequireExplicitR2ApplicationAndSemiIdentity(t *testing.T) {
	for _, tc := range []struct {
		name, body string
		valid      bool
	}{{"local", localIdentityResponse, true}, {"semi", strings.Replace(strings.Replace(localIdentityResponse, `"mallType":100`, `"mallType":1`, 1), `"regionId":211`, `"regionId":211,"semiUniqueId":"semi-1"`, 1), true}, {"missing-semi", strings.Replace(localIdentityResponse, `"mallType":100`, `"mallType":1`, 1), false}, {"full", strings.Replace(localIdentityResponse, `"mallType":100`, `"mallType":0`, 1), false}, {"other-region", strings.Replace(localIdentityResponse, `"regionId":211`, `"regionId":212`, 1), false}} {
		t.Run(tc.name, func(t *testing.T) {
			app := storecenter.OfficialApplication{AppID: "test-app", Version: "v:temu:GLOBAL:manual_token:3:r2", Platform: storecenter.PlatformTemu, IdentityPartition: storecenter.PartitionGlobal, AuthorizationMethod: storecenter.AuthorizationManualToken, SupportedModes: storecenter.ModesLocal | storecenter.ModesSemiManaged}
			c, e := NewOfficialClient(OfficialClientOptions{Application: app, AppSecret: "test-secret-0123456789", HTTPClient: &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
				require.Equal(t, "openapi-b-global.temu.com", r.URL.Host)
				return jsonResponse(tc.body), nil
			})}})
			require.NoError(t, e)
			c.now = func() time.Time { return time.Unix(1760000000, 0) }
			info, e := c.QueryStore(context.Background(), tokenCredential())
			if !tc.valid {
				require.Error(t, e)
				return
			}
			require.NoError(t, e)
			require.Equal(t, "temu-regional-order-identity-v2", info.Verification.ContractRevision)
			_, e = info.Verification.MerchantIdentityHash()
			require.NoError(t, e)
			if tc.name == "semi" {
				r1 := identityClient(t, storecenter.PartitionGlobal, roundTripFunc(func(*http.Request) (*http.Response, error) { return jsonResponse(tc.body), nil }))
				_, e = r1.QueryStore(context.Background(), tokenCredential())
				require.Error(t, e)
			}
		})
	}
	app := storecenter.OfficialApplication{AppID: "test-app", Version: "v:temu:EU:manual_token:3:r2", Platform: storecenter.PlatformTemu, IdentityPartition: storecenter.PartitionEU, AuthorizationMethod: storecenter.AuthorizationManualToken, SupportedModes: storecenter.ModesLocal | storecenter.ModesSemiManaged}
	_, e := NewOfficialClient(OfficialClientOptions{Application: app, AppSecret: "test-secret-0123456789"})
	require.Error(t, e)
}
