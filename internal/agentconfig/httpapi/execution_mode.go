package httpapi

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"task-processor/internal/agent"
	"task-processor/internal/agentconfig"
	"task-processor/internal/authz"
	"task-processor/internal/httproute"
)

func modeRoutes(h *Handler) []httproute.Descriptor {
	out := make([]httproute.Descriptor, 0, 2)
	for _, method := range []string{http.MethodGet, http.MethodPut} {
		permission := ""
		if method == http.MethodPut {
			permission = authz.PermissionWorkbenchAgentConfigure
		}
		out = append(out, httproute.Descriptor{Method: method, Path: Base + "/execution-modes/:subject_kind/:subject_id", Module: Module, AuthPolicy: httproute.AuthPolicyCurrentIdentity, OrganizationAccessPolicy: httproute.OrganizationAccessPolicyLiveWrite, Permission: permission, RequestTimeout: 10 * time.Second, RejectUnreadRequestBody: method == http.MethodGet, Handler: h.serveMode})
	}
	return out
}

func modeProjection(subject agentconfig.ResolvedModeSubject, record agentconfig.ExecutionModeRecord, configure bool) gin.H {
	return gin.H{"subject": subject.Subject, "definition": gin.H{"id": subject.Definition.ID, "version": subject.Definition.Version}, "mode": record.Mode, "revision": record.Revision, "policyVersion": agent.ExecutionModePolicyVersion, "canConfigure": configure, "updatedAt": record.UpdatedAt, "updatedBy": record.UpdatedBy}
}

func (h *Handler) serveMode(c *gin.Context) {
	if h == nil || h.ModeRepository == nil || h.ResolveModeSubject == nil {
		Failure(c, agentconfig.ErrUnavailable)
		return
	}
	ctx := c.Request.Context()
	if h.Bind != nil {
		bound, err := h.Bind(ctx, c.GetHeader("Authorization"))
		if err != nil {
			Failure(c, agentconfig.ErrForbidden)
			return
		}
		ctx = bound
		c.Request = c.Request.WithContext(ctx)
	}
	if c.Request.URL.RawQuery != "" {
		Failure(c, agentconfig.ErrInvalid)
		return
	}
	subject := agentconfig.ModeSubject{Kind: agentconfig.ModeSubjectKind(c.Param("subject_kind")), ID: c.Param("subject_id")}
	if !subject.Valid() {
		Failure(c, agentconfig.ErrInvalid)
		return
	}
	write := c.Request.Method == http.MethodPut
	identity, resolved, err := h.ResolveModeSubject(ctx, subject, write)
	if err != nil {
		Failure(c, err)
		return
	}
	if !identity.Valid() || !resolved.Valid() || resolved.Subject != subject {
		Failure(c, agentconfig.ErrForbidden)
		return
	}
	if !write {
		record, err := h.ModeRepository.ReadExecutionMode(ctx, identity, resolved)
		if err != nil {
			Failure(c, err)
			return
		}
		configure := false
		if h.Authorize != nil {
			scope, err := h.Authorize(ctx, authz.PermissionWorkbenchAgentConfigure)
			configure = err == nil && scope.OrganizationID == identity.OrganizationID && scope.ActorID == identity.ActorID
		}
		reply(c, modeProjection(resolved, record, configure), record.Revision)
		return
	}
	var input struct {
		Mode             agent.ExecutionMode `json:"mode"`
		ExpectedRevision *string             `json:"expectedRevision"`
	}
	if err := body(c, &input); err != nil || !input.Mode.Valid() {
		Failure(c, agentconfig.ErrInvalid)
		return
	}
	if input.ExpectedRevision == nil {
		Failure(c, agentconfig.ErrPrecondition)
		return
	}
	revision, err := agentconfig.ParseModeRevision(*input.ExpectedRevision)
	if err != nil {
		Failure(c, err)
		return
	}
	keys := c.Request.Header.Values("Idempotency-Key")
	if len(keys) != 1 || !agentconfig.UUID(keys[0]) {
		Failure(c, agentconfig.ErrInvalid)
		return
	}
	command := agentconfig.ModeCommand{Identity: identity, Subject: resolved, Key: keys[0], ExpectedRevision: revision, Mode: input.Mode}
	eligibility := func(ctx context.Context) error {
		fresh, current, err := h.ResolveModeSubject(ctx, subject, true)
		if err != nil {
			return err
		}
		if fresh != identity {
			return agentconfig.ErrForbidden
		}
		if current != resolved {
			return agentconfig.ErrChanged
		}
		return nil
	}
	receipt, err := h.ModeRepository.ExecuteMode(ctx, command, eligibility)
	if err != nil {
		Failure(c, err)
		return
	}
	projection := modeProjection(resolved, agentconfig.ExecutionModeRecord{Subject: subject, Mode: receipt.Mode, Revision: receipt.Revision, UpdatedBy: identity.ActorID, UpdatedAt: receipt.CommittedAt}, true)
	projection["receipt"] = receipt
	reply(c, projection, receipt.Revision)
}
