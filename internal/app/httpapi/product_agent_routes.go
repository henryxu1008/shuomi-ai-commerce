package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"task-processor/internal/agent"
	"task-processor/internal/agentconfig"
	confighttp "task-processor/internal/agentconfig/httpapi"
	"task-processor/internal/aicapability"
	"task-processor/internal/authidentity"
	"task-processor/internal/authz"
	"task-processor/internal/core/config"
	"task-processor/internal/httproute"
	kernelmodule "task-processor/internal/kernel/module"
	"task-processor/internal/knowledge"
	"task-processor/internal/product/collection"
	"task-processor/internal/product/contentagent"
	"task-processor/internal/product/enrichment"
	"task-processor/internal/product/review"
	skill "task-processor/internal/skill"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

const productAgentBase = productAcquisitionBase + "/:operation_id/product-agent/runs"
const productContentAgentBase = productAcquisitionBase + "/:operation_id/content-agent/runs"

func isProductAgentHTTPPath(path string) bool {
	const ownPrefix = "/api/v1/workbench/collections/items/"
	if strings.HasPrefix(path, ownPrefix) {
		parts := strings.Split(strings.TrimPrefix(path, ownPrefix), "/")
		return len(parts) >= 3 && acquisitionHTTPUUID(parts[0]) && parts[1] == "content-agent" && parts[2] == "runs" && (len(parts) == 3 || len(parts) == 4 && acquisitionHTTPUUID(parts[3]) || len(parts) == 5 && acquisitionHTTPUUID(parts[3]) && (parts[4] == "resume" || parts[4] == "review"))
	}
	if !strings.HasPrefix(path, productAcquisitionBase+"/") {
		return false
	}
	parts := strings.Split(strings.TrimPrefix(path, productAcquisitionBase+"/"), "/")
	if len(parts) < 3 || !acquisitionHTTPUUID(parts[0]) || parts[1] != "product-agent" && parts[1] != "content-agent" || parts[2] != "runs" {
		return false
	}
	return len(parts) == 3 || len(parts) == 4 && acquisitionHTTPUUID(parts[3]) || len(parts) == 5 && acquisitionHTTPUUID(parts[3]) && (parts[4] == "resume" || parts[4] == "review")
}

type productAgentModule struct {
	application   *productAgentApplication
	routes        []httproute.Descriptor
	recoverPoints func(context.Context) (int, error)
}

func (productAgentModule) Name() string                { return "product-agent" }
func (productAgentModule) Enabled(*config.Config) bool { return true }
func (m productAgentModule) Register(reg *kernelmodule.Registry) error {
	reg.AddRoutes(m.routes...)
	return nil
}

func WithProductAgent(deps ProductAgentDependencies) CurrentApplicationOption {
	return func(options *currentApplicationOptions) {
		options.productAgent = &deps
		options.productAgents++
		options.agentConfigurationDB = deps.RunDB
	}
}

func buildProductAgentModule(ctx context.Context, productDB *gorm.DB, deps routeAuthDependencies, auth *authz.ListingKitAuthorizer, agentCfg ProductAgentDependencies, appCfg *config.Config) (kernelmodule.Module, error) {
	receipts, err := buildPublishedAcquisitionReader(ctx, productDB, appCfg, deps, auth)
	if err != nil {
		return nil, err
	}
	a, err := buildProductAgentApplication(ctx, agentCfg.ReviewDB, receipts, deps.organizationResolver, auth, agentCfg)
	if err != nil {
		return nil, err
	}
	a.originExecution = productAgentOriginAuthorization(appCfg, auth)
	if a.content != nil {
		a.content.originExecution = a.originExecution
	}
	routes := productAgentProfileRoutes(a, agentCfg.ContentTitleOnly)
	// Candidate intake returns an ID understood by the existing review UI.
	// Its original decision/Apply handlers retain their current permission checks.
	binder := productReviewCapabilityBinder{now: time.Now}
	for _, route := range productReviewOnlyRoutes(a.reviews, binder.Bind, a.projectKnowledge) {
		previous := route.Handler
		route.Handler = func(c *gin.Context) {
			c.Set(productReviewModeProjectorKey, productReviewModeProjector(a.projectReviewMode))
			previous(c)
		}
		routes = append(routes, route)
	}
	return productAgentModule{application: a, routes: routes, recoverPoints: a.points.RecoverDue}, nil
}

