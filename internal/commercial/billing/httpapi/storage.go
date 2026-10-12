package httpapi

import (
	"encoding/json"
	"github.com/gin-gonic/gin"
	"net/http"
	"strconv"
	"task-processor/internal/authz"
	"task-processor/internal/commercial/billing"
	"task-processor/internal/httproute"
	"time"
)

const StorageSummaryPath = "/api/v1/workbench/storage/summary"
const StorageOffersPath = "/api/v1/workbench/commercial/storage-offers"
const PublicStorageOffersPath = "/api/v1/commercial/storage-offers"

func ValidateStorageDescriptor(r httproute.Descriptor) bool {
	if r.Path == PublicStorageOffersPath {
		return ValidatePublicPriceDescriptor(r)
	}
	return (r.Path == StorageSummaryPath || r.Path == StorageOffersPath) && r.Method == http.MethodGet && r.Module == "commercial-billing" && r.Permission == authz.PermissionWorkbenchCommercialRead && r.AuthPolicy == httproute.AuthPolicyCurrentIdentity && r.OrganizationAccessPolicy == httproute.OrganizationAccessPolicyLiveWrite && r.RequestTimeout == 15*time.Second && r.RejectUnreadRequestBody && r.Handler != nil
}

func storageRoutes(h *Handler) []httproute.Descriptor {
	return []httproute.Descriptor{
		{Method: http.MethodGet, Path: StorageSummaryPath, Module: "commercial-billing", Permission: authz.PermissionWorkbenchCommercialRead, AuthPolicy: httproute.AuthPolicyCurrentIdentity, OrganizationAccessPolicy: httproute.OrganizationAccessPolicyLiveWrite, RequestTimeout: 15 * time.Second, RejectUnreadRequestBody: true, Handler: h.StorageSummary},
		{Method: http.MethodGet, Path: StorageOffersPath, Module: "commercial-billing", Permission: authz.PermissionWorkbenchCommercialRead, AuthPolicy: httproute.AuthPolicyCurrentIdentity, OrganizationAccessPolicy: httproute.OrganizationAccessPolicyLiveWrite, RequestTimeout: 15 * time.Second, RejectUnreadRequestBody: true, Handler: h.StorageOffers},
		{Method: http.MethodGet, Path: PublicStorageOffersPath, Module: "commercial-billing", AuthPolicy: httproute.AuthPolicyPublic, OrganizationAccessPolicy: httproute.OrganizationAccessPolicyNone, RequestTimeout: 5 * time.Second, RejectUnreadRequestBody: true, Handler: h.PublicStorageOffers},
	}
}
func (h *Handler) StorageSummary(c *gin.Context) {
	if !validReadRequest(c) || c.Request.URL.RawQuery != "" || c.Request.URL.ForceQuery {
		writeError(c, 400, "INVALID_REQUEST")
		return
	}
	org, ok := effectiveOrganization(c)
	if !ok {
		writeError(c, 409, "ORGANIZATION_SELECTION_REQUIRED")
		return
	}
	if h.storage == nil {
		writeServiceError(c, billing.ErrFeatureUnavailable)
		return
	}
	s, e := h.storage.ReadSummary(c.Request.Context(), org)
	if e != nil {
		writeServiceError(c, billing.ErrFeatureUnavailable)
		return
	}
	boundedStorageJSON(c, map[string]any{"organization_id": org, "summary": s})
}
func (h *Handler) StorageOffers(c *gin.Context) {
	if !validReadRequest(c) || c.Request.URL.RawQuery != "" || c.Request.URL.ForceQuery {
		writeError(c, 400, "INVALID_REQUEST")
		return
	}
	org, ok := effectiveOrganization(c)
	if !ok {
		writeError(c, 409, "ORGANIZATION_SELECTION_REQUIRED")
		return
	}
	h.writeStorageOffers(c, org)
}
func (h *Handler) PublicStorageOffers(c *gin.Context) {
	c.Header("Cache-Control", "no-store")
	if c.Request.ContentLength != 0 || len(c.Request.TransferEncoding) != 0 {
		httproute.RejectUnreadRequestBody(c)
		writeError(c, 400, "INVALID_REQUEST")
		return
	}
	if c.Request.Method != http.MethodGet || c.Request.URL.RawQuery != "" || c.Request.URL.ForceQuery {
		writeError(c, 400, "INVALID_REQUEST")
		return
	}
	h.writeStorageOffers(c, "")
}
func (h *Handler) writeStorageOffers(c *gin.Context, org string) {
	offers, e := h.service.ListStorageOffers(c.Request.Context())
	if e != nil {
		writeServiceError(c, e)
		return
	}
	items := []map[string]string{}
	for _, o := range offers {
		items = append(items, map[string]string{"offer_id": o.OfferID, "product_kind": string(o.ProductKind), "currency": o.Currency, "unit_price_minor": strconv.FormatInt(o.UnitPriceMinor, 10), "min_quantity": strconv.FormatInt(o.MinQuantity, 10), "max_quantity": strconv.FormatInt(o.MaxQuantity, 10), "storage_unit_bytes": strconv.FormatInt(o.StorageUnitBytes, 10), "storage_duration_seconds": strconv.FormatInt(o.StorageDurationSeconds, 10), "pricing_version": o.PricingVersion})
	}
	payload := map[string]any{"schema_version": "storage-price-catalog-v1", "items": items}
	if org != "" {
		payload["organization_id"] = org
	}
	boundedStorageJSON(c, payload)
}
func boundedStorageJSON(c *gin.Context, v any) {
	body, e := json.Marshal(v)
	if e != nil || len(body) > 32*1024 {
		writeServiceError(c, billing.ErrFeatureUnavailable)
		return
	}
	c.Header("Cache-Control", "private, no-store")
	c.Header("X-Content-Type-Options", "nosniff")
	c.Data(200, "application/json; charset=utf-8", body)
}
