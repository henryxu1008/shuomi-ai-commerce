package shein

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"
)

func TestSupplyQuantitiesAreExactAndMissingReceiptFactsStayUnknown(t *testing.T) {
	raw := `{"code":"0","info":{"count":1,"list":[{"orderNo":"original","status":2,"sceneType":1,"type":2,"countryMarket":null,"requestReceiptTime":"2026-11-05 12:00:00","orderExtends":[{"skuCode":"sku","orderQuantity":5.0,"deliveryQuantity":0}]}]}}`
	c := testClient(t, roundTripFunc(func(r *http.Request) (*http.Response, error) { return observationResponse(raw), nil }))
	source, err := c.QueryFulfillmentSource(context.Background(), goodsCredential(), "platform_supply", "original")
	if err != nil || len(source.Lines) != 1 || source.Lines[0].Quantity != 5 {
		t.Fatal(source, err)
	}
	encoded, _ := json.Marshal(source)
	var facts struct {
		Market *int `json:"countryMarket"`
		Lines  []struct {
			Received *int64 `json:"received"`
		} `json:"lines"`
	}
	_ = json.Unmarshal(encoded, &facts)
	if facts.Market != nil || facts.Lines[0].Received != nil || source.DisplayMarket() != "unknown" {
		t.Fatal("missing facts converted into known zero or US")
	}
	raw = strings.Replace(raw, `"deliveryQuantity":0`, `"otherField":0`, 1)
	if _, err = c.QueryFulfillmentSource(context.Background(), goodsCredential(), "platform_supply", "original"); err == nil {
		t.Fatal("missing dispatched quantity guessed as zero")
	}
}
