//go:build integration

package aiworkbenchpersistence

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
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
	paused, err := store.ChangeScheduledPlan(ctx, scope, plan.ID, controlKey, 2, "pause", nil, now)
	require.NoError(t, err)
	require.Equal(t, "PAUSED", paused.State)
	require.Equal(t, occurrence.ID, paused.ActiveOccurrenceID, "pause only controls future admission")
	_, err = store.ChangeScheduledPlan(ctx, scope, plan.ID, controlKey, 2, "close", nil, now)
	require.ErrorIs(t, err, aiworkbench.ErrIdempotencyConflict)
	repeated, err := store.ChangeScheduledPlan(ctx, scope, plan.ID, controlKey, 2, "pause", nil, now)
	require.NoError(t, err)
	require.Equal(t, paused.Revision, repeated.Revision)
	_, err = store.ChangeScheduledPlan(ctx, scope, plan.ID, uuid.NewString(), 2, "resume", nil, now)
	require.ErrorIs(t, err, aiworkbench.ErrRevisionMismatch)
	closed, err := store.ChangeScheduledPlan(ctx, scope, plan.ID, uuid.NewString(), paused.Revision, "close", nil, now)
	require.NoError(t, err)
	_, err = store.ChangeScheduledPlan(ctx, scope, plan.ID, uuid.NewString(), closed.Revision, "resume", nil, now)
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
	paused, err := store.ChangeScheduledPlan(ctx, scope, p.ID, uuid.NewString(), 2, "pause", nil, now)
	require.NoError(t, err)
	require.NoError(t, store.RecordScheduledPreflightReceipt(ctx, scope, p.ID, p.Rule.AtUTC, p.Rule.AtUTC, "SKIPPED", "PLAN_NOT_ACTIVE"))
	_, err = store.ChangeScheduledPlan(ctx, scope, p.ID, uuid.NewString(), paused.Revision, "resume", nil, now)
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
