// Package materials owns local platform-material suggestions and validation.
// Catalog content, configuration, bounded runs and Target facts retain their
// existing owners. It provides no platform publication or provider client.
package materials

import (
	"task-processor/internal/agent"
	"task-processor/internal/agentconfig"
	"task-processor/internal/commercetool"
)

const (
	AgentID         = agentconfig.MaterialsAgentID
	AgentVersion    = agentconfig.MaterialsAgentVersion
	ProposalKind    = agentconfig.MaterialsProposalKind
	PolicyVersion   = agentconfig.MaterialsPolicyVersion
	PromptVersion   = agentconfig.MaterialsPromptVersion
	ActionVersion   = agentconfig.MaterialsActionVersion
	ParameterSchema = agentconfig.MaterialsParameterSchema
)

// EvaluateAdoption receives only checks resolved by the local domain owner.
// It cannot approve publication or bypass the account's business permissions.
func EvaluateAdoption(mode agent.ExecutionMode, checks agent.ExecutionPolicyChecks, requiresHuman bool) agent.ExecutionPolicyDecision {
	decision := agent.EvaluateExecutionPolicy(mode, agent.ExecutionAction{
		Ref:   commercetool.ToolRef{ID: "listing.materials.adopt", Version: AgentVersion},
		Class: agent.ActionLocalWrite, Risk: commercetool.RiskWrite,
		SideEffect: commercetool.SideEffectBusinessMutation, AutomaticApproval: true,
	}, checks)
	if decision.Outcome == agent.PolicyAllow && !checks.HumanContentConfirmed &&
		(requiresHuman || checks.Automatic == agent.RuleNeedsHuman) {
		return agent.ExecutionPolicyDecision{Outcome: agent.PolicyHumanRequired, Reason: "LISTING_DOMAIN_CONFIRMATION_REQUIRED"}
	}
	return decision
}
