package aiworkbenchpersistence

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"task-processor/internal/agent"
	"task-processor/internal/aiworkbench"
)

// ClaimScheduledTask is the application dispatch transaction. Preparation and
// live authorization happen before entry; no external owner is called while
// the plan lock is held. A failed task insert rolls back the occurrence/count.
func (s *Store) ClaimScheduledTask(ctx context.Context, scope aiworkbench.Scope, planID string, revision int64, slot, now time.Time, prepared aiworkbench.PreparedTask) (aiworkbench.ScheduledOccurrence, aiworkbench.BusinessTask, bool, error) {
	if s == nil || s.db == nil || ctx == nil || !validScope(scope) || !validKey(planID) || revision < 1 {
		return aiworkbench.ScheduledOccurrence{}, aiworkbench.BusinessTask{}, false, aiworkbench.ErrInvalid
	}
	var occurrence aiworkbench.ScheduledOccurrence
	var result aiworkbench.BusinessTask
	var replay bool
	var catchupPending bool
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var row scheduledPlanRow
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id=? AND organization_id=? AND owner_user_id=?", planID, scope.OrganizationID, scope.ActorID).Take(&row).Error; err != nil {
			return missing(err)
		}
		// Replay a durable binding before testing a later plan revision. A new
		// revision cannot mint a second task for an already occupied UTC slot.
		var prior scheduledOccurrenceRow
		err := tx.Where("plan_id=? AND organization_id=? AND owner_user_id=? AND scheduled_at_utc=?", planID, scope.OrganizationID, scope.ActorID, slot.UTC()).Take(&prior).Error
		if err == nil {
			occurrence, replay = occurrenceReceipt(prior), true
			if prior.BusinessTaskID != nil {
				result, err = (&Store{db: tx}).GetTask(ctx, scope, *prior.BusinessTaskID)
				canonical, _, parseErr := canonicalRequest(prepared.ExecutionRequest)
				if err == nil && (parseErr != nil || digest(json.RawMessage(canonical)) != result.ExecutionRequestDigest) {
					return aiworkbench.ErrIdempotencyConflict
				}
			}
			return err
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return unavailable(err)
		}
		if row.Revision != revision {
			return aiworkbench.ErrRevisionMismatch
		}
		policy, err := schedulePlanValue(row)
		if err != nil {
			return err
		}
		var proposal aiworkbench.ExecutionProposal
		canonical, request, err := canonicalRequest(prepared.ExecutionRequest)
		if err != nil || len(canonical) > 8192 || json.Unmarshal(policy.ExecutionSpec, &proposal) != nil ||
			proposal.Kind != aiworkbench.ContentTaskKind || proposal.Scope != scope || proposal.GoalSummary != policy.GoalSummary ||
			proposal.Selection() == nil || *proposal.Selection() != policy.Selection || !proposal.MatchesRequest(request) ||
			request.Key != aiworkbench.ScheduledOccurrenceID(scope, planID, slot) || !request.ManualReviewOnly ||
			request.ScheduledSource != (agent.ScheduledSourceRef{PlanID: planID, OccurrenceID: request.Key}) ||
			request.GoalSummary != policy.GoalSummary || request.ConfigurationSnapshotRef != prepared.ConfigurationSnapshotRef ||
			request.ConfigurationSnapshotRef.Absent() || !request.ConfigurationSnapshotRef.ValidOrAbsent() ||
			request.ContextSnapshotRef != prepared.ContextSnapshotRef || !request.ContextSnapshotRef.ValidOrAbsent() ||
			request.WorkbenchContextRef != proposal.WorkbenchContextRef || !request.Limits.Valid() {
			return aiworkbench.ErrInvalid
		}
		inner := &Store{db: tx}
		occurrence, _, err = inner.ClaimScheduledOccurrence(ctx, scope, planID, slot, now)
		if errors.Is(err, aiworkbench.ErrScheduleCatchupPending) {
			catchupPending = true
			return nil // commit only the bounded SKIPPED batch; no paid task
		}
		if err != nil {
			return err
		}
		if occurrence.State != "CLAIMED" {
			return nil
		}
		configJSON, _ := json.Marshal(prepared.ConfigurationSnapshotRef)
		contextJSON, _ := json.Marshal(prepared.ContextSnapshotRef)
		taskValue := taskRow{ID: uuid.NewString(), OrganizationID: scope.OrganizationID, OwnerUserID: scope.ActorID,
			SourceKind: "SCHEDULE_OCCURRENCE", ScheduledOccurrenceID: &occurrence.ID,
			ProposalDigest: occurrence.PayloadDigest, ConfirmationFingerprint: occurrence.PayloadDigest,
			Kind: proposal.Kind, Title: initialTitle(policy.GoalSummary), GoalSummary: policy.GoalSummary,
			OperationID: proposal.OperationID, ProductKey: proposal.ProductKey, TargetPlatform: proposal.TargetPlatform,
			AgentID: proposal.AgentID, AgentVersion: proposal.AgentVersion, ExecutionRequestKey: request.Key,
			ConfigurationSnapshotRef: configJSON, ContextSnapshotRef: contextJSON,
			ExecutionRequestDigest: digest(json.RawMessage(canonical)), ExecutionRequest: canonical, CreatedAt: now.UTC()}
		if err := tx.Create(&taskValue).Error; err != nil {
			return unavailable(err)
		}
		if err := tx.Model(&scheduledOccurrenceRow{}).Where("id=? AND organization_id=? AND owner_user_id=? AND business_task_id IS NULL", occurrence.ID, scope.OrganizationID, scope.ActorID).
			Updates(map[string]any{"business_task_id": taskValue.ID, "updated_at": now.UTC()}).Error; err != nil {
			return unavailable(err)
		}
		result, err = task(taskValue)
		occurrence.BusinessTaskID = taskValue.ID
		return err
	})
	if err == nil && catchupPending {
		err = errors.Join(aiworkbench.ErrUnavailable, aiworkbench.ErrScheduleCatchupPending)
	}
	return occurrence, result, replay, err
}