type productAgentStepDTO struct {
	Step         int    `json:"step"`
	Tool         string `json:"tool,omitempty"`
	CallID       string `json:"callId,omitempty"`
	InvocationID string `json:"invocationId,omitempty"`
	AuditStatus  string `json:"auditStatus,omitempty"`
}
type productAgentResultDTO struct {
	SkillVersionRef       agent.SkillVersionRef       `json:"skillVersionRef,omitzero"`
	ContentSelection      enrichment.ContentSelection `json:"contentSelection,omitzero"`
	ExecutionMode         agent.ExecutionMode         `json:"executionMode,omitempty"`
	ModeRevision          string                      `json:"modeRevision,omitempty"`
	ModeSnapshotRef       *agent.ModeSnapshotRef      `json:"modeSnapshotRef,omitempty"`
	ModeUnavailableReason string                      `json:"modeUnavailableReason,omitempty"`
	PolicyAdoption        *review.PolicyProvenance    `json:"policyAdoption,omitempty"`
	ReviewID              string                      `json:"reviewId,omitempty"`
	TemplateSelection     *agentconfig.TemplateRef    `json:"templateSelection,omitempty"`
	Knowledge             *productKnowledgeDTO        `json:"knowledge,omitempty"`
	RunID                 string                      `json:"runId"`
	RequestKey            string                      `json:"requestKey"`
	OperationID           string                      `json:"operationId"`
	ContextKind           string                      `json:"contextKind,omitempty"`
	ProductKey            string                      `json:"productKey"`
	CatalogVersion        string                      `json:"catalogVersion"`
	PublicationID         string                      `json:"publicationId"`
	TargetPlatform        string                      `json:"targetPlatform"`
	Phase                 agent.Phase                 `json:"phase"`
	Revision              string                      `json:"revision"`
	StopReason            agent.StopReason            `json:"stopReason,omitempty"`
	HumanReviewRequired   bool                        `json:"humanReviewRequired"`
	Candidate             enrichment.Candidate        `json:"candidate"`
	Confidence            []agent.FieldConfidence     `json:"confidence"`
	Unresolved            []string                    `json:"unresolved"`
	Steps                 []productAgentStepDTO       `json:"steps"`
	Tokens                int64                       `json:"tokens"`
	EstimatedCostMicros   int64                       `json:"estimatedCostMicros"`
	Currency              string                      `json:"currency"`
	UsageStatus           string                      `json:"usageStatus"`
	CanSubmitReview       bool                        `json:"canSubmitReview"`
}

func productAgentRoutes(a *productAgentApplication) []httproute.Descriptor {
	return registeredProductAgentRoutes(a, false)
}

// Mounting and global admission share the trusted profile's exact inventory.
func productAgentProfileRoutes(a *productAgentApplication, titleOnly bool) []httproute.Descriptor {
	var routes []httproute.Descriptor
	if !titleOnly {
		routes = productAgentRoutes(a)
	}
	var content *productAgentApplication
	if a != nil {
		content = a.content
	}
	return append(routes, registeredProductAgentRoutes(content, true)...)
}

func registeredProductAgentRoutes(a *productAgentApplication, content bool) []httproute.Descriptor {
	return productAgentRoutesForSource(a, content, false)
}

