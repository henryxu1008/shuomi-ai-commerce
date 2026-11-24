package listinghttp

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	listingapp "github.com/henryxu1008/shuomi-ai-commerce/internal/app/listing"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/httproute"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/listing/preparation"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/listing/submission"
)

const GoogleListingPath = ListingBasePath + "/google"

func GoogleRoutes(service *listingapp.GooglePublication, bind func(context.Context, string) (context.Context, error)) []httproute.Descriptor {
	specs := []struct{ method, path, permission string }{
		{http.MethodGet, "/options/:source_id", preparation.PermissionRead},
		{http.MethodPost, "/prepare", preparation.PermissionManage},
		{http.MethodPost, "/confirm", preparation.PermissionSubmit},
		{http.MethodPost, "/result", preparation.PermissionRead},
	}
	var routes []httproute.Descriptor
	for _, spec := range specs {
		routes = append(routes, httproute.Descriptor{
			Method: spec.method, Path: GoogleListingPath + spec.path, Module: "supply-chain",
			Permission: spec.permission, AuthPolicy: httproute.AuthPolicyVerifiedIdentity,
			OrganizationAccessPolicy: httproute.OrganizationAccessPolicyLiveWrite, RequestTimeout: 60 * time.Second,
			Handler: httproute.WithRequestBodyReadTimeout(3*time.Second, func(c *gin.Context) {
				c.Header("Cache-Control", "private, no-store")
				if service == nil || bind == nil {
					listingError(c, preparation.ErrUnavailable)
					return
				}
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
					var input listingapp.GooglePrepareInput
					var key string
					key, err = listingBodyLimit(c.Request, &input, true, 32768)
					if err == nil {
						var plan submission.GooglePlan
						plan, err = service.Prepare(ctx, key, input)
						if err == nil {
							result = struct {
								submission.GooglePlan
								AccountID       string `json:"accountId"`
								DataSourceID    string `json:"dataSourceId"`
								ContentLanguage string `json:"contentLanguage"`
								FeedLabel       string `json:"feedLabel"`
							}{plan, plan.Target.AccountID, plan.Target.DataSourceID, plan.Target.ContentLanguage, plan.Target.FeedLabel}
						}
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
