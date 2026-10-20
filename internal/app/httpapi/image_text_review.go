package httpapi

import (
	"context"
	"encoding/json"
	"strconv"

	"task-processor/internal/agent"
	"task-processor/internal/authidentity"
	"task-processor/internal/imageagent"
	"task-processor/internal/product/review"
)

// The existing content/review owners perform their live access checks. This
// adapter never queries Review tables or accepts client-authored copy/approval.
type imageTextReviewReader struct{ content *productAgentApplication }

func (r imageTextReviewReader) ReadConfirmedImageText(ctx context.Context, identity imageagent.ExecutionIdentity, ref imageagent.ImageTextReviewReference) (imageagent.ImageTextReviewCopy, error) {
	var zero imageagent.ImageTextReviewCopy
	i, ok := authidentity.AuthenticatedIdentityFromContext(ctx)
	a := r.content
	if !ok || i.TenantID != identity.TenantID || i.EffectiveOrganizationID != identity.TenantID || i.UserID != identity.UserID || i.EffectiveMemberID != identity.MemberID {
		return zero, imageagent.ErrIdentityRequired
	}
	if a == nil || a.reviews == nil || a.store == nil || a.configuration == nil {
		return zero, imageagent.ErrCommandBlocked
	}
	view, err := a.reviews.Get(ctx, ref.ID)
	if err != nil {
		return zero, err
	}
	if view.ID != ref.ID || view.Revision != ref.Revision || view.State != "accepted" || view.Content == nil || view.Execution == nil || view.PolicyAdoption != nil || view.Receipt != nil || len(view.Content.Decisions) == 0 {
		return zero, imageagent.ErrRevisionConflict
	}
	e := view.Execution
	if e.Scope.OrganizationID != identity.TenantID || view.Input.ProductKey != e.Binding.ProductKey || strconv.FormatUint(view.Input.BaseVersion, 10) != e.Binding.CatalogVersion || e.ContentSelection != view.Content.Selection {
		return zero, imageagent.ErrRevisionConflict
	}
	last := view.Content.Decisions[len(view.Content.Decisions)-1]
	if last.Action != "accept" || last.Revision != view.Revision || !review.ValidKey(last.Actor) {
		return zero, imageagent.ErrRevisionConflict
	}
	origin, found, err := a.store.Lookup(ctx, e.Scope, e.Binding, e.RequestKey)
	if err != nil {
		return zero, err
	}
	if !found || origin.State.RunID != e.RunID || !agentRunReviewable(origin.State) || *agentExecutionProvenance(origin.State) != *e {
		return zero, imageagent.ErrRevisionConflict
	}
	configuration, err := a.configuration.LoadSnapshot(ctx, e.Scope, e.Configuration)
	if err != nil {
		return zero, err
	}
	if configuration.Scope != e.Scope || configuration.Request.ModeSnapshotRef != e.Mode || configuration.Request.ContentSelection != e.ContentSelection {
		return zero, imageagent.ErrRevisionConflict
	}
	mode, err := a.configuration.LoadModeSnapshot(ctx, agent.ExecutionIdentity{OrganizationID: e.Scope.OrganizationID, ActorID: e.Scope.ActorID, MemberID: configuration.MemberID}, e.Mode)
	if err != nil {
		return zero, err
	}
	if mode.Mode != agent.ModeDefault {
		return zero, agent.ErrManualReviewRequired
	}
	lines, err := imageReviewLines(view)
	if err != nil {
		return zero, err
	}
	return imageagent.ImageTextReviewCopy{Review: ref, ProductID: e.Binding.ProductKey, PublicationID: e.Binding.PublicationID, Version: view.Input.BaseVersion, Language: view.Content.Selection.TargetLanguage, OriginRunID: e.RunID, ModeSnapshot: e.Mode, Lines: lines}, nil
}

func imageReviewLines(view review.View) ([]string, error) {
	if view.Content == nil {
		return nil, imageagent.ErrValidation
	}
	var title string
	var points []string
	for _, value := range view.Content.After {
		switch value.Field {
		case "title":
			title = value.Value
		case "selling_points":
			if json.Unmarshal([]byte(value.Value), &points) != nil {
				return nil, imageagent.ErrValidation
			}
		}
	}
	var lines []string
	if title != "" {
		lines = append(lines, title)
	}
	if len(points) > 0 {
		lines = append(lines, points[0])
	}
	if len(lines) == 0 {
		return nil, imageagent.ErrValidation
	}
	// Structural/Unicode limits are checked again by the image domain, without
	// truncating, translating or silently replacing an accepted value.
	return lines, nil
}
