//go:build integration

package aiworkbenchpersistence

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
	"task-processor/internal/agent"
	"task-processor/internal/agentconfig"
	"task-processor/internal/aiworkbench"
	reviewproposal "task-processor/internal/product/review/proposal"
)

func TestScheduledTaskHasOneDurableSourceAndAtomicClaim(t *testing.T) {
	db := workbenchDB(t)
	store, err := New(db)
	require.NoError(t, err)
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Second)
	scope := aiworkbench.Scope{OrganizationID: "org-a", ActorID: "alice"}
	selection := aiworkbench.TaskSelection{Subject: aiworkbench.Subject{Kind: "own_product", ID: uuid.NewString(), CatalogVersion: "1", PublicationID: uuid.NewString()}, Capability: aiworkbench.ContentTaskKind, Parameters: reviewproposal.ContentSelection{TargetLanguage: "en-US", Description: true}}
	p := aiworkbench.ExecutionProposal{Kind: aiworkbench.ContentTaskKind, Scope: scope, GoalSummary: "Localize product", ContextKind: "own_product", OperationID: selection.Subject.ID, ProductKey: "product-a", CatalogVersion: "1", PublicationID: selection.Subject.PublicationID, TargetPlatform: "product", AgentID: agentconfig.ContentAgentID, AgentVersion: agentconfig.ContentAgentVersion, ContentSelection: selection.Parameters, ExecutionLimits: agent.Limits{Steps: 3, ModelCalls: 2, Tokens: 1000, CostMicros: 2500, Currency: "USD", Runtime: time.Minute}}
	spec, err := json.Marshal(p)
	require.NoError(t, err)
	plan := aiworkbench.ScheduledPlanDraft{ID: uuid.NewString(), Scope: scope, MemberID: "member-a", Selection: selection, GoalSummary: p.GoalSummary, Rule: aiworkbench.ScheduleRule{Frequency: aiworkbench.ScheduleOnce, TimeZone: "UTC", AtUTC: now.Add(time.Minute)}, Budget: aiworkbench.ScheduleBudget{MaxOccurrences: 1, PerRunUpperBoundMicros: 2500}, CreatedAt: now, ExecutionSpec: spec}
	require.NoError(t, store.CreateScheduledPlan(ctx, plan))
	require.NoError(t, store.MarkScheduleRegistered(ctx, scope, plan.ID, 1))
	key := aiworkbench.ScheduledOccurrenceID(scope, plan.ID, plan.Rule.AtUTC)
	ref := agent.ConfigurationSnapshotRef{Kind: agentconfig.SnapshotKind, ID: uuid.NewString(), Digest: strings.Repeat("a", 64)}
	r := agent.Request{Key: key, Binding: agent.Binding{ContextKind: "own_product", ContextID: p.OperationID, ProductKey: p.ProductKey, CatalogVersion: "1", PublicationID: p.PublicationID, TargetPlatform: "product"}, GoalSummary: p.GoalSummary, ContentSelection: p.ContentSelection, PolicyVersion: reviewproposal.ContentPolicyVersion, PromptVersion: agentconfig.ContentPromptVersion, Limits: p.ExecutionLimits, ManualReviewOnly: true, ConfigurationSnapshotRef: ref}
	r.ScheduledSource = agent.ScheduledSourceRef{PlanID: plan.ID, OccurrenceID: key}
	raw, err := json.Marshal(r)
	require.NoError(t, err)
	prepared := aiworkbench.PreparedTask{ExecutionRequest: raw, ConfigurationSnapshotRef: ref}
	_, _, _, err = store.ClaimScheduledTask(ctx, scope, plan.ID, 1, plan.Rule.AtUTC, plan.Rule.AtUTC, prepared)
	require.ErrorIs(t, err, aiworkbench.ErrRevisionMismatch)
	require.NoError(t, db.Exec(`CREATE FUNCTION ai_workbench.reject_schedule_task_test() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN RAISE EXCEPTION 'forced insert failure'; END $$`).Error)
	require.NoError(t, db.Exec(`CREATE TRIGGER reject_schedule_task_test BEFORE INSERT ON ai_workbench.business_tasks FOR EACH ROW EXECUTE FUNCTION ai_workbench.reject_schedule_task_test()`).Error)
	_, _, _, err = store.ClaimScheduledTask(ctx, scope, plan.ID, 2, plan.Rule.AtUTC, plan.Rule.AtUTC, prepared)
	require.Error(t, err)
	var occurrences int64
	require.NoError(t, db.Model(&scheduledOccurrenceRow{}).Count(&occurrences).Error)
	require.Zero(t, occurrences, "failed task creation rolls back the occurrence and its quota claim")
	require.NoError(t, db.Exec(`DROP TRIGGER reject_schedule_task_test ON ai_workbench.business_tasks`).Error)
	occurrence, task, replay, err := store.ClaimScheduledTask(ctx, scope, plan.ID, 2, plan.Rule.AtUTC, plan.Rule.AtUTC, prepared)
	require.NoError(t, err)
	require.False(t, replay)
	require.Equal(t, occurrence.ID, task.ScheduledOccurrenceID)
	for _, kind := range []string{"all", "unrecorded"} {
		page, err := store.ListScopedTasks(ctx, scope, aiworkbench.ScopeSelection{Kind: kind}, "", 20, func(*aiworkbench.BusinessScope, aiworkbench.BusinessTask) bool { return true })
		require.NoError(t, err)
		require.Len(t, page.Tasks, 1, "scheduled source must be included in the task center's scoped query")
		require.Equal(t, task.ID, page.Tasks[0].ID)
	}
	require.Empty(t, task.ConversationID)
	require.Empty(t, task.ProposalID)
	require.Equal(t, task.ID, occurrence.BusinessTaskID)
	_, same, replay, err := store.ClaimScheduledTask(ctx, scope, plan.ID, 2, plan.Rule.AtUTC, plan.Rule.AtUTC, prepared)
	require.NoError(t, err)
	require.True(t, replay)
	require.Equal(t, task.ID, same.ID)
	r.GoalSummary = "A different goal"
	prepared.ExecutionRequest, err = json.Marshal(r)
	require.NoError(t, err)
	_, _, _, err = store.ClaimScheduledTask(ctx, scope, plan.ID, 2, plan.Rule.AtUTC, plan.Rule.AtUTC, prepared)
	require.ErrorIs(t, err, aiworkbench.ErrIdempotencyConflict)
	var count int64
	require.NoError(t, db.Model(&taskRow{}).Count(&count).Error)
	require.EqualValues(t, 1, count)
	require.Error(t, db.Exec("UPDATE ai_workbench.business_tasks SET source_kind='CHAT_PROPOSAL' WHERE id=?", task.ID).Error)
	require.Error(t, db.Exec("UPDATE ai_workbench.business_tasks SET scheduled_occurrence_id=? WHERE id=?", uuid.NewString(), task.ID).Error)
	other := newScheduledOccurrence(scheduledPlanRow{ID: plan.ID, OrganizationID: scope.OrganizationID, OwnerUserID: scope.ActorID, Revision: 2}, plan, plan.Rule.AtUTC.Add(time.Hour), now, "SKIPPED")
	require.NoError(t, db.Create(&other).Error)
	require.Error(t, db.Model(&scheduledOccurrenceRow{}).Where("id=?", other.ID).Update("business_task_id", task.ID).Error, "a task cannot be attached to another occurrence under the same actor")
	controlKey := uuid.NewString()
	paused, err := store.ChangeScheduledPlan(ctx, scope, plan.ID, controlKey, 2, "pause", nil, "", now)
	require.NoError(t, err)
	require.Equal(t, "PAUSED", paused.State)
	require.Equal(t, occurrence.ID, paused.ActiveOccurrenceID, "pause only controls future admission")
	_, err = store.ChangeScheduledPlan(ctx, scope, plan.ID, controlKey, 2, "close", nil, "", now)
	require.ErrorIs(t, err, aiworkbench.ErrIdempotencyConflict)
	repeated, err := store.ChangeScheduledPlan(ctx, scope, plan.ID, controlKey, 2, "pause", nil, "", now)
	require.NoError(t, err)
	require.Equal(t, paused.Revision, repeated.Revision)
	_, err = store.ChangeScheduledPlan(ctx, scope, plan.ID, uuid.NewString(), 2, "resume", nil, "", now)
	require.ErrorIs(t, err, aiworkbench.ErrRevisionMismatch)
	closed, err := store.ChangeScheduledPlan(ctx, scope, plan.ID, uuid.NewString(), paused.Revision, "close", nil, "", now)
	require.NoError(t, err)
	_, err = store.ChangeScheduledPlan(ctx, scope, plan.ID, uuid.NewString(), closed.Revision, "resume", nil, "", now)
	require.ErrorIs(t, err, aiworkbench.ErrRevisionMismatch, "closed plans cannot resume")
}

