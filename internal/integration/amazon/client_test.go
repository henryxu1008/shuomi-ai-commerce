package amazon

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"
)

type transportFunc func(*http.Request) (*http.Response, error)

func (f transportFunc) RoundTrip(request *http.Request) (*http.Response, error) { return f(request) }

func response(status int, body string) *http.Response {
	return &http.Response{StatusCode: status, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(body))}
}

func clientWithTransport(t *testing.T, transport transportFunc) *Client {
	t.Helper()
	client, err := NewClient(Options{
		ClientID: "fixture-client", ClientSecret: "fixture-secret", RedirectURI: "https://app.example.org/oauth/amazon/redirect",
		UserAgent: "Shuomi/1.0 (Language=Go)", HTTPClient: &http.Client{Transport: transport, Timeout: time.Second},
	})
	if err != nil {
		t.Fatal(err)
	}
	return client
}

func TestExchangeAndVerifySellerAllowEmptyStore(t *testing.T) {
	var requests int
	client := clientWithTransport(t, func(request *http.Request) (*http.Response, error) {
		requests++
		switch requests {
		case 1:
			if request.Method != http.MethodPost || request.URL.String() != "https://api.amazon.com/auth/o2/token" || request.ParseForm() != nil ||
				request.PostForm.Get("grant_type") != "authorization_code" || request.PostForm.Get("code") != "fixture-code" ||
				request.PostForm.Get("client_secret") != "fixture-secret" || request.PostForm.Get("redirect_uri") != "https://app.example.org/oauth/amazon/redirect" {
				t.Fatal("incorrect code exchange")
			}
			return response(200, `{"access_token":"fixture-access","refresh_token":"fixture-refresh","token_type":"bearer","expires_in":3600}`), nil
		case 2:
			if request.URL.Host != "sellingpartnerapi-na.amazon.com" || request.URL.Path != "/sellers/v1/marketplaceParticipations" || request.Header.Get("X-Amz-Access-Token") != "fixture-access" {
				t.Fatal("incorrect marketplace request")
			}
			return response(200, `{"payload":[{"marketplace":{"id":"ATVPDKIKX0DER"},"participation":{"isParticipating":true,"hasSuspendedListings":false}}]}`), nil
		case 3:
			if request.URL.Path != "/listings/2021-08-01/items/FIXTURESELLER1" || request.URL.Query().Get("pageSize") != "1" ||
				request.URL.Query().Get("marketplaceIds") != USMarketplace || request.Header.Get("X-Amz-Access-Token") != "fixture-access" {
				t.Fatal("incorrect seller access probe")
			}
			return response(200, `{"numberOfResults":0,"items":[]}`), nil
		}
		t.Fatal("unexpected request")
		return nil, errors.New("unexpected request")
	})
	tokens, err := client.ExchangeCode(context.Background(), "fixture-code")
	if err != nil || tokens.RefreshToken != "fixture-refresh" {
		t.Fatalf("exchange: %v", err)
	}
	if err := client.VerifyUSSeller(context.Background(), tokens.AccessToken, "FIXTURESELLER1"); err != nil || requests != 3 {
		t.Fatalf("verification: requests=%d, error=%v", requests, err)
	}
}

func TestVerifySellerRequiresBothMarketplaceAndSellerAccess(t *testing.T) {
	t.Run("missing explicit seller eligibility", func(t *testing.T) {
		calls := 0
		client := clientWithTransport(t, func(_ *http.Request) (*http.Response, error) {
			calls++
			return response(200, `{"payload":[{"marketplace":{"id":"ATVPDKIKX0DER"},"participation":{"isParticipating":true}}]}`), nil
		})
		if err := client.VerifyUSSeller(context.Background(), "fixture-access", "FIXTURESELLER1"); !errors.Is(err, ErrSellerIneligible) || calls != 1 {
			t.Fatalf("calls=%d, error=%v", calls, err)
		}
	})
	t.Run("mismatched seller despite marketplace eligibility", func(t *testing.T) {
		calls := 0
		client := clientWithTransport(t, func(_ *http.Request) (*http.Response, error) {
			calls++
			if calls == 1 {
				return response(200, `{"payload":[{"marketplace":{"id":"ATVPDKIKX0DER"},"participation":{"isParticipating":true,"hasSuspendedListings":false}}]}`), nil
			}
			return response(403, `{"errors":[{"code":"Unauthorized","message":"sensitive provider details"}]}`), nil
		})
		err := client.VerifyUSSeller(context.Background(), "fixture-access", "FIXTURESELLER2")
		if !errors.Is(err, ErrUnauthorized) || strings.Contains(err.Error(), "sensitive") || calls != 2 {
			t.Fatalf("calls=%d, error=%v", calls, err)
		}
	})
}

