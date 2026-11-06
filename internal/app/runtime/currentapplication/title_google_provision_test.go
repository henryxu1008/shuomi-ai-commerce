package currentapplication

import (
	"github.com/stretchr/testify/require"
	governed "task-processor/internal/integration/aicapability/einomodel"
	"task-processor/internal/integration/googleinteractions"
	"testing"
)

func TestGoogleConnectionRequiresTheOfficialVersionedOrigin(t *testing.T) {
	c, in := channelProvisionFixture()
	p := c.ProductAgent.TextPolicies["org"]
	p.Profile.ProviderID, p.Profile.AdapterKind, p.Profile.ModelID = "google", "google-interactions", googleinteractions.Model
	p.Profile.OfficialTariff.Model = p.Profile.ModelID
	p.Profile.AdapterPolicyVersion, p.Profile.UsageMappingVersion = googleinteractions.AdapterPolicyVersion, googleinteractions.UsageMappingVersion
	p.Profile.MaximumInputBytes, p.Profile.MaximumOutputBytes = 128<<10, 256<<10
	p.AdmittedEndpointIdentityDigest = governed.EndpointIdentityDigest(googleinteractions.Origin)
	p.Profile.EndpointIdentityDigest = p.AdmittedEndpointIdentityDigest
	c.ProductAgent.TextPolicies["org"] = p
	c.ModelRoutePricing.Bindings[0].Text = &p
	in.BaseURL = googleinteractions.Origin
	require.NoError(t, ValidateChannelCredentialProvision(c, in))
	for _, origin := range []string{"https://other.example.test", "http://127.0.0.1:8080"} {
		bad := in
		bad.BaseURL = origin
		p.Profile.EndpointIdentityDigest = governed.EndpointIdentityDigest(origin)
		require.Error(t, ValidateChannelCredentialProvision(c, bad))
	}
	p.AdmittedEndpointIdentityDigest = governed.EndpointIdentityDigest(googleinteractions.Origin)
	p.Profile.EndpointIdentityDigest = p.AdmittedEndpointIdentityDigest
	p.Profile.AdapterPolicyVersion = "unqualified"
	c.ProductAgent.TextPolicies["org"] = p
	require.Error(t, ValidateChannelCredentialProvision(c, in))
}