func TestScheduledPreflightReceiptCannotBecomePaidAfterResume(t *testing.T) {
	db := workbenchDB(t)
	store, err := New(db)
	require.NoError(t, err)
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Second)
	scope := aiworkbench.Scope{OrganizationID: "org-a", ActorID: "alice"}
	p := aiworkbench.ScheduledPlanDraft{ID: uuid.NewString(), Scope: scope, MemberID: "member-a", GoalSummary: "Localize", Selection: aiworkbench.TaskSelection{Subject: aiworkbench.Subject{Kind: "own_product", ID: uuid.NewString(), CatalogVersion: "1", PublicationID: uuid.NewString()}, Capability: aiworkbench.ContentTaskKind, Parameters: reviewproposal.ContentSelection{TargetLanguage: "en-US", Description: true}}, Rule: aiworkbench.ScheduleRule{Frequency: aiworkbench.ScheduleOnce, TimeZone: "UTC", AtUTC: now.Add(time.Minute)}, Budget: aiworkbench.ScheduleBudget{MaxOccurrences: 1, PerRunUpperBoundMicros: 2500}, CreatedAt: now}
	require.NoError(t, store.CreateScheduledPlan(ctx, p))
	require.NoError(t, store.MarkScheduleRegistered(ctx, scope, p.ID, 1))
	paused, err := store.ChangeScheduledPlan(ctx, scope, p.ID, uuid.NewString(), 2, "pause", nil, "", now)
	require.NoError(t, err)
	require.NoError(t, store.RecordScheduledPreflightReceipt(ctx, scope, p.ID, p.Rule.AtUTC, p.Rule.AtUTC, "SKIPPED", "PLAN_NOT_ACTIVE"))
	_, err = store.ChangeScheduledPlan(ctx, scope, p.ID, uuid.NewString(), paused.Revision, "resume", nil, "", now)
	require.NoError(t, err)
	receipt, replay, err := store.ClaimScheduledOccurrence(ctx, scope, p.ID, p.Rule.AtUTC, p.Rule.AtUTC)
	require.NoError(t, err)
	require.True(t, replay)
	require.Equal(t, "SKIPPED", receipt.State)
	current, err := store.ReadScheduledPlan(ctx, scope, p.ID)
	require.NoError(t, err)
	require.Zero(t, current.ConsumedOccurrences)
	require.Empty(t, current.ActiveOccurrenceID)
}

