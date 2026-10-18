package httpapi

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"task-processor/internal/agentconfig"
	"task-processor/internal/imageagent"
)

type imageReleaseFactReader func(context.Context, imageagent.SlotExternalEffectIdentity) (imageagent.GenerationFact, error)

func (f imageReleaseFactReader) ReadGenerationFact(ctx context.Context, id imageagent.SlotExternalEffectIdentity) (imageagent.GenerationFact, error) {
	return f(ctx, id)
}

func imageReleaseFixture(t *testing.T) (imageagent.RunProjection, imageagent.GenerationFact) {
	t.Helper()
	id := imageagent.SlotExternalEffectIdentity{RunScope: imageagent.RunScope{TenantID: "org", OwnerUserID: "actor", RunID: "run"}, PlanRevision: 1, SlotID: "main", Attempt: 1}
	intent := imageagent.GenerationIntent{Identity: id, MemberID: "member", CatalogHash: "catalog-v2:" + strings.Repeat("a", 64), SourceDigest: strings.Repeat("b", 64), PromptVersion: "image-v1", RouteReference: "route", CredentialReference: "credential", ConfigurationVersion: "config", Provider: "bailian", Model: "qwen-image-3.0-pro", Protocol: "dashscope-image-sync-v1", Resolution: "1024x1024", Quality: "auto", PriceVersion: "price", Points: 1, LimitVersion: 2, MonthStart: time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)}
	fact, err := imageagent.NewGenerationFact(intent)
	require.NoError(t, err)
	fact, err = fact.BindReservation(imageagent.GenerationReservationReceipt{IntentID: fact.IntentID, Fingerprint: fact.Fingerprint, OrganizationID: "org", MemberID: "member", OperationID: "image-reserve:" + fact.IntentID, ReservationID: "reservation", ResourceType: "ai_point", Points: 1, PriceVersion: "price", LimitVersion: 2, MonthStart: intent.MonthStart})
	require.NoError(t, err)
	fact, err = fact.RecordNoGeneration()
	require.NoError(t, err)
	fact, err = fact.BindSettlement(imageagent.GenerationSettlementReceipt{IntentID: fact.IntentID, Fingerprint: fact.Fingerprint, OperationID: "image-finalize:" + fact.IntentID, ReservationID: "reservation", State: "released", ProofDigest: fact.TerminalProofDigest(), Points: 1})
	require.NoError(t, err)
	definition := imageagent.Slot{ID: "main", Recipe: &imageagent.ImageSlotRecipe{PromptVersion: intent.PromptVersion, Quote: imageagent.ImageGenerationQuote{Provider: intent.Provider, Model: intent.Model, Protocol: intent.Protocol, Resolution: intent.Resolution, Quality: intent.Quality, PriceVersion: intent.PriceVersion, Points: intent.Points, RouteReference: intent.RouteReference, CredentialReference: intent.CredentialReference, ConfigurationVersion: intent.ConfigurationVersion}}}
	current := definition
	current.Status = imageagent.SlotStatusBlocked
	p := imageagent.RunProjection{Run: imageagent.Run{ID: "run", TenantID: "org", UserID: "actor", MemberID: "member", ScopeProtocol: imageagent.OrganizationScopeProtocol, Status: imageagent.RunStatusBlocked, ImageAdmission: &agentconfig.ImageRunAdmissionReceipt{}, Block: &imageagent.Block{Code: "slot_provider_outcome_unknown"}}, Plan: imageagent.Plan{Revision: 1, Set: &imageagent.ImageSetPlan{MaxPoints: 1}, Slots: []imageagent.Slot{definition}}, Slots: []imageagent.SlotProjection{{Slot: current, Attempt: 1, ErrorCode: "slot_provider_outcome_unknown"}}, RecoverableEffects: []imageagent.RecoverableEffect{{SlotID: "main", Attempt: 1, Code: "slot_provider_outcome_unknown"}}}
	return p, fact
}

func TestImageReleaseSummaryReadsTheOriginalSettlementWithoutChangingFailures(t *testing.T) {
	p, fact := imageReleaseFixture(t)
	before := p
	a := &fullImageApplication{generationFacts: imageReleaseFactReader(func(_ context.Context, id imageagent.SlotExternalEffectIdentity) (imageagent.GenerationFact, error) {
		require.Equal(t, fact.Intent.Identity, id)
		return fact, nil
	})}
	got, err := a.releaseSummary(context.Background(), p)
	require.NoError(t, err)
	require.Equal(t, &fullImageReleaseSummary{ReleasedPoints: 1}, got)
	require.Equal(t, before, p)
	require.Equal(t, "slot_provider_outcome_unknown", p.Slots[0].ErrorCode)
}

