package imageagent

import (
	"fmt"
	"strings"
)

func IsRecoverableEffectBlockCode(code string) bool {
	switch strings.TrimSpace(code) {
	case "recovery_requested", "recovery_start_failed",
		SlotProviderOutcomeUnknownCode, SlotStagingOutcomeUnknownCode, SlotPublicationOutcomeUnknownCode,
		SlotRecoveryBlockedCode, SlotEffectPhaseInvalidCode, SlotEffectPolicyInvalidCode:
		return true
	default:
		return false
	}
}

func NormalizeRecoverableEffects(effects []RecoverableEffect) ([]RecoverableEffect, error) {
	if len(effects) == 0 {
		return nil, nil
	}
	normalized := make([]RecoverableEffect, 0, len(effects))
	seen := make(map[string]string, len(effects))
	for _, effect := range effects {
		slotID := strings.TrimSpace(effect.SlotID)
		code := strings.TrimSpace(effect.Code)
		if slotID == "" || effect.Attempt <= 0 || !IsRecoverableEffectBlockCode(code) {
			return nil, ErrValidation
		}
		key := recoverableEffectKey(slotID, effect.Attempt)
		if existingCode, exists := seen[key]; exists {
			if existingCode != code {
				return nil, fmt.Errorf("%w: conflicting recoverable effect code for %s", ErrRevisionConflict, key)
			}
			continue
		}
		seen[key] = code
		normalized = append(normalized, RecoverableEffect{SlotID: slotID, Attempt: effect.Attempt, Code: code})
	}
	return normalized, nil
}

func FindRecoverableEffect(projection RunProjection, slotID string, attempt int) (RecoverableEffect, Slot, bool) {
	return findRecoverableEffect(projection, slotID, attempt, projection.Plan.Set == nil && projection.Plan.Artwork == nil)
}

// FindOriginalSettlementSlot identifies a slot for a read of its original
// generation and settlement facts. It does not authorize a recovery command.
func FindOriginalSettlementSlot(projection RunProjection, slotID string, attempt int) (Slot, bool) {
	_, slot, found := findRecoverableEffect(projection, slotID, attempt, true)
	return slot, found
}

func findRecoverableEffect(projection RunProjection, slotID string, attempt int, allowOriginalBlock bool) (RecoverableEffect, Slot, bool) {
	slotID = strings.TrimSpace(slotID)
	if slotID == "" || attempt <= 0 {
		return RecoverableEffect{}, Slot{}, false
	}
	planSlot, hasPlanSlot := findProjectionPlanSlot(projection.Plan, slotID)
	if !hasPlanSlot {
		return RecoverableEffect{}, Slot{}, false
	}
	effects, err := NormalizeRecoverableEffects(projection.RecoverableEffects)
	if err != nil {
		return RecoverableEffect{}, Slot{}, false
	}
	for _, effect := range effects {
		if effect.SlotID == slotID && effect.Attempt == attempt && slotProjectionMatchesRecoverableEffect(projection.Slots, effect) {
			return effect, planSlot, true
		}
	}
	block := projection.Run.Block
	if !allowOriginalBlock || len(effects) != 0 || block == nil || strings.TrimSpace(block.SlotID) != slotID || !IsRecoverableEffectBlockCode(block.Code) {
		return RecoverableEffect{}, Slot{}, false
	}
	original := RecoverableEffect{SlotID: slotID, Attempt: attempt, Code: strings.TrimSpace(block.Code)}
	if !slotProjectionMatchesRecoverableEffect(projection.Slots, original) {
		return RecoverableEffect{}, Slot{}, false
	}
	return original, planSlot, true
}

func recoverableEffectKey(slotID string, attempt int) string {
	return fmt.Sprintf("%s:%d", strings.TrimSpace(slotID), attempt)
}

func findProjectionPlanSlot(plan Plan, slotID string) (Slot, bool) {
	for _, slot := range plan.Slots {
		if slot.ID == slotID {
			return slot, true
		}
	}
	return Slot{}, false
}

func slotProjectionMatchesRecoverableEffect(slots []SlotProjection, effect RecoverableEffect) bool {
	for _, slot := range slots {
		if slot.Slot.ID == effect.SlotID && slot.Attempt == effect.Attempt &&
			slot.Slot.Status == SlotStatusBlocked && strings.TrimSpace(slot.ErrorCode) == effect.Code {
			return true
		}
	}
	return false
}