func TestScheduledResumeFencesUndeliveredPausedSlots(t *testing.T) {
	for _, days := range []int{0, 300} {
		t.Run(fmt.Sprint(days), func(t *testing.T) {
			db := workbenchDB(t)
			store, err := New(db)
			require.NoError(t, err)
			ctx := context.Background()
			scope := aiworkbench.Scope{OrganizationID: "org-a", ActorID: "alice"}
			created := time.Date(2027, 1, 1, 8, 58, 0, 0, time.UTC)
			p := aiworkbench.ScheduledPlanDraft{ID: uuid.NewString(), Scope: scope, MemberID: "member", GoalSummary: "Localize", Selection: aiworkbench.TaskSelection{Subject: aiworkbench.Subject{Kind: "own_product", ID: uuid.NewString(), CatalogVersion: "1", PublicationID: uuid.NewString()}, Capability: aiworkbench.ContentTaskKind, Parameters: reviewproposal.ContentSelection{TargetLanguage: "en-US", Description: true}}, Rule: aiworkbench.ScheduleRule{Frequency: aiworkbench.ScheduleDaily, TimeZone: "UTC", Hour: 9}, Budget: aiworkbench.ScheduleBudget{MaxOccurrences: 3, PerRunUpperBoundMicros: 1}, CreatedAt: created}
			require.NoError(t, store.CreateScheduledPlan(ctx, p))
			require.NoError(t, store.MarkScheduleRegistered(ctx, scope, p.ID, 1))
			paused, err := store.ChangeScheduledPlan(ctx, scope, p.ID, uuid.NewString(), 2, "pause", nil, "", created.Add(time.Minute))
			require.NoError(t, err)
			// No callback or Temporal pause acknowledgement arrives before resume.
			resumeAt := created.AddDate(0, 0, days).Add(7 * time.Minute)
			key := uuid.NewString()
			resumed, err := store.ChangeScheduledPlan(ctx, scope, p.ID, key, paused.Revision, "resume", nil, "", resumeAt)
			if days > 256 {
				require.ErrorIs(t, err, aiworkbench.ErrScheduleCatchupPending)
				current, readErr := store.ReadScheduledPlan(ctx, scope, p.ID)
				require.NoError(t, readErr)
				require.Equal(t, "PAUSED", current.State)
				require.Equal(t, paused.Revision, current.Revision)
				_, found, readErr := store.ReadScheduledControlReceipt(ctx, scope, p.ID, key, paused.Revision, "resume", nil, "")
				require.NoError(t, readErr)
				require.False(t, found, "partial history progress must not acknowledge resume")
				resumed, err = store.ChangeScheduledPlan(ctx, scope, p.ID, key, paused.Revision, "resume", nil, "", resumeAt)
			}
			require.NoError(t, err)
			require.NoError(t, store.MarkScheduleRegistered(ctx, scope, p.ID, resumed.Revision))
			lastPausedSlot := time.Date(resumeAt.Year(), resumeAt.Month(), resumeAt.Day(), 9, 0, 0, 0, time.UTC)
			receipt, replay, err := store.ClaimScheduledOccurrence(ctx, scope, p.ID, lastPausedSlot, resumeAt.Add(time.Minute))
			require.NoError(t, err)
			require.True(t, replay)
			require.Equal(t, "SKIPPED", receipt.State)
			require.Equal(t, "PLAN_NOT_ACTIVE", receipt.Reason)
			current, err := store.ReadScheduledPlan(ctx, scope, p.ID)
			require.NoError(t, err)
			require.Zero(t, current.ConsumedOccurrences)
			require.Empty(t, current.ActiveOccurrenceID)
			next, err := p.Rule.NextAfter(resumeAt)
			require.NoError(t, err)
			receipt, replay, err = store.ClaimScheduledOccurrence(ctx, scope, p.ID, next, next)
			require.NoError(t, err)
			require.False(t, replay)
			require.Equal(t, "CLAIMED", receipt.State, "resume must still admit future slots")
		})
	}
}

