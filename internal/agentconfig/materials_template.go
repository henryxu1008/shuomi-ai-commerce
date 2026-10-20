package agentconfig

import "task-processor/internal/commercetool"

// Fixed code-owned registration contract shared by configuration and the
// governed text adapter. Platform facts/DTO validation remain in Listing.
const (
	MaterialsAgentID         = "listing.materials.agent"
	MaterialsAgentVersion    = "v1.0.0"
	MaterialsProposalKind    = "listing-materials-v1"
	MaterialsPolicyVersion   = "listing-materials-policy-v1"
	MaterialsPromptVersion   = "listing-materials-prompt-v1"
	MaterialsActionVersion   = "listing-materials-action-v1"
	MaterialsParameterSchema = "listing-materials-config-v1"
)

func MaterialsInspectToolRef() commercetool.ToolRef {
	return commercetool.ToolRef{ID: "listing.materials.inspect", Version: "v1.0.0"}
}
