package agentconfig

import "task-processor/internal/commercetool"

// Fixed code-owned registration contract shared by configuration and the
// governed text adapter. Platform facts/DTO validation remain in Listing.
const (
	MaterialsAgentID         = "listing.materials.agent"
	MaterialsAgentVersion    = "v4.0.0"
	MaterialsProposalKind    = "listing-materials-v2"
	MaterialsPolicyVersion   = "listing-materials-policy-v4"
	MaterialsPromptVersion   = "listing-materials-prompt-v4"
	MaterialsActionVersion   = "listing-materials-action-v2"
	MaterialsParameterSchema = "listing-materials-config-v1"
)

func MaterialsInspectToolRef() commercetool.ToolRef {
	return commercetool.ToolRef{ID: "listing.materials.inspect", Version: "v3.0.0"}
}

// D1 permits only these exact stored formats to be read. This is never a new
// execution admission; current Prepare/Claim/resume/adopt require v4.
func MaterialsStoredVersion(version, policy, prompt string) bool {
	return version == MaterialsAgentVersion && policy == MaterialsPolicyVersion && prompt == MaterialsPromptVersion ||
		version == "v3.0.0" && policy == "listing-materials-policy-v3" && prompt == "listing-materials-prompt-v3" ||
		version == "v2.0.0" && policy == "listing-materials-policy-v2" && prompt == "listing-materials-prompt-v2" ||
		version == "v1.0.0" && policy == "listing-materials-policy-v1" && prompt == "listing-materials-prompt-v1"
}

func MaterialsStoredRequest(policy, prompt string) bool {
	return MaterialsStoredVersion(MaterialsAgentVersion, policy, prompt) ||
		MaterialsStoredVersion("v3.0.0", policy, prompt) ||
		MaterialsStoredVersion("v2.0.0", policy, prompt) ||
		MaterialsStoredVersion("v1.0.0", policy, prompt)
}

// Stored formats are selected by their exact original policy/prompt tuple.
// These helpers never authorize execution or reinterpret an old payload as v2.
func MaterialsStoredActionVersion(policy, prompt string) string {
	if policy == MaterialsPolicyVersion && prompt == MaterialsPromptVersion {
		return MaterialsActionVersion
	}
	for _, v := range []string{"v1", "v2", "v3"} {
		if policy == "listing-materials-policy-"+v && prompt == "listing-materials-prompt-"+v {
			return "listing-materials-action-v1"
		}
	}
	return ""
}

func MaterialsStoredProposalKind(policy, prompt string) string {
	switch MaterialsStoredActionVersion(policy, prompt) {
	case "listing-materials-action-v1":
		return "listing-materials-v1"
	case MaterialsActionVersion:
		return MaterialsProposalKind
	}
	return ""
}

func MaterialsRulesToolRef() commercetool.ToolRef {
	return commercetool.ToolRef{ID: "listing.materials.rules.query", Version: "v1.0.0"}
}

func MaterialsAllowedTools() []commercetool.ToolRef {
	return []commercetool.ToolRef{MaterialsInspectToolRef(), MaterialsRulesToolRef()}
}