func TestScheduledControlPreservesOldRuleElapsedSlots(t *testing.T) {
	for _, action := range []string{"edit", "close"} {
		for _, days := range []int{0, 300} {
			t.Run(fmt.Sprintf("%s/%d", action, days), func(t *testing.T) {
				db := workbenchDB(t)
				store, err := New(db)
				require.NoError(t, err)
				ctx := context.Background()
				scope := aiworkbench.Scope{OrganizationID: "org-a", ActorID: "alice"}
				created := time.Date(2027, 1, 1, 8, 58, 0, 0, time.UTC)
				p := aiworkbench.ScheduledPlanDraft{ID: uuid.NewString(), Scope: scope, MemberID: "member", GoalSummary: "Localize", Selection: aiworkbench.TaskSelection{Subject: aiworkbench.Subject{Kind: "own_product", ID: uuid.NewString(), CatalogVersion: "1", PublicationID: uuid.NewString()}, Capability: aiworkbench.ContentTaskKind, Parameters: reviewproposal.ContentSelection{TargetLanguage: "en-US", Description: true}}, Rule: aiworkbench.ScheduleRule{Frequency: aiworkbench.ScheduleDaily, TimeZone: "UTC", Hour: 9}, Budget: aiworkbench.ScheduleBudget{MaxOccurrences: 3, PerRunUpperBoundMicros: 1}, CreatedAt: created}
				require.NoError(t, store.CreateScheduledPlan(ctx, p))
				require.NoError(t, store.MarkScheduleRegistered(ctx, scope, p.ID, 1))
				editAt := created.AddDate(0, 0, days).Add(22 * time.Minute)
				key := uuid.NewString()
				rule := aiworkbench.ScheduleRule{Frequency: aiworkbench.ScheduleDaily, TimeZone: "UTC", Hour: 10}
				var change *aiworkbench.ScheduleRule
				if action == "edit" {
					change = &rule
				}
				edited, err := store.ChangeScheduledPlan(ctx, scope, p.ID, key, 2, action, change, "", editAt)
				if days > 256 {
					require.ErrorIs(t, err, aiworkbench.ErrScheduleCatchupPending)
					unchanged, readErr := store.ReadScheduledPlan(ctx, scope, p.ID)
					require.NoError(t, readErr)
					require.Equal(t, p.Rule, unchanged.Rule)
					require.EqualValues(t, 2, unchanged.Revision)
					_, found, readErr := store.ReadScheduledControlReceipt(ctx, scope, p.ID, key, 2, action, change, "")
					require.NoError(t, readErr)
					require.False(t, found)
					edited, err = store.ChangeScheduledPlan(ctx, scope, p.ID, key, 2, action, change, "", editAt)
				}
				require.NoError(t, err)
				if action == "edit" {
					require.Equal(t, rule, edited.Rule)
				} else {
					require.Equal(t, "CLOSED", edited.State)
				}
				oldSlot := time.Date(editAt.Year(), editAt.Month(), editAt.Day(), 9, 0, 0, 0, time.UTC)
				receipt, err := store.ReadScheduledOccurrence(ctx, scope, p.ID, oldSlot)
				require.NoError(t, err)
				require.Equal(t, "SKIPPED", receipt.State)
				require.Equal(t, "MISSED_TRIGGER", receipt.Reason)
				require.EqualValues(t, 2, receipt.PlanRevision)
				require.Zero(t, edited.ConsumedOccurrences)
				require.Empty(t, edited.ActiveOccurrenceID)
				receipt, replay, err := store.ClaimScheduledOccurrence(ctx, scope, p.ID, oldSlot, editAt)
				require.NoError(t, err)
				require.True(t, replay)
				require.Equal(t, "SKIPPED", receipt.State)
				if action == "close" {
					return
				}
				require.NoError(t, store.MarkScheduleRegistered(ctx, scope, p.ID, edited.Revision))
				next, err := rule.NextAfter(editAt)
				require.NoError(t, err)
				receipt, _, err = store.ClaimScheduledOccurrence(ctx, scope, p.ID, next, next)
				require.NoError(t, err)
				require.Equal(t, "CLAIMED", receipt.State)
			})
		}
	}
}

func TestScheduledRecoveryPassesTwentyUnresolvedPlans(t *testing.T) {
	db := workbenchDB(t)
	store, err := New(db)
	require.NoError(t, err)
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Second)
	scope := aiworkbench.Scope{OrganizationID: "org-a", ActorID: "alice"}
	pending := ""
	for n := 1; n <= 21; n++ {
		p := aiworkbench.ScheduledPlanDraft{ID: fmt.Sprintf("00000000-0000-4000-8000-%012d", n), Scope: scope, MemberID: "member", GoalSummary: "Localize", Selection: aiworkbench.TaskSelection{Subject: aiworkbench.Subject{Kind: "own_product", ID: uuid.NewString(), CatalogVersion: "1", PublicationID: uuid.NewString()}, Capability: aiworkbench.ContentTaskKind, Parameters: reviewproposal.ContentSelection{TargetLanguage: "en-US", Title: true}}, Rule: aiworkbench.ScheduleRule{Frequency: aiworkbench.ScheduleOnce, TimeZone: "UTC", AtUTC: now.Add(time.Minute)}, Budget: aiworkbench.ScheduleBudget{MaxOccurrences: 1, PerRunUpperBoundMicros: 1}, CreatedAt: now}
		require.NoError(t, store.CreateScheduledPlan(ctx, p))
		if n == 21 {
			pending = p.ID
			continue
		}
		require.NoError(t, store.MarkScheduleRegistered(ctx, scope, p.ID, 1))
		_, _, err := store.ClaimScheduledOccurrence(ctx, scope, p.ID, p.Rule.AtUTC, p.Rule.AtUTC)
		require.NoError(t, err)
	}
	first, next, err := store.DueScheduledRecovery(ctx, "", 20)
	require.NoError(t, err)
	require.Len(t, first, 20)
	require.NotEmpty(t, next)
	second, _, err := store.DueScheduledRecovery(ctx, next, 20)
	require.NoError(t, err)
	require.Len(t, second, 1)
	require.Equal(t, pending, second[0].ID)
	current, err := store.ReadScheduledPlan(ctx, scope, first[0].ID)
	require.NoError(t, err)
	require.NotEmpty(t, current.ActiveOccurrenceID, "fairness never releases UNKNOWN occupancy")
}

