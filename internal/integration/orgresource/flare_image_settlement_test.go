package orgresourceadapter

import (
	"context"
	"github.com/stretchr/testify/require"
	"strings"
	"task-processor/internal/aicapability"
	"task-processor/internal/imageagent"
	"task-processor/internal/ledger/orgresource"
	"testing"
)

func flareSettlementBilling() imageagent.ImageBilling {
	return imageagent.ImageBilling{OfficialTariff: aicapability.OfficialModelTariff{UnitVersion: aicapability.AIPointMicrounitV1, PriceVersion: "reference-v1", FXVersion: "identity-v1", Currency: "CNY", CNYMicrosPerCurrencyUnit: 1000000, Model: aicapability.FlareImageModel, Channel: "openai-direct", UsageVersion: aicapability.ReferenceImagesTotalsV1, Rates: aicapability.OfficialModelRates{InputText: 400, InputImage: 800, OutputImage: 400}}, ReferenceEstimate: aicapability.ReferenceImageEstimate{InputTokens: 100, OutputTokens: 100, OperationScope: aicapability.ReferenceImageOperationScope, Size: "1024x1024", Quality: "auto", AllowAutoSize: true, MaximumPromptBytes: 65536, MaximumReferenceImages: 3, MaximumReferenceBytes: 16 << 20, MaximumReferencePixels: 64000000}}
}

func TestFlareReferenceSettlementUsesTheOriginalReservationAndReplaysOnce(t *testing.T) {
	for _, tc := range []struct {
		name                  string
		input, output, actual int64
		fine, over            bool
	}{{"D1", 100, 50, 10, false, false}, {"complete", 100, 50, 8, true, false}, {"explicit zero", 0, 0, 0, false, false}, {"over reserve", 200, 100, 0, false, true}} {
		t.Run(tc.name, func(t *testing.T) {
			db, repo, owner, res := reservedImageSettlementFixture(t, flareSettlementBilling())
			ctx := context.Background()
			f, _, err := owner.fact.BeginDispatch()
			require.NoError(t, err)
			usage := aicapability.BillableModelUsage{MappingVersion: aicapability.ReferenceImagesTotalsV1, Complete: true, OutputImage: tc.output, ReferenceImageTotals: aicapability.ReferenceImageTotals{Known: true, Input: tc.input, Output: tc.output, Total: tc.input + tc.output}}
			if tc.fine {
				usage.InputText = 50
				usage.InputImage = 50
				usage.ReferenceImageTotals.InputTextKnown = true
				usage.ReferenceImageTotals.InputImageKnown = true
			}
			owner.fact, err = f.RecordSuccess(imageagent.GenerationSuccess{BillableUsage: usage, UsageKnown: true, InputTokens: tc.input, OutputTokens: tc.output, TotalTokens: tc.input + tc.output, ResponseID: "synthetic", ResultDigest: strings.Repeat("a", 64), ResultUnavailable: "invalid_result"})
			require.NoError(t, err)
			settled, err := repo.FinalizeImageGeneration(ctx, owner.fact.Intent.Identity)
			if tc.over {
				require.Error(t, err)
				require.Equal(t, imageagent.GenerationUnknown, owner.fact.State)
				assertImageSettlementStillReserved(t, db, owner.fact, res)
				return
			}
			require.NoError(t, err)
			require.EqualValues(t, tc.actual, settled.Points)
			require.EqualValues(t, 12, settled.ReservedMicropoints)
			assertBucket(t, db, "org-1", orgresource.ResourceAIPoint, 100-tc.actual, 0, tc.actual)
			again, err := repo.FinalizeImageGeneration(ctx, owner.fact.Intent.Identity)
			require.NoError(t, err)
			require.Equal(t, settled, again)
			owner.fact, err = owner.fact.BindSettlement(settled)
			require.NoError(t, err)
			projected, err := owner.fact.ProjectReferenceImageCost(imageagent.SlotUsageReceipt{Actual: imageagent.UsageVector{Images: 1}})
			require.NoError(t, err)
			require.Equal(t, imageagent.UsageCostReferenceEstimate, projected.CostBasis)
			require.Equal(t, aicapability.ReferenceImageCNYMicros(tc.actual), projected.Actual.CostMicros)
			assertTableCount(t, db, "saas_organization_resource_operations", 3)
		})
	}
}
