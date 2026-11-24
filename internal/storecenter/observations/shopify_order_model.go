package observations

import (
	"regexp"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"
)

// ShopifyOrder retains only the fields approved for the order records view.
type ShopifyOrder struct {
	ID        string             `json:"id"`
	Number    string             `json:"number"`
	CreatedAt time.Time          `json:"createdAt"`
	UpdatedAt time.Time          `json:"updatedAt"`
	Status    string             `json:"status"`
	Cancelled bool               `json:"cancelled"`
	Items     []ShopifyOrderItem `json:"items"`
}

const MaxShopifyOrderItems = 250

type ShopifyOrderItem struct {
	ID       string `json:"id"`
	Title    string `json:"title"`
	SKU      string `json:"sku"`
	Quantity int    `json:"quantity"`
}

var shopifyOrderID = regexp.MustCompile(`^[1-9][0-9]{0,19}$`)
var shopifyOrderLineGID = regexp.MustCompile(`^gid://shopify/LineItem/[1-9][0-9]{0,19}$`)
var platformOrderStatusCode = regexp.MustCompile(`^[A-Z][A-Z0-9_]{0,63}$`)

func (o ShopifyOrder) Valid() bool {
	if !shopifyOrderID.MatchString(o.ID) || o.Number == "" || !validObservedOrderText(o.Number, 128) || !platformOrderStatusCode.MatchString(o.Status) ||
		o.CreatedAt.IsZero() || o.UpdatedAt.Before(o.CreatedAt) || o.CreatedAt.Year() > 9999 || o.UpdatedAt.Year() > 9999 ||
		o.Items == nil || len(o.Items) > MaxShopifyOrderItems {
		return false
	}
	seen := make(map[string]struct{}, len(o.Items))
	for _, item := range o.Items {
		if !shopifyOrderLineGID.MatchString(item.ID) || item.Title == "" || !validObservedOrderText(item.Title, 255) ||
			!validObservedOrderText(item.SKU, 255) || item.Quantity < 1 || item.Quantity > 2147483647 {
			return false
		}
		if _, duplicate := seen[item.ID]; duplicate {
			return false
		}
		seen[item.ID] = struct{}{}
	}
	return true
}

func validObservedOrderText(value string, maximum int) bool {
	return utf8.ValidString(value) && strings.TrimSpace(value) == value &&
		utf8.RuneCountInString(value) <= maximum && !strings.ContainsFunc(value, unicode.IsControl)
}

type ShopifyOrderQuery struct {
	Start, End time.Time
	Cursor     string
}
type ShopifyOrderPage struct {
	Orders     []ShopifyOrder
	Next       string
	Incomplete bool
}