func TestScheduledControlReplayAfterEditedSlotExpires(t *testing.T) {
	db := workbenchDB(t)
	store, err := New(db)
	require.NoError(t, err)
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Second)
	scope := aiworkbench.Scope{OrganizationID: "org-a", ActorID: "alice"}
	p := aiworkbench.ScheduledPlanDraft{ID: uuid.NewString(), Scope: scope, MemberID: "member", GoalSummary: "Localize", Selection: aiworkbench.TaskSelection{Subject: aiworkbench.Subject{Kind: "own_product", ID: uuid.NewString(), CatalogVersion: "1", PublicationID: uuid.NewString()}, Capability: aiworkbench.ContentTaskKind, Parameters: reviewproposal.ContentSelection{TargetLanguage: "en-US", Title: true}}, Rule: aiworkbench.ScheduleRule{Frequency: aiworkbench.ScheduleOnce, TimeZone: "UTC", AtUTC: now.Add(time.Hour)}, Budget: aiworkbench.ScheduleBudget{MaxOccurrences: 1, PerRunUpperBoundMicros: 1}, CreatedAt: now}
	require.NoError(t, store.CreateScheduledPlan(ctx, p))
	require.NoError(t, store.MarkScheduleRegistered(ctx, scope, p.ID, 1))
	rule := p.Rule
	rule.AtUTC = now.Add(2 * time.Hour)
	key := uuid.NewString()
	edited, err := store.ChangeScheduledPlan(ctx, scope, p.ID, key, 2, "edit", &rule, "", now)
	require.NoError(t, err)
	replayed, err := store.ChangeScheduledPlan(ctx, scope, p.ID, key, 2, "edit", &rule, "", now.Add(3*time.Hour))
	require.NoError(t, err)
	require.Equal(t, edited.Revision, replayed.Revision)
	rule.AtUTC = now.Add(4 * time.Hour)
	_, err = store.ChangeScheduledPlan(ctx, scope, p.ID, key, 2, "edit", &rule, "", now.Add(3*time.Hour))
	require.ErrorIs(t, err, aiworkbench.ErrIdempotencyConflict)
}

func TestScheduledRecoveryFindsUnreceivedOneTimePlan(t *testing.T) {
	db := workbenchDB(t)
	store, err := New(db)
	require.NoError(t, err)
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Second)
	scope := aiworkbench.Scope{OrganizationID: "org-a", ActorID: "alice"}
	p := aiworkbench.ScheduledPlanDraft{ID: uuid.NewString(), Scope: scope, MemberID: "member", GoalSummary: "Localize", Selection: aiworkbench.TaskSelection{Subject: aiworkbench.Subject{Kind: "own_product", ID: uuid.NewString(), CatalogVersion: "1", PublicationID: uuid.NewString()}, Capability: aiworkbench.ContentTaskKind, Parameters: reviewproposal.ContentSelection{TargetLanguage: "en-US", Title: true}}, Rule: aiworkbench.ScheduleRule{Frequency: aiworkbench.ScheduleOnce, TimeZone: "UTC", AtUTC: now.Add(time.Minute)}, Budget: aiworkbench.ScheduleBudget{MaxOccurrences: 1, PerRunUpperBoundMicros: 1}, CreatedAt: now}
	require.NoError(t, store.CreateScheduledPlan(ctx, p))
	require.NoError(t, store.MarkScheduleRegistered(ctx, scope, p.ID, 1))
	plans, _, err := store.DueScheduledRecovery(ctx, "", 20)
	require.NoError(t, err)
	require.Len(t, plans, 1, "matched and unoccupied cannot hide an unreceived slot")
	withinWindow, err := store.RecordExpiredScheduledSlots(ctx, scope, p.ID, p.Rule.AtUTC.Add(14*time.Minute))
	require.NoError(t, err)
	require.Equal(t, "ACTIVE", withinWindow.State)
	_, err = store.ReadScheduledOccurrence(ctx, scope, p.ID, p.Rule.AtUTC)
	require.ErrorIs(t, err, aiworkbench.ErrNotFound)
	current, err := store.RecordExpiredScheduledSlots(ctx, scope, p.ID, p.Rule.AtUTC.Add(16*time.Minute))
	require.NoError(t, err)
	require.Equal(t, "CLOSED", current.State)
	require.Zero(t, current.ConsumedOccurrences)
	require.Empty(t, current.ActiveOccurrenceID)
	receipt, err := store.ReadScheduledOccurrence(ctx, scope, p.ID, p.Rule.AtUTC)
	require.NoError(t, err)
	require.Equal(t, "SKIPPED", receipt.State)
	require.Equal(t, "CATCHUP_EXPIRED", receipt.Reason)
	require.Empty(t, receipt.BusinessTaskID)
	repeated, err := store.RecordExpiredScheduledSlots(ctx, scope, p.ID, p.Rule.AtUTC.Add(time.Hour))
	require.NoError(t, err)
	require.Equal(t, current.Revision, repeated.Revision)
	var count int64
	require.NoError(t, db.Model(&scheduledOccurrenceRow{}).Count(&count).Error)
	require.EqualValues(t, 1, count)
}

