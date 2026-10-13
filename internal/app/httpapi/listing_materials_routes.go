package httpapi

import (
	"context"
	"encoding/hex"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"task-processor/internal/agent"
	"task-processor/internal/agentconfig"
	supplyhttp "task-processor/internal/app/supplychain/httpapi"
	"task-processor/internal/authidentity"
	"task-processor/internal/authz"
	"task-processor/internal/core/config"
	"task-processor/internal/httproute"
	kernelmodule "task-processor/internal/kernel/module"
	"task-processor/internal/listing/materials"
	"task-processor/internal/listing/preparation"
	record "task-processor/internal/listing/record/target"
	"task-processor/internal/product/review"
)

const listingMaterialsBase = supplyhttp.SupplyBasePath + "/records/:record_id/materials-agent"

type listingMaterialsHTTPService interface {
	prepare(context.Context, string, materialsPrepareInput) (agentconfig.Snapshot, error)
	start(context.Context, string, string, agent.ConfigurationSnapshotRef) (agent.Record, bool, error)
	readRun(context.Context, string, string) (agent.Record, agentconfig.Snapshot, error)
	resume(context.Context, string, string, agent.ConfigurationSnapshotRef, uint64, string) (agent.Record, bool, error)
	adopt(context.Context, string, string, agent.ConfigurationSnapshotRef, string, bool) (record.TargetReceipt, error)
	adoption(context.Context, string, string) (record.TargetReceipt, error)
	projectRun(context.Context, string, agent.Record, agentconfig.Snapshot) (materialsRunDTO, error)
}

type listingMaterialsModule struct{ app *listingMaterialsApplication }

func (listingMaterialsModule) Name() string                { return "listing-materials-agent" }
func (listingMaterialsModule) Enabled(*config.Config) bool { return true }
func (m listingMaterialsModule) Register(reg *kernelmodule.Registry) error {
	reg.AddRoutes(listingMaterialsRoutes(m.app, m.app.bindRequest)...)
	return nil
}

func (a *listingMaterialsApplication) bindRequest(ctx context.Context, bearer string) (context.Context, error) {
	ctx, err := (productReviewCapabilityBinder{now: time.Now}).Bind(ctx, bearer)
	if err != nil {
		return ctx, err
	}
	id, err := a.parent.freshIdentity(ctx)
	if err != nil {
		return ctx, err
	}
	ctx = authidentity.WithAuthenticatedIdentity(ctx, id)
	if !authz.AllowedOrganization(ctx, a.parent.authorizer, id.UserID, id.EffectiveOrganizationID, id.Roles, authz.PermissionWorkbenchAgentUse) {
		return ctx, agentconfig.ErrForbidden
	}
	return ctx, nil
}

type materialsStartBody struct {
	ConfigurationSnapshotRef agent.ConfigurationSnapshotRef `json:"configurationSnapshotRef"`
}
type materialsResumeBody struct {
	ConfigurationSnapshotRef agent.ConfigurationSnapshotRef `json:"configurationSnapshotRef"`
	Revision                 string                         `json:"revision"`
	Feedback                 string                         `json:"feedback"`
}
type materialsAdoptBody struct {
	ConfigurationSnapshotRef agent.ConfigurationSnapshotRef `json:"configurationSnapshotRef"`
	CandidateHash            string                         `json:"candidateHash"`
	HumanConfirmed           bool                           `json:"humanConfirmed"`
}

func materialsRefValid(ref agent.ConfigurationSnapshotRef) bool {
	return !ref.Absent() && ref.ValidOrAbsent()
}
func materialsHashValid(value string) bool {
	bytes, err := hex.DecodeString(value)
	return err == nil && len(bytes) == 32 && hex.EncodeToString(bytes) == value
}

