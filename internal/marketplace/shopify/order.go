package shopify

import (
	"regexp"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"
)

// ObservedOrder retains only the fields approved for the order records view.
type ObservedOrder struct {
	ID        string              `json:"id"`
	Number    string              `json:"number"`
	CreatedAt time.Time           `json:"createdAt"`
	UpdatedAt time.Time           `json:"updatedAt"`
	Status    string              `json:"status"`
	Cancelled bool                `json:"cancelled"`
	Items     []ObservedOrderItem `json:"items"`
}

const MaxObservedOrderItems = 250

type ObservedOrderItem struct {
	ID       string `json:"id"`
	Title    string `json:"title"`
	SKU      string `json:"sku"`
	Quantity int    `json:"quantity"`
}

var orderID = regexp.MustCompile(`^[1-9][0-9]{0,19}$`)
var orderLineGID = regexp.MustCompile(`^gid://shopify/LineItem/[1-9][0-9]{0,19}$`)
var orderStatusCode = regexp.MustCompile(`^[A-Z][A-Z0-9_]{0,63}$`)

func (o ObservedOrder) Valid() bool {
	if !orderID.MatchString(o.ID) || o.Number == "" || !validOrderText(o.Number, 128) || !orderStatusCode.MatchString(o.Status) ||
		o.CreatedAt.IsZero() || o.UpdatedAt.Before(o.CreatedAt) || o.CreatedAt.Year() > 9999 || o.UpdatedAt.Year() > 9999 ||
		o.Items == nil || len(o.Items) > MaxObservedOrderItems {
		return false
	}
	seen := make(map[string]struct{}, len(o.Items))
	for _, item := range o.Items {
		if !orderLineGID.MatchString(item.ID) || item.Title == "" || !validOrderText(item.Title, 255) ||
			!validOrderText(item.SKU, 255) || item.Quantity < 1 || item.Quantity > 2147483647 {
			return false
		}
		if _, duplicate := seen[item.ID]; duplicate {
			return false
		}
		seen[item.ID] = struct{}{}
	}
	return true
}

func validOrderText(value string, maximum int) bool {
	return utf8.ValidString(value) && strings.TrimSpace(value) == value &&
		utf8.RuneCountInString(value) <= maximum && !strings.ContainsFunc(value, unicode.IsControl)
}

type OrderQuery struct {
	Start, End time.Time
	Cursor     string
}
type OrderPage struct {
	Orders     []ObservedOrder
	Next       string
	Incomplete bool
}
