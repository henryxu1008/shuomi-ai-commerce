package routeconfiguration

import (
	"testing"

	"github.com/stretchr/testify/require"
	d "task-processor/internal/aicapability/routeconfig"
	governed "task-processor/internal/integration/aicapability/einomodel"
)

func TestVIPBindingRequiresExactExistingEnterpriseCredential(t *testing.T) {
	sc := d.Scope{OrganizationID: "org-a", Capability: d.ImageCapability}
	endpoint := "https://grsaiapi.com/v1/api/generate"
	c, err := NewCatalog([]Definition{{Scope: sc, Image: &ImageAdmission{ClientName: "image-vip", Provider: "grsai", Model: "gpt-image-2-vip", Endpoint: endpoint, CredentialVersion: "credential-v1", MaximumPointsPerImage: 12}}})
	require.NoError(t, err)
	row := d.Credential{OrganizationID: "org-a", ClientName: "image-vip", Model: "gpt-image-2-vip", APIStyle: "grsai", Version: "credential-v1", EndpointDigest: governed.EndpointIdentityDigest(endpoint), Reference: "db:1:1:image-vip", TimeoutSeconds: 120, Enabled: true, KeyPresent: true}
	b, err := c.Qualify(sc, "image-vip", row)
	require.NoError(t, err)
	require.Equal(t, "gpt-image-2-vip", b.Model)
	for _, field := range []string{"organization", "enabled", "key", "model", "version", "endpoint", "protocol"} {
		bad := row
		switch field {
		case "organization":
			bad.OrganizationID = "org-b"
		case "enabled":
			bad.Enabled = false
		case "key":
			bad.KeyPresent = false
		case "model":
			bad.Model = "gpt-image-2.5"
		case "version":
			bad.Version = "rotated"
		case "endpoint":
			bad.EndpointDigest = "changed"
		case "protocol":
			bad.APIStyle = "openai"
		}
		_, err = c.Qualify(sc, "image-vip", bad)
		require.ErrorIs(t, err, d.ErrNeedsConfiguration, field)
	}
}
