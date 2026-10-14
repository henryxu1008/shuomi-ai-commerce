package httpapi

import (
	"context"
	"github.com/gin-gonic/gin"
	"net/http"
	"strconv"
	"task-processor/internal/agent"
	"task-processor/internal/agentconfig"
	"task-processor/internal/authidentity"
	"task-processor/internal/authz"
	"task-processor/internal/httproute"
	"task-processor/internal/imageagent"
	"time"
)

const artworkAgentBase = "/api/v1/workbench/pod-artwork"

func (m fullImageModule) artworkRoutes() []httproute.Descriptor {
	result := []httproute.Descriptor{}
	for _, a := range []struct{ method, path, action string }{
		{http.MethodPost, "/prepare", "prepare"}, {http.MethodGet, "/by-key/:context_id/:request_id", "by-key"}, {http.MethodGet, "/runs/:run_id", "read"},
		{http.MethodPost, "/runs/:run_id/confirm", "confirm"}, {http.MethodPost, "/runs/:run_id/save", "save"},
		{http.MethodPost, "/runs/:run_id/recover", "recover"}, {http.MethodPost, "/runs/:run_id/resume", "resume"},
		{http.MethodPost, "/runs/:run_id/cancel", "cancel"},
	} {
		a := a
		permission := authz.PermissionImageAgentWrite
		if a.method == http.MethodGet {
			permission = authz.PermissionImageAgentRead
		}
		result = append(result, httproute.Descriptor{Method: a.method, Path: artworkAgentBase + a.path, Module: m.Name(), Permission: permission, AuthPolicy: httproute.AuthPolicyVerifiedIdentity, OrganizationAccessPolicy: httproute.OrganizationAccessPolicyLiveWrite, RequestTimeout: 30 * time.Second, Handler: func(c *gin.Context) { m.handleArtwork(c, a.action) }})
	}
	return result
}

type artworkPrepareBody struct {
	Theme                string                   `json:"theme"`
	Requirements         string                   `json:"requirements"`
	ContextID            string                   `json:"contextId"`
	ExpectedModeRevision string                   `json:"expectedModeRevision"`
	Template             *agentconfig.TemplateRef `json:"template,omitempty"`
}
type artworkCommandBody struct {
	PlanRevision int64  `json:"planRevision"`
	PlanDigest   string `json:"planDigest,omitempty"`
	QuoteDigest  string `json:"quoteDigest,omitempty"`
	ResultDigest string `json:"resultDigest,omitempty"`
	SlotID       string `json:"slotId,omitempty"`
	Attempt      int    `json:"attempt,omitempty"`
}

