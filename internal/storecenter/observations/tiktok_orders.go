package observations

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"regexp"
	"time"
)

// TikTokOrder contains only the original order-level product subtotal. Line,
// package, buyer and payment-instrument details are deliberately not retained.
type TikTokOrder struct {
	ID     string `json:"id"`
	Site   string `json:"site"`
	Date   string `json:"date"`
	Status string `json:"status,omitempty"`
	Type   string `json:"type,omitempty"`
	Amount *Price `json:"amount"`
}

var tikTokOrderDecimal = regexp.MustCompile(`^[0-9]{1,20}(\.[0-9]{1,8})?$`)

func (o TikTokOrder) Valid() bool {
	if !Identity(o.ID) || o.Site != "tiktokshop-us" || !Text(o.Status, 64) || !Text(o.Type, 64) {
		return false
	}
	if o.Date != "" {
		v, e := time.Parse(time.RFC3339, o.Date)
		if e != nil || v.Year() < 1 || v.Year() > 9999 || v.Nanosecond() != 0 {
			return false
		}
	}
	return o.Amount == nil || (o.Amount.Special == "" && (o.Amount.Value == "" || tikTokOrderDecimal.MatchString(o.Amount.Value)) && (o.Amount.Currency == "" || orderAmountCurrency.MatchString(o.Amount.Currency)))
}

type TikTokOrderPage struct {
	Total     int
	Items     []TikTokOrder
	NextToken string
}
type TikTokOrderQuery struct {
	Start, End time.Time
	Token      string
}

// No network implementation is registered in this source-preparation phase.
type TikTokOrderReader interface {
	ReadTikTokOrders(context.Context, Scope, Binding, TikTokOrderQuery) (TikTokOrderPage, error)
}
type TikTokOrderProgress struct {
	Token       string   `json:"token,omitempty"`
	TokenHashes []string `json:"tokenHashes,omitempty"`
	Total       *int     `json:"total,omitempty"`
	Complete    bool     `json:"complete"`
}

func advanceTikTokOrders(current Checkpoint, count, total int, token string) (Checkpoint, string, error) {
	c := copyCheckpoint(current)
	if c.TikTokOrders == nil {
		c.TikTokOrders = &TikTokOrderProgress{}
	}
	p := c.TikTokOrders
	if p.Complete || c.Page < 1 || len(c.Windows) == 0 || count < 0 || count > 100 || total < 0 || len(token) > 8192 || !Text(token, 8192) {
		return c, "", ErrInvalid
	}
	c.Pages++
	c.Seen += count
	if p.Total == nil {
		v := total
		p.Total = &v
	} else if *p.Total != total {
		c.Note("tiktok_order_count_changed")
	}
	if total > 10000 || c.Seen > 10000 || c.Pages >= 30000 {
		c.Note("tiktok_order_limit")
		return c, "partial", nil
	}
	if token != "" {
		hash := sha256.Sum256([]byte(token))
		digest := hex.EncodeToString(hash[:])
		for _, old := range p.TokenHashes {
			if old == digest {
				c.Note("provider_cursor_cycle")
				return c, "partial", nil
			}
		}
		if c.Page >= 512 || len(p.TokenHashes) >= 512 {
			c.Note("provider_checkpoint_limit")
			return c, "partial", nil
		}
		p.TokenHashes = append(p.TokenHashes, digest)
		p.Token = token
		c.Page++
		return c, "running", nil
	}
	if c.Seen != *p.Total {
		c.Note("tiktok_order_count_mismatch")
	}
	p.Token = ""
	p.TokenHashes = nil
	p.Total = nil
	c.Page = 1
	c.Seen = 0
	c.Windows = c.Windows[1:]
	if len(c.Windows) > 0 {
		return c, "running", nil
	}
	p.Complete = !c.Incomplete
	return finish(c)
}
func (s *Service) stepTikTokOrders(ctx context.Context, source Sync, merchant Merchant) (Sync, error) {
	if s.TikTokOrders == nil || source.Range == nil || len(source.Progress.Windows) == 0 {
		return Sync{}, ErrUnavailable
	}
	c := copyCheckpoint(source.Progress)
	w := c.Windows[0]
	token := ""
	if c.TikTokOrders != nil {
		token = c.TikTokOrders.Token
	}
	page, e := s.TikTokOrders.ReadTikTokOrders(ctx, source.Owner, source.Binding, TikTokOrderQuery{Start: w.Start, End: w.End, Token: token})
	if e != nil {
		return Sync{}, e
	}
	if len(page.Items) > 100 {
		return Sync{}, ErrUnavailable
	}
	rows := []Record{}
	seen := map[string]bool{}
	for _, order := range page.Items {
		if !order.Valid() || seen[order.ID] {
			return Sync{}, ErrUnavailable
		}
		seen[order.ID] = true
		if order.Date != "" {
			date, _ := time.Parse(time.RFC3339, order.Date)
			if date.Before(w.Start) || !date.Before(w.End) {
				return Sync{}, ErrUnavailable
			}
		}
		value := order
		rows = append(rows, Record{StoreID: source.StoreID, SyncID: source.ID, ID: order.ID, TikTokOrder: &value, WindowKey: w.Key(), ObservedAt: s.now()})
	}
	if e = s.Access.Authorize(ctx, source.Owner, Orders, true); e != nil {
		return Sync{}, e
	}
	if e = merchant.Check(ctx); e != nil {
		return Sync{}, e
	}
	if merchant.Binding() != source.Binding {
		return Sync{}, ErrConflict
	}
	next, status, e := advanceTikTokOrders(c, len(page.Items), page.Total, page.NextToken)
	if e != nil {
		return Sync{}, e
	}
	return s.Repository.CommitPage(ctx, source, next, rows, status, s.now())
}
