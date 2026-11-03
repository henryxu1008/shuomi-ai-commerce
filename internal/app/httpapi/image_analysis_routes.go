package httpapi

import (
	"github.com/gin-gonic/gin"
	"net/http"
	"task-processor/internal/agent"
	"task-processor/internal/agentconfig"
	"task-processor/internal/authidentity"
	"task-processor/internal/authz"
	"task-processor/internal/core/config"
	"task-processor/internal/httproute"
	module "task-processor/internal/kernel/module"
	"time"
)

type imageAnalysisModule struct{ app *imageAnalysisApplication }

func (imageAnalysisModule) Name() string                { return "product-image-analysis" }
func (imageAnalysisModule) Enabled(*config.Config) bool { return true }
func (m imageAnalysisModule) Register(r *module.Registry) error {
	r.AddRoutes(imageAnalysisRoutes(m.app)...)
	return nil
}

func imageAnalysisRoutes(a *imageAnalysisApplication) []httproute.Descriptor {
	var out []httproute.Descriptor
	for _, owner := range []struct{ base, param, kind string }{
		{productAcquisitionBase + "/:operation_id/images/analysis", "operation_id", "acquisition"},
		{"/api/v1/workbench/supply-preparations/sources/:source_id/images/analysis", "source_id", "collection"},
		{"/api/v1/workbench/collections/items/:item_id/images/analysis", "item_id", "own_product"},
	} {
		for _, action := range []struct{ method, suffix, name string }{{"GET", "", "sources"}, {"GET", "/runs", "recent"}, {"POST", "/prepare", "prepare"}, {"GET", "/by-key/:request_key", "read"}, {"POST", "/by-key/:request_key/confirm", "confirm"}} {
			owner, action := owner, action
			out = append(out, httproute.Descriptor{Method: action.method, Path: owner.base + action.suffix, Module: "product-image-analysis", Permission: authz.PermissionWorkbenchAgentUse, AuthPolicy: httproute.AuthPolicyVerifiedIdentity, OrganizationAccessPolicy: httproute.OrganizationAccessPolicyLiveWrite, RequestTimeout: 2 * time.Minute, Handler: func(c *gin.Context) {
				if a == nil {
					writeProductAgentError(c, agent.ErrUnavailable)
					return
				}
				id, key := c.Param(owner.param), c.Param("request_key")
				if !agentconfig.UUID(id) || action.name != "recent" && c.Request.URL.RawQuery != "" {
					writeProductAgentError(c, agent.ErrInvalid)
					return
				}
				if action.name == "prepare" {
					values := c.Request.Header.Values("Idempotency-Key")
					if len(values) != 1 {
						writeProductAgentError(c, agent.ErrInvalid)
						return
					}
					key = values[0]
				} else if c.Request.Header.Get("Idempotency-Key") != "" {
					writeProductAgentError(c, agent.ErrInvalid)
					return
				}
				if action.name != "sources" && action.name != "recent" && !agentconfig.UUID(key) {
					writeProductAgentError(c, agent.ErrInvalid)
					return
				}
				ctx, err := (productReviewCapabilityBinder{now: time.Now}).Bind(c.Request.Context(), c.GetHeader("Authorization"))
				if err != nil {
					writeProductAgentError(c, err)
					return
				}
				i, err := a.parent.freshWorkbenchIdentity(ctx, authz.PermissionWorkbenchAgentUse)
				if action.method == "POST" {
					i, err = a.parent.freshIdentity(ctx)
				}
				if err != nil {
					writeProductAgentError(c, err)
					return
				}
				ctx = authidentity.WithAuthenticatedIdentity(ctx, i)
				c.Header("Cache-Control", "no-store")
				if action.method == "GET" && readProductReviewGETBody(c) != nil {
					writeProductAgentError(c, agent.ErrInvalid)
					return
				}
				if action.name == "sources" {
					v, e := a.describe(ctx, owner.kind, id)
					if e != nil {
						writeProductAgentError(c, e)
						return
					}
					c.JSON(http.StatusOK, v)
					return
				}
				if action.name == "recent" {
					query := c.Request.URL.Query()
					cursor := query.Get("cursor")
					if len(query) > 1 || len(query) == 1 && (len(query["cursor"]) != 1 || !agentconfig.UUID(cursor)) {
						writeProductAgentError(c, agent.ErrInvalid)
						return
					}
					if _, e := a.currentBinding(ctx, owner.kind, id); e != nil {
						writeProductAgentError(c, e)
						return
					}
					items, next, e := a.parent.configuration.RecentImageAnalysis(ctx, agent.Scope{OrganizationID: i.TenantID, ActorID: i.UserID}, i.EffectiveMemberID, owner.kind, id, cursor, 20, a.parent.store, func(b agent.Binding) error { _, e := a.Authorize(ctx, b); return e })
					if e != nil {
						writeProductAgentError(c, e)
						return
					}
					c.JSON(http.StatusOK, gin.H{"contextKind": owner.kind, "contextId": id, "items": items, "nextCursor": next})
					return
				}
				var snapshot agentconfig.Snapshot
				if action.name == "prepare" {
					var body imageAnalysisPrepareBody
					if readAcquisitionImageJSON(c.Request, &body) != nil {
						writeProductAgentError(c, agent.ErrInvalid)
						return
					}
					snapshot, err = a.prepare(ctx, owner.kind, id, key, body)
				} else {
					var found bool
					snapshot, found, err = a.preparation(ctx, key)
					if err == nil && !found {
						err = agentconfig.ErrNotFound
					}
				}
				if err != nil {
					writeProductAgentError(c, err)
					return
				}
				if snapshot.Request.Binding.ContextKind != owner.kind || snapshot.Request.Binding.ContextID != id {
					writeProductAgentError(c, agentconfig.ErrConflict)
					return
				}
				if action.name == "confirm" {
					var consent imageAnalysisConfirmBody
					if readAcquisitionImageJSON(c.Request, &consent) != nil || !consent.matches(snapshot) {
						writeProductAgentError(c, agent.ErrInvalid)
						return
					}
					request := snapshot.Request
					request.ConfigurationSnapshotRef = agent.ConfigurationSnapshotRef{Kind: agentconfig.SnapshotKind, ID: snapshot.ID, Digest: snapshot.Digest}
					runtime, e := a.runtime(snapshot.ImageAnalysis.Kind == "approved")
					if e != nil {
						writeProductAgentError(c, e)
						return
					}
					_, attempted, e := runtime.StartWithClaimAttempt(ctx, request)
					if e != nil {
						if attempted {
							c.JSON(http.StatusServiceUnavailable, gin.H{"code": "OUTCOME_UNKNOWN"})
							return
						}
						writeProductAgentRunError(c, e, attempted)
						return
					}
				}
				v, e := a.result(ctx, snapshot)
				if e != nil {
					if action.name == "confirm" {
						// This can follow a completed send/settlement, including an
						// original-key replay. A read failure proves no rejection.
						c.JSON(http.StatusServiceUnavailable, gin.H{"code": "OUTCOME_UNKNOWN"})
						return
					}
					writeProductAgentError(c, e)
					return
				}
				c.JSON(http.StatusOK, v)
			}})
		}
	}
	return out
}
