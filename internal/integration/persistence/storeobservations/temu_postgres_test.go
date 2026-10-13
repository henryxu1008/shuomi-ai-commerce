//go:build integration

package storeobservations

import (
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	tm "task-processor/internal/marketplace/temu/observations"
	"task-processor/internal/storecenter"
	o "task-processor/internal/storecenter/observations"
	"testing"
	"time"
)

func TestTemuObservationOriginalOwnerPersistsAndQueriesAllModesWithoutSHEINFacts(t *testing.T) {
	ctx, owner, r := observationDatabase(t)
	scope := o.Scope{OrganizationID: "org-native", ActorID: "actor-a", MemberID: "member-a"}
	for _, mode := range []storecenter.OfficialMerchantMode{storecenter.MerchantFullyManaged, storecenter.MerchantSemiManaged, storecenter.MerchantLocal} {
		store := uuid.NewString()
		in, children := syncInput(scope, store)
		children[0].Binding.Platform = storecenter.PlatformTemu
		children[0].Binding.MerchantMode = mode
		command, err := r.Begin(ctx, scope, uuid.NewString(), string(mode), in, children)
		require.NoError(t, err)
		sync := command.Syncs[0]
		p := tm.Product{ID: "9007199254740994", ProductID: "9007199254740993", Entity: "supplier_skc", Mode: string(mode), Title: "Native title", SaleState: "unknown", SKUs: []tm.SKU{{ID: "9007199254740995", ExternalCode: "native-external", StockKind: "virtualStock"}}, Sites: []tm.Site{}}
		if mode == storecenter.MerchantLocal {
			p.Entity = "local_goods"
			p.ProductID = p.ID
			p.SaleState = "active"
			p.SKUs[0].StockKind = "stock"
			zero := int64(0)
			p.SKUs[0].Stock = &zero
		}
		_, err = r.CommitPage(ctx, sync, sync.Progress, []o.Record{{ID: p.ID, StoreID: store, SyncID: sync.ID, TemuProduct: &p}}, "completed", time.Now().UTC())
		require.NoError(t, err)
		read, err := r.Record(ctx, scope.OrganizationID, store, o.Products, sync.ID, p.ID)
		require.NoError(t, err)
		require.Nil(t, read.Product)
		require.Equal(t, p.ProductID, read.TemuProduct.ProductID)
		page, err := r.List(ctx, scope.OrganizationID, o.Query{Kind: o.Products, Limit: 10, Keyword: p.ProductID, Sources: map[string]string{store: sync.ID}})
		require.NoError(t, err)
		require.Len(t, page.Items, 1)
		require.Equal(t, 1, page.Summary.Total)
		require.Nil(t, page.Items[0].Product)
		if mode == storecenter.MerchantLocal {
			require.Equal(t, 1, page.Summary.Active)
		} else {
			require.Equal(t, 1, page.Summary.Unknown)
		}
		aggregates, err := r.StoreAggregates(ctx, scope.OrganizationID, o.Products, map[string]string{store: sync.ID}, true)
		require.NoError(t, err)
		require.Len(t, aggregates[store].Inventory, 1)
		if mode == storecenter.MerchantLocal {
			require.EqualValues(t, 1, aggregates[store].Active)
			require.NotNil(t, aggregates[store].Inventory[0].Quantity)
			require.Zero(t, *aggregates[store].Inventory[0].Quantity)
			threshold := int64(5)
			metric := o.LowInventory(aggregates[store].Inventory, &threshold)
			require.True(t, metric.Complete, "known native SKU stock is not a missing physical warehouse")
			require.Zero(t, metric.Unknown)
			require.EqualValues(t, 1, *metric.Value, "explicit zero stock is below the configured threshold")
			child := children[0]
			child.ID, child.Key = uuid.NewString(), uuid.NewString()
			missing, e := r.Begin(ctx, scope, uuid.NewString(), "local-missing", in, []o.Sync{child})
			require.NoError(t, e)
			p.SKUs[0].Stock = nil
			missingSync := missing.Syncs[0]
			_, e = r.CommitPage(ctx, missingSync, missingSync.Progress, []o.Record{{ID: p.ID, StoreID: store, SyncID: missingSync.ID, TemuProduct: &p}}, "completed", time.Now().UTC())
			require.NoError(t, e)
			unknown, e := r.StoreAggregates(ctx, scope.OrganizationID, o.Products, map[string]string{store: missingSync.ID}, true)
			require.NoError(t, e)
			metric = o.LowInventory(unknown[store].Inventory, &threshold)
			require.False(t, metric.Complete)
			require.EqualValues(t, 1, metric.Unknown, "missing stock cannot become zero")
		} else {
			require.EqualValues(t, 1, aggregates[store].StatusUnknown)
			require.Nil(t, aggregates[store].Inventory[0].Quantity, "virtual stock does not prove available warehouse inventory")
		}
		empty, err := r.List(ctx, "other-org", o.Query{Kind: o.Products, Limit: 10, Sources: map[string]string{store: sync.ID}})
		require.NoError(t, err)
		require.Empty(t, empty.Items)
	}
	require.NoError(t, owner.Exec("CREATE SCHEMA shein_observations").Error)
	require.Error(t, Install(ctx, owner), "fresh owner cannot wrap or migrate the retired schema")
	require.Error(t, VerifyRuntime(ctx, r.db), "serving rejects a retained retired owner")
}
