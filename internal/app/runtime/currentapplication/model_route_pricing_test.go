package currentapplication

import (
	"context"
	"task-processor/internal/aicapability"
	d "task-processor/internal/aicapability/routeconfig"
	governed "task-processor/internal/integration/aicapability/einomodel"
	qualified "task-processor/internal/integration/aicapability/routeconfig"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestManagedCandidateUsesExistingPrivateProvisionOwner(t *testing.T) {
	c, in := contentProvisionFixture()
	c.ProductAgent.Enabled = true
	writer := c.ProductAgent.Database
	writer.User = "model_route_config_writer"
	writer.MaxConnections = 2
	admitted := c.ProductAgent.ContentTextPolicies["org"]
	admitted.AdmittedEndpointIdentityDigest = governed.EndpointIdentityDigest(in.BaseURL)
	c.ProductAgent.ContentTextPolicies["org"] = admitted
	candidate := c.ProductAgent.ContentTextPolicies["org"]
	candidate.Profile.ClientName = "content-candidate"
	in.ClientName = "content-candidate"
	c.ModelRoutePricing = &ModelRoutePricingConfig{TextWriter: &writer, Bindings: []qualified.Definition{{Scope: d.Scope{OrganizationID: "org", Capability: string(aicapability.OperationProductContentDecision)}, Text: &candidate}}}
	_, err := c.modelRouteDefinitions()
	require.NoError(t, err)
	require.NoError(t, ValidateTitleCredentialProvision(c, in))
	result, e := SaveTitleCredential(context.Background(), c, in, contentProvisionDB(t))
	require.NoError(t, e)
	require.NotEmpty(t, result.CredentialVersion)
	in.ClientName = "unadmitted"
	require.Error(t, ValidateTitleCredentialProvision(c, in))
}
func TestManagedSelectionKeepsExistingHalfRunCostCeiling(t *testing.T) {
	c := agentRuntimeConfig()
	writer := c.ProductAgent.Database
	writer.User = "model_route_config_writer"
	writer.MaxConnections = 2
	p := c.ProductAgent.TextPolicies["org"]
	p.Profile.ClientName = "selection"
	c.ProductAgent.SelectionTextPolicies = map[string]governed.RoutePolicy{"org": p}
	c.ModelRoutePricing = &ModelRoutePricingConfig{TextWriter: &writer}
	defs, e := c.modelRouteDefinitions()
	require.NoError(t, e)
	for _, def := range defs {
		if def.Scope.Capability == string(aicapability.OperationProductSelectionDecision) {
			require.Equal(t, c.ProductAgent.CostMicros/2, def.MaximumCostMicros)
		}
	}
}

func TestModelRoutePricingRequiresSameOwnerAndBoundedWriter(t *testing.T) {
	c := agentRuntimeConfig()
	writer := c.ProductAgent.Database
	writer.User = "model_route_config_writer"
	writer.MaxConnections = 2
	c.ModelRoutePricing = &ModelRoutePricingConfig{TextWriter: &writer}
	defs, e := c.modelRouteDefinitions()
	require.NoError(t, e)
	require.NotEmpty(t, defs)
	_, e = qualified.NewCatalog(defs)
	require.NoError(t, e)
	for _, change := range []func(*DatabaseConfig){func(v *DatabaseConfig) { v.Database = "other" }, func(v *DatabaseConfig) { v.User = c.ProductAgent.Database.User }, func(v *DatabaseConfig) { v.MaxConnections = 9 }} {
		bad := writer
		change(&bad)
		c.ModelRoutePricing.TextWriter = &bad
		_, e = c.modelRouteDefinitions()
		require.Error(t, e)
	}
	c.ModelRoutePricing.TextWriter = &writer
	def := defs[0]
	p := *def.Text
	p.Profile.ClientName = "other-binding"
	p.Profile.PromptVersion = "wrong"
	def.Text = &p
	c.ModelRoutePricing.Bindings = []qualified.Definition{def}
	_, e = c.modelRouteDefinitions()
	require.Error(t, e)
	c.ModelRoutePricing.Bindings = []qualified.Definition{{Scope: d.Scope{OrganizationID: "unadmitted", Capability: def.Scope.Capability}, Text: def.Text}}
	_, e = c.modelRouteDefinitions()
	require.Error(t, e)
}
