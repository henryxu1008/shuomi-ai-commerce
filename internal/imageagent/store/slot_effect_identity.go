package store

import (
	"fmt"
	"strings"
	"task-processor/internal/imageagent"
)

func validateSlotEffectIdentity(identity imageagent.SlotExternalEffectIdentity) error {
	if err := validateScope(identity.RunScope); err != nil {
		return err
	}
	if identity.PlanRevision <= 0 || strings.TrimSpace(identity.SlotID) == "" || identity.Attempt <= 0 {
		return imageagent.ErrValidation
	}
	return nil
}

func validateMemorySlotEffectScope(r *memoryRepository, identity imageagent.SlotExternalEffectIdentity) error {
	run, ok := r.runs[scopeKey(identity.RunScope)]
	if !ok {
		return imageagent.ErrRunNotFound
	}
	if run.ActivePlanRevision != identity.PlanRevision {
		return imageagent.ErrRevisionConflict
	}
	if _, ok := r.slots[slotKey(identity.RunScope, identity.PlanRevision, identity.SlotID)]; !ok {
		return imageagent.ErrRevisionConflict
	}
	return nil
}

func slotEffectKey(identity imageagent.SlotExternalEffectIdentity) string {
	return fmt.Sprintf("%s\x00%d\x00%s\x00%d", scopeKey(identity.RunScope), identity.PlanRevision, identity.SlotID, identity.Attempt)
}
