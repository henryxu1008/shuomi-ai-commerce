package agentconfig

import "task-processor/internal/commercetool"

// Fixed code-owned registration contract shared by configuration and the
// governed text adapter. Platform facts/DTO validation remain in Listing.
const (
	MaterialsAgentID         = "listing.materials.agent"
	MaterialsAgentVersion    = "v3.0.0"
	MaterialsProposalKind    = "listing-materials-v1"
	MaterialsPolicyVersion   = "listing-materials-policy-v3"
	MaterialsPromptVersion   = "listing-materials-prompt-v3"
	MaterialsActionVersion   = "listing-materials-action-v1"
	MaterialsParameterSchema = "listing-materials-config-v1"
)

func MaterialsInspectToolRef() commercetool.ToolRef {
	return commercetool.ToolRef{ID: "listing.materials.inspect", Version: "v3.0.0"}
}

// D1 permits only these exact stored formats to be read. This is never a new
// execution admission; current Prepare/Claim/resume/adopt require v3.
func MaterialsStoredVersion(version, policy, prompt string) bool {
	return version == MaterialsAgentVersion && policy == MaterialsPolicyVersion && prompt == MaterialsPromptVersion ||
		version == "v2.0.0" && policy == "listing-materials-policy-v2" && prompt == "listing-materials-prompt-v2" ||
		version == "v1.0.0" && policy == "listing-materials-policy-v1" && prompt == "listing-materials-prompt-v1"
}

func MaterialsStoredRequest(policy, prompt string) bool {
	return MaterialsStoredVersion(MaterialsAgentVersion, policy, prompt) ||
		MaterialsStoredVersion("v2.0.0", policy, prompt) ||
		MaterialsStoredVersion("v1.0.0", policy, prompt)
}