func TestPreviewCannotBeConfusedWithSubmit(t *testing.T) {
	const sku = "SHUOMI-123"
	var calls int
	client := clientWithTransport(t, func(request *http.Request) (*http.Response, error) {
		calls++
		if request.Method != http.MethodPut || request.URL.Path != "/listings/2021-08-01/items/FIXTURESELLER1/"+sku ||
			request.URL.Query().Get("marketplaceIds") != USMarketplace || request.GetBody != nil {
			t.Fatal("incorrect listing request or request is replayable")
		}
		if calls == 1 {
			if request.URL.Query().Get("mode") != "VALIDATION_PREVIEW" {
				t.Fatal("preview must send the Amazon preview mode")
			}
			return response(200, `{"sku":"SHUOMI-123","status":"VALID","submissionId":"preview-id"}`), nil
		}
		if request.URL.Query().Has("mode") {
			t.Fatal("official submission must omit preview mode")
		}
		return response(200, `{"sku":"SHUOMI-123","status":"ACCEPTED","submissionId":"fixture-id"}`), nil
	})
	input := ListingInput{ProductType: "HOME", Attributes: []byte(`{"example_attribute":[]}`)}
	preview, err := client.Preview(context.Background(), "fixture-access", "FIXTURESELLER1", sku, input)
	if err != nil || !preview.Valid {
		t.Fatalf("preview=%+v, error=%v", preview, err)
	}
	receipt, err := client.Submit(context.Background(), "fixture-access", "FIXTURESELLER1", sku, input)
	if err != nil || !receipt.Accepted || receipt.SubmissionID != "fixture-id" || calls != 2 {
		t.Fatalf("receipt=%+v, calls=%d, error=%v", receipt, calls, err)
	}
}

