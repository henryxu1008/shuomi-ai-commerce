package walmart

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	marketplace "task-processor/internal/marketplace/walmart"
	auth "task-processor/internal/storecenter/walmart"
)

const productionOrigin = "https://marketplace.walmartapis.com"
const sandboxOrigin = "https://sandbox.walmartapis.com"
const authorizeURL = "https://login.account.wal-mart.com/authorize"
const maxResponseBytes = 8 << 20

type Options struct {
	Application               auth.Application
	ClientSecret              string
	HTTPClient                *http.Client
	FullVersion, MatchVersion string
}
type OfficialClient struct {
	app                                       auth.Application
	secret, origin, fullVersion, matchVersion string
	http                                      *http.Client
}

func NewOfficialClient(o Options) (*OfficialClient, error) {
	u, e := url.Parse(o.Application.CallbackURL)
	if e != nil || u.Scheme != "https" || u.Hostname() == "" || u.User != nil || u.RawQuery != "" || u.Fragment != "" || o.Application.ClientID == "" || len(o.Application.ClientID) > 200 || o.Application.Revision == "" || len(o.Application.Revision) > 128 || o.ClientSecret == "" || strings.ContainsAny(o.Application.ClientID+o.ClientSecret, "\r\n\x00") {
		return nil, auth.ErrInvalid
	}
	origin := productionOrigin
	switch o.Application.Environment {
	case "production":
	case "sandbox":
		origin = sandboxOrigin
	default:
		return nil, auth.ErrInvalid
	}
	h := http.Client{Timeout: 10 * time.Second}
	if o.HTTPClient != nil {
		h = *o.HTTPClient
	}
	h.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	if h.Timeout <= 0 || h.Timeout > 10*time.Second {
		h.Timeout = 10 * time.Second
	}
	return &OfficialClient{app: o.Application, secret: o.ClientSecret, origin: origin, http: &h, fullVersion: o.FullVersion, matchVersion: o.MatchVersion}, nil
}
func (c *OfficialClient) Application() auth.Application { return c.app }
func (c *OfficialClient) AuthorizationURL(state, nonce string) (string, error) {
	if state == "" || nonce == "" || len(state) > 256 || len(nonce) > 256 {
		return "", auth.ErrInvalid
	}
	q := url.Values{"responseType": {"code"}, "clientId": {c.app.ClientID}, "redirectUri": {c.app.CallbackURL}, "clientType": {"seller"}, "nonce": {nonce}, "state": {state}}
	return authorizeURL + "?" + q.Encode(), nil
}

