package agentconfig

import reviewproposal "task-processor/internal/product/review/proposal"

const (
	ContentAgentID          = "product.content.agent"
	ContentAgentVersion     = "v1.0.0"
	ContentParameterSchema  = "content-config-v1"
	ContentPromptVersion    = "product-content-agent-v1"
	ContentActionVersion    = "product-content-action-v1"
	MaxContentTemplateBytes = 4096
)

// Template preferences are data; the execution confirmation freezes the actual
// selection separately in Request and never trusts a later template head.
type ContentTemplate struct {
	Schema    string                          `json:"schema"`
	Selection reviewproposal.ContentSelection `json:"selection"`
}

func (t ContentTemplate) Valid() bool {
	return t.Schema == ContentParameterSchema && t.Selection.Valid()
}
