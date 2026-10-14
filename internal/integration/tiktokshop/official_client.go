// Package tiktokshop implements the bounded official TikTok Shop protocol.
// It has no automatic token refresh, persistence, background work or public
// arbitrary-path request API. Store owns consent, credentials and token fencing.
package tiktokshop

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"mime"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	sigjson "sigs.k8s.io/json"

	authorization "task-processor/internal/storecenter/tiktokshop"
)

const (
	MaxAuthorizationResponseBytes = 64 << 10
	MaxWebhookBodyBytes           = 64 << 10
	authOrigin                    = "https://auth.tiktok-shops.com"
	apiOrigin                     = "https://open-api.tiktokglobalshop.com"
	providerTimeout               = 5 * time.Second
)

type OfficialClientOptions struct {
	Application authorization.Application
	AppSecret   string `json:"-"`
	HTTPClient  *http.Client
}

type OfficialClient struct {
	application authorization.Application
	appSecret   string
	http        *http.Client
	authHTTP    *http.Client
	now         func() time.Time
}

var _ authorization.SellerAuthorization = (*OfficialClient)(nil)

func NewOfficialClient(options OfficialClientOptions) (*OfficialClient, error) {
	callback, err := url.Parse(options.Application.CallbackURL)
	if err != nil || callback.Scheme != "https" || callback.Host == "" || callback.User != nil || callback.RawQuery != "" || callback.Fragment != "" ||
		!bounded(options.Application.AppKey, 128) || !bounded(options.Application.ServiceID, 128) || !bounded(options.Application.Revision, 128) || !bounded(options.AppSecret, 512) {
		return nil, authorization.ErrInvalid
	}
	client := http.Client{Timeout: providerTimeout}
	if options.HTTPClient != nil {
		client = *options.HTTPClient
	}
	if client.Timeout <= 0 || client.Timeout > providerTimeout {
		client.Timeout = providerTimeout
	}
	client.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	authClient := client
	authTransport := client.Transport
	if authTransport == nil {
		authTransport = http.DefaultTransport
	}
	if transport, ok := authTransport.(*http.Transport); ok {
		// net/http may transparently replay a GET on a failed reused connection.
		// Token GETs consume/rotate credentials: use fresh HTTP/1 connections.
		isolated := transport.Clone()
		isolated.DisableKeepAlives = true
		isolated.Protocols = new(http.Protocols)
		isolated.Protocols.SetHTTP1(true)
		authTransport = isolated
	}
	// A caller-supplied nonstandard RoundTripper is a trusted injection; it must
	// itself perform no retries. The normal application uses stdlib Transport.
	authClient.Transport = authTransport
	return &OfficialClient{application: options.Application, appSecret: options.AppSecret, http: &client, authHTTP: &authClient, now: time.Now}, nil
}

func (*OfficialClient) String() string   { return "TikTok Shop official client [redacted]" }
func (*OfficialClient) GoString() string { return "TikTok Shop official client [redacted]" }

func (c *OfficialClient) Application() authorization.Application { return c.application }

func (c *OfficialClient) AuthorizationURL(state string) (string, error) {
	if c == nil || !bounded(state, 256) {
		return "", authorization.ErrInvalid
	}
	query := url.Values{"service_id": {c.application.ServiceID}, "state": {state}}
	return "https://services.us.tiktokshop.com/open/authorize?" + query.Encode(), nil
}

func bounded(value string, maximum int) bool {
	if value == "" || len(value) > maximum || !utf8.ValidString(value) || strings.TrimSpace(value) != value {
		return false
	}
	for _, r := range value {
		if r < 32 || r == 127 {
			return false
		}
	}
	return true
}

func decode(raw []byte, target any) bool {
	strict, err := sigjson.UnmarshalStrict(raw, target, sigjson.DisallowDuplicateFields)
	return err == nil && len(strict) == 0
}