func productAgentRoutesForSource(a *productAgentApplication, content, own bool) []httproute.Descriptor {
	base := productAgentBase
	if content {
		base = productContentAgentBase
	}
	if own {
		base = ownContentAgentBase
	}
	specs := []struct{ method, path, action string }{{"POST", base, "start"}, {"GET", base + "/:request_key", "read"}, {"POST", base + "/:request_key/resume", "resume"}, {"POST", base + "/:request_key/review", "review"}}
	var routes []httproute.Descriptor
	for _, spec := range specs {
		spec := spec
		routes = append(routes, httproute.Descriptor{Method: spec.method, Path: spec.path, Module: "product-agent", Permission: authz.PermissionLocalAgentWrite, AuthPolicy: httproute.AuthPolicyVerifiedIdentity, OrganizationAccessPolicy: httproute.OrganizationAccessPolicyLiveWrite, RequestTimeout: 2 * time.Minute, Handler: func(c *gin.Context) {
			if a == nil {
				writeProductAgentError(c, agent.ErrUnavailable)
				return
			}
			operationID := c.Param("operation_id")
			if own {
				operationID = c.Param("item_id")
			}
			key := c.Param("request_key")
			if spec.action == "start" {
				values := c.Request.Header.Values("Idempotency-Key")
				if len(values) != 1 {
					writeProductAgentError(c, agent.ErrInvalid)
					return
				}
				key = values[0]
			}
			if !acquisitionHTTPUUID(operationID) || !acquisitionHTTPUUID(key) || c.Request.URL.RawQuery != "" {
				writeProductAgentError(c, agent.ErrInvalid)
				return
			}
			ctx, err := (productReviewCapabilityBinder{now: time.Now}).Bind(c.Request.Context(), c.GetHeader("Authorization"))
			if err != nil {
				writeProductAgentError(c, err)
				return
			}
			i, err := a.freshIdentity(ctx)
			if err != nil {
				writeProductAgentError(c, err)
				return
			}
			ctx = authidentity.WithAuthenticatedIdentity(ctx, i)
			if !authz.AllowedOrganization(ctx, a.authorizer, i.UserID, i.EffectiveOrganizationID, i.Roles, authz.PermissionWorkbenchAgentUse) {
				writeProductAgentError(c, agentconfig.ErrForbidden)
				return
			}
			ctx, err = knowledgeRequestContext(ctx)
			if err != nil {
				writeProductAgentError(c, err)
				return
			}
			var body productAgentRequestBody
			if spec.action == "read" {
				if err = readProductReviewGETBody(c); err != nil {
					writeProductAgentError(c, err)
					return
				}
			} else {
				if err = readAcquisitionImageJSON(c.Request, &body); err != nil || (spec.action != "resume" && (body.Revision != "" || body.Feedback != "")) || (spec.action != "start" && (body.TargetPlatform != "" || body.ExpectedModeRevision != "")) {
					writeProductAgentError(c, agent.ErrInvalid)
					return
				}
			}
			if content && spec.action == "start" && (body.ContentSelection == nil || !body.ContentSelection.Valid()) || (!content || spec.action != "start") && body.ContentSelection != nil {
				writeProductAgentError(c, agent.ErrInvalid)
				return
			}
			if (own && spec.action == "start") != (body.OwnSource != nil) {
				writeProductAgentError(c, agent.ErrInvalid)
				return
			}
			selection, selectionErr := body.knowledgeSelection(spec.action)
			if selectionErr != nil {
				writeProductAgentError(c, selectionErr)
				return
			}
			template, templateErr := body.templateSelection(spec.action)
			if templateErr != nil {
				writeProductAgentError(c, templateErr)
				return
			}
			method, methodErr := body.skillSelection(spec.action)
			if methodErr != nil || method != nil && (!content || body.ContentSelection == nil || !body.ContentSelection.TitleOnly()) {
				writeProductAgentError(c, agent.ErrInvalid)
				return
			}
			var record agent.Record
			var binding agent.Binding
			runMayExist := false
			if spec.action == "start" {
				if own {
					binding, runMayExist, err = a.ownStartBinding(ctx, i, operationID, key, body)
				} else {
					binding, err = a.binding(ctx, operationID, body.TargetPlatform)
				}
				if err == nil && !own {
					var existing agent.Record
					var found bool
					existing, found, err = a.store.Lookup(ctx, agent.Scope{OrganizationID: i.TenantID, ActorID: i.UserID}, binding, key)
					if err == nil && found {
						runMayExist = true
						binding = existing.State.Request.Binding
						if binding.TargetPlatform != body.TargetPlatform {
							err = agent.ErrConflict
						}
					} else if err == nil {
						binding, err = a.currentAcquisitionBinding(ctx, i, operationID, body.TargetPlatform)
					}
				}
				if err == nil {
					var request agent.Request
					if content {
						request, err = a.startTextRequestWithSkill(ctx, binding, key, selection, "", template, aicapability.ModelProfile{}, "", body.ExpectedModeRevision, *body.ContentSelection, method)
					} else {
						request, err = a.startRequest(ctx, binding, key, selection, template, body.ExpectedModeRevision)
					}
					if err == nil {
						var attempted bool
						record, attempted, err = a.runtime.StartWithClaimAttempt(ctx, request)
						runMayExist = runMayExist || attempted
					}
				}
			} else {
				if own {
					var found bool
					record, found, err = a.store.LookupContext(ctx, agent.Scope{OrganizationID: i.TenantID, ActorID: i.UserID}, contentagent.OwnProductContext, operationID, key)
					if err == nil && !found {
						err = agent.ErrConflict
					}
				} else {
					record, err = a.store.Read(ctx, agent.Scope{OrganizationID: i.TenantID, ActorID: i.UserID}, operationID, key)
				}
				if err == nil {
					runMayExist = true
					binding = record.State.Request.Binding
					_, err = a.Authorize(ctx, binding)
				}
				if err == nil && content != (record.State.Request.PolicyVersion == enrichment.ContentPolicyVersion) {
					err = agent.ErrConflict
				}
				if err == nil && spec.action == "resume" {
					if err = a.checkContentTitleSelection(record.State.Request.ContentSelection); err != nil {
						writeProductAgentRunError(c, err, true)
						return
					}
					revision, parseErr := strconv.ParseUint(body.Revision, 10, 64)
					if parseErr != nil || revision == 0 || strconv.FormatUint(revision, 10) != body.Revision {
						err = agent.ErrInvalid
					} else {
						record, err = a.runtime.Resume(ctx, record.State.Request, revision, body.Feedback)
					}
				}
			}
			if err != nil {
				writeProductAgentRunError(c, err, runMayExist)
				return
			}
			if spec.action == "review" {
				if !agentRunReviewable(record.State) {
					writeProductAgentError(c, agent.ErrConflict)
					return
				}
				if selectionErr := a.checkContentTitleSelection(record.State.Request.ContentSelection); selectionErr != nil {
					// Existing receipts remain owned by Review. Delegate an exact
					// replay to its normal fingerprint/auth preflight; never use
					// the current View as a replacement for the saved receipt.
					_, found, lookupErr := a.reviews.FindAgentReview(ctx, record.State.RunID)
					if lookupErr != nil {
						writeProductAgentError(c, lookupErr)
						return
					}
					if !found {
						writeProductAgentError(c, selectionErr)
						return
					}
				}
				input := agentReviewInput(binding, record.State.Request.PolicyVersion, record.State.Candidate)
				input.Execution = agentExecutionProvenance(record.State)
				input.ContentSelection = record.State.Request.ContentSelection
				input.ContextProvenance, err = agentContextProvenance(record.State)
				if err != nil {
					writeProductAgentError(c, err)
					return
				}
				view, e := a.reviews.CreateFromCandidate(ctx, "agent:"+record.State.RunID, input)
				if e != nil {
					writeProductAgentError(c, e)
					return
				}
				c.Header("Cache-Control", "no-store")
				c.JSON(http.StatusOK, gin.H{"proposalId": view.ID, "requestKey": key, "operationId": operationID})
				return
			}
			if spec.action == "start" || spec.action == "resume" {
				if _, err := a.finishTitleCandidate(ctx, record); err != nil {
					writeProductAgentRunError(c, err, true)
					return
				}
			}
			state := record.State
			provenance, provenanceErr := agentContextProvenance(state)
			if provenanceErr != nil {
				writeProductAgentError(c, provenanceErr)
				return
			}
			result := productAgentResultDTO{RunID: state.RunID, RequestKey: key, OperationID: operationID, ProductKey: binding.ProductKey, CatalogVersion: binding.CatalogVersion, PublicationID: binding.PublicationID, TargetPlatform: binding.TargetPlatform, Phase: state.Phase, Revision: strconv.FormatUint(state.Revision, 10), StopReason: state.StopReason, HumanReviewRequired: true, Candidate: state.Candidate, Confidence: state.Confidence, Unresolved: state.Unresolved, Steps: []productAgentStepDTO{}, Tokens: state.Usage.Tokens, EstimatedCostMicros: state.Usage.CostMicros, Currency: state.Request.Limits.Currency, UsageStatus: "observed", CanSubmitReview: agentRunReviewable(state) && a.checkContentTitleSelection(state.Request.ContentSelection) == nil}
			result.ContentSelection = state.Request.ContentSelection
			if own {
				result.ContextKind = contentagent.OwnProductContext
			}
			result.SkillVersionRef = state.Request.SkillVersionRef
			if content {
				result.Unresolved = contentRunDiagnostics(state)
			}
			if state.PendingInvocationID != "" || state.StopReason == agent.StopUsageUnknown || state.StopReason == agent.StopModelUnknown || state.Phase == agent.Running {
				result.UsageStatus = "unknown_reserved"
			}
			result.Knowledge = a.projectKnowledge(ctx, provenance)
			if a.configuration == nil {
				writeProductAgentError(c, agentconfig.ErrUnavailable)
				return
			}
			frozen, frozenErr := a.configuration.LoadSnapshot(ctx, state.Scope, state.Request.ConfigurationSnapshotRef)
			if frozenErr != nil {
				writeProductAgentError(c, frozenErr)
				return
			}
			result.TemplateSelection = frozen.Template
			if state.Request.ModeSnapshotRef.Absent() {
				result.ModeUnavailableReason = "EXECUTION_MODE_NOT_FROZEN"
				result.CanSubmitReview = false
			} else {
				mode, modeErr := a.configuration.LoadModeSnapshot(ctx, agent.ExecutionIdentity{OrganizationID: state.Scope.OrganizationID, ActorID: state.Scope.ActorID, MemberID: frozen.MemberID}, state.Request.ModeSnapshotRef)
				if modeErr != nil {
					writeProductAgentError(c, modeErr)
					return
				}
				ref := mode.Ref()
				result.ExecutionMode, result.ModeRevision, result.ModeSnapshotRef = mode.Mode, mode.Revision, &ref
			}
			if view, found, err := a.reviews.FindAgentReview(ctx, state.RunID); err != nil {
				writeProductAgentError(c, err)
				return
			} else if found {
				result.ReviewID = view.ID
				result.PolicyAdoption = view.PolicyAdoption
				if view.PolicyAdoption != nil && view.State == "applied" {
					result.HumanReviewRequired = false
					result.CanSubmitReview = false
				}
			}
			for _, step := range state.History {
				result.Steps = append(result.Steps, productAgentStepDTO{Step: step.Step, Tool: step.Tool.ID, CallID: step.CallID, InvocationID: step.InvocationID, AuditStatus: string(step.AuditStatus)})
			}
			wire, e := json.Marshal(result)
			if e != nil || len(wire) > 128<<10 {
				writeProductAgentError(c, agent.ErrUnavailable)
				return
			}
			c.Header("Cache-Control", "no-store")
			c.Data(http.StatusOK, "application/json; charset=utf-8", wire)
		}})
	}
	return routes
}

