package productsourcing

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"strings"
	"task-processor/internal/agent"
	"task-processor/internal/agentconfig"
	"task-processor/internal/imageagent"
	"task-processor/internal/product/collection"
	"testing"
	"time"
)

type artworkFactsFixture struct {
	imageagent.Repository
	imageagent.GenerationFactRepository
	imageagent.SlotExternalEffectV3Repository
	p      imageagent.RunProjection
	fact   imageagent.GenerationFact
	effect imageagent.SlotEffectV3Attempt
}

func (f *artworkFactsFixture) GetProjection(_ context.Context, s imageagent.RunScope) (imageagent.RunProjection, error) {
	if s != imageagent.ScopeForRun(f.p.Run) {
		return imageagent.RunProjection{}, imageagent.ErrRunNotFound
	}
	return f.p, nil
}
func (f *artworkFactsFixture) ReadGenerationFact(_ context.Context, id imageagent.SlotExternalEffectIdentity) (imageagent.GenerationFact, error) {
	if id != f.fact.Intent.Identity {
		return imageagent.GenerationFact{}, imageagent.ErrRevisionConflict
	}
	return f.fact, nil
}
func (f *artworkFactsFixture) GetSlotExternalEffectV3(_ context.Context, id imageagent.SlotExternalEffectIdentity) (imageagent.SlotEffectV3Attempt, error) {
	if id != f.effect.Identity {
		return imageagent.SlotEffectV3Attempt{}, imageagent.ErrRevisionConflict
	}
	return f.effect, nil
}

type artworkModeFixture struct {
	agentconfig.ArtworkConfigurationRepository
	denied bool
	calls  int
}

func (m *artworkModeFixture) CheckArtworkExecutionMode(context.Context, agent.ExecutionIdentity, agent.ConfigurationSnapshotRef, agent.ModeSnapshotRef) (agentconfig.ModeSnapshot, error) {
	m.calls++
	if m.denied {
		return agentconfig.ModeSnapshot{}, agentconfig.ErrChanged
	}
	return agentconfig.ModeSnapshot{Mode: agent.ModeDefault}, nil
}

type artworkURLFixture struct{}

