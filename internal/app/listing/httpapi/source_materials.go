package listinghttp

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	listingapp "task-processor/internal/app/listing"
	"task-processor/internal/httproute"
	"task-processor/internal/listing/preparation"
)

func IsSourceMaterialRoute(method, path string) bool {
	return method == http.MethodPost && (path == ListingBasePath+"/images/approve" || path == ListingBasePath+"/images/inventory") ||
		method == http.MethodGet && path == ListingBasePath+"/sources/:source_id/product-inputs"
}

// These routes add no target, upload, Store or operation capability.
func SourceMaterialRoutes(app *listingapp.Application, bind func(context.Context, string) (context.Context, error)) []httproute.Descriptor {
	var out []httproute.Descriptor
	for _, route := range ListingRoutes(app, bind) {
		if IsSourceMaterialRoute(route.Method, route.Path) {
			out = append(out, route)
		}
	}
	return append(out, httproute.Descriptor{Method: http.MethodGet, Path: ListingBasePath + "/sources/:source_id/product-inputs", Module: "supply-chain", Permission: preparation.PermissionManage, AuthPolicy: httproute.AuthPolicyVerifiedIdentity, OrganizationAccessPolicy: httproute.OrganizationAccessPolicyLiveWrite, RequestTimeout: 20 * time.Second, Handler: httproute.WithRequestBodyReadTimeout(3*time.Second, func(c *gin.Context) {
		c.Header("Cache-Control", "no-store")
		if app == nil || bind == nil {
			listingError(c, preparation.ErrUnavailable)
			return
		}
		if c.Request.URL.RawQuery != "" || listingEmptyBody(c.Request) != nil {
			listingError(c, preparation.ErrInvalid)
			return
		}
		ctx, err := bind(c.Request.Context(), c.GetHeader("Authorization"))
		if err != nil {
			listingError(c, preparation.ErrForbidden)
			return
		}
		raw, err := app.ProductInputs(ctx, c.Param("source_id"))
		if err != nil {
			listingError(c, err)
			return
		}
		c.JSON(http.StatusOK, struct {
			SourceID string          `json:"sourceId"`
			Inputs   json.RawMessage `json:"inputs"`
		}{SourceID: c.Param("source_id"), Inputs: raw})
	})})
}