func TestScheduledPreflightPreservesEarlierMissingHistory(t *testing.T) {
	for _, gap := range []int{2, 300} {
		t.Run(fmt.Sprint(gap), func(t *testing.T) {
			db := workbenchDB(t)
			store, err := New(db)
			require.NoError(t, err)
			ctx := context.Background()
			now := time.Date(2026, 1, 1, 8, 0, 0, 0, time.UTC)
			scope := aiworkbench.Scope{OrganizationID: "org", ActorID: "alice"}
			p := aiworkbench.ScheduledPlanDraft{ID: uuid.NewString(), Scope: scope, MemberID: "member", GoalSummary: "Localize", Selection: aiworkbench.TaskSelection{Subject: aiworkbench.Subject{Kind: "own_product", ID: uuid.NewString(), CatalogVersion: "1", PublicationID: uuid.NewString()}, Capability: aiworkbench.ContentTaskKind, Parameters: reviewproposal.ContentSelection{TargetLanguage: "en-US", Title: true}}, Rule: aiworkbench.ScheduleRule{Frequency: aiworkbench.ScheduleDaily, TimeZone: "UTC", Hour: 9}, Budget: aiworkbench.ScheduleBudget{MaxOccurrences: 3, PerRunUpperBoundMicros: 1}, CreatedAt: now}
			require.NoError(t, store.CreateScheduledPlan(ctx, p))
			require.NoError(t, store.MarkScheduleRegistered(ctx, scope, p.ID, 1))
			slot := now.AddDate(0, 0, gap).Add(time.Hour)
			err = store.RecordScheduledPreflightReceipt(ctx, scope, p.ID, slot, slot, "BLOCKED", "CURRENT_AUTHORIZATION_UNAVAILABLE")
			if gap > 256 {
				require.ErrorIs(t, err, aiworkbench.ErrScheduleCatchupPending)
				_, err = store.ReadScheduledOccurrence(ctx, scope, p.ID, slot)
				require.ErrorIs(t, err, aiworkbench.ErrNotFound)
				err = store.RecordScheduledPreflightReceipt(ctx, scope, p.ID, slot, slot, "BLOCKED", "CURRENT_AUTHORIZATION_UNAVAILABLE")
			}
			require.NoError(t, err)
			var count int64
			require.NoError(t, db.Model(&scheduledOccurrenceRow{}).Count(&count).Error)
			require.EqualValues(t, gap+1, count)
			first, err := store.ReadScheduledOccurrence(ctx, scope, p.ID, now.Add(time.Hour))
			require.NoError(t, err)
			require.Equal(t, "SKIPPED", first.State)
			last, err := store.ReadScheduledOccurrence(ctx, scope, p.ID, slot)
			require.NoError(t, err)
			require.Equal(t, "BLOCKED", last.State)
			current, err := store.ReadScheduledPlan(ctx, scope, p.ID)
			require.NoError(t, err)
			require.Zero(t, current.ConsumedOccurrences)
			require.Empty(t, current.ActiveOccurrenceID)
		})
	}
}

func TestScheduledRejectedCreateSerializesWithOriginalCommit(t *testing.T) {
	for _, rejectFirst := range []bool{false, true} {
		t.Run(fmt.Sprint(rejectFirst), func(t *testing.T) {
			db := workbenchDB(t)
			store, err := New(db)
			require.NoError(t, err)
			ctx := context.Background()
			now := time.Now().UTC().Truncate(time.Second)
			scope := aiworkbench.Scope{OrganizationID: "org", ActorID: "alice"}
			key := uuid.NewString()
			p := aiworkbench.ScheduledPlanDraft{ID: uuid.NewString(), Scope: scope, MemberID: "member", GoalSummary: "Localize", CreationFingerprint: strings.Repeat("a", 64), Selection: aiworkbench.TaskSelection{Subject: aiworkbench.Subject{Kind: "own_product", ID: uuid.NewString(), CatalogVersion: "1", PublicationID: uuid.NewString()}, Capability: aiworkbench.ContentTaskKind, Parameters: reviewproposal.ContentSelection{TargetLanguage: "en-US", Title: true}}, Rule: aiworkbench.ScheduleRule{Frequency: aiworkbench.ScheduleOnce, TimeZone: "UTC", AtUTC: now.Add(time.Hour)}, Budget: aiworkbench.ScheduleBudget{MaxOccurrences: 1, PerRunUpperBoundMicros: 1}, CreatedAt: now}
			if !rejectFirst {
				require.NoError(t, store.CreateScheduledPlanForCommand(ctx, p, key))
			}
			result, rejected, err := store.RejectScheduledCreate(ctx, scope, p.ID, key, p.CreationFingerprint)
			require.NoError(t, err)
			require.Equal(t, rejectFirst, rejected)
			if rejectFirst {
				require.Empty(t, result.ID)
				require.ErrorIs(t, store.CreateScheduledPlanForCommand(ctx, p, key), aiworkbench.ErrScheduleCommandRejected)
			} else {
				require.Equal(t, p.ID, result.ID)
				require.NoError(t, store.CreateScheduledPlanForCommand(ctx, p, key))
			}
			var count int64
			require.NoError(t, db.Model(&scheduledPlanRow{}).Count(&count).Error)
			if rejectFirst {
				require.Zero(t, count)
			} else {
				require.EqualValues(t, 1, count)
			}
			invalidOutcome := "REJECTED"
			if rejectFirst {
				invalidOutcome = "COMPLETED"
			}
			require.Error(t, db.Model(&scheduledControlRow{}).Where("organization_id=? AND owner_user_id=? AND idempotency_key=?", scope.OrganizationID, scope.ActorID, key).Update("outcome", invalidOutcome).Error, "mixed successful/rejected receipt facts must be rejected by PostgreSQL")
			_, _, err = store.RejectScheduledCreate(ctx, scope, p.ID, key, strings.Repeat("b", 64))
			require.ErrorIs(t, err, aiworkbench.ErrIdempotencyConflict)
		})
	}
}

