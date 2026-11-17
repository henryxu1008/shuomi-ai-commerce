package currentapplication

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"task-processor/internal/agent"
	"task-processor/internal/agentconfig"
	"task-processor/internal/aicapability"
	d "task-processor/internal/aicapability/routeconfig"
	governed "task-processor/internal/integration/aicapability/einomodel"
	qualified "task-processor/internal/integration/aicapability/routeconfig"
	"task-processor/internal/integration/skillsandbox"
)

func TestScriptTrialConfigurationKeepsExistingSandboxAndRequiresExplicitPrivatePriceScope(t *testing.T) {
	cfg := agentRuntimeConfig()
	product := cfg.ProductAgent
	existingText := product.TextPolicies
	content := existingText["org"]
	content.Profile.PromptVersion = agentconfig.ContentPromptVersion
	content.Profile.OutputSchemaVersion = agentconfig.ContentActionVersion
	product.ContentTextPolicies = map[string]governed.RoutePolicy{"org": content}
	digest := "sha256:" + strings.Repeat("a", 64)
	sandbox := &skillsandbox.Config{
		Domain: "sandbox.example.test:8443", CAFile: "private/ca.pem", CredentialFile: "private/credential.json", TrialEvidence: digest,
		Descriptor: agent.ScriptDescriptor{Profile: "python-title-json-v1", Revision: "fixture-v1", NetworkRevision: "fixture-v1", ToolVersion: "v1.0.0", ImageDigest: digest, ExecdDigest: digest},
	}
	product.ScriptSandbox = sandbox

	t.Run("ordinary sandbox has no trial identity requirement", func(t *testing.T) {
		require.NoError(t, product.validatePolicyMetadata())
	})
	t.Run("explicit identity requires the same sandbox and content organization", func(t *testing.T) {
		identity := agent.ExecutionIdentity{OrganizationID: "org", ActorID: "user", MemberID: "member"}
		product.ScriptTrialIdentity = &identity
		defer func() { product.ScriptTrialIdentity, product.ScriptSandbox = nil, sandbox }()
		require.NoError(t, product.validatePolicyMetadata())
		identity.OrganizationID = "other"
		require.Error(t, product.validatePolicyMetadata())
		identity.OrganizationID = "org"
		product.ScriptSandbox = nil
		require.Error(t, product.validatePolicyMetadata())
	})

	writer := product.Database
	writer.User, writer.MaxConnections = "model_route_config_writer", 2
	const endpoint = "http://127.0.0.1:18086/v1"
	content.AdmittedEndpointIdentityDigest = governed.EndpointIdentityDigest(endpoint)
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	trialPrice := &qualified.PrivateTrialPrice{NotBefore: now, NotAfter: now.Add(30 * time.Minute), Tariff: content.Profile.OfficialTariff}
	binding := qualified.Definition{
		Scope: d.Scope{OrganizationID: "org", Capability: string(aicapability.OperationProductContentDecision)}, Text: &content,
		ConnectionEndpoint: endpoint, ConnectionID: "fixture-connection", ConnectionProvider: content.Profile.ProviderID,
		Channel: trialPrice.Tariff.Channel, PriceID: "fixture-price", PrivateTrialPrice: trialPrice,
	}
	cfg.ModelRoutePricing = &ModelRoutePricingConfig{Writer: &writer, Bindings: []qualified.Definition{binding}}
	t.Run("private price requires one exact trial tenant binding", func(t *testing.T) {
		product.TextPolicies = nil
		defer func() { product.TextPolicies, product.ScriptTrialIdentity = existingText, nil }()
		_, err := cfg.modelRouteDefinitions()
		require.Error(t, err, "unscoped private price")
		identity := agent.ExecutionIdentity{OrganizationID: "other", ActorID: "user", MemberID: "member"}
		product.ScriptTrialIdentity = &identity
		_, err = cfg.modelRouteDefinitions()
		require.Error(t, err, "private price scope differs from verified trial tenant")
		identity.OrganizationID = "org"
		defs, err := cfg.modelRouteDefinitions()
		require.NoError(t, err)
		require.Len(t, defs, 1)
		require.Equal(t, trialPrice, defs[0].PrivateTrialPrice)
		product.TextPolicies = existingText
		_, err = cfg.modelRouteDefinitions()
		require.Error(t, err, "a second consumer for the same tenant is outside the approved configuration")
	})
}