func listingMaterialsRoutes(app listingMaterialsHTTPService, bind func(context.Context, string) (context.Context, error)) []httproute.Descriptor {
	specs := []struct{ method, suffix, action string }{
		{"POST", "/prepare", "prepare"}, {"POST", "/runs", "start"},
		{"GET", "/runs/:request_key", "read"}, {"POST", "/runs/:request_key/resume", "resume"},
		{"POST", "/runs/:request_key/adopt", "adopt"}, {"GET", "/runs/:request_key/adoption", "adoption"},
	}
	var routes []httproute.Descriptor
	for _, spec := range specs {
		routes = append(routes, httproute.Descriptor{Method: spec.method, Path: listingMaterialsBase + spec.suffix, Module: "listing-materials-agent", Permission: authz.PermissionLocalAgentWrite, AuthPolicy: httproute.AuthPolicyVerifiedIdentity, OrganizationAccessPolicy: httproute.OrganizationAccessPolicyLiveWrite, RequestTimeout: 2 * time.Minute, Handler: httproute.WithRequestBodyReadTimeout(3*time.Second, func(c *gin.Context) {
			c.Header("Cache-Control", "no-store")
			recordID, key := c.Param("record_id"), c.Param("request_key")
			if !agentconfig.UUID(recordID) || c.Request.URL.RawQuery != "" || spec.action != "prepare" && spec.action != "start" && !agentconfig.UUID(key) {
				writeMaterialsError(c, agentconfig.ErrInvalid, recordID, key, false)
				return
			}
			if spec.action == "start" {
				values := c.Request.Header.Values("Idempotency-Key")
				if len(values) != 1 || !agentconfig.UUID(values[0]) {
					writeMaterialsError(c, agentconfig.ErrInvalid, recordID, key, false)
					return
				}
				key = values[0]
			}
			var prepare materialsPrepareInput
			var start materialsStartBody
			var resume materialsResumeBody
			var adopt materialsAdoptBody
			var err error
			switch spec.action {
			case "read", "adoption":
				err = readProductReviewGETBody(c)
			case "prepare":
				err = readAcquisitionImageJSON(c.Request, &prepare)
				if !agentconfig.UUID(prepare.RequestKey) || !agent.ValidGoalSummary(prepare.GoalSummary) {
					err = agentconfig.ErrInvalid
				}
				if _, e := agentconfig.ParseModeRevision(prepare.ExpectedModeRevision); e != nil {
					err = e
				}
				key = prepare.RequestKey
			case "start":
				err = readAcquisitionImageJSON(c.Request, &start)
				if !materialsRefValid(start.ConfigurationSnapshotRef) {
					err = agentconfig.ErrInvalid
				}
			case "resume":
				err = readAcquisitionImageJSON(c.Request, &resume)
				revision, e := strconv.ParseUint(resume.Revision, 10, 64)
				if e != nil || revision == 0 || strconv.FormatUint(revision, 10) != resume.Revision || !materialsRefValid(resume.ConfigurationSnapshotRef) || !agent.ValidGoalSummary(resume.Feedback) {
					err = agentconfig.ErrInvalid
				}
			case "adopt":
				err = readAcquisitionImageJSON(c.Request, &adopt)
				if !materialsRefValid(adopt.ConfigurationSnapshotRef) || !materialsHashValid(adopt.CandidateHash) {
					err = agentconfig.ErrInvalid
				}
			}
			if err != nil {
				writeMaterialsError(c, agentconfig.ErrInvalid, recordID, key, false)
				return
			}
			if app == nil || bind == nil {
				writeMaterialsError(c, agentconfig.ErrUnavailable, recordID, key, false)
				return
			}
			ctx, err := bind(c.Request.Context(), c.GetHeader("Authorization"))
			if err != nil {
				writeMaterialsError(c, err, recordID, key, false)
				return
			}
			var out any
			var run agent.Record
			var snapshot agentconfig.Snapshot
			runMayExist := spec.action != "prepare"
			switch spec.action {
			case "prepare":
				snapshot, err = app.prepare(ctx, recordID, prepare)
				if err == nil {
					out = gin.H{"recordId": recordID, "requestKey": key, "configurationSnapshotRef": agent.ConfigurationSnapshotRef{Kind: agentconfig.SnapshotKind, ID: snapshot.ID, Digest: snapshot.Digest}, "modeSnapshotRef": snapshot.Request.ModeSnapshotRef, "modeRevision": snapshot.ExpectedModeRevision, "templateSelection": snapshot.Template}
				}
			case "start":
				run, runMayExist, err = app.start(ctx, recordID, key, start.ConfigurationSnapshotRef)
			case "read":
				run, snapshot, err = app.readRun(ctx, recordID, key)
			case "resume":
				revision, _ := strconv.ParseUint(resume.Revision, 10, 64)
				run, runMayExist, err = app.resume(ctx, recordID, key, resume.ConfigurationSnapshotRef, revision, resume.Feedback)
			case "adopt":
				out, err = app.adopt(ctx, recordID, key, adopt.ConfigurationSnapshotRef, adopt.CandidateHash, adopt.HumanConfirmed)
			case "adoption":
				out, err = app.adoption(ctx, recordID, key)
			}
			if err == nil && (spec.action == "start" || spec.action == "resume" || spec.action == "read") {
				out, err = app.projectRun(ctx, recordID, run, snapshot)
			}
			if err != nil {
				writeMaterialsError(c, err, recordID, key, runMayExist)
				return
			}
			c.JSON(http.StatusOK, out)
		})})
	}
	return routes
}

