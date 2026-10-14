package temporal

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"task-processor/internal/agent"
	"task-processor/internal/agentconfig"
	"task-processor/internal/imageagent"
	"task-processor/internal/imageagent/store"
	"task-processor/internal/product/collection"
)

type artworkRecoveryReceipts struct {
	receipt      imageagent.ArtworkAdoptionReceipt
	err          error
	publications int
}

func (r *artworkRecoveryReceipts) ReadArtworkReceipt(context.Context, imageagent.ExecutionIdentity) (imageagent.ArtworkAdoptionReceipt, error) {
	return r.receipt, r.err
}
func (r *artworkRecoveryReceipts) PublishArtwork(context.Context, imageagent.PublishArtworkInput) (imageagent.ArtworkAdoptionReceipt, error) {
	r.publications++
	return r.receipt, r.err
}

func artworkRecoveryFixture(t *testing.T) (*Activities, imageagent.ExecutionIdentity, PersistRunStateActivityInput, *artworkRecoveryReceipts, *approvalRecoveryAuthorization) {
	t.Helper()
	hash := strings.Repeat("a", 64)
	quote := imageagent.ImageGenerationQuote{Provider: "grsai", Model: "gpt-image-2.5", Protocol: "grsai-json-sync-v1", Resolution: "1024x1024", Quality: "auto", Points: 12, PriceVersion: "price", RouteReference: "route", CredentialReference: "credential", ConfigurationVersion: "1"}
	art := &imageagent.ArtworkPlan{Schema: imageagent.ArtworkSchema, Theme: "原创狐狸", PromptVersion: imageagent.ArtworkSchema, Prompt: "Create original fox artwork", Configuration: agent.ConfigurationSnapshotRef{Kind: agentconfig.SnapshotKind, ID: uuid.NewString(), Digest: hash}, ModeSnapshotRef: agent.ModeSnapshotRef{Kind: agent.ExecutionModeSnapshotKind, ID: uuid.NewString(), Digest: hash}, ConfigurationEpoch: "1", ParametersDigest: hash, InputDigest: hash, MaxPoints: 12, Quote: quote, QuoteDigest: imageagent.ArtworkQuoteDigest(quote)}
	plan := imageagent.Plan{Revision: 1, CreatedBy: "actor", IdempotencyKey: "plan", Artwork: art, Slots: []imageagent.Slot{{ID: "artwork", Role: imageagent.SlotRoleMain, Brief: art.Theme, IdempotencyKey: "slot", Status: imageagent.SlotStatusPending}}}
	now := time.Now().UTC()
	run := imageagent.Run{ID: uuid.NewString(), TenantID: "org", UserID: "actor", MemberID: "member", BusinessTaskID: "creative", ScopeProtocol: imageagent.OrganizationScopeProtocol, TargetPlatform: "sds", Mode: imageagent.RunModeManual, IdempotencyKey: "prepare", Status: imageagent.RunStatusAwaitingFinalApproval, CurrentNode: "approve_results", Version: 1, ActivePlanRevision: 1, StartedAt: now, MaxConcurrentSlots: 1}
	limits := agentconfig.ImageRunLimits{Images: 1, Points: 12, ElapsedSeconds: 300}
	run.Budget = imageagent.ImageSetBudget(limits)
	digest, err := imageagent.ArtworkPlanDigest(plan)
	require.NoError(t, err)
	admission := agentconfig.ImageRunAdmissionReceipt{ID: uuid.NewString(), AdmittedAt: now, Deadline: now.Add(300 * time.Second), Command: agentconfig.ImageRunAdmissionCommand{Scope: agent.Scope{OrganizationID: run.TenantID, ActorID: run.UserID}, Snapshot: art.Configuration, MemberID: run.MemberID, RunID: run.ID, ConfirmActionID: uuid.NewString(), SourceDigest: art.InputDigest, InputDigest: art.InputDigest, PlanDigest: digest, QuoteDigest: art.QuoteDigest, Limits: limits}}
	raw, err := json.Marshal(admission)
	require.NoError(t, err)
	sum := sha256.Sum256(raw)
	admission.Digest = hex.EncodeToString(sum[:])
	run.ImageAdmission = &admission
	require.NoError(t, imageagent.ValidateArtworkAdmission(run, plan))
	owner, err := imageagent.ArtifactOwnerKey(run.UserID)
	require.NoError(t, err)
	provenance := &imageagent.ArtworkProvenance{Protocol: imageagent.ArtworkSchema, TenantID: run.TenantID, OwnerUserID: run.UserID, RunID: run.ID, SlotID: "artwork", PlanRevision: 1, Attempt: 1, InputDigest: hash, IntentID: hash, Fingerprint: hash}
	slot := imageagent.SlotProjection{Slot: plan.Slots[0], Attempt: 1, Closure: &imageagent.ImageSlotClosure{Kind: "settled", IntentID: hash, Fingerprint: hash, SettlementProofDigest: hash, Points: 12}, Candidates: []imageagent.AssetCandidate{{AssetID: "artwork-asset", Width: 1024, Height: 1024, DurableAsset: imageagent.DurableAssetIdentity{ObjectKey: "image-agent/public/org/" + owner + "/" + run.ID + "/1/artwork/1/0-" + hash + ".png", SHA256: hash}, Artwork: provenance, Operations: []string{"generate_artwork"}, GenerationProof: &imageagent.ImageGenerationProof{IntentID: hash, Fingerprint: hash, SettlementProofDigest: hash, Points: 12}}}}
	slot.Slot.Status = imageagent.SlotStatusAccepted
	result, err := imageagent.ArtworkResultDigest(plan, []imageagent.SlotProjection{slot}, nil)
	require.NoError(t, err)
	pending := &imageagent.PendingCommandReceipt{ActionID: uuid.NewString(), Kind: "approve_results", Phase: imageagent.ImageSetApprovalPublicationStarted, PlanRevision: 1, ResultDigest: result, Status: "pending", Attempt: 1}
	catalog, err := imageagent.NormalizeAssetCatalog(imageagent.AssetCatalog{})
	require.NoError(t, err)
	repo := store.NewMemoryRepository()
	_, err = repo.InitializeRun(context.Background(), imageagent.ProjectionInitialization{Scope: imageagent.ScopeForRun(run), Run: run, Plan: plan, Catalog: catalog, Snapshot: imageagent.RunProjection{Run: run, Plan: plan, Slots: []imageagent.SlotProjection{slot}, ResultDigest: result, PendingCommand: pending}, CommitID: "prepared", EventType: "run.initialized", EventPayload: []byte(`{}`)})
	require.NoError(t, err)
	id := imageagent.ExecutionIdentity{ScopeProtocol: run.ScopeProtocol, RunID: run.ID, TenantID: run.TenantID, UserID: run.UserID, MemberID: run.MemberID, BusinessTaskID: run.BusinessTaskID}
	receipt := &artworkRecoveryReceipts{receipt: imageagent.ArtworkAdoptionReceipt{RunID: run.ID, PlanRevision: 1, SlotID: "artwork", ArtifactHash: hash, OperationID: "operation", BatchID: "batch", ItemID: "item", ProductKey: "product", PublicationID: "operation", SourceVersion: 1}}
	auth := &approvalRecoveryAuthorization{revoked: true}
	a := &Activities{repository: repo, artworkPublisher: receipt, executionAuthorizer: auth}
	complete := PersistRunStateActivityInput{RunID: run.ID, Identity: id, PlanRevision: 1, CurrentNode: "complete", CommitID: "complete-original", Projection: WorkflowResult{Status: imageagent.RunStatusCompleted, Plan: plan, Slots: []imageagent.SlotProjection{slot}, ResultDigest: result}}
	return a, id, complete, receipt, auth
}

