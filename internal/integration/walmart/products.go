package walmart

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	marketplace "task-processor/internal/marketplace/walmart"
	auth "task-processor/internal/storecenter/walmart"
)

func (c *OfficialClient) Products(ctx context.Context, t auth.TokenSet, page int) (marketplace.ProductPage, error) {
	var out marketplace.ProductPage
	if page < 1 || page > 1000 {
		return out, marketplace.ErrInvalid
	}
	raw, status, e := c.request(ctx, http.MethodGet, "/v3/items", url.Values{"offset": {strconv.Itoa((page - 1) * 10)}, "limit": {"10"}}, nil, t, false, "")
	if e != nil {
		return out, e
	}
	var body struct {
		Total *int                   `json:"totalItems"`
		Items *[]marketplace.Product `json:"ItemResponse"`
	}
	if status != 200 || json.Unmarshal(raw, &body) != nil || body.Total == nil || body.Items == nil || *body.Total < 0 || len(*body.Items) > 10 || len(*body.Items) > *body.Total {
		return marketplace.ProductPage{}, marketplace.ErrUnavailable
	}
	out.Total = *body.Total
	out.Items = *body.Items
	for _, p := range out.Items {
		if !p.Valid() {
			return marketplace.ProductPage{}, marketplace.ErrUnavailable
		}
	}
	out.ObservedAt = time.Now().UTC()
	return out, nil
}
func (c *OfficialClient) ProductBySKU(ctx context.Context, t auth.TokenSet, sku string) (*marketplace.Product, error) {
	if sku == "" || len(sku) > 128 {
		return nil, marketplace.ErrInvalid
	}
	raw, status, e := c.request(ctx, http.MethodGet, "/v3/items", url.Values{"sku": {sku}, "limit": {"20"}}, nil, t, false, "")
	if e != nil {
		return nil, e
	}
	var result struct {
		Total *int                   `json:"totalItems"`
		Items *[]marketplace.Product `json:"ItemResponse"`
	}
	if status != 200 || json.Unmarshal(raw, &result) != nil || result.Total == nil || result.Items == nil || *result.Total < 0 || *result.Total != len(*result.Items) || *result.Total > 1 {
		return nil, marketplace.ErrUnavailable
	}
	if *result.Total == 0 {
		return nil, nil
	}
	if (*result.Items)[0].SKU != sku || !(*result.Items)[0].Valid() {
		return nil, marketplace.ErrUnavailable
	}
	return &(*result.Items)[0], nil
}
func (c *OfficialClient) Search(ctx context.Context, t auth.TokenSet, identifier string) (json.RawMessage, error) {
	if identifier == "" || len(identifier) > 128 {
		return nil, marketplace.ErrInvalid
	}
	raw, status, e := c.request(ctx, http.MethodGet, "/v3/items/walmart/search", url.Values{"query": {identifier}, "limit": {"20"}}, nil, t, false, "")
	if e != nil || status != 200 {
		return nil, marketplace.ErrUnavailable
	}
	return raw, nil
}
func (c *OfficialClient) SubmitFeed(ctx context.Context, t auth.TokenSet, d marketplace.Draft, correlation string) (marketplace.AcceptedFeed, error) {
	var result marketplace.AcceptedFeed
	if d.ValidateWire() != nil || uuid.Validate(correlation) != nil || d.Selection.Version != c.version(d.Selection.FeedType) {
		return result, marketplace.ErrInvalid
	}
	raw, status, e := c.request(ctx, http.MethodPost, "/v3/feeds", url.Values{"feedType": {d.Selection.FeedType}}, d.Payload, t, false, correlation)
	if e != nil {
		return result, marketplace.ErrUnknown
	}
	if status != 200 && status != 201 || json.Unmarshal(raw, &result) != nil || result.FeedID == "" || len(result.FeedID) > 256 || strings.ContainsAny(result.FeedID, "/\\\r\n\x00") {
		return marketplace.AcceptedFeed{}, marketplace.ErrUnknown
	}
	result.CorrelationID = correlation
	return result, nil
}
func (c *OfficialClient) FeedPage(ctx context.Context, t auth.TokenSet, feedID string, offset int) (marketplace.FeedObservation, error) {
	var out marketplace.FeedObservation
	if feedID == "" || len(feedID) > 256 || strings.ContainsAny(feedID, "/\\\r\n\x00") || offset < 0 || offset > marketplace.MaxSKUs || offset%50 != 0 {
		return out, marketplace.ErrInvalid
	}
	raw, status, e := c.request(ctx, http.MethodGet, "/v3/feeds/"+url.PathEscape(feedID), url.Values{"includeDetails": {"true"}, "offset": {strconv.Itoa(offset)}, "limit": {"50"}}, nil, t, false, "")
	if e != nil {
		return out, e
	}
	var response struct {
		FeedID     string `json:"feedId"`
		FeedStatus string `json:"feedStatus"`
		Received   *int   `json:"itemsReceived"`
		Succeeded  *int   `json:"itemsSucceeded"`
		Failed     *int   `json:"itemsFailed"`
		Processing *int   `json:"itemsProcessing"`
		Offset     *int   `json:"offset"`
		Limit      *int   `json:"limit"`
		Details    struct {
			Items []struct {
				SKU     string `json:"sku"`
				Status  string `json:"ingestionStatus"`
				ItemID  string `json:"itemid"`
				Pending string `json:"pendingStatusDescription"`
				Errors  struct {
					Errors []marketplace.ItemError `json:"ingestionError"`
				} `json:"ingestionErrors"`
			} `json:"itemIngestionStatus"`
		} `json:"itemDetails"`
	}
	if status != 200 || json.Unmarshal(raw, &response) != nil || response.FeedID != feedID || response.Received == nil || response.Succeeded == nil || response.Failed == nil || response.Processing == nil || response.Offset != nil && *response.Offset != offset || len(response.Details.Items) > 50 {
		return out, marketplace.ErrUnavailable
	}
	out = marketplace.FeedObservation{FeedID: feedID, FeedStatus: response.FeedStatus, Received: *response.Received, Succeeded: *response.Succeeded, Failed: *response.Failed, Processing: *response.Processing, ResponseHashes: []string{marketplace.Hash(json.RawMessage(raw))}, ObservedAt: time.Now().UTC()}
	if out.Received < 0 || out.Received > marketplace.MaxSKUs || out.Succeeded < 0 || out.Failed < 0 || out.Processing < 0 {
		return marketplace.FeedObservation{}, marketplace.ErrUnavailable
	}
	for _, i := range response.Details.Items {
		errors := i.Errors.Errors
		if i.Pending != "" {
			errors = append(errors, marketplace.ItemError{Code: "PENDING_REVIEW", Description: i.Pending})
		}
		out.Items = append(out.Items, marketplace.ItemResult{SKU: i.SKU, IngestionStatus: i.Status, ItemID: i.ItemID, Errors: errors})
	}
	return out, nil
}
