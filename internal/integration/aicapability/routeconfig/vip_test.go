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
	c, err := NewCatalog([]Definition{{Scope: sc, Image: &ImageAdmission{UnitVersion: "ai-micropoint-v1", MaximumMicropoints: 100000000, ClientName: "image-vip", Provider: "grsai", Model: "gpt-image-2-vip", Endpoint: endpoint, CredentialVersion: "credential-v1"}}})
	require.NoError(t, err)
	row := d.Credential{OrganizationID: "org-a", EndpointDigest: governed.EndpointIdentityDigest(endpoint), Reference: "db:1:1:image-vip", Enabled: true, KeyPresent: true}
	bindTestChannel(c, &row)
	b, err := c.Qualify(sc, "image-vip", row)
	require.NoError(t, err)
	require.Equal(t, "gpt-image-2-vip", b.Model)
	for _, field := range []string{"organization", "enabled", "key", "connection", "reference", "endpoint", "provider"} {
		bad := row
		switch field {
		case "organization":
			bad.OrganizationID = "org-b"
		case "enabled":
			bad.Enabled = false
		case "key":
			bad.KeyPresent = false
		case "connection":
			bad.ConnectionID = "other"
		case "reference":
			bad.Reference = "invalid"
		case "endpoint":
			bad.EndpointDigest = "changed"
		case "provider":
			bad.Provider = "other"
		}
		_, err = c.Qualify(sc, "image-vip", bad)
		require.ErrorIs(t, err, d.ErrNeedsConfiguration, field)
	}
}