func TestImageReleaseSummaryNeverClaimsUnknownPartialOrUnsettledWorkWasReleased(t *testing.T) {
	for _, scenario := range []string{"missing", "unknown", "unsettled", "no_reservation", "candidate", "closure", "partial", "old_attempt", "later_attempt", "later_plan", "not_admitted", "executing", "wrong_quote", "wrong_scope", "wrong_member", "corrupt_receipt"} {
		t.Run(scenario, func(t *testing.T) {
			p, fact := imageReleaseFixture(t)
			var readErr error
			switch scenario {
			case "missing":
				readErr = imageagent.ErrRunNotFound
			case "unknown":
				fact.State, fact.Settlement = imageagent.GenerationUnknown, imageagent.GenerationSettlementReceipt{}
			case "unsettled":
				fact.Settlement = imageagent.GenerationSettlementReceipt{}
			case "no_reservation":
				fact.Reservation = imageagent.GenerationReservationReceipt{}
				fact.Settlement.ReservationID, fact.Settlement.State = "", "no_reservation"
			case "candidate":
				p.Slots[0].Candidates = []imageagent.AssetCandidate{{AssetID: "generated"}}
			case "closure":
				p.Slots[0].Closure = &imageagent.ImageSlotClosure{Kind: "generated", Points: 1}
			case "partial":
				p.Plan.Slots = append(p.Plan.Slots, imageagent.Slot{ID: "detail"})
			case "old_attempt":
				p.RecoverableEffects[0].Attempt = 2
			case "later_attempt":
				p.Slots[0].Attempt = 2
			case "later_plan":
				p.Plan.Revision = 2
			case "not_admitted":
				p.Run.ImageAdmission = nil
			case "executing":
				p.Run.Status = imageagent.RunStatusExecuting
			case "wrong_quote":
				p.Plan.Slots[0].Recipe.Quote.PriceVersion = "other-price"
			case "wrong_scope":
				p.Run.TenantID = "other-org"
			case "wrong_member":
				p.Run.MemberID = "other-member"
			case "corrupt_receipt":
				fact.Settlement.ProofDigest = strings.Repeat("f", 64)
			}
			a := &fullImageApplication{generationFacts: imageReleaseFactReader(func(context.Context, imageagent.SlotExternalEffectIdentity) (imageagent.GenerationFact, error) {
				return fact, readErr
			})}
			got, err := a.releaseSummary(context.Background(), p)
			require.Nil(t, got)
			if scenario == "wrong_scope" || scenario == "wrong_member" || scenario == "corrupt_receipt" || scenario == "wrong_quote" {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
	t.Run("read_failure", func(t *testing.T) {
		p, _ := imageReleaseFixture(t)
		failure := errors.New("database unavailable")
		a := &fullImageApplication{generationFacts: imageReleaseFactReader(func(context.Context, imageagent.SlotExternalEffectIdentity) (imageagent.GenerationFact, error) {
			return imageagent.GenerationFact{}, failure
		})}
		got, err := a.releaseSummary(context.Background(), p)
		require.Nil(t, got)
		require.ErrorIs(t, err, failure)
	})
}

func TestFullImageRecentReleaseSummaryFollowsLiveSourceAuthorization(t *testing.T) {
	for _, denied := range []bool{false, true} {
		t.Run(map[bool]string{false: "visible", true: "denied"}[denied], func(t *testing.T) {
			a, ctx, items, projections := recentImageSetFixture(t, 1)
			p, fact := imageReleaseFixture(t)
			p.Run.ID, p.Run.BusinessTaskID = items[0].RunID, items[0].ContextID
			p.Plan.Set.Source = projections[items[0].RunID].Plan.Set.Source
			p.Plan.Set.Target = projections[items[0].RunID].Plan.Set.Target
			// Bind the same valid receipt to this fixture's original run identity.
			intent := fact.Intent
			intent.Identity.RunID = p.Run.ID
			fact, err := imageagent.NewGenerationFact(intent)
			require.NoError(t, err)
			fact, err = fact.RecordNoGeneration()
			require.NoError(t, err)
			fact, err = fact.BindSettlement(imageagent.GenerationSettlementReceipt{IntentID: fact.IntentID, Fingerprint: fact.Fingerprint, OperationID: "image-finalize:" + fact.IntentID, ReservationID: "original-reservation", State: "released", ProofDigest: fact.TerminalProofDigest(), Points: 1})
			require.NoError(t, err)
			projections[p.Run.ID] = p
			reads := 0
			a.generationFacts = imageReleaseFactReader(func(_ context.Context, id imageagent.SlotExternalEffectIdentity) (imageagent.GenerationFact, error) {
				reads++
				require.Equal(t, fact.Intent.Identity, id)
				return fact, nil
			})
			a.readSources = recentImageSetSources(func(context.Context, imageagent.ExecutionIdentity, imageagent.PrepareImageSetInput) (imageagent.ImageSetPreparation, error) {
				if denied {
					return imageagent.ImageSetPreparation{}, imageagent.ErrIdentityRequired
				}
				return imageagent.ImageSetPreparation{}, nil
			})
			got, _, err := a.readRecent(ctx, "", "previous", 1)
			require.NoError(t, err)
			if denied {
				require.Empty(t, got)
				require.Zero(t, reads)
			} else {
				require.Len(t, got, 1)
				require.Equal(t, imageagent.RunStatusBlocked, got[0].Status)
				require.Equal(t, &fullImageReleaseSummary{ReleasedPoints: 1}, got[0].ReleaseSummary)
				require.Equal(t, 1, reads)
			}
		})
	}
}
