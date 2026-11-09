package walmart

import (
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func TestOrderReadDoesNotBorrowItemScope(t *testing.T) {
	now := time.Now()
	d := TokenDetail{Valid: true, ExpiresAt: now.Add(time.Hour), Scopes: map[string]string{"item": "full_access"}}
	require.False(t, d.Allows("store_orders_read", now))
	d.Scopes = map[string]string{"orders": "view_only"}
	require.True(t, d.Allows("store_orders_read", now))
	require.False(t, d.Allows("store_products_read", now))
	require.False(t, d.Allows("unknown", now))
	d.Scopes["orders"] = "full_access"
	require.True(t, d.Allows("store_orders_read", now))
	d.ExpiresAt = now
	require.False(t, d.Allows("store_orders_read", now))
}
