package shein

import (
	"bytes"
	"context"
	"crypto/aes"
	"crypto/cipher"
	"encoding/base64"
	"encoding/json"
	"fmt"
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
func testClient(t *testing.T, transport http.RoundTripper) *OfficialClient {
	t.Helper()
	client, err := NewOfficialClient(OfficialClientOptions{Application: storecenter.OfficialApplication{AppID: "test-app", Version: "v1", CallbackURL: "https://app.example/workbench/stores/shein/callback"}, AppSecret: "0123456789abcdef0123456789abcdef", APIOrigin: "https://openapi.sheincorp.com", HTTPClient: &http.Client{Transport: transport}})
	require.NoError(t, err)
	client.now = func() time.Time { return time.UnixMilli(1752560461085) }
	client.random = bytes.NewReader(make([]byte, 32))
	return client
}
func wireEncryptedSecret(t *testing.T, secret string) string {
	t.Helper()
	block, err := aes.NewCipher([]byte("0123456789abcdef"))
	require.NoError(t, err)
	data := []byte(secret)
	pad := aes.BlockSize - len(data)%aes.BlockSize
	data = append(data, bytes.Repeat([]byte{byte(pad)}, pad)...)
	cipher.NewCBCEncrypter(block, []byte("space-station-de")).CryptBlocks(data, data)
	return base64.StdEncoding.EncodeToString(data)
}
func TestOfficialExchangeUsesAppSignatureAndValidatesReturnedBindings(t *testing.T) {
	secret := wireEncryptedSecret(t, "merchant-secret")
	calls := 0
	client := testClient(t, roundTripFunc(func(r *http.Request) (*http.Response, error) {
		calls++
		require.Equal(t, http.MethodPost, r.Method)
		require.Equal(t, "/open-api/auth/get-by-token", r.URL.Path)
		require.Equal(t, "test-app", r.Header.Get("x-lt-appid"))
		require.Empty(t, r.Header.Get("x-lt-openKeyId"))
		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		require.JSONEq(t, `{"tempToken":"temporary"}`, string(body))
		signature := r.Header.Get("x-lt-signature")
		require.Len(t, signature, 93)
		decoded, err := base64.StdEncoding.DecodeString(signature[5:])
		require.NoError(t, err)
		require.Len(t, decoded, 64)
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{"code":"0","info":{"secretKey":"` + secret + `","openKeyId":"open-key","appid":"test-app","state":"original-state","supplierId":123}}`)), Header: make(http.Header)}, nil
	}))
	credential, err := client.Exchange(context.Background(), "temporary", "original-state")
	require.NoError(t, err)
	require.Equal(t, "merchant-secret", credential.SecretKey)
	require.Equal(t, "123", credential.SupplierID)
	require.Equal(t, 1, calls)
	_, err = client.Exchange(context.Background(), "temporary", "wrong-state")
	require.ErrorIs(t, err, storecenter.ErrOfficialExchangeUnknown)
	url, err := client.AuthorizationURL("original-state")
	require.NoError(t, err)
	require.Contains(t, url, "https://openapi-sem.sheincorp.com/#/empower?appid=test-app&")
	require.Contains(t, url, "state=original-state")
}

func TestOfficialExchangePreservesBusinessModeEvidence(t *testing.T) {
	for _, mode := range []any{nil, "", "POP-US", "POP-GLOBAL", "SFS", "FUTURE-MODE", true, " bad", "a\tb", strings.Repeat("X", 65)} {
		t.Run(fmt.Sprint(mode), func(t *testing.T) {
			info := map[string]any{"secretKey": wireEncryptedSecret(t, "merchant-secret"), "openKeyId": "open-key", "appid": "test-app", "state": "state", "supplierId": 123, "supplierBusinessMode": mode}
			raw, err := json.Marshal(map[string]any{"code": "0", "info": info})
			require.NoError(t, err)
			client := testClient(t, roundTripFunc(func(*http.Request) (*http.Response, error) {
				return &http.Response{StatusCode: 200, Body: io.NopCloser(bytes.NewReader(raw)), Header: make(http.Header)}, nil
			}))
			credential, err := client.Exchange(context.Background(), "temporary", "state")
			value, ok := mode.(string)
			invalid := mode != nil && !ok || ok && (value == " bad" || strings.Contains(value, "\t") || len(value) > 64)
			if invalid {
				require.ErrorIs(t, err, storecenter.ErrOfficialExchangeUnknown)
				return
			}
			require.NoError(t, err)
			require.Equal(t, value, credential.SheinBusinessMode)
		})
	}
}

func TestOfficialQueryDoesNotOverwriteAuthorizationBusinessMode(t *testing.T) {
	for _, test := range []struct {
		auth, query string
		unavailable bool
	}{{"", "", false}, {"POP-US", "", false}, {"POP-US", "POP-US", false}, {"", "POP-US", true}, {"POP-US", "SFS", true}, {"", " bad", true}} {
		t.Run(test.auth+"/"+test.query, func(t *testing.T) {
			raw, err := json.Marshal(map[string]any{"code": "0", "info": map[string]any{"storeInfo": map[string]any{"supplierId": 123, "supplierBusinessMode": test.query}}})
			require.NoError(t, err)
			client := testClient(t, roundTripFunc(func(*http.Request) (*http.Response, error) {
				return &http.Response{StatusCode: 200, Body: io.NopCloser(bytes.NewReader(raw)), Header: make(http.Header)}, nil
			}))
			credential := goodsCredential()
			credential.SheinBusinessMode = test.auth
			_, err = client.QueryStore(context.Background(), credential)
			if test.unavailable {
				require.ErrorIs(t, err, storecenter.ErrOfficialConnectionUnavailable)
			} else {
				require.NoError(t, err)
			}
			require.Equal(t, test.auth, credential.SheinBusinessMode)
		})
	}
}
func TestOfficialQueryKeepsOptionalStoreFactsMissingAndBindsSupplierWhenPresent(t *testing.T) {
	response := `{"code":"0","info":{"storeProductQuota":{"totalLimit":9999},"storeInfo":{"storeName":null,"storeStatus":null}}}`
	client := testClient(t, roundTripFunc(func(r *http.Request) (*http.Response, error) {
		require.Equal(t, "/open-api/openapi-business-backend/query-store-info", r.URL.Path)
		require.Equal(t, "open-key", r.Header.Get("x-lt-openKeyId"))
		require.Empty(t, r.Header.Get("x-lt-appid"))
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(response)), Header: make(http.Header)}, nil
	}))
	credential := storecenter.OfficialMerchantCredential{AppID: "test-app", OpenKeyID: "open-key", SecretKey: "merchant-secret", SupplierID: "123"}
	info, err := client.QueryStore(context.Background(), credential)
	require.NoError(t, err)
	require.Nil(t, info.StoreName)
	require.Nil(t, info.StoreStatus)
	require.Nil(t, info.SupplierID)
	response = `{"code":"0","info":{"storeInfo":{"supplierId":456}}}`
	_, err = client.QueryStore(context.Background(), credential)
	require.ErrorIs(t, err, storecenter.ErrOfficialConnectionUnavailable)
	response = `{"code":"openapi00004","msg":"provider supplied private details"}`
	_, err = client.QueryStore(context.Background(), credential)
	require.ErrorIs(t, err, storecenter.ErrOfficialCredentialExpired)
	require.NotContains(t, err.Error(), "private details")
}
func TestOfficialExchangeDoesNotRetryUnknownResponseOrFollowRedirect(t *testing.T) {
	calls := 0
	client := testClient(t, roundTripFunc(func(r *http.Request) (*http.Response, error) {
		calls++
		return &http.Response{StatusCode: 302, Body: io.NopCloser(strings.NewReader("")), Header: http.Header{"Location": []string{"https://other.example/?secret=private"}}}, nil
	}))
	_, err := client.Exchange(context.Background(), "temporary", "state")
	require.ErrorIs(t, err, storecenter.ErrOfficialExchangeUnknown)
	require.Equal(t, 1, calls)
	_, err = NewOfficialClient(OfficialClientOptions{APIOrigin: "https://other.example"})
	require.Error(t, err)
}