// get sends once to a fixed official endpoint. Token URLs contain secrets, so
// neither http/url errors nor provider message/body are returned to consumers.
func (c *OfficialClient) get(ctx context.Context, origin, path string, query url.Values, accessToken string) ([]byte, error) {
	if c == nil || ctx == nil || ctx.Err() != nil {
		return nil, authorization.ErrInvalid
	}
	ctx, cancel := context.WithTimeout(ctx, providerTimeout)
	defer cancel()
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, origin+path+"?"+query.Encode(), nil)
	if err != nil {
		return nil, authorization.ErrUnavailable
	}
	request.Header.Set("Content-Type", "application/json")
	if accessToken != "" {
		request.Header.Set("x-tts-access-token", accessToken)
	}
	client := c.http
	if origin == authOrigin {
		client = c.authHTTP
		request.Close = true
	}
	response, err := client.Do(request)
	if err != nil {
		return nil, authorization.ErrUnavailable
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return nil, authorization.ErrUnavailable
	}
	if header := response.Header.Get("Content-Type"); header != "" {
		media, params, err := mime.ParseMediaType(header)
		if err != nil || media != "application/json" || params["charset"] != "" && !strings.EqualFold(params["charset"], "utf-8") {
			return nil, authorization.ErrUnavailable
		}
	}
	raw, err := io.ReadAll(io.LimitReader(response.Body, MaxAuthorizationResponseBytes+1))
	if err != nil || len(raw) == 0 || len(raw) > MaxAuthorizationResponseBytes || !utf8.Valid(raw) {
		return nil, authorization.ErrUnavailable
	}
	return raw, nil
}

func (c *OfficialClient) Exchange(ctx context.Context, code string) (authorization.TokenSet, error) {
	if c == nil || ctx == nil || ctx.Err() != nil || !bounded(code, 2048) {
		return authorization.TokenSet{}, authorization.ErrInvalid
	}
	if code == "null" {
		return authorization.TokenSet{}, authorization.ErrAuthorizationRejected
	}
	return c.token(ctx, "/api/v2/token/get", "auth_code", code, "authorized_code")
}

func (c *OfficialClient) Refresh(ctx context.Context, token string) (authorization.TokenSet, error) {
	if c == nil || ctx == nil || ctx.Err() != nil || !bounded(token, 8192) {
		return authorization.TokenSet{}, authorization.ErrInvalid
	}
	return c.token(ctx, "/api/v2/token/refresh", "refresh_token", token, "refresh_token")
}

func (c *OfficialClient) token(ctx context.Context, path, key, value, grantType string) (authorization.TokenSet, error) {
	query := url.Values{"app_key": {c.application.AppKey}, "app_secret": {c.appSecret}, key: {value}, "grant_type": {grantType}}
	raw, err := c.get(ctx, authOrigin, path, query, "")
	if err != nil {
		return authorization.TokenSet{}, authorization.ErrOutcomeUnknown
	}
	var response struct {
		Code *int `json:"code"`
		Data *struct {
			AccessToken      string   `json:"access_token"`
			RefreshToken     string   `json:"refresh_token"`
			AccessExpires    int64    `json:"access_token_expire_in"`
			RefreshExpires   int64    `json:"refresh_token_expire_in"`
			OpenID           string   `json:"open_id"`
			SellerBaseRegion string   `json:"seller_base_region"`
			UserType         *int     `json:"user_type"`
			Scopes           []string `json:"granted_scopes"`
		} `json:"data"`
	}
	if !decode(raw, &response) || response.Code == nil {
		return authorization.TokenSet{}, authorization.ErrOutcomeUnknown
	}
	if *response.Code != 0 {
		return authorization.TokenSet{}, authorization.ErrAuthorizationRejected
	}
	data := response.Data
	if data == nil || data.UserType == nil || !bounded(data.AccessToken, 8192) || !bounded(data.RefreshToken, 8192) || !bounded(data.OpenID, 512) ||
		data.AccessExpires <= c.now().Unix() || data.RefreshExpires <= c.now().Unix() || data.AccessExpires > 253402300799 || data.RefreshExpires > 253402300799 || len(data.Scopes) > 128 {
		return authorization.TokenSet{}, authorization.ErrOutcomeUnknown
	}
	if *data.UserType != 0 && *data.UserType != 4 && *data.UserType != 5 {
		return authorization.TokenSet{}, authorization.ErrAuthorizationRejected
	}
	seen := make(map[string]bool, len(data.Scopes))
	for _, scope := range data.Scopes {
		if !bounded(scope, 128) || seen[scope] {
			return authorization.TokenSet{}, authorization.ErrOutcomeUnknown
		}
		seen[scope] = true
	}
	if !seen["seller.authorization.info"] {
		return authorization.TokenSet{}, authorization.ErrAuthorizationRejected
	}
	return authorization.TokenSet{AccessToken: data.AccessToken, RefreshToken: data.RefreshToken, OpenID: data.OpenID,
		SellerBaseRegion: data.SellerBaseRegion, UserType: *data.UserType, GrantedScopes: append([]string(nil), data.Scopes...),
		AccessExpiresAt: time.Unix(data.AccessExpires, 0).UTC(), RefreshExpiresAt: time.Unix(data.RefreshExpires, 0).UTC()}, nil
}

