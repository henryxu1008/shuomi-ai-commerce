package agentconfig

const (
	TitleAgentID                = "product.title.agent"
	TitleAgentVersion           = "v2.0.0"
	TitlePolicyVersion          = "title-review-v1"
	TitlePromptVersion          = "product-title-agent-v2"
	TitleKnowledgePromptVersion = "product-title-agent-knowledge-v2"
)

func CurrentTitleRequest(policy, prompt string) bool {
	return policy == TitlePolicyVersion && (prompt == TitlePromptVersion || prompt == TitleKnowledgePromptVersion)
}
func StoredTitleVersion(version, policy, prompt string) bool {
	return version == TitleAgentVersion && CurrentTitleRequest(policy, prompt) ||
		version == "v1.0.0" && policy == TitlePolicyVersion && (prompt == "product-title-agent-v1" || prompt == "product-title-agent-knowledge-v1")
}