func TestScheduledCreateAndRejectionUseSameCommitLock(t *testing.T) {
	for _, rejectFirst := range []bool{false, true} {
		t.Run(fmt.Sprint(rejectFirst), func(t *testing.T) {
			db := workbenchDB(t)
			store, err := New(db)
			require.NoError(t, err)
			ctx := context.Background()
			now := time.Now().UTC().Truncate(time.Second)
			scope := aiworkbench.Scope{OrganizationID: "org", ActorID: "alice"}
			key := uuid.NewString()
			p := aiworkbench.ScheduledPlanDraft{ID: uuid.NewString(), Scope: scope, MemberID: "member", GoalSummary: "Localize", CreationFingerprint: strings.Repeat("a", 64), Selection: aiworkbench.TaskSelection{Subject: aiworkbench.Subject{Kind: "own_product", ID: uuid.NewString(), CatalogVersion: "1", PublicationID: uuid.NewString()}, Capability: aiworkbench.ContentTaskKind, Parameters: reviewproposal.ContentSelection{TargetLanguage: "en-US", Title: true}}, Rule: aiworkbench.ScheduleRule{Frequency: aiworkbench.ScheduleOnce, TimeZone: "UTC", AtUTC: now.Add(time.Hour)}, Budget: aiworkbench.ScheduleBudget{MaxOccurrences: 1, PerRunUpperBoundMicros: 1}, CreatedAt: now}
			held := db.Begin()
			require.NoError(t, held.Error)
			defer held.Rollback()
			owner := &Store{db: held}
			type result struct {
				plan     aiworkbench.ScheduledPlan
				rejected bool
				err      error
			}
			done := make(chan result, 1)
			if rejectFirst {
				_, rejected, err := owner.RejectScheduledCreate(ctx, scope, p.ID, key, p.CreationFingerprint)
				require.NoError(t, err)
				require.True(t, rejected)
				go func() { done <- result{err: store.CreateScheduledPlanForCommand(ctx, p, key)} }()
			} else {
				require.NoError(t, owner.CreateScheduledPlanForCommand(ctx, p, key))
				go func() {
					plan, rejected, err := store.RejectScheduledCreate(ctx, scope, p.ID, key, p.CreationFingerprint)
					done <- result{plan: plan, rejected: rejected, err: err}
				}()
			}
			select {
			case early := <-done:
				t.Fatalf("competing decision escaped commit lock: %+v", early)
			case <-time.After(100 * time.Millisecond):
			}
			require.NoError(t, held.Commit().Error)
			select {
			case settled := <-done:
				if rejectFirst {
					require.ErrorIs(t, settled.err, aiworkbench.ErrScheduleCommandRejected)
				} else {
					require.NoError(t, settled.err)
					require.False(t, settled.rejected)
					require.Equal(t, p.ID, settled.plan.ID)
				}
			case <-time.After(5 * time.Second):
				t.Fatal("commit decision did not recover")
			}
		})
	}
}

