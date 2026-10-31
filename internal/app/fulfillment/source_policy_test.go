package fulfillmentapp

import (
	sf "task-processor/internal/storecenter/fulfillment"
	"testing"
	"time"
)

func TestUSNeedsOriginalSelfShippingPermission(t *testing.T) {
	now := time.Now()
	source := sf.Source{Kind: "consumer_order", ID: "order", Market: "shein-us", Type: 1, StockMode: 3, Status: 1, PrintStatus: 1, Logistics: []int{1}, Deadline: now.Add(time.Hour), Lines: []sf.Line{{ID: "1", SKU: "sku", Quantity: 1, GoodsStatus: 1, ExchangeTag: 1}}}
	if eligibleSource(source, now) == nil {
		t.Fatal("US label substituted for original shipping permission")
	}
	source.Logistics = []int{2}
	if err := eligibleSource(source, now); err != nil {
		t.Fatal(err)
	}
	source.StockMode = 2
	if eligibleSource(source, now) == nil {
		t.Fatal("platform fulfilled order became merchant dispatchable")
	}
	source.StockMode = 3
	source.Type = 5
	if eligibleSource(source, now) == nil {
		t.Fatal("certified warehouse order became merchant dispatchable")
	}
}

func TestManagedMarketUnknownAndNoShippingScene(t *testing.T) {
	now := time.Now()
	north := 2
	source := sf.Source{Kind: "platform_supply", ID: "purchase", Status: 2, Scene: 1, Type: 2, CountryMarket: &north, Deadline: now.Add(time.Hour), Lines: []sf.Line{{ID: "sku", SKU: "sku", Quantity: 5}}}
	if source.DisplayMarket() != "unknown" {
		t.Fatal("north region guessed as US")
	}
	if err := eligibleSource(source, now); err != nil {
		t.Fatal(err)
	}
	for _, scene := range []int{0, 3} {
		source.Scene = scene
		if eligibleSource(source, now) == nil {
			t.Fatal("JIT mother/no-shipping scene allowed")
		}
	}
}
func TestPartlyShippedConsumerOrderAllowsOnlyCurrentEligibleGoods(t *testing.T) {
	now := time.Now()
	source := sf.Source{Kind: "consumer_order", ID: "order", Market: "shein-us", Type: 1, StockMode: 3, Status: 4, PrintStatus: 1, SelectedLogistics: 2, Logistics: []int{2}, Deadline: now.Add(time.Hour), Lines: []sf.Line{{ID: "1", SKU: "first", Quantity: 1, GoodsStatus: 4, ExchangeTag: 1}, {ID: "2", SKU: "second", Quantity: 1, GoodsStatus: 2, ExchangeTag: 1}}}
	if err := eligibleSource(source, now); err != nil {
		t.Fatal("first pickup blocked remaining original goods", err)
	}
	if lines := eligibleSourceLines(source); len(lines) != 1 || lines[0].ID != "2" {
		t.Fatal("completed goods became a new allocation", lines)
	}
	source.Lines[1].GoodsStatus = 6
	if eligibleSource(source, now) == nil {
		t.Fatal("refunded goods remained eligible")
	}
}
