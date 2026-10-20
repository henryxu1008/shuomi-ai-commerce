package agentconfig

import "task-processor/internal/commercetool"

const (
	DiagnosisAgentID         = "shop.operations-diagnosis.agent"
	DiagnosisAgentVersion    = "v1.0.0"
	DiagnosisProposalKind    = "shop-operations-diagnosis-v1"
	DiagnosisPolicyVersion   = "shop-operations-diagnosis-policy-v1"
	DiagnosisPromptVersion   = "shop-operations-diagnosis-prompt-v1"
	DiagnosisActionVersion   = "shop-operations-diagnosis-action-v1"
	DiagnosisParameterSchema = "shop-operations-diagnosis-config-v1"
)

func DiagnosisInspectToolRef() commercetool.ToolRef {
	return commercetool.ToolRef{ID: "shop.operations-diagnosis.inspect", Version: "v1.0.0"}
}