func (artworkURLFixture) PublicURL(string) string {
	return "https://cdn.example.org/original-artwork.png"
}
func artworkFixtureHash(v any) string {
	b, _ := json.Marshal(v)
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:])
}
func completedArtworkFixture(t *testing.T, scope collection.Scope) *artworkFactsFixture {
	t.Helper()
	runID := uuid.NewString()
	identity := imageagent.ExecutionIdentity{ScopeProtocol: imageagent.OrganizationScopeProtocol, TenantID: scope.OrganizationID, UserID: scope.ActorID, MemberID: scope.MemberID, RunID: runID, BusinessTaskID: uuid.NewString()}
	catalog, err := imageagent.NormalizeAssetCatalog(imageagent.AssetCatalog{})
	require.NoError(t, err)
	quote := imageagent.ImageGenerationQuote{Provider: "grsai", Model: "gpt-image-2.5", Protocol: "grsai-json-sync-v1", Resolution: "1024x1024", Quality: "auto", Points: 12, PriceVersion: "price-1", RouteReference: "route-1", CredentialReference: "credential-1", ConfigurationVersion: "config-1"}
	art := &imageagent.ArtworkPlan{Schema: imageagent.ArtworkSchema, Theme: "原创水彩狐狸", Requirements: "独立图案", PromptVersion: imageagent.ArtworkSchema, Prompt: "Create original fox artwork", Configuration: agent.ConfigurationSnapshotRef{Kind: agentconfig.SnapshotKind, ID: uuid.NewString(), Digest: strings.Repeat("a", 64)}, ModeSnapshotRef: agent.ModeSnapshotRef{Kind: agent.ExecutionModeSnapshotKind, ID: uuid.NewString(), Digest: strings.Repeat("b", 64)}, ConfigurationEpoch: "1", ParametersDigest: strings.Repeat("c", 64), InputDigest: strings.Repeat("d", 64), MaxPoints: 12, Quote: quote, QuoteDigest: imageagent.ArtworkQuoteDigest(quote)}
	plan := imageagent.Plan{Revision: 1, Artwork: art, CreatedBy: scope.ActorID, IdempotencyKey: uuid.NewString(), Slots: []imageagent.Slot{{ID: "artwork", Role: imageagent.SlotRoleMain, Brief: art.Theme, IdempotencyKey: "artwork", Status: imageagent.SlotStatusPending}}}
	limits := agentconfig.ImageRunLimits{Images: 1, Points: 12, ElapsedSeconds: 300}
	digest, err := imageagent.ArtworkPlanDigest(plan)
	require.NoError(t, err)
	now := time.Now().UTC()
	admission := agentconfig.ImageRunAdmissionReceipt{ID: uuid.NewString(), Command: agentconfig.ImageRunAdmissionCommand{Scope: agent.Scope{OrganizationID: scope.OrganizationID, ActorID: scope.ActorID}, Snapshot: art.Configuration, MemberID: scope.MemberID, RunID: runID, ConfirmActionID: uuid.NewString(), SourceDigest: art.InputDigest, InputDigest: art.InputDigest, PlanDigest: digest, QuoteDigest: art.QuoteDigest, Limits: limits}, AdmittedAt: now, Deadline: now.Add(300 * time.Second)}
	admission.Digest = artworkFixtureHash(admission)
	run := imageagent.Run{ID: runID, ScopeProtocol: identity.ScopeProtocol, TenantID: identity.TenantID, UserID: identity.UserID, MemberID: identity.MemberID, BusinessTaskID: identity.BusinessTaskID, TargetPlatform: "sds", Status: imageagent.RunStatusAwaitingFinalApproval, Mode: imageagent.RunModeManual, ActivePlanRevision: 1, ImageAdmission: &admission, StartedAt: now, Budget: imageagent.ImageSetBudget(limits), IdempotencyKey: plan.IdempotencyKey}
	require.NoError(t, imageagent.ValidateArtworkAdmission(run, plan))
	input := imageagent.SlotExecutionInput{RunID: runID, TenantID: scope.OrganizationID, UserID: scope.ActorID, TargetPlatform: "sds", PlanRevision: 1, Slot: plan.Slots[0], Attempt: 1, IdempotencyKey: "artwork-attempt", AssetCatalog: catalog, Artwork: art, OrganizationIdentity: identity}
	executionDigest, err := imageagent.ArtworkGenerationInputDigest(input)
	require.NoError(t, err)
	id := imageagent.SlotExternalEffectIdentity{RunScope: imageagent.ScopeForRun(run), PlanRevision: 1, SlotID: "artwork", Attempt: 1}
	intent := imageagent.GenerationIntent{Identity: id, MemberID: scope.MemberID, CatalogHash: catalog.Manifest.Hash, SourceDigest: executionDigest, PromptVersion: imageagent.ArtworkSchema, RouteReference: quote.RouteReference, CredentialReference: quote.CredentialReference, ConfigurationVersion: quote.ConfigurationVersion, Provider: quote.Provider, Model: quote.Model, Protocol: quote.Protocol, Resolution: quote.Resolution, Quality: quote.Quality, PriceVersion: quote.PriceVersion, Points: quote.Points, LimitVersion: 1, MonthStart: time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC), InputProtocol: imageagent.ArtworkSchema, InputDigest: executionDigest}
	fact, err := imageagent.NewGenerationFact(intent)
	require.NoError(t, err)
	fact, err = fact.BindReservation(imageagent.GenerationReservationReceipt{IntentID: fact.IntentID, Fingerprint: fact.Fingerprint, OrganizationID: scope.OrganizationID, MemberID: scope.MemberID, OperationID: "image-reserve:" + fact.IntentID, ReservationID: "original-reservation", ResourceType: "ai_point", Points: 12, PriceVersion: intent.PriceVersion, LimitVersion: 1, MonthStart: intent.MonthStart})
	require.NoError(t, err)
	fact, _, err = fact.BeginDispatch()
	require.NoError(t, err)
	fact, err = fact.RecordSuccess(imageagent.GenerationSuccess{ResponseID: "original-response", ResultDigest: strings.Repeat("e", 64), ResultURL: "https://provider.example/result.png"})
	require.NoError(t, err)
	fact, err = fact.BindSettlement(imageagent.GenerationSettlementReceipt{IntentID: fact.IntentID, Fingerprint: fact.Fingerprint, OperationID: "image-finalize:" + fact.IntentID, ReservationID: fact.Reservation.ReservationID, State: "committed", ProofDigest: fact.TerminalProofDigest(), Points: 12})
	require.NoError(t, err)
	provenance, err := imageagent.NewArtworkProvenance(input, fact)
	require.NoError(t, err)
	owner, err := imageagent.ArtifactOwnerKey(scope.ActorID)
	require.NoError(t, err)
	ref := imageagent.PublishedAssetRef{ObjectKey: "image-agent/public/" + scope.OrganizationID + "/" + owner + "/" + runID + "/1/artwork/1/0-" + strings.Repeat("f", 64) + ".png", SHA256: strings.Repeat("f", 64), SizeBytes: 512, Width: 1024, Height: 1024, ContentType: "image/png", Operations: []string{"generate_artwork"}, Artwork: &provenance}
	candidate := imageagent.AssetCandidate{AssetID: "original-artwork", DurableAsset: imageagent.DurableAssetIdentity{ObjectKey: ref.ObjectKey, SHA256: ref.SHA256}, Width: 1024, Height: 1024, Artwork: &provenance, Operations: ref.Operations, GenerationProof: &imageagent.ImageGenerationProof{IntentID: fact.IntentID, Fingerprint: fact.Fingerprint, SettlementProofDigest: fact.TerminalProofDigest(), Points: 12}}
	result, err := imageagent.NewSlotEffectV3PublishedResult(imageagent.SlotExecutionResult{SlotID: "artwork", Attempt: 1, Candidates: []imageagent.AssetCandidate{candidate}})
	require.NoError(t, err)
	fingerprint, err := imageagent.SlotEffectV3PublishedResultFingerprint(result)
	require.NoError(t, err)
	closure, err := imageagent.ImageGenerationClosure(fact)
	require.NoError(t, err)
	slot := imageagent.SlotProjection{Slot: plan.Slots[0], Attempt: 1, Candidates: []imageagent.AssetCandidate{candidate}, Closure: closure}
	slot.Slot.Status = imageagent.SlotStatusAccepted
	p := imageagent.RunProjection{Run: run, Plan: plan, AssetCatalog: catalog, Slots: []imageagent.SlotProjection{slot}}
	p.ResultDigest, err = imageagent.ArtworkResultDigest(plan, p.Slots, nil)
	require.NoError(t, err)
	p.PendingCommand = &imageagent.PendingCommandReceipt{ActionID: uuid.NewString(), Kind: "approve_results", Phase: imageagent.ImageSetApprovalPublicationStarted, PlanRevision: 1, ResultDigest: p.ResultDigest}
	return &artworkFactsFixture{p: p, fact: fact, effect: imageagent.SlotEffectV3Attempt{Identity: id, IdempotencyKey: input.IdempotencyKey, InputFingerprint: imageagent.SlotExecutionFingerprint(input), Phase: imageagent.SlotEffectV3PublicationComplete, FinalManifest: imageagent.FinalManifest{Assets: []imageagent.PublishedAssetRef{ref}}, Published: result, ResultFingerprint: fingerprint}}
}
