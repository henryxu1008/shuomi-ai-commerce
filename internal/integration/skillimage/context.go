// Package skillimage connects the current Skill owner to the existing image
// consumer. It owns no identities, plans, provider effects or saved material.
package skillimage

import (
	"context"
	"strings"
	"task-processor/internal/agent"
	"task-processor/internal/imageagent"
	d "task-processor/internal/skill"
)

type UseStore interface {
	AcquireUseForTarget(context.Context, d.Principal, agent.SkillVersionRef, string) (d.Package, func(), error)
}

type Context struct {
	store      UseStore
	authorizer imageagent.ExecutionAuthorizer
}

func New(store UseStore, authorizer imageagent.ExecutionAuthorizer) (*Context, error) {
	if store == nil || authorizer == nil {
		return nil, imageagent.ErrCommandBlocked
	}
	return &Context{store: store, authorizer: authorizer}, nil
}

func (c *Context) AcquireImageMethod(ctx context.Context, id imageagent.ExecutionIdentity, ref agent.SkillVersionRef) (string, func(), error) {
	if imageagent.ValidateOrganizationExecution(id, id.RunID) != nil || id.MemberID == "" {
		return "", nil, imageagent.ErrIdentityRequired
	}
	if err := c.authorizer.AuthorizeExecution(ctx, id); err != nil {
		return "", nil, err
	}
	p := d.Principal{UserID: id.UserID, OrganizationID: id.TenantID, MemberID: id.MemberID, UseOrganization: true}
	material, release, err := c.store.AcquireUseForTarget(ctx, p, ref, d.ImageTarget)
	if err != nil {
		return "", nil, err
	}
	if release == nil {
		return "", nil, imageagent.ErrCommandBlocked
	}
	canonical, err := d.ValidatePackage(material.Files)
	if err != nil || canonical.Digest != ref.Digest || canonical.Digest != material.Digest || !canonical.Supports(d.ImageTarget) {
		release()
		return "", nil, d.ErrConflict
	}
	var text strings.Builder
	for _, file := range canonical.Files {
		text.WriteString("Method file: " + file.Path + "\n")
		text.WriteString(file.Text)
		text.WriteString("\n")
	}
	return text.String(), release, nil
}
