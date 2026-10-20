package currentapplication

import (
	"context"
	"slices"
	"task-processor/internal/agentconfig"
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
	c.ProductCollections = true
	writer := c.ProductAgent.Database
	writer.User = "model_route_config_writer"
	writer.MaxConnections = 2
	p := c.ProductAgent.TextPolicies["org"]
	p.Profile.ClientName = "selection"
	c.ProductAgent.SelectionTextPolicies = map[string]governed.RoutePolicy{"org": p}
	c.ModelRoutePricing = &ModelRoutePricingConfig{TextWriter: &writer}
	defs, e := c.modelRouteDefinitions()
	require.NoError(t, e)
	found := false
	for _, def := range defs {
		if def.Scope.Capability == string(aicapability.OperationProductSelectionDecision) {
			found = true
			require.Equal(t, c.ProductAgent.CostMicros/2, def.MaximumCostMicros)
		}
	}
	require.True(t, found)
}

func TestModelRoutePricingAdmitsOnlyAssembledConsumers(t *testing.T) {
	for _, tc := range []struct {
		name       string
		capability aicapability.Operation
		prompt     string
		schema     string
		policies   func(*ProductAgentConfig, map[string]governed.RoutePolicy)
		enable     func(*Config)
	}{
		{"materials", aicapability.OperationListingMaterialsDecision, agentconfig.MaterialsPromptVersion, agentconfig.MaterialsActionVersion,
			func(p *ProductAgentConfig, policies map[string]governed.RoutePolicy) {
				p.MaterialsTextPolicies = policies
			},
			func(c *Config) { c.SupplyChain = &SupplyChainConfig{} }},
		{"selection", aicapability.OperationProductSelectionDecision, agentconfig.SelectionPromptVersion, agentconfig.SelectionActionVersion,
			func(p *ProductAgentConfig, policies map[string]governed.RoutePolicy) {
				p.SelectionTextPolicies = policies
			},
			func(c *Config) { c.ProductCollections = true }},
		{"diagnosis", aicapability.OperationShopDiagnosisDecision, agentconfig.DiagnosisPromptVersion, agentconfig.DiagnosisActionVersion,
			func(p *ProductAgentConfig, policies map[string]governed.RoutePolicy) {
				p.DiagnosisTextPolicies = policies
			},
			func(c *Config) { c.StoreCenter = &StoreCenterConfig{Enabled: true, OperationsCockpit: true} }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			for _, enabled := range []bool{false, true} {
				c := agentRuntimeConfig()
				writer := c.ProductAgent.Database
				writer.User, writer.MaxConnections = "model_route_config_writer", 2
				c.ModelRoutePricing = &ModelRoutePricingConfig{TextWriter: &writer}
				policy := c.ProductAgent.TextPolicies["org"]
				policy.Profile.ClientName = tc.name
				policy.Profile.PromptVersion, policy.Profile.OutputSchemaVersion = tc.prompt, tc.schema
				tc.policies(c.ProductAgent, map[string]governed.RoutePolicy{"org": policy})
				if enabled {
					tc.enable(c)
				}
				defs, err := c.modelRouteDefinitions()
				require.NoError(t, err)
				catalog, err := qualified.NewCatalog(defs)
				require.NoError(t, err)
				scope := d.Scope{OrganizationID: "org", Capability: string(tc.capability)}
				require.Equal(t, enabled, slices.Contains(catalog.Scopes("org"), scope), "consumer enabled=%v", enabled)
				candidate := policy
				candidate.Profile.ClientName += "-candidate"
				c.ModelRoutePricing.Bindings = []qualified.Definition{{Scope: scope, Text: &candidate}}
				_, err = c.modelRouteDefinitions()
				if enabled {
					require.NoError(t, err)
				} else {
					require.Error(t, err, "candidate descriptors must not reopen an unavailable consumer")
					c.ModelRoutePricing.Bindings = nil
					c.ProductAgent.TextPolicies = nil
					_, err = c.modelRouteDefinitions()
					require.Error(t, err, "management requires at least one assembled text consumer")
				}
			}
		})
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
