package listinghttp

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	listingapp "github.com/henryxu1008/shuomi-ai-commerce/internal/app/listing"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/httproute"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/listing/preparation"
)

const MetaListingPath = ListingBasePath + "/meta-catalog"

func MetaCatalogSimulationRoutes(service *listingapp.MetaPublication, bind func(context.Context, string) (context.Context, error)) []httproute.Descriptor {
	if service == nil || bind == nil {
		return nil
	}
	specs := []struct{ method, path, permission string }{
		{http.MethodGet, "/options/:source_id", preparation.PermissionRead},
		{http.MethodPost, "/prepare", preparation.PermissionManage},
		{http.MethodPost, "/confirm", preparation.PermissionSubmit},
		{http.MethodPost, "/result", preparation.PermissionRead},
	}
	var routes []httproute.Descriptor
	for _, spec := range specs {
		routes = append(routes, httproute.Descriptor{
			Method: spec.method, Path: MetaListingPath + spec.path, Module: "supply-chain", Permission: spec.permission,
			AuthPolicy: httproute.AuthPolicyVerifiedIdentity, OrganizationAccessPolicy: httproute.OrganizationAccessPolicyLiveWrite,
			RequestTimeout: 45 * time.Second,
			Handler: httproute.WithRequestBodyReadTimeout(3*time.Second, func(c *gin.Context) {
				c.Header("Cache-Control", "private, no-store")
				ctx, err := bind(c.Request.Context(), c.GetHeader("Authorization"))
				if err != nil {
					listingError(c, preparation.ErrForbidden)
					return
				}
				var result any
				switch spec.path {
				case "/options/:source_id":
					if err = listingEmptyBody(c.Request); err == nil {
						result, err = service.Options(ctx, c.Param("source_id"))
					}
				case "/prepare":
					var input listingapp.MetaPrepareInput
					var key string
					key, err = listingBodyLimit(c.Request, &input, true, 16384)
					if err == nil {
						result, err = service.Prepare(ctx, key, input)
					}
				case "/confirm", "/result":
					var input struct {
						SourceID string `json:"sourceId"`
						PlanID   string `json:"planId"`
					}
					_, err = listingBodyLimit(c.Request, &input, false, 4096)
					if err == nil && spec.path == "/confirm" {
						result, err = service.Confirm(ctx, input.SourceID, input.PlanID)
					} else if err == nil {
						result, err = service.Result(ctx, input.SourceID, input.PlanID)
					}
				}
				if err != nil {
					listingError(c, err)
					return
				}
				c.JSON(http.StatusOK, result)
			}),
		})
	}
	return routes
}
