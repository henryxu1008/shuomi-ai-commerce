package supplymarketpersistence

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"task-processor/internal/product/supplymarket"
	"testing"
	"time"
)

func TestMarketFiltersBeforeCountAndPaging(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	container, err := tcpostgres.Run(ctx, "postgres:16-alpine", tcpostgres.WithDatabase("market_filters"), tcpostgres.WithUsername("supply_owner"), tcpostgres.WithPassword("isolated-filter-test"), tcpostgres.BasicWaitStrategies())
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, container.Terminate(context.Background())) })
	dsn, err := container.ConnectionString(ctx, "sslmode=disable")
	require.NoError(t, err)
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	require.NoError(t, err)
	pool, err := db.DB()
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, pool.Close()) })
	require.NoError(t, InstallSchema(db))
	r := &Repository{db: db}
	id := func(n int) string { return fmt.Sprintf("00000000-0000-4000-8000-%012d", n) }
	variant := func(currency string, price float64) supplymarket.PublicVariant {
		return supplymarket.PublicVariant{Currency: currency, Price: &price}
	}
	seed := func(n int, channel, province string, moq, days int, active bool, variants ...supplymarket.PublicVariant) {
		record := uuid.NewString()
		now := time.Now().UTC()
		release := supplymarket.Release{ID: id(n), Channel: channel, Revision: 1, Active: active, PublishedAt: now, Product: supplymarket.PublicProduct{Title: "筛选商品", Images: []string{"https://images.example/product.png"}, Variants: variants}, Supply: supplymarket.SupplyDeclaration{Stock: 10, Province: province, City: "示例城市", MinimumQuantity: moq, LeadDays: days, PriceNote: "50元仅为说明，不替代款式价格"}}
		raw, e := json.Marshal(release)
		require.NoError(t, e)
		require.NoError(t, db.Exec("INSERT INTO supply_market_records(id,organization_id,actor_id,member_id,kind,stage,revision,record_json,created_at) VALUES(?,'org-a','actor-a','member-a',?,'APPROVED',1,'{}',?)", record, channel, now).Error)
		require.NoError(t, db.Exec("INSERT INTO supply_market_releases(id,record_id,organization_id,actor_id,member_id,channel,revision,active,release_json,source_json,published_at) VALUES(?,?,'org-a','actor-a','member-a',?,1,?,?::jsonb,'{}',?)", id(n), record, channel, active, string(raw), now).Error)
	}
	seed(1, "official", "浙江省", 2, 0, true, variant("CNY", 10), variant("CNY", 100))
	seed(2, "official", "浙江省", 2, 0, true, variant("CNY", 50))
	seed(3, "official", "浙江省", 2, 0, true, variant("USD", 50))
	seed(4, "selected", "浙江省", 2, 0, true, variant("CNY", 50))
	seed(5, "official", "广东省", 5, 7, true, variant("CNY", 50))
	seed(6, "official", "浙江省", 2, 0, true, variant("CNY", 0))
	seed(7, "official", "浙江省", 2, 0, true)
	seed(8, "official", "浙江省", 2, 0, false, variant("CNY", 50))
	seed(9, "official", "浙江省", 2, 0, true, variant("CNY", 10), variant("USD", 50))
	seed(10, "official", "浙江省", 2, 0, true, variant("CNY", 55))
	q := supplymarket.Query{Kind: "official", Limit: 1, Filters: supplymarket.MarketFilters{Province: "浙江省", MinPrice: new(40.0), MaxPrice: new(60.0), PriceCurrency: "CNY", MaxMOQ: new(2), MaxLeadDays: new(0)}}
	first, err := r.ListReleases(ctx, q)
	require.NoError(t, err)
	require.EqualValues(t, 2, first.Total)
	require.Len(t, first.Items, 1)
	require.Equal(t, id(2), first.Items[0].ID)
	require.Equal(t, id(2), first.NextCursor)
	q.After = first.NextCursor
	second, err := r.ListReleases(ctx, q)
	require.NoError(t, err)
	require.EqualValues(t, 2, second.Total)
	require.Len(t, second.Items, 1)
	require.Equal(t, id(10), second.Items[0].ID)
	require.Empty(t, second.NextCursor)
	q.After = ""
	q.Limit = 20
	for _, test := range []struct {
		name    string
		filters supplymarket.MarketFilters
		ids     []string
	}{
		{"zero is a price", supplymarket.MarketFilters{MinPrice: new(0.0), MaxPrice: new(0.0), PriceCurrency: "CNY"}, []string{id(6)}},
		{"currency belongs to the same matching variant", supplymarket.MarketFilters{MinPrice: new(50.0), MaxPrice: new(50.0), PriceCurrency: "USD"}, []string{id(3), id(9)}},
		{"no-price records can match non-price filters", supplymarket.MarketFilters{Province: "浙江省", MaxMOQ: new(2), MaxLeadDays: new(0)}, []string{id(1), id(2), id(3), id(6), id(7), id(9), id(10)}},
		{"minimum only", supplymarket.MarketFilters{MinPrice: new(90.0), PriceCurrency: "CNY"}, []string{id(1)}},
		{"all conditions apply", supplymarket.MarketFilters{Province: "广东省", MaxMOQ: new(2)}, []string{}},
	} {
		t.Run(test.name, func(t *testing.T) {
			q.Filters = test.filters
			page, e := r.ListReleases(ctx, q)
			require.NoError(t, e)
			actual := []string{}
			for _, item := range page.Items {
				actual = append(actual, item.ID)
			}
			require.Equal(t, test.ids, actual)
			require.EqualValues(t, len(test.ids), page.Total)
		})
	}
	q.Kind = "selected"
	q.Filters = supplymarket.MarketFilters{Province: "浙江省", MaxMOQ: new(2), MaxLeadDays: new(0)}
	selected, err := r.ListReleases(ctx, q)
	require.NoError(t, err)
	require.Len(t, selected.Items, 1)
	require.Equal(t, id(4), selected.Items[0].ID)
}
