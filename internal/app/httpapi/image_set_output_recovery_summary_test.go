package httpapi

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"task-processor/internal/imageagent"
)

func paidImageOutputFixture(t *testing.T) (imageagent.RunProjection, imageagent.GenerationFact) {
	p, released := imageReleaseFixture(t)
	f, err := imageagent.NewGenerationFact(released.Intent)
	require.NoError(t, err)
	f, err = f.BindReservation(released.Reservation)
	require.NoError(t, err)
	f, _, err = f.BeginDispatch()
	require.NoError(t, err)
	f, err = f.RecordSuccess(imageagent.GenerationSuccess{ResponseID: "original-response", ResultDigest: released.Intent.SourceDigest}.WithResultLocator("https://output.example/original.png", ""))
	require.NoError(t, err)
	r := released.Settlement
	r.State, r.ProofDigest = "committed", f.TerminalProofDigest()
	f, err = f.BindSettlement(r)
	require.NoError(t, err)
	return p, f
}

func TestPaidOriginalOutputSummaryPreservesTheFailureAndSettlement(t *testing.T) {
	p, fact := paidImageOutputFixture(t)
	before := p
	a := &fullImageApplication{generationFacts: imageReleaseFactReader(func(_ context.Context, id imageagent.SlotExternalEffectIdentity) (imageagent.GenerationFact, error) {
		require.Equal(t, fact.Intent.Identity, id)
		return fact, nil
	})}
	summary, err := a.outputRecoverySummary(context.Background(), p)
	require.NoError(t, err)
	require.NotNil(t, summary)
	require.EqualValues(t, 1, summary.SettledPoints)
	require.Len(t, summary.Slots, 1)
	require.Equal(t, "main", summary.Slots[0].SlotID)
	require.False(t, summary.Slots[0].Unavailable)
	require.Equal(t, before, p)
}

func TestPaidOriginalOutputSummaryRejectsUnprovedAndMismatchedFacts(t *testing.T) {
	for _, scenario := range []string{"unknown", "unsettled", "wrong_member", "wrong_scope", "wrong_quote", "corrupt", "closed", "candidate", "wrong_attempt"} {
		t.Run(scenario, func(t *testing.T) {
			p, fact := paidImageOutputFixture(t)
			switch scenario {
			case "unknown":
				fact.State, fact.Success, fact.Settlement = imageagent.GenerationUnknown, imageagent.GenerationSuccess{}, imageagent.GenerationSettlementReceipt{}
			case "unsettled":
				fact.Settlement = imageagent.GenerationSettlementReceipt{}
			case "wrong_member":
				p.Run.MemberID = "other"
			case "wrong_scope":
				p.Run.TenantID = "other"
			case "wrong_quote":
				p.Plan.Slots[0].Recipe.Quote.PriceVersion = "other"
			case "corrupt":
				fact.Settlement.ProofDigest = "corrupt"
			case "closed":
				p.Slots[0].Closure = &imageagent.ImageSlotClosure{Kind: "generated", Points: 1}
			case "candidate":
				p.Slots[0].Candidates = []imageagent.AssetCandidate{{AssetID: "candidate"}}
			case "wrong_attempt":
				p.RecoverableEffects[0].Attempt = 2
			}
			a := &fullImageApplication{generationFacts: imageReleaseFactReader(func(context.Context, imageagent.SlotExternalEffectIdentity) (imageagent.GenerationFact, error) {
				return fact, nil
			})}
			summary, _ := a.outputRecoverySummary(context.Background(), p)
			require.Nil(t, summary)
		})
	}
}

func TestPaidOriginalOutputSummaryNeverOffersAnUnavailableLocator(t *testing.T) {
	p, fact := paidImageOutputFixture(t)
	fact.Success.ResultURL, fact.Success.ResultUnavailable = "", "invalid_result"
	fact.Settlement.ProofDigest = fact.TerminalProofDigest()
	a := &fullImageApplication{generationFacts: imageReleaseFactReader(func(context.Context, imageagent.SlotExternalEffectIdentity) (imageagent.GenerationFact, error) {
		return fact, nil
	})}
	summary, err := a.outputRecoverySummary(context.Background(), p)
	require.NoError(t, err)
	require.NotNil(t, summary)
	require.True(t, summary.Slots[0].Unavailable)
}

func TestPaidOriginalOutputSummaryKeepsUnknownSiblingsAndClosedPointsSeparate(t *testing.T) {
	p, paid := paidImageOutputFixture(t)
	other := p.Plan.Slots[0]
	other.ID = "unknown"
	p.Plan.Slots = append(p.Plan.Slots, other)
	current := other
	current.Status = imageagent.SlotStatusBlocked
	p.Slots = append(p.Slots, imageagent.SlotProjection{Slot: current, Attempt: 1, ErrorCode: imageagent.SlotProviderOutcomeUnknownCode})
	p.RecoverableEffects = append(p.RecoverableEffects, imageagent.RecoverableEffect{SlotID: other.ID, Attempt: 1, Code: imageagent.SlotProviderOutcomeUnknownCode})
	p.Plan.Set.MaxPoints = 2
	intent := paid.Intent
	intent.Identity.SlotID = other.ID
	unknown, err := imageagent.NewGenerationFact(intent)
	require.NoError(t, err)
	receipt := paid.Reservation
	receipt.IntentID, receipt.Fingerprint = unknown.IntentID, unknown.Fingerprint
	receipt.OperationID, receipt.ReservationID = "image-reserve:"+unknown.IntentID, "other-reservation"
	unknown, err = unknown.BindReservation(receipt)
	require.NoError(t, err)
	unknown, _, err = unknown.BeginDispatch()
	require.NoError(t, err)
	unknown, err = unknown.MarkUnknown()
	require.NoError(t, err)
	a := &fullImageApplication{generationFacts: imageReleaseFactReader(func(_ context.Context, id imageagent.SlotExternalEffectIdentity) (imageagent.GenerationFact, error) {
		if id.SlotID == "main" {
			return paid, nil
		}
		return unknown, nil
	})}
	summary, err := a.outputRecoverySummary(context.Background(), p)
	require.NoError(t, err)
	require.Len(t, summary.Slots, 1)
	require.EqualValues(t, 1, summary.SettledPoints)
	// A materialized closure is accounted by the old projection, never twice.
	p.Slots[0].Closure = &imageagent.ImageSlotClosure{Kind: "generated", Points: 1}
	summary, err = a.outputRecoverySummary(context.Background(), p)
	require.NoError(t, err)
	require.Nil(t, summary)
}

func TestPaidOriginalOutputSummaryAcceptsOriginalBlockFallback(t *testing.T) {
	p, fact := paidImageOutputFixture(t)
	p.RecoverableEffects = nil
	p.Run.Block.SlotID = "main"
	a := &fullImageApplication{generationFacts: imageReleaseFactReader(func(context.Context, imageagent.SlotExternalEffectIdentity) (imageagent.GenerationFact, error) {
		return fact, nil
	})}
	summary, err := a.outputRecoverySummary(context.Background(), p)
	require.NoError(t, err)
	require.NotNil(t, summary)
	require.EqualValues(t, 1, summary.SettledPoints)
	require.Nil(t, p.RecoverableEffects, "a read must not rewrite the historical snapshot")
}
