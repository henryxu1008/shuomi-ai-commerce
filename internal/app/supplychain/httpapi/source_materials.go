package supplychainhttp

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	supplyapp "task-processor/internal/app/supplychain"
	"task-processor/internal/httproute"
	"task-processor/internal/listing/preparation"
)

func IsSourceMaterialRoute(method, path string) bool {
	return method == http.MethodPost && (path == SupplyBasePath+"/images/approve" || path == SupplyBasePath+"/images/inventory") ||
		method == http.MethodGet && path == SupplyBasePath+"/sources/:source_id/product-inputs"
}

// These routes add no target, upload, Store or operation capability.
func SourceMaterialRoutes(app *supplyapp.Application, bind func(context.Context, string) (context.Context, error)) []httproute.Descriptor {
	var out []httproute.Descriptor
	for _, route := range SupplyRoutes(app, bind) {
		if IsSourceMaterialRoute(route.Method, route.Path) {
			out = append(out, route)
		}
	}
	return append(out, httproute.Descriptor{Method: http.MethodGet, Path: SupplyBasePath + "/sources/:source_id/product-inputs", Module: "supply-chain", Permission: preparation.PermissionManage, AuthPolicy: httproute.AuthPolicyVerifiedIdentity, OrganizationAccessPolicy: httproute.OrganizationAccessPolicyLiveWrite, RequestTimeout: 20 * time.Second, Handler: httproute.WithRequestBodyReadTimeout(3*time.Second, func(c *gin.Context) {
		c.Header("Cache-Control", "no-store")
		if app == nil || bind == nil {
			supplyError(c, preparation.ErrUnavailable)
			return
		}
		if c.Request.URL.RawQuery != "" || supplyEmptyBody(c.Request) != nil {
			supplyError(c, preparation.ErrInvalid)
			return
		}
		ctx, err := bind(c.Request.Context(), c.GetHeader("Authorization"))
		if err != nil {
			supplyError(c, preparation.ErrForbidden)
			return
		}
		raw, err := app.ProductInputs(ctx, c.Param("source_id"))
		if err != nil {
			supplyError(c, err)
			return
		}
		c.JSON(http.StatusOK, struct {
			SourceID string          `json:"sourceId"`
			Inputs   json.RawMessage `json:"inputs"`
		}{SourceID: c.Param("source_id"), Inputs: raw})
	})})
}