func agentRunReviewable(s agent.State) bool {
	if !agent.ValidContextCitations(s.Request.ContextSnapshotRef, s.ContextCitationRefs) || s.Phase != agent.HumanReviewRequired || s.Validation == nil || !s.Validation.Valid || s.StopReason != "" {
		return false
	}
	if s.Request.PolicyVersion == enrichment.ContentPolicyVersion {
		return len(s.Unresolved) == 0 && len(s.Candidate.Warnings) == 0 && len(s.Candidate.Rejections) == 0 && s.Request.ContentSelection.Valid() && len(s.Candidate.Changes) == len(s.Request.ContentSelection.Fields()) && contentCandidateEvidenceObserved(s.Request, s.Candidate, s.History)
	}
	return s.Request.ContentSelection.Absent() && len(s.Candidate.Changes) == 1 && agentCandidateEvidenceObserved(s.Request.Binding, s.Candidate, s.History)
}

func contentRunDiagnostics(s agent.State) []string {
	items := append([]string(nil), s.Unresolved...)
	for _, warning := range s.Candidate.Warnings {
		items = append(items, warning.Code+": "+warning.Message)
	}
	for _, rejection := range s.Candidate.Rejections {
		items = append(items, rejection.Code+": "+rejection.Message)
	}
	return items
}

