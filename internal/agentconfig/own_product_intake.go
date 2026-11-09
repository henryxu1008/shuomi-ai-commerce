package agentconfig

import (
	"task-processor/internal/agent"
	"task-processor/internal/aicapability"
	"task-processor/internal/commercetool"
)

const (
	OwnIntakeAgentID       = "product.own.intake.agent"
	OwnIntakeAgentVersion  = "v1.2.0"
	OwnIntakePolicyVersion = "own-product-intake-policy-v1"
	OwnIntakePromptVersion = "own-product-intake-prompt-v3"
	OwnIntakeActionVersion = "own-product-intake-candidate-v3"
)

func OwnIntakeDefinition() commercetool.AgentDefinition {
	return commercetool.AgentDefinition{ID: OwnIntakeAgentID, Version: OwnIntakeAgentVersion}
}

func OwnIntakeSubject() ResolvedModeSubject {
	return ResolvedModeSubject{Subject: ModeSubject{Kind: RegisteredAgentSubject, ID: OwnIntakeAgentID}, Definition: commercetool.AgentRef{ID: OwnIntakeAgentID, Version: OwnIntakeAgentVersion}}
}

func OwnIntakeRequest(r agent.Request) bool {
	return r.Binding.ContextKind == agent.OwnProductIntakeContextKind && r.Binding.Valid() && r.Binding.ContextID == r.Key &&
		r.PolicyVersion == OwnIntakePolicyVersion && r.PromptVersion == OwnIntakePromptVersion && r.Limits.ModelCalls == 1 &&
		r.ContentSelection.Absent() && r.SkillVersionRef.Absent() && r.WorkbenchContextRef.Absent() && !r.ManualReviewOnly && r.GoalSummary == ""
}

func OwnIntakeProfile(p aicapability.ModelProfile) bool {
	return OwnIntakeProfileShape(p) && aicapability.SingleImageProfile(p)
}

func OwnIntakeProfileShape(p aicapability.ModelProfile) bool {
	return p.PromptVersion == OwnIntakePromptVersion && p.OutputSchemaVersion == OwnIntakeActionVersion && aicapability.SingleImageProfileShape(p)
}
