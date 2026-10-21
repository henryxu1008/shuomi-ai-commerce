package httpapi

import (
	"context"
	"errors"
	"math"

	"task-processor/internal/imageagent"
)

// This is a read DTO, not a replacement workflow or accounting state. The
// original projection and generation/resource receipts remain authoritative.
type fullImageReleaseSummary struct {
	ReleasedPoints int64 `json:"releasedPoints"`
}

type fullImageRecentRun struct {
	imageagent.ImageSetRunSummary
	ReleaseSummary        *fullImageReleaseSummary        `json:"releaseSummary,omitempty"`
	OutputRecoverySummary *fullImageOutputRecoverySummary `json:"outputRecoverySummary,omitempty"`
}

type imageGenerationFactReader interface {
	ReadGenerationFact(context.Context, imageagent.SlotExternalEffectIdentity) (imageagent.GenerationFact, error)
}

func (a *fullImageApplication) releaseSummary(ctx context.Context, p imageagent.RunProjection) (*fullImageReleaseSummary, error) {
	if a.generationFacts == nil || p.Run.ScopeProtocol != imageagent.OrganizationScopeProtocol || p.Run.ImageAdmission == nil || p.PendingCommand != nil ||
		(p.Run.Status != imageagent.RunStatusBlocked && p.Run.Status != imageagent.RunStatusFailed) || p.Plan.Set == nil || p.Plan.Revision != 1 || len(p.Plan.Slots) == 0 || len(p.Plan.Slots) != len(p.Slots) {
		return nil, nil
	}
	current := make(map[string]imageagent.SlotProjection, len(p.Slots))
	for _, slot := range p.Slots {
		if _, duplicate := current[slot.Slot.ID]; duplicate || slot.Attempt != 1 || slot.Slot.Status == imageagent.SlotStatusAccepted || len(slot.Candidates) != 0 || slot.Closure != nil {
			return nil, nil
		}
		current[slot.Slot.ID] = slot
	}
	for _, effect := range p.RecoverableEffects {
		if slot, ok := current[effect.SlotID]; !ok || slot.Attempt != effect.Attempt {
			return nil, nil
		}
	}
	var released int64
	for _, definition := range p.Plan.Slots {
		slot, ok := current[definition.ID]
		if !ok || definition.Recipe == nil {
			return nil, nil
		}
		delete(current, definition.ID)
		identity := imageagent.SlotExternalEffectIdentity{RunScope: imageagent.ScopeForRun(p.Run), PlanRevision: p.Plan.Revision, SlotID: definition.ID, Attempt: slot.Attempt}
		fact, err := a.generationFacts.ReadGenerationFact(ctx, identity)
		if errors.Is(err, imageagent.ErrRunNotFound) {
			return nil, nil
		}
		if err != nil {
			return nil, err
		}
		if fact.Validate() != nil || fact.Intent.Identity != identity || fact.Intent.MemberID != p.Run.MemberID {
			return nil, imageagent.ErrRevisionConflict
		}
		intent, quote := fact.Intent, definition.Recipe.Quote
		if intent.Provider != quote.Provider || intent.Model != quote.Model || intent.Protocol != quote.Protocol || intent.Resolution != quote.Resolution || intent.Quality != quote.Quality ||
			intent.PriceVersion != quote.PriceVersion || intent.Points != quote.Points || intent.RouteReference != quote.RouteReference || intent.CredentialReference != quote.CredentialReference || intent.ConfigurationVersion != quote.ConfigurationVersion || intent.PromptVersion != definition.Recipe.PromptVersion {
			return nil, imageagent.ErrRevisionConflict
		}
		if fact.State != imageagent.GenerationNoEffect || fact.Settlement.State != "released" || fact.Settlement.ReservationID == "" {
			return nil, nil
		}
		if released > math.MaxInt64-fact.Settlement.Points {
			return nil, imageagent.ErrValidation
		}
		released += fact.Settlement.Points
	}
	if released != p.Plan.Set.MaxPoints {
		return nil, imageagent.ErrRevisionConflict
	}
	return &fullImageReleaseSummary{ReleasedPoints: released}, nil
}
