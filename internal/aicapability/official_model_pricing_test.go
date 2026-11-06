package aicapability

import (
	"math"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestReferenceTextTotalsPricesOnlyItsDeclaredBasicCounters(t *testing.T) {
	p := OfficialModelTariff{UnitVersion: AIPointMicrounitV1, PriceVersion: "reference-price-v1", FXVersion: "cny-identity-v1", Currency: "CNY", CNYMicrosPerCurrencyUnit: 1_000_000, Model: "deepseek-v4.1-flash", Channel: "bailian-beijing", UsageVersion: "reference-text-totals-v1", Rates: OfficialModelRates{InputText: 2_000_000, OutputText: 8_000_000}}
	u := BillableModelUsage{MappingVersion: p.UsageVersion, Complete: true, InputText: 32000, OutputText: 4000}
	r, err := p.ReserveTextMicropoints(u.InputText, u.OutputText)
	require.NoError(t, err)
	require.EqualValues(t, 9_600_000, r)
	a, err := p.SettlementMicropoints(u)
	require.NoError(t, err)
	require.Equal(t, r, a)
	// This strategy never treats an unobserved cache subdivision as an actual
	// provider cache fact and cannot accept a discount from another strategy.
	u.CachedInputText = 10
	_, err = p.SettlementMicropoints(u)
	require.Error(t, err)
	p.Rates.CachedInputText = 40_000
	require.Error(t, p.Validate())
	p.Rates.CachedInputText = 0
	p.Rates.OutputImage = 1
	require.Error(t, p.Validate(), "text estimation must not admit image usage")
}

func officialMiniPrice() OfficialModelTariff {
	return OfficialModelTariff{
		UnitVersion: AIPointMicrounitV1, PriceVersion: "official-mini-standard-v1",
		FXVersion: "synthetic-fx-7", Currency: "USD", CNYMicrosPerCurrencyUnit: 7_000_000,
		Model: "gpt-5.4-mini-2026-03-17", Channel: "openai-direct", UsageVersion: "openai-text-v1",
		Rates: OfficialModelRates{InputText: 750_000, CachedInputText: 75_000, OutputText: 4_500_000},
	}
}

func TestOfficialModelPricingUsesManufacturerPartitionsAndSingleFinalRounding(t *testing.T) {
	price := officialMiniPrice()
	usage := BillableModelUsage{MappingVersion: price.UsageVersion, Complete: true, InputText: 1000, CachedInputText: 200, OutputText: 100}
	// USD .001065 * synthetic FX 7 / CNY .01 per point = .7455 point.
	actual, err := price.SettlementMicropoints(usage)
	require.NoError(t, err)
	require.EqualValues(t, 745_500, actual)
	bound, err := price.ReservationMicropoints(usage)
	require.NoError(t, err)
	// Reservations cannot assume that an input cache hit will occur.
	require.EqualValues(t, 840_000, bound)

	price.CNYMicrosPerCurrencyUnit = 7_123_457
	actual, err = price.SettlementMicropoints(BillableModelUsage{MappingVersion: price.UsageVersion, Complete: true, InputText: 1, OutputText: 1})
	require.NoError(t, err)
	require.EqualValues(t, 3_739, actual)
	bound, err = price.ReservationMicropoints(BillableModelUsage{MappingVersion: price.UsageVersion, Complete: true, InputText: 1, OutputText: 1})
	require.NoError(t, err)
	require.EqualValues(t, 3_740, bound)
}

func TestOfficialModelPricingImagePartitionsAndSubMicropoint(t *testing.T) {
	price := officialMiniPrice()
	price.Model, price.UsageVersion = "gpt-image-2.5-flare-2026-09-08", "openai-images-direct-v1"
	price.Rates = OfficialModelRates{InputText: 5_000_000, InputImage: 8_000_000, OutputImage: 30_000_000}
	usage := BillableModelUsage{MappingVersion: price.UsageVersion, Complete: true, InputText: 20, InputImage: 50, OutputImage: 196}
	actual, err := price.SettlementMicropoints(usage)
	require.NoError(t, err)
	require.EqualValues(t, 4_466_000, actual)
	// Pure arithmetic on observed synthetic quantities does not admit a Flare request.
	price.Currency, price.FXVersion, price.CNYMicrosPerCurrencyUnit = "CNY", "cny-identity", 1_000_000
	price.Rates = OfficialModelRates{InputText: 1}
	actual, err = price.SettlementMicropoints(BillableModelUsage{MappingVersion: price.UsageVersion, Complete: true, InputText: 1})
	require.NoError(t, err)
	require.Zero(t, actual)
	bound, err := price.ReservationMicropoints(BillableModelUsage{MappingVersion: price.UsageVersion, Complete: true, InputText: 1})
	require.NoError(t, err)
	require.EqualValues(t, 1, bound)
}

func TestOfficialModelPricingRejectsMissingUnknownOrInconsistentEvidence(t *testing.T) {
	price := officialMiniPrice()
	for _, usage := range []BillableModelUsage{
		{InputText: 1},
		{MappingVersion: price.UsageVersion, Complete: true, InputText: -1},
		{MappingVersion: price.UsageVersion, Complete: true, InputText: 1, CachedInputText: 2},
		{MappingVersion: price.UsageVersion, Complete: true, InputImage: 1},
		{MappingVersion: price.UsageVersion, Complete: true, OutputImage: 1},
		{MappingVersion: price.UsageVersion, Complete: true, InputText: math.MaxInt64},
	} {
		_, err := price.SettlementMicropoints(usage)
		require.Error(t, err, "%+v", usage)
		_, err = price.ReservationMicropoints(usage)
		require.Error(t, err, "reservation must reject malformed bound: %+v", usage)
	}
	for _, mutate := range []func(*OfficialModelTariff){
		func(p *OfficialModelTariff) { p.UnitVersion = "whole-point-v1" },
		func(p *OfficialModelTariff) { p.PriceVersion = "" },
		func(p *OfficialModelTariff) { p.FXVersion = "" },
		func(p *OfficialModelTariff) { p.Model = "" },
		func(p *OfficialModelTariff) { p.Currency = "CNY" }, // identity FX is required for CNY
		func(p *OfficialModelTariff) { p.CNYMicrosPerCurrencyUnit = 0 },
		func(p *OfficialModelTariff) { p.Rates.InputText = math.MaxInt64 },
	} {
		candidate := price
		mutate(&candidate)
		_, err := candidate.ReservationMicropoints(BillableModelUsage{MappingVersion: price.UsageVersion, Complete: true, InputText: 1})
		require.Error(t, err)
	}
}

func TestOfficialModelPricingRejectsFinalInt64Overflow(t *testing.T) {
	price := officialMiniPrice()
	price.CNYMicrosPerCurrencyUnit = 1_000_000_000_000
	price.Rates.InputText = 1_000_000_000_000
	for _, round := range []func(BillableModelUsage) (int64, error){price.ReservationMicropoints, price.SettlementMicropoints} {
		_, err := round(BillableModelUsage{MappingVersion: price.UsageVersion, Complete: true, InputText: 1 << 30})
		require.Error(t, err)
	}
}

func TestOfficialModelPricingSnapshotDigestBindsEveryExecutionPriceFact(t *testing.T) {
	price := officialMiniPrice()
	digest, err := price.Digest()
	require.NoError(t, err)
	for _, mutate := range []func(*OfficialModelTariff){
		func(p *OfficialModelTariff) { p.PriceVersion += "-new" },
		func(p *OfficialModelTariff) { p.FXVersion += "-new" },
		func(p *OfficialModelTariff) { p.CNYMicrosPerCurrencyUnit++ },
		func(p *OfficialModelTariff) { p.Rates.CachedInputText++ },
		func(p *OfficialModelTariff) { p.Model += "-new" },
		func(p *OfficialModelTariff) { p.Channel += "-new" },
		func(p *OfficialModelTariff) { p.UsageVersion += "-new" },
	} {
		candidate := price
		mutate(&candidate)
		other, err := candidate.Digest()
		require.NoError(t, err)
		require.NotEqual(t, digest, other)
	}
}

func TestOfficialModelPricingObservedZeroDiffersFromMissingUsage(t *testing.T) {
	price := officialMiniPrice()
	actual, err := price.SettlementMicropoints(BillableModelUsage{MappingVersion: price.UsageVersion, Complete: true})
	require.NoError(t, err)
	require.Zero(t, actual)
	_, err = price.SettlementMicropoints(BillableModelUsage{})
	require.Error(t, err)
	_, err = price.ReservationMicropoints(BillableModelUsage{MappingVersion: price.UsageVersion, Complete: true})
	require.Error(t, err)
}

func TestOfficialPerImageCurrencyPriceUsesTheSameMicropointLedgerUnit(t *testing.T) {
	price := OfficialModelTariff{UnitVersion: AIPointMicrounitV1, PriceVersion: "synthetic-official-image", FXVersion: "synthetic-cny-identity", Currency: "CNY", CNYMicrosPerCurrencyUnit: 1000000, Model: "synthetic-qwen", Channel: "synthetic-channel", UsageVersion: "qwen-image-count-v1", Rates: OfficialModelRates{OutputImagePerItem: 200000}}
	billing := OfficialModelBilling{OfficialTariff: price, BoundEvidenceVersion: "synthetic-count-bound", UsageBound: BillableModelUsage{MappingVersion: "qwen-image-count-v1", Complete: true, OutputImageItems: 1}}
	reserved, err := billing.ReservationMicropoints("synthetic-qwen")
	require.NoError(t, err)
	require.EqualValues(t, 20000000, reserved, "CNY0.2 = 20 points; no independent points-per-image price")
	actual, err := billing.ActualMicropoints("synthetic-qwen", billing.UsageBound)
	require.NoError(t, err)
	require.Equal(t, reserved, actual)
	wrong := billing.UsageBound
	wrong.OutputImageItems = 2
	_, err = billing.ActualMicropoints("synthetic-qwen", wrong)
	require.Error(t, err)
	price.Rates.OutputImage = 1
	require.Error(t, price.Validate(), "image-count protocol cannot double-charge an inferred token dimension")
}