func validateArtworkHTTPRun(org, actor, member, runID string, p imageagent.RunProjection) error {
	if org == "" || actor == "" || member == "" || p.Run.ID != runID || p.Run.TenantID != org || p.Run.UserID != actor || p.Run.MemberID != member || p.Run.ScopeProtocol != imageagent.OrganizationScopeProtocol || imageagent.ValidateArtworkPlan(p.Plan) != nil {
		return imageagent.ErrRunNotFound
	}
	return nil
}
func (m fullImageModule) handleArtwork(c *gin.Context, action string) {
	a := m.application
	id, ok := authidentity.AuthenticatedIdentityFromContext(c.Request.Context())
	if !ok || id.TenantID != id.EffectiveOrganizationID || id.EffectiveMemberID == "" {
		writeFullImageError(c, imageagent.ErrIdentityRequired, false)
		return
	}
	if a == nil || a.service == nil || a.artwork == nil || m.bind == nil {
		writeFullImageError(c, imageagent.ErrCommandBlocked, false)
		return
	}
	if c.Request.URL.RawQuery != "" {
		writeFullImageError(c, imageagent.ErrValidation, false)
		return
	}
	ctx, err := m.bind(c.Request.Context(), c.GetHeader("Authorization"))
	if err != nil {
		writeFullImageError(c, err, false)
		return
	}
	var p imageagent.RunProjection
	mutation := c.Request.Method != http.MethodGet
	key := c.GetHeader("Idempotency-Key")
	if mutation && !acquisitionHTTPUUID(key) {
		writeFullImageError(c, imageagent.ErrValidation, false)
		return
	}
	if action == "by-key" {
		if !emptyAcquisitionImageBody(c) {
			return
		}
		var prepared imageagent.PreparedArtwork
		prepared, err = a.service.GetPreparedArtworkByKey(ctx, c.Param("context_id"), c.Param("request_id"))
		p = prepared.Projection
	} else if action == "prepare" {
		var body artworkPrepareBody
		err = readFullImageJSON(c.Request, &body)
		if err == nil && !acquisitionHTTPUUID(body.ContextID) {
			err = imageagent.ErrValidation
		}
		if err == nil {
			var prepared imageagent.PreparedArtwork
			prepared, err = a.service.PrepareArtwork(ctx, imageagent.PrepareArtworkInput{RequestID: key, ContextID: body.ContextID, Theme: body.Theme, Requirements: body.Requirements, ExpectedModeRevision: body.ExpectedModeRevision, Template: body.Template})
			p = prepared.Projection
		}
	} else {
		runID := c.Param("run_id")
		if !acquisitionHTTPUUID(runID) {
			writeFullImageError(c, imageagent.ErrValidation, false)
			return
		}
		var prepared imageagent.PreparedArtwork
		prepared, err = a.service.GetPreparedArtwork(ctx, runID)
		p = prepared.Projection
		if err == nil {
			err = validateArtworkHTTPRun(id.TenantID, id.UserID, id.EffectiveMemberID, runID, p)
		}
		if err == nil {
			if !mutation {
				if !emptyAcquisitionImageBody(c) {
					return
				}
			} else {
				var body artworkCommandBody
				err = readFullImageJSON(c.Request, &body)
				if err == nil {
					err = validateArtworkCommandBody(action, body)
				}
				if err == nil && body.PlanRevision != p.Plan.Revision {
					err = imageagent.ErrRevisionConflict
				}
				if err == nil {
					switch action {
					case "confirm":
						if body.ResultDigest != "" || body.SlotID != "" || body.Attempt != 0 {
							err = imageagent.ErrValidation
							break
						}
						p, err = a.service.ConfirmArtwork(ctx, imageagent.ConfirmArtworkInput{RunID: runID, ActionID: key, ExpectedRevision: body.PlanRevision, PlanDigest: body.PlanDigest, QuoteDigest: body.QuoteDigest})
					case "save":
						if body.PlanDigest != "" || body.QuoteDigest != "" || body.SlotID != "" || body.Attempt != 0 {
							err = imageagent.ErrValidation
							break
						}
						err = a.service.AdoptArtwork(ctx, runID, body.PlanRevision, body.ResultDigest, key)
					case "recover":
						err = a.service.RecoverEffect(ctx, runID, body.SlotID, body.Attempt, body.PlanRevision, key)
					case "resume":
						_, err = a.service.Resume(ctx, runID, key)
					case "cancel":
						err = a.service.Cancel(ctx, runID, body.PlanRevision, key)
					default:
						err = imageagent.ErrValidation
					}
				}
				if err == nil && action != "confirm" {
					prepared, err = a.service.GetPreparedArtwork(ctx, runID)
					p = prepared.Projection
				}
			}
		}
	}
	if err != nil {
		writeFullImageError(c, err, mutation)
		return
	}
	response, err := a.artworkResponse(ctx, p)
	if err != nil {
		writeFullImageError(c, err, mutation)
		return
	}
	c.JSON(http.StatusOK, response)
}
func (a *fullImageApplication) artworkResponse(ctx context.Context, p imageagent.RunProjection) (gin.H, error) {
	digest, err := imageagent.ArtworkPlanDigest(p.Plan)
	if err != nil || a.configuration == nil {
		return nil, imageagent.ErrCommandBlocked
	}
	art := p.Plan.Artwork
	snapshot, err := a.configuration.LoadArtworkConfiguration(ctx, agent.Scope{OrganizationID: p.Run.TenantID, ActorID: p.Run.UserID}, art.Configuration)
	if err != nil {
		return nil, err
	}
	if snapshot.RunID != p.Run.ID || snapshot.MemberID != p.Run.MemberID || snapshot.ContextID != p.Run.BusinessTaskID || snapshot.ModeSnapshotRef != art.ModeSnapshotRef {
		return nil, imageagent.ErrRevisionConflict
	}
	frozen, err := a.configuration.LoadModeSnapshot(ctx, agent.ExecutionIdentity{OrganizationID: p.Run.TenantID, ActorID: p.Run.UserID, MemberID: p.Run.MemberID}, art.ModeSnapshotRef)
	if err != nil {
		return nil, err
	}
	mode, modeErr := a.configuration.CheckArtworkExecutionMode(ctx, agent.ExecutionIdentity{OrganizationID: p.Run.TenantID, ActorID: p.Run.UserID, MemberID: p.Run.MemberID}, art.Configuration, art.ModeSnapshotRef)
	allowed := modeErr == nil && mode.Mode != agent.ModeReadOnly
	candidates := []gin.H{}
	settled := int64(0)
	for _, slot := range p.Slots {
		if slot.Closure != nil {
			settled += slot.Closure.Points
		}
		for i, candidate := range slot.Candidates {
			if candidate.Artwork == nil {
				return nil, imageagent.ErrRevisionConflict
			}
			url, e := acquisitionImagePublishedURL(p, slot.Slot, slot.Attempt, candidate, i, a.publicURLs, a.trial)
			if e != nil {
				return nil, e
			}
			candidates = append(candidates, gin.H{"assetId": candidate.AssetID, "url": url, "width": candidate.Width, "height": candidate.Height, "hash": candidate.DurableAsset.SHA256})
		}
	}
	result, closedErr := imageagent.ArtworkResultDigest(p.Plan, p.Slots, p.RecoverableEffects)
	saved := any(nil)
	if p.Run.Status == imageagent.RunStatusCompleted {
		reader, ok := a.artwork.(imageagent.ArtworkReceiptReader)
		if !ok {
			return nil, imageagent.ErrCommandBlocked
		}
		receipt, e := reader.ReadArtworkReceipt(ctx, imageagent.ExecutionIdentity{ScopeProtocol: p.Run.ScopeProtocol, TenantID: p.Run.TenantID, UserID: p.Run.UserID, MemberID: p.Run.MemberID, RunID: p.Run.ID, BusinessTaskID: p.Run.BusinessTaskID})
		if e != nil {
			return nil, e
		}
		saved = gin.H{"operationId": receipt.OperationID, "batchId": receipt.BatchID, "itemId": receipt.ItemID, "productKey": receipt.ProductKey, "publicationId": receipt.PublicationID, "sourceVersion": strconv.FormatUint(receipt.SourceVersion, 10), "artifactHash": receipt.ArtifactHash}
	}
	var block, pending any
	if p.Run.Block != nil {
		block = gin.H{"Code": p.Run.Block.Code, "SlotID": p.Run.Block.SlotID}
	}
	if p.PendingCommand != nil {
		pending = gin.H{"ActionID": p.PendingCommand.ActionID, "Kind": p.PendingCommand.Kind, "Phase": p.PendingCommand.Phase, "PlanRevision": p.PendingCommand.PlanRevision}
	}
	confirmation := ""
	if p.Run.ImageAdmission != nil {
		confirmation = p.Run.ImageAdmission.Command.ConfirmActionID
	}
	return gin.H{"runId": p.Run.ID, "requestKey": p.Run.IdempotencyKey, "contextId": p.Run.BusinessTaskID, "theme": art.Theme, "requirements": art.Requirements, "status": p.Run.Status, "planRevision": p.Plan.Revision, "planDigest": digest, "quoteDigest": art.QuoteDigest, "points": art.MaxPoints, "settledPoints": settled, "candidates": candidates, "resultDigest": p.ResultDigest, "confirmationActionId": confirmation, "generationAdmitted": p.Run.ImageAdmission != nil, "generationAllowed": allowed, "saveAvailable": allowed && p.PendingCommand == nil && closedErr == nil && result == p.ResultDigest && p.Run.Status == imageagent.RunStatusAwaitingFinalApproval, "saved": saved, "mode": frozen.Mode, "modeRevision": frozen.Revision, "template": snapshot.Template, "block": block, "pendingCommand": pending, "recoverableEffects": p.RecoverableEffects}, nil
}

func validateArtworkCommandBody(action string, b artworkCommandBody) error {
	if b.PlanRevision <= 0 {
		return imageagent.ErrValidation
	}
	switch action {
	case "confirm":
		if !agentconfig.ImageDigest(b.PlanDigest) || !agentconfig.ImageDigest(b.QuoteDigest) || b.ResultDigest != "" || b.SlotID != "" || b.Attempt != 0 {
			return imageagent.ErrValidation
		}
	case "save":
		if !agentconfig.ImageDigest(b.ResultDigest) || b.PlanDigest != "" || b.QuoteDigest != "" || b.SlotID != "" || b.Attempt != 0 {
			return imageagent.ErrValidation
		}
	case "recover":
		if b.SlotID != "artwork" || b.Attempt <= 0 || b.ResultDigest != "" || b.PlanDigest != "" || b.QuoteDigest != "" {
			return imageagent.ErrValidation
		}
	case "resume", "cancel":
		if b.SlotID != "" || b.Attempt != 0 || b.ResultDigest != "" || b.PlanDigest != "" || b.QuoteDigest != "" {
			return imageagent.ErrValidation
		}
	default:
		return imageagent.ErrValidation
	}
	return nil
}
