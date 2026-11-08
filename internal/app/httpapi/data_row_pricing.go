package httpapi

import (
	"errors"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"strconv"
	"strings"
	"task-processor/internal/authz"
	"task-processor/internal/commercial/billing"
	"task-processor/internal/httproute"
	"time"
)

const dataRowPriceBase = "/api/v1/workbench/commercial/data-row-pricing"

type dataRowPricingModule struct {
	store      billing.DataRowPriceStore
	authorizer *authz.ListingKitAuthorizer
}

func (m dataRowPricingModule) routes() []httproute.Descriptor {
	out := []httproute.Descriptor{}
	for _, s := range []struct{ method, path, op string }{{"GET", "", "current"}, {"GET", "/history", "history"}, {"GET", "/receipts/:key", "receipt"}, {"POST", "/publish", "publish"}} {
		handler := gin.HandlerFunc(func(c *gin.Context) { m.serve(c, s.op) })
		if s.method == "POST" {
			handler = httproute.WithRequestBodyReadTimeout(10*time.Second, handler)
		}
		out = append(out, httproute.Descriptor{Method: s.method, Path: dataRowPriceBase + s.path, Module: "commercial-billing", Permission: authz.PermissionListingKitPlatformAdm, AuthPolicy: httproute.AuthPolicyCurrentIdentity, OrganizationAccessPolicy: httproute.OrganizationAccessPolicyLiveWrite, RequestTimeout: 15 * time.Second, RejectUnreadRequestBody: s.method == "GET", Handler: handler})
	}
	return out
}
func validateDataRowPricingDescriptor(d httproute.Descriptor) bool {
	for _, r := range (dataRowPricingModule{}).routes() {
		if d.Method == r.Method && d.Path == r.Path {
			return d.Module == r.Module && d.Permission == r.Permission && d.AuthPolicy == r.AuthPolicy && d.OrganizationAccessPolicy == r.OrganizationAccessPolicy && d.RequestTimeout == r.RequestTimeout && d.RejectUnreadRequestBody == r.RejectUnreadRequestBody && d.OrganizationTargetResolver == nil && d.Handler != nil
		}
	}
	return false
}
func dataRowOfferView(o billing.Offer) gin.H {
	if o.UnitPriceMinor > 0 {
		o.SettlementMode = billing.SettlementWallet
	}
	return gin.H{"offer_id": o.OfferID, "product_kind": o.ProductKind, "resource_type": o.ResourceType, "currency": o.Currency, "unit_price_minor": strconv.FormatInt(o.UnitPriceMinor, 10), "pricing_version": o.PricingVersion, "min_quantity": strconv.FormatInt(o.MinQuantity, 10), "max_quantity": strconv.FormatInt(o.MaxQuantity, 10), "settlement_mode": o.SettlementMode}
}
func dataRowReceiptView(r billing.DataRowPriceReceipt) gin.H {
	return gin.H{"id": r.ID, "actor_id": r.ActorID, "organization_id": r.OrganizationID, "previous_offer_id": r.PreviousOfferID, "previous_version": r.PreviousVersion, "offer": dataRowOfferView(r.Offer), "reason": r.Reason, "created_at": r.CreatedAt}
}
func priceHTTPFailure(c *gin.Context, status int, code string) {
	c.AbortWithStatusJSON(status, gin.H{"code": code, "message": "Data row price request could not be completed", "requestId": uuid.NewString(), "fieldErrors": []any{}})
}
func priceHTTPError(c *gin.Context, e error) {
	status, code := 503, "DEPENDENCY_UNAVAILABLE"
	switch {
	case errors.Is(e, billing.ErrInvalid):
		status, code = 400, "INVALID_REQUEST"
	case errors.Is(e, billing.ErrConflict):
		status, code = 409, "CONFLICT"
	case errors.Is(e, billing.ErrNotFound):
		status, code = 404, "NOT_FOUND"
	case errors.Is(e, billing.ErrOfferUnavailable):
		code = "OFFER_UNAVAILABLE"
	}
	priceHTTPFailure(c, status, code)
}
func (m dataRowPricingModule) serve(c *gin.Context, op string) {
	c.Header("Cache-Control", "private, no-store")
	c.Header("X-Content-Type-Options", "nosniff")
	actor, org, e := modelRouteIdentity(c.Request.Context(), m.authorizer, op == "publish")
	if e != nil {
		priceHTTPFailure(c, 403, "PERMISSION_DENIED")
		return
	}
	if m.store == nil {
		priceHTTPError(c, billing.ErrFeatureUnavailable)
		return
	}
	if op != "history" && (c.Request.URL.RawQuery != "" || c.Request.URL.ForceQuery) {
		priceHTTPError(c, billing.ErrInvalid)
		return
	}
	switch op {
	case "current":
		v, e := m.store.CurrentDataRowOffer(c.Request.Context())
		if e != nil {
			priceHTTPError(c, e)
			return
		}
		c.JSON(200, dataRowOfferView(v))
	case "publish":
		var input struct {
			ExpectedVersion string  `json:"expected_version"`
			UnitPriceMinor  *string `json:"unit_price_minor"`
			Reason          string  `json:"reason"`
		}
		if modelRouteDecode(c, &input) != nil || input.UnitPriceMinor == nil {
			priceHTTPError(c, billing.ErrInvalid)
			return
		}
		amount, e := strconv.ParseInt(*input.UnitPriceMinor, 10, 64)
		if e != nil || amount < 0 || strconv.FormatInt(amount, 10) != *input.UnitPriceMinor {
			priceHTTPError(c, billing.ErrInvalid)
			return
		}
		keys := c.Request.Header.Values("Idempotency-Key")
		if len(keys) != 1 || strings.TrimSpace(keys[0]) != keys[0] {
			priceHTTPError(c, billing.ErrInvalid)
			return
		}
		v, e := m.store.PublishDataRowPrice(c.Request.Context(), billing.DataRowPriceCommand{ActorID: actor, OrganizationID: org, ExpectedVersion: input.ExpectedVersion, UnitPriceMinor: amount, IdempotencyKey: keys[0], Reason: input.Reason})
		if e != nil {
			priceHTTPError(c, e)
			return
		}
		c.JSON(201, dataRowReceiptView(v))
	case "receipt":
		v, e := m.store.ReadDataRowPriceReceipt(c.Request.Context(), actor, c.Param("key"))
		if e != nil {
			priceHTTPError(c, e)
			return
		}
		c.JSON(200, dataRowReceiptView(v))
	case "history":
		values := c.Request.URL.Query()
		for key, v := range values {
			if (key != "cursor" && key != "limit") || len(v) != 1 {
				priceHTTPError(c, billing.ErrInvalid)
				return
			}
		}
		limit := 20
		if raw := values.Get("limit"); raw != "" {
			limit, e = strconv.Atoi(raw)
			if e != nil {
				priceHTTPError(c, billing.ErrInvalid)
				return
			}
		}
		v, e := m.store.DataRowPriceHistory(c.Request.Context(), values.Get("cursor"), limit)
		if e != nil {
			priceHTTPError(c, e)
			return
		}
		items := []gin.H{}
		for _, r := range v.Items {
			items = append(items, dataRowReceiptView(r))
		}
		c.JSON(200, gin.H{"items": items, "next_cursor": v.NextCursor})
	}
}
