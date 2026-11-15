package httpapi

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"strconv"
	"time"

	"task-processor/internal/agent"
	"task-processor/internal/agentconfig"
	"task-processor/internal/aicapability"
	"task-processor/internal/aiworkbench"
	"task-processor/internal/authidentity"
	"task-processor/internal/product/review"
)

type scheduledInput struct {
	Kind           string                       `json:"kind,omitempty"`
	Report         *aiworkbench.DailyReportSpec `json:"report,omitempty"`
	ProposalID     string                       `json:"proposalId,omitempty"`
	ScheduleID     string                       `json:"scheduleId,omitempty"`
	Selection      *aiworkbench.TaskSelection   `json:"selection,omitempty"`
	GoalSummary    string                       `json:"goalSummary,omitempty"`
	Rule           aiworkbench.ScheduleRule     `json:"rule"`
	MaxOccurrences string                       `json:"maxOccurrences"`
	Confirmation   string                       `json:"confirmation,omitempty"`
}

type scheduledQuote struct {
	StoreName              string                       `json:"storeName,omitempty"`
	Kind                   string                       `json:"kind,omitempty"`
	Report                 *aiworkbench.DailyReportSpec `json:"report,omitempty"`
	Confirmation           string                       `json:"confirmation"`
	Selection              *aiworkbench.TaskSelection   `json:"selection,omitempty"`
	GoalSummary            string                       `json:"goalSummary"`
	Rule                   aiworkbench.ScheduleRule     `json:"rule"`
	MaxOccurrences         string                       `json:"maxOccurrences"`
	Unit                   string                       `json:"unit"`
	PerRunUpperBoundMicros string                       `json:"perRunUpperBoundMicros"`
	TotalUpperBoundMicros  string                       `json:"totalUpperBoundMicros"`
	NextAt                 time.Time                    `json:"nextAt"`
	ExecutionMode          agent.ExecutionMode          `json:"executionMode,omitempty"`
}

func scheduledHash(value any) string {
	wire, _ := json.Marshal(value)
	sum := sha256.Sum256(wire)
	return hex.EncodeToString(sum[:])
}

func scheduledQuoteConfirmation(p aiworkbench.ExecutionProposal, rule aiworkbench.ScheduleRule, budget aiworkbench.ScheduleBudget) string {
	return scheduledHash(struct {
		Proposal aiworkbench.ExecutionProposal
		Rule     aiworkbench.ScheduleRule
		Budget   aiworkbench.ScheduleBudget
	}{p, rule, budget})
}

func (a *aiWorkbenchApplication) validateScheduledProposal(ctx context.Context, p aiworkbench.ExecutionProposal) (agent.ExecutionMode, error) {
	if a == nil || a.agent == nil || !p.ValidCapability() || p.Kind != aiworkbench.ContentTaskKind || p.KnowledgeBaseID != "" {
		return "", aiworkbench.ErrUnavailable
	}
	i, err := a.agent.freshIdentity(ctx)
	if err != nil || p.Scope != (aiworkbench.Scope{OrganizationID: i.TenantID, ActorID: i.UserID}) {
		return "", review.ErrForbidden
	}
	if a.proposalHistoryUnavailable(ctx, p.Scope, p, i) {
		return "", review.ErrForbidden
	}
	b, err := a.agent.workbenchProposalBinding(ctx, i, p)
	if err != nil {
		return "", err
	}
	if b.ProductKey != p.ProductKey || b.PublicationID != p.PublicationID || b.CatalogVersion != p.CatalogVersion || p.ExecutionLimits != a.agent.config.Limits {
		return "", aiworkbench.ErrProposalStale
	}
	configuration, err := a.agent.configuration.ReadAgent(ctx, agent.Scope{OrganizationID: i.TenantID, ActorID: i.UserID}, p.AgentID)
	if err != nil {
		return "", err
	}
	if configuration.Activation != "ENABLED" || configuration.Revision != p.ObservedAgentRevision || configuration.ActivationEpoch != p.ObservedActivationEpoch {
		return "", aiworkbench.ErrProposalStale
	}
	mode, err := a.agent.configuration.ReadExecutionMode(ctx, agent.ExecutionIdentity{OrganizationID: i.TenantID, ActorID: i.UserID, MemberID: i.EffectiveMemberID}, a.agent.modeSubject())
	if err != nil {
		return "", err
	}
	if mode.Revision != p.ObservedModeRevision || (mode.Mode != agent.ModeDefault && mode.Mode != agent.ModeReadOnly) {
		return "", agent.ErrManualReviewRequired
	}
	var frozen aicapability.ModelProfile
	if json.Unmarshal(p.ExecutionModelProfile, &frozen) != nil {
		return "", aiworkbench.ErrUnavailable
	}
	current, err := a.agent.selectTitleProfile(ctx, i.TenantID)
	if err != nil {
		return "", err
	}
	if !p.WorkbenchContextRef.Absent() {
		current = agentconfig.ScopedContentProfile(current)
	}
	if current.Validate() != nil || current != frozen {
		return "", aiworkbench.ErrProposalStale
	}
	if !p.WorkbenchContextRef.Absent() {
		if _, err = a.authorizedFrozenScope(ctx, p.Scope, p.WorkbenchContextRef, true); err != nil {
			return "", err
		}
		conversation, err := a.store.Get(ctx, p.Scope, p.WorkbenchContextRef.ConversationID)
		if err != nil {
			return "", err
		}
		if conversation.Archived {
			return "", aiworkbench.ErrConversationArchived
		}
	}
	return mode.Mode, nil
}

