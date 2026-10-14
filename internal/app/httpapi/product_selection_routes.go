package httpapi

import (
	"context"
	"errors"
	"github.com/gin-gonic/gin"
	"net/http"
	"strconv"
	"strings"
	"task-processor/internal/agent"
	"task-processor/internal/agentconfig"
	"task-processor/internal/authz"
	"task-processor/internal/core/config"
	"task-processor/internal/httproute"
	kernelmodule "task-processor/internal/kernel/module"
	"task-processor/internal/product/collection"
	"task-processor/internal/product/review"
	"task-processor/internal/product/selection"
	"time"
)

const productSelectionBase = "/api/v1/workbench/product-selection"

func isContextAgentRunHTTPPath(path, base string) bool {
	prefix := base + "/contexts/"
	if !strings.HasPrefix(path, prefix) {
		return false
	}
	parts := strings.Split(strings.TrimPrefix(path, prefix), "/")
	return len(parts) >= 2 && agentconfig.UUID(parts[0]) && parts[1] == "runs" &&
		(len(parts) == 2 || len(parts) == 4 && agentconfig.UUID(parts[2]) && parts[3] == "resume")
}

type selectionHTTPService interface {
	inspectInput(context.Context, selectionInput) (agent.Binding, selection.Inspection, error)
	prepare(context.Context, selectionPrepareInput) (agentconfig.Snapshot, error)
	start(context.Context, string, string, agent.ConfigurationSnapshotRef) (agent.Record, bool, error)
	readRun(context.Context, string, string) (agent.Record, agentconfig.Snapshot, error)
	resume(context.Context, string, string, agent.ConfigurationSnapshotRef, uint64, string) (agent.Record, bool, error)
	projectRun(context.Context, agent.Record, agentconfig.Snapshot) (selectionRunDTO, error)
}
type productSelectionModule struct{ app *productSelectionApplication }

