package productsourcing

import (
	"task-processor/internal/aicapability"
	"task-processor/internal/imageagent"
)

// Explicit synthetic price/usage for the existing offline owner regressions.
// This helper is never linked into production or used as missing-usage fallback.
func syntheticImageBilling(model, version string, maximum int64) imageagent.ImageBilling {
	return imageagent.ImageBilling{BoundEvidenceVersion: "synthetic-bound-v1", OfficialTariff: aicapability.OfficialModelTariff{UnitVersion: aicapability.AIPointMicrounitV1, PriceVersion: version, FXVersion: "synthetic-fx-identity", Currency: "CNY", CNYMicrosPerCurrencyUnit: 1000000, Model: model, Channel: "synthetic-channel", UsageVersion: "openai-images-direct-v1", Rates: aicapability.OfficialModelRates{InputText: maximum * 50, OutputImage: maximum * 50}}, UsageBound: syntheticImageUsage(100, 100)}
}
func syntheticImageUsage(input, output int64) aicapability.BillableModelUsage {
	return aicapability.BillableModelUsage{MappingVersion: "openai-images-direct-v1", Complete: true, InputText: input, OutputImage: output}
}
