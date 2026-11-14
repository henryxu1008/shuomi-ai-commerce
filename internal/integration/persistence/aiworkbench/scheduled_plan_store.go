package aiworkbenchpersistence

import (
	"context"
	"encoding/json"
	"errors"
	"math"
	"time"

	"gorm.io/gorm"

	"task-processor/internal/aiworkbench"
)

// scheduledPlanRow is the Workbench-owned scheduled intent. It is never an
// AgentRun, Review result or a replacement for the original cost ledger.
type scheduledPlanRow struct {
	ID                     string
	OrganizationID         string
	OwnerUserID            string
	MemberID               string
	Kind                   string
	TaskSelection          json.RawMessage `gorm:"type:jsonb"`
	ReportSpec             json.RawMessage `gorm:"type:jsonb"`
	ExecutionSpec          json.RawMessage `gorm:"type:jsonb"`
	RuleStartAt            *time.Time
	RegistrationState      string `gorm:"default:PENDING"`
	CreationFingerprint    string
	GoalSummary            string
	Frequency              string
	TimeZone               string
	OnceAtUTC              *time.Time
	LocalHour              *int16
	LocalMinute            *int16
	LocalWeekday           *int16
	UntilUTC               *time.Time
	MaxOccurrences         int64
	BudgetUnit             string `gorm:"default:ai-micropoint-v1"`
	PerRunUpperBoundMicros int64
	TotalUpperBoundMicros  int64
	ConsumedOccurrences    int64
	ActiveOccurrenceID     *string
	State                  string
	Revision               int64
	CreatedAt              time.Time
	UpdatedAt              time.Time
}

func (scheduledPlanRow) TableName() string { return "ai_workbench.scheduled_plans" }

func schedulePlanValue(row scheduledPlanRow) (aiworkbench.ScheduledPlanDraft, error) {
	var selection aiworkbench.TaskSelection
	var report *aiworkbench.DailyReportSpec
	if row.Kind == aiworkbench.DailyReportKind {
		if json.Unmarshal(row.ReportSpec, &report) != nil || report == nil || len(row.TaskSelection) != 0 && string(row.TaskSelection) != "null" {
			return aiworkbench.ScheduledPlanDraft{}, aiworkbench.ErrUnavailable
		}
	} else if row.Kind != aiworkbench.ContentTaskKind || json.Unmarshal(row.TaskSelection, &selection) != nil || len(row.ReportSpec) != 0 && string(row.ReportSpec) != "null" {
		return aiworkbench.ScheduledPlanDraft{}, aiworkbench.ErrUnavailable
	}
	if row.MaxOccurrences <= 0 || row.PerRunUpperBoundMicros < 0 || row.TotalUpperBoundMicros < 0 {
		return aiworkbench.ScheduledPlanDraft{}, aiworkbench.ErrUnavailable
	}
	rule := aiworkbench.ScheduleRule{
		Frequency: aiworkbench.ScheduleFrequency(row.Frequency),
		TimeZone:  row.TimeZone,
	}
	if row.OnceAtUTC != nil {
		rule.AtUTC = row.OnceAtUTC.UTC()
	}
	if row.LocalHour != nil {
		rule.Hour = int(*row.LocalHour)
	}
	if row.LocalMinute != nil {
		rule.Minute = int(*row.LocalMinute)
	}
	if row.LocalWeekday != nil {
		rule.Weekday = time.Weekday(*row.LocalWeekday)
	}
	if row.UntilUTC != nil {
		rule.UntilUTC = row.UntilUTC.UTC()
	}
	value := aiworkbench.ScheduledPlanDraft{
		Report:   report,
		ID:       row.ID,
		Scope:    aiworkbench.Scope{OrganizationID: row.OrganizationID, ActorID: row.OwnerUserID},
		MemberID: row.MemberID, Selection: selection, GoalSummary: row.GoalSummary,
		Rule: rule, Budget: aiworkbench.ScheduleBudget{
			Unit:                   row.BudgetUnit,
			MaxOccurrences:         uint64(row.MaxOccurrences),
			PerRunUpperBoundMicros: uint64(row.PerRunUpperBoundMicros),
		},
		CreatedAt:           row.CreatedAt.UTC(),
		ExecutionSpec:       row.ExecutionSpec,
		CreationFingerprint: row.CreationFingerprint,
	}
	if row.RuleStartAt != nil {
		value.RuleStartAt = row.RuleStartAt.UTC()
	}
	total, err := value.TotalUpperBound()
	if err != nil || total > math.MaxInt64 || int64(total) != row.TotalUpperBoundMicros || !value.Valid() {
		return aiworkbench.ScheduledPlanDraft{}, aiworkbench.ErrUnavailable
	}
	return value, nil
}

