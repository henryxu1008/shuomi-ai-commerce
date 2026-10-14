package imageagent

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestImageGenerationClosureRequiresOriginalTerminalSettlement(t *testing.T) {
	intent := generationTestIntent()
	intent.InputProtocol, intent.InputDigest = ImageSetSchema, strings.Repeat("d", 64)
	fact, err := NewGenerationFact(intent)
	require.NoError(t, err)
	_, err = ImageGenerationClosure(fact)
	require.ErrorIs(t, err, ErrCommandBlocked)
	fact, err = fact.BindReservation(generationTestReceipt(fact))
	require.NoError(t, err)
	fact, _, err = fact.BeginDispatch()
	require.NoError(t, err)
	_, err = ImageGenerationClosure(fact)
	require.ErrorIs(t, err, ErrCommandBlocked)
	fact, err = fact.RecordSuccess(GenerationSuccess{ResponseID: "response", ResultDigest: strings.Repeat("c", 64), ResultUnavailable: "invalid_result"})
	require.NoError(t, err)
	_, err = ImageGenerationClosure(fact)
	require.ErrorIs(t, err, ErrCommandBlocked, "provider success without settlement cannot be approved")
	settlement := GenerationSettlementReceipt{IntentID: fact.IntentID, Fingerprint: fact.Fingerprint, OperationID: "image-finalize:" + fact.IntentID, ReservationID: fact.Reservation.ReservationID, State: "committed", ProofDigest: fact.TerminalProofDigest(), Points: intent.Points}
	fact, err = fact.BindSettlement(settlement)
	require.NoError(t, err)
	closure, err := ImageGenerationClosure(fact)
	require.NoError(t, err)
	require.Equal(t, &ImageSlotClosure{Kind: "settled", IntentID: fact.IntentID, Fingerprint: fact.Fingerprint, SettlementProofDigest: fact.TerminalProofDigest(), Points: intent.Points}, closure)
	noEffect, err := NewGenerationFact(intent)
	require.NoError(t, err)
	noEffect, err = noEffect.RecordNoGeneration()
	require.NoError(t, err)
	settlement.State, settlement.ReservationID, settlement.ProofDigest = "no_reservation", "", noEffect.TerminalProofDigest()
	noEffect, err = noEffect.BindSettlement(settlement)
	require.NoError(t, err)
	closure, err = ImageGenerationClosure(noEffect)
	require.NoError(t, err)
	require.Equal(t, "no_generation", closure.Kind)
	require.Zero(t, closure.Points, "original quoted points are not incurred points for a no-effect result")
}
func TestArtworkSettlementCanCloseTheOriginalVisualSlot(t *testing.T) {
	input := artworkContractInput(t)
	intent := generationTestIntent()
	intent.CatalogHash = input.AssetCatalog.Manifest.Hash
	intent.InputProtocol, intent.InputDigest, intent.SourceDigest, intent.PromptVersion = ArtworkSchema, strings.Repeat("d", 64), strings.Repeat("d", 64), ArtworkSchema
	fact, err := NewGenerationFact(intent)
	require.NoError(t, err)
	_, err = ImageGenerationClosure(fact)
	require.ErrorIs(t, err, ErrCommandBlocked)
	fact, err = fact.BindReservation(generationTestReceipt(fact))
	require.NoError(t, err)
	fact, _, err = fact.BeginDispatch()
	require.NoError(t, err)
	fact, err = fact.RecordSuccess(GenerationSuccess{ResponseID: "original-artwork", ResultDigest: strings.Repeat("c", 64), ResultURL: "https://cdn.example/art.png"})
	require.NoError(t, err)
	_, err = ImageGenerationClosure(fact)
	require.ErrorIs(t, err, ErrCommandBlocked)
	fact, err = fact.BindSettlement(GenerationSettlementReceipt{IntentID: fact.IntentID, Fingerprint: fact.Fingerprint, OperationID: "image-finalize:" + fact.IntentID, ReservationID: fact.Reservation.ReservationID, State: "committed", ProofDigest: fact.TerminalProofDigest(), Points: fact.Intent.Points})
	require.NoError(t, err)
	closure, err := ImageGenerationClosure(fact)
	require.NoError(t, err)
	require.Equal(t, "settled", closure.Kind)
	require.Equal(t, fact.TerminalProofDigest(), closure.SettlementProofDigest)
	require.Equal(t, fact.Intent.Points, closure.Points)
}
