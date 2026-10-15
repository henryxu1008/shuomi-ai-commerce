package supplychainhttp

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/gin-gonic/gin"
	"net/http"
	"task-processor/internal/httproute"
	"task-processor/internal/listing/completiondraft"
	"task-processor/internal/listing/preparation"
	"task-processor/internal/product/collection"
	"time"
)

type CompletionDraftApplication interface {
	Save(context.Context, string, completiondraft.SaveInput) (completiondraft.Receipt, error)
	Read(context.Context, string) (completiondraft.Record, error)
	ReadCommand(context.Context, string) (completiondraft.Receipt, error)
	List(context.Context, string, collection.Query) (collection.Page[completiondraft.Record], error)
	ListForCollectionItem(context.Context, string, collection.Query) (completiondraft.ItemPage, error)
	ListHistory(context.Context, string, completiondraft.HistoryQuery) (completiondraft.HistoryPage, error)
	ReadRevision(context.Context, string, int64) (completiondraft.Record, error)
}

func CompletionDraftRoutes(app CompletionDraftApplication, bind func(context.Context, string) (context.Context, error)) []httproute.Descriptor {
	specs := []struct{ method, path, action, permission string }{
		{http.MethodPost, "/completion-drafts", "save", preparation.PermissionManage},
		{http.MethodGet, "/completion-drafts/commands/:key", "command", preparation.PermissionRead},
		{http.MethodGet, "/sources/:source_id/completion-draft", "read", preparation.PermissionRead},
		{http.MethodGet, "/sources/:source_id/completion-draft/history", "history", preparation.PermissionRead},
		{http.MethodGet, "/sources/:source_id/completion-draft/history/:revision", "history-detail", preparation.PermissionRead},
		{http.MethodGet, "/:preparation_id/completion-drafts", "list", preparation.PermissionRead},
		{http.MethodGet, "/collection-items/:item_id/completion-drafts", "item-list", preparation.PermissionRead},
	}
	routes := []httproute.Descriptor{}
	for _, spec := range specs {
		routes = append(routes, httproute.Descriptor{Method: spec.method, Path: SupplyBasePath + spec.path, Module: "supply-completion-drafts", Permission: spec.permission, AuthPolicy: httproute.AuthPolicyVerifiedIdentity, OrganizationAccessPolicy: httproute.OrganizationAccessPolicyLiveWrite, RequestTimeout: 20 * time.Second, Handler: httproute.WithRequestBodyReadTimeout(3*time.Second, func(c *gin.Context) {
			c.Header("Cache-Control", "no-store")
			if app == nil || bind == nil {
				supplyError(c, preparation.ErrUnavailable)
				return
			}
			ctx, err := bind(c.Request.Context(), c.GetHeader("Authorization"))
			if err != nil {
				supplyError(c, preparation.ErrForbidden)
				return
			}
			if spec.action != "list" && spec.action != "item-list" && spec.action != "history" && c.Request.URL.RawQuery != "" {
				supplyError(c, preparation.ErrInvalid)
				return
			}
			if spec.method == http.MethodGet {
				if (spec.action == "history" || spec.action == "history-detail") && len(c.Request.Header.Values("Content-Encoding")) > 0 {
					supplyError(c, preparation.ErrInvalid)
					return
				}
				if err := supplyEmptyBody(c.Request); err != nil {
					supplyError(c, err)
					return
				}
			}
			var output any
			switch spec.action {
			case "history":
				var q completiondraft.HistoryQuery
				q, err = completionHistoryQuery(c.Request)
				if err == nil {
					if !collection.ValidID(c.Param("source_id")) {
						err = preparation.ErrInvalid
					} else {
						output, err = app.ListHistory(ctx, c.Param("source_id"), q)
					}
				}
			case "history-detail":
				var revision int64
				revision, err = completiondraft.ParseRevision(c.Param("revision"))
				if err == nil {
					if !collection.ValidID(c.Param("source_id")) {
						err = preparation.ErrInvalid
					} else {
						output, err = app.ReadRevision(ctx, c.Param("source_id"), revision)
					}
				}
			case "save":
				var in completiondraft.SaveInput
				var key string
				key, err = supplyBodyLimit(c.Request, &in, true, completiondraft.MaxPayloadBytes)
				if err == nil {
					if err = in.Validate(); err == nil {
						output, err = app.Save(ctx, key, in)
					}
				}
			case "read":
				if !collection.ValidID(c.Param("source_id")) {
					err = preparation.ErrInvalid
				} else {
					output, err = app.Read(ctx, c.Param("source_id"))
				}
			case "command":
				if !collection.ValidID(c.Param("key")) {
					err = preparation.ErrInvalid
				} else {
					output, err = app.ReadCommand(ctx, c.Param("key"))
				}
			case "list":
				var q collection.Query
				q, err = supplyQuery(c.Request)
				if err == nil {
					if !collection.ValidID(c.Param("preparation_id")) {
						err = preparation.ErrInvalid
					} else {
						output, err = app.List(ctx, c.Param("preparation_id"), q)
					}
				}
			case "item-list":
				var q collection.Query
				q, err = supplyQuery(c.Request)
				if err == nil {
					if !c.Request.URL.Query().Has("limit") {
						q.Limit = 20
					}
					if !collection.ValidID(c.Param("item_id")) {
						err = preparation.ErrInvalid
					} else {
						output, err = app.ListForCollectionItem(ctx, c.Param("item_id"), q)
					}
				}
			}
			if err != nil {
				supplyError(c, err)
				return
			}
			raw, err := json.Marshal(output)
			if err != nil || len(raw) > supplyMaxBytes {
				supplyError(c, preparation.ErrUnavailable)
				return
			}
			c.Data(http.StatusOK, "application/json; charset=utf-8", raw)
		})})
	}
	return routes
}
func IsCompletionDraftRoute(method, path string) bool {
	for _, r := range CompletionDraftRoutes(nil, nil) {
		if r.Method == method && r.Path == path {
			return true
		}
	}
	return false
}
func ValidateCompletionDraftDescriptor(d httproute.Descriptor) error {
	for _, r := range CompletionDraftRoutes(nil, nil) {
		if r.Method == d.Method && r.Path == d.Path {
			if d.Module == r.Module && d.Permission == r.Permission && d.AuthPolicy == r.AuthPolicy && d.OrganizationAccessPolicy == r.OrganizationAccessPolicy && d.OrganizationTargetResolver == nil && d.RequestTimeout == r.RequestTimeout && d.Handler != nil {
				return nil
			}
			break
		}
	}
	return errors.New("completion draft route loses its admitted owner, permission or deadline")
}
