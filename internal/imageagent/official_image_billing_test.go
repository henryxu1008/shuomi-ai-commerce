package imageagent

import (
	"github.com/stretchr/testify/require"
	"task-processor/internal/aicapability"
	"testing"
)

func TestFixedImagePointPriceCannotAuthorizeCurrentClaim(t *testing.T) {
	intent := generationTestIntent()
	intent.Billing = ImageBilling{}
	_, err := NewGenerationFact(intent)
	require.Error(t, err, "old per-image point numbers have no official price or bounded usage proof")
}
func TestOfficialImageChargeUsesSameMicropointPriceAndCompleteActualUsage(t *testing.T) {
	price := aicapability.OfficialModelTariff{UnitVersion: aicapability.AIPointMicrounitV1, PriceVersion: "synthetic-official", FXVersion: "synthetic-fx", Currency: "CNY", CNYMicrosPerCurrencyUnit: 1000000, Model: "synthetic-image", Channel: "synthetic-channel", UsageVersion: "openai-images-direct-v1", Rates: aicapability.OfficialModelRates{InputText: 10000, InputImage: 10000, OutputImage: 20000}}
	billing := ImageBilling{OfficialTariff: price, BoundEvidenceVersion: "synthetic-bound", UsageBound: aicapability.BillableModelUsage{MappingVersion: "openai-images-direct-v1", Complete: true, InputText: 3, InputImage: 3, OutputImage: 4}}
	maximum, err := billing.ReservationMicropoints("synthetic-image")
	require.NoError(t, err)
	require.EqualValues(t, 14, maximum)
	success := GenerationSuccess{UsageKnown: true, InputTokens: 3, OutputTokens: 2, TotalTokens: 5, BillableUsage: aicapability.BillableModelUsage{MappingVersion: "openai-images-direct-v1", Complete: true, InputText: 1, InputImage: 2, OutputImage: 2}}
	actual, err := billing.ActualMicropoints("synthetic-image", success.BillableUsage)
	require.NoError(t, err)
	require.EqualValues(t, 7, actual)
	success.BillableUsage.Complete = false
	_, err = billing.ActualMicropoints("synthetic-image", success.BillableUsage)
	require.Error(t, err, "observed result without complete billing evidence keeps the original reservation")
	success.BillableUsage.Complete = true
	success.BillableUsage.OutputImage = 5
	success.OutputTokens = 5
	success.TotalTokens = 8
	_, err = billing.ActualMicropoints("synthetic-image", success.BillableUsage)
	require.Error(t, err, "actual usage exceeding frozen bounds cannot debit extra points")
}
