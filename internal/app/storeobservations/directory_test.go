package storeobservationsapp

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"task-processor/internal/storecenter"
	o "task-processor/internal/storecenter/observations"
)

type mixedStoreDirectory struct {
	storecenter.Repository
	stores []storecenter.Store
}

func (r mixedStoreDirectory) List(_ context.Context, organization string, query storecenter.StoreListQuery) (storecenter.StorePage, error) {
	var out []storecenter.Store
	for _, store := range r.stores {
		if store.OrganizationID() == organization && (query.Platform == "" || query.Platform == store.Platform()) {
			out = append(out, store)
		}
	}
	return storecenter.StorePage{Stores: out, Total: int64(len(out))}, nil
}

func TestMixedDirectoryAllOrdersExcludeTemuAndProductsKeepBoth(t *testing.T) {
	repo := mixedStoreDirectory{}
	for _, platform := range []string{"shein", "temu", "walmart", "tiktokshop"} {
		store, err := storecenter.NewStore(storecenter.CreateStoreInput{ID: uuid.NewString(), OrganizationID: "org-a", ActorSubject: "actor-a", Name: platform, Platform: platform, Region: "US", CreateIdempotencyKey: uuid.NewString(), OccurredAt: time.Now()})
		require.NoError(t, err)
		repo.stores = append(repo.stores, *store)
	}
	d := Directory{Stores: repo}
	scope := o.Scope{OrganizationID: "org-a", ActorID: "actor-a", MemberID: "member-a"}
	orders, err := d.ListStores(context.Background(), scope, o.Orders, "")
	require.NoError(t, err)
	require.Equal(t, []string{repo.stores[0].ID(), repo.stores[2].ID(), repo.stores[3].ID()}, orders)
	products, err := d.ListStores(context.Background(), scope, o.Products, "")
	require.NoError(t, err)
	require.Len(t, products, 3)
	_, err = d.ListStores(context.Background(), scope, o.Kind("unsupported"), "")
	require.Error(t, err)
	temu, err := d.ListStores(context.Background(), scope, o.Products, "temu")
	require.NoError(t, err)
	require.Equal(t, []string{repo.stores[1].ID()}, temu)
	walmart, err := d.ListStores(context.Background(), scope, o.Orders, "walmart")
	require.NoError(t, err)
	require.Equal(t, []string{repo.stores[2].ID()}, walmart)
	shein, err := d.ListStores(context.Background(), scope, o.Orders, "shein")
	require.NoError(t, err)
	require.Equal(t, []string{repo.stores[0].ID()}, shein)
	tiktok, err := d.ListStores(context.Background(), scope, o.Orders, "tiktokshop")
	require.NoError(t, err)
	require.Equal(t, []string{repo.stores[3].ID()}, tiktok)
	_, err = d.ListStores(context.Background(), scope, o.Products, "tiktokshop")
	require.ErrorIs(t, err, o.ErrInvalid)
	_, err = d.ListStores(context.Background(), scope, o.Orders, "temu")
	require.ErrorIs(t, err, o.ErrInvalid)
}
