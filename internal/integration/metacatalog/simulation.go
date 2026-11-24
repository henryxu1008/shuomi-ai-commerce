package metacatalog

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"time"

	domain "github.com/henryxu1008/shuomi-ai-commerce/internal/storecenter/metacatalog"
)

const (
	appID   = "shuomi-meta-simulation"
	version = "/v27.0/"
)

type Simulation struct {
	origin string
	client *http.Client
}

func NewSimulation(origin string) (*Simulation, error) {
	u, err := url.Parse(origin)
	if err != nil || u.Scheme != "http" || u.Hostname() != "127.0.0.1" || u.User != nil ||
		u.Path != "" || u.RawQuery != "" || u.Fragment != "" || origin != "http://"+u.Host {
		return nil, domain.ErrUnavailable
	}
	port, err := strconv.ParseUint(u.Port(), 10, 16)
	if err != nil || port == 0 || u.Host != net.JoinHostPort("127.0.0.1", u.Port()) {
		return nil, domain.ErrUnavailable
	}
	return &Simulation{origin: origin, client: &http.Client{
		Transport:     &http.Transport{DialContext: (&net.Dialer{Timeout: time.Second}).DialContext},
		Timeout:       3 * time.Second,
		CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
	}}, nil
}

func (s *Simulation) ConsentURL(callback, state string) (string, error) {
	u, err := url.Parse(callback)
	if err != nil || u.Scheme != "http" || u.Hostname() != "127.0.0.1" || u.User != nil ||
		u.Port() == "" || u.Path != "/workbench/stores/meta-catalog/callback" || u.RawQuery != "" || u.Fragment != "" || state == "" {
		return "", domain.ErrUnauthorized
	}
	query := url.Values{"client_id": {appID}, "redirect_uri": {callback}, "response_type": {"code"}, "state": {state}}
	return s.origin + "/dialog/oauth?" + query.Encode(), nil
}

func (s *Simulation) Exchange(ctx context.Context, callback, code string) (domain.Credentials, error) {
	if _, err := s.ConsentURL(callback, "validate"); err != nil || code == "" || len(code) > 1024 {
		return domain.Credentials{}, domain.ErrUnauthorized
	}
	var token struct {
		AccessToken string `json:"access_token"`
	}
	values := url.Values{"client_id": {appID}, "redirect_uri": {callback}, "code": {code}}
	if err := s.call(ctx, http.MethodGet, "oauth/access_token?"+values.Encode(), "", nil, &token); err != nil ||
		token.AccessToken == "" || len(token.AccessToken) > 4096 || strings.ContainsAny(token.AccessToken, "\r\n") {
		return domain.Credentials{}, domain.ErrUnauthorized
	}
	var info struct {
		Data struct {
			AppID     string   `json:"app_id"`
			UserID    string   `json:"user_id"`
			Valid     bool     `json:"is_valid"`
			ExpiresAt int64    `json:"expires_at"`
			Scopes    []string `json:"scopes"`
		} `json:"data"`
	}
	values = url.Values{"input_token": {token.AccessToken}}
	if err := s.call(ctx, http.MethodGet, "debug_token?"+values.Encode(), "", nil, &info); err != nil {
		return domain.Credentials{}, domain.ErrUnauthorized
	}
	data := info.Data
	expires := time.Unix(data.ExpiresAt, 0).UTC()
	if !data.Valid || data.AppID != appID || !domain.ValidID(data.UserID) || !time.Now().Before(expires) ||
		!slices.Contains(data.Scopes, "business_management") || !slices.Contains(data.Scopes, "catalog_management") {
		return domain.Credentials{}, domain.ErrUnauthorized
	}
	whatsapp := slices.Contains(data.Scopes, "whatsapp_business_management") &&
		slices.Contains(data.Scopes, "whatsapp_business_messaging") && slices.Contains(data.Scopes, "public_profile")
	return domain.Credentials{Token: token.AccessToken, Subject: data.UserID, ExpiresAt: expires, WhatsApp: whatsapp}, nil
}

type choiceResponse struct {
	Data   []domain.Choice
	Paging struct{ Next string }
}

