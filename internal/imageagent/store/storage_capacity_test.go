package store

import (
	"context"
	"github.com/stretchr/testify/require"
	"strings"
	"task-processor/internal/imageagent"
	resourceadapter "task-processor/internal/integration/orgresource"
	capacitystore "task-processor/internal/integration/persistence/storagecapacity"
	"task-processor/internal/ledger/storagecapacity"
	"testing"
)

func TestGenerationStorageCapacityPrecedesPaidDispatchAndPreservesUnknown(t *testing.T) {
	for _, mode := range []string{"unknown_retained", "capacity_rejected", "invalid_result_closed"} {
		t.Run(mode, func(t *testing.T) {
			deny := mode == "capacity_rejected"
			owner, commercial, limits, intent := generationPointDatabases(t, true)
			ctx := context.Background()
			require.NoError(t, capacitystore.AutoMigrate(commercial))
			capacity, e := capacitystore.New(commercial)
			require.NoError(t, e)
			if deny {
				_, e = capacity.ReserveWrite(ctx, storagecapacity.WriteIntent{OrganizationID: intent.Identity.TenantID, Owner: "fixture", OperationID: "full", Fingerprint: storagecapacity.Digest("full"), MaxBytes: storagecapacity.BaseBytes, MaxObjects: 1})
				require.NoError(t, e)
			}
			resources, e := resourceadapter.NewGormImageGenerationRepository(commercial, resourceadapter.TransactionConfig{}, owner, generationPointTestAuth{})
			require.NoError(t, e)
			catalog, e := owner.GetAssetCatalog(ctx, intent.Identity.RunScope)
			require.NoError(t, e)
			input := imageagent.SlotExecutionInput{RunID: intent.Identity.RunID, TenantID: intent.Identity.TenantID, UserID: intent.Identity.OwnerUserID, PlanRevision: 1, Attempt: 1, TargetPlatform: "product", ImagePolicyContext: &imageagent.ImagePolicyContext{Country: "zz", Family: "default", SceneCategory: "general"}, Slot: imageagent.Slot{ID: "slot-1", Role: imageagent.SlotRoleMain, SourceAssetIDs: []string{"source-1"}}, AssetCatalog: catalog, OrganizationIdentity: imageagent.ExecutionIdentity{ScopeProtocol: imageagent.OrganizationScopeProtocol, RunID: intent.Identity.RunID, TenantID: intent.Identity.TenantID, UserID: intent.Identity.OwnerUserID, MemberID: intent.MemberID, BusinessTaskID: "task-" + intent.Identity.RunID}}
			provider := &generationExecutionProvider{unknown: mode != "invalid_result_closed"}
			executor, e := imageagent.NewGenerationExecution(imageagent.GenerationExecutionDependencies{Facts: owner, Resources: resources, Authorizer: &generationExecutionAuth{}, MaxSourceBytes: 1024, ReadMemberLimit: func(context.Context, string, string) (imageagent.GenerationMemberLimit, error) {
				return imageagent.GenerationMemberLimit{Version: 1, MonthStart: intent.MonthStart}, nil
			}, ReadSourceBytes: func(context.Context, imageagent.SlotExecutionInput) ([]byte, error) {
				return []byte("exact-source-bytes"), nil
			}, PrepareProvider: func(_ context.Context, observer func(context.Context, imageagent.GenerationSuccess) error) (imageagent.PreparedGenerationProvider, error) {
				provider.observer = observer
				return imageagent.PreparedGenerationProvider{Executor: provider, Metadata: imageagent.GenerationProviderMetadata{Provider: "grsai", Model: "gpt-image-2.5", Protocol: "grsai-json-sync-v1", RouteReference: "route-1", CredentialReference: "credential-1", ConfigurationVersion: "config-1", PromptVersion: "prompt-1", PriceVersion: "price-1", Points: 12}}, nil
			}, RevalidateProvider: func(context.Context, imageagent.GenerationProviderMetadata) error { return nil }})
			require.NoError(t, e)
			executor.ConfigureStorageCapacity(capacity, 1024)
			quote, e := executor.QuoteSlot(ctx, input, imageagent.BudgetPolicy{})
			require.NoError(t, e)
			_, e = executor.GenerateQuotedSlot(ctx, input, quote)
			require.Error(t, e)
			if deny {
				require.ErrorIs(t, e, storagecapacity.ErrExceeded, "native no-dispatch wrapper must preserve the quota cause")
			}
			fact, e := owner.ReadGenerationFact(ctx, intent.Identity)
			require.NoError(t, e)
			points, e := limits.ReadMonthlyLimit(ctx, intent.Identity.TenantID, intent.MemberID)
			require.NoError(t, e)
			if deny {
				require.Zero(t, provider.dispatches)
				require.Equal(t, imageagent.GenerationNoEffect, fact.State)
				require.Zero(t, points.Reserved)
				require.Zero(t, points.Consumed)
			} else if mode == "unknown_retained" {
				require.Equal(t, 1, provider.dispatches)
				require.Equal(t, imageagent.GenerationUnknown, fact.State)
				summary, e := capacity.ReadSummary(ctx, intent.Identity.TenantID)
				require.NoError(t, e)
				require.Equal(t, "1024", summary.ReservedBytes)
				_, e = executor.GenerateQuotedSlot(ctx, input, quote)
				require.Error(t, e)
				require.Equal(t, 1, provider.dispatches)
			} else {
				require.Equal(t, imageagent.GenerationSucceeded, fact.State)
				require.Equal(t, "invalid_result", fact.Success.ResultUnavailable)
				require.Equal(t, "committed", fact.Settlement.State)
				require.EqualValues(t, 12, points.Consumed)
				summary, err := capacity.ReadSummary(ctx, intent.Identity.TenantID)
				require.NoError(t, err)
				require.Equal(t, "0", summary.ReservedBytes)
				require.Equal(t, "0", summary.OccupiedBytes)
				recovery, err := imageagent.NewGenerationRecovery(owner, resources)
				require.NoError(t, err)
				imageagent.ConfigureGenerationRecoveryStorage(recovery, capacity)
				_, err = recovery.ReconcileExistingGeneration(ctx, intent.Identity)
				require.NoError(t, err)
				require.Equal(t, 1, provider.dispatches)
			}
		})
	}
}

