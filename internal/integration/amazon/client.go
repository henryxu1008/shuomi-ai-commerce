// Package amazon implements the narrow US Amazon SP-API product transport.
package amazon

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	USMarketplace = "ATVPDKIKX0DER"
	spAPIOrigin   = "https://sellingpartnerapi-na.amazon.com"
	tokenEndpoint = "https://api.amazon.com/auth/o2/token"
	listingsPath  = "/listings/2021-08-01/items/"
	maxResponse   = 1 << 20
	maxPayload    = 256 << 10
	requestLimit  = 10 * time.Second
)

var (
	ErrInvalid           = errors.New("invalid Amazon request")
	ErrUnauthorized      = errors.New("Amazon authorization unavailable")
	ErrUnavailable       = errors.New("Amazon operation unavailable")
	ErrSellerIneligible  = errors.New("Amazon US seller participation unavailable")
	ErrNotFound          = errors.New("Amazon listing not found")
	ErrSubmissionUnknown = errors.New("Amazon submission result unknown")
)

type Options struct {
	ClientID, ClientSecret, RedirectURI, UserAgent string
	HTTPClient                                     *http.Client
}

type Client struct {
	clientID, secret, redirectURI, userAgent string
	http                                     http.Client
}

func NewClient(options Options) (*Client, error) {
	callback, err := url.Parse(options.RedirectURI)
	if err != nil || callback.Scheme != "https" || callback.Hostname() == "" || callback.User != nil ||
		callback.Path == "" || callback.RawQuery != "" || callback.ForceQuery || callback.Fragment != "" ||
		!visibleASCII(options.ClientID, 256) || !visibleASCII(options.ClientSecret, 2048) ||
		!validUserAgent(options.UserAgent) {
		return nil, ErrInvalid
	}
	var httpClient http.Client
	if options.HTTPClient != nil {
		httpClient = *options.HTTPClient
	}
	httpClient.Jar = nil
	httpClient.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	if httpClient.Timeout <= 0 || httpClient.Timeout > requestLimit {
		httpClient.Timeout = requestLimit
	}
	return &Client{options.ClientID, options.ClientSecret, options.RedirectURI, options.UserAgent, httpClient}, nil
}

type Tokens struct {
	AccessToken    string `json:"-"`
	RefreshToken   string `json:"-"`
	AccessLifetime time.Duration
}

func (c *Client) ExchangeCode(ctx context.Context, code string) (Tokens, error) {
	if !visibleASCII(code, 8192) {
		return Tokens{}, ErrInvalid
	}
	return c.exchange(ctx, url.Values{
		"grant_type": {"authorization_code"}, "code": {code}, "redirect_uri": {c.redirectURI},
	})
}

func (c *Client) RefreshAccess(ctx context.Context, refreshToken string) (Tokens, error) {
	if !visibleASCII(refreshToken, 8192) {
		return Tokens{}, ErrInvalid
	}
	return c.exchange(ctx, url.Values{"grant_type": {"refresh_token"}, "refresh_token": {refreshToken}})
}

func (c *Client) exchange(ctx context.Context, form url.Values) (Tokens, error) {
	form.Set("client_id", c.clientID)
	form.Set("client_secret", c.secret)
	raw, metadata, err := c.request(ctx, http.MethodPost, tokenEndpoint, "application/x-www-form-urlencoded", "", []byte(form.Encode()))
	if err != nil {
		return Tokens{}, err
	}
	if metadata.status == http.StatusBadRequest {
		var result struct {
			Code string `json:"error"`
		}
		if json.Unmarshal(raw, &result) == nil && (result.Code == "invalid_grant" || result.Code == "invalid_client") {
			return Tokens{}, ErrUnauthorized
		}
	}
	if err := responseStatus(metadata.status); err != nil {
		return Tokens{}, err
	}
	var result struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		TokenType    string `json:"token_type"`
		ExpiresIn    int    `json:"expires_in"`
	}
	if json.Unmarshal(raw, &result) != nil || !visibleASCII(result.AccessToken, 8192) ||
		!strings.EqualFold(result.TokenType, "Bearer") || result.ExpiresIn <= 0 || result.ExpiresIn > 3600 {
		return Tokens{}, ErrUnavailable
	}
	if result.RefreshToken == "" && form.Get("grant_type") == "refresh_token" {
		result.RefreshToken = form.Get("refresh_token")
	}
	if !visibleASCII(result.RefreshToken, 8192) {
		return Tokens{}, ErrUnavailable
	}
	return Tokens{result.AccessToken, result.RefreshToken, time.Duration(result.ExpiresIn) * time.Second}, nil
}

