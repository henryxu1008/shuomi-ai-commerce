package agentconfig

import "task-processor/internal/commercetool"

const (
	SelectionAgentID         = "product.selection.agent"
	SelectionAgentVersion    = "v1.0.0"
	SelectionProposalKind    = "product-selection-v1"
	SelectionPolicyVersion   = "product-selection-policy-v1"
	SelectionPromptVersion   = "product-selection-prompt-v1"
	SelectionActionVersion   = "product-selection-action-v1"
	SelectionParameterSchema = "product-selection-config-v1"
)

func SelectionInspectToolRef() commercetool.ToolRef {
	return commercetool.ToolRef{ID: "product.selection.inspect", Version: "v1.0.0"}
}
