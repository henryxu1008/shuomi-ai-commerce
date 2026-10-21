package httpapi

import (
	"context"
	"errors"
	"math"

	"task-processor/internal/imageagent"
)

// This read DTO describes original paid effects; it never grants dispatch or
// changes the historical workflow state, failure, or Commercial receipt.
type fullImageOutputRecoverySummary struct {
	SettledPoints int64                     `json:"settledPoints"`
	Slots         []fullImagePaidOutputSlot `json:"slots"`
}

type fullImagePaidOutputSlot struct {
	SlotID      string `json:"slotId"`
	Attempt     int    `json:"attempt"`
	Points      int64  `json:"points"`
	Unavailable bool   `json:"unavailable"`
}

func (a *fullImageApplication) outputRecoverySummary(ctx context.Context, p imageagent.RunProjection) (*fullImageOutputRecoverySummary, error) {
	if a.generationFacts == nil || p.Run.ScopeProtocol != imageagent.OrganizationScopeProtocol || p.Run.ImageAdmission == nil || p.PendingCommand != nil || p.Plan.Set == nil ||
		(p.Run.Status != imageagent.RunStatusBlocked && p.Run.Status != imageagent.RunStatusFailed) {
		return nil, nil
	}
	result := &fullImageOutputRecoverySummary{}
	seen := map[string]bool{}
	for _, slot := range p.Slots {
		if seen[slot.Slot.ID] {
			return nil, imageagent.ErrRevisionConflict
		}
		seen[slot.Slot.ID] = true
		if slot.Closure != nil || len(slot.Candidates) != 0 {
			continue // Its original closure already accounts for these points.
		}
		_, definition, recoverable := imageagent.FindRecoverableEffect(p, slot.Slot.ID, slot.Attempt)
		if !recoverable || definition.Recipe == nil {
			continue
		}
		id := imageagent.SlotExternalEffectIdentity{RunScope: imageagent.ScopeForRun(p.Run), PlanRevision: p.Plan.Revision, SlotID: definition.ID, Attempt: slot.Attempt}
		fact, err := a.generationFacts.ReadGenerationFact(ctx, id)
		if errors.Is(err, imageagent.ErrRunNotFound) {
			continue
		}
		if err != nil {
			return nil, err
		}
		if fact.Validate() != nil || fact.Intent.Identity != id || fact.Intent.MemberID != p.Run.MemberID {
			return nil, imageagent.ErrRevisionConflict
		}
		intent, quote := fact.Intent, definition.Recipe.Quote
		if intent.Provider != quote.Provider || intent.Model != quote.Model || intent.Protocol != quote.Protocol || intent.Resolution != quote.Resolution || intent.Quality != quote.Quality ||
			intent.PriceVersion != quote.PriceVersion || intent.Points != quote.Points || intent.RouteReference != quote.RouteReference || intent.CredentialReference != quote.CredentialReference || intent.ConfigurationVersion != quote.ConfigurationVersion || intent.PromptVersion != definition.Recipe.PromptVersion ||
			(p.AssetCatalog.Manifest.Hash != "" && intent.CatalogHash != p.AssetCatalog.Manifest.Hash) {
			return nil, imageagent.ErrRevisionConflict
		}
		if fact.State != imageagent.GenerationSucceeded || fact.Settlement.State != "committed" || fact.Settlement.ReservationID == "" {
			continue
		}
		if result.SettledPoints > math.MaxInt64-fact.Settlement.Points {
			return nil, imageagent.ErrValidation
		}
		_, urlErr := imageagent.ValidateSafeImageURL(fact.Success.ResultURL)
		result.SettledPoints += fact.Settlement.Points
		result.Slots = append(result.Slots, fullImagePaidOutputSlot{SlotID: definition.ID, Attempt: slot.Attempt, Points: fact.Settlement.Points, Unavailable: fact.Success.ResultUnavailable != "" || urlErr != nil})
	}
	if len(result.Slots) == 0 {
		return nil, nil
	}
	if result.SettledPoints > p.Plan.Set.MaxPoints {
		return nil, imageagent.ErrRevisionConflict
	}
	return result, nil
}
