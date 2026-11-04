package einomodel

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"task-processor/internal/aicapability"
	credential "task-processor/internal/aicapability/channelcredential"
)

type channelReaderFixture struct {
	secret      credential.Secret
	unavailable bool
	reads       int
}

func testChannelReader(org, id, endpoint string) *channelReaderFixture {
	m := (credential.Metadata{OrganizationID: org, ConnectionID: id, Provider: "fixture-provider", Channel: "fixture-channel", Account: "fixture-account", Region: "fixture-region", BaseURL: endpoint, Revision: 1, Enabled: true, KeyPresent: true}).WithIdentity()
	return &channelReaderFixture{secret: credential.Secret{Metadata: m, APIKey: "synthetic-organization-key"}}
}

func (r *channelReaderFixture) Resolve(_ context.Context, org, id, ref string) (credential.Secret, error) {
	r.reads++
	if r.unavailable {
		return credential.Secret{}, credential.ErrUnavailable
	}
	if org != r.secret.Metadata.OrganizationID || id != r.secret.Metadata.ConnectionID {
		return credential.Secret{}, credential.ErrNotFound
	}
	if ref != r.secret.Metadata.Reference || !r.secret.Metadata.Enabled {
		return credential.Secret{}, credential.ErrChanged
	}
	return r.secret, nil
}

func fixtureChannelPolicy(p aicapability.ModelProfile, r *channelReaderFixture) RoutePolicy {
	p.ConnectionID, p.CredentialVersion = r.secret.Metadata.ConnectionID, r.secret.Metadata.Reference
	return RoutePolicy{Profile: p, AdmittedCredentialVersion: p.CredentialVersion, AdmittedEndpointIdentityDigest: endpointDigest(r.secret.Metadata.BaseURL)}
}

func TestSharedChannelKeepsModelAndOrganizationExecutionIdentity(t *testing.T) {
	reader := testChannelReader("org", "one-channel", "https://example.org/v1")
	p := testTextProfile(reader.secret.Metadata.BaseURL)
	policy := fixtureChannelPolicy(p, reader)
	second := policy
	second.Profile.ModelID = "another-admitted-model"
	second.Profile.ClientName = "second-binding"
	second.Profile.OfficialTariff.Model = second.Profile.ModelID
	r, err := NewOrganizationRouteResolver(reader, map[RouteKey]RoutePolicy{{"org", aicapability.OperationProductAgentDecision}: policy, {"org", aicapability.OperationProductContentDecision}: second})
	require.NoError(t, err)
	first, err := r.Resolve(context.Background(), aicapability.TextInputIdentity{OrganizationID: "org", ActorID: "member-a", Operation: aicapability.OperationProductAgentDecision})
	require.NoError(t, err)
	next, err := r.Resolve(context.Background(), aicapability.TextInputIdentity{OrganizationID: "org", ActorID: "member-b", Operation: aicapability.OperationProductContentDecision})
	require.NoError(t, err)
	require.Equal(t, first.APIKey, next.APIKey)
	require.Equal(t, first.Profile.CredentialVersion, next.Profile.CredentialVersion)
	require.NotEqual(t, first.Profile.ModelID, next.Profile.ModelID)
	a, _ := first.Profile.Digest()
	b, _ := next.Profile.Digest()
	require.NotEqual(t, a, b, "a shared key cannot share a model execution/cache identity")
	_, err = r.Resolve(context.Background(), aicapability.TextInputIdentity{OrganizationID: "other-org", Operation: aicapability.OperationProductAgentDecision})
	require.ErrorIs(t, err, ErrNotDispatched)
	reader.secret.Metadata.Enabled = false
	require.Equal(t, RouteNeedsConfiguration, r.Readiness(context.Background(), aicapability.TextInputIdentity{OrganizationID: "org", Operation: aicapability.OperationProductAgentDecision}))
}

func TestListingMaterialsRouteDoesNotBorrowContentOrTitlePolicy(t *testing.T) {
	rd := testChannelReader("org", "channel", "https://example.org/v1")
	r, err := NewOrganizationRouteResolver(rd, map[RouteKey]RoutePolicy{{"org", aicapability.OperationListingMaterialsDecision}: fixtureChannelPolicy(testTextProfile(rd.secret.Metadata.BaseURL), rd)})
	require.NoError(t, err)
	for _, op := range []aicapability.Operation{aicapability.OperationProductAgentDecision, aicapability.OperationProductContentDecision} {
		require.Equal(t, RouteUnavailable, r.Readiness(context.Background(), aicapability.TextInputIdentity{OrganizationID: "org", Operation: op}))
	}
}

func TestMissingChannelSecretReaderKeepsConstructionAndRejectsExecution(t *testing.T) {
	r, err := NewManagedOrganizationRouteResolver(nil, &changingPolicies{missing: true})
	require.NoError(t, err, "metadata/history construction must not require an executable secret reader")
	_, err = r.Resolve(context.Background(), aicapability.TextInputIdentity{OrganizationID: "org", Operation: aicapability.OperationProductAgentDecision})
	require.ErrorIs(t, err, ErrNotDispatched)
}