func validateListingMaterialsDescriptor(d httproute.Descriptor) error {
	for _, expected := range listingMaterialsRoutes(nil, nil) {
		if d.Method != expected.Method || d.Path != expected.Path {
			continue
		}
		if d.Module != expected.Module || d.Permission != expected.Permission || d.AuthPolicy != expected.AuthPolicy || d.OrganizationAccessPolicy != expected.OrganizationAccessPolicy || d.OrganizationTargetResolver != nil || d.RequestTimeout != expected.RequestTimeout || d.RejectUnreadRequestBody != expected.RejectUnreadRequestBody || d.Handler == nil {
			return errors.New("listing materials route loses exact live business boundary")
		}
		return nil
	}
	return errors.New("listing materials route not admitted")
}

func writeMaterialsError(c *gin.Context, err error, recordID, key string, runMayExist bool) {
	if runMayExist && (errors.Is(err, agentconfig.ErrChanged) || errors.Is(err, agentconfig.ErrNotEnabled) || errors.Is(err, agentconfig.ErrArchived) || errors.Is(err, agentconfig.ErrDefinition)) {
		err = agent.ErrConflict
	}
	status, code := http.StatusServiceUnavailable, "DEPENDENCY_UNAVAILABLE"
	switch {
	case errors.Is(err, agentconfig.ErrInvalid), errors.Is(err, agent.ErrInvalid), errors.Is(err, preparation.ErrInvalid), errors.Is(err, record.ErrInvalid), errors.Is(err, materials.ErrInvalidProposal):
		status, code = 400, "INVALID_REQUEST"
	case errors.Is(err, agentconfig.ErrForbidden), errors.Is(err, review.ErrForbidden), errors.Is(err, preparation.ErrForbidden), errors.Is(err, record.ErrForbidden):
		status, code = 403, "PERMISSION_DENIED"
	case errors.Is(err, agentconfig.ErrNotFound), errors.Is(err, record.ErrNotFound), errors.Is(err, preparation.ErrNotFound):
		status, code = 404, "NOT_FOUND"
	case errors.Is(err, errMaterialsHumanRequired):
		status, code = 409, "HUMAN_CONFIRMATION_REQUIRED"
	case errors.Is(err, record.ErrTargetUnknown), errors.Is(err, context.Canceled), errors.Is(err, context.DeadlineExceeded):
		status, code = 409, "OUTCOME_UNKNOWN"
	case errors.Is(err, agent.ErrConflict), errors.Is(err, agentconfig.ErrConflict), errors.Is(err, materials.ErrInspectionChanged), errors.Is(err, record.ErrConflict), errors.Is(err, preparation.ErrConflict):
		status, code = 409, "AGENT_CONFLICT"
	case errors.Is(err, record.ErrNotReady):
		status, code = 409, "NOT_READY"
	case errors.Is(err, record.ErrTooLarge), errors.Is(err, materials.ErrInspectionTooLarge):
		status, code = 413, "INPUT_TOO_LARGE"
	case errors.Is(err, agentconfig.ErrChanged), errors.Is(err, agentconfig.ErrNotEnabled), errors.Is(err, agentconfig.ErrArchived), errors.Is(err, agentconfig.ErrRevision), errors.Is(err, agentconfig.ErrPrecondition), errors.Is(err, agentconfig.ErrDefinition):
		status, code = 409, err.Error()
	}
	out := gin.H{"code": code}
	if agentconfig.UUID(recordID) && agentconfig.UUID(key) {
		base := strings.Replace(listingMaterialsBase, ":record_id", recordID, 1) + "/runs/" + key
		out["requestKey"], out["runMayExist"], out["runReadUrl"], out["adoptionReadUrl"] = key, runMayExist, base, base+"/adoption"
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(status, out)
}
