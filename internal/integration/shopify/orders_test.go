package shopify

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestShopifyOrderReadPaginatesWithoutExposingOtherData(t *testing.T) {
	start := time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)
	end := start.Add(30 * 24 * time.Hour)
	var requests, checks int
	read := func(changed bool) (OrderPage, error) {
		requests, checks = 0, 0
		reader, err := NewSyntheticOrderReader(transportFunc(func(r *http.Request) (*http.Response, error) {
			requests++
			var req struct {
				Query     string
				Variables map[string]any
			}
			require.NoError(t, json.NewDecoder(r.Body).Decode(&req))
			require.Equal(t, requests, checks)
			require.Equal(t, "orders-fixture", r.Header.Get("X-Shopify-Access-Token"))
			require.NotContains(t, req.Query, "customer")
			require.NotContains(t, req.Query, "totalPrice")
			if requests == 1 {
				require.Contains(t, req.Query, "CREATED_AT")
				require.Equal(t, "created_at:>='2026-10-01T00:00:00Z' created_at:<'2026-10-31T00:00:00Z'", req.Variables["window"])
				require.Nil(t, req.Variables["after"])
				return fixtureResponse(200, shopifyOrderFirstPageFixture), nil
			}
			require.Equal(t, "gid://shopify/Order/17", req.Variables["id"])
			require.Equal(t, "lines-next", req.Variables["after"])
			response := shopifyOrderSecondLinePageFixture
			if changed {
				response = strings.Replace(response, "2026-10-22", "2026-10-23", 1)
			}
			return fixtureResponse(200, response), nil
		}))
		require.NoError(t, err)
		return reader.Orders(context.Background(), "fixture.myshopify.com", "orders-fixture", OrderQuery{Start: start, End: end},
			func(context.Context) error { checks++; return nil })
	}
	page, err := read(false)
	require.NoError(t, err)
	require.Equal(t, "orders-next", page.Next)
	require.Equal(t, 2, requests)
	require.Len(t, page.Orders, 1)
	require.Len(t, page.Orders[0].Items, 2)
	require.Empty(t, page.Orders[0].Items[1].SKU)
	require.False(t, page.Incomplete)

	page, err = read(true)
	require.NoError(t, err)
	require.True(t, page.Incomplete)
	require.Empty(t, page.Orders)
}
func TestShopifyOrderGraphQLErrorDoesNotMeanEmptyOrders(t *testing.T) {
	reader, err := NewSyntheticOrderReader(transportFunc(func(*http.Request) (*http.Response, error) {
		return fixtureResponse(200, `{"data":{"orders":null},"errors":[{"message":"Access denied"}]}`), nil
	}))
	require.NoError(t, err)
	start := time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)
	_, err = reader.Orders(context.Background(), "fixture.myshopify.com", "orders-fixture",
		OrderQuery{Start: start, End: start.Add(30 * 24 * time.Hour)}, func(context.Context) error { return nil })
	require.Error(t, err)
}

func TestShopifyOrderRequiresAnExplicitCancellationField(t *testing.T) {
	withoutCancellation := strings.Replace(shopifyOrderFirstPageFixture, `"cancelledAt":null,`, "", 1)
	withoutCancellation = strings.Replace(withoutCancellation, `"hasNextPage":true,"endCursor":"lines-next"`, `"hasNextPage":false,"endCursor":null`, 1)
	reader, err := NewSyntheticOrderReader(transportFunc(func(*http.Request) (*http.Response, error) {
		return fixtureResponse(200, withoutCancellation), nil
	}))
	require.NoError(t, err)
	start := time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)
	page, err := reader.Orders(context.Background(), "fixture.myshopify.com", "orders-fixture",
		OrderQuery{Start: start, End: start.Add(30 * 24 * time.Hour)}, func(context.Context) error { return nil })
	require.NoError(t, err)
	require.True(t, page.Incomplete)
	require.Empty(t, page.Orders)
}

const shopifyOrderFirstPageFixture = `{"data":{"orders":{"pageInfo":{"hasNextPage":true,"endCursor":"orders-next"},"nodes":[{"id":"gid://shopify/Order/17","name":"#1017","createdAt":"2026-10-20T12:00:00Z","updatedAt":"2026-10-22T12:00:00Z","cancelledAt":null,"displayFulfillmentStatus":"UNFULFILLED","lineItems":{"pageInfo":{"hasNextPage":true,"endCursor":"lines-next"},"nodes":[{"id":"gid://shopify/LineItem/51","title":"Fixture A","sku":"A01","quantity":2}]}}]}}}`

const shopifyOrderSecondLinePageFixture = `{"data":{"order":{"id":"gid://shopify/Order/17","updatedAt":"2026-10-22T12:00:00Z","lineItems":{"pageInfo":{"hasNextPage":false,"endCursor":null},"nodes":[{"id":"gid://shopify/LineItem/52","title":"Fixture B","sku":null,"quantity":1}]}}}}`
