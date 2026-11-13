package httpapi

import (
	"context"
	"encoding/json"
	"strconv"
	"unicode/utf8"

	"task-processor/internal/agent"
	"task-processor/internal/aiworkbench"
	"task-processor/internal/authidentity"
	"task-processor/internal/integration/aiworkbench/einoplanner"
	"task-processor/internal/product/catalog"
)

func (p *workbenchPlanner) authorizeProductHistory(ctx context.Context, scope aiworkbench.Scope, conversation string, through uint64) error {
	if p == nil || p.workbench == nil || p.agent == nil {
		return aiworkbench.ErrUnavailable
	}
	i, err := p.agent.freshChatIdentity(ctx)
	if err != nil || i.TenantID != scope.OrganizationID || i.UserID != scope.ActorID {
		return aiworkbench.ErrUnavailable
	}
	return p.workbench.authorizeProductHistory(ctx, scope, conversation, through, i)
}

func (a *aiWorkbenchApplication) taskHistoryUnavailable(ctx context.Context, scope aiworkbench.Scope, task aiworkbench.BusinessTask, i authidentity.AuthenticatedIdentity) bool {
	if task.Kind != aiworkbench.ContentTaskKind {
		return false
	}
	var p aiworkbench.ExecutionProposal
	if task.ScheduledOccurrenceID == "" {
		if task.ProposalID == "" || a.store == nil {
			return true
		}
		var err error
		p, err = a.store.GetProposal(ctx, scope, task.ProposalID)
		if err != nil || p.Digest != task.ProposalDigest || p.ConversationID != task.ConversationID {
			return true
		}
	} else {
		var request agent.Request
		if json.Unmarshal(task.ExecutionRequest, &request) != nil || !request.ScheduledSource.Valid() {
			return true
		}
		plan, err := a.store.ReadScheduledPlan(ctx, scope, request.ScheduledSource.PlanID)
		if err != nil || json.Unmarshal(plan.ExecutionSpec, &p) != nil || p.GoalSummary != task.GoalSummary {
			return true
		}
	}
	return a.proposalHistoryUnavailable(ctx, scope, p, i)
}

func (a *aiWorkbenchApplication) authorizeProductHistory(ctx context.Context, scope aiworkbench.Scope, conversation string, through uint64, i authidentity.AuthenticatedIdentity) error {
	if a == nil || a.store == nil || i.TenantID != scope.OrganizationID || i.UserID != scope.ActorID {
		return aiworkbench.ErrUnavailable
	}
	selections, err := a.store.ProductDependencies(ctx, scope, conversation, through)
	if err != nil {
		return err
	}
	if len(selections) != 0 && (a.agent == nil || !a.agent.workbenchKind(aiworkbench.ContentTaskKind)) {
		return aiworkbench.ErrUnavailable
	}
	for _, selection := range selections {
		if _, err := a.agent.workbenchProductSnapshot(ctx, i, selection); err != nil {
			return err
		}
	}
	return nil
}

// In metadata-only mode there is no product owner from which to obtain live
// authorization. Such a mode must not expose product-derived replies.
func (a *aiWorkbenchApplication) productHistoryUnavailable(ctx context.Context, scope aiworkbench.Scope, conversation string, through uint64) bool {
	if a == nil || a.store == nil {
		return true
	}
	i, ok := authidentity.AuthenticatedIdentityFromContext(ctx)
	return !ok || a.authorizeProductHistory(ctx, scope, conversation, through, i) != nil
}

// A directly authored schedule has no originating Chat message. A persisted
// proposal must retain an exact scoped assistant-message association.
func (a *aiWorkbenchApplication) proposalHistoryUnavailable(ctx context.Context, scope aiworkbench.Scope, p aiworkbench.ExecutionProposal, i authidentity.AuthenticatedIdentity) bool {
	if p.Scope != scope {
		return true
	}
	if p.ID == "" && p.ConversationID == "" && p.AssistantMessageID == "" && p.WorkbenchContextRef.Absent() {
		return false
	}
	if a == nil || a.store == nil || p.ConversationID == "" || p.AssistantMessageID == "" {
		return true
	}
	seq, err := a.store.AssistantSequence(ctx, scope, p.ConversationID, p.AssistantMessageID)
	return err != nil || a.authorizeProductHistory(ctx, scope, p.ConversationID, seq, i) != nil
}

func projectWorkbenchProduct(p catalog.ProductSnapshot) *einoplanner.ProductContext {
	result := &einoplanner.ProductContext{}
	cut := func(s string, n int) string {
		if len(s) <= n {
			return s
		}
		result.Truncated = true
		s = s[:n]
		for !utf8.ValidString(s) && len(s) > 0 {
			s = s[:len(s)-1]
		}
		return s
	}
	result.Title = cut(p.Title, 512)
	result.Description = cut(p.PlainDescription(), 4096)
	for i, point := range p.SellingPoints {
		if i == 5 {
			result.Truncated = true
			break
		}
		result.SellingPoints = append(result.SellingPoints, cut(point, 512))
	}
	return result
}

// workbenchProductSnapshot resolves the exact historical selection through the
// original owner's authorization. It never substitutes the current version.
func (a *productAgentApplication) workbenchProductSnapshot(ctx context.Context, i authidentity.AuthenticatedIdentity, s *aiworkbench.TaskSelection) (catalog.ProductSnapshot, error) {
	if s == nil || !s.Valid() {
		return catalog.ProductSnapshot{}, aiworkbench.ErrInvalid
	}
	if s.Subject.Kind == "own_product" {
		source, err := a.ownSource(ctx, i, s.Subject.ID, &ownSourceSelection{CatalogVersion: s.Subject.CatalogVersion, PublicationID: s.Subject.PublicationID})
		if err != nil {
			return catalog.ProductSnapshot{}, err
		}
		return source.Effective.Snapshot, nil
	}
	original, err := a.acquisitionSnapshot(ctx, i, s.Subject.ID)
	if err != nil || a.snapshots == nil {
		return catalog.ProductSnapshot{}, aiworkbench.ErrUnavailable
	}
	v, err := strconv.ParseUint(s.Subject.CatalogVersion, 10, 63)
	if err != nil {
		return catalog.ProductSnapshot{}, aiworkbench.ErrInvalid
	}
	published, err := a.snapshots.GetSnapshot(ctx, original.Identity, v)
	if err != nil || published.PublicationID != s.Subject.PublicationID || published.Version != v || published.Identity != original.Identity {
		return catalog.ProductSnapshot{}, aiworkbench.ErrProposalStale
	}
	b := acquisitionBinding(s.Subject.ID, "product", published)
	if err = a.authorizeAcquisitionBinding(ctx, i, b); err != nil {
		return catalog.ProductSnapshot{}, err
	}
	return published.Snapshot, nil
}
