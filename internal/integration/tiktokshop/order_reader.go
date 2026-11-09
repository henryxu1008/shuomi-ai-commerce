package tiktokshop

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	o "task-processor/internal/storecenter/observations"
	"time"
)

// Orders is the fixed, read-only order search; tokens never become URLs.
func (c *OfficialClient) Orders(ctx context.Context, token, cipher string, in o.TikTokOrderQuery) (o.TikTokOrderPage, error) {
	if !bounded(cipher, 4096) || !bounded(token, 8192) || !in.Start.Before(in.End) || in.End.Sub(in.Start) > 48*time.Hour || in.Start.Nanosecond() != 0 || in.End.Nanosecond() != 0 || in.Start.Unix() <= 0 || len(in.Token) > 8192 || !o.Text(in.Token, 8192) {
		return o.TikTokOrderPage{}, o.ErrInvalid
	}
	q := url.Values{"shop_cipher": {cipher}, "page_size": {"100"}, "sort_field": {"create_time"}, "sort_order": {"ASC"}}
	if in.Token != "" {
		q.Set("page_token", in.Token)
	}
	body, _ := json.Marshal(struct {
		Start int64 `json:"create_time_ge"`
		End   int64 `json:"create_time_lt"`
	}{in.Start.Unix(), in.End.Unix()})
	raw, e := c.resourceReadRequest(ctx, http.MethodPost, "/order/202309/orders/search", q, body, token, 2<<20)
	if e != nil {
		return o.TikTokOrderPage{}, o.ErrUnavailable
	}
	return DecodeOrderPage(raw)
}
