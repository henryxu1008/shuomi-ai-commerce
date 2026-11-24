package observations

import (
	"regexp"
	"time"
)

const MaxAmazonOrderItems = 250
const AmazonOrderPageSize = 10

var amazonOrderID = regexp.MustCompile(`^[0-9]{3}-[0-9]{7}-[0-9]{7}$`)

type AmazonOrder struct {
	ID        string            `json:"id"`
	CreatedAt time.Time         `json:"createdAt"`
	UpdatedAt time.Time         `json:"updatedAt"`
	Status    string            `json:"status"`
	Items     []AmazonOrderItem `json:"items"`
}

type AmazonOrderItem struct {
	ID       string `json:"id"`
	Title    string `json:"title"`
	SKU      string `json:"sku"`
	Quantity int    `json:"quantity"`
}

func (o AmazonOrder) Valid() bool {
	if !amazonOrderID.MatchString(o.ID) || o.CreatedAt.IsZero() || o.UpdatedAt.Before(o.CreatedAt) ||
		o.CreatedAt.Year() > 9999 || o.UpdatedAt.Year() > 9999 ||
		o.Status != "" && !platformOrderStatusCode.MatchString(o.Status) ||
		o.Items == nil || len(o.Items) > MaxAmazonOrderItems {
		return false
	}
	seen := make(map[string]bool, len(o.Items))
	for _, item := range o.Items {
		if !Identity(item.ID) || !validObservedOrderText(item.Title, 255) ||
			!validObservedOrderText(item.SKU, 255) || item.Quantity < 1 || item.Quantity > 2147483647 ||
			seen[item.ID] {
			return false
		}
		seen[item.ID] = true
	}
	return true
}

type AmazonOrderQuery struct {
	Start, End time.Time
	Cursor     string
}

type AmazonOrderPage struct {
	Orders     []AmazonOrder
	Next       string
	Incomplete bool
}

func validAmazonOrderStatus(s string) bool {
	switch s {
	case "", "unknown", "PENDING_AVAILABILITY", "PENDING", "UNSHIPPED", "PARTIALLY_SHIPPED", "SHIPPED", "CANCELLED", "UNFULFILLABLE":
		return true
	default:
		return false
	}
}
