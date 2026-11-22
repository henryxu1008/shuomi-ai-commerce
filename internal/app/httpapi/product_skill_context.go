package httpapi

import (
	"context"
	"errors"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/agent"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/agentconfig"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/product/contentagent"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/product/review"
	d "github.com/henryxu1008/shuomi-ai-commerce/internal/skill"
	"gorm.io/gorm"
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
	if ref.PolicyVersion == d.ScriptPolicyVersion {
		if err := c.application.checkScriptTrialIdentity(identity); err != nil {
			return nil, err
		}
		// Recheck exact live use before probing infrastructure, without holding
		// a Skill transaction across the network read. AcquireUse below still
		// closes the revocation race at the original before-send fence.
		if _, e = c.application.skills.ReadForUse(ctx, p, ref); e != nil {
			return nil, e
		}
		if c.application.script == nil {
			return nil, agent.ErrUnavailable
		}
		if e = c.application.script.CheckControl(ctx); e != nil {
			return nil, e
		}
	}
	_, release, e := c.application.skills.AcquireUse(ctx, p, ref)
	return release, e
}
func (a *productAgentApplication) checkSkillClaim(ctx context.Context, tx *gorm.DB, identity agent.ExecutionIdentity, ref agent.SkillVersionRef) error {
	if ref.PolicyVersion == d.ScriptPolicyVersion {
		if err := a.checkScriptTrialIdentity(identity); err != nil {
			return err
		}
	}
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
func ownTitleScriptReview(p review.ExecutionProvenance) bool {
	return p.Binding.ContextKind == contentagent.OwnProductContext && p.Binding.TargetPlatform == "product" &&
		p.SkillVersionRef.PolicyVersion == d.ScriptPolicyVersion && p.ContentSelection.TitleOnly()
}

func (a *productAgentApplication) checkTitleReviewMode(ctx context.Context, origin agent.ExecutionIdentity, p review.ExecutionProvenance, human bool) (agentconfig.ModeSnapshot, error) {
	if human && ownTitleScriptReview(p) {
		return a.configuration.CheckCompletedTitleScriptReviewMode(ctx, origin, p.Configuration, p.Mode)
	}
	return a.configuration.CheckRegisteredTextExecutionMode(ctx, origin, p.Configuration, p.Mode)
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
	if snapshot.AgentID != agentconfig.ContentAgentID || snapshot.Request.SkillVersionRef != p.SkillVersionRef || snapshot.Request.ScriptDescriptor != p.ScriptDescriptor || snapshot.Request.ContentSelection != p.ContentSelection || !p.ContentSelection.TitleOnly() {
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
	mode, e := a.checkTitleReviewMode(ctx, origin, p, true)
	if e != nil {
		return nil, e
	}
	if p.SkillVersionRef.PolicyVersion == d.ScriptPolicyVersion && mode.Mode != agent.ModeDefault {
		return nil, review.ErrForbidden
	}
	consumer := a
	if a.definition.ID != agentconfig.ContentAgentID {
		consumer = a.content
	}
	if consumer == nil || consumer.skills == nil {
		return nil, d.ErrUnavailable
	}
	if p.SkillVersionRef.PolicyVersion == d.ScriptPolicyVersion && (consumer.script == nil || consumer.scriptDescriptor != p.ScriptDescriptor) {
		return nil, review.ErrConflict
	}
	// The reviewer never receives the original user's method body. Exact live
	// origin access is required; the existing reviewer checks still run in UoW.
	principal := d.Principal{UserID: origin.ActorID, OrganizationID: origin.OrganizationID, MemberID: origin.MemberID, UseOrganization: true}
	acquire := consumer.skills.AcquireUse
	if ownTitleScriptReview(p) {
		acquire = consumer.skills.AcquireCompletedTitleScriptReview
	}
	_, release, e := acquire(ctx, principal, p.SkillVersionRef)
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
