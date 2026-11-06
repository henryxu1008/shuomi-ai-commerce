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
	"task-processor/internal/product/research"
	"task-processor/internal/product/research/analysis"
	"task-processor/internal/product/review"
	"time"
)

const marketResearchAnalysisBase = "/api/v1/workbench/research/projects/:project_id/analysis"

func isResearchAnalysisRunHTTPPath(path string) bool {
	const prefix = "/api/v1/workbench/research/projects/"
	if !strings.HasPrefix(path, prefix) {
		return false
	}
	parts := strings.Split(strings.TrimPrefix(path, prefix), "/")
	return len(parts) >= 3 && agentconfig.UUID(parts[0]) && parts[1] == "analysis" && parts[2] == "runs" && (len(parts) == 3 || len(parts) == 5 && agentconfig.UUID(parts[3]) && parts[4] == "resume")
}

type researchHTTPService interface {
	inspectInput(context.Context, researchInput) (agent.Binding, analysis.Inspection, error)
	prepare(context.Context, researchPrepareInput) (agentconfig.Snapshot, error)
	readPreparation(context.Context, string, string) (agentconfig.Snapshot, error)
	recent(context.Context, string, string) (any, string, error)
	start(context.Context, string, string, agent.ConfigurationSnapshotRef) (agent.Record, bool, error)
	readRun(context.Context, string, string) (agent.Record, agentconfig.Snapshot, error)
	resume(context.Context, string, string, agent.ConfigurationSnapshotRef, uint64, string) (agent.Record, bool, error)
	projectRun(context.Context, agent.Record, agentconfig.Snapshot) (researchRunDTO, error)
}
type marketResearchAnalysisModule struct{ app *marketResearchApplication }