func (s *Simulation) Choices(ctx context.Context, cred domain.Credentials) ([]domain.Choice, []domain.Choice, error) {
	if !domain.ValidID(cred.Subject) || !time.Now().Before(cred.ExpiresAt) {
		return nil, nil, domain.ErrUnauthorized
	}
	read := func(path string) ([]domain.Choice, error) {
		var response choiceResponse
		if err := s.call(ctx, http.MethodGet, path, cred.Token, nil, &response); err != nil || response.Paging.Next != "" || len(response.Data) > 30 {
			return nil, domain.ErrUnavailable
		}
		for _, choice := range response.Data {
			if !domain.ValidID(choice.ID) || len(choice.Name) > 200 {
				return nil, domain.ErrUnavailable
			}
		}
		return response.Data, nil
	}
	ctx, cancel := context.WithTimeout(ctx, 6*time.Second)
	defer cancel()
	businesses, err := read("me/businesses")
	if err != nil {
		return nil, nil, err
	}
	catalogs, accounts := []domain.Choice{}, []domain.Choice{}
	for _, business := range businesses {
		owned, err := read(business.ID + "/owned_product_catalogs")
		if err != nil {
			return nil, nil, err
		}
		for _, item := range owned {
			var assigned struct {
				Data []struct {
					ID    string
					Tasks []string
				}
				Paging struct{ Next string }
			}
			if err := s.call(ctx, http.MethodGet, item.ID+"/assigned_users?business="+business.ID, cred.Token, nil, &assigned); err != nil || assigned.Paging.Next != "" {
				return nil, nil, domain.ErrUnavailable
			}
			for _, user := range assigned.Data {
				if user.ID == cred.Subject && slices.Contains(user.Tasks, "MANAGE") {
					item.BusinessID = business.ID
					catalogs = append(catalogs, item)
					break
				}
			}
		}
		if !cred.WhatsApp {
			continue
		}
		ownedAccounts, err := read(business.ID + "/owned_whatsapp_business_accounts")
		if err != nil {
			return nil, nil, err
		}
		for _, item := range ownedAccounts {
			item.BusinessID = business.ID
			accounts = append(accounts, item)
		}
	}
	return catalogs, accounts, nil
}

func (s *Simulation) Create(ctx context.Context, token, catalog string, p domain.Product) (string, error) {
	if !domain.ValidID(catalog) {
		return "", domain.ErrUnauthorized
	}
	fields := map[string]any{"retailer_id": p.RetailerID, "name": p.Title, "description": p.Description,
		"url": p.ProductURL, "image_url": p.ImageURL, "brand": p.Brand, "price": p.PriceCents,
		"currency": "USD", "availability": "in stock", "condition": "new", "allow_upsert": false}
	var result struct{ ID string }
	if err := s.call(ctx, http.MethodPost, catalog+"/products", token, fields, &result); err != nil {
		return "", err
	}
	if !domain.ValidID(result.ID) {
		return "", domain.ErrUnavailable
	}
	return result.ID, nil
}

func (s *Simulation) ReadProduct(ctx context.Context, token, catalog, id, retailer string) (bool, error) {
	if !domain.ValidID(catalog) || !domain.ValidID(id) || retailer == "" {
		return false, domain.ErrUnauthorized
	}
	var result struct {
		ID         string
		RetailerID string              `json:"retailer_id"`
		Catalog    struct{ ID string } `json:"product_catalog"`
	}
	err := s.call(ctx, http.MethodGet, id+"?fields=id,retailer_id,product_catalog", token, nil, &result)
	if err != nil {
		return false, err
	}
	return result.ID == id && result.RetailerID == retailer && result.Catalog.ID == catalog, nil
}

func (s *Simulation) WhatsAppLinked(ctx context.Context, token, account, catalog string) (bool, error) {
	if !domain.ValidID(account) || !domain.ValidID(catalog) {
		return false, domain.ErrUnauthorized
	}
	var result choiceResponse
	if err := s.call(ctx, http.MethodGet, account+"/product_catalogs", token, nil, &result); err != nil {
		return false, err
	}
	if result.Paging.Next != "" {
		return false, domain.ErrUnavailable
	}
	return slices.ContainsFunc(result.Data, func(c domain.Choice) bool { return c.ID == catalog }), nil
}

func (s *Simulation) call(ctx context.Context, method, path, token string, body, result any) error {
	var payload io.Reader
	if body != nil {
		encoded, err := json.Marshal(body)
		if err != nil {
			return domain.ErrUnavailable
		}
		payload = bytes.NewReader(encoded)
	}
	request, err := http.NewRequestWithContext(ctx, method, s.origin+version+path, payload)
	if err != nil {
		return domain.ErrUnavailable
	}
	if token != "" {
		request.Header.Set("Authorization", "Bearer "+token)
	}
	request.Header.Set("Accept", "application/json")
	if body != nil {
		request.Header.Set("Content-Type", "application/json")
	}
	response, err := s.client.Do(request)
	if err != nil {
		return domain.ErrUnavailable
	}
	defer response.Body.Close()
	data, err := io.ReadAll(io.LimitReader(response.Body, 65537))
	if err != nil || len(data) > 65536 {
		return domain.ErrUnavailable
	}
	if response.StatusCode != http.StatusOK {
		var rejected struct {
			Error *struct {
				Code int
				Type string
			}
		}
		if method == http.MethodPost && response.StatusCode == http.StatusBadRequest && json.Unmarshal(data, &rejected) == nil &&
			rejected.Error != nil && rejected.Error.Type == "OAuthException" && (rejected.Error.Code == 100 || rejected.Error.Code == 10800) {
			return domain.ErrRejected
		}
		return domain.ErrUnavailable
	}
	if err := json.Unmarshal(data, result); err != nil {
		return domain.ErrUnavailable
	}
	return nil
}

var _ domain.Provider = (*Simulation)(nil)
