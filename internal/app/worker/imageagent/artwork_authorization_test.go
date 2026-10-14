package imageagentworker

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
	"testing"
	"time"
)

type dispatchArtworkContexts struct {
	dispatchSourceContexts
	mode         agent.ExecutionMode
	changed      bool
	artworkCalls int
}

func (c *dispatchArtworkContexts) AuthorizeArtworkExecution(_ context.Context, _ imageagent.ExecutionIdentity, _ imageagent.RunProjection) error {
	c.artworkCalls++
	if c.changed {
		return agentconfig.ErrChanged
	}
	if c.mode == agent.ModeReadOnly {
		return agentconfig.ErrForbidden
	}
	return nil
}

type unusedArtworkFacts struct {
	imageagent.GenerationFactRepository
}
type unusedArtworkResources struct{ imageagent.GenerationResources }

func admittedWorkerArtwork(t *testing.T) (generationProjectionReader, imageagent.SlotExecutionInput) {
	t.Helper()
	id := imageagent.ExecutionIdentity{ScopeProtocol: imageagent.OrganizationScopeProtocol, TenantID: "org", UserID: "actor", MemberID: "member", RunID: uuid.NewString(), BusinessTaskID: uuid.NewString()}
	quote := imageagent.ImageGenerationQuote{Provider: "grsai", Model: "gpt-image-2.5", Protocol: "grsai-json-sync-v1", Resolution: "1024x1024", Quality: "auto", Points: 12, PriceVersion: "price", RouteReference: "route", CredentialReference: "credential", ConfigurationVersion: "config"}
	art := &imageagent.ArtworkPlan{Schema: imageagent.ArtworkSchema, Theme: "原创动物", PromptVersion: imageagent.ArtworkSchema, Prompt: "Create original artwork", Configuration: agent.ConfigurationSnapshotRef{Kind: agentconfig.SnapshotKind, ID: uuid.NewString(), Digest: strings.Repeat("a", 64)}, ModeSnapshotRef: agent.ModeSnapshotRef{Kind: agent.ExecutionModeSnapshotKind, ID: uuid.NewString(), Digest: strings.Repeat("b", 64)}, ConfigurationEpoch: "1", ParametersDigest: strings.Repeat("c", 64), InputDigest: strings.Repeat("d", 64), Quote: quote, MaxPoints: 12, QuoteDigest: imageagent.ArtworkQuoteDigest(quote)}
	plan := imageagent.Plan{Revision: 1, Artwork: art, IdempotencyKey: uuid.NewString(), CreatedBy: id.UserID, Slots: []imageagent.Slot{{ID: "artwork", Role: imageagent.SlotRoleMain, Brief: art.Theme, IdempotencyKey: "artwork", Status: imageagent.SlotStatusPending}}}
	limits := agentconfig.ImageRunLimits{Images: 1, Points: 12, ElapsedSeconds: 300}
	digest, err := imageagent.ArtworkPlanDigest(plan)
	require.NoError(t, err)
	now := time.Now().UTC()
	receipt := agentconfig.ImageRunAdmissionReceipt{ID: uuid.NewString(), Command: agentconfig.ImageRunAdmissionCommand{Scope: agent.Scope{OrganizationID: id.TenantID, ActorID: id.UserID}, Snapshot: art.Configuration, MemberID: id.MemberID, RunID: id.RunID, ConfirmActionID: uuid.NewString(), SourceDigest: art.InputDigest, InputDigest: art.InputDigest, PlanDigest: digest, QuoteDigest: art.QuoteDigest, Limits: limits}, AdmittedAt: now, Deadline: now.Add(300 * time.Second)}
	raw, err := json.Marshal(receipt)
	require.NoError(t, err)
	hash := sha256.Sum256(raw)
	receipt.Digest = hex.EncodeToString(hash[:])
	run := imageagent.Run{ID: id.RunID, ScopeProtocol: id.ScopeProtocol, TenantID: id.TenantID, UserID: id.UserID, MemberID: id.MemberID, BusinessTaskID: id.BusinessTaskID, TargetPlatform: "sds", StartedAt: now, ImageAdmission: &receipt, Budget: imageagent.ImageSetBudget(limits)}
	require.NoError(t, imageagent.ValidateArtworkAdmission(run, plan))
	catalog, err := imageagent.NormalizeAssetCatalog(imageagent.AssetCatalog{})
	require.NoError(t, err)
	return generationProjectionReader{run: run, plan: plan}, imageagent.SlotExecutionInput{Artwork: art, RunID: id.RunID, TenantID: id.TenantID, UserID: id.UserID, OrganizationIdentity: id, TargetPlatform: "sds", PlanRevision: 1, Attempt: 1, Slot: plan.Slots[0], IdempotencyKey: "artwork-attempt", AssetCatalog: catalog}
}
func TestAdmittedArtworkChecksCurrentModeAndConfigurationBeforeProviderPreparation(t *testing.T) {
	repo, input := admittedWorkerArtwork(t)
	for _, name := range []string{"default", "auto_approve", "full_access", "read_only", "disabled_or_changed_configuration"} {
		t.Run(name, func(t *testing.T) {
			guard := &dispatchArtworkContexts{mode: agent.ExecutionMode(name), changed: name == "disabled_or_changed_configuration"}
			authorizer := imageSetGenerationAuthorizer{repository: repo, live: &generationLiveAuthorizer{}, sources: guard}
			err := authorizer.AuthorizeExecution(context.Background(), input.OrganizationIdentity)
			require.Equal(t, 1, guard.artworkCalls)
			require.Zero(t, guard.calls, "source-free creation never authorizes product image sources")
			if name != "read_only" && !guard.changed {
				require.NoError(t, err)
				return
			}
			require.Error(t, err)
			providerPreparations := 0
			executor, err := imageagent.NewGenerationExecution(imageagent.GenerationExecutionDependencies{Facts: unusedArtworkFacts{}, Resources: unusedArtworkResources{}, Authorizer: authorizer, MaxSourceBytes: 1, ReadMemberLimit: func(context.Context, string, string) (imageagent.GenerationMemberLimit, error) {
				panic("denied generation cannot read limits")
			}, ReadSourceBytes: func(context.Context, imageagent.SlotExecutionInput) ([]byte, error) {
				panic("source-free generation cannot read originals")
			}, PrepareProvider: func(context.Context, func(context.Context, imageagent.GenerationSuccess) error) (imageagent.PreparedGenerationProvider, error) {
				providerPreparations++
				return imageagent.PreparedGenerationProvider{}, imageagent.ErrCommandBlocked
			}, RevalidateProvider: func(context.Context, imageagent.GenerationProviderMetadata) error { return nil }})
			require.NoError(t, err)
			_, err = executor.QuoteSlot(context.Background(), input, imageagent.BudgetPolicy{})
			require.Error(t, err)
			_, err = executor.GenerateQuotedSlot(context.Background(), input, imageagent.SlotUsageQuote{})
			require.Error(t, err)
			require.Zero(t, providerPreparations, "current mode/configuration refusal precedes all reserve/dispatch/provider work")
		})
	}
}
