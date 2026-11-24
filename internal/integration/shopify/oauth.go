package shopify

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"net/http"
	"net/url"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"sync/atomic"
	"time"
)

const (
	accessTokenPath       = "/admin/oauth/access_token"
	fiveMinutesInSeconds  = 300
	maxTokenResponseBytes = 64 << 10
)

var (
	ErrInvalidCallback    = errors.New("invalid Shopify authorization callback")
	ErrShopOwnerRequired  = errors.New("Shopify store owner with product access is required")
	ErrTokenUnavailable   = errors.New("Shopify token could not be obtained")
	oauthParameterPattern = regexp.MustCompile(`^[a-zA-Z0-9_-]{1,256}$`)
)

type OAuthClient struct {
	http         *http.Client
	clientID     string
	clientSecret string
	callbackURL  string
}

func (*OAuthClient) String() string   { return "Shopify OAuth client [redacted]" }
func (*OAuthClient) GoString() string { return "Shopify OAuth client [redacted]" }

func NewOAuthClient(transport *http.Client, clientID, clientSecret, callbackURL string) (*OAuthClient, error) {
	callback, err := url.Parse(callbackURL)
	if err != nil || callback.Scheme != "https" || callback.Hostname() == "" || callback.User != nil || callback.Fragment != "" ||
		callback.RawQuery != "" || callback.ForceQuery || !oauthParameterPattern.MatchString(clientID) || !validToken(clientSecret) {
		return nil, ErrInvalidRequest
	}
	return &OAuthClient{http: boundedClient(transport), clientID: clientID, clientSecret: clientSecret, callbackURL: callbackURL}, nil
}

// AuthorizationURL is used after Shopify has initiated installation for the
// selected store. Store Center must create the browser-bound state first.
func (c *OAuthClient) AuthorizationURL(shop, state string) (string, error) {
	if c == nil || c.http == nil || !shopDomainPattern.MatchString(shop) || !oauthParameterPattern.MatchString(state) {
		return "", ErrInvalidRequest
	}
	query := url.Values{
		"client_id": {c.clientID}, "scope": {productWriteScope}, "redirect_uri": {c.callbackURL},
		"state": {state}, "grant_options[]": {"per-user"},
	}
	return "https://" + shop + "/admin/oauth/authorize?" + query.Encode(), nil
}

// OwnerCallback represents a signed callback, not a locally authorized
// installation. Store Center must claim the original pairing before exchange.
type OwnerCallback struct {
	client *OAuthClient
	shop   string
	code   string
	used   *atomic.Bool
}

func (p OwnerCallback) Shop() string   { return p.shop }
func (OwnerCallback) String() string   { return "Shopify authorization callback [redacted]" }
func (OwnerCallback) GoString() string { return "Shopify authorization callback [redacted]" }

func (c *OAuthClient) VerifyCallback(rawQuery, expectedState string, now time.Time) (OwnerCallback, error) {
	query, err := c.verifySignedQuery(rawQuery, now)
	if err != nil || !oauthParameterPattern.MatchString(expectedState) {
		return OwnerCallback{}, ErrInvalidCallback
	}
	shop, state, code := query.Get("shop"), query.Get("state"), query.Get("code")
	if !hmac.Equal([]byte(state), []byte(expectedState)) || !oauthParameterPattern.MatchString(code) {
		return OwnerCallback{}, ErrInvalidCallback
	}
	return OwnerCallback{client: c, shop: shop, code: code, used: new(atomic.Bool)}, nil
}

// VerifyLaunch accepts Shopify's signed app launch. The browser must
// separately present its Workbench pairing context before OAuth can start.
func (c *OAuthClient) VerifyLaunch(rawQuery string, now time.Time) (string, error) {
	query, err := c.verifySignedQuery(rawQuery, now)
	if err != nil || query.Has("code") || query.Has("state") {
		return "", ErrInvalidCallback
	}
	return query.Get("shop"), nil
}

func (c *OAuthClient) verifySignedQuery(rawQuery string, now time.Time) (url.Values, error) {
	if c == nil || c.http == nil || len(rawQuery) > 8192 || now.IsZero() {
		return nil, ErrInvalidCallback
	}
	query, err := url.ParseQuery(rawQuery)
	if err != nil || len(query) > 16 {
		return nil, ErrInvalidCallback
	}
	keys := make([]string, 0, len(query))
	for key, values := range query {
		if !oauthParameterPattern.MatchString(key) || len(values) != 1 || strings.ContainsAny(values[0], "&\x00\r\n") {
			return nil, ErrInvalidCallback
		}
		if key != "hmac" {
			keys = append(keys, key)
		}
	}
	signature, err := hex.DecodeString(query.Get("hmac"))
	timestamp, timestampErr := strconv.ParseInt(query.Get("timestamp"), 10, 64)
	if err != nil || len(signature) != sha256.Size || !shopDomainPattern.MatchString(query.Get("shop")) || timestampErr != nil ||
		timestamp < now.Unix()-fiveMinutesInSeconds || timestamp > now.Unix()+fiveMinutesInSeconds {
		return nil, ErrInvalidCallback
	}
	slices.Sort(keys)
	canonical := make([]string, 0, len(keys))
	for _, key := range keys {
		canonical = append(canonical, key+"="+query.Get(key))
	}
	digest := hmac.New(sha256.New, []byte(c.clientSecret))
	_, _ = digest.Write([]byte(strings.Join(canonical, "&")))
	if !hmac.Equal(signature, digest.Sum(nil)) {
		return nil, ErrInvalidCallback
	}
	return query, nil
}

type AccessToken struct {
	Value     string `json:"-"`
	ExpiresAt time.Time
}

func (AccessToken) String() string   { return "Shopify access token [redacted]" }
func (AccessToken) GoString() string { return "Shopify access token [redacted]" }

type OwnerAccessToken struct{ AccessToken }

func (c *OAuthClient) ExchangeOwnerCode(ctx context.Context, proof OwnerCallback) (OwnerAccessToken, error) {
	if c == nil || proof.client != c || proof.used == nil || ctx == nil || ctx.Err() != nil || !proof.used.CompareAndSwap(false, true) {
		return OwnerAccessToken{}, ErrInvalidCallback
	}
	body := url.Values{"client_id": {c.clientID}, "client_secret": {c.clientSecret}, "code": {proof.code}}
	now := time.Now()
	raw, err := postShopify(ctx, c.http, proof.shop, accessTokenPath, "", "application/x-www-form-urlencoded", []byte(body.Encode()), maxTokenResponseBytes)
	if err != nil {
		return OwnerAccessToken{}, ErrTokenUnavailable
	}
	var response struct {
		Token      string `json:"access_token"`
		ExpiresIn  int64  `json:"expires_in"`
		Scope      string `json:"scope"`
		UserScope  string `json:"associated_user_scope"`
		Associated *struct {
			AccountOwner bool `json:"account_owner"`
		} `json:"associated_user"`
	}
	if !decodeAdminJSON(raw, &response) || !validToken(response.Token) || response.ExpiresIn <= 0 || response.ExpiresIn > 24*60*60 {
		return OwnerAccessToken{}, ErrTokenUnavailable
	}
	if response.Associated == nil || !response.Associated.AccountOwner || !writeScope(response.Scope) || !writeScope(response.UserScope) {
		return OwnerAccessToken{}, ErrShopOwnerRequired
	}
	return OwnerAccessToken{AccessToken{Value: response.Token, ExpiresAt: now.Add(time.Duration(response.ExpiresIn) * time.Second)}}, nil
}

func writeScope(scope string) bool {
	return slices.Contains(strings.Split(scope, ","), productWriteScope)
}
