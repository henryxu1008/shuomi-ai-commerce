package agentconfig

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCandidateRulesUsesNewExactMaterialsDefinition(t *testing.T) {
	require.Equal(t, "v4.0.0", MaterialsAgentVersion)
	require.Equal(t, "listing-materials-policy-v4", MaterialsPolicyVersion)
	require.Equal(t, "listing-materials-prompt-v4", MaterialsPromptVersion)
	require.Equal(t, "listing-materials-action-v2", MaterialsActionVersion)
	require.Equal(t, "listing-materials-v2", MaterialsProposalKind)
	require.True(t, MaterialsStoredVersion("v3.0.0", "listing-materials-policy-v3", "listing-materials-prompt-v3"))
	require.False(t, MaterialsStoredVersion("v3.0.0", MaterialsPolicyVersion, MaterialsPromptVersion))
}
