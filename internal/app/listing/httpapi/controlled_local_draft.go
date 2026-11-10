package listinghttp

import (
	"context"
	"github.com/gin-gonic/gin"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	listingapp "task-processor/internal/app/listing"
	"task-processor/internal/httproute"
	"task-processor/internal/listing/preparation"
	record "task-processor/internal/listing/record/target"
	"time"
)

const ControlledLocalDraftBasePath = "/api/v1/workbench/local-platform-drafts"
const ControlledLocalDraftModule = "controlled-local-drafts"

func IsControlledLocalDraftRoute(method, path string, enabled bool) bool {
	if method == http.MethodGet {
		return path == ControlledLocalDraftBasePath+"/context" || path == ControlledLocalDraftBasePath+"/sources/:source_id/targets/:store_id" || path == ControlledLocalDraftBasePath+"/records/:record_id" || path == ControlledLocalDraftBasePath+"/target-commands/:key"
	}
	if !enabled || method != http.MethodPost {
		return false
	}
	return path == ControlledLocalDraftBasePath+"/target-rules" || path == ControlledLocalDraftBasePath+"/shein-completion-preview" || path == ControlledLocalDraftBasePath+"/shein-sales-mapping-preview" || path == ControlledLocalDraftBasePath+"/targets"
}
func ControlledLocalDraftRoutes(app *listingapp.Application, bind func(context.Context, string) (context.Context, error), enabled bool) []httproute.Descriptor {
	var out []httproute.Descriptor
	for _, r := range ListingRoutes(app, bind) {
		r.Path = strings.Replace(r.Path, ListingBasePath, ControlledLocalDraftBasePath, 1)
		r.Module = ControlledLocalDraftModule
		if IsControlledLocalDraftRoute(r.Method, r.Path, enabled) {
			out = append(out, r)
		}
	}
	out = append(out, httproute.Descriptor{Method: http.MethodGet, Path: ControlledLocalDraftBasePath + "/context", Module: ControlledLocalDraftModule, Permission: preparation.PermissionRead, AuthPolicy: httproute.AuthPolicyVerifiedIdentity, OrganizationAccessPolicy: httproute.OrganizationAccessPolicyLiveWrite, RequestTimeout: 20 * time.Second, Handler: httproute.WithRequestBodyReadTimeout(3*time.Second, func(c *gin.Context) {
		c.Header("Cache-Control", "no-store")
		if app == nil || bind == nil || app.LocalDraftContext == nil {
			listingError(c, preparation.ErrUnavailable)
			return
		}
		if err := listingEmptyBody(c.Request); err != nil {
			listingError(c, err)
			return
		}
		q, e := url.ParseQuery(c.Request.URL.RawQuery)
		if e != nil || len(q) != 3 || len(q["sourceId"]) != 1 || len(q["completionRecordId"]) != 1 || len(q["revision"]) != 1 {
			listingError(c, preparation.ErrInvalid)
			return
		}
		revision, e := strconv.ParseInt(q.Get("revision"), 10, 64)
		if e != nil || revision < 1 {
			listingError(c, preparation.ErrInvalid)
			return
		}
		ctx, e := bind(c.Request.Context(), c.GetHeader("Authorization"))
		if e != nil {
			listingError(c, preparation.ErrForbidden)
			return
		}
		value, e := app.LocalDraftContext(ctx, q.Get("sourceId"), record.CompletionDraftSelection{RecordID: q.Get("completionRecordId"), Revision: revision})
		if e != nil {
			listingError(c, e)
			return
		}
		c.JSON(http.StatusOK, value)
	})})
	return out
}
