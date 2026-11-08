package httpapi

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"task-processor/internal/agent"
	"task-processor/internal/agentconfig"
	"task-processor/internal/authz"
	"task-processor/internal/core/config"
	"task-processor/internal/httproute"
	module "task-processor/internal/kernel/module"
)

const ownIntakeBase = "/api/v1/workbench/own-product-intake"
const ownIntakeModuleName = "own-product-intake"

type ownIntakeModule struct{ app *ownIntakeApplication }

func (ownIntakeModule) Name() string                { return ownIntakeModuleName }
func (ownIntakeModule) Enabled(*config.Config) bool { return true }
func (m ownIntakeModule) Register(r *module.Registry) error {
	r.AddRoutes(ownIntakeRoutes(m.app)...)
	return nil
}

func ownIntakeRoutes(a *ownIntakeApplication) []httproute.Descriptor {
	var routes []httproute.Descriptor
	for _, spec := range []struct{ method, path string }{
		{http.MethodPost, "/prepare"}, {http.MethodPost, "/:request_key/confirm"}, {http.MethodGet, "/:request_key"},
	} {
		routes = append(routes, httproute.Descriptor{
			Method: spec.method, Path: ownIntakeBase + spec.path, Module: ownIntakeModuleName,
			Permission: authz.PermissionWorkbenchAgentUse, AuthPolicy: httproute.AuthPolicyVerifiedIdentity,
			OrganizationAccessPolicy: httproute.OrganizationAccessPolicyLiveWrite, RequestTimeout: 2 * time.Minute,
			Handler: httproute.WithRequestBodyReadTimeout(3*time.Second, func(c *gin.Context) {
				c.Header("Cache-Control", "no-store")
				if a == nil {
					writeProductAgentError(c, agent.ErrUnavailable)
					return
				}
				key := c.Param("request_key")
				keys := c.Request.Header.Values("Idempotency-Key")
				if spec.path == "/prepare" && len(keys) == 1 {
					key = keys[0]
				}
				if !agentconfig.UUID(key) || c.Request.URL.RawQuery != "" ||
					spec.method == http.MethodPost && (len(keys) != 1 || keys[0] != key) ||
					spec.method == http.MethodGet && (len(keys) != 0 || readProductReviewGETBody(c) != nil) {
					writeProductAgentError(c, agent.ErrInvalid)
					return
				}
				ctx, err := (productReviewCapabilityBinder{now: time.Now}).Bind(c.Request.Context(), c.GetHeader("Authorization"))
				if err != nil {
					writeProductAgentError(c, err)
					return
				}
				var result any
				switch {
				case spec.path == "/prepare":
					var body ownIntakeBody
					if readAcquisitionImageJSON(c.Request, &body) != nil {
						err = agent.ErrInvalid
						break
					}
					result, err = a.prepare(ctx, key, body)
				case spec.method == http.MethodGet:
					result, err = a.read(ctx, key)
				default:
					var body ownIntakeConfirm
					if readAcquisitionImageJSON(c.Request, &body) != nil {
						err = agent.ErrInvalid
						break
					}
					result, err = a.confirm(ctx, key, body)
				}
				if err != nil {
					writeProductAgentError(c, err)
					return
				}
				c.JSON(http.StatusOK, result)
			}),
		})
	}
	return routes
}
