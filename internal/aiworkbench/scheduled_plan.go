package aiworkbench

import (
	"math"
	"time"

	"github.com/google/uuid"
	"task-processor/internal/agent"
)

// ScheduledPlanDraft is a SERVER-AUTHORIZED proposal to persist a bounded
// execution schedule. Never bind this type directly from an HTTP payload:
// membership, source binding, mode and cost limits must be verified by the
// existing domain owners before the application calls the persistence port.
type ScheduledPlanDraft struct {
	ID          string
	Scope       Scope
	MemberID    string
	Selection   TaskSelection
	GoalSummary string
	Rule        ScheduleRule
	Budget      ScheduleBudget
	CreatedAt   time.Time
}

func (p ScheduledPlanDraft) Valid() bool {
	id, err := uuid.Parse(p.ID)
	if err != nil || id.String() != p.ID ||
		p.Scope.OrganizationID == "" || len(p.Scope.OrganizationID) > 128 ||
		p.Scope.ActorID == "" || len(p.Scope.ActorID) > 128 ||
		p.MemberID == "" || len(p.MemberID) > 128 ||
		!p.Selection.Valid() || !agent.ValidGoalSummary(p.GoalSummary) ||
		p.CreatedAt.IsZero() || !p.Rule.Valid() {
		return false
	}
	total, err := p.Budget.TotalUpperBoundMicros()
	if err != nil || total > math.MaxInt64 ||
		p.Budget.MaxOccurrences > math.MaxInt64 ||
		p.Budget.PerRunUpperBoundMicros > math.MaxInt64 {
		return false
	}
	if p.Rule.Frequency == ScheduleOnce && p.Budget.MaxOccurrences != 1 {
		return false
	}
	_, err = p.Rule.NextAfter(p.CreatedAt)
	return err == nil // an expired rule cannot be registered as ACTIVE
}
