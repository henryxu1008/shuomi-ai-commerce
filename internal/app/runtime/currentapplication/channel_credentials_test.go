package currentapplication

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
	"task-processor/internal/aicapability"
	credential "task-processor/internal/aicapability/channelcredential"
	d "task-processor/internal/aicapability/routeconfig"
	governed "task-processor/internal/integration/aicapability/einomodel"
	qualified "task-processor/internal/integration/aicapability/routeconfig"
)

func channelProvisionFixture() (*Config, ChannelCredentialProvision) {
	c := agentRuntimeConfig()
	c.ProductAgent.Enabled = false
	c.ChannelCredentials = &ChannelCredentialsConfig{ProtectionKeyID: "fixture-v1", ProtectionKey: base64.StdEncoding.EncodeToString([]byte(strings.Repeat("x", 32)))}
	writer := c.ProductAgent.Database
	writer.User = "model_route_config_writer"
	writer.MaxConnections = 2
	p := c.ProductAgent.TextPolicies["org"]
	p.Profile.EndpointIdentityDigest = governed.EndpointIdentityDigest("https://grsaiapi.com/v1")
	c.ProductAgent.TextPolicies["org"] = p
	c.ModelRoutePricing = &ModelRoutePricingConfig{Writer: &writer, Bindings: []qualified.Definition{{Scope: d.Scope{OrganizationID: "org", Capability: string(aicapability.OperationProductAgentDecision)}, ConnectionID: "shared", ConnectionProvider: "grsai", Channel: "grsai-direct", Region: "global", Text: &p}}}

	keyWriter := c.ProductAgent.Database
	keyWriter.User = "title_credential_writer"
	keyWriter.MaxConnections = 2
	return c, ChannelCredentialProvision{Command: credential.Command{OrganizationID: "org", ActorID: "operator", Key: "request-1", ConnectionID: "shared", Provider: "grsai", Channel: "grsai-direct", Account: "service", Region: "global", BaseURL: "https://grsaiapi.com/v1", Enabled: true}, APIKey: "synthetic-key", WriterDatabase: keyWriter}
}

func TestChannelProvisionAdmissionSharesDeclaredConnectionWithoutModelOwnership(t *testing.T) {
	c, in := channelProvisionFixture()
	require.NoError(t, ValidateChannelCredentialProvision(c, in))
	candidate := c.ModelRoutePricing.Bindings[0]
	copy := *candidate.Text
	copy.Profile.ClientName = "other-model"
	copy.Profile.ModelID = "other-admitted-model"
	candidate.Text = &copy
	c.ModelRoutePricing.Bindings = append(c.ModelRoutePricing.Bindings, candidate)
	require.NoError(t, ValidateChannelCredentialProvision(c, in), "multiple explicit models may reference one connection")
	for _, name := range []string{"org", "connection", "origin", "region", "provider", "writer", "database", "pool", "CAS", "key"} {
		t.Run(name, func(t *testing.T) {
			bad := in
			switch name {
			case "org":
				bad.OrganizationID = "outside"
			case "connection":
				bad.ConnectionID = "undeclared"
			case "origin":
				bad.BaseURL = "https://unadmitted.example/v1"
			case "region":
				bad.Region = "other"
			case "provider":
				bad.Provider = "other"
			case "writer":
				bad.WriterDatabase.User = c.ProductAgent.Database.User
			case "database":
				bad.WriterDatabase.Database = "other"
			case "pool":
				bad.WriterDatabase.MaxConnections = 3
			case "CAS":
				bad.ExpectedRevision = -1
			case "key":
				bad.APIKey = ""
			}
			require.Error(t, ValidateChannelCredentialProvision(c, bad))
		})
	}
	raw, e := json.Marshal(credential.Metadata{OrganizationID: "org", Reference: "public-reference"})
	require.NoError(t, e)
	require.NotContains(t, string(raw), in.APIKey)
}

func TestClosedAIManagementDoesNotOpenOrDecryptChannelSecrets(t *testing.T) {
	c, _ := channelProvisionFixture()
	c.ChannelCredentials.ProtectionKey = "unavailable"
	calls := 0
	reader, db, e := openChannelCredentials(context.Background(), c, func(context.Context, DatabaseConfig) (*gorm.DB, error) {
		calls++
		t.Fatal("closed AI must not open secret pool")
		return nil, nil
	})
	require.NoError(t, e)
	require.Nil(t, reader)
	require.Nil(t, db)
	require.Zero(t, calls)
	require.NoError(t, c.validateChannelCredentials())
	_, e = c.modelRouteDefinitions()
	require.NoError(t, e, "model management metadata remains available")
	c.ProductAgent.Enabled = true
	require.Error(t, c.validateChannelCredentials(), "execution needs an admitted bounded reader and protection")
}

