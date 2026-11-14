package aiworkbenchpersistence

import (
	"context"
	"errors"
	"gorm.io/gorm"
	"task-processor/internal/aiworkbench"
	"time"
)

func scheduledCreateDigest(id, fingerprint string) string {
	return digest(struct{ Action, Plan, Fingerprint string }{"create", id, fingerprint})
}

func (s *Store) readScheduledCommand(ctx context.Context, scope aiworkbench.Scope, id, key, fingerprint string) (aiworkbench.ScheduledPlan, bool, error) {
	var receipt scheduledControlRow
	err := s.db.WithContext(ctx).Where("organization_id=? AND owner_user_id=? AND idempotency_key=?", scope.OrganizationID, scope.ActorID, key).Take(&receipt).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return aiworkbench.ScheduledPlan{}, false, nil
	}
	if err != nil {
		return aiworkbench.ScheduledPlan{}, false, unavailable(err)
	}
	if receipt.RequestDigest != fingerprint {
		return aiworkbench.ScheduledPlan{}, false, aiworkbench.ErrIdempotencyConflict
	}
	if receipt.Outcome == "REJECTED" {
		return aiworkbench.ScheduledPlan{}, false, aiworkbench.ErrScheduleCommandRejected
	}
	if receipt.PlanID != id {
		return aiworkbench.ScheduledPlan{}, false, aiworkbench.ErrIdempotencyConflict
	}
	plan, err := s.ReadScheduledPlan(ctx, scope, id)
	return plan, err == nil, err
}

func (s *Store) ReadScheduledCreateReceipt(ctx context.Context, scope aiworkbench.Scope, id, key, fingerprint string) (aiworkbench.ScheduledPlan, bool, error) {
	if s == nil || s.db == nil || ctx == nil || !validScope(scope) || !validKey(id) || !validKey(key) || !validDigest(fingerprint) {
		return aiworkbench.ScheduledPlan{}, false, aiworkbench.ErrInvalid
	}
	return s.readScheduledCommand(ctx, scope, id, key, scheduledCreateDigest(id, fingerprint))
}

// Create and negative admission share one scope/key lock at final commit.
// No model, IAM or source owner call runs while this lock is held.
func (s *Store) CreateScheduledPlanForCommand(ctx context.Context, p aiworkbench.ScheduledPlanDraft, key string) error {
	if s == nil || s.db == nil || ctx == nil || !validKey(key) || !validScope(p.Scope) || !validKey(p.ID) || !validDigest(p.CreationFingerprint) {
		return aiworkbench.ErrInvalid
	}
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := s.lockTaskKey(tx, p.Scope, key); err != nil {
			return err
		}
		store := &Store{db: tx}
		_, found, err := store.ReadScheduledCreateReceipt(ctx, p.Scope, p.ID, key, p.CreationFingerprint)
		if err != nil || found {
			return err
		}
		if err = store.CreateScheduledPlan(ctx, p); err != nil {
			return err
		}
		return unavailable(tx.Create(&scheduledControlRow{OrganizationID: p.Scope.OrganizationID, OwnerUserID: p.Scope.ActorID, IdempotencyKey: key, PlanID: p.ID, Outcome: "COMPLETED", RequestDigest: scheduledCreateDigest(p.ID, p.CreationFingerprint), AppliedRevision: 1, CreatedAt: time.Now().UTC()}).Error)
	})
}

func (s *Store) RejectScheduledCreate(ctx context.Context, scope aiworkbench.Scope, id, key, fingerprint string) (aiworkbench.ScheduledPlan, bool, error) {
	if !validDigest(fingerprint) {
		return aiworkbench.ScheduledPlan{}, false, aiworkbench.ErrInvalid
	}
	return s.rejectScheduledCommand(ctx, scope, id, key, scheduledCreateDigest(id, fingerprint))
}
func (s *Store) RejectScheduledControl(ctx context.Context, scope aiworkbench.Scope, id, key string, revision int64, action string, rule *aiworkbench.ScheduleRule, confirmation string) (aiworkbench.ScheduledPlan, bool, error) {
	return s.rejectScheduledCommand(ctx, scope, id, key, scheduledControlDigest(id, revision, action, rule, confirmation))
}

// A committed REJECTED receipt is a fence against older in-flight requests.
// Returning a business error inside this transaction would roll the fence back.
func (s *Store) rejectScheduledCommand(ctx context.Context, scope aiworkbench.Scope, id, key, fingerprint string) (aiworkbench.ScheduledPlan, bool, error) {
	if s == nil || s.db == nil || ctx == nil || !validScope(scope) || !validKey(id) || !validKey(key) || !validDigest(fingerprint) {
		return aiworkbench.ScheduledPlan{}, false, aiworkbench.ErrInvalid
	}
	var result aiworkbench.ScheduledPlan
	rejected := false
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := s.lockTaskKey(tx, scope, key); err != nil {
			return err
		}
		var found bool
		var err error
		result, found, err = (&Store{db: tx}).readScheduledCommand(ctx, scope, id, key, fingerprint)
		if errors.Is(err, aiworkbench.ErrScheduleCommandRejected) {
			rejected = true
			return nil
		}
		if err != nil || found {
			return err
		}
		if err = tx.Create(&scheduledControlRow{OrganizationID: scope.OrganizationID, OwnerUserID: scope.ActorID, IdempotencyKey: key, Outcome: "REJECTED", RequestDigest: fingerprint, CreatedAt: time.Now().UTC()}).Error; err != nil {
			return unavailable(err)
		}
		rejected = true
		return nil
	})
	if err != nil {
		if errors.Is(err, aiworkbench.ErrIdempotencyConflict) {
			return aiworkbench.ScheduledPlan{}, false, err
		}
		return aiworkbench.ScheduledPlan{}, false, unavailable(err)
	}
	return result, rejected, nil
}
