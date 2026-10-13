package store

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"task-processor/internal/agent"
	"task-processor/internal/agentconfig"
	"task-processor/internal/imageagent"
	resourceadapter "task-processor/internal/integration/orgresource"
)

type artworkGenerationProvider struct {
	generationExecutionProvider
	result string
}

func (p *artworkGenerationProvider) QuoteSlot(context.Context, imageagent.SlotExecutionInput, imageagent.BudgetPolicy) (imageagent.SlotUsageQuote, error) {
	q, _ := p.generationExecutionProvider.QuoteSlot(context.Background(), imageagent.SlotExecutionInput{}, imageagent.BudgetPolicy{})
	q.Operations[0].Name = "generate_artwork"
	return q, nil
}

func (p *artworkGenerationProvider) GenerateQuotedSlot(ctx context.Context, input imageagent.SlotExecutionInput, quote imageagent.SlotUsageQuote) (imageagent.SlotGeneratedOutput, error) {
	p.dispatches++
	p.input = input
	if input.Artwork == nil || len(input.SourceBytes) != 0 || len(input.SourceReferences) != 0 || len(input.Slot.SourceAssetIDs) != 0 {
		return imageagent.SlotGeneratedOutput{}, imageagent.ErrValidation
	}
	if p.result == "unknown" {
		return imageagent.SlotGeneratedOutput{}, errors.New("controlled lost response")
	}
	if err := p.observer(ctx, imageagent.GenerationSuccess{ResponseID: "artwork-response", ResultDigest: strings.Repeat("c", 64), ResultURL: "https://example.test/generated.png"}); err != nil {
		return imageagent.SlotGeneratedOutput{}, err
	}
	if p.result == "download_failure" {
		return imageagent.SlotGeneratedOutput{}, errors.New("controlled download failure")
	}
	return imageagent.SlotGeneratedOutput{SlotID: input.Slot.ID, Attempt: input.Attempt, Assets: []imageagent.GeneratedAsset{{Operations: []string{"generate_artwork"}}}}, nil
}

