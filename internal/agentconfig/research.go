package agentconfig

import "task-processor/internal/commercetool"

const (
	ResearchAgentID         = "product.research.agent"
	ResearchAgentVersion    = "v1.0.0"
	ResearchProposalKind    = "market-research-selection-v1"
	ResearchPolicyVersion   = "market-research-policy-v1"
	ResearchPromptVersion   = "market-research-prompt-v1"
	ResearchActionVersion   = "market-research-action-v1"
	ResearchParameterSchema = "market-research-config-v1"
)

func ResearchInspectToolRef() commercetool.ToolRef {
	return commercetool.ToolRef{ID: "product.research.inspect", Version: "v1.0.0"}
}
