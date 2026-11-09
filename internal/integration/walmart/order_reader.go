package walmart

import (
	"context"
	"net/http"
	"net/url"
	"strings"
	o "task-processor/internal/storecenter/observations"
	auth "task-processor/internal/storecenter/walmart"
	"time"
)

// Orders reads one page, never follows a provider URL or mutates an order.
func (c *OfficialClient) Orders(ctx context.Context, token auth.TokenSet, in o.WalmartOrderQuery) (o.WalmartOrderPage, error) {
	if in.Partition < 0 || in.Partition >= len(o.WalmartOrderPartitions) || !in.Start.Before(in.End) || in.End.Sub(in.Start) > 31*24*time.Hour || token.SellerID == "" || token.AccessToken == "" {
		return o.WalmartOrderPage{}, o.ErrInvalid
	}
	q := url.Values{"createdStartDate": {in.Start.UTC().Format(time.RFC3339)}, "createdEndDate": {in.End.UTC().Format(time.RFC3339)}, "shipNodeType": {o.WalmartOrderPartitions[in.Partition]}, "limit": {"30"}, "replacementInfo": {"true"}}
	if in.Cursor != "" {
		if len(in.Cursor) > 8192 || !strings.HasPrefix(in.Cursor, "?") || strings.ContainsAny(in.Cursor, "\r\n\x00#") {
			return o.WalmartOrderPage{}, o.ErrUnavailable
		}
		next, e := url.ParseQuery(in.Cursor[1:])
		if e != nil || len(next) == 0 {
			return o.WalmartOrderPage{}, o.ErrUnavailable
		}
		for key, values := range next {
			if len(values) != 1 || values[0] == "" || strings.ContainsAny(values[0], "\r\n\x00") {
				return o.WalmartOrderPage{}, o.ErrUnavailable
			}
			value := values[0]
			switch key {
			case "limit", "shipNodeType", "replacementInfo":
				if value != q.Get(key) {
					return o.WalmartOrderPage{}, o.ErrUnavailable
				}
			case "createdStartDate", "createdEndDate":
				date, e := time.Parse(time.RFC3339, value)
				fixed := in.Start
				if key == "createdEndDate" {
					fixed = in.End
				}
				if e != nil || !date.Equal(fixed) {
					return o.WalmartOrderPage{}, o.ErrUnavailable
				}
			case "sellerId":
				if value != token.SellerID {
					return o.WalmartOrderPage{}, o.ErrUnavailable
				}
				q.Set(key, value)
			case "poIndex", "soIndex", "hasMoreElements", "partnerId":
				q.Set(key, value)
			default:
				return o.WalmartOrderPage{}, o.ErrUnavailable
			}
		}
	}
	raw, status, e := c.request(ctx, http.MethodGet, "/v3/orders", q, nil, token, false, "")
	if e != nil || status != http.StatusOK {
		return o.WalmartOrderPage{}, o.ErrUnavailable
	}
	return DecodeOrderPage(raw)
}
