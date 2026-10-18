package store

import (
	"bytes"
	"context"
	"encoding/json"

	"gorm.io/gorm"
	"task-processor/internal/imageagent"
)

type generationRejectionEvent struct {
	CommitID    string
	IntentID    string
	Fingerprint string
	Evidence    imageagent.GenerationRejection
}

// Existing run/effect locks survive the nested savepoint until the outer
// transaction commits. No terminal fact can exist without its evidence event.
func (r *gormRepository) RecordGenerationRejection(ctx context.Context, intent imageagent.GenerationIntent, proof imageagent.GenerationRejection) (imageagent.GenerationFact, error) {
	var result imageagent.GenerationFact
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		scoped := *r
		scoped.db = tx
		priorWasTerminal := false
		fact, err := scoped.mutateGenerationFact(ctx, intent, true, func(prior *imageagent.GenerationFact) (imageagent.GenerationFact, error) {
			priorWasTerminal = prior.State == imageagent.GenerationNoEffect
			return prior.RecordRejection(proof)
		})
		if err != nil {
			return err
		}
		payload, err := json.Marshal(generationRejectionEvent{CommitID: "generation.no_generation:" + fact.IntentID, IntentID: fact.IntentID, Fingerprint: fact.Fingerprint, Evidence: proof})
		if err != nil {
			return err
		}
		var events []eventRecord
		scope := intent.Identity.RunScope
		if err := tx.Where("tenant_id = ? AND owner_user_id = ? AND run_id = ? AND type = ?", scope.TenantID, scope.OwnerUserID, scope.RunID, "generation.no_generation").Find(&events).Error; err != nil {
			return err
		}
		for _, event := range events {
			var stored generationRejectionEvent
			if json.Unmarshal(event.Payload, &stored) != nil {
				return imageagent.ErrCorruptPersistedEffect
			}
			if stored.IntentID == fact.IntentID {
				if !bytes.Equal(event.Payload, payload) {
					return imageagent.ErrRevisionConflict
				}
				result = fact
				return nil
			}
		}
		if priorWasTerminal {
			return imageagent.ErrRevisionConflict
		}
		cursor, err := nextEventCursor(ctx, tx, scope)
		if err != nil {
			return err
		}
		if err := tx.Create(&eventRecord{TenantID: scope.TenantID, OwnerUserID: scope.OwnerUserID, RunID: scope.RunID, Type: "generation.no_generation", Cursor: cursor, ProjectionVersion: cursor, Payload: payload}).Error; err != nil {
			return err
		}
		result = fact
		return nil
	})
	if err != nil {
		return imageagent.GenerationFact{}, err
	}
	return result, nil
}
