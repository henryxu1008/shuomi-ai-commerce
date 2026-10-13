package temu

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"task-processor/internal/storecenter"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func identityClient(t *testing.T, partition storecenter.OfficialIdentityPartition, transport http.RoundTripper) *OfficialClient {
	t.Helper()
	modes := storecenter.ModesLocal
	if partition == storecenter.PartitionCN {
		modes = storecenter.ModesFullyManaged | storecenter.ModesSemiManaged
	}
	c, err := NewOfficialClient(OfficialClientOptions{
		Application: storecenter.OfficialApplication{AppID: "test-app", Version: "test-v1", Platform: storecenter.PlatformTemu, IdentityPartition: partition, AuthorizationMethod: storecenter.AuthorizationManualToken, SupportedModes: modes},
		AppSecret:   "test-secret-0123456789", HTTPClient: &http.Client{Transport: transport},
	})
	require.NoError(t, err)
	c.now = func() time.Time { return time.Unix(1760000000, 0) }
	return c
}

func tokenCredential() storecenter.OfficialMerchantCredential {
	return storecenter.OfficialMerchantCredential{AppID: "test-app", Temu: &storecenter.TemuMerchantCredential{AccessToken: "test-token"}}
}

func jsonResponse(body string) *http.Response {
	return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(body)), Header: make(http.Header)}
}

const supplierIdentityResponse = `{"success":true,"errorCode":1000000,"result":{"uniqueId":"pa-mall-123","mallName":"Verified supplier","expiredTime":1760003600,"apiScopeList":["bg.glo.goods.add"]}}`
const localIdentityResponse = `{"success":true,"errorCode":1000000,"result":{"mallId":9223372036854775806,"mallType":100,"regionId":211,"expiredTime":1760003600,"apiScopeList":["temu.local.goods.v2.add"]}}`

func TestSupplierVerificationRequiresSameCredentialPAIdentityAndExplicitCNMode(t *testing.T) {
	for _, tc := range []struct {
		name, modeBody string
		wantMode       storecenter.OfficialMerchantMode
		valid          bool
	}{
		{"full", `{"semiManagedMall":false}`, storecenter.MerchantFullyManaged, true},
		{"semi", `{"semiManagedMall":true}`, storecenter.MerchantSemiManaged, true},
		{"missing boolean", `{}`, "", false},
		{"null boolean", `{"semiManagedMall":null}`, "", false},
		{"string boolean", `{"semiManagedMall":"false"}`, "", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			calls := 0
			var firstDeadline time.Time
			c := identityClient(t, storecenter.PartitionCN, roundTripFunc(func(r *http.Request) (*http.Response, error) {
				calls++
				require.Equal(t, http.MethodPost, r.Method)
				require.Equal(t, "/openapi/router", r.URL.Path)
				var payload map[string]json.RawMessage
				require.NoError(t, json.NewDecoder(r.Body).Decode(&payload))
				require.JSONEq(t, `"test-app"`, string(payload["app_key"]))
				require.JSONEq(t, `"test-token"`, string(payload["access_token"]))
				deadline, ok := r.Context().Deadline()
				require.True(t, ok)
				require.LessOrEqual(t, time.Until(deadline), 5*time.Second)
				if calls == 1 {
					firstDeadline = deadline
					require.Equal(t, "openapi-b-partner.temu.com", r.URL.Host)
					require.JSONEq(t, `"bg.open.accesstoken.info.get.global"`, string(payload["type"]))
					return jsonResponse(supplierIdentityResponse), nil
				}
				require.Equal(t, 2, calls)
				require.Equal(t, firstDeadline, deadline, "the second read cannot renew the total verification deadline")
				require.Equal(t, "openapi.kuajingmaihuo.com", r.URL.Host)
				require.JSONEq(t, `"bg.mall.info.get"`, string(payload["type"]))
				return jsonResponse(`{"success":true,"errorCode":1000000,"result":` + tc.modeBody + `}`), nil
			}))
			info, err := c.QueryStore(context.Background(), tokenCredential())
			if !tc.valid {
				require.Error(t, err)
				require.Nil(t, info.Verification)
				return
			}
			require.NoError(t, err)
			require.Equal(t, 2, calls)
			require.NotNil(t, info.Verification)
			require.Equal(t, "pa-mall-123", info.Verification.MerchantID)
			require.Equal(t, tc.wantMode, info.Verification.Mode)
			require.Equal(t, storecenter.MerchantSupplierNamespace, info.Verification.Namespace)
			require.Equal(t, storecenter.PartitionCN, info.Verification.IdentityPartition)
			require.Zero(t, info.Verification.RegionID)
			require.Equal(t, []string{"bg.glo.goods.add"}, info.Verification.Scopes)
			require.Nil(t, info.SupplierID, "TEMU is not a SHEIN credential variant")
			projection, err := json.Marshal(info)
			require.NoError(t, err)
			require.NotContains(t, string(projection), "test-token")
			require.NotContains(t, string(projection), "test-secret")
		})
	}
}