func TestSubmitUncertaintyDoesNotRetryOrReturnReceipt(t *testing.T) {
	for _, tc := range []struct {
		name, body string
		status     int
		transport  error
	}{
		{name: "transport dropped", transport: io.ErrUnexpectedEOF},
		{name: "preview status returned", status: 200, body: `{"sku":"SHUOMI-123","status":"VALID","submissionId":"preview-id"}`},
		{name: "wrong sku", status: 200, body: `{"sku":"OTHER","status":"ACCEPTED","submissionId":"fixture-id"}`},
		{name: "missing accepted receipt", status: 200, body: `{"sku":"SHUOMI-123","status":"ACCEPTED"}`},
		{name: "throttled", status: 429, body: `{"errors":[{"code":"QuotaExceeded"}]}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			calls := 0
			client := clientWithTransport(t, func(_ *http.Request) (*http.Response, error) {
				calls++
				if tc.transport != nil {
					return nil, tc.transport
				}
				return response(tc.status, tc.body), nil
			})
			receipt, err := client.Submit(context.Background(), "fixture-access", "FIXTURESELLER1", "SHUOMI-123", ListingInput{
				ProductType: "HOME", Attributes: []byte(`{"example_attribute":[]}`),
			})
			if !errors.Is(err, ErrSubmissionUnknown) || receipt.SubmissionID != "" || receipt.Accepted || calls != 1 {
				t.Fatalf("receipt=%+v, calls=%d, error=%v", receipt, calls, err)
			}
		})
	}
}

func TestReadListingKeepsDiscoverableAndBuyableDistinct(t *testing.T) {
	client := clientWithTransport(t, func(request *http.Request) (*http.Response, error) {
		if request.Method != http.MethodGet || request.URL.Query().Get("includedData") != "summaries,issues" {
			t.Fatal("incorrect listing read")
		}
		return response(200, `{"sku":"SHUOMI-123","summaries":[{"marketplaceId":"ATVPDKIKX0DER","asin":"B000123456","status":["DISCOVERABLE"]}],"issues":[{"code":"99022","severity":"ERROR","message":"private provider detail","attributeNames":["brand"]}]}`), nil
	})
	got, err := client.GetListing(context.Background(), "fixture-access", "FIXTURESELLER1", "SHUOMI-123")
	if err != nil || got.ASIN != "B000123456" || !got.Discoverable || got.Buyable || len(got.Issues) != 1 || got.Issues[0].Code != "99022" {
		t.Fatalf("listing=%+v, error=%v", got, err)
	}
}

func TestListingNotFoundNeedsAmazonErrorEvidence(t *testing.T) {
	tests := []struct {
		name, body, requestID string
		want                  error
	}{
		{"official response", `{"errors":[{"code":"NOT_FOUND","message":"No listing found"}]}`, "fixture-request", ErrNotFound},
		{"missing Amazon request identifier", `{"errors":[{"code":"NOT_FOUND"}]}`, "", ErrUnavailable},
		{"proxy error", "<html>404</html>", "fixture-request", ErrUnavailable},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			client := clientWithTransport(t, func(_ *http.Request) (*http.Response, error) {
				result := response(404, tc.body)
				result.Header.Set("X-Amzn-RequestId", tc.requestID)
				return result, nil
			})
			_, err := client.GetListing(context.Background(), "fixture-access", "FIXTURESELLER1", "SHUOMI-123")
			if !errors.Is(err, tc.want) {
				t.Fatalf("error=%v, want=%v", err, tc.want)
			}
		})
	}
}

func TestSubmitRejectsRedirectAndKeepsRecognizedInvalidReceipt(t *testing.T) {
	for _, redirect := range []bool{false, true} {
		calls := 0
		client := clientWithTransport(t, func(_ *http.Request) (*http.Response, error) {
			calls++
			if redirect {
				result := response(307, "")
				result.Header.Set("Location", "https://other.example.org/")
				return result, nil
			}
			return response(200, `{"sku":"SHUOMI-123","status":"INVALID","submissionId":"fixture-invalid","issues":[{"code":"99022","severity":"ERROR","attributeNames":["brand"],"message":"do not expose"}]}`), nil
		})
		receipt, err := client.Submit(context.Background(), "fixture-access", "FIXTURESELLER1", "SHUOMI-123", ListingInput{
			ProductType: "HOME", Attributes: []byte(`{"example_attribute":[]}`),
		})
		if calls != 1 || receipt.Accepted {
			t.Fatalf("receipt=%+v, calls=%d", receipt, calls)
		}
		if redirect {
			if !errors.Is(err, ErrSubmissionUnknown) || receipt.SubmissionID != "" {
				t.Fatalf("redirect did not preserve uncertainty: receipt=%+v, error=%v", receipt, err)
			}
		} else if err != nil || receipt.SubmissionID != "fixture-invalid" || len(receipt.Issues) != 1 {
			t.Fatalf("rejection lost its receipt: receipt=%+v, error=%v", receipt, err)
		}
	}
}

func TestContinueAuthorizationUsesTheRegisteredAmazonApp(t *testing.T) {
	const appID = "amzn1.sellerapps.app.2eca283f-9f5a-4d13-b16c-474000000057"
	const target = "https://amazon.com/apps/authorize/confirm/" + appID
	const redirect = "https://app.example.org/oauth/amazon/redirect"
	tests := []struct {
		name, callback string
		allowed        bool
	}{
		{"official root", target, true},
		{"official Seller Central", strings.Replace(target, "amazon.com", "sellercentral.amazon.com", 1), true},
		{"external site", strings.Replace(target, "amazon.com", "other.example.org", 1), false},
		{"wrong app", target + "-other", false},
		{"url credentials", strings.Replace(target, "amazon.com", "user@amazon.com", 1), false},
		{"query override", target + "?state=other", false},
		{"encoded path", strings.Replace(target, "/confirm/", "/%63onfirm/", 1), false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := ContinueAuthorization(appID, redirect, tc.callback, "amazon-state", "workbench-state", false)
			if !tc.allowed {
				if !errors.Is(err, ErrInvalid) || got != "" {
					t.Fatalf("unexpected redirect: %q, %v", got, err)
				}
				return
			}
			parsed, parseErr := url.Parse(got)
			if err != nil || parseErr != nil || parsed.Scheme != "https" || parsed.Path != "/apps/authorize/confirm/"+appID ||
				parsed.Query().Get("state") != "workbench-state" || parsed.Query().Get("amazon_state") != "amazon-state" ||
				parsed.Query().Get("redirect_uri") != redirect || parsed.Query().Has("version") {
				t.Fatalf("redirect=%q, error=%v", got, err)
			}
		})
	}
}

func TestConsentAndConfirmationUseBetaOnlyWhenConfigured(t *testing.T) {
	const appID = "amzn1.sellerapps.app.2eca283f-9f5a-4d13-b16c-474000000057"
	for _, beta := range []bool{false, true} {
		consent, err := ConsentURL(appID, "initial-state", beta)
		if err != nil {
			t.Fatal(err)
		}
		start, err := url.Parse(consent)
		if err != nil || start.Host != "sellercentral.amazon.com" || start.Path != "/apps/authorize/consent" ||
			start.Query().Get("application_id") != appID || start.Query().Get("state") != "initial-state" ||
			start.Query().Has("version") != beta {
			t.Fatalf("invalid consent URL: %q", consent)
		}
		confirmation, err := ContinueAuthorization(appID, "https://app.example.org/oauth/amazon/redirect",
			"https://amazon.com/apps/authorize/confirm/"+appID, "amazon-state", "workbench-state", beta)
		if err != nil {
			t.Fatal(err)
		}
		got, err := url.Parse(confirmation)
		if err != nil || got.Query().Has("version") != beta || (beta && got.Query().Get("version") != "beta") {
			t.Fatalf("incorrect confirmation mode: %q", confirmation)
		}
	}
}
