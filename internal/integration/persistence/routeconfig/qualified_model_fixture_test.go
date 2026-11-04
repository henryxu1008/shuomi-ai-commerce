package routeconfigstore

import (
	"task-processor/internal/aicapability"
	d "task-processor/internal/aicapability/routeconfig"
	"time"
)

func syntheticQualifiedModel(q d.ModelQualification) d.ModelQualification {
	if q.Category == d.ModelCategoryImage {
		billing := aicapability.OfficialModelBilling{OfficialTariff: q.Price, BoundEvidenceVersion: q.BoundEvidenceVersion, UsageBound: aicapability.BillableModelUsage{MappingVersion: q.Price.UsageVersion, Complete: true, InputText: 100, OutputText: 100}}
		maximum, _ := billing.ReservationMicropoints(q.Model)
		q.Image = &d.ImageProfile{Model: q.Model, PriceVersion: q.Price.PriceVersion, Points: maximum, Billing: billing}
	} else {
		q.Text = &aicapability.ModelProfile{ClientName: q.Reference.ID, ProviderID: "synthetic", AdapterKind: "openai-compatible", ModelID: q.Model, EndpointIdentityDigest: "endpoint", CredentialVersion: "credential", RoutingPolicyVersion: "routing", AdapterPolicyVersion: "adapter", PromptVersion: "prompt", OutputSchemaVersion: "schema", UsageMappingVersion: q.Price.UsageVersion, CostPricingVersion: "cost", OfficialTariff: q.Price, Currency: "USD", InputMicrosPerMillion: 1, OutputMicrosPerMillion: 1, MaximumPromptTokens: 100, MaximumCompletionTokens: 100, MaximumInputBytes: 1024, MaximumOutputBytes: 1024, DeadlineBound: time.Second}
	}
	return q
}