func TestOfficialIdentityRedirectIsRejectedWithoutFollowingAnotherOrigin(t *testing.T) {
	calls := 0
	c := identityClient(t, storecenter.PartitionCN, roundTripFunc(func(*http.Request) (*http.Response, error) {
		calls++
		response := jsonResponse("redirect")
		response.StatusCode = http.StatusTemporaryRedirect
		response.Header.Set("Location", "https://another-origin.invalid/openapi/router")
		return response, nil
	}))
	_, err := c.QueryStore(context.Background(), tokenCredential())
	require.ErrorIs(t, err, storecenter.ErrOfficialConnectionUnavailable)
	require.Equal(t, 1, calls)
}

func TestLocalVerificationUsesOriginalPartitionAndPreservesLongMallID(t *testing.T) {
	for partition, host := range map[storecenter.OfficialIdentityPartition]string{
		storecenter.PartitionUS:     "openapi-b-us.temu.com",
		storecenter.PartitionEU:     "openapi-b-eu.temu.com",
		storecenter.PartitionGlobal: "openapi-b-global.temu.com",
	} {
		t.Run(string(partition), func(t *testing.T) {
			calls := 0
			c := identityClient(t, partition, roundTripFunc(func(r *http.Request) (*http.Response, error) {
				calls++
				require.Equal(t, host, r.URL.Host)
				var payload map[string]json.RawMessage
				require.NoError(t, json.NewDecoder(r.Body).Decode(&payload))
				require.JSONEq(t, `"bg.open.accesstoken.info.get"`, string(payload["type"]))
				require.NotContains(t, payload, "mallId")
				require.NotContains(t, payload, "siteId")
				return jsonResponse(localIdentityResponse), nil
			}))
			info, err := c.QueryStore(context.Background(), tokenCredential())
			require.NoError(t, err)
			require.Equal(t, 1, calls)
			require.Equal(t, "9223372036854775806", info.Verification.MerchantID)
			require.Equal(t, storecenter.MerchantLocal, info.Verification.Mode)
			require.Equal(t, int64(211), info.Verification.RegionID)
			require.Equal(t, partition, info.Verification.IdentityPartition, "region must not change application routing")
		})
	}
}

