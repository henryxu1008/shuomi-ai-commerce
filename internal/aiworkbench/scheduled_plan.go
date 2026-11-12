package aiworkbench

import (
	"encoding/json"
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
	// ExecutionSpec freezes the existing typed execution proposal/configuration.
	// It is populated by the application owner after current authorization.
	ExecutionSpec       json.RawMessage
	RuleStartAt         time.Time
	CreationFingerprint string
}

func (p ScheduledPlanDraft) Valid() bool {
	id, err := uuid.Parse(p.ID)
	if err != nil || id.String() != p.ID ||
		p.Scope.OrganizationID == "" || len(p.Scope.OrganizationID) > 128 ||
		p.Scope.ActorID == "" || len(p.Scope.ActorID) > 128 ||
		p.MemberID == "" || len(p.MemberID) > 128 ||
		!p.Selection.Valid() || !agent.ValidGoalSummary(p.GoalSummary) ||
		p.CreatedAt.IsZero() || !p.Rule.Valid() || len(p.ExecutionSpec) > 16384 ||
		len(p.ExecutionSpec) != 0 && !json.Valid(p.ExecutionSpec) {
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

type ScheduledPlan struct {
	ScheduledPlanDraft
	State               string
	Revision            int64
	ConsumedOccurrences uint64
	ActiveOccurrenceID  string
	RegistrationState   string
}

type ScheduledOccurrence struct {
	ID             string
	PlanID         string
	Scope          Scope
	ScheduledAtUTC time.Time
	PlanRevision   int64
	PayloadDigest  string
	State          string
	BusinessTaskID string
	Reason         string
}

func ScheduledOccurrenceID(scope Scope, planID string, slot time.Time) string {
	return uuid.NewSHA1(uuid.NameSpaceOID, []byte(scope.OrganizationID+"\x1f"+planID+"\x1f"+slot.UTC().Format(time.RFC3339Nano))).String()
}
