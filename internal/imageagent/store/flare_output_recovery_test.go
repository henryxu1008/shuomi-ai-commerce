package store

import (
	"context"
	"fmt"
	"github.com/stretchr/testify/require"
	"strings"
	"task-processor/internal/aicapability"
	"task-processor/internal/imageagent"
	resourceadapter "task-processor/internal/integration/orgresource"
	"testing"
)

func TestFlareBundleRecoveryUsesOriginalCommittedCostAndNoDispatch(t *testing.T) {
	owner, commercial, limits, intent := generationPointDatabases(t, true)
	ctx := context.Background()
	scope, policy := initializeBudgetedSlotEffectRun(t, owner, "run-flare-recovery")
	require.NoError(t, owner.db.Model(&runRecord{}).Where("id = ?", scope.RunID).Updates(map[string]any{"scope_protocol": imageagent.OrganizationScopeProtocol, "member_id": intent.MemberID}).Error)
	reservation := budgetedV3Reservation(scope, policy, 1, "frozen-flare-quote")
	reservation.Quote.Maximum = imageagent.UsageVector{Images: 1, AgentSteps: 1, ModelCalls: 1, CostMicros: 1}
	reservation.Quote.Operations[0].Maximum = reservation.Quote.Maximum
	_, won, err := owner.ReserveSlotProviderV3(ctx, reservation)
	require.NoError(t, err)
	require.True(t, won)
	cat, err := owner.GetAssetCatalog(ctx, scope)
	require.NoError(t, err)
	intent.Identity = reservation.Identity
	intent.CatalogHash = cat.Manifest.Hash
	intent.Provider = "openai"
	intent.Model = aicapability.FlareImageModel
	intent.Protocol = aicapability.FlareImageProtocol
	intent.PriceVersion = "reference-v1"
	intent.Billing = flareStoreBilling()
	intent.Billing.OfficialTariff.Rates = aicapability.OfficialModelRates{InputText: 400, InputImage: 800, OutputImage: 400}
	_, err = owner.PrepareGenerationIntent(ctx, intent)
	require.NoError(t, err)
	resources, err := resourceadapter.NewGormImageGenerationRepository(commercial, resourceadapter.TransactionConfig{}, owner, generationPointTestAuth{})
	require.NoError(t, err)
	reserved, err := resources.ReserveImageGeneration(ctx, intent.Identity)
	require.NoError(t, err)
	_, err = owner.BindGenerationReservation(ctx, intent, reserved)
	require.NoError(t, err)
	_, won, err = owner.BeginGenerationDispatch(ctx, intent)
	require.NoError(t, err)
	require.True(t, won)
	proof := imageagent.GenerationSuccess{UsageKnown: true, InputTokens: 100, OutputTokens: 50, TotalTokens: 150, ResponseID: "response", RequestID: "vendor-request", ResultDigest: strings.Repeat("c", 64), ResultLocatorKind: "artifact_bundle", BillableUsage: aicapability.BillableModelUsage{MappingVersion: aicapability.ReferenceImagesTotalsV1, Complete: true, OutputImage: 50, ReferenceImageTotals: aicapability.ReferenceImageTotals{Known: true, Input: 100, Output: 50, Total: 150}}}
	_, err = owner.RecordGenerationSuccess(ctx, intent, proof)
	require.NoError(t, err)
	settled, err := resources.FinalizeImageGeneration(ctx, intent.Identity)
	require.NoError(t, err)
	require.EqualValues(t, 10, settled.Points)
	fact, err := owner.BindGenerationSettlement(ctx, intent, settled)
	require.NoError(t, err)
	manifest := v3StagingManifest()
	ownerKey, err := imageagent.ArtifactOwnerKey(scope.OwnerUserID)
	require.NoError(t, err)
	manifest.Assets[0].ObjectKey = fmt.Sprintf("image-agent/staging/%s/%s/%s/1/slot-1/1/0-%s.png", scope.TenantID, ownerKey, scope.RunID, manifest.Assets[0].SHA256)
	manifest.Assets[0].ProviderReceiptID = proof.ResultDigest
	for replay := 0; replay < 2; replay++ {
		recovered, err := owner.RecoverGenerationStaging(ctx, reservation, intent, fact.TerminalProofDigest(), manifest)
		require.NoError(t, err)
		require.Equal(t, imageagent.SlotEffectV3StagingPrepared, recovered.Phase)
		require.Equal(t, imageagent.UsageCostReferenceEstimate, recovered.Receipt.CostBasis)
		require.EqualValues(t, 1, recovered.Receipt.Actual.CostMicros)
		require.Equal(t, []string{proof.ResultDigest}, recovered.Receipt.ProviderRequestIDs)
	}
	_, won, err = owner.BeginGenerationDispatch(ctx, intent)
	require.NoError(t, err)
	require.False(t, won)
	monthly, err := limits.ReadMonthlyLimit(ctx, intent.Identity.TenantID, intent.MemberID)
	require.NoError(t, err)
	require.EqualValues(t, 10, monthly.Consumed)
}