func (s *Store) ReadScheduledPlan(ctx context.Context, scope aiworkbench.Scope, id string) (aiworkbench.ScheduledPlan, error) {
	if s == nil || s.db == nil || ctx == nil || !validScope(scope) || !validKey(id) {
		return aiworkbench.ScheduledPlan{}, aiworkbench.ErrInvalid
	}
	var row scheduledPlanRow
	if err := s.db.WithContext(ctx).Where("id=? AND organization_id=? AND owner_user_id=?", id, scope.OrganizationID, scope.ActorID).Take(&row).Error; err != nil {
		return aiworkbench.ScheduledPlan{}, missing(err)
	}
	return scheduledPlanRecord(row)
}

func scheduledPlanRecord(row scheduledPlanRow) (aiworkbench.ScheduledPlan, error) {
	draft, err := schedulePlanValue(row)
	value := aiworkbench.ScheduledPlan{ScheduledPlanDraft: draft, State: row.State, Revision: row.Revision, RegistrationState: row.RegistrationState, ConsumedOccurrences: uint64(row.ConsumedOccurrences)}
	if row.ActiveOccurrenceID != nil {
		value.ActiveOccurrenceID = *row.ActiveOccurrenceID
	}
	return value, err
}

// ReadScheduledExecutionPlan is a server-only worker lookup. Callers cannot
// obtain execution authority from its return value: the application rechecks
// the exact member, current plan, source and configuration at every send.
func (s *Store) ReadScheduledExecutionPlan(ctx context.Context, id string) (aiworkbench.ScheduledPlan, error) {
	if s == nil || s.db == nil || ctx == nil || !validKey(id) {
		return aiworkbench.ScheduledPlan{}, aiworkbench.ErrInvalid
	}
	var row scheduledPlanRow
	if err := s.db.WithContext(ctx).Where("id=?", id).Take(&row).Error; err != nil {
		return aiworkbench.ScheduledPlan{}, missing(err)
	}
	return scheduledPlanRecord(row)
}

func (s *Store) ReadScheduledOccurrence(ctx context.Context, scope aiworkbench.Scope, planID string, slot time.Time) (aiworkbench.ScheduledOccurrence, error) {
	if s == nil || s.db == nil || ctx == nil || !validScope(scope) || !validKey(planID) || slot.IsZero() {
		return aiworkbench.ScheduledOccurrence{}, aiworkbench.ErrInvalid
	}
	var row scheduledOccurrenceRow
	if err := s.db.WithContext(ctx).Where("organization_id=? AND owner_user_id=? AND plan_id=? AND scheduled_at_utc=?", scope.OrganizationID, scope.ActorID, planID, slot.UTC()).Take(&row).Error; err != nil {
		return aiworkbench.ScheduledOccurrence{}, missing(err)
	}
	return occurrenceReceipt(row), nil
}