func TestArtworkGenerationUsesOriginalEconomicsWithoutReadingAnOriginal(t *testing.T) {
	for _, outcome := range []string{"succeeded", "download_failure", "unknown", "revoked"} {
		t.Run(outcome, func(t *testing.T) {
			owner, commercial, limits, original := generationPointDatabases(t, true)
			ctx := context.Background()
			catalog, err := imageagent.NormalizeAssetCatalog(imageagent.AssetCatalog{})
			require.NoError(t, err)
			a := &imageagent.ArtworkPlan{Schema: imageagent.ArtworkSchema, Theme: "原创森林动物", Requirements: "独立图案", PromptVersion: imageagent.ArtworkSchema, Prompt: "Create original forest animal artwork.", Configuration: agent.ConfigurationSnapshotRef{Kind: agentconfig.SnapshotKind, ID: "00000000-0000-4000-8000-000000000001", Digest: strings.Repeat("a", 64)}, ModeSnapshotRef: agent.ModeSnapshotRef{Kind: agent.ExecutionModeSnapshotKind, ID: "00000000-0000-4000-8000-000000000002", Digest: strings.Repeat("b", 64)}, ConfigurationEpoch: "1", ParametersDigest: strings.Repeat("c", 64), InputDigest: strings.Repeat("d", 64), MaxPoints: 12, Quote: imageagent.ImageGenerationQuote{Provider: "grsai", Model: "gpt-image-2.5", Protocol: "grsai-json-sync-v1", Resolution: "1024x1024", Quality: "auto", PriceVersion: "price-1", Points: 12, RouteReference: "route-1", CredentialReference: "credential-1", ConfigurationVersion: "config-1"}}
			a.QuoteDigest = imageagent.ArtworkQuoteDigest(a.Quote)
			run := *manualRun("run-artwork", original.Identity.TenantID)
			run.BusinessTaskID = "artwork-context-1"
			run.ScopeProtocol, run.MemberID, run.TargetPlatform = imageagent.OrganizationScopeProtocol, original.MemberID, "sds"
			plan := imageagent.Plan{Artwork: a, Revision: 1, IdempotencyKey: "artwork-plan", CreatedBy: run.UserID, Slots: []imageagent.Slot{{ID: "artwork", Role: imageagent.SlotRoleMain, Brief: a.Theme, IdempotencyKey: "artwork-slot", Status: imageagent.SlotStatusPending}}}
			_, err = owner.InitializeRun(ctx, imageagent.ProjectionInitialization{Scope: imageagent.ScopeForRun(run), Run: run, Plan: plan, Catalog: catalog, Snapshot: imageagent.RunProjection{Run: run, Plan: plan}, CommitID: "initialize-artwork", EventType: "run.initialized", EventPayload: []byte(`{}`)})
			require.NoError(t, err)
			input := imageagent.SlotExecutionInput{Artwork: a, AssetCatalog: catalog, RunID: run.ID, TenantID: run.TenantID, UserID: run.UserID, PlanRevision: 1, Attempt: 1, TargetPlatform: "sds", Slot: plan.Slots[0], IdempotencyKey: "artwork-attempt", OrganizationIdentity: imageagent.ExecutionIdentity{ScopeProtocol: run.ScopeProtocol, TenantID: run.TenantID, UserID: run.UserID, MemberID: run.MemberID, RunID: run.ID, BusinessTaskID: run.BusinessTaskID}}
			id := imageagent.SlotExternalEffectIdentity{RunScope: imageagent.ScopeForRun(run), PlanRevision: 1, SlotID: "artwork", Attempt: 1}
			r := v3Reservation("artwork")
			r.Identity, r.InputFingerprint, r.IdempotencyKey = id, imageagent.SlotExecutionFingerprint(input), input.IdempotencyKey
			_, won, err := owner.ReserveSlotProviderV3(ctx, r)
			require.NoError(t, err)
			require.True(t, won)
			resources, err := resourceadapter.NewGormImageGenerationRepository(commercial, resourceadapter.TransactionConfig{}, owner, generationPointTestAuth{})
			require.NoError(t, err)
			auth := &generationExecutionAuth{}
			var economicOwner imageagent.GenerationResources = resources
			if outcome == "revoked" {
				economicOwner = afterGenerationReserve{GenerationResources: resources, after: func() { auth.denied = true }}
			}
			provider := &artworkGenerationProvider{result: outcome}
			sourceReads := 0
			executor, err := imageagent.NewGenerationExecution(imageagent.GenerationExecutionDependencies{Facts: owner, Resources: economicOwner, Authorizer: auth, MaxSourceBytes: 1024, ReadMemberLimit: func(context.Context, string, string) (imageagent.GenerationMemberLimit, error) {
				return imageagent.GenerationMemberLimit{Version: 1, MonthStart: original.MonthStart}, nil
			}, ReadSourceBytes: func(context.Context, imageagent.SlotExecutionInput) ([]byte, error) {
				sourceReads++
				return nil, errors.New("no original exists")
			}, PrepareProvider: func(_ context.Context, observe func(context.Context, imageagent.GenerationSuccess) error) (imageagent.PreparedGenerationProvider, error) {
				provider.observer = observe
				return imageagent.PreparedGenerationProvider{Executor: provider, Metadata: imageagent.GenerationProviderMetadata{RouteReference: "route-1", CredentialReference: "credential-1", ConfigurationVersion: "config-1", PromptVersion: "old-white-prompt", PriceVersion: "price-1", Points: 12}}, nil
			}, RevalidateProvider: func(context.Context, imageagent.GenerationProviderMetadata) error { return nil }})
			require.NoError(t, err)
			quote, err := executor.QuoteSlot(ctx, input, imageagent.BudgetPolicy{})
			require.NoError(t, err)
			output, generatedErr := executor.GenerateQuotedSlot(ctx, input, quote)
			require.Zero(t, sourceReads)
			fact, err := owner.ReadGenerationFact(ctx, id)
			require.NoError(t, err)
			require.Equal(t, imageagent.ArtworkSchema, fact.Intent.InputProtocol)
			require.Equal(t, fact.Intent.InputDigest, fact.Intent.SourceDigest)
			monthly, err := limits.ReadMonthlyLimit(ctx, run.TenantID, run.MemberID)
			require.NoError(t, err)
			switch outcome {
			case "revoked":
				require.Error(t, generatedErr)
				require.Zero(t, provider.dispatches)
				require.Equal(t, imageagent.GenerationNoEffect, fact.State)
				require.Zero(t, monthly.Consumed)
				require.Zero(t, monthly.Reserved)
			case "unknown":
				require.Error(t, generatedErr)
				require.Equal(t, imageagent.GenerationUnknown, fact.State)
				require.EqualValues(t, 12, monthly.Reserved)
				require.Zero(t, monthly.Consumed)
			default:
				require.Equal(t, imageagent.GenerationSucceeded, fact.State)
				require.EqualValues(t, 12, monthly.Consumed)
				require.Zero(t, monthly.Reserved)
				if outcome == "succeeded" {
					require.NoError(t, generatedErr)
					require.NoError(t, output.Assets[0].Artwork.ValidateForInput(input))
				} else {
					require.Error(t, generatedErr)
				}
			}
			_, err = executor.GenerateQuotedSlot(ctx, input, quote)
			require.Error(t, err)
			if outcome != "revoked" {
				require.Equal(t, 1, provider.dispatches, "original attempt never permits a second paid POST")
			}
		})
	}
}
