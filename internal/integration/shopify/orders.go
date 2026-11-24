package shopify

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"sort"
	"strings"
	"time"

	o "github.com/henryxu1008/shuomi-ai-commerce/internal/storecenter/observations"
)

const shopifyOrdersPageSize = 10
const shopifyOrderLinePageSize = 25
const shopifyOrderLineReadLimit = 8

const orderLineSelection = `pageInfo { hasNextPage endCursor } nodes { id title sku quantity }`

const orderPageQuery = `query OrderObservations($window: String!, $after: String) {
  orders(first: 10, query: $window, sortKey: CREATED_AT, after: $after) {
    pageInfo { hasNextPage endCursor }
    nodes { id name createdAt updatedAt cancelledAt displayFulfillmentStatus
      lineItems(first: 25) { ` + orderLineSelection + ` }
    }
  }
}`

const orderLinePageQuery = `query OrderObservationLines($id: ID!, $after: String!) {
  order(id: $id) {
    id updatedAt lineItems(first: 25, after: $after) { ` + orderLineSelection + ` }
  }
}`

var ErrOrderPageUnavailable = errors.New("Shopify order page unavailable")

// SyntheticOrderReader has no default transport and is only injected into
// isolated fixtures. No Shopify order credential or transport is configured in
// Current Application for this delivery.
type SyntheticOrderReader struct {
	admin *AdminClient
}

func NewSyntheticOrderReader(fixture http.RoundTripper) (*SyntheticOrderReader, error) {
	if fixture == nil {
		return nil, ErrInvalidRequest
	}
	return &SyntheticOrderReader{admin: NewDevelopmentAdminClient(&http.Client{Transport: fixture})}, nil
}

type orderPageInfo struct {
	HasNext *bool  `json:"hasNextPage"`
	End     string `json:"endCursor"`
}

func (p *orderPageInfo) next() (string, bool) {
	if p == nil || p.HasNext == nil || !validOrderCursor(p.End) || *p.HasNext && p.End == "" {
		return "", false
	}
	if *p.HasNext {
		return p.End, true
	}
	return "", true
}

func validOrderCursor(s string) bool {
	return len(s) <= 2048 && (s == "" || validToken(s))
}

type orderLines struct {
	Page  *orderPageInfo       `json:"pageInfo"`
	Nodes []o.ShopifyOrderItem `json:"nodes"`
}

type orderNode struct {
	ID          string          `json:"id"`
	Name        string          `json:"name"`
	CreatedAt   time.Time       `json:"createdAt"`
	UpdatedAt   time.Time       `json:"updatedAt"`
	CancelledAt json.RawMessage `json:"cancelledAt"`
	Status      string          `json:"displayFulfillmentStatus"`
	Lines       *orderLines     `json:"lineItems"`
}

func (r *SyntheticOrderReader) Orders(ctx context.Context, domain, token string, q o.ShopifyOrderQuery, authorize func(context.Context) error) (o.ShopifyOrderPage, error) {
	if r == nil || authorize == nil || q.Start.IsZero() || !q.End.After(q.Start) || q.End.Sub(q.Start) > 30*24*time.Hour ||
		!validOrderCursor(q.Cursor) || q.Start.Year() < 1 || q.End.Year() > 9999 {
		return o.ShopifyOrderPage{}, ErrInvalidRequest
	}
	if err := authorize(ctx); err != nil {
		return o.ShopifyOrderPage{}, err
	}
	window := "created_at:>='" + q.Start.UTC().Format(time.RFC3339Nano) + "' created_at:<'" + q.End.UTC().Format(time.RFC3339Nano) + "'"
	var after *string
	if q.Cursor != "" {
		after = &q.Cursor
	}
	var data struct {
		Orders *struct {
			Page  *orderPageInfo `json:"pageInfo"`
			Nodes []orderNode    `json:"nodes"`
		} `json:"orders"`
	}
	if err := r.admin.query(ctx, domain, token, orderPageQuery, map[string]any{"window": window, "after": after}, &data); err != nil || data.Orders == nil ||
		data.Orders.Nodes == nil || len(data.Orders.Nodes) > shopifyOrdersPageSize {
		return o.ShopifyOrderPage{}, ErrOrderPageUnavailable
	}
	next, ok := data.Orders.Page.next()
	if !ok || next != "" && (next == q.Cursor || len(data.Orders.Nodes) == 0) {
		return o.ShopifyOrderPage{Incomplete: true}, nil
	}
	out := o.ShopifyOrderPage{Next: next, Orders: make([]o.ShopifyOrder, 0, len(data.Orders.Nodes))}
	lineReads := 0
	for _, node := range data.Orders.Nodes {
		if !validGID(node.ID, "Order") || node.CreatedAt.Before(q.Start) || !node.CreatedAt.Before(q.End) {
			out.Incomplete = true
			continue
		}
		var cancelledAt *time.Time
		if len(node.CancelledAt) == 0 || json.Unmarshal(node.CancelledAt, &cancelledAt) != nil ||
			cancelledAt != nil && (cancelledAt.Before(node.CreatedAt) || cancelledAt.After(node.UpdatedAt)) {
			out.Incomplete = true
			continue
		}
		items, err := r.readOrderLines(ctx, domain, token, node, &lineReads, authorize)
		if errors.Is(err, ErrOrderPageUnavailable) {
			out.Incomplete = true
			continue
		}
		if err != nil {
			return o.ShopifyOrderPage{}, err
		}
		order := o.ShopifyOrder{ID: strings.TrimPrefix(node.ID, "gid://shopify/Order/"), Number: strings.TrimSpace(node.Name), CreatedAt: node.CreatedAt.UTC(),
			UpdatedAt: node.UpdatedAt.UTC(), Status: node.Status, Cancelled: cancelledAt != nil, Items: items}
		if !order.Valid() {
			out.Incomplete = true
			continue
		}
		out.Orders = append(out.Orders, order)
	}
	return out, nil
}

func (r *SyntheticOrderReader) readOrderLines(ctx context.Context, domain, token string, node orderNode, reads *int, authorize func(context.Context) error) ([]o.ShopifyOrderItem, error) {
	items := []o.ShopifyOrderItem{}
	page := node.Lines
	seen := map[string]bool{}
	for {
		if page == nil || page.Nodes == nil || len(page.Nodes) > shopifyOrderLinePageSize || len(items)+len(page.Nodes) > o.MaxShopifyOrderItems {
			return nil, ErrOrderPageUnavailable
		}
		items = append(items, page.Nodes...)
		next, ok := page.Page.next()
		if !ok || next != "" && (len(page.Nodes) == 0 || seen[next] || *reads == shopifyOrderLineReadLimit) {
			return nil, ErrOrderPageUnavailable
		}
		if next == "" {
			sort.Slice(items, func(i, j int) bool { return items[i].ID < items[j].ID })
			return items, nil
		}
		if err := authorize(ctx); err != nil {
			return nil, err
		}
		seen[next] = true
		(*reads)++
		var data struct {
			Order *orderNode `json:"order"`
		}
		if err := r.admin.query(ctx, domain, token, orderLinePageQuery, map[string]string{"id": node.ID, "after": next}, &data); err != nil ||
			data.Order == nil || data.Order.ID != node.ID || !data.Order.UpdatedAt.Equal(node.UpdatedAt) {
			return nil, ErrOrderPageUnavailable
		}
		page = data.Order.Lines
	}
}
