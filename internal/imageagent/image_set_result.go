package imageagent

import (
	"reflect"

	"task-processor/internal/agentconfig"
)

// ImageSetResultDigest is a separate contract from ResultDigestV3. It binds
// known terminal failures and unstarted work without declaring them accepted.
// Financial references must still be resolved by their owners on selection.
func ImageSetResultDigest(plan Plan, slots []SlotProjection, recoverable []RecoverableEffect) (string, error) {
	return imageSetEffectsDigest(plan, slots, recoverable, true, false)
}

// Regeneration eligibility also covers a wholly failed run whose original
// effects all closed. This does not make it eligible for result approval.
func ImageSetClosedEffectsDigest(plan Plan, slots []SlotProjection, recoverable []RecoverableEffect) (string, error) {
	return imageSetEffectsDigest(plan, slots, recoverable, false, false)
}

// These statuses permit checking closed effects; the status alone never proves
// closure, settlement or materialization.
func ImageSetClosedRunStatus(status RunStatus) bool {
	switch status {
	case RunStatusAwaitingFinalApproval, RunStatusCompleted, RunStatusBlocked, RunStatusFailed, RunStatusCancelled:
		return true
	default:
		return false
	}
}

// Candidate reuse is distinct from terminating a Run's approval. A cancelled
// parent may never have entered final approval, so derive its exact candidate
// digest from the original persisted plan/slots. Never overwrite or ignore a
// conflicting stored digest, and keep UNKNOWN/unclosed effects ineligible.
func ImageSetCandidateResultDigest(projection RunProjection) (string, error) {
	if !ImageSetClosedRunStatus(projection.Run.Status) || projection.Run.ActivePlanRevision != projection.Plan.Revision || ValidateImageSetAdmission(projection.Run, projection.Plan) != nil {
		return "", ErrRevisionConflict
	}
	digest, err := ImageSetResultDigest(projection.Plan, projection.Slots, projection.RecoverableEffects)
	if err != nil || projection.ResultDigest != "" && projection.ResultDigest != digest || projection.Run.Status == RunStatusAwaitingFinalApproval && projection.ResultDigest == "" {
		return "", ErrRevisionConflict
	}
	return digest, nil
}

func ArtworkResultDigest(plan Plan, slots []SlotProjection, recoverable []RecoverableEffect) (string, error) {
	return imageSetEffectsDigest(plan, slots, recoverable, true, true)
}
func VisualResultDigest(plan Plan, slots []SlotProjection, recoverable []RecoverableEffect) (string, error) {
	if plan.Artwork != nil {
		return ArtworkResultDigest(plan, slots, recoverable)
	}
	return ImageSetResultDigest(plan, slots, recoverable)
}
func imageSetEffectsDigest(plan Plan, slots []SlotProjection, recoverable []RecoverableEffect, requireAccepted, artwork bool) (string, error) {
	if (artwork && plan.Artwork == nil || !artwork && (plan.Set == nil || plan.Artwork != nil)) || ValidateSubmittedPlan(plan) != nil || len(slots) != len(plan.Slots) || len(recoverable) != 0 {
		return "", ErrRevisionConflict
	}
	accepted := 0
	seen := map[string]bool{}
	for index, declared := range plan.Slots {
		actual := slots[index]
		definition := actual.Slot
		definition.Status = declared.Status
		if !reflect.DeepEqual(definition, declared) || actual.Closure == nil {
			return "", ErrRevisionConflict
		}
		closure := actual.Closure
		points := int64(0)
		if artwork {
			points = plan.Artwork.Quote.Points
		} else {
			points = declared.Recipe.Quote.Points
		}
		switch closure.Kind {
		case "settled":
			if !agentconfig.ImageDigest(closure.IntentID) || !agentconfig.ImageDigest(closure.Fingerprint) || !agentconfig.ImageDigest(closure.SettlementProofDigest) || closure.Points != points || actual.Attempt <= 0 {
				return "", ErrRevisionConflict
			}
		case "no_generation":
			if !agentconfig.ImageDigest(closure.IntentID) || !agentconfig.ImageDigest(closure.Fingerprint) || !agentconfig.ImageDigest(closure.SettlementProofDigest) || closure.Points != 0 || actual.Attempt <= 0 {
				return "", ErrRevisionConflict
			}
		case "not_dispatched":
			if closure.IntentID != "" || closure.Fingerprint != "" || closure.SettlementProofDigest != "" || closure.Points != 0 || actual.Attempt != 0 {
				return "", ErrRevisionConflict
			}
		default:
			return "", ErrRevisionConflict
		}
		switch actual.Slot.Status {
		case SlotStatusAccepted:
			if closure.Kind != "settled" || actual.ErrorCode != "" || len(actual.Candidates) != 1 {
				return "", ErrRevisionConflict
			}
			candidate := actual.Candidates[0]
			proof := candidate.GenerationProof
			if ValidateProvenanceAssetID(candidate.AssetID) != nil || seen[candidate.AssetID] || candidate.Width <= 0 || candidate.Height <= 0 || proof == nil || proof.IntentID != closure.IntentID || proof.Fingerprint != closure.Fingerprint || proof.SettlementProofDigest != closure.SettlementProofDigest || proof.Points != closure.Points {
				return "", ErrRevisionConflict
			}
			if artwork {
				if candidate.SourceAssetID != "" || candidate.Artwork == nil || candidate.Artwork.Validate() != nil || candidate.Artwork.PlanRevision != plan.Revision || candidate.Artwork.SlotID != declared.ID || candidate.Artwork.Attempt != actual.Attempt || candidate.Artwork.IntentID != closure.IntentID || candidate.Artwork.Fingerprint != closure.Fingerprint || !reflect.DeepEqual(candidate.Operations, []string{"generate_artwork"}) {
					return "", ErrRevisionConflict
				}
			} else if candidate.Artwork != nil || candidate.SourceAssetID != declared.SourceAssetIDs[0] || !reflect.DeepEqual(candidate.Operations, []string{"render_source_edit"}) {
				return "", ErrRevisionConflict
			}
			if _, err := NormalizeDurableAssetIdentity(candidate.DurableAsset); err != nil {
				return "", err
			}
			seen[candidate.AssetID] = true
			accepted++
		case SlotStatusBlocked, SlotStatusRejected:
			if !canonicalImageValue(actual.ErrorCode) || len(actual.Candidates) != 0 {
				return "", ErrRevisionConflict
			}
		default:
			return "", ErrRevisionConflict
		}
	}
	if requireAccepted && accepted == 0 {
		return "", ErrCommandBlocked
	}
	// SlotProjection's explicit JSON preserves durable objects, observations,
	// original settlement references and each recipe in the ordered result.
	schema := "product-image-set-result-v1"
	if artwork {
		schema = "pod-artwork-result-v1"
	}
	return resultDigestSHA256(struct {
		Schema string
		Plan   Plan
		Slots  []SlotProjection
	}{schema, plan, slots})
}
