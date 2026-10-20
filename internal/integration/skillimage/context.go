// Package skillimage connects the current Skill owner to the existing image
// consumer. It owns no identities, plans, provider effects or saved material.
package skillimage

import (
	"context"
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

func (c *Context) AcquireImageMethod(ctx context.Context, id imageagent.ExecutionIdentity, ref agent.SkillVersionRef, policy string) (string, func(), error) {
	if !(imageagent.ImageMethodBinding{Skill: ref, PolicyVersion: policy}).Valid() {
		return "", nil, imageagent.ErrCommandBlocked
	}
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
	if policy == imageagent.HistoricalImageMethodPolicyVersion {
		return "", release, nil
	}
	brief, err := d.ImageVisualBrief(canonical)
	if err != nil {
		release()
		return "", nil, err
	}
	return brief, release, nil
}
