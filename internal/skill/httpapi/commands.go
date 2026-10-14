package skillhttp

import (
	"context"
	"github.com/gin-gonic/gin"
	"strconv"
	"task-processor/internal/agent"
	"task-processor/internal/agentconfig"
	d "task-processor/internal/skill"
)

func (h *Handler) command(c *gin.Context, s routeSpec) {
	var input struct {
		Files       []d.File `json:"files,omitempty"`
		AllowCopy   bool     `json:"allowCopy"`
		AllowExport bool     `json:"allowExport"`
	}
	command := d.Command{Operation: s.operation, ID: c.Param("skill_id")}
	var e error
	if s.operation == "copy" {
		var input struct {
			Source      agent.SkillVersionRef `json:"source"`
			AllowCopy   bool                  `json:"allowCopy"`
			AllowExport bool                  `json:"allowExport"`
		}
		if e = body(c, &input); e != nil {
			Failure(c, e)
			return
		}
		command.Source, command.AllowCopy, command.AllowExport = &input.Source, input.AllowCopy, input.AllowExport
	} else if s.operation == "lifecycle" {
		var input struct{}
		if e = body(c, &input); e != nil {
			Failure(c, e)
			return
		}
		command.Operation = c.Param("action")
		switch command.Operation {
		case "publish", "enable", "disable", "revoke":
		default:
			Failure(c, d.ErrInvalid)
			return
		}
		command.Version, e = number(c.Param("version"), false)
		if e != nil {
			Failure(c, e)
			return
		}
	} else {
		if e = body(c, &input); e != nil {
			Failure(c, e)
			return
		}
		pack, e := d.ValidatePackage(input.Files)
		if e != nil {
			Failure(c, e)
			return
		}
		command.Package, command.AllowCopy, command.AllowExport = pack, input.AllowCopy, input.AllowExport
	}
	if s.operation == "validate" {
		if _, e := h.Service.Principal(c.Request.Context()); e != nil {
			Failure(c, e)
			return
		}
		reply(c, gin.H{"valid": true, "name": command.Package.Name, "description": command.Package.Description, "digest": command.Package.Digest, "fileCount": len(command.Package.Files)}, 0)
		return
	}
	command.Key, e = key(c)
	if e != nil {
		Failure(c, e)
		return
	}
	if s.operation == "save-version" || s.operation == "lifecycle" {
		command.Expected, e = expected(c, false)
		if e != nil {
			Failure(c, e)
			return
		}
	}
	receipt, e := h.Service.Execute(c.Request.Context(), s.scope, command)
	if e != nil {
		Failure(c, e)
		return
	}
	reply(c, receipt, receipt.Revision)
}
func (h *Handler) share(c *gin.Context, s routeSpec) {
	ctx := c.Request.Context()
	if s.operation == "shares" {
		items, e := h.Service.Shares(ctx, s.scope == "organization")
		if e != nil {
			Failure(c, e)
			return
		}
		reply(c, gin.H{"items": items}, 0)
		return
	}
	if s.operation == "read-shared" {
		v, e := h.Service.ReadShared(ctx, c.Param("share_id"))
		if e != nil {
			Failure(c, e)
			return
		}
		v.Metadata.HeadVersion = v.Number
		v.Metadata.Revision = 0
		reply(c, v, 0)
		return
	}
	k, e := key(c)
	if e != nil {
		Failure(c, e)
		return
	}
	if s.operation == "share" {
		var input struct {
			SkillID string `json:"skillId"`
			Version uint64 `json:"version"`
			Digest  string `json:"digest"`
			Consent bool   `json:"consent"`
		}
		if e = body(c, &input); e != nil {
			Failure(c, e)
			return
		}
		value, e := h.Service.Share(ctx, k, input.SkillID, input.Version, input.Digest, input.Consent)
		if e != nil {
			Failure(c, e)
			return
		}
		reply(c, value, 0)
		return
	}
	var input struct{}
	if e = body(c, &input); e != nil {
		Failure(c, e)
		return
	}
	action := c.Param("action")
	if s.operation == "withdraw" {
		action = "withdraw"
	}
	if s.operation == "decide-share" && action != "approve" && action != "reject" {
		Failure(c, d.ErrInvalid)
		return
	}
	value, e := h.Service.DecideShare(ctx, k, c.Param("share_id"), action)
	if e != nil {
		Failure(c, e)
		return
	}
	reply(c, value, 0)
}
func (h *Handler) admission(c *gin.Context, s routeSpec) {
	if h.Policy == nil {
		Failure(c, d.ErrUnavailable)
		return
	}
	ctx := c.Request.Context()
	p, e := h.Service.Principal(ctx)
	if e != nil {
		Failure(c, e)
		return
	}
	if !p.ReadOrganization && !p.UseOrganization && !p.ConfigureOrganization {
		Failure(c, d.ErrForbidden)
		return
	}
	scope := agent.Scope{OrganizationID: p.OrganizationID, ActorID: p.UserID}
	if s.operation == "read-admission" {
		v, e := h.Policy.ReadSkillAdmission(ctx, scope)
		if e != nil {
			Failure(c, e)
			return
		}
		reply(c, gin.H{"policy": v, "canConfigure": p.ConfigureOrganization}, 0)
		return
	}
	if !p.ConfigureOrganization {
		Failure(c, d.ErrForbidden)
		return
	}
	var input struct {
		AllowPersonal bool `json:"allowPersonal"`
	}
	if e = body(c, &input); e != nil {
		Failure(c, e)
		return
	}
	k, e := key(c)
	if e != nil {
		Failure(c, e)
		return
	}
	n, e := expected(c, true)
	if e != nil {
		Failure(c, e)
		return
	}
	value, e := h.Policy.SetSkillAdmission(ctx, agentconfig.SkillAdmissionCommand{Scope: scope, Key: k, Expected: n, AllowPersonal: input.AllowPersonal}, func(ctx context.Context) error {
		fresh, e := h.Service.Principal(ctx)
		if e != nil {
			return e
		}
		if fresh.UserID != p.UserID || fresh.OrganizationID != p.OrganizationID || fresh.MemberID != p.MemberID || !fresh.ConfigureOrganization {
			return d.ErrForbidden
		}
		return nil
	})
	if e != nil {
		Failure(c, e)
		return
	}
	revision, _ := strconv.ParseUint(value.Revision, 10, 63)
	reply(c, gin.H{"policy": value, "canConfigure": true}, revision)
}