func (a *aiWorkbenchApplication) quoteScheduled(ctx context.Context, owner aiworkbench.Scope, input scheduledInput) (scheduledQuote, aiworkbench.ExecutionProposal, aiworkbench.ScheduleBudget, error) {
	if input.Report != nil || input.Kind == aiworkbench.DailyReportKind {
		q, e := a.quoteDailyReport(ctx, owner, input)
		count, _ := strconv.ParseUint(input.MaxOccurrences, 10, 63)
		return q, aiworkbench.ExecutionProposal{}, aiworkbench.ScheduleBudget{Unit: "none", MaxOccurrences: count}, e
	}
	if input.Kind != "" && input.Kind != aiworkbench.ContentTaskKind {
		return scheduledQuote{}, aiworkbench.ExecutionProposal{}, aiworkbench.ScheduleBudget{}, aiworkbench.ErrInvalid
	}
	if input.ScheduleID != "" {
		if p, e := a.store.ReadScheduledPlan(ctx, owner, input.ScheduleID); e == nil && p.Report != nil {
			input.Report = p.Report
			input.Kind = aiworkbench.DailyReportKind
			q, e := a.quoteDailyReport(ctx, owner, input)
			return q, aiworkbench.ExecutionProposal{}, p.Budget, e
		}
	}
	if a == nil || a.agent == nil || a.schedules == nil || !a.schedules.Ready() {
		return scheduledQuote{}, aiworkbench.ExecutionProposal{}, aiworkbench.ScheduleBudget{}, aiworkbench.ErrUnavailable
	}
	count, err := strconv.ParseUint(input.MaxOccurrences, 10, 63)
	if err != nil || count == 0 || strconv.FormatUint(count, 10) != input.MaxOccurrences || !input.Rule.Valid() || input.Rule.Frequency == aiworkbench.ScheduleOnce && (count != 1 || input.Rule.AtUTC.Nanosecond() != 0) {
		return scheduledQuote{}, aiworkbench.ExecutionProposal{}, aiworkbench.ScheduleBudget{}, aiworkbench.ErrInvalid
	}
	next, err := input.Rule.NextAfter(time.Now().UTC())
	if err != nil {
		return scheduledQuote{}, aiworkbench.ExecutionProposal{}, aiworkbench.ScheduleBudget{}, aiworkbench.ErrInvalid
	}
	var p aiworkbench.ExecutionProposal
	switch {
	case input.ScheduleID != "":
		if input.ProposalID != "" || input.Selection != nil || input.GoalSummary != "" {
			return scheduledQuote{}, p, aiworkbench.ScheduleBudget{}, aiworkbench.ErrInvalid
		}
		plan, err := a.store.ReadScheduledPlan(ctx, owner, input.ScheduleID)
		if err != nil {
			return scheduledQuote{}, p, aiworkbench.ScheduleBudget{}, err
		}
		if count != plan.Budget.MaxOccurrences || json.Unmarshal(plan.ExecutionSpec, &p) != nil {
			return scheduledQuote{}, p, aiworkbench.ScheduleBudget{}, aiworkbench.ErrInvalid
		}
	case input.ProposalID != "":
		if input.Selection != nil || input.GoalSummary != "" {
			return scheduledQuote{}, p, aiworkbench.ScheduleBudget{}, aiworkbench.ErrInvalid
		}
		p, err = a.store.GetProposal(ctx, owner, input.ProposalID)
	case input.Selection != nil && input.Selection.Valid() && input.GoalSummary != "" && agent.ValidGoalSummary(input.GoalSummary):
		p, err = (&workbenchPlanner{agent: a.agent, workbench: a}).buildProposal(ctx, aiworkbench.PlanningCommand{Scope: owner, WorkScope: aiworkbench.WorkScope{TaskSelection: input.Selection}}, input.GoalSummary)
	default:
		return scheduledQuote{}, p, aiworkbench.ScheduleBudget{}, aiworkbench.ErrInvalid
	}
	if err != nil {
		return scheduledQuote{}, p, aiworkbench.ScheduleBudget{}, err
	}
	mode, err := a.validateScheduledProposal(ctx, p)
	if err != nil {
		return scheduledQuote{}, p, aiworkbench.ScheduleBudget{}, err
	}
	var profile aicapability.ModelProfile
	if json.Unmarshal(p.ExecutionModelProfile, &profile) != nil {
		return scheduledQuote{}, p, aiworkbench.ScheduleBudget{}, aiworkbench.ErrUnavailable
	}
	budget, err := aiworkbench.BudgetForScheduledExecution(profile, p.ExecutionLimits, count)
	if err != nil {
		return scheduledQuote{}, p, budget, err
	}
	total, _ := budget.TotalUpperBoundMicros()
	confirmation := scheduledQuoteConfirmation(p, input.Rule, budget)
	return scheduledQuote{Confirmation: confirmation, Selection: p.Selection(), GoalSummary: p.GoalSummary, Rule: input.Rule, MaxOccurrences: input.MaxOccurrences, Unit: budget.Unit, PerRunUpperBoundMicros: strconv.FormatUint(budget.PerRunUpperBoundMicros, 10), TotalUpperBoundMicros: strconv.FormatUint(total, 10), NextAt: next, ExecutionMode: mode}, p, budget, nil
}