// CreateScheduledPlan stores a validated, previously authorized intent.
// An external registration response is not part of this transaction. Until
// a verified Temporal registration receipt is applied, the plan is NOT active.
func (s *Store) CreateScheduledPlan(ctx context.Context, input aiworkbench.ScheduledPlanDraft) error {
	if s == nil || s.db == nil || ctx == nil || !validScope(input.Scope) ||
		!validName(input.MemberID) || !input.Valid() {
		return aiworkbench.ErrInvalid
	}
	raw, err := json.Marshal(input.Selection)
	if err != nil || len(raw) > 2048 {
		return aiworkbench.ErrInvalid
	}
	total, err := input.TotalUpperBound()
	if err != nil || total > math.MaxInt64 {
		return aiworkbench.ErrInvalid
	}
	row := scheduledPlanRow{
		ID: input.ID, OrganizationID: input.Scope.OrganizationID,
		OwnerUserID: input.Scope.ActorID, MemberID: input.MemberID,
		Kind: input.Kind(), TaskSelection: raw,
		ExecutionSpec:       input.ExecutionSpec,
		CreationFingerprint: input.CreationFingerprint,
		GoalSummary:         input.GoalSummary,
		Frequency:           string(input.Rule.Frequency), TimeZone: input.Rule.TimeZone,
		MaxOccurrences:         int64(input.Budget.MaxOccurrences),
		BudgetUnit:             input.Budget.Unit,
		PerRunUpperBoundMicros: int64(input.Budget.PerRunUpperBoundMicros),
		TotalUpperBoundMicros:  int64(total),
		State:                  "PENDING_REGISTRATION", Revision: 1,
		CreatedAt: input.CreatedAt.UTC(), UpdatedAt: input.CreatedAt.UTC(),
	}
	if input.Report != nil {
		row.TaskSelection = nil
		row.ReportSpec, err = json.Marshal(input.Report)
		if err != nil {
			return aiworkbench.ErrInvalid
		}
	}
	start := input.CreatedAt.UTC()
	if !input.RuleStartAt.IsZero() {
		start = input.RuleStartAt.UTC()
	}
	row.RuleStartAt = &start
	if row.CreationFingerprint == "" {
		row.CreationFingerprint = digest(struct {
			Selection aiworkbench.TaskSelection
			Goal      string
			Rule      aiworkbench.ScheduleRule
			Budget    aiworkbench.ScheduleBudget
		}{input.Selection, input.GoalSummary, input.Rule, input.Budget})
		if input.Report!=nil{row.CreationFingerprint=digest(input)}
	}
	if !validDigest(row.CreationFingerprint) {
		return aiworkbench.ErrInvalid
	}
	if len(row.ExecutionSpec) == 0 {
		row.ExecutionSpec = json.RawMessage(`{}`)
	}
	if !input.Rule.UntilUTC.IsZero() {
		t := input.Rule.UntilUTC.UTC()
		row.UntilUTC = &t
	}
	if input.Rule.Frequency == aiworkbench.ScheduleOnce {
		t := input.Rule.AtUTC.UTC()
		row.OnceAtUTC = &t
	} else {
		h, m := int16(input.Rule.Hour), int16(input.Rule.Minute)
		row.LocalHour, row.LocalMinute = &h, &m
		if input.Rule.Frequency == aiworkbench.ScheduleWeekly {
			weekday := int16(input.Rule.Weekday)
			row.LocalWeekday = &weekday
		}
	}
	create := s.db.WithContext(ctx)
	if input.Report == nil {
		create = create.Omit("ReportSpec")
	}
	if err := create.Create(&row).Error; err != nil {
		return unavailable(err)
	}
	return nil
}

// GetScheduledPlan requires the exact active actor/Organization scope. Its
// status is returned separately; PENDING_REGISTRATION is never "enabled".
func (s *Store) GetScheduledPlan(ctx context.Context, scope aiworkbench.Scope, id string) (aiworkbench.ScheduledPlanDraft, string, int64, error) {
	if s == nil || s.db == nil || ctx == nil || !validScope(scope) || !validKey(id) {
		return aiworkbench.ScheduledPlanDraft{}, "", 0, aiworkbench.ErrInvalid
	}
	var row scheduledPlanRow
	err := s.db.WithContext(ctx).Where("id = ? AND organization_id = ? AND owner_user_id = ?",
		id, scope.OrganizationID, scope.ActorID).Take(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return aiworkbench.ScheduledPlanDraft{}, "", 0, aiworkbench.ErrNotFound
	}
	if err != nil {
		return aiworkbench.ScheduledPlanDraft{}, "", 0, unavailable(err)
	}
	value, err := schedulePlanValue(row)
	return value, row.State, row.Revision, err
}

// MarkScheduleRegistered is called only after the scheduling adapter verifies
// the exact stable external schedule ID and rule/revision. It cannot be exposed
// to HTTP callers as a way to assert that Temporal succeeded.
func (s *Store) MarkScheduleRegistered(ctx context.Context, scope aiworkbench.Scope, id string, expectedRevision int64) error {
	if s == nil || s.db == nil || ctx == nil || !validScope(scope) ||
		!validKey(id) || expectedRevision < 1 {
		return aiworkbench.ErrInvalid
	}
	res := s.db.WithContext(ctx).Model(&scheduledPlanRow{}).Where(
		"id = ? AND organization_id = ? AND owner_user_id = ? AND revision = ? AND state IN ?",
		id, scope.OrganizationID, scope.ActorID, expectedRevision,
		[]string{"PENDING_REGISTRATION", "REGISTRATION_UNKNOWN", "PAUSED", "CLOSED", "ACTIVE"}).
		Updates(map[string]any{"state": gorm.Expr("CASE WHEN state IN ('PENDING_REGISTRATION','REGISTRATION_UNKNOWN') THEN 'ACTIVE' ELSE state END"), "registration_state": "MATCHED", "revision": gorm.Expr("revision + 1"), "updated_at": time.Now().UTC()})
	if res.Error != nil {
		return unavailable(res.Error)
	}
	if res.RowsAffected != 1 {
		return aiworkbench.ErrRevisionMismatch
	}
	return nil
}