// request never retries, follows redirects, returns provider bodies as errors,
// or exposes caller-selected origins. Each exported method fixes its route.
func (c *OfficialClient) request(ctx context.Context, method, path string, q url.Values, body []byte, tokens auth.TokenSet, form bool, correlation string) ([]byte, int, error) {
	if ctx == nil || len(body) > marketplace.MaxFeedBytes {
		return nil, 0, marketplace.ErrInvalid
	}
	endpoint := c.origin + path
	if len(q) > 0 {
		endpoint += "?" + q.Encode()
	}
	r, e := http.NewRequestWithContext(ctx, method, endpoint, bytes.NewReader(body))
	if e != nil {
		return nil, 0, marketplace.ErrUnavailable
	}
	r.Header.Set("Accept", "application/json")
	r.Header.Set("WM_MARKET", "us")
	r.Header.Set("WM_SVC.NAME", "Walmart Marketplace")
	if correlation == "" {
		correlation = uuid.NewString()
	}
	r.Header.Set("WM_QOS.CORRELATION_ID", correlation)
	if form {
		r.SetBasicAuth(c.app.ClientID, c.secret)
		r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	} else {
		r.Header.Set("Content-Type", "application/json")
		r.Header.Set("WM_SEC.ACCESS_TOKEN", tokens.AccessToken)
	}
	if path == "/v3/token/detail" {
		r.SetBasicAuth(c.app.ClientID, c.secret)
	}
	if tokens.SellerID != "" {
		r.Header.Set("WM_PARTNER.ID", tokens.SellerID)
	}
	if path == "/v3/settings/partnerprofile" {
		r.Header.Set("WM_GLOBAL_VERSION", "3.1")
	}
	response, e := c.http.Do(r)
	if e != nil {
		return nil, 0, marketplace.ErrUnknown
	}
	defer response.Body.Close()
	raw, e := io.ReadAll(io.LimitReader(response.Body, maxResponseBytes+1))
	if e != nil || len(raw) > maxResponseBytes {
		return nil, response.StatusCode, marketplace.ErrUnavailable
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		if response.StatusCode == 401 || response.StatusCode == 403 {
			return nil, response.StatusCode, marketplace.ErrRejected
		}
		return nil, response.StatusCode, marketplace.ErrUnavailable
	}
	return raw, response.StatusCode, nil
}
func (c *OfficialClient) token(ctx context.Context, form url.Values, seller string, prior *auth.TokenSet) (auth.TokenSet, error) {
	var result auth.TokenSet
	if seller == "" || len(seller) > 200 || strings.ContainsAny(seller, "\r\n\x00") {
		return result, auth.ErrInvalid
	}
	raw, _, e := c.request(ctx, http.MethodPost, "/v3/token", nil, []byte(form.Encode()), auth.TokenSet{SellerID: seller}, true, "")
	if e != nil {
		return result, e
	}
	var response struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		ExpiresIn    int64  `json:"expires_in"`
		TokenType    string `json:"token_type"`
	}
	if json.Unmarshal(raw, &response) != nil || response.AccessToken == "" || len(response.AccessToken) > 8192 || response.ExpiresIn <= 0 || response.ExpiresIn > 86400 || !strings.EqualFold(response.TokenType, "Bearer") {
		return result, marketplace.ErrUnknown
	}
	now := time.Now().UTC()
	result = auth.TokenSet{AccessToken: response.AccessToken, RefreshToken: response.RefreshToken, SellerID: seller, AccessExpiresAt: now.Add(time.Duration(response.ExpiresIn) * time.Second), RefreshExpiresAt: now.AddDate(1, 0, 0)}
	if prior != nil {
		result.RefreshExpiresAt = prior.RefreshExpiresAt
		if result.RefreshToken == "" {
			result.RefreshToken = prior.RefreshToken
		}
	}
	if result.RefreshToken == "" || len(result.RefreshToken) > 8192 {
		return auth.TokenSet{}, marketplace.ErrUnknown
	}
	return result, nil
}
func (c *OfficialClient) Exchange(ctx context.Context, code, seller string) (auth.TokenSet, error) {
	if code == "" || len(code) > 2048 {
		return auth.TokenSet{}, auth.ErrInvalid
	}
	return c.token(ctx, url.Values{"grant_type": {"authorization_code"}, "code": {code}, "redirect_uri": {c.app.CallbackURL}}, seller, nil)
}
func (c *OfficialClient) Refresh(ctx context.Context, old auth.TokenSet) (auth.TokenSet, error) {
	if old.RefreshToken == "" || !time.Now().Before(old.RefreshExpiresAt) {
		return auth.TokenSet{}, marketplace.ErrRejected
	}
	return c.token(ctx, url.Values{"grant_type": {"refresh_token"}, "refresh_token": {old.RefreshToken}}, old.SellerID, &old)
}
func (c *OfficialClient) TokenDetail(ctx context.Context, t auth.TokenSet) (auth.TokenDetail, error) {
	var result auth.TokenDetail
	raw, _, e := c.request(ctx, http.MethodGet, "/v3/token/detail", nil, nil, t, false, "")
	if e != nil {
		return result, e
	}
	var v struct {
		Valid   bool              `json:"is_valid"`
		Expires json.Number       `json:"expire_at"`
		Scopes  map[string]string `json:"scopes"`
	}
	if json.Unmarshal(raw, &v) != nil {
		return result, marketplace.ErrUnavailable
	}
	ms, e := strconv.ParseInt(string(v.Expires), 10, 64)
	if e != nil {
		return result, marketplace.ErrUnavailable
	}
	result = auth.TokenDetail{Valid: v.Valid, ExpiresAt: time.UnixMilli(ms), Scopes: v.Scopes}
	return result, nil
}
func (c *OfficialClient) Merchant(ctx context.Context, t auth.TokenSet) (auth.Merchant, error) {
	raw, _, e := c.request(ctx, http.MethodGet, "/v3/settings/partnerprofile", nil, nil, t, false, "")
	if e != nil {
		return auth.Merchant{}, e
	}
	var v struct {
		Partner struct {
			PartnerID   string `json:"partnerId"`
			PartnerName string `json:"partnerName"`
		} `json:"partner"`
	}
	if json.Unmarshal(raw, &v) != nil || v.Partner.PartnerID == "" || v.Partner.PartnerID != t.SellerID {
		return auth.Merchant{}, marketplace.ErrRejected
	}
	return auth.Merchant{PartnerID: v.Partner.PartnerID, Name: v.Partner.PartnerName}, nil
}
func (c *OfficialClient) Taxonomy(ctx context.Context, t auth.TokenSet, feedType string) (json.RawMessage, error) {
	version := c.version(feedType)
	if version == "" {
		return nil, marketplace.ErrUnavailable
	}
	raw, status, e := c.request(ctx, http.MethodGet, "/v3/items/taxonomy", url.Values{"feedType": {feedType}, "version": {version}}, nil, t, false, "")
	if e != nil || status != 200 {
		return nil, marketplace.ErrUnavailable
	}
	return raw, nil
}
func (c *OfficialClient) version(feedType string) string {
	switch feedType {
	case marketplace.FullFeed:
		return c.fullVersion
	case marketplace.MatchFeed:
		return c.matchVersion
	}
	return ""
}
func (c *OfficialClient) SpecVersion(feedType string) string { return c.version(feedType) }
func (c *OfficialClient) Rules(ctx context.Context, t auth.TokenSet, s marketplace.RuleSelection) (marketplace.RuleSnapshot, error) {
	var out marketplace.RuleSnapshot
	if !s.Valid() || c.version(s.FeedType) == "" || s.Version != c.version(s.FeedType) {
		return out, marketplace.ErrInvalid
	}
	var productTypes []string
	if s.FeedType == marketplace.FullFeed {
		productTypes = []string{s.ProductType}
	}
	body, _ := json.Marshal(struct {
		FeedType     string   `json:"feedType"`
		Version      string   `json:"version"`
		ProductTypes []string `json:"productTypes,omitempty"`
	}{s.FeedType, s.Version, productTypes})
	raw, status, e := c.request(ctx, http.MethodPost, "/v3/items/spec", nil, body, t, false, "")
	if e != nil || status != 200 {
		return out, marketplace.ErrUnavailable
	}
	var v struct {
		Schema json.RawMessage   `json:"schema"`
		Errors []json.RawMessage `json:"errors"`
	}
	if json.Unmarshal(raw, &v) != nil || len(v.Errors) > 0 {
		return out, marketplace.ErrUnavailable
	}
	header, e := specHeader(v.Schema, s)
	if e != nil {
		return out, e
	}
	out = marketplace.RuleSnapshot{Selection: s, Header: header, Schema: v.Schema, ObservedAt: time.Now().UTC()}
	if out.Validate() != nil {
		return marketplace.RuleSnapshot{}, marketplace.ErrUnavailable
	}
	return out, nil
}
func specHeader(raw json.RawMessage, s marketplace.RuleSelection) (marketplace.FeedHeader, error) {
	var root struct {
		Properties map[string]struct {
			Properties map[string]struct {
				Enum  []string `json:"enum"`
				Const string   `json:"const"`
			} `json:"properties"`
			Required []string `json:"required"`
		} `json:"properties"`
	}
	if json.Unmarshal(raw, &root) != nil {
		return marketplace.FeedHeader{}, marketplace.ErrUnavailable
	}
	shape, ok := root.Properties["MPItemFeedHeader"]
	if !ok {
		return marketplace.FeedHeader{}, marketplace.ErrUnavailable
	}
	values := map[string]string{"locale": "en", "version": s.Version, "businessUnit": "WALMART_US"}
	for _, key := range shape.Required {
		if _, ok := values[key]; ok {
			continue
		}
		p := shape.Properties[key]
		if p.Const != "" {
			values[key] = p.Const
		} else if len(p.Enum) == 1 {
			values[key] = p.Enum[0]
		} else {
			return marketplace.FeedHeader{}, marketplace.ErrUnavailable
		}
	}
	// Emit only schema-declared header keys. No legacy sample version defaults.
	for k := range values {
		if _, ok := shape.Properties[k]; !ok {
			delete(values, k)
		}
	}
	b, _ := json.Marshal(values)
	var h marketplace.FeedHeader
	if json.Unmarshal(b, &h) != nil {
		return h, marketplace.ErrUnavailable
	}
	return h, nil
}
