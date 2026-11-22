package httpapi

import (
	"context"
	"encoding/json"
	"strconv"

	"github.com/henryxu1008/shuomi-ai-commerce/internal/aiworkbench"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/authidentity"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/product/review"
)

type scheduledChanges struct {
	Selection      *aiworkbench.TaskSelection   `json:"selection,omitempty"`
	Report         *aiworkbench.DailyReportSpec `json:"report,omitempty"`
	GoalSummary    string                       `json:"goalSummary,omitempty"`
	MaxOccurrences string                       `json:"maxOccurrences"`
}

func (a *aiWorkbenchApplication) scheduledInputSource(ctx context.Context, owner aiworkbench.Scope, input scheduledInput) (*aiworkbench.ScheduledPlan, error) {
	if input.ScheduleID != "" && input.SourceScheduleID != "" || input.ProposalID != "" && (input.ScheduleID != "" || input.SourceScheduleID != "") || input.ExpectedRevision != "" && input.ScheduleID == "" {
		return nil, aiworkbench.ErrInvalid
	}
	id := input.ScheduleID
	if id == "" {
		id = input.SourceScheduleID
	}
	if id == "" {
		return nil, nil
	}
	if a == nil || a.store == nil {
		return nil, aiworkbench.ErrUnavailable
	}
	p, err := a.store.ReadScheduledPlan(ctx, owner, id)
	if err != nil {
		return nil, err
	}
	i, ok := authidentity.AuthenticatedIdentityFromContext(ctx)
	if !ok || i.TenantID != owner.OrganizationID || i.UserID != owner.ActorID || i.EffectiveMemberID != p.MemberID {
		return nil, review.ErrForbidden
	}
	if p.Deleted || input.ScheduleID != "" && p.State == "CLOSED" {
		return nil, aiworkbench.ErrRevisionMismatch
	}
	if input.ExpectedRevision != "" && input.ExpectedRevision != strconv.FormatInt(p.Revision, 10) {
		return nil, aiworkbench.ErrRevisionMismatch
	}
	if input.Kind != "" && input.Kind != p.Kind() || p.Report == nil && input.Report != nil {
		return nil, aiworkbench.ErrInvalid
	}
	if !a.scheduledView(ctx, p).DetailsAvailable {
		return nil, review.ErrForbidden
	}
	return &p, nil
}

func scheduledSourceConfirmation(q scheduledQuote, input scheduledInput, source *aiworkbench.ScheduledPlan) scheduledQuote {
	q.source = source
	if source == nil || input.SourceScheduleID == "" && input.ExpectedRevision == "" {
		return q
	}
	purpose, consumed := "copy", uint64(0)
	if input.ScheduleID != "" {
		purpose, consumed = "edit", source.ConsumedOccurrences
		q.ConsumedOccurrences = strconv.FormatUint(consumed, 10)
	}
	q.Confirmation = scheduledHash(struct {
		Confirmation, Purpose, Source, Digest string
		Revision                              int64
		Consumed                              uint64
	}{q.Confirmation, purpose, source.ID, aiworkbench.ScheduledPlanSourceDigest(source.ScheduledPlanDraft), source.Revision, consumed})
	return q
}

func (a *aiWorkbenchApplication) replanScheduledContent(ctx context.Context, owner aiworkbench.Scope, source *aiworkbench.ScheduledPlan, input scheduledInput) (aiworkbench.ExecutionProposal, error) {
	var original aiworkbench.ExecutionProposal
	if json.Unmarshal(source.ExecutionSpec, &original) != nil {
		return original, aiworkbench.ErrUnavailable
	}
	if input.ScheduleID != "" && input.ExpectedRevision == "" {
		if input.Selection != nil || input.GoalSummary != "" || input.MaxOccurrences != strconv.FormatUint(source.Budget.MaxOccurrences, 10) {
			return original, aiworkbench.ErrInvalid
		}
		return original, nil
	}
	if input.Selection == nil || !input.Selection.Valid() || input.GoalSummary == "" {
		return original, aiworkbench.ErrInvalid
	}
	command := aiworkbench.PlanningCommand{Scope: owner, WorkScope: aiworkbench.WorkScope{TaskSelection: input.Selection, TemplateID: original.TemplateID, TemplateRevision: original.TemplateRevision, KnowledgeBaseID: original.KnowledgeBaseID}}
	if !original.WorkbenchContextRef.Absent() {
		f, err := a.authorizedFrozenScope(ctx, owner, original.WorkbenchContextRef, true)
		if err != nil {
			return original, err
		}
		command.FrozenScope = &f
		command.ConversationID = original.WorkbenchContextRef.ConversationID
		command.IdempotencyKey = original.WorkbenchContextRef.CommandKey
		command.ScopeDigest = original.WorkbenchContextRef.Digest
		command.MemberID = f.MemberID
	}
	replanned, err := (&workbenchPlanner{agent: a.agent, workbench: a}).buildProposal(ctx, command, input.GoalSummary)
	// Retain the real originating message for current history authorization.
	// This is a plan revision, not a new immutable Chat proposal with its ID.
	replanned.ConversationID = original.ConversationID
	replanned.AssistantMessageID = original.AssistantMessageID
	return replanned, err
}
