package imageagent

import (
	"context"
	"errors"
	"task-processor/internal/ledger/storagecapacity"
)

// GenerationRecovery is an internal effect-owner seam, not an HTTP command.
// It accepts only an existing identity and cannot reserve, dispatch or fetch.
type GenerationRecovery interface {
	ReconcileExistingGeneration(context.Context, SlotExternalEffectIdentity) (GenerationState, error)
}

type GenerationRecoveryFacts interface {
	ReadGenerationFact(context.Context, SlotExternalEffectIdentity) (GenerationFact, error)
	RecordGenerationNoEffect(context.Context, GenerationIntent) (GenerationFact, error)
	MarkGenerationUnknown(context.Context, GenerationIntent) (GenerationFact, error)
	BindGenerationSettlement(context.Context, GenerationIntent, GenerationSettlementReceipt) (GenerationFact, error)
}
type GenerationFinalizationResource interface {
	FinalizeImageGeneration(context.Context, SlotExternalEffectIdentity) (GenerationSettlementReceipt, error)
}
type generationRecovery struct {
	capacity  storagecapacity.Writer
	facts     GenerationRecoveryFacts
	resources GenerationFinalizationResource
}

func NewGenerationRecovery(facts GenerationRecoveryFacts, resources GenerationFinalizationResource) (GenerationRecovery, error) {
	if facts == nil || resources == nil {
		return nil, ErrValidation
	}
	return &generationRecovery{facts: facts, resources: resources}, nil
}
func (r *generationRecovery) ReconcileExistingGeneration(ctx context.Context, id SlotExternalEffectIdentity) (GenerationState, error) {
	fact, err := r.facts.ReadGenerationFact(ctx, id)
	if errors.Is(err, ErrRunNotFound) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	if fact.Validate() != nil || fact.Intent.Identity != id {
		return "", ErrCorruptPersistedEffect
	}
	switch fact.State {
	case GenerationPrepared:
		fact, err = r.facts.RecordGenerationNoEffect(ctx, fact.Intent)
	case GenerationDispatchStarted:
		fact, err = r.facts.MarkGenerationUnknown(ctx, fact.Intent)
	}
	if err != nil {
		return "", err
	}
	if fact.State == GenerationUnknown {
		return fact.State, nil
	}
	if fact.State != GenerationNoEffect && fact.State != GenerationSucceeded {
		return fact.State, ErrRevisionConflict
	}
	if fact.Settlement != (GenerationSettlementReceipt{}) {
		return fact.State, closeNoGenerationCapacity(ctx, r.capacity, fact)
	}
	receipt, err := r.resources.FinalizeImageGeneration(ctx, id)
	if err != nil {
		return fact.State, err
	}
	_, err = r.facts.BindGenerationSettlement(ctx, fact.Intent, receipt)
	if err == nil {
		err = closeNoGenerationCapacity(ctx, r.capacity, fact)
	}
	return fact.State, err
}
