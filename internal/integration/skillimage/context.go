// Package skillimage connects the current Skill owner to the existing image
// consumer. It owns no identities, plans, provider effects or saved material.
package skillimage

import (
	"context"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/agent"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/imageagent"
	d "github.com/henryxu1008/shuomi-ai-commerce/internal/skill"
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
	brief, err := d.ImageVisualBrief(canonical)
	if err != nil {
		release()
		return "", nil, err
	}
	return brief, release, nil
}

func (c *Context) AcquireImageMethods(ctx context.Context, uses []imageagent.ImageMethodUse) (func(), error) {
	store, ok := c.store.(interface {
		AcquireImageUses(context.Context, d.Principal, []agent.SkillVersionRef) (func(), error)
	})
	if !ok || len(uses) == 0 || len(uses) > 40 {
		return nil, imageagent.ErrCommandBlocked
	}
	origin := uses[0].Identity
	principal := d.Principal{UserID: origin.UserID, OrganizationID: origin.TenantID, MemberID: origin.MemberID, UseOrganization: true}
	seen := make(map[agent.SkillVersionRef]bool, len(uses))
	authorized := make(map[imageagent.ExecutionIdentity]bool, len(uses))
	var refs []agent.SkillVersionRef
	for _, use := range uses {
		id := use.Identity
		if !use.Binding.Valid() || imageagent.ValidateOrganizationExecution(id, id.RunID) != nil ||
			id.MemberID == "" || id.TenantID != origin.TenantID || id.UserID != origin.UserID || id.MemberID != origin.MemberID {
			return nil, imageagent.ErrIdentityRequired
		}
		if !authorized[id] {
			if err := c.authorizer.AuthorizeExecution(ctx, id); err != nil {
				return nil, err
			}
			authorized[id] = true
		}
		ref := use.Binding.Skill
		if !seen[ref] {
			refs = append(refs, ref)
			seen[ref] = true
		}
	}
	return store.AcquireImageUses(ctx, principal, refs)
}