func (marketResearchAnalysisModule) Name() string                { return "market-research-agent" }
func (marketResearchAnalysisModule) Enabled(*config.Config) bool { return true }
func (m marketResearchAnalysisModule) Register(reg *kernelmodule.Registry) error {
	reg.AddRoutes(marketResearchAnalysisRoutes(m.app, m.app.bindRequest)...)
	return nil
}
func (a *marketResearchApplication) bindRequest(ctx context.Context, bearer, action string) (context.Context, error) {
	ctx, e := (productReviewCapabilityBinder{now: time.Now}).Bind(ctx, bearer)
	if e != nil {
		return ctx, e
	}
	id, e := a.readIdentity(ctx)
	if e != nil {
		return ctx, e
	}
	if action != "inspect" && !authz.AllowedOrganization(ctx, a.authorizer, id.UserID, id.EffectiveOrganizationID, id.Roles, authz.PermissionWorkbenchAgentUse) {
		return ctx, agentconfig.ErrForbidden
	}
	return ctx, nil
}
func marketResearchAnalysisRoutes(app researchHTTPService, bind func(context.Context, string, string) (context.Context, error)) []httproute.Descriptor {
	specs := []struct{ method, path, action string }{{"POST", "/inspect", "inspect"}, {"POST", "/prepare", "prepare"}, {"GET", "/preparations/:request_key", "preparation"}, {"GET", "/runs", "recent"}, {"POST", "/runs", "start"}, {"GET", "/runs/:request_key", "read"}, {"POST", "/runs/:request_key/resume", "resume"}}

	var routes []httproute.Descriptor
	for _, spec := range specs {
		permission := authz.PermissionLocalAgentWrite
		if spec.action == "inspect" {
			permission = collection.PermissionRead
		}
		routes = append(routes, httproute.Descriptor{Method: spec.method, Path: marketResearchAnalysisBase + spec.path, Module: "market-research-agent", Permission: permission, AuthPolicy: httproute.AuthPolicyVerifiedIdentity, OrganizationAccessPolicy: httproute.OrganizationAccessPolicyLiveWrite, RequestTimeout: 2 * time.Minute, Handler: httproute.WithRequestBodyReadTimeout(3*time.Second, func(c *gin.Context) {
			c.Header("Cache-Control", "no-store")
			contextID, key := c.Param("project_id"), c.Param("request_key")
			if (c.Request.URL.RawQuery != "" && spec.action != "recent") || !agentconfig.UUID(contextID) {
				writeResearchAnalysisError(c, agent.ErrInvalid, contextID, key, false)
				return
			}
			if spec.action == "start" {
				keys := c.Request.Header.Values("Idempotency-Key")
				if len(keys) != 1 || !agentconfig.UUID(keys[0]) {
					writeResearchAnalysisError(c, agent.ErrInvalid, contextID, key, false)
					return
				}
				key = keys[0]
			}
			var in researchInput
			var prepare researchPrepareInput
			var start materialsStartBody
			var resume materialsResumeBody
			var e error
			switch spec.action {
			case "inspect":
				e = readAcquisitionImageJSON(c.Request, &in)
				in.ProjectID = contextID
				if _, err := in.inputs(); err != nil {
					e = err
				}
				contextID = in.ProjectID
			case "prepare":
				e = readAcquisitionImageJSON(c.Request, &prepare)
				prepare.ProjectID = contextID
				if _, err := prepare.researchInput.inputs(); err != nil {
					e = err
				}
				if !agentconfig.UUID(prepare.RequestKey) || prepare.GoalSummary == "" || !agent.ValidGoalSummary(prepare.GoalSummary) {
					e = agent.ErrInvalid
				}
				contextID, key = prepare.ProjectID, prepare.RequestKey
			case "start":
				e = readAcquisitionImageJSON(c.Request, &start)
				if !materialsRefValid(start.ConfigurationSnapshotRef) {
					e = agent.ErrInvalid
				}
			case "recent":
				e = readProductReviewGETBody(c)
				q := c.Request.URL.Query()
				if len(q) > 1 || len(q["after"]) > 1 || len(q) > 0 && len(q["after"]) == 0 || q.Get("after") != "" && !agentconfig.UUID(q.Get("after")) {
					e = agent.ErrInvalid
				}
			case "read", "preparation":
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
				writeResearchAnalysisError(c, agent.ErrInvalid, contextID, key, false)
				return
			}
			if app == nil || bind == nil {
				writeResearchAnalysisError(c, agent.ErrUnavailable, contextID, key, false)
				return
			}
			mayExist := agentconfig.UUID(key) && spec.action != "inspect"
			ctx, e := bind(c.Request.Context(), c.GetHeader("Authorization"), spec.action)
			if e != nil {
				writeResearchAnalysisError(c, e, contextID, key, mayExist)
				return
			}
			var out any
			var run agent.Record
			var snapshot agentconfig.Snapshot
			mayExist = spec.action == "prepare" || spec.action == "preparation" || spec.action == "read" || spec.action == "start" || spec.action == "resume"
			switch spec.action {
			case "inspect":
				_, out, e = app.inspectInput(ctx, in)
			case "recent":
				var next string
				out, next, e = app.recent(ctx, contextID, c.Request.URL.Query().Get("after"))
				out = gin.H{"items": out, "nextCursor": next}
			case "preparation":
				snapshot, e = app.readPreparation(ctx, contextID, key)
				if e == nil {
					out, e = researchPreparationDTO(snapshot)
				}
				if errors.Is(e, agentconfig.ErrNotFound) {
					mayExist = false
				}
			case "prepare":
				snapshot, e = app.prepare(ctx, prepare)
				if e == nil {
					out, e = researchPreparationDTO(snapshot)
				}
			case "start":
				run, mayExist, e = app.start(ctx, contextID, key, start.ConfigurationSnapshotRef)
				mayExist = mayExist || e == nil
			case "read":
				run, snapshot, e = app.readRun(ctx, contextID, key)
			case "resume":
				revision, _ := strconv.ParseUint(resume.Revision, 10, 64)
				run, _, e = app.resume(ctx, contextID, key, resume.ConfigurationSnapshotRef, revision, resume.Feedback)
			}
			if e == nil && spec.action != "inspect" && spec.action != "prepare" && spec.action != "preparation" && spec.action != "recent" {
				if snapshot.ID == "" {
					run, snapshot, e = app.readRun(ctx, contextID, key)
				}
				if e == nil {
					out, e = app.projectRun(ctx, run, snapshot)
				}
			}
			if e != nil {
				writeResearchAnalysisError(c, e, contextID, key, mayExist)
				return
			}
			c.JSON(http.StatusOK, out)
		})})
	}
	return routes
}
func validateMarketResearchAnalysisDescriptor(d httproute.Descriptor) error {
	for _, expected := range marketResearchAnalysisRoutes(nil, nil) {
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
func writeResearchAnalysisError(c *gin.Context, e error, contextID, key string, mayExist bool) {
	status, code := http.StatusServiceUnavailable, "DEPENDENCY_UNAVAILABLE"
	switch {
	case errors.Is(e, research.ErrInvalid), errors.Is(e, agent.ErrInvalid), errors.Is(e, agentconfig.ErrInvalid), errors.Is(e, collection.ErrInvalid):
		status, code = 400, "INVALID_REQUEST"
	case errors.Is(e, research.ErrForbidden), errors.Is(e, agentconfig.ErrForbidden), errors.Is(e, review.ErrForbidden), errors.Is(e, collection.ErrForbidden):
		status, code = 403, "PERMISSION_DENIED"
	case errors.Is(e, agentconfig.ErrNotFound), errors.Is(e, collection.ErrNotFound):
		status, code = 404, "NOT_FOUND"
	case errors.Is(e, research.ErrConflict), errors.Is(e, research.ErrArchived), errors.Is(e, agent.ErrConflict), errors.Is(e, agentconfig.ErrConflict), errors.Is(e, collection.ErrConflict):
		status, code = 409, "AGENT_CONFLICT"
	case errors.Is(e, context.Canceled), errors.Is(e, context.DeadlineExceeded):
		status, code = 409, "OUTCOME_UNKNOWN"
	case errors.Is(e, agentconfig.ErrChanged), errors.Is(e, agentconfig.ErrNotEnabled), errors.Is(e, agentconfig.ErrRevision), errors.Is(e, agentconfig.ErrPrecondition):
		status, code = 409, e.Error()
	}
	out := gin.H{"code": code}
	if agentconfig.UUID(contextID) && agentconfig.UUID(key) {
		out["contextId"], out["requestKey"], out["operationMayExist"] = contextID, key, mayExist
	}
	c.JSON(status, out)
}

func researchPreparationDTO(snapshot agentconfig.Snapshot) (any, error) {
	cost, e := snapshot.ExecutionModelProfile.MaximumCost()
	if e != nil {
		return nil, e
	}
	return gin.H{"contextId": snapshot.Request.Binding.ContextID, "requestKey": snapshot.Request.Key, "configurationSnapshotRef": agent.ConfigurationSnapshotRef{Kind: agentconfig.SnapshotKind, ID: snapshot.ID, Digest: snapshot.Digest}, "modeSnapshotRef": snapshot.Request.ModeSnapshotRef, "modeRevision": snapshot.ExpectedModeRevision, "maximumTokens": snapshot.Request.Limits.Tokens, "maximumCostMicros": snapshot.Request.Limits.CostMicros, "currency": snapshot.Request.Limits.Currency, "maximumCallCostMicros": cost, "referencePriceVersion": snapshot.ExecutionModelProfile.OfficialTariff.ReferencePriceVersion()}, nil
}
