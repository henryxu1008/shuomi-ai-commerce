package temu

import (
	"context"
	"encoding/json"
	"errors"
	"strconv"
	"strings"
	"time"

	"task-processor/internal/storecenter"
	o "task-processor/internal/storecenter/observations"
)

const consumerOrderListAPI = "bg.order.list.v2.get"
const consumerOrderAmountAPI = "bg.order.amount.query"

// QueryObservedTemuOrders is a named read-only capability of the regional r2
// application. It does not accept arbitrary API types or mutation payloads.
func (c *OfficialClient) QueryObservedTemuOrders(ctx context.Context, credential storecenter.OfficialMerchantCredential, query o.TemuOrderQuery) (o.TemuOrderPage, error) {
	var zero o.TemuOrderPage
	start, end, e := query.ClosedCreationRange()
	app := c.application
	if ctx == nil || e != nil || !c.validCredential(credential) || app.SupportedModes != storecenter.ModesLocal|storecenter.ModesSemiManaged || (app.IdentityPartition != storecenter.PartitionUS && app.IdentityPartition != storecenter.PartitionGlobal) || !strings.HasSuffix(app.Version, ":temu:"+string(app.IdentityPartition)+":manual_token:3:r2") {
		return zero, o.ErrInvalid
	}
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	information, e := c.QueryStore(ctx, credential)
	if e != nil || information.Verification == nil {
		return zero, o.ErrUnavailable
	}
	facts := information.Verification
	if facts.RegionID != 211 || (facts.Mode != storecenter.MerchantLocal && facts.Mode != storecenter.MerchantSemiManaged) || !hasScope(facts.Scopes, consumerOrderListAPI) {
		return zero, o.ErrForbidden
	}
	origin := localAPIOrigin(app.IdentityPartition)
	fields := map[string]json.RawMessage{"pageNumber": json.RawMessage(strconv.Itoa(query.Page)), "pageSize": json.RawMessage("100"), "createAfter": json.RawMessage(strconv.FormatInt(start, 10)), "createBefore": json.RawMessage(strconv.FormatInt(end, 10))}
	var list json.RawMessage
	if e = c.readOfficialResponse(ctx, origin, consumerOrderListAPI, credential.Temu.AccessToken, fields, &list); e != nil {
		return zero, o.ErrUnavailable
	}
	wrap := func(result json.RawMessage) []byte {
		raw, _ := json.Marshal(struct {
			Success bool            `json:"success"`
			Result  json.RawMessage `json:"result"`
		}{true, result})
		return raw
	}
	raw := wrap(list)
	defer clear(raw)
	page, e := DecodeConsumerOrderPage(raw, nil)
	if e != nil {
		return zero, e
	}
	if hasScope(facts.Scopes, consumerOrderAmountAPI) {
		for i := range page.Items {
			id := page.Items[i].ID
			encoded, _ := json.Marshal(id)
			var result json.RawMessage
			e = c.readOfficialResponse(ctx, origin, consumerOrderAmountAPI, credential.Temu.AccessToken, map[string]json.RawMessage{"parentOrderSn": encoded}, &result)
			if e != nil {
				if ctx.Err() == nil && errors.Is(e, errOfficialReadRejected) {
					continue
				}
				return zero, o.ErrUnavailable
			}
			amountRaw := wrap(result)
			amount, err := decodeConsumerAmount(amountRaw, id)
			clear(amountRaw)
			clear(result)
			if err != nil {
				return zero, err
			}
			page.Items[i].Amount = amount
		}
	}
	if ctx.Err() != nil {
		return zero, o.ErrUnavailable
	}
	return page, nil
}
