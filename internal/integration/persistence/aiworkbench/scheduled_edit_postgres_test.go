//go:build integration

package aiworkbenchpersistence

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/agent"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/agentconfig"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/aiworkbench"
	reviewproposal "github.com/henryxu1008/shuomi-ai-commerce/internal/product/review/proposal"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestScheduledFullEditKeepsInFlightAndCopiesOnce(t *testing.T) {
	db := workbenchDB(t)
	store, e := New(db)
	require.NoError(t, e)
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Second)
	scope := aiworkbench.Scope{OrganizationID: "edit-org", ActorID: "alice"}
	selection := aiworkbench.TaskSelection{Subject: aiworkbench.Subject{Kind: "own_product", ID: uuid.NewString(), CatalogVersion: "1", PublicationID: uuid.NewString()}, Capability: aiworkbench.ContentTaskKind, Parameters: reviewproposal.ContentSelection{TargetLanguage: "en-US", Description: true}}
	p := aiworkbench.ExecutionProposal{Kind: aiworkbench.ContentTaskKind, Scope: scope, GoalSummary: "Original content", ContextKind: "own_product", OperationID: selection.Subject.ID, ProductKey: "edit-product", CatalogVersion: "1", PublicationID: selection.Subject.PublicationID, TargetPlatform: "product", AgentID: agentconfig.ContentAgentID, AgentVersion: agentconfig.ContentAgentVersion, ContentSelection: selection.Parameters, ExecutionLimits: agent.Limits{Steps: 3, ModelCalls: 2, Tokens: 1000, CostMicros: 2500, Currency: "USD", Runtime: time.Minute}}
	spec, e := json.Marshal(p)
	require.NoError(t, e)
	draft := aiworkbench.ScheduledPlanDraft{ID: uuid.NewString(), Scope: scope, MemberID: "member", GoalSummary: p.GoalSummary, Selection: selection, ExecutionSpec: spec, Rule: aiworkbench.ScheduleRule{Frequency: aiworkbench.ScheduleOnce, TimeZone: "UTC", AtUTC: now.Add(time.Minute)}, Budget: aiworkbench.ScheduleBudget{Unit: "ai-micropoint-v1", MaxOccurrences: 1, PerRunUpperBoundMicros: 100}, CreatedAt: now}
	require.NoError(t, store.CreateScheduledPlan(ctx, draft))
	require.NoError(t, store.MarkScheduleRegistered(ctx, scope, draft.ID, 1))
	key := aiworkbench.ScheduledOccurrenceID(scope, draft.ID, draft.Rule.AtUTC)
	ref := agent.ConfigurationSnapshotRef{Kind: agentconfig.SnapshotKind, ID: uuid.NewString(), Digest: strings.Repeat("a", 64)}
	request := agent.Request{Key: key, Binding: agent.Binding{ContextKind: "own_product", ContextID: p.OperationID, ProductKey: p.ProductKey, CatalogVersion: "1", PublicationID: p.PublicationID, TargetPlatform: "product"}, GoalSummary: p.GoalSummary, ContentSelection: p.ContentSelection, PolicyVersion: reviewproposal.ContentPolicyVersion, PromptVersion: agentconfig.ContentPromptVersion, Limits: p.ExecutionLimits, ManualReviewOnly: true, ConfigurationSnapshotRef: ref, ScheduledSource: agent.ScheduledSourceRef{PlanID: draft.ID, OccurrenceID: key}}
	wire, e := json.Marshal(request)
	require.NoError(t, e)
	prepared := aiworkbench.PreparedTask{ExecutionRequest: wire, ConfigurationSnapshotRef: ref}
	occurrence, task, _, e := store.ClaimScheduledTask(ctx, scope, draft.ID, 2, draft.Rule.AtUTC, draft.Rule.AtUTC, prepared)
	require.NoError(t, e)
	before, e := store.ReadScheduledPlan(ctx, scope, draft.ID)
	require.NoError(t, e)
	desired := before.ScheduledPlanDraft
	desired.CopySource = nil
	desired.Rule = aiworkbench.ScheduleRule{Frequency: aiworkbench.ScheduleDaily, TimeZone: "UTC", Hour: 9}
	desired.Budget.MaxOccurrences = 3
	desired.Budget.PerRunUpperBoundMicros = 200
	desired.GoalSummary = "Future French content"
	desired.Selection.Parameters.TargetLanguage = "fr-FR"
	p.GoalSummary = desired.GoalSummary
	p.ContentSelection = desired.Selection.Parameters
	desired.ExecutionSpec, e = json.Marshal(p)
	require.NoError(t, e)
	replacement := aiworkbench.ScheduledPlanReplacement{Draft: desired, ConsumedOccurrences: 1}
	require.NoError(t, db.Exec(`CREATE ROLE ai_workbench_runtime NOLOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE NOREPLICATION NOBYPASSRLS`).Error)
	require.NoError(t, GrantRuntime(db, "ai_workbench_runtime"))
	commandKey := uuid.NewString()
	confirmation := strings.Repeat("b", 64)
	var edited aiworkbench.ScheduledPlan
	require.NoError(t, db.Transaction(func(tx *gorm.DB) error {
		if e := tx.Exec(`SET LOCAL ROLE ai_workbench_runtime`).Error; e != nil {
			return e
		}
		edited, e = (&Store{db: tx}).ChangeScheduledPlan(ctx, scope, draft.ID, commandKey, before.Revision, "edit", &desired.Rule, confirmation, draft.Rule.AtUTC.Add(time.Second), replacement)
		return e
	}))
	require.Equal(t, occurrence.ID, edited.ActiveOccurrenceID)
	require.EqualValues(t, 1, edited.ConsumedOccurrences)
	require.EqualValues(t, 3, edited.Budget.MaxOccurrences)
	original, e := store.GetTask(ctx, scope, task.ID)
	require.NoError(t, e)
	require.Equal(t, task.ExecutionRequest, original.ExecutionRequest)
	_, same, replay, e := store.ClaimScheduledTask(ctx, scope, draft.ID, 2, draft.Rule.AtUTC, draft.Rule.AtUTC, prepared)
	require.NoError(t, e)
	require.True(t, replay)
	require.Equal(t, task.ID, same.ID)
	require.NoError(t, store.RecordScheduledPreflightReceipt(ctx, scope, draft.ID, 2, draft.Rule.AtUTC, draft.Rule.AtUTC, "BLOCKED", "FROZEN_EXECUTION_UNAVAILABLE"))
	slot, e := desired.Rule.NextAfter(draft.Rule.AtUTC.Add(time.Second))
	require.NoError(t, e)
	require.ErrorIs(t, store.RecordScheduledPreflightReceipt(ctx, scope, draft.ID, 2, slot, slot, "BLOCKED", "FROZEN_EXECUTION_UNAVAILABLE"), aiworkbench.ErrRevisionMismatch)
	_, e = store.ReadScheduledOccurrence(ctx, scope, draft.ID, slot)
	require.ErrorIs(t, e, aiworkbench.ErrNotFound)
	stale := replacement
	stale.ConsumedOccurrences = 0
	_, e = store.ChangeScheduledPlan(ctx, scope, draft.ID, uuid.NewString(), edited.Revision, "edit", &desired.Rule, confirmation, now, stale)
	require.ErrorIs(t, e, aiworkbench.ErrRevisionMismatch)
	tooFew := replacement
	tooFew.Draft.Budget.MaxOccurrences = 1
	_, e = store.ChangeScheduledPlan(ctx, scope, draft.ID, uuid.NewString(), edited.Revision, "edit", &desired.Rule, confirmation, now, tooFew)
	require.ErrorIs(t, e, aiworkbench.ErrRevisionMismatch)
	paused, e := store.ChangeScheduledPlan(ctx, scope, draft.ID, uuid.NewString(), edited.Revision, "pause", nil, "", now)
	require.NoError(t, e)
	again, e := store.ChangeScheduledPlan(ctx, scope, draft.ID, uuid.NewString(), paused.Revision, "edit", &desired.Rule, strings.Repeat("c", 64), now, replacement)
	require.NoError(t, e)
	require.Equal(t, "PAUSED", again.State)
	recovered, e := store.ChangeScheduledPlan(ctx, scope, draft.ID, commandKey, before.Revision, "edit", &desired.Rule, confirmation, now, replacement)
	require.NoError(t, e)
	require.Equal(t, again.Revision, recovered.Revision)
	copyDraft := desired
	copyDraft.ID = uuid.NewString()
	copyDraft.Rule = aiworkbench.ScheduleRule{Frequency: aiworkbench.ScheduleOnce, TimeZone: "UTC", AtUTC: now.Add(time.Hour)}
	copyDraft.Budget.MaxOccurrences = 1
	copyDraft.CreationFingerprint = strings.Repeat("d", 64)
	copyDraft.CopySource = &aiworkbench.ScheduledPlanCopySource{ID: again.ID, Revision: edited.Revision, Digest: aiworkbench.ScheduledPlanSourceDigest(edited.ScheduledPlanDraft)}
	copyKey := uuid.NewString()
	require.ErrorIs(t, store.CreateScheduledPlanForCommand(ctx, copyDraft, copyKey), aiworkbench.ErrRevisionMismatch)
	_, rejected, e := store.RejectScheduledCreate(ctx, scope, copyDraft.ID, copyKey, copyDraft.CreationFingerprint)
	require.NoError(t, e)
	require.True(t, rejected)
	copyDraft.CopySource = &aiworkbench.ScheduledPlanCopySource{ID: again.ID, Revision: again.Revision, Digest: aiworkbench.ScheduledPlanSourceDigest(again.ScheduledPlanDraft)}
	require.ErrorIs(t, store.CreateScheduledPlanForCommand(ctx, copyDraft, copyKey), aiworkbench.ErrScheduleCommandRejected)
	copyKey = uuid.NewString()
	errs := make(chan error, 6)
	var wait sync.WaitGroup
	for i := 0; i < 6; i++ {
		wait.Add(1)
		go func() { defer wait.Done(); errs <- store.CreateScheduledPlanForCommand(ctx, copyDraft, copyKey) }()
	}
	wait.Wait()
	close(errs)
	for e := range errs {
		require.NoError(t, e)
	}
	copied, e := store.ReadScheduledPlan(ctx, scope, copyDraft.ID)
	require.NoError(t, e)
	require.Zero(t, copied.ConsumedOccurrences)
	require.Empty(t, copied.ActiveOccurrenceID)
	var count int64
	require.NoError(t, db.Model(&scheduledOccurrenceRow{}).Where("plan_id=?", copied.ID).Count(&count).Error)
	require.Zero(t, count)
	require.NoError(t, db.Model(&scheduledControlRow{}).Where("idempotency_key=?", copyKey).Count(&count).Error)
	require.EqualValues(t, 1, count)
	for _, column := range []string{"owner_user_id", "member_id", "kind", "creation_fingerprint", "budget_unit"} {
		err := db.Transaction(func(tx *gorm.DB) error {
			if e := tx.Exec(`SET LOCAL ROLE ai_workbench_runtime`).Error; e != nil {
				return e
			}
			return tx.Exec("UPDATE ai_workbench.scheduled_plans SET "+column+"="+column+" WHERE id=?", draft.ID).Error
		})
		require.Error(t, err, "runtime cannot update %s", column)
	}
	require.NoError(t, SetScheduledEditPermissions(db, "ai_workbench_runtime", false))
	require.NoError(t, SetScheduledEditPermissions(db, "ai_workbench_runtime", false))
	require.NoError(t, validateWorkbenchPrivilegesForEdit(db, "ai_workbench_runtime", false))
	require.Error(t, validateBoundedWorkbenchPrivileges(db, "ai_workbench_runtime"))
	require.NoError(t, db.Exec(`GRANT UPDATE(goal_summary) ON ai_workbench.scheduled_plans TO ai_workbench_runtime`).Error)
	require.Error(t, SetScheduledEditPermissions(db, "ai_workbench_runtime", true))
	require.NoError(t, db.Exec(`REVOKE UPDATE(goal_summary) ON ai_workbench.scheduled_plans FROM ai_workbench_runtime`).Error)
	require.NoError(t, SetScheduledEditPermissions(db, "ai_workbench_runtime", true))
	require.NoError(t, SetScheduledEditPermissions(db, "ai_workbench_runtime", true))
	t.Run("claim and edit serialize the same plan", func(t *testing.T) {
		raceDraft := draft
		raceDraft.ID = uuid.NewString()
		raceDraft.Rule.AtUTC = now.Add(2 * time.Minute)
		require.NoError(t, store.CreateScheduledPlan(ctx, raceDraft))
		require.NoError(t, store.MarkScheduleRegistered(ctx, scope, raceDraft.ID, 1))
		raceRequest := request
		raceRequest.Key = aiworkbench.ScheduledOccurrenceID(scope, raceDraft.ID, raceDraft.Rule.AtUTC)
		raceRequest.ScheduledSource = agent.ScheduledSourceRef{PlanID: raceDraft.ID, OccurrenceID: raceRequest.Key}
		raw, e := json.Marshal(raceRequest)
		require.NoError(t, e)
		r := replacement
		r.Draft.ID = raceDraft.ID
		r.ConsumedOccurrences = 0
		start := make(chan struct{})
		results := make(chan error, 2)
		go func() {
			<-start
			_, _, _, e := store.ClaimScheduledTask(ctx, scope, raceDraft.ID, 2, raceDraft.Rule.AtUTC, raceDraft.Rule.AtUTC, aiworkbench.PreparedTask{ExecutionRequest: raw, ConfigurationSnapshotRef: ref})
			results <- e
		}()
		go func() {
			<-start
			_, e := store.ChangeScheduledPlan(ctx, scope, raceDraft.ID, uuid.NewString(), 2, "edit", &r.Draft.Rule, confirmation, now, r)
			results <- e
		}()
		close(start)
		success := 0
		for i := 0; i < 2; i++ {
			e := <-results
			if e == nil {
				success++
			} else {
				require.ErrorIs(t, e, aiworkbench.ErrRevisionMismatch)
			}
		}
		require.Equal(t, 1, success)
	})
	t.Run("source deletion and copy commit serialize", func(t *testing.T) {
		candidate := copyDraft
		candidate.ID = uuid.NewString()
		candidate.CreationFingerprint = strings.Repeat("f", 64)
		start := make(chan struct{})
		copied := make(chan error, 1)
		deleted := make(chan error, 1)
		key := uuid.NewString()
		go func() { <-start; copied <- store.CreateScheduledPlanForCommand(ctx, candidate, key) }()
		go func() {
			<-start
			_, e := store.ChangeScheduledPlan(ctx, scope, again.ID, uuid.NewString(), again.Revision, "delete", nil, "", now)
			deleted <- e
		}()
		close(start)
		copyErr := <-copied
		require.NoError(t, <-deleted)
		if copyErr == nil {
			require.NoError(t, store.CreateScheduledPlanForCommand(ctx, candidate, key))
		} else {
			require.ErrorIs(t, copyErr, aiworkbench.ErrRevisionMismatch)
			_, e := store.ReadScheduledPlan(ctx, scope, candidate.ID)
			require.ErrorIs(t, e, aiworkbench.ErrNotFound)
		}
	})
	sentinel := errors.New("rollback partial capability fixture")
	require.ErrorIs(t, db.Transaction(func(tx *gorm.DB) error {
		if e := tx.Exec(`ALTER TABLE ai_workbench.scheduled_plans ADD COLUMN report_spec jsonb`).Error; e != nil {
			return e
		}
		require.Error(t, SetScheduledEditPermissions(tx, "ai_workbench_runtime", true))
		return sentinel
	}), sentinel)
	require.NoError(t, InstallDailyReportSchema(db))
	require.NoError(t, GrantDailyReportRuntime(db, "ai_workbench_runtime"))
	require.NoError(t, validateBoundedWorkbenchPrivileges(db, "ai_workbench_runtime"))
	require.NoError(t, SetScheduledEditPermissions(db, "ai_workbench_runtime", false))
	require.NoError(t, validateWorkbenchPrivilegesForEdit(db, "ai_workbench_runtime", false))
	require.NoError(t, SetScheduledEditPermissions(db, "ai_workbench_runtime", true))
}
