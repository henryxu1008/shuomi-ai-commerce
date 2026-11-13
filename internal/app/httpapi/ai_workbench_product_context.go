package httpapi

import (
	"context"
	"strconv"
	"unicode/utf8"

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
	selections, err := p.workbench.store.ProductDependencies(ctx, scope, conversation, through)
	if err != nil {
		return err
	}
	for _, selection := range selections {
		if _, err := p.agent.workbenchProductSnapshot(ctx, i, selection); err != nil {
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
	if a.plan != nil && a.agent != nil && a.agent.workbenchKind(aiworkbench.ContentTaskKind) {
		return a.plan.authorizeProductHistory(ctx, scope, conversation, through) != nil
	}
	selections, err := a.store.ProductDependencies(ctx, scope, conversation, through)
	return err != nil || len(selections) != 0
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
