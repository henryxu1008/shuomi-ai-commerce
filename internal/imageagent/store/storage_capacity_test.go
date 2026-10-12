package store

import (
	"context"
	"github.com/stretchr/testify/require"
	"task-processor/internal/imageagent"
	resourceadapter "task-processor/internal/integration/orgresource"
	capacitystore "task-processor/internal/integration/persistence/storagecapacity"
	"task-processor/internal/ledger/storagecapacity"
	"testing"
)

func TestGenerationStorageCapacityPrecedesPaidDispatchAndPreservesUnknown(t *testing.T) {
	for _, deny := range []bool{false, true} {
		t.Run(map[bool]string{false: "unknown_retained", true: "capacity_rejected"}[deny], func(t *testing.T) {
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
			provider := &generationExecutionProvider{unknown: true}
			executor, e := imageagent.NewGenerationExecution(imageagent.GenerationExecutionDependencies{Facts: owner, Resources: resources, Authorizer: &generationExecutionAuth{}, MaxSourceBytes: 1024, ReadMemberLimit: func(context.Context, string, string) (imageagent.GenerationMemberLimit, error) {
				return imageagent.GenerationMemberLimit{Version: 1, MonthStart: intent.MonthStart}, nil
			}, ReadSourceBytes: func(context.Context, imageagent.SlotExecutionInput) ([]byte, error) {
				return []byte("exact-source-bytes"), nil
			}, PrepareProvider: func(_ context.Context, observer func(context.Context, imageagent.GenerationSuccess) error) (imageagent.PreparedGenerationProvider, error) {
				provider.observer = observer
				return imageagent.PreparedGenerationProvider{Executor: provider, Metadata: imageagent.GenerationProviderMetadata{RouteReference: "route-1", CredentialReference: "credential-1", ConfigurationVersion: "config-1", PromptVersion: "prompt-1", PriceVersion: "price-1", Points: 12}}, nil
			}, RevalidateProvider: func(context.Context, imageagent.GenerationProviderMetadata) error { return nil }})
			require.NoError(t, e)
			executor.ConfigureStorageCapacity(capacity, 1024)
			quote, e := executor.QuoteSlot(ctx, input, imageagent.BudgetPolicy{})
			require.NoError(t, e)
			_, e = executor.GenerateQuotedSlot(ctx, input, quote)
			require.Error(t, e)
			fact, e := owner.ReadGenerationFact(ctx, intent.Identity)
			require.NoError(t, e)
			points, e := limits.ReadMonthlyLimit(ctx, intent.Identity.TenantID, intent.MemberID)
			require.NoError(t, e)
			if deny {
				require.Zero(t, provider.dispatches)
				require.Equal(t, imageagent.GenerationNoEffect, fact.State)
				require.Zero(t, points.Reserved)
				require.Zero(t, points.Consumed)
			} else {
				require.Equal(t, 1, provider.dispatches)
				require.Equal(t, imageagent.GenerationUnknown, fact.State)
				summary, e := capacity.ReadSummary(ctx, intent.Identity.TenantID)
				require.NoError(t, e)
				require.Equal(t, "1024", summary.ReservedBytes)
				_, e = executor.GenerateQuotedSlot(ctx, input, quote)
				require.Error(t, e)
				require.Equal(t, 1, provider.dispatches)
			}
		})
	}
}
