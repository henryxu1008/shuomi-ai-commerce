package currentapplication

import (
	"testing"

	"github.com/stretchr/testify/require"
	"task-processor/internal/agentconfig"
	"task-processor/internal/aicapability"
	d "task-processor/internal/aicapability/routeconfig"
	governed "task-processor/internal/integration/aicapability/einomodel"
	qualified "task-processor/internal/integration/aicapability/routeconfig"
)

func TestRetainedOwnIntakeFourVersionFieldsUseOriginalAdmission(t *testing.T) {
	c := agentRuntimeConfig()
	c.ProductCollections = true
	p := c.ProductAgent.TextPolicies["org"]
	c.ProductAgent.TextPolicies = nil
	p.Profile.ClientName, p.Profile.ProviderID, p.Profile.ModelID = "mini", "openai", aicapability.SingleImageMiniModel
	p.Profile.AdapterPolicyVersion, p.Profile.UsageMappingVersion = aicapability.SingleImageMiniAdapterPolicy, "openai-text-v1"
	p.AdmittedEndpointIdentityDigest = governed.EndpointIdentityDigest("https://api.openai.com/v1")
	p.Profile.MaximumPromptTokens, p.Profile.MaximumCompletionTokens = 24000, 4000
	p.Profile.PromptVersion, p.Profile.OutputSchemaVersion = "own-product-intake-prompt-v1", "own-product-intake-candidate-v1"
	c.ProductAgent.OwnIntakePolicies = map[string]governed.RoutePolicy{"org": p}
	writer := c.ProductAgent.Database
	writer.User, writer.MaxConnections = "model_route_config_writer", 2
	binding := p
	c.ModelRoutePricing = &ModelRoutePricingConfig{Writer: &writer, Bindings: []qualified.Definition{{Scope: d.Scope{OrganizationID: "org", Capability: string(aicapability.OperationOwnProductIntake)}, Text: &binding, ConnectionID: "original-openai-connection", ConnectionProvider: "openai", ConnectionEndpoint: "https://api.openai.com/v1", Manufacturer: "openai", Region: "global", ServiceType: "standard-sync", Channel: "openai-direct", PriceID: "original-price", ReferenceVersion: "original-reference", BoundEvidenceVersion: "original-evidence"}}}
	require.Error(t, c.validate(), "v1 must remain refused by the current v3 consumer")
	beforePolicy := p
	beforeBinding := *c.ModelRoutePricing.Bindings[0].Text
	p.Profile.PromptVersion, p.Profile.OutputSchemaVersion = agentconfig.OwnIntakePromptVersion, agentconfig.OwnIntakeActionVersion
	c.ProductAgent.OwnIntakePolicies["org"] = p
	require.Error(t, c.validate(), "half-aligned consumer and explicit binding must refuse")
	aligned := *c.ModelRoutePricing.Bindings[0].Text
	aligned.Profile.PromptVersion, aligned.Profile.OutputSchemaVersion = agentconfig.OwnIntakePromptVersion, agentconfig.OwnIntakeActionVersion
	c.ModelRoutePricing.Bindings[0].Text = &aligned
	require.NoError(t, c.validate())
	_, err := c.modelRouteDefinitions()
	require.NoError(t, err)
	// Undo only the allowed metadata in local comparison values: every other
	// field (credential, rates, FX, evidence and budgets) must be identical.
	p.Profile.PromptVersion, p.Profile.OutputSchemaVersion = beforePolicy.Profile.PromptVersion, beforePolicy.Profile.OutputSchemaVersion
	aligned.Profile.PromptVersion, aligned.Profile.OutputSchemaVersion = beforeBinding.Profile.PromptVersion, beforeBinding.Profile.OutputSchemaVersion
	require.Equal(t, beforePolicy, p)
	require.Equal(t, beforeBinding, aligned)
	c.ModelRoutePricing.Bindings = append(c.ModelRoutePricing.Bindings, c.ModelRoutePricing.Bindings[0])
	_, err = c.modelRouteDefinitions()
	require.Error(t, err, "ambiguous bindings cannot be matched by an array position")
}
