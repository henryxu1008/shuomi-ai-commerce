package tiktokshop

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	authorization "task-processor/internal/storecenter/tiktokshop"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

var fixtureTime = time.Unix(1791800000, 0).UTC()

func newFixtureClient(t *testing.T, transport roundTripFunc) *OfficialClient {
	t.Helper()
	httpClient := &http.Client{}
	if transport != nil {
		httpClient.Transport = transport
	}
	client, err := NewOfficialClient(OfficialClientOptions{
		Application: authorization.Application{AppKey: "fixture-app", ServiceID: "fixture-service", Revision: "r1", CallbackURL: "https://example.org/callback"},
		AppSecret:   "fixture-secret-never-log",
		HTTPClient:  httpClient,
	})
	if err != nil {
		t.Fatal(err)
	}
	client.now = func() time.Time { return fixtureTime }
	return client
}

func jsonResponse(body string) *http.Response {
	return &http.Response{StatusCode: 200, Header: http.Header{"Content-Type": {"application/json"}}, Body: io.NopCloser(strings.NewReader(body))}
}

func tokenResponse(userType int, scopes string) string {
	return fmt.Sprintf(`{"code":0,"request_id":"fixture-request","data":{"access_token":"fixture-access","access_token_expire_in":%d,"refresh_token":"fixture-refresh","refresh_token_expire_in":%d,"open_id":"fixture-seller","seller_name":"Fixture","seller_base_region":"CN","user_type":%d,"granted_scopes":%s}}`, fixtureTime.Unix()+3600, fixtureTime.Unix()+86400, userType, scopes)
}

func TestOfficialSignatureVectorAndExactBody(t *testing.T) {
	query := url.Values{"app_key": {"29a39d"}, "timestamp": {"1623812664"}, "sign": {"ignored"}, "access_token": {"ignored"}}
	got, err := signature("/authorization/202309/shops", query, nil, "application/json", "e59af819cc")
	if err != nil || got != "b596b73e0cc6de07ac26f036364178ab16b0a907af13d43f0a0cd2345f582dc8" {
		t.Fatalf("official vector: %q, %v", got, err)
	}
	a, _ := signature("/product/202309/products", query, []byte(`{"title":"A"}`), "application/json", "e59af819cc")
	b, _ := signature("/product/202309/products", query, []byte(`{ "title": "A" }`), "application/json", "e59af819cc")
	if a == b {
		t.Fatal("JSON whitespace was discarded before signing")
	}
	first, _ := signature("/product/202309/images/upload", query, []byte("first multipart body"), "multipart/form-data; boundary=fixture", "e59af819cc")
	second, _ := signature("/product/202309/images/upload", query, []byte("second multipart body"), "multipart/form-data; boundary=fixture", "e59af819cc")
	if first != second {
		t.Fatal("multipart body included in signature")
	}
	query["app_key"] = []string{"one", "two"}
	if _, err := signature("/authorization/202309/shops", query, nil, "application/json", "e59af819cc"); err == nil {
		t.Fatal("ambiguous query values signed")
	}
}

func TestUSAuthorizationLinkUsesServiceID(t *testing.T) {
	client := newFixtureClient(t, nil)
	link, err := client.AuthorizationURL("single-use-server-state")
	if err != nil {
		t.Fatal(err)
	}
	parsed, _ := url.Parse(link)
	if parsed.Scheme != "https" || parsed.Host != "services.us.tiktokshop.com" || parsed.Path != "/open/authorize" || parsed.Query().Get("service_id") != "fixture-service" || parsed.Query().Get("state") != "single-use-server-state" || parsed.Query().Get("app_key") != "" {
		t.Fatalf("wrong US authorization link: %s", link)
	}
}

func TestDefaultTokenTransportCannotReplayMutatingGETOnReusedConnection(t *testing.T) {
	client := newFixtureClient(t, nil)
	transport, ok := client.authHTTP.Transport.(*http.Transport)
	if !ok || !transport.DisableKeepAlives || transport.Protocols == nil || transport.Protocols.HTTP2() || !transport.Protocols.HTTP1() {
		t.Fatal("one-time token GET permits transparent transport retries on a reused connection")
	}
	if http.DefaultTransport.(*http.Transport).DisableKeepAlives {
		t.Fatal("constructor modified shared default transport")
	}
}