type scheduledPlanView struct {
	StoreName              string                       `json:"storeName,omitempty"`
	Kind                   string                       `json:"kind,omitempty"`
	Report                 *aiworkbench.DailyReportSpec `json:"report,omitempty"`
	ID                     string                       `json:"id"`
	Revision               string                       `json:"revision"`
	State                  string                       `json:"state"`
	RegistrationState      string                       `json:"registrationState"`
	GoalSummary            string                       `json:"goalSummary,omitempty"`
	Selection              *aiworkbench.TaskSelection   `json:"selection,omitempty"`
	DetailsAvailable       bool                         `json:"detailsAvailable"`
	Rule                   aiworkbench.ScheduleRule     `json:"rule"`
	MaxOccurrences         string                       `json:"maxOccurrences"`
	ConsumedOccurrences    string                       `json:"consumedOccurrences"`
	Unit                   string                       `json:"unit"`
	PerRunUpperBoundMicros string                       `json:"perRunUpperBoundMicros"`
	TotalUpperBoundMicros  string                       `json:"totalUpperBoundMicros"`
	NextAt                 *time.Time                   `json:"nextAt,omitempty"`
	ActiveOccurrenceID     string                       `json:"activeOccurrenceId,omitempty"`
	CreatedAt              time.Time                    `json:"createdAt"`
}

func (a *aiWorkbenchApplication) scheduledView(ctx context.Context, plan aiworkbench.ScheduledPlan) scheduledPlanView {
	total, _ := plan.TotalUpperBound()
	view := scheduledPlanView{ID: plan.ID, Revision: strconv.FormatInt(plan.Revision, 10), State: plan.State, RegistrationState: plan.RegistrationState, Rule: plan.Rule, MaxOccurrences: strconv.FormatUint(plan.Budget.MaxOccurrences, 10), ConsumedOccurrences: strconv.FormatUint(plan.ConsumedOccurrences, 10), Unit: plan.Budget.Unit, PerRunUpperBoundMicros: strconv.FormatUint(plan.Budget.PerRunUpperBoundMicros, 10), TotalUpperBoundMicros: strconv.FormatUint(total, 10), ActiveOccurrenceID: plan.ActiveOccurrenceID, CreatedAt: plan.CreatedAt}
	i, ok := authidentity.AuthenticatedIdentityFromContext(ctx)
	if plan.Report != nil {
		view.Kind = aiworkbench.DailyReportKind
		if ok && a.dailyReports != nil && i.EffectiveMemberID == plan.MemberID {
			if store, e := a.dailyReports.Store(ctx, reportOriginForPlan(plan), false); e == nil {
				view.DetailsAvailable = true
				view.StoreName = store.Name()
				view.Report = plan.Report
				view.GoalSummary = plan.GoalSummary
			}
		}
	} else if a.agent != nil {
		var p aiworkbench.ExecutionProposal
		if ok && json.Unmarshal(plan.ExecutionSpec, &p) == nil && i.TenantID == plan.Scope.OrganizationID && i.UserID == plan.Scope.ActorID {
			readable := true
			if !p.WorkbenchContextRef.Absent() {
				_, err := a.authorizedFrozenScope(ctx, plan.Scope, p.WorkbenchContextRef, false)
				readable = err == nil
			}
			if readable && !a.proposalHistoryUnavailable(ctx, plan.Scope, p, i) {
				view.GoalSummary = plan.GoalSummary
				binding := agent.Binding{ContextKind: p.ContextKind, ContextID: p.OperationID, ProductKey: p.ProductKey, CatalogVersion: p.CatalogVersion, PublicationID: p.PublicationID, TargetPlatform: p.TargetPlatform}
				if a.agent.workbenchAuthorizeBinding(ctx, i, binding) == nil {
					view.DetailsAvailable = true
					view.Selection = &plan.Selection
				}
			}
		}
	}
	if plan.State == "ACTIVE" && plan.RegistrationState == "MATCHED" && plan.ConsumedOccurrences < plan.Budget.MaxOccurrences {
		if next, err := plan.Rule.NextAfter(time.Now().UTC()); err == nil {
			view.NextAt = &next
		}
	}
	return view
}
