package tiktokshop

import (
	"context"
	"io"
	"mime"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	marketplace "task-processor/internal/marketplace/tiktokshop"
)

// Only the named official endpoints below use this private transport helper.
// It does not acquire a Store proof, refresh a token or send a mutation.
func (c *OfficialClient) resourceRead(ctx context.Context, path string, query url.Values, token string) ([]byte, error) {
	if c == nil || ctx == nil || ctx.Err() != nil || !bounded(token, 8192) {
		return nil, marketplace.ErrListingRules
	}
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	query.Set("app_key", c.application.AppKey)
	query.Set("timestamp", strconv.FormatInt(c.now().Unix(), 10))
	signed, err := signature(path, query, nil, "application/json", c.appSecret)
	if err != nil {
		return nil, marketplace.ErrListingRules
	}
	query.Set("sign", signed)
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, apiOrigin+path+"?"+query.Encode(), nil)
	if err != nil {
		return nil, marketplace.ErrListingRules
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("x-tts-access-token", token)
	response, err := c.http.Do(request)
	if err != nil {
		return nil, marketplace.ErrListingRules
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return nil, marketplace.ErrListingRules
	}
	media, params, err := mime.ParseMediaType(response.Header.Get("Content-Type"))
	if err != nil || media != "application/json" || params["charset"] != "" && !strings.EqualFold(params["charset"], "utf-8") {
		return nil, marketplace.ErrListingRules
	}
	raw, err := io.ReadAll(io.LimitReader(response.Body, marketplace.MaxOfficialResultBytes+1))
	if err != nil || len(raw) == 0 || len(raw) > marketplace.MaxOfficialResultBytes || !utf8.Valid(raw) {
		return nil, marketplace.ErrListingRules
	}
	return raw, nil
}

func (c *OfficialClient) QueryListingRules(ctx context.Context, token, cipher string) (marketplace.ListingRules, error) {
	if !bounded(cipher, 4096) {
		return marketplace.ListingRules{}, marketplace.ErrListingRules
	}
	raw, err := c.resourceRead(ctx, "/product/202507/global_listing_rules", url.Values{"shop_cipher": {cipher}}, token)
	if err != nil {
		return marketplace.ListingRules{}, err
	}
	return marketplace.DecodeListingRules(raw)
}

func categoryQuery(method marketplace.ListingMethod, cipher string) (url.Values, bool) {
	query := url.Values{"category_version": {"v2"}, "locale": {"en-US"}}
	if !bounded(cipher, 4096) || method != marketplace.LocalReplication && method != marketplace.GlobalPublishing {
		return nil, false
	}
	if method == marketplace.LocalReplication {
		query.Set("shop_cipher", cipher)
	}
	return query, true
}

func (c *OfficialClient) QueryCategories(ctx context.Context, token, cipher string, method marketplace.ListingMethod) ([]marketplace.Category, error) {
	query, valid := categoryQuery(method, cipher)
	if !valid {
		return nil, marketplace.ErrListingRules
	}
	path := "/product/202309/categories"
	if method == marketplace.GlobalPublishing {
		path = "/product/202309/global_categories"
	}
	raw, err := c.resourceRead(ctx, path, query, token)
	if err != nil {
		return nil, err
	}
	return marketplace.Categories(raw)
}

func (c *OfficialClient) QueryCategoryRules(ctx context.Context, token, cipher string, selection marketplace.RuleSelection) (marketplace.CategoryRules, error) {
	query, valid := categoryQuery(selection.Method, cipher)
	if !valid || !marketplace.ValidSelection(selection) {
		return marketplace.CategoryRules{}, marketplace.ErrListingRules
	}
	suffix := "rules"
	if selection.Method == marketplace.GlobalPublishing {
		suffix = "global_rules"
	}
	raw, err := c.resourceRead(ctx, "/product/202309/categories/"+url.PathEscape(selection.CategoryID)+"/"+suffix, query, token)
	if err != nil {
		return marketplace.CategoryRules{}, err
	}
	return marketplace.DecodeCategoryRules(raw)
}

func (c *OfficialClient) QueryAttributes(ctx context.Context, token, cipher string, selection marketplace.RuleSelection) ([]marketplace.AttributeRule, error) {
	query, valid := categoryQuery(selection.Method, cipher)
	if !valid || !marketplace.ValidSelection(selection) {
		return nil, marketplace.ErrListingRules
	}
	suffix := "attributes"
	if selection.Method == marketplace.GlobalPublishing {
		suffix = "global_attributes"
	}
	raw, err := c.resourceRead(ctx, "/product/202309/categories/"+url.PathEscape(selection.CategoryID)+"/"+suffix, query, token)
	if err != nil {
		return nil, err
	}
	return marketplace.DecodeAttributes(raw)
}

func (c *OfficialClient) QueryWarehouses(ctx context.Context, token, cipher string) ([]marketplace.Warehouse, error) {
	if !bounded(cipher, 4096) {
		return nil, marketplace.ErrListingRules
	}
	raw, err := c.resourceRead(ctx, "/logistics/202309/warehouses", url.Values{"shop_cipher": {cipher}}, token)
	if err != nil {
		return nil, err
	}
	var response struct {
		Code *int `json:"code"`
		Data *struct {
			Warehouses []marketplace.Warehouse `json:"warehouses"`
		} `json:"data"`
	}
	if !decode(raw, &response) || response.Code == nil || *response.Code != 0 || response.Data == nil || len(response.Data.Warehouses) > 1000 {
		return nil, marketplace.ErrListingRules
	}
	seen := map[string]bool{}
	for _, warehouse := range response.Data.Warehouses {
		if !bounded(warehouse.ID, 128) || seen[warehouse.ID] {
			return nil, marketplace.ErrListingRules
		}
		seen[warehouse.ID] = true
	}
	return response.Data.Warehouses, nil
}

func (c *OfficialClient) QueryGlobalWarehouses(ctx context.Context, token string) ([]marketplace.GlobalWarehouse, error) {
	raw, err := c.resourceRead(ctx, "/logistics/202309/global_warehouses", url.Values{}, token)
	if err != nil {
		return nil, err
	}
	var response struct {
		Code *int `json:"code"`
		Data *struct {
			Warehouses []marketplace.GlobalWarehouse `json:"global_warehouses"`
		} `json:"data"`
	}
	if !decode(raw, &response) || response.Code == nil || *response.Code != 0 || response.Data == nil || len(response.Data.Warehouses) > 1000 {
		return nil, marketplace.ErrListingRules
	}
	seen := map[string]bool{}
	for _, warehouse := range response.Data.Warehouses {
		if !bounded(warehouse.ID, 128) || seen[warehouse.ID] || warehouse.Ownership != "SELLER" && warehouse.Ownership != "PLATFORM_COOPERATION" {
			return nil, marketplace.ErrListingRules
		}
		seen[warehouse.ID] = true
	}
	return response.Data.Warehouses, nil
}
