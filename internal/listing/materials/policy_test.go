package materials

import (
	"task-processor/internal/agent"
	"testing"
)

func TestListingAdoptionModesRetainDomainHumanConfirmation(t *testing.T) {
	for _, scenario := range []struct {
		name                     string
		mode                     agent.ExecutionMode
		rule                     agent.AutomaticRuleOutcome
		requiresHuman, confirmed bool
		want                     agent.PolicyOutcome
	}{
		{"default review", agent.ModeDefault, agent.RulePassed, false, false, agent.PolicyHumanRequired},
		{"default confirmed", agent.ModeDefault, agent.RuleNeedsHuman, true, true, agent.PolicyAllow},
		{"readonly even confirmed", agent.ModeReadOnly, agent.RulePassed, false, true, agent.PolicyDeny},
		{"auto qualified", agent.ModeAutoApprove, agent.RulePassed, false, false, agent.PolicyAllow},
		{"auto changed category", agent.ModeAutoApprove, agent.RulePassed, true, false, agent.PolicyHumanRequired},
		{"full qualified", agent.ModeFullAccess, agent.RulePassed, false, false, agent.PolicyAllow},
		{"full changed category", agent.ModeFullAccess, agent.RulePassed, true, false, agent.PolicyHumanRequired},
		{"full actual unresolved rule", agent.ModeFullAccess, agent.RuleNeedsHuman, false, false, agent.PolicyHumanRequired},
		{"full confirmed category", agent.ModeFullAccess, agent.RuleNeedsHuman, true, true, agent.PolicyAllow},
		{"full missing dependency", agent.ModeFullAccess, agent.RuleUnavailable, true, true, agent.PolicyDeny},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			decision := EvaluateAdoption(scenario.mode, agent.ExecutionPolicyChecks{OwnerAuthorized: true, ContentValid: true, Automatic: scenario.rule, HumanContentConfirmed: scenario.confirmed}, scenario.requiresHuman)
			if decision.Outcome != scenario.want {
				t.Fatalf("mode=%s rule=%s human=%v confirmation=%v: outcome=%s want=%s", scenario.mode, scenario.rule, scenario.requiresHuman, scenario.confirmed, decision.Outcome, scenario.want)
			}
		})
	}
}

func TestListingAdoptionCannotReplaceOwnerAuthorization(t *testing.T) {
	decision := EvaluateAdoption(agent.ModeFullAccess, agent.ExecutionPolicyChecks{ContentValid: true, Automatic: agent.RulePassed, HumanContentConfirmed: true}, false)
	if decision.Outcome != agent.PolicyDeny {
		t.Fatal("full access cannot replace the current owner's authorization")
	}
}
