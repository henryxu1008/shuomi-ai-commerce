package observations

import (
	"context"
	"github.com/stretchr/testify/require"
	tm "task-processor/internal/marketplace/temu/observations"
	"task-processor/internal/storecenter"
	"testing"
)

func (m *serviceMerchant) TemuProducts(ctx context.Context, query ProductQuery) (ProductPage, error) {
	return m.Products(ctx, query.Page)
}

func TestNativeSyncStoresItsVariantAndPreservesOriginalMemberAndCursor(t *testing.T) {
	for _, mode := range []storecenter.OfficialMerchantMode{storecenter.MerchantLocal, storecenter.MerchantFullyManaged, storecenter.MerchantSemiManaged} {
		s, r, a, m := serviceFixture()
		r.sync.Binding.Platform = storecenter.PlatformTemu
		r.sync.Binding.MerchantMode = mode
		m.binding = r.sync.Binding
		p := tm.Product{ID: "21", ProductID: "10", Entity: "supplier_skc", Mode: string(mode), SaleState: "unknown", SKUs: []tm.SKU{}, Sites: []tm.Site{}}
		if mode == storecenter.MerchantLocal {
			p.Entity = "local_goods"
			p.ProductID = p.ID
		}
		m.products = ProductPage{Total: 1, TemuItems: []tm.Product{p}, TokenPagination: mode == storecenter.MerchantLocal}
		next := "next-page"
		if mode == storecenter.MerchantLocal {
			m.products.NextToken = &next
		}
		out, err := s.Step(context.Background(), r.sync.Owner.OrganizationID, r.sync.ID)
		require.NoError(t, err)
		require.Equal(t, 1, r.commits)
		require.Equal(t, r.sync.Owner, a.subject)
		require.Len(t, r.rows, 1)
		require.Nil(t, r.rows[0].Product)
		require.Equal(t, string(mode), r.rows[0].TemuProduct.Mode)
		if mode == storecenter.MerchantLocal {
			require.Equal(t, "running", out.Status)
			require.Equal(t, next, out.Progress.TemuPageToken)
		} else {
			require.Equal(t, "completed", out.Status)
		}
	}
}

func TestNativeSyncRevocationAfterProviderDoesNotCommitFacts(t *testing.T) {
	s, r, _, m := serviceFixture()
	r.sync.Binding.Platform = storecenter.PlatformTemu
	r.sync.Binding.MerchantMode = storecenter.MerchantLocal
	m.binding = r.sync.Binding
	m.products = ProductPage{Total: 1, TokenPagination: true, TemuItems: []tm.Product{{ID: "21", ProductID: "21", Entity: "local_goods", Mode: "local", SaleState: "unknown", SKUs: []tm.SKU{}, Sites: []tm.Site{}}}}
	m.after = func() { m.err = ErrForbidden }
	_, err := s.Step(context.Background(), r.sync.Owner.OrganizationID, r.sync.ID)
	require.NoError(t, err)
	require.Zero(t, r.commits)
	require.Equal(t, 1, r.stops)
}