func (productSelectionModule) Name() string                { return "product-selection-agent" }
func (productSelectionModule) Enabled(*config.Config) bool { return true }
func (m productSelectionModule) Register(reg *kernelmodule.Registry) error {
	reg.AddRoutes(productSelectionRoutes(m.app, m.app.bindRequest)...)
	return nil
}
func (a *productSelectionApplication) bindRequest(ctx context.Context, bearer, action string) (context.Context, error) {
	ctx, e := (productReviewCapabilityBinder{now: time.Now}).Bind(ctx, bearer)
	if e != nil {
		return ctx, e
	}
	id, e := a.freshIdentity(ctx)
	if e != nil {
		return ctx, e
	}
	if action != "inspect" && !authz.AllowedOrganization(ctx, a.authorizer, id.UserID, id.EffectiveOrganizationID, id.Roles, authz.PermissionWorkbenchAgentUse) {
		return ctx, agentconfig.ErrForbidden
	}
	return ctx, nil
}
func productSelectionRoutes(app selectionHTTPService, bind func(context.Context, string, string) (context.Context, error)) []httproute.Descriptor {
	specs := []struct{ method, path, action string }{{"POST", "/inspect", "inspect"}, {"POST", "/prepare", "prepare"}, {"POST", "/contexts/:context_id/runs", "start"}, {"GET", "/contexts/:context_id/runs/:request_key", "read"}, {"POST", "/contexts/:context_id/runs/:request_key/resume", "resume"}}
	var routes []httproute.Descriptor
	for _, spec := range specs {
		permission := authz.PermissionLocalAgentWrite
		if spec.action == "inspect" {
			permission = collection.PermissionRead
		}
		routes = append(routes, httproute.Descriptor{Method: spec.method, Path: productSelectionBase + spec.path, Module: "product-selection-agent", Permission: permission, AuthPolicy: httproute.AuthPolicyVerifiedIdentity, OrganizationAccessPolicy: httproute.OrganizationAccessPolicyLiveWrite, RequestTimeout: 2 * time.Minute, Handler: httproute.WithRequestBodyReadTimeout(3*time.Second, func(c *gin.Context) {
			c.Header("Cache-Control", "no-store")
			contextID, key := c.Param("context_id"), c.Param("request_key")
			if c.Request.URL.RawQuery != "" || spec.action != "inspect" && spec.action != "prepare" && !agentconfig.UUID(contextID) {
				writeSelectionError(c, agent.ErrInvalid, contextID, key, false)
				return
			}
			if spec.action == "start" {
				keys := c.Request.Header.Values("Idempotency-Key")
				if len(keys) != 1 || !agentconfig.UUID(keys[0]) {
					writeSelectionError(c, agent.ErrInvalid, contextID, key, false)
					return
				}
				key = keys[0]
			}
			var in selectionInput
			var prepare selectionPrepareInput
			var start materialsStartBody
			var resume materialsResumeBody
			var e error
			switch spec.action {
			case "inspect":
				e = readAcquisitionImageJSON(c.Request, &in)
				if _, err := in.inputs(); err != nil {
					e = err
				}
				contextID = in.ContextID
			case "prepare":
				e = readAcquisitionImageJSON(c.Request, &prepare)
				if _, err := prepare.selectionInput.inputs(); err != nil {
					e = err
				}
				if !agentconfig.UUID(prepare.RequestKey) || prepare.GoalSummary == "" || !agent.ValidGoalSummary(prepare.GoalSummary) {
					e = agent.ErrInvalid
				}
				contextID, key = prepare.ContextID, prepare.RequestKey
			case "start":
				e = readAcquisitionImageJSON(c.Request, &start)
				if !materialsRefValid(start.ConfigurationSnapshotRef) {
					e = agent.ErrInvalid
				}
			case "read":
				e = readProductReviewGETBody(c)
				if !agentconfig.UUID(key) {
					e = agent.ErrInvalid
				}
			case "resume":
				e = readAcquisitionImageJSON(c.Request, &resume)
				revision, err := strconv.ParseUint(resume.Revision, 10, 64)
				if !agentconfig.UUID(key) || err != nil || revision < 1 || strconv.FormatUint(revision, 10) != resume.Revision || !materialsRefValid(resume.ConfigurationSnapshotRef) || !agent.ValidGoalSummary(resume.Feedback) {
					e = agent.ErrInvalid
				}
			}
			if e != nil {
				writeSelectionError(c, agent.ErrInvalid, contextID, key, false)
				return
			}
			if app == nil || bind == nil {
				writeSelectionError(c, agent.ErrUnavailable, contextID, key, false)
				return
			}
			ctx, e := bind(c.Request.Context(), c.GetHeader("Authorization"), spec.action)
			if e != nil {
				writeSelectionError(c, e, contextID, key, false)
				return
			}
			var out any
			var run agent.Record
			var snapshot agentconfig.Snapshot
			mayExist := spec.action == "read" || spec.action == "start" || spec.action == "resume"
			switch spec.action {
			case "inspect":
				_, out, e = app.inspectInput(ctx, in)
			case "prepare":
				snapshot, e = app.prepare(ctx, prepare)
				if e == nil {
					cost, err := snapshot.ExecutionModelProfile.MaximumCost()
					if err != nil {
						e = err
					} else {
						out = gin.H{"contextId": contextID, "requestKey": key, "configurationSnapshotRef": agent.ConfigurationSnapshotRef{Kind: agentconfig.SnapshotKind, ID: snapshot.ID, Digest: snapshot.Digest}, "modeSnapshotRef": snapshot.Request.ModeSnapshotRef, "modeRevision": snapshot.ExpectedModeRevision, "maximumTokens": snapshot.Request.Limits.Tokens, "maximumCostMicros": snapshot.Request.Limits.CostMicros, "currency": snapshot.Request.Limits.Currency, "maximumCallCostMicros": cost}
					}
				}
			case "start":
				run, _, e = app.start(ctx, contextID, key, start.ConfigurationSnapshotRef)
			case "read":
				run, snapshot, e = app.readRun(ctx, contextID, key)
			case "resume":
				revision, _ := strconv.ParseUint(resume.Revision, 10, 64)
				run, _, e = app.resume(ctx, contextID, key, resume.ConfigurationSnapshotRef, revision, resume.Feedback)
			}
			if e == nil && spec.action != "inspect" && spec.action != "prepare" {
				if snapshot.ID == "" {
					run, snapshot, e = app.readRun(ctx, contextID, key)
				}
				if e == nil {
					out, e = app.projectRun(ctx, run, snapshot)
				}
			}
			if e != nil {
				writeSelectionError(c, e, contextID, key, mayExist)
				return
			}
			c.JSON(http.StatusOK, out)
		})})
	}
	return routes
}
func validateProductSelectionDescriptor(d httproute.Descriptor) error {
	for _, expected := range productSelectionRoutes(nil, nil) {
		if d.Method != expected.Method || d.Path != expected.Path {
			continue
		}
		if d.Module != expected.Module || d.Permission != expected.Permission || d.AuthPolicy != expected.AuthPolicy || d.OrganizationAccessPolicy != expected.OrganizationAccessPolicy || d.OrganizationTargetResolver != nil || d.RequestTimeout != expected.RequestTimeout || d.RejectUnreadRequestBody != expected.RejectUnreadRequestBody || d.Handler == nil {
			return errors.New("selection route loses its exact live boundary")
		}
		return nil
	}
	return errors.New("selection route not admitted")
}
func writeSelectionError(c *gin.Context, e error, contextID, key string, mayExist bool) {
	status, code := http.StatusServiceUnavailable, "DEPENDENCY_UNAVAILABLE"
	switch {
	case errors.Is(e, agent.ErrInvalid), errors.Is(e, agentconfig.ErrInvalid), errors.Is(e, collection.ErrInvalid):
		status, code = 400, "INVALID_REQUEST"
	case errors.Is(e, agentconfig.ErrForbidden), errors.Is(e, review.ErrForbidden), errors.Is(e, collection.ErrForbidden):
		status, code = 403, "PERMISSION_DENIED"
	case errors.Is(e, agentconfig.ErrNotFound), errors.Is(e, collection.ErrNotFound):
		status, code = 404, "NOT_FOUND"
	case errors.Is(e, agent.ErrConflict), errors.Is(e, agentconfig.ErrConflict), errors.Is(e, collection.ErrConflict):
		status, code = 409, "AGENT_CONFLICT"
	case errors.Is(e, context.Canceled), errors.Is(e, context.DeadlineExceeded):
		status, code = 409, "OUTCOME_UNKNOWN"
	case errors.Is(e, agentconfig.ErrChanged), errors.Is(e, agentconfig.ErrNotEnabled), errors.Is(e, agentconfig.ErrRevision), errors.Is(e, agentconfig.ErrPrecondition):
		status, code = 409, e.Error()
	}
	out := gin.H{"code": code}
	if agentconfig.UUID(contextID) && agentconfig.UUID(key) {
		out["contextId"], out["requestKey"], out["runMayExist"] = contextID, key, mayExist
	}
	c.JSON(status, out)
}