func TestScheduledScopedStoreIndexPreservesOriginalMemberAndCursor(t *testing.T) {
	db := workbenchDB(t)
	store, err := New(db)
	require.NoError(t, err)
	ctx := context.Background()
	owner := aiworkbench.Scope{OrganizationID: "org-a", ActorID: "alice"}
	now := time.Now().UTC().Truncate(time.Second)
	selection := aiworkbench.ScopeSelection{Kind: "store", StoreID: "shop"}
	fixed := aiworkbench.BusinessScope{Version: aiworkbench.BusinessScopeVersion, Kind: "store", StoreID: "shop"}
	conversation, _, err := store.CreateScoped(ctx, owner, uuid.NewString(), aiworkbench.CreateInput{Selection: &selection, MemberID: "member-a"}, &fixed, nil)
	require.NoError(t, err)
	command, _, err := store.AppendUser(ctx, owner, conversation.ID, uuid.NewString(), aiworkbench.MessageInput{Content: "Localize", OperationID: uuid.NewString(), TargetPlatform: "shein"}, func(history []aiworkbench.Message, invocation string) (aiworkbench.PreparedPlan, error) {
		prepared, err := testPlanPreparer(history, invocation)
		prepared.FrozenScope = &aiworkbench.FrozenScope{Owner: owner, ConversationID: conversation.ID, MemberID: "member-a", Scope: fixed, StoreVersion: 1, ModelInput: aiworkbench.ScopeModelInput{StoreName: "Fixture shop"}}
		return prepared, err
	})
	require.NoError(t, err)
	ref := command.ScopeRef()
	require.True(t, ref.Valid())
	chosen := aiworkbench.TaskSelection{Subject: aiworkbench.Subject{Kind: "own_product", ID: uuid.NewString(), CatalogVersion: "1", PublicationID: uuid.NewString()}, Capability: aiworkbench.ContentTaskKind, Parameters: reviewproposal.ContentSelection{TargetLanguage: "en-US", Description: true}}
	proposal := aiworkbench.ExecutionProposal{WorkbenchContextRef: ref, Kind: aiworkbench.ContentTaskKind, Scope: owner, GoalSummary: "Localize", ContextKind: "own_product", OperationID: chosen.Subject.ID, ProductKey: "product", CatalogVersion: "1", PublicationID: chosen.Subject.PublicationID, TargetPlatform: "product", AgentID: agentconfig.ContentAgentID, AgentVersion: agentconfig.ContentAgentVersion, ContentSelection: chosen.Parameters, ExecutionLimits: agent.Limits{Steps: 3, ModelCalls: 2, Tokens: 1000, CostMicros: 2500, Currency: "USD", Runtime: time.Minute}}
	spec, err := json.Marshal(proposal)
	require.NoError(t, err)
	slot := now.Add(time.Minute)
	ids := []string{}
	for n := 0; n < 2; n++ {
		plan := aiworkbench.ScheduledPlanDraft{ID: uuid.NewString(), Scope: owner, MemberID: "member-a", Selection: chosen, GoalSummary: proposal.GoalSummary, Rule: aiworkbench.ScheduleRule{Frequency: aiworkbench.ScheduleOnce, TimeZone: "UTC", AtUTC: slot}, Budget: aiworkbench.ScheduleBudget{MaxOccurrences: 1, PerRunUpperBoundMicros: 1}, CreatedAt: now, ExecutionSpec: spec}
		require.NoError(t, store.CreateScheduledPlan(ctx, plan))
		require.NoError(t, store.MarkScheduleRegistered(ctx, owner, plan.ID, 1))
		key := aiworkbench.ScheduledOccurrenceID(owner, plan.ID, slot)
		configRef := agent.ConfigurationSnapshotRef{Kind: agentconfig.SnapshotKind, ID: uuid.NewString(), Digest: strings.Repeat("a", 64)}
		request := agent.Request{Key: key, Binding: agent.Binding{ContextKind: "own_product", ContextID: proposal.OperationID, ProductKey: proposal.ProductKey, CatalogVersion: "1", PublicationID: proposal.PublicationID, TargetPlatform: "product"}, GoalSummary: proposal.GoalSummary, ContentSelection: proposal.ContentSelection, PolicyVersion: reviewproposal.ContentPolicyVersion, PromptVersion: agentconfig.ScopedContentPromptVersion, Limits: proposal.ExecutionLimits, ManualReviewOnly: true, ScheduledSource: agent.ScheduledSourceRef{PlanID: plan.ID, OccurrenceID: key}, ConfigurationSnapshotRef: configRef, WorkbenchContextRef: ref}
		raw, err := json.Marshal(request)
		require.NoError(t, err)
		_, task, _, err := store.ClaimScheduledTask(ctx, owner, plan.ID, 2, slot, slot, aiworkbench.PreparedTask{ExecutionRequest: raw, ConfigurationSnapshotRef: configRef})
		require.NoError(t, err)
		ids = append(ids, task.ID)
		require.Empty(t, task.ConversationID)
	}
	visible := func(scope *aiworkbench.BusinessScope, task aiworkbench.BusinessTask) bool {
		require.Equal(t, &fixed, scope)
		require.Equal(t, ref, task.WorkbenchContextRef)
		return true
	}
	filter := aiworkbench.ScopeSelection{Kind: "store", StoreID: "shop"}
	first, err := store.ListScopedTasks(ctx, owner, filter, "", 1, visible)
	require.NoError(t, err)
	require.Len(t, first.Tasks, 1)
	require.NotEmpty(t, first.Next)
	second, err := store.ListScopedTasks(ctx, owner, filter, first.Next, 1, visible)
	require.NoError(t, err)
	require.Len(t, second.Tasks, 1)
	require.Empty(t, second.Next)
	require.ElementsMatch(t, ids, []string{first.Tasks[0].ID, second.Tasks[0].ID})
	tasks, err := store.ListNewStoreTasks(ctx, owner, "member-a", []string{"shop"}, now, slot.Add(time.Second), 10)
	require.NoError(t, err)
	require.Len(t, tasks, 2)
	for _, member := range []string{"other-member"} {
		tasks, err = store.ListNewStoreTasks(ctx, owner, member, []string{"shop"}, now, slot.Add(time.Second), 10)
		require.NoError(t, err)
		require.Empty(t, tasks)
	}
	tasks, err = store.ListNewStoreTasks(ctx, aiworkbench.Scope{OrganizationID: "other-org", ActorID: owner.ActorID}, "member-a", []string{"shop"}, now, slot.Add(time.Second), 10)
	require.NoError(t, err)
	require.Empty(t, tasks)
	tasks, err = store.ListNewStoreTasks(ctx, owner, "member-a", []string{"other-shop"}, now, slot.Add(time.Second), 10)
	require.NoError(t, err)
	require.Empty(t, tasks)
	_, err = store.ListScopedTasks(ctx, owner, filter, first.Next, 1, func(*aiworkbench.BusinessScope, aiworkbench.BusinessTask) bool { return false })
	require.ErrorIs(t, err, aiworkbench.ErrInvalid, "revoked cursor anchors cannot expose scheduled records")
}

func TestScheduledRejectionCommitLossDoesNotProveNonacceptance(t *testing.T) {
	db := workbenchDB(t)
	ctx := context.Background()
	scope := aiworkbench.Scope{OrganizationID: "org", ActorID: "alice"}
	id, key := uuid.NewString(), uuid.NewString()
	var commits atomic.Int32
	err := db.Connection(func(tx *gorm.DB) error {
		conn, ok := tx.Statement.ConnPool.(*sql.Conn)
		require.True(t, ok)
		tx.Statement.ConnPool = lostCommitPool{conn: conn, commits: &commits}
		_, rejected, err := (&Store{db: tx}).RejectScheduledCreate(ctx, scope, id, key, strings.Repeat("a", 64))
		require.False(t, rejected, "lost commit acknowledgement cannot expose a definitive negative proof")
		return err
	})
	require.Error(t, err)
	require.EqualValues(t, 1, commits.Load())
	store, err := New(db)
	require.NoError(t, err)
	_, _, err = store.ReadScheduledCreateReceipt(ctx, scope, id, key, strings.Repeat("a", 64))
	require.ErrorIs(t, err, aiworkbench.ErrScheduleCommandRejected, "only a subsequent durable read may prove rejection")
}
