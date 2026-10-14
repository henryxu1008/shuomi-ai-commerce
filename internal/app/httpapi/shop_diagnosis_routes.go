package httpapi

import (
	"context"
	"errors"
	"github.com/gin-gonic/gin"
	"net/http"
	"strconv"
	"task-processor/internal/agent"
	"task-processor/internal/agentconfig"
	"task-processor/internal/authz"
	"task-processor/internal/core/config"
	"task-processor/internal/httproute"
	kernelmodule "task-processor/internal/kernel/module"
	cockpit "task-processor/internal/operationscockpit"
	"task-processor/internal/product/review"
	"task-processor/internal/shopdiagnosis"
	"time"
)

const shopDiagnosisBase = "/api/v1/workbench/shop-diagnosis"

type diagnosisHTTPService interface {
	inspectInput(context.Context, diagnosisInput) (agent.Binding, shopdiagnosis.Inspection, error)
	prepare(context.Context, diagnosisPrepareInput) (agentconfig.Snapshot, error)
	start(context.Context, string, string, agent.ConfigurationSnapshotRef) (agent.Record, bool, error)
	readRun(context.Context, string, string) (agent.Record, agentconfig.Snapshot, error)
	resume(context.Context, string, string, agent.ConfigurationSnapshotRef, uint64, string) (agent.Record, bool, error)
	projectRun(context.Context, agent.Record, agentconfig.Snapshot) (diagnosisRunDTO, error)
}
type shopDiagnosisModule struct{ app *shopDiagnosisApplication }

func (shopDiagnosisModule) Name() string                { return "shop-diagnosis-agent" }
func (shopDiagnosisModule) Enabled(*config.Config) bool { return true }
func (m shopDiagnosisModule) Register(reg *kernelmodule.Registry) error {
	reg.AddRoutes(shopDiagnosisRoutes(m.app, m.app.bindRequest)...)
	return nil
}
func (a *shopDiagnosisApplication) bindRequest(ctx context.Context, bearer, action string) (context.Context, error) {
	ctx, e := (productReviewCapabilityBinder{now: time.Now}).Bind(ctx, bearer)
	if e != nil {
		return ctx, e
	}
	ctx, e = a.bindNative(ctx, bearer)
	if e != nil {
		return ctx, e
	}
	id, e := a.freshIdentity(ctx)
	if e != nil {
		return ctx, e
	}
	if !authz.AllowedOrganization(ctx, a.authorizer, id.UserID, id.EffectiveOrganizationID, id.Roles, authz.PermissionWorkbenchAgentUse) {
		return ctx, agentconfig.ErrForbidden
	}
	return ctx, nil
}
func shopDiagnosisRoutes(app diagnosisHTTPService, bind func(context.Context, string, string) (context.Context, error)) []httproute.Descriptor {
	specs := []struct{ method, path, action string }{{"POST", "/inspect", "inspect"}, {"POST", "/prepare", "prepare"}, {"POST", "/contexts/:context_id/runs", "start"}, {"GET", "/contexts/:context_id/runs/:request_key", "read"}, {"POST", "/contexts/:context_id/runs/:request_key/resume", "resume"}}
	var routes []httproute.Descriptor
	for _, spec := range specs {
		permission := authz.PermissionLocalAgentWrite
		if spec.action == "inspect" {
			permission = authz.PermissionCockpitAdviceRead
		}
		routes = append(routes, httproute.Descriptor{Method: spec.method, Path: shopDiagnosisBase + spec.path, Module: "shop-diagnosis-agent", Permission: permission, AuthPolicy: httproute.AuthPolicyVerifiedIdentity, OrganizationAccessPolicy: httproute.OrganizationAccessPolicyLiveWrite, RequestTimeout: 2 * time.Minute, Handler: httproute.WithRequestBodyReadTimeout(3*time.Second, func(c *gin.Context) {
			c.Header("Cache-Control", "no-store")
			contextID, key := c.Param("context_id"), c.Param("request_key")
			if c.Request.URL.RawQuery != "" || spec.action != "inspect" && spec.action != "prepare" && !agentconfig.UUID(contextID) {
				writeDiagnosisError(c, agent.ErrInvalid, contextID, key, false)
				return
			}
			if spec.action == "start" {
				keys := c.Request.Header.Values("Idempotency-Key")
				if len(keys) != 1 || !agentconfig.UUID(keys[0]) {
					writeDiagnosisError(c, agent.ErrInvalid, contextID, key, false)
					return
				}
				key = keys[0]
			}
			var in diagnosisInput
			var prepare diagnosisPrepareInput
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
				if _, err := prepare.diagnosisInput.inputs(); err != nil {
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
				writeDiagnosisError(c, agent.ErrInvalid, contextID, key, false)
				return
			}
			if app == nil || bind == nil {
				writeDiagnosisError(c, agent.ErrUnavailable, contextID, key, false)
				return
			}
			ctx, e := bind(c.Request.Context(), c.GetHeader("Authorization"), spec.action)
			if e != nil {
				writeDiagnosisError(c, e, contextID, key, false)
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
						out = gin.H{"contextId": contextID, "requestKey": key, "inspectionRef": snapshot.Request.Binding.StoreDiagnosis, "configurationSnapshotRef": agent.ConfigurationSnapshotRef{Kind: agentconfig.SnapshotKind, ID: snapshot.ID, Digest: snapshot.Digest}, "modeSnapshotRef": snapshot.Request.ModeSnapshotRef, "modeRevision": snapshot.ExpectedModeRevision, "maximumTokens": snapshot.Request.Limits.Tokens, "maximumCostMicros": snapshot.Request.Limits.CostMicros, "currency": snapshot.Request.Limits.Currency, "maximumCallCostMicros": cost}
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
				writeDiagnosisError(c, e, contextID, key, mayExist)
				return
			}
			c.JSON(http.StatusOK, out)
		})})
	}
	return routes
}
func validateShopDiagnosisDescriptor(d httproute.Descriptor) error {
	for _, expected := range shopDiagnosisRoutes(nil, nil) {
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
func writeDiagnosisError(c *gin.Context, e error, contextID, key string, mayExist bool) {
	status, code := http.StatusServiceUnavailable, "DEPENDENCY_UNAVAILABLE"
	switch {
	case errors.Is(e, agent.ErrInvalid), errors.Is(e, agentconfig.ErrInvalid), errors.Is(e, cockpit.ErrInvalid):
		status, code = 400, "INVALID_REQUEST"
	case errors.Is(e, agentconfig.ErrForbidden), errors.Is(e, review.ErrForbidden), errors.Is(e, cockpit.ErrForbidden):
		status, code = 403, "PERMISSION_DENIED"
	case errors.Is(e, agentconfig.ErrNotFound), errors.Is(e, cockpit.ErrNotFound):
		status, code = 404, "NOT_FOUND"
	case errors.Is(e, agent.ErrConflict), errors.Is(e, agentconfig.ErrConflict), errors.Is(e, cockpit.ErrConflict):
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