func TestGenerationRecoveryClosesOnlyInvalidSettledOutputCapacity(t *testing.T) {
	for _, invalid := range []bool{false, true} {
		t.Run(map[bool]string{false: "valid_output_retained", true: "invalid_output_closed"}[invalid], func(t *testing.T) {
			owner, commercial, limits, intent := generationPointDatabases(t)
			ctx := context.Background()
			require.NoError(t, capacitystore.AutoMigrate(commercial))
			capacity, err := capacitystore.New(commercial)
			require.NoError(t, err)
			fact, err := owner.ReadGenerationFact(ctx, intent.Identity)
			require.NoError(t, err)
			reservation, err := capacity.ReserveWrite(ctx, storagecapacity.WriteIntent{OrganizationID: intent.Identity.TenantID, Owner: "image_agent", OperationID: imageagent.StorageGenerationOperation(intent.Identity), Fingerprint: fact.Fingerprint, MaxBytes: 1024, MaxObjects: 1})
			require.NoError(t, err)
			require.NoError(t, capacity.ClaimDispatch(ctx, intent.Identity.TenantID, reservation.ID))
			resources, err := resourceadapter.NewGormImageGenerationRepository(commercial, resourceadapter.TransactionConfig{}, owner, generationPointTestAuth{})
			require.NoError(t, err)
			pointsReceipt, err := resources.ReserveImageGeneration(ctx, intent.Identity)
			require.NoError(t, err)
			_, err = owner.BindGenerationReservation(ctx, intent, pointsReceipt)
			require.NoError(t, err)
			_, won, err := owner.BeginGenerationDispatch(ctx, intent)
			require.NoError(t, err)
			require.True(t, won)
			success := imageagent.GenerationSuccess{ResponseID: "response-1", ResultDigest: strings.Repeat("c", 64), ResultURL: "https://images.example.test/image.png"}
			if invalid {
				success.ResultURL = ""
				success.ResultUnavailable = "invalid_result"
			}
			_, err = owner.RecordGenerationSuccess(ctx, intent, success)
			require.NoError(t, err)
			recovery, err := imageagent.NewGenerationRecovery(owner, resources)
			require.NoError(t, err)
			imageagent.ConfigureGenerationRecoveryStorage(recovery, capacity)
			for i := 0; i < 2; i++ {
				state, err := recovery.ReconcileExistingGeneration(ctx, intent.Identity)
				require.NoError(t, err)
				require.Equal(t, imageagent.GenerationSucceeded, state)
			}
			summary, err := capacity.ReadSummary(ctx, intent.Identity.TenantID)
			require.NoError(t, err)
			expected := "1024"
			if invalid {
				expected = "0"
			}
			require.Equal(t, expected, summary.ReservedBytes)
			require.Equal(t, "0", summary.OccupiedBytes)
			points, err := limits.ReadMonthlyLimit(ctx, intent.Identity.TenantID, intent.MemberID)
			require.NoError(t, err)
			require.EqualValues(t, 12, points.Consumed)
			require.Zero(t, points.Reserved)
			if invalid {
				success.ResultURL = "https://images.example.test/image.png"
				success.ResultUnavailable = ""
				_, err = owner.RecordGenerationSuccess(ctx, intent, success)
				require.ErrorIs(t, err, imageagent.ErrRevisionConflict, "terminal invalid result cannot later acquire a publication locator")
			}
		})
	}
}
