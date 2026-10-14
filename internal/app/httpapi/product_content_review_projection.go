package httpapi

import (
	"context"
	"task-processor/internal/agent"
	"task-processor/internal/agentconfig"
	"task-processor/internal/product/review"
)

const productReviewModeProjectorKey = "product-review-frozen-mode-projector"

type productReviewModeProjector func(context.Context, *review.ExecutionProvenance) *productReviewModeDTO
type productReviewModeDTO struct {
	Mode     agent.ExecutionMode   `json:"mode"`
	Revision string                `json:"revision"`
	Snapshot agent.ModeSnapshotRef `json:"snapshot"`
}

// Display the original immutable mode, never the current head or an inferred
// title grant. This read projection is not a permission to perform a new effect.
func (a *productAgentApplication) projectContentReviewMode(ctx context.Context, p *review.ExecutionProvenance) *productReviewModeDTO {
	if a == nil || a.configuration == nil || a.store == nil || p == nil || !p.Valid() || !p.ContentSelection.Valid() {
		return nil
	}
	run, found, err := a.store.Lookup(ctx, p.Scope, p.Binding, p.RequestKey)
	if err != nil || !found || *agentExecutionProvenance(run.State) != *p {
		return nil
	}
	config, err := a.configuration.LoadSnapshot(ctx, p.Scope, p.Configuration)
	if err != nil || config.AgentID != agentconfig.ContentAgentID || config.AgentVersion != agentconfig.ContentAgentVersion || config.Request.ContentSelection != p.ContentSelection {
		return nil
	}
	mode, err := a.configuration.LoadModeSnapshot(ctx, agent.ExecutionIdentity{OrganizationID: p.Scope.OrganizationID, ActorID: p.Scope.ActorID, MemberID: config.MemberID}, p.Mode)
	if err != nil || mode.Subject != a.contentModeSubject() || mode.Binding.ContextKind != p.Binding.ContextKind || mode.Binding.ContextID != p.Binding.ContextID || mode.Binding.RequestKey != p.RequestKey {
		return nil
	}
	return &productReviewModeDTO{Mode: mode.Mode, Revision: mode.Revision, Snapshot: mode.Ref()}
}

func (a *productAgentApplication) contentModeSubject() agentconfig.ResolvedModeSubject {
	if a.definition.ID == agentconfig.ContentAgentID {
		return a.modeSubject()
	}
	if a.content != nil {
		return a.content.modeSubject()
	}
	return agentconfig.ResolvedModeSubject{}
}
