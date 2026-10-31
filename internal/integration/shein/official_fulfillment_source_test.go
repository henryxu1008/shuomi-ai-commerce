package shein

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"
)

func TestCompletedGoodsWithoutTimeoutDoNotHideRemainingConsumerGoods(t *testing.T) {
	deadline := time.Now().Add(time.Hour).Format("2006-01-02T15:04:05.999Z0700")
	raw := `{"code":"0","info":[{"orderNo":"original","salesSite":"shein-us","orderType":1,"orderStatus":4,"stockMode":3,"printOrderStatus":1,"optionalLogisticsList":[2],"orderLogisticsType":2,"orderGoodsInfoList":[{"goodsId":101,"skuCode":"first","newGoodsStatus":4,"goodsExchangeTag":1},{"goodsId":102,"skuCode":"second","newGoodsStatus":2,"goodsExchangeTag":1,"timeOutList":[{"timeOutType":2,"timeOutTime":"` + deadline + `"}]}]}]}`
	c := testClient(t, roundTripFunc(func(*http.Request) (*http.Response, error) { return observationResponse(raw), nil }))
	source, err := c.QueryFulfillmentSource(context.Background(), goodsCredential(), "consumer_order", "original")
	if err != nil || len(source.Lines) != 2 || source.Deadline.IsZero() || source.Lines[0].Delivered != 0 {
		t.Fatal("platform state became local delivery or blocked remaining goods", source, err)
	}
}

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
