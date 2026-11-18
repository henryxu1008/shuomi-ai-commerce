package aicapability

// SkillAuthoringOpenAIParameterRejectedV1 marks an exact pre-inference
// validation rejection observed by the governed OpenAI transport.
const SkillAuthoringOpenAIParameterRejectedV1 = "skill_authoring_openai_parameter_rejected_v1"
const ChatIntentOpenAIParameterRejectedV1 = "chat_intent_openai_parameter_rejected_v1"

// IsSkillAuthoringOpenAIParameterRejection checks the stored zero-charge fact.
// Usage remains unobserved. Only the governed transport establishes the fact;
// a recovery caller cannot supply this code to resolve an old dispatch.
func (fact InvocationRecord) IsSkillAuthoringOpenAIParameterRejection() bool {
	return fact.ErrorCode == SkillAuthoringOpenAIParameterRejectedV1 && fact.Operation == OperationAIWorkbenchSkillAuthoring && fact.isWorkbenchOpenAIParameterRejection()
}

func (fact InvocationRecord) IsChatIntentOpenAIParameterRejection() bool {
	return fact.ErrorCode == ChatIntentOpenAIParameterRejectedV1 && fact.Operation == OperationAIWorkbenchChatIntent && fact.isWorkbenchOpenAIParameterRejection()
}

// Each proof is bound to its own purpose. Neither an old unknown invocation nor
// another operation may claim this transport-only zero-charge terminal.
func (fact InvocationRecord) IsWorkbenchOpenAIParameterRejection() bool {
	return fact.IsSkillAuthoringOpenAIParameterRejection() || fact.IsChatIntentOpenAIParameterRejection()
}

func (fact InvocationRecord) isWorkbenchOpenAIParameterRejection() bool {
	return fact.Outcome == InvocationFailed && fact.ErrorCategory == ErrorProviderRejected &&
		fact.Capability == CapabilityAIWorkbenchChatPlanning &&
		fact.AgentRunID == "" && fact.BusinessTaskID == "" && fact.ProviderID == "openai" && fact.ModelID == SingleImageMiniModel &&
		fact.TenantID != "" && fact.UserID != "" && fact.MemberID != "" && fact.InvocationID != "" && fact.InputHash != "" &&
		fact.OfficialTariff.Valid() && fact.OfficialTariff.Channel == "openai-direct" &&
		fact.OfficialTariff.MatchesWireModel(fact.ModelID) && fact.OfficialTariff.UsageVersion == "openai-text-v1" &&
		!fact.UsageKnown && fact.BillableUsage == nil && fact.ProviderRejectionProof == nil &&
		fact.PromptTokens == 0 && fact.CompletionTokens == 0 && fact.TotalTokens == 0 && fact.ImageCount == 0 &&
		fact.EstimatedCostKnown && fact.EstimatedCostMicros == 0 && fact.OutputHash == "" &&
		!fact.StartedAt.IsZero() && !fact.FinishedAt.Before(fact.StartedAt) && validRejectionHash(fact.ProviderRequestID)
}
