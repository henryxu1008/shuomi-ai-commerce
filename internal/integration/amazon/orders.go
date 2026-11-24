package amazon

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"strconv"
	"time"

	o "github.com/henryxu1008/shuomi-ai-commerce/internal/storecenter/observations"
)

const amazonOrdersPath = "/orders/2026-01-01/orders"

// SyntheticOrderReader is wired only by the isolated order checks. It requires
// an explicit fixture transport and exposes only the US Orders read operation.
type SyntheticOrderReader struct{ client Client }

func NewSyntheticOrderReader(fixture http.RoundTripper) (*SyntheticOrderReader, error) {
	if fixture == nil {
		return nil, ErrInvalid
	}
	return &SyntheticOrderReader{client: Client{
		userAgent: "Shuomi-Isolated-Orders/1.0",
		http: http.Client{Transport: fixture, Timeout: requestLimit,
			CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }},
	}}, nil
}

type orderSearchResult struct {
	Orders     []orderSearchEntry `json:"orders"`
	Pagination json.RawMessage    `json:"pagination"`
}

type orderSearchEntry struct {
	ID           string    `json:"orderId"`
	CreatedAt    time.Time `json:"createdTime"`
	UpdatedAt    time.Time `json:"lastUpdatedTime"`
	SalesChannel struct {
		Marketplace string `json:"marketplaceId"`
	} `json:"salesChannel"`
	Fulfillment struct {
		Status string `json:"fulfillmentStatus"`
	} `json:"fulfillment"`
	Items []struct {
		ID       string `json:"orderItemId"`
		Quantity int    `json:"quantityOrdered"`
		Product  struct {
			Title string `json:"title"`
			SKU   string `json:"sellerSku"`
		} `json:"product"`
	} `json:"orderItems"`
}

func (r *SyntheticOrderReader) Orders(ctx context.Context, token string, q o.AmazonOrderQuery) (o.AmazonOrderPage, error) {
	if r == nil || r.client.http.Transport == nil || q.Start.IsZero() || !q.End.After(q.Start) ||
		q.End.Sub(q.Start) > 30*24*time.Hour || q.Cursor != "" && !visibleASCII(q.Cursor, 2048) {
		return o.AmazonOrderPage{}, ErrInvalid
	}
	params := url.Values{
		"marketplaceIds": {USMarketplace}, "maxResultsPerPage": {strconv.Itoa(o.AmazonOrderPageSize)},
		"createdAfter": {q.Start.UTC().Format(time.RFC3339)}, "createdBefore": {q.End.UTC().Format(time.RFC3339)},
	}
	if q.Cursor != "" {
		params.Set("paginationToken", q.Cursor)
	}
	raw, metadata, err := r.client.api(ctx, http.MethodGet, amazonOrdersPath, params, token, nil)
	if err != nil {
		return o.AmazonOrderPage{}, err
	}
	if metadata.status == http.StatusBadRequest && q.Cursor != "" {
		return o.AmazonOrderPage{Incomplete: true}, nil
	}
	if err := responseStatus(metadata.status); err != nil {
		return o.AmazonOrderPage{}, err
	}
	var result orderSearchResult
	if json.Unmarshal(raw, &result) != nil || result.Orders == nil || len(result.Orders) > o.AmazonOrderPageSize {
		return o.AmazonOrderPage{}, ErrUnavailable
	}
	page := o.AmazonOrderPage{Orders: make([]o.AmazonOrder, 0, len(result.Orders))}
	if result.Pagination != nil {
		var pagination struct {
			Next string `json:"nextToken"`
		}
		if json.Unmarshal(result.Pagination, &pagination) != nil || !visibleASCII(pagination.Next, 2048) {
			page.Incomplete = true
		} else {
			page.Next = pagination.Next
		}
	}
	for _, entry := range result.Orders {
		if order, valid := projectAmazonOrder(entry); valid {
			page.Orders = append(page.Orders, order)
		} else {
			page.Incomplete = true
		}
	}
	return page, nil
}

func projectAmazonOrder(entry orderSearchEntry) (o.AmazonOrder, bool) {
	if entry.SalesChannel.Marketplace != USMarketplace || entry.Items == nil || len(entry.Items) > o.MaxAmazonOrderItems {
		return o.AmazonOrder{}, false
	}
	order := o.AmazonOrder{ID: entry.ID, CreatedAt: entry.CreatedAt.UTC(), UpdatedAt: entry.UpdatedAt.UTC(),
		Status: entry.Fulfillment.Status, Items: make([]o.AmazonOrderItem, 0, len(entry.Items))}
	for _, item := range entry.Items {
		order.Items = append(order.Items, o.AmazonOrderItem{ID: item.ID, Title: item.Product.Title,
			SKU: item.Product.SKU, Quantity: item.Quantity})
	}
	return order, order.Valid()
}