func TestTokenExchangeAndRefreshUseAbsoluteExpiryWithoutRetry(t *testing.T) {
	for _, refresh := range []bool{false, true} {
		t.Run(fmt.Sprintf("refresh=%v", refresh), func(t *testing.T) {
			calls := 0
			client := newFixtureClient(t, func(req *http.Request) (*http.Response, error) {
				calls++
				if req.Method != "GET" || req.URL.Host != "auth.tiktok-shops.com" {
					t.Fatal("wrong auth endpoint")
				}
				q := req.URL.Query()
				if q.Get("app_key") != "fixture-app" || q.Get("app_secret") != "fixture-secret-never-log" {
					t.Fatal("wrong app binding")
				}
				if refresh {
					if req.URL.Path != "/api/v2/token/refresh" || q.Get("grant_type") != "refresh_token" || q.Get("refresh_token") != "fixture-refresh" {
						t.Fatal("wrong refresh protocol")
					}
				} else if req.URL.Path != "/api/v2/token/get" || q.Get("grant_type") != "authorized_code" || q.Get("auth_code") != "fixture-code" {
					t.Fatal("wrong one-time code protocol")
				}
				deadline, ok := req.Context().Deadline()
				if !ok || time.Until(deadline) > 5*time.Second {
					t.Fatal("missing bounded deadline")
				}
				return jsonResponse(tokenResponse(5, `["seller.authorization.info"]`)), nil
			})
			var tokens authorization.TokenSet
			var err error
			if refresh {
				tokens, err = client.Refresh(context.Background(), "fixture-refresh")
			} else {
				tokens, err = client.Exchange(context.Background(), "fixture-code")
			}
			if err != nil || calls != 1 || tokens.OpenID != "fixture-seller" || tokens.AccessExpiresAt.Unix() != fixtureTime.Unix()+3600 {
				t.Fatalf("token expiry/identity: %v, calls=%d, %v", tokens, calls, err)
			}
			encoded, _ := json.Marshal(tokens)
			for _, rendered := range []string{string(encoded), fmt.Sprint(tokens), fmt.Sprintf("%#v", tokens)} {
				if strings.Contains(rendered, "fixture-access") || strings.Contains(rendered, "fixture-refresh") {
					t.Fatal("credential exposed by ordinary rendering")
				}
			}
		})
	}
}

func TestUnusableTokenAndAmbiguousResponseAreRejected(t *testing.T) {
	for _, userType := range []int{1, 2, 3, 6} {
		client := newFixtureClient(t, func(*http.Request) (*http.Response, error) {
			return jsonResponse(tokenResponse(userType, `["seller.authorization.info"]`)), nil
		})
		if _, err := client.Exchange(context.Background(), "fixture-code"); !errors.Is(err, authorization.ErrAuthorizationRejected) {
			t.Fatalf("role %d: %v", userType, err)
		}
	}
	for _, body := range []string{
		`{"data":{}}`,
		strings.Replace(tokenResponse(0, `["seller.authorization.info"]`), `"user_type":0,`, "", 1),
		strings.Replace(tokenResponse(0, `["seller.authorization.info"]`), `"code":0`, `"code":9,"code":0`, 1),
		strings.Replace(tokenResponse(0, `["seller.authorization.info"]`), fmt.Sprint(fixtureTime.Unix()+3600), "3600", 1),
		tokenResponse(0, `["seller.authorization.info","seller.authorization.info"]`),
	} {
		client := newFixtureClient(t, func(*http.Request) (*http.Response, error) { return jsonResponse(body), nil })
		if _, err := client.Exchange(context.Background(), "fixture-code"); !errors.Is(err, authorization.ErrOutcomeUnknown) {
			t.Fatalf("ambiguous/expired token accepted: %v", err)
		}
	}
	client := newFixtureClient(t, func(*http.Request) (*http.Response, error) {
		return jsonResponse(tokenResponse(4, `["seller.product.write"]`)), nil
	})
	if _, err := client.Exchange(context.Background(), "fixture-code"); !errors.Is(err, authorization.ErrAuthorizationRejected) {
		t.Fatal("missing Shop authorization scope accepted")
	}
}

func TestInvalidAndCancelledInputsNeverDispatch(t *testing.T) {
	calls := 0
	client := newFixtureClient(t, func(*http.Request) (*http.Response, error) { calls++; return nil, errors.New("unexpected I/O") })
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := client.Exchange(ctx, "fixture-code"); !errors.Is(err, authorization.ErrInvalid) {
		t.Fatal("cancelled exchange accepted")
	}
	if _, err := client.Refresh(context.Background(), "token\nheader"); !errors.Is(err, authorization.ErrInvalid) {
		t.Fatal("control character accepted")
	}
	if _, err := client.Exchange(context.Background(), "null"); !errors.Is(err, authorization.ErrAuthorizationRejected) {
		t.Fatal("denial was sent for exchange")
	}
	if calls != 0 {
		t.Fatal("invalid input consumed provider credentials")
	}
	app := client.Application()
	for _, callback := range []string{"http://example.org/callback", "https://user@example.org/callback", "https://example.org/callback?code=leak", "https://example.org/callback#fragment"} {
		app.CallbackURL = callback
		if _, err := NewOfficialClient(OfficialClientOptions{Application: app, AppSecret: "fixture-secret"}); !errors.Is(err, authorization.ErrInvalid) {
			t.Fatalf("invalid callback admitted: %s", callback)
		}
	}
}