func (c *OfficialClient) AuthorizedShops(ctx context.Context, accessToken string) ([]authorization.Shop, error) {
	if c == nil || !bounded(accessToken, 8192) {
		return nil, authorization.ErrInvalid
	}
	path := "/authorization/202309/shops"
	query := url.Values{"app_key": {c.application.AppKey}, "timestamp": {strconv.FormatInt(c.now().Unix(), 10)}}
	signed, err := signature(path, query, nil, "application/json", c.appSecret)
	if err != nil {
		return nil, authorization.ErrUnavailable
	}
	query.Set("sign", signed)
	raw, err := c.get(ctx, apiOrigin, path, query, accessToken)
	if err != nil {
		return nil, err
	}
	var response struct {
		Code *int `json:"code"`
		Data *struct {
			Shops []struct {
				ID         string `json:"id"`
				Code       string `json:"code"`
				Region     string `json:"region"`
				SellerType string `json:"seller_type"`
				Cipher     string `json:"cipher"`
			} `json:"shops"`
		} `json:"data"`
	}
	if !decode(raw, &response) || response.Code == nil || *response.Code != 0 || response.Data == nil || response.Data.Shops == nil {
		return nil, authorization.ErrUnavailable
	}
	shops := make([]authorization.Shop, 0, len(response.Data.Shops))
	seenIDs, seenCodes := map[string]bool{}, map[string]bool{}
	for _, shop := range response.Data.Shops {
		if !bounded(shop.ID, 128) || !bounded(shop.Code, 128) || !bounded(shop.Cipher, 8192) || len(shop.Region) != 2 || shop.Region != strings.ToUpper(shop.Region) ||
			shop.SellerType != "LOCAL" && shop.SellerType != "CROSS_BORDER" || seenIDs[shop.ID] || seenCodes[shop.Code] {
			return nil, authorization.ErrUnavailable
		}
		seenIDs[shop.ID], seenCodes[shop.Code] = true, true
		shops = append(shops, authorization.Shop{ID: shop.ID, Code: shop.Code, Region: shop.Region, SellerType: shop.SellerType, Cipher: shop.Cipher})
	}
	return shops, nil
}

// VerifyDeauthorization verifies raw bytes before JSON decoding. A verified
// event still needs the Store transaction to match app/Shop/current grant.
func (c *OfficialClient) VerifyDeauthorization(raw []byte, header string) (authorization.Deauthorization, error) {
	if c == nil || len(raw) == 0 || len(raw) > MaxWebhookBodyBytes || len(header) != 64 || header != strings.ToLower(header) {
		return authorization.Deauthorization{}, authorization.ErrInvalid
	}
	provided, err := hex.DecodeString(header)
	if err != nil {
		return authorization.Deauthorization{}, authorization.ErrInvalid
	}
	mac := hmac.New(sha256.New, []byte(c.appSecret))
	_, _ = mac.Write([]byte(c.application.AppKey))
	_, _ = mac.Write(raw)
	if !hmac.Equal(provided, mac.Sum(nil)) {
		return authorization.Deauthorization{}, authorization.ErrInvalid
	}
	var event struct {
		Type           *int   `json:"type"`
		ShopID         string `json:"shop_id"`
		Timestamp      int64  `json:"timestamp"`
		NotificationID string `json:"tts_notification_id"`
	}
	if !utf8.Valid(raw) || !decode(raw, &event) || event.Type == nil || *event.Type != 6 || !bounded(event.ShopID, 128) || event.Timestamp <= 0 || event.Timestamp > 253402300799 ||
		event.NotificationID != "" && !bounded(event.NotificationID, 128) {
		return authorization.Deauthorization{}, authorization.ErrInvalid
	}
	hash := sha256.Sum256(raw)
	bodyHash := hex.EncodeToString(hash[:])
	if event.NotificationID == "" {
		event.NotificationID = "body:" + bodyHash
	}
	return authorization.Deauthorization{AppKey: c.application.AppKey, ShopID: event.ShopID, NotificationID: event.NotificationID, BodyHash: bodyHash, OccurredAt: time.Unix(event.Timestamp, 0).UTC()}, nil
}
