package httpapi

import (
	"context"
	"errors"
	"gorm.io/gorm"
	"task-processor/internal/agent"
	"task-processor/internal/agentconfig"
	"task-processor/internal/product/review"
	d "task-processor/internal/skill"
)

type productSkillContext struct{ application *productAgentApplication }

func (a *productAgentApplication) skillPrincipal(ctx context.Context, expected agent.ExecutionIdentity) (d.Principal, error) {
	identity, e := a.currentModeIdentity(ctx)
	if e != nil || identity != expected {
		return d.Principal{}, d.ErrForbidden
	}
	return d.Principal{UserID: identity.ActorID, OrganizationID: identity.OrganizationID, MemberID: identity.MemberID, UseOrganization: true}, nil
}
func (c productSkillContext) ReadMethod(ctx context.Context, identity agent.ExecutionIdentity, ref agent.SkillVersionRef) (d.Package, error) {
	p, e := c.application.skillPrincipal(ctx, identity)
	if e != nil {
		return d.Package{}, e
	}
	return c.application.skills.ReadForUse(ctx, p, ref)
}
func (c productSkillContext) AcquireMethod(ctx context.Context, identity agent.ExecutionIdentity, ref agent.SkillVersionRef) (func(), error) {
	p, e := c.application.skillPrincipal(ctx, identity)
	if e != nil {
		return nil, e
	}
	_, release, e := c.application.skills.AcquireUse(ctx, p, ref)
	return release, e
}
func (a *productAgentApplication) checkSkillClaim(ctx context.Context, tx *gorm.DB, identity agent.ExecutionIdentity, ref agent.SkillVersionRef) error {
	if a.skills == nil || !a.skills.UsesPool(a.config.RunDB) {
		return d.ErrUnavailable
	}
	p, e := a.skillPrincipal(ctx, identity)
	if e != nil {
		return e
	}
	_, e = a.skills.CheckUseInTransaction(ctx, tx, p, ref)
	return e
}
func (a *productAgentApplication) acquireSkillReview(ctx context.Context, p review.ExecutionProvenance) (func(), error) {
	if !p.Valid() || p.SkillVersionRef.Absent() {
		return nil, review.ErrInvalid
	}
	if a.configuration == nil || a.store == nil {
		return nil, d.ErrUnavailable
	}
	snapshot, e := a.configuration.LoadSnapshot(ctx, p.Scope, p.Configuration)
	if e != nil {
		return nil, e
	}
	if snapshot.AgentID != agentconfig.ContentAgentID || snapshot.Request.SkillVersionRef != p.SkillVersionRef || snapshot.Request.ContentSelection != p.ContentSelection || !p.ContentSelection.TitleOnly() {
		return nil, review.ErrConflict
	}
	current, e := a.currentModeIdentity(ctx)
	if e != nil || current.OrganizationID != p.Scope.OrganizationID {
		return nil, review.ErrForbidden
	}
	origin := agent.ExecutionIdentity{OrganizationID: p.Scope.OrganizationID, ActorID: p.Scope.ActorID, MemberID: snapshot.MemberID}
	if current.ActorID == origin.ActorID {
		if current != origin {
			return nil, review.ErrForbidden
		}
	} else {
		if a.originExecution == nil {
			return nil, review.ErrUnavailable
		}
		if e = a.originExecution(ctx, origin); e != nil {
			return nil, e
		}
	}
	run, found, e := a.store.Lookup(ctx, p.Scope, p.Binding, p.RequestKey)
	if e != nil {
		return nil, e
	}
	if !found || run.State.RunID != p.RunID || !agentRunReviewable(run.State) || *agentExecutionProvenance(run.State) != p {
		return nil, review.ErrConflict
	}
	if _, e = a.configuration.CheckRegisteredTextExecutionMode(ctx, origin, p.Configuration, p.Mode); e != nil {
		return nil, e
	}
	consumer := a
	if a.definition.ID != agentconfig.ContentAgentID {
		consumer = a.content
	}
	if consumer == nil || consumer.skills == nil {
		return nil, d.ErrUnavailable
	}
	// The reviewer never receives the original user's method body. Exact live
	// origin access is required; the existing reviewer checks still run in UoW.
	principal := d.Principal{UserID: origin.ActorID, OrganizationID: origin.OrganizationID, MemberID: origin.MemberID, UseOrganization: true}
	_, release, e := consumer.skills.AcquireUse(ctx, principal, p.SkillVersionRef)
	if errors.Is(e, d.ErrForbidden) {
		e = review.ErrForbidden
	}
	if errors.Is(e, d.ErrNotEnabled) || errors.Is(e, d.ErrNotFound) || errors.Is(e, d.ErrConflict) || errors.Is(e, d.ErrRevision) {
		e = review.ErrConflict
	}
	if errors.Is(e, d.ErrUnavailable) {
		e = review.ErrUnavailable
	}
	return release, e
}