func TestTransportLossRedirectAndLimitNeverLeakOrRepeatTokenRequest(t *testing.T) {
	for _, scenario := range []string{"loss", "redirect", "oversize", "server"} {
		t.Run(scenario, func(t *testing.T) {
			calls := 0
			client := newFixtureClient(t, func(req *http.Request) (*http.Response, error) {
				calls++
				switch scenario {
				case "loss":
					return nil, errors.New("secret transport failure " + req.URL.String())
				case "redirect":
					return &http.Response{StatusCode: 302, Header: http.Header{"Location": {"https://example.org/leak"}}, Body: io.NopCloser(strings.NewReader(""))}, nil
				case "server":
					return &http.Response{StatusCode: 500, Body: io.NopCloser(strings.NewReader("secret provider message"))}, nil
				default:
					return jsonResponse(strings.Repeat("x", MaxAuthorizationResponseBytes+1)), nil
				}
			})
			_, err := client.Refresh(context.Background(), "fixture-refresh")
			if !errors.Is(err, authorization.ErrOutcomeUnknown) || calls != 1 {
				t.Fatalf("calls=%d error=%v", calls, err)
			}
			if strings.Contains(err.Error(), "fixture-") || strings.Contains(err.Error(), "secret") || strings.Contains(err.Error(), "https://") {
				t.Fatal("sensitive provider error leaked")
			}
		})
	}
}

func TestAuthorizedShopsAreSignedAndSelectedByExactUSCrossBorderIdentity(t *testing.T) {
	client := newFixtureClient(t, func(req *http.Request) (*http.Response, error) {
		if req.URL.Host != "open-api.tiktokglobalshop.com" || req.URL.Path != "/authorization/202309/shops" || req.Header.Get("x-tts-access-token") != "fixture-access" || req.URL.Query().Get("access_token") != "" {
			t.Fatal("wrong resource credential transport")
		}
		expected, _ := signature(req.URL.Path, req.URL.Query(), nil, "application/json", "fixture-secret-never-log")
		if req.URL.Query().Get("sign") != expected {
			t.Fatal("wrong signed query")
		}
		return jsonResponse(`{"code":0,"data":{"shops":[{"id":"uk-shop","code":"UKCODE","cipher":"uk-cipher","region":"GB","seller_type":"CROSS_BORDER"},{"id":"us-local","code":"LOCAL","cipher":"local-cipher","region":"US","seller_type":"LOCAL"},{"id":"us-cross","code":"USCODE","cipher":"us-cipher","region":"US","seller_type":"CROSS_BORDER"}]}}`), nil
	})
	shops, err := client.AuthorizedShops(context.Background(), "fixture-access")
	if err != nil {
		t.Fatal(err)
	}
	shop, err := authorization.SelectUSCrossBorderShop(shops, "USCODE")
	if err != nil || shop.ID != "us-cross" || shop.Cipher != "us-cipher" {
		t.Fatalf("selection: %v, %v", shop, err)
	}
	if _, err := authorization.SelectUSCrossBorderShop(shops, "LOCAL"); err == nil {
		t.Fatal("local seller accepted")
	}
	if _, err := authorization.SelectUSCrossBorderShop(append(shops, shop), "USCODE"); err == nil {
		t.Fatal("duplicate official identity accepted")
	}
	if _, err := authorization.SelectUSCrossBorderShop(shops, "us-cross"); err == nil {
		t.Fatal("internal ID silently treated as Seller Center code")
	}
}

func TestSignedDeauthorizationUsesRawBodyAndNeverCreatesAuthority(t *testing.T) {
	client := newFixtureClient(t, nil)
	body := []byte(`{"type":6,"shop_id":"us-cross","timestamp":1791800000,"tts_notification_id":"fixture-event","data":{"message":"secret provider message"}}`)
	mac := hmac.New(sha256.New, []byte("fixture-secret-never-log"))
	_, _ = mac.Write(append([]byte("fixture-app"), body...))
	signature := hex.EncodeToString(mac.Sum(nil))
	event, err := client.VerifyDeauthorization(body, signature)
	if err != nil || event.ShopID != "us-cross" || event.NotificationID != "fixture-event" || event.BodyHash == "" {
		t.Fatalf("valid signed event: %v, %v", event, err)
	}
	for _, bad := range []string{"Bearer " + signature, strings.ToUpper(signature), strings.Repeat("0", 64)} {
		if _, err := client.VerifyDeauthorization(body, bad); err == nil {
			t.Fatal("wrong signature accepted")
		}
	}
	if _, err := client.VerifyDeauthorization(append(body, ' '), signature); err == nil {
		t.Fatal("raw body modification accepted")
	}
	if strings.Contains(fmt.Sprint(event), "secret provider") {
		t.Fatal("provider message included in event projection")
	}
	for _, raw := range [][]byte{
		[]byte(`{"type":6,"type":7,"shop_id":"us-cross","timestamp":1791800000}`),
		[]byte(`{"type":3,"shop_id":"us-cross","timestamp":1791800000}`),
		[]byte(`{"type":6,"shop_id":"","timestamp":1791800000}`),
		[]byte(strings.Repeat(" ", MaxWebhookBodyBytes+1)),
	} {
		mac := hmac.New(sha256.New, []byte("fixture-secret-never-log"))
		_, _ = mac.Write(append([]byte("fixture-app"), raw...))
		if _, err := client.VerifyDeauthorization(raw, hex.EncodeToString(mac.Sum(nil))); err == nil {
			t.Fatal("invalid signed lifecycle event admitted")
		}
	}
}