func TestIdentityFailureDoesNotProbeOtherGatewaysOrLeakProviderText(t *testing.T) {
	responses := []string{
		`{"success":false,"errorMsg":"test-token test-secret-0123456789"}`,
		`{"result":{"uniqueId":"123"}}`,
		`{"success":true,"result":null}`,
		strings.Replace(supplierIdentityResponse, `"uniqueId":"pa-mall-123"`, `"uniqueId":""`, 1),
		strings.Replace(supplierIdentityResponse, `"expiredTime":1760003600`, `"expiredTime":1760000000`, 1),
		strings.Replace(supplierIdentityResponse, `"apiScopeList":["bg.glo.goods.add"]`, `"apiScopeList":null`, 1),
		`{"success":true,"result":{}} trailing`,
		strings.Repeat("x", 2*1024*1024+1),
	}
	for i, body := range responses {
		t.Run(string(rune('a'+i)), func(t *testing.T) {
			calls := 0
			c := identityClient(t, storecenter.PartitionCN, roundTripFunc(func(*http.Request) (*http.Response, error) {
				calls++
				return jsonResponse(body), nil
			}))
			info, err := c.QueryStore(context.Background(), tokenCredential())
			require.Error(t, err)
			require.Nil(t, info.Verification)
			require.Equal(t, 1, calls)
			require.NotContains(t, err.Error(), "test-token")
			require.NotContains(t, err.Error(), "test-secret")
		})
	}
}

func TestLocalRejectsSemiAndMissingIdentityFacts(t *testing.T) {
	for _, body := range []string{
		strings.Replace(localIdentityResponse, `"mallType":100`, `"mallType":1`, 1),
		strings.Replace(localIdentityResponse, `"mallType":100`, `"mallType":null`, 1),
		strings.Replace(localIdentityResponse, `"regionId":211`, `"regionId":0`, 1),
		strings.Replace(localIdentityResponse, `"mallId":9223372036854775806`, `"mallId":1.1`, 1),
		strings.Replace(localIdentityResponse, `"mallId":9223372036854775806`, `"mallId":null`, 1),
	} {
		c := identityClient(t, storecenter.PartitionUS, roundTripFunc(func(*http.Request) (*http.Response, error) { return jsonResponse(body), nil }))
		info, err := c.QueryStore(context.Background(), tokenCredential())
		require.Error(t, err)
		require.Nil(t, info.Verification)
	}
}

func TestInvalidCredentialsAndCanceledRequestDoNotDispatch(t *testing.T) {
	calls := 0
	c := identityClient(t, storecenter.PartitionCN, roundTripFunc(func(*http.Request) (*http.Response, error) {
		calls++
		return nil, errors.New("transport must not run")
	}))
	for _, credential := range []storecenter.OfficialMerchantCredential{
		{},
		{AppID: "another-app", Temu: tokenCredential().Temu},
		{AppID: "test-app", OpenKeyID: "wrong-platform", Temu: tokenCredential().Temu},
		{AppID: "test-app", Temu: &storecenter.TemuMerchantCredential{AccessToken: strings.Repeat("x", 4097)}},
	} {
		_, err := c.QueryStore(context.Background(), credential)
		require.Error(t, err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := c.QueryStore(ctx, tokenCredential())
	require.Error(t, err)
	require.Zero(t, calls)
}

func TestClientRejectsAmbiguousOrWrongApplicationContract(t *testing.T) {
	base := storecenter.OfficialApplication{AppID: "test-app", Version: "test-v1", Platform: storecenter.PlatformTemu, IdentityPartition: storecenter.PartitionCN, AuthorizationMethod: storecenter.AuthorizationManualToken, SupportedModes: storecenter.ModesSemiManaged | storecenter.ModesFullyManaged}
	for _, change := range []func(*storecenter.OfficialApplication){
		func(a *storecenter.OfficialApplication) { a.Platform = storecenter.PlatformShein },
		func(a *storecenter.OfficialApplication) { a.IdentityPartition = "PA" },
		func(a *storecenter.OfficialApplication) { a.SupportedModes = storecenter.ModesSemiManaged },
		func(a *storecenter.OfficialApplication) { a.SupportedModes = storecenter.ModesLocal },
		func(a *storecenter.OfficialApplication) { a.AuthorizationMethod = storecenter.AuthorizationCallback },
		func(a *storecenter.OfficialApplication) { a.CallbackURL = "https://another.example" },
	} {
		app := base
		change(&app)
		_, err := NewOfficialClient(OfficialClientOptions{Application: app, AppSecret: "test-secret-0123456789"})
		require.ErrorIs(t, err, storecenter.ErrOfficialConnectionUnavailable)
	}
}
