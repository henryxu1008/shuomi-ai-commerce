package observations

import (
	"context"
	"github.com/stretchr/testify/require"
	"task-processor/internal/marketplace/walmart"
	"task-processor/internal/storecenter"
	"testing"
)

func TestWalmartReadUsesOriginalObservationWorkerAndKeepsNativeUnknownFields(t *testing.T) {
	s, r, a, m := serviceFixture()
	r.sync.Binding.Platform = storecenter.PlatformWalmart
	r.sync.Binding.Site = "walmart-us"
	m.binding = r.sync.Binding
	m.products = ProductPage{Total: 1, WalmartItems: []walmart.Product{{SKU: "seller-sku", ProductName: "Actual title", PublicationStatus: "UNPUBLISHED", LifecycleStatus: "ACTIVE"}}}
	result, e := s.Step(context.Background(), r.sync.Owner.OrganizationID, r.sync.ID)
	require.NoError(t, e)
	require.Equal(t, "completed", result.Status)
	require.Len(t, r.rows, 1)
	require.Equal(t, r.sync.Owner, a.subject)
	require.Nil(t, r.rows[0].Product)
	require.Nil(t, r.rows[0].WalmartProduct.Price)
	require.Equal(t, "UNPUBLISHED", r.rows[0].WalmartProduct.PublicationStatus)
}
func TestWalmartOffsetBudgetRetainsPartialCoverage(t *testing.T) {
	s, r, _, m := serviceFixture()
	r.sync.Binding.Platform = storecenter.PlatformWalmart
	m.binding = r.sync.Binding
	r.sync.Progress.Page = 1000
	r.sync.Progress.Seen = 9990
	m.products = ProductPage{Total: 12000}
	for i := range 10 {
		m.products.WalmartItems = append(m.products.WalmartItems, walmart.Product{SKU: string(rune('a' + i))})
	}
	result, e := s.Step(context.Background(), r.sync.Owner.OrganizationID, r.sync.ID)
	require.NoError(t, e)
	require.Equal(t, "partial", result.Status)
	require.True(t, result.Progress.Incomplete)
}
