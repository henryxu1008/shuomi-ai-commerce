package fulfillment

import (
	"testing"
	"time"
)

func TestUSNeedsOriginalSelfShippingPermission(t *testing.T) {
	now := time.Now()
	source := Source{Kind: "consumer_order", ID: "order", Market: "shein-us", Type: 1, StockMode: 3, Status: 1, PrintStatus: 1, Logistics: []int{1}, Deadline: now.Add(time.Hour), Lines: []Line{{ID: "1", SKU: "sku", Quantity: 1, GoodsStatus: 1, ExchangeTag: 1}}}
	if source.Eligible(now) == nil {
		t.Fatal("US label substituted for original shipping permission")
	}
	source.Logistics = []int{2}
	if err := source.Eligible(now); err != nil {
		t.Fatal(err)
	}
	source.StockMode = 2
	if source.Eligible(now) == nil {
		t.Fatal("platform fulfilled order became merchant dispatchable")
	}
	source.StockMode = 3
	source.Type = 5
	if source.Eligible(now) == nil {
		t.Fatal("certified warehouse order became merchant dispatchable")
	}
}

func TestManagedMarketUnknownAndNoShippingScene(t *testing.T) {
	now := time.Now()
	north := 2
	source := Source{Kind: "platform_supply", ID: "purchase", Status: 2, Scene: 1, Type: 2, CountryMarket: &north, Deadline: now.Add(time.Hour), Lines: []Line{{ID: "sku", SKU: "sku", Quantity: 5}}}
	if source.DisplayMarket() != "unknown" {
		t.Fatal("north region guessed as US")
	}
	if err := source.Eligible(now); err != nil {
		t.Fatal(err)
	}
	for _, scene := range []int{0, 3} {
		source.Scene = scene
		if source.Eligible(now) == nil {
			t.Fatal("JIT mother/no-shipping scene allowed")
		}
	}
}