func TestChannelManagementWriterIsIndependentAndClosedAIStaysClosed(t *testing.T) {
	c, in := channelProvisionFixture()
	c.ChannelCredentials.ManagementWriter = &in.WriterDatabase
	c.ModelRoutePricing.Bindings[0].ConnectionEndpoint = in.BaseURL
	c.ModelRoutePricing.Bindings[0].ReferenceVersion = "v1"
	require.NoError(t, c.validateChannelCredentials())
	definitions, e := c.modelRouteDefinitions()
	require.NoError(t, e)
	connections, e := channelManagementDefinitions(definitions, true)
	require.NoError(t, e)
	require.Len(t, connections, 1)
	require.Equal(t, in.BaseURL, connections[0].BaseURL)
	calls := 0
	_, _, e = openChannelCredentials(context.Background(), c, func(context.Context, DatabaseConfig) (*gorm.DB, error) { calls++; return nil, nil })
	require.NoError(t, e)
	require.Zero(t, calls, "management must not open the execution secret reader")
	for _, change := range []func(*DatabaseConfig){func(w *DatabaseConfig) { w.User = "product_agent_runtime" }, func(w *DatabaseConfig) { w.Database = "another-db" }, func(w *DatabaseConfig) { w.MaxConnections = 3 }} {
		bad := in.WriterDatabase
		change(&bad)
		c.ChannelCredentials.ManagementWriter = &bad
		require.Error(t, c.validateChannelCredentials())
	}
	c.ChannelCredentials.ManagementWriter = &in.WriterDatabase
	c.ChannelCredentials.ProtectionKey = "bad"
	require.Error(t, c.validateChannelCredentials(), "management writer requires protection even with execution off")
}

func TestChannelManagementEndpointCannotBeGuessedOrSuppliedByTheBrowser(t *testing.T) {
	c, in := channelProvisionFixture()
	c.ModelRoutePricing.Bindings[0].ReferenceVersion = "v1"
	defs, e := c.modelRouteDefinitions()
	require.NoError(t, e)
	_, e = channelManagementDefinitions(defs, true)
	require.Error(t, e, "a digest is not a trusted endpoint URL")
	c.ModelRoutePricing.Bindings[0].ConnectionEndpoint = "https://unadmitted.example.test/v1"
	defs, e = c.modelRouteDefinitions()
	require.NoError(t, e)
	_, e = channelManagementDefinitions(defs, true)
	require.Error(t, e)
	c.ModelRoutePricing.Bindings[0].ConnectionEndpoint = in.BaseURL
	defs, e = c.modelRouteDefinitions()
	require.NoError(t, e)
	_, e = channelManagementDefinitions(defs, true)
	require.NoError(t, e)
}

func TestChannelManagementSharedImageEndpointQualifiesTextRegardlessOfDefinitionOrder(t *testing.T) {
	c, in := channelProvisionFixture()
	text := c.ModelRoutePricing.Bindings[0]
	text.ReferenceVersion = "v1"
	image := qualified.Definition{Scope: d.Scope{OrganizationID: "org", Capability: d.ImageCapability}, ConnectionID: text.ConnectionID, ConnectionProvider: text.ConnectionProvider, Channel: text.Channel, Region: text.Region, ReferenceVersion: "v1", Image: &qualified.ImageAdmission{ClientName: "image", Model: "fixture-image", Endpoint: in.BaseURL}}
	for _, defs := range [][]qualified.Definition{{text, image}, {image, text}} {
		connections, e := channelManagementDefinitions(defs, true)
		require.NoError(t, e)
		require.Len(t, connections, 1)
		require.Len(t, connections[0].Models, 2)
		require.Equal(t, in.BaseURL, connections[0].BaseURL)
	}
	image.Image.Endpoint = "https://not-admitted.example.test/v1"
	_, e := channelManagementDefinitions([]qualified.Definition{text, image}, true)
	require.Error(t, e, "an image definition cannot widen the text endpoint digest")
}