func TestArtworkCommittedCompletionAndPendingRecoverySurviveRevocation(t *testing.T) {
	a, id, complete, receipts, auth := artworkRecoveryFixture(t)
	current, err := a.repository.GetProjection(context.Background(), imageagent.RunScope{TenantID: id.TenantID, OwnerUserID: id.UserID, RunID: id.RunID})
	require.NoError(t, err)
	pending := clonePendingReceipt(current.PendingCommand)
	pending.Phase = string(updatePhaseApprovalPersistComplete)
	pending.Attempt++
	pending.Status = "failed"
	require.NoError(t, a.PersistPendingCommand(context.Background(), PersistPendingCommandActivityInput{RunID: id.RunID, Identity: id, Receipt: pending, CommandIngress: current.CommandIngress, CommitID: "pending-original"}))
	require.NoError(t, a.PersistRunState(context.Background(), complete))
	require.NoError(t, a.PersistRunState(context.Background(), complete))
	current, err = a.repository.GetProjection(context.Background(), imageagent.RunScope{TenantID: id.TenantID, OwnerUserID: id.UserID, RunID: id.RunID})
	require.NoError(t, err)
	require.Equal(t, imageagent.RunStatusCompleted, current.Run.Status)
	require.Nil(t, current.PendingCommand)
	require.Zero(t, auth.calls)
	require.Zero(t, receipts.publications)
}

func TestArtworkRecoveryCannotAuthorizeUncommittedOrChangedFacts(t *testing.T) {
	for _, kind := range []string{"missing", "read_error", "receipt_run", "receipt_hash", "actor", "plan", "result", "pending_action"} {
		t.Run(kind, func(t *testing.T) {
			a, id, complete, r, _ := artworkRecoveryFixture(t)
			switch kind {
			case "missing":
				r.err = collection.ErrNotFound
			case "read_error":
				r.err = context.DeadlineExceeded
			case "receipt_run":
				r.receipt.RunID = uuid.NewString()
			case "receipt_hash":
				r.receipt.ArtifactHash = strings.Repeat("b", 64)
			case "actor":
				complete.Identity.UserID = "other"
			case "plan":
				complete.PlanRevision++
			case "result":
				complete.Projection.ResultDigest = strings.Repeat("b", 64)
			case "pending_action":
				current, err := a.repository.GetProjection(context.Background(), imageagent.RunScope{TenantID: id.TenantID, OwnerUserID: id.UserID, RunID: id.RunID})
				require.NoError(t, err)
				pending := clonePendingReceipt(current.PendingCommand)
				pending.ActionID = uuid.NewString()
				require.Error(t, a.PersistPendingCommand(context.Background(), PersistPendingCommandActivityInput{RunID: id.RunID, Identity: id, Receipt: pending, CommandIngress: current.CommandIngress, CommitID: "changed"}))
				return
			}
			require.Error(t, a.PersistRunState(context.Background(), complete))
			require.Zero(t, r.publications)
		})
	}
}