// VerifyUSSeller checks the site and the seller path with the same grant. A
// successful probe persists no listing data and permits an empty seller store.
func (c *Client) VerifyUSSeller(ctx context.Context, accessToken, sellerID string) error {
	if !validIdentifier(sellerID, 64) {
		return ErrInvalid
	}
	raw, metadata, err := c.api(ctx, http.MethodGet, "/sellers/v1/marketplaceParticipations", nil, accessToken, nil)
	if err != nil {
		return err
	}
	if err := responseStatus(metadata.status); err != nil {
		return err
	}
	var result struct {
		Payload []struct {
			Marketplace struct {
				ID string
			}
			Participation struct {
				IsParticipating, HasSuspendedListings *bool
			}
		}
	}
	if json.Unmarshal(raw, &result) != nil || result.Payload == nil {
		return ErrUnavailable
	}
	var matches int
	for _, item := range result.Payload {
		if item.Marketplace.ID != USMarketplace {
			continue
		}
		matches++
		p := item.Participation
		if p.IsParticipating == nil || !*p.IsParticipating || p.HasSuspendedListings == nil || *p.HasSuspendedListings {
			return ErrSellerIneligible
		}
	}
	if matches != 1 {
		return ErrSellerIneligible
	}
	query := url.Values{"marketplaceIds": {USMarketplace}, "pageSize": {"1"}, "includedData": {"summaries"}}
	raw, metadata, err = c.api(ctx, http.MethodGet, listingsPath+sellerID, query, accessToken, nil)
	if err != nil {
		return err
	}
	if err := responseStatus(metadata.status); err != nil {
		return err
	}
	var probe struct {
		NumberOfResults *int
		Items           []json.RawMessage
	}
	if json.Unmarshal(raw, &probe) != nil || probe.NumberOfResults == nil ||
		*probe.NumberOfResults < len(probe.Items) || probe.Items == nil || len(probe.Items) > 1 {
		return ErrUnavailable
	}
	return nil
}

type responseMetadata struct {
	status    int
	requestID string
}

func (c *Client) api(ctx context.Context, method, path string, query url.Values, token string, body []byte) ([]byte, responseMetadata, error) {
	if !visibleASCII(token, 8192) {
		return nil, responseMetadata{}, ErrInvalid
	}
	target := spAPIOrigin + path
	if len(query) > 0 {
		target += "?" + query.Encode()
	}
	return c.request(ctx, method, target, "application/json", token, body)
}

// request never follows redirects or makes provider messages part of an error.
// Removing GetBody also prevents transport replay of a request carrying a body.
func (c *Client) request(ctx context.Context, method, target, contentType, accessToken string, body []byte) ([]byte, responseMetadata, error) {
	if ctx == nil || len(body) > maxPayload {
		return nil, responseMetadata{}, ErrInvalid
	}
	req, err := http.NewRequestWithContext(ctx, method, target, bytes.NewReader(body))
	if err != nil {
		return nil, responseMetadata{}, ErrInvalid
	}
	req.GetBody = nil
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", c.userAgent)
	if len(body) != 0 {
		req.Header.Set("Content-Type", contentType)
	}
	if accessToken != "" {
		req.Header.Set("X-Amz-Access-Token", accessToken)
		req.Header.Set("X-Amz-Date", time.Now().UTC().Format("20060102T150405Z"))
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, responseMetadata{}, ErrUnavailable
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(resp.Body, maxResponse+1))
	if err != nil || len(raw) > maxResponse {
		return nil, responseMetadata{}, ErrUnavailable
	}
	return raw, responseMetadata{resp.StatusCode, resp.Header.Get("X-Amzn-RequestId")}, nil
}

func responseStatus(status int) error {
	switch status {
	case http.StatusOK:
		return nil
	case http.StatusUnauthorized, http.StatusForbidden:
		return ErrUnauthorized
	default:
		return ErrUnavailable
	}
}

func visibleASCII(value string, max int) bool {
	if value == "" || len(value) > max {
		return false
	}
	for _, char := range value {
		if char < '!' || char > '~' {
			return false
		}
	}
	return true
}

func validUserAgent(value string) bool {
	return strings.TrimSpace(value) != "" && visibleASCII(strings.ReplaceAll(value, " ", "!"), 500)
}

func validIdentifier(value string, max int) bool {
	if value == "" || len(value) > max {
		return false
	}
	for _, char := range value {
		if !(char >= 'A' && char <= 'Z' || char >= 'a' && char <= 'z' ||
			char >= '0' && char <= '9' || char == '-' || char == '_') {
			return false
		}
	}
	return true
}
