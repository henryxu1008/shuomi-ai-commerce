package temu

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"task-processor/internal/storecenter"
	o "task-processor/internal/storecenter/observations"
)

func TestTemuOrderReaderUsesRegionalScopesAndOriginalClosedWindow(t *testing.T) {
	for _, mode := range []int{100, 1} {
		for _, state := range []string{"known", "no_scope", "rejected", "warning", "wrong_parent", "oversized_amount", "invalid_amount", "canceled_amount"} {
			t.Run(strconv.Itoa(mode)+state, func(t *testing.T) {
				ctx, cancel := context.WithCancel(context.Background())
				defer cancel()
				var calls []string
				c := identityClient(t, storecenter.PartitionUS, roundTripFunc(func(r *http.Request) (*http.Response, error) {
					require.Equal(t, "openapi-b-us.temu.com", r.URL.Host)
					require.Equal(t, "/openapi/router", r.URL.Path)
					require.Equal(t, http.MethodPost, r.Method)
					raw, e := io.ReadAll(r.Body)
					require.NoError(t, e)
					var fields map[string]json.RawMessage
					require.NoError(t, json.Unmarshal(raw, &fields))
					var api, sign string
					require.NoError(t, json.Unmarshal(fields["type"], &api))
					require.NoError(t, json.Unmarshal(fields["sign"], &sign))
					delete(fields, "sign")
					calls = append(calls, api)
					keys := []string{}
					for k := range fields {
						keys = append(keys, k)
					}
					sort.Strings(keys)
					signed := "test-secret-0123456789"
					for _, k := range keys {
						var s string
						if json.Unmarshal(fields[k], &s) != nil {
							s = string(fields[k])
						}
						signed += k + s
					}
					signed += "test-secret-0123456789"
					hash := md5.Sum([]byte(signed))
					require.Equal(t, strings.ToUpper(hex.EncodeToString(hash[:])), sign)
					switch api {
					case localInfoAPI:
						scopes := []string{"bg.order.list.v2.get"}
						if state != "no_scope" {
							scopes = append(scopes, "bg.order.amount.query")
						}
						raw, _ := json.Marshal(map[string]any{"success": true, "result": map[string]any{"mallId": 123, "mallType": mode, "semiUniqueId": "semi-123", "regionId": 211, "expiredTime": 1760003600, "apiScopeList": scopes}})
						return jsonResponse(string(raw)), nil
					case "bg.order.list.v2.get":
						require.Equal(t, "2", string(fields["pageNumber"]))
						require.Equal(t, "100", string(fields["pageSize"]))
						require.Equal(t, "1759999800", string(fields["createAfter"]))
						require.Equal(t, "1759999999", string(fields["createBefore"]))
						require.Len(t, fields, 9)
						return jsonResponse(`{"success":true,"result":{"totalItemNum":1,"pageItems":[{"parentOrderMap":{"parentOrderSn":"PO-A","parentOrderTime":1759999999,"regionId":211},"orderList":[{"orderSn":"C1","quantity":9},{"orderSn":"C2","quantity":4}],"sites":[1,2],"buyerEmail":"PRIVATE"}]}}`), nil
					case "bg.order.amount.query":
						require.Equal(t, `"PO-A"`, string(fields["parentOrderSn"]))
						require.Len(t, fields, 6)
						if state == "oversized_amount" {
							return jsonResponse(strings.Repeat(" ", 2<<20) + `{}`), nil
						}
						if state == "invalid_amount" {
							return jsonResponse(`{"success":true,"result":{"parentOrderMap":{"parentOrderSn":"PO-A","retailPriceTotal":{"amount":-1,"currency":"USD"}}}}`), nil
						}
						if state == "canceled_amount" {
							cancel()
						}
						if state == "rejected" {
							return jsonResponse(`{"success":false}`), nil
						}
						id := "PO-A"
						if state == "wrong_parent" {
							id = "OTHER"
						}
						warning := ""
						if state == "warning" {
							warning = `,"warning":["calculating"]`
						}
						return jsonResponse(`{"success":true,"result":{"parentOrderMap":{"parentOrderSn":"` + id + `","retailPriceTotal":{"amount":2525,"currency":"USD"}}` + warning + `}}`), nil
					default:
						t.Fatalf("unexpected API %s", api)
						return nil, nil
					}
				}))
				c.application.Version = "test:temu:US:manual_token:3:r2"
				c.application.SupportedModes = storecenter.ModesLocal | storecenter.ModesSemiManaged
				page, e := c.QueryObservedTemuOrders(ctx, tokenCredential(), o.TemuOrderQuery{Start: time.Unix(1759999800, 0), End: time.Unix(1760000000, 0), Page: 2})
				if state == "wrong_parent" || state == "oversized_amount" || state == "invalid_amount" || state == "canceled_amount" {
					require.Error(t, e)
					return
				}
				require.NoError(t, e)
				require.Len(t, page.Items, 1)
				require.Equal(t, "PO-A", page.Items[0].ID)
				if state == "known" {
					require.Equal(t, "25.25", page.Items[0].Amount.Value)
				} else {
					require.Nil(t, page.Items[0].Amount)
				}
				expected := 2
				if state != "no_scope" {
					expected = 3
				}
				require.Len(t, calls, expected)
				encoded, _ := json.Marshal(page)
				require.NotContains(t, string(encoded), "PRIVATE")
				require.NotContains(t, string(encoded), "C1")
			})
		}
	}
}

func TestTemuOrderReaderRejectsProductOnlyScopeAndOversizedList(t *testing.T) {
	for _, test := range []string{"product_scope", "oversize", "cancel"} {
		t.Run(test, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			calls := 0
			c := identityClient(t, storecenter.PartitionUS, roundTripFunc(func(r *http.Request) (*http.Response, error) {
				calls++
				if calls == 1 {
					scope := "bg.order.list.v2.get"
					if test == "product_scope" {
						scope = localGoodsListAPI
					}
					return jsonResponse(`{"success":true,"result":{"mallId":123,"mallType":100,"regionId":211,"expiredTime":1760003600,"apiScopeList":["` + scope + `"]}}`), nil
				}
				if test == "cancel" {
					cancel()
					return jsonResponse(`{"success":true,"result":{"totalItemNum":0,"pageItems":[]}}`), nil
				}
				return jsonResponse(strings.Repeat(" ", 2<<20) + `{}`), nil
			}))
			c.application.Version = "test:temu:US:manual_token:3:r2"
			c.application.SupportedModes = storecenter.ModesLocal | storecenter.ModesSemiManaged
			_, e := c.QueryObservedTemuOrders(ctx, tokenCredential(), o.TemuOrderQuery{Start: time.Unix(1759999800, 0), End: time.Unix(1760000000, 0), Page: 1})
			require.Error(t, e)
			if test == "product_scope" {
				require.Equal(t, 1, calls)
			}
		})
	}
}