func writeProductAgentRunError(c *gin.Context, err error, runMayExist bool) {
	// Preserve the original request key once an execution may have been claimed.
	if runMayExist && (errors.Is(err, agentconfig.ErrChanged) || errors.Is(err, agentconfig.ErrNotEnabled) || errors.Is(err, agentconfig.ErrArchived) || errors.Is(err, agentconfig.ErrDefinition)) {
		err = agent.ErrConflict
	}
	writeProductAgentError(c, err)
}

func writeProductAgentError(c *gin.Context, err error) {
	for _, e := range []error{agentconfig.ErrInvalid, agentconfig.ErrForbidden, agentconfig.ErrNotFound, agentconfig.ErrConflict, agentconfig.ErrChanged, agentconfig.ErrNotEnabled, agentconfig.ErrArchived, agentconfig.ErrDefault, agentconfig.ErrDefinition, agentconfig.ErrRevision, agentconfig.ErrPrecondition, agentconfig.ErrUnavailable} {
		if errors.Is(err, e) {
			confighttp.Failure(c, err)
			return
		}
	}
	status, code := http.StatusServiceUnavailable, "PRODUCT_AGENT_UNAVAILABLE"
	switch {
	case errors.Is(err, skill.ErrInvalid), errors.Is(err, skill.ErrUnsupported):
		status, code = 400, err.Error()
	case errors.Is(err, skill.ErrForbidden):
		status, code = 403, skill.ErrForbidden.Error()
	case errors.Is(err, skill.ErrConflict), errors.Is(err, skill.ErrNotEnabled), errors.Is(err, skill.ErrNotFound), errors.Is(err, skill.ErrRevision):
		status, code = 409, err.Error()
	case errors.Is(err, skill.ErrUnavailable):
		status, code = 503, skill.ErrUnavailable.Error()
	case errors.Is(err, knowledge.ErrContextTooLarge):
		status, code = 409, knowledge.ErrContextTooLarge.Error()
	case errors.Is(err, knowledge.ErrForbidden):
		status, code = 403, knowledge.ErrForbidden.Error()
	case errors.Is(err, knowledge.ErrInactive), errors.Is(err, knowledge.ErrNotReadable), errors.Is(err, knowledge.ErrNotFound):
		status, code = 409, err.Error()
	case errors.Is(err, knowledge.ErrConflict), errors.Is(err, knowledge.ErrIntegrity):
		status, code = 409, "AGENT_CONFLICT"
	case errors.Is(err, agent.ErrInvalid), errors.Is(err, review.ErrInvalid):
		status, code = 400, "INVALID_AGENT_REQUEST"
	case errors.Is(err, collection.ErrForbidden), errors.Is(err, review.ErrForbidden):
		status, code = 403, "FORBIDDEN"
	case errors.Is(err, collection.ErrNotFound), errors.Is(err, collection.ErrConflict), errors.Is(err, agent.ErrConflict), errors.Is(err, review.ErrConflict):
		status, code = 409, "AGENT_CONFLICT"
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(status, gin.H{"code": code})
}
