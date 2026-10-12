package agent

import (
	"strings"
	"testing"

	"task-processor/internal/commercetool"
)

func TestExecutionModePolicyBoundaries(t *testing.T) {
	read := ExecutionAction{Ref: commercetool.ToolRef{ID: "product.inspect", Version: "v1.0.0"}, Class: ActionObservation, Risk: commercetool.RiskRead, SideEffect: commercetool.SideEffectNone}
	proposal := read
	proposal.Ref.ID, proposal.Class, proposal.Risk = "product.propose", ActionProposal, commercetool.RiskPropose
	title := ExecutionAction{Ref: commercetool.ToolRef{ID: "product.title.adopt", Version: "v1.0.0"}, Class: ActionLocalWrite, Risk: commercetool.RiskWrite, SideEffect: commercetool.SideEffectBusinessMutation, AutomaticApproval: true}
	image := title
	image.Ref.ID, image.AutomaticApproval = "product.image.adopt", false
	generation := ExecutionAction{Ref: commercetool.ToolRef{ID: "product.image.generate", Version: "v1.0.0"}, Class: ActionImageGeneration, Risk: commercetool.RiskCompute, SideEffect: commercetool.SideEffectBusinessMutation, RequiresEconomicIntent: true}
	protected := title
	protected.Ref.ID, protected.Class, protected.Risk, protected.SideEffect = "listing.publish", ActionProtected, commercetool.RiskPublish, commercetool.SideEffectExternalMutation
	protected.AutomaticApproval = false
	checks := ExecutionPolicyChecks{OwnerAuthorized: true, ContentValid: true, EconomicIntentConfirmed: true, Automatic: RulePassed}
	for _, mode := range []ExecutionMode{ModeDefault, ModeAutoApprove, ModeReadOnly, ModeFullAccess} {
		for _, action := range []ExecutionAction{read, proposal} {
			if got := EvaluateExecutionPolicy(mode, action, checks); got.Outcome != PolicyAllow {
				t.Errorf("%s %s: %+v", mode, action.Ref.ID, got)
			}
		}
	}
	tests := []struct {
		name   string
		mode   ExecutionMode
		action ExecutionAction
		checks ExecutionPolicyChecks
		want   PolicyOutcome
	}{
		{"default title requires genuine human decision", ModeDefault, title, checks, PolicyHumanRequired},
		{"automatic title rule passed", ModeAutoApprove, title, checks, PolicyAllow},
		{"readonly title", ModeReadOnly, title, checks, PolicyDeny},
		{"full title", ModeFullAccess, title, checks, PolicyAllow},
		{"automatic image cannot prove fidelity", ModeAutoApprove, image, checks, PolicyHumanRequired},
		{"full image preauthorized", ModeFullAccess, image, checks, PolicyAllow},
		{"readonly generation", ModeReadOnly, generation, checks, PolicyDeny},
		{"confirmed generation", ModeDefault, generation, checks, PolicyAllow},
		{"invalid mode", ExecutionMode("admin"), title, checks, PolicyDeny},
		{"missing descriptor", ModeFullAccess, ExecutionAction{}, checks, PolicyDeny},
	}
	for _, mode := range []ExecutionMode{ModeDefault, ModeAutoApprove, ModeFullAccess} {
		tests = append(tests, struct {
			name   string
			mode   ExecutionMode
			action ExecutionAction
			checks ExecutionPolicyChecks
			want   PolicyOutcome
		}{"protected " + string(mode), mode, protected, checks, PolicySeparateAuthorization})
	}
	unauthorized := checks
	unauthorized.OwnerAuthorized = false
	tests = append(tests, struct {
		name   string
		mode   ExecutionMode
		action ExecutionAction
		checks ExecutionPolicyChecks
		want   PolicyOutcome
	}{"full grants no new owner permission", ModeFullAccess, title, unauthorized, PolicyDeny})
	invalid := checks
	invalid.ContentValid = false
	tests = append(tests, struct {
		name   string
		mode   ExecutionMode
		action ExecutionAction
		checks ExecutionPolicyChecks
		want   PolicyOutcome
	}{"full keeps owner validation", ModeFullAccess, title, invalid, PolicyDeny})
	noQuote := checks
	noQuote.EconomicIntentConfirmed = false
	for _, mode := range []ExecutionMode{ModeDefault, ModeAutoApprove, ModeFullAccess} {
		tests = append(tests, struct {
			name   string
			mode   ExecutionMode
			action ExecutionAction
			checks ExecutionPolicyChecks
			want   PolicyOutcome
		}{"quote remains required " + string(mode), mode, generation, noQuote, PolicyHumanRequired})
	}
	for _, rule := range []AutomaticRuleOutcome{RuleNeedsHuman, RuleNotEvaluated} {
		c := checks
		c.Automatic = rule
		tests = append(tests, struct {
			name   string
			mode   ExecutionMode
			action ExecutionAction
			checks ExecutionPolicyChecks
			want   PolicyOutcome
		}{"auto unknown does not approve " + string(rule), ModeAutoApprove, title, c, PolicyHumanRequired})
	}
	for _, rule := range []AutomaticRuleOutcome{RuleRejected, RuleUnavailable} {
		c := checks
		c.Automatic = rule
		for _, mode := range []ExecutionMode{ModeAutoApprove, ModeFullAccess} {
			tests = append(tests, struct {
				name   string
				mode   ExecutionMode
				action ExecutionAction
				checks ExecutionPolicyChecks
				want   PolicyOutcome
			}{"rule fails closed " + string(rule) + string(mode), mode, title, c, PolicyDeny})
		}
	}
	human := checks
	human.HumanContentConfirmed = true
	tests = append(tests,
		struct {
			name   string
			mode   ExecutionMode
			action ExecutionAction
			checks ExecutionPolicyChecks
			want   PolicyOutcome
		}{"default human decision", ModeDefault, title, human, PolicyAllow},
		struct {
			name   string
			mode   ExecutionMode
			action ExecutionAction
			checks ExecutionPolicyChecks
			want   PolicyOutcome
		}{"readonly has no human write exception", ModeReadOnly, title, human, PolicyDeny},
		struct {
			name   string
			mode   ExecutionMode
			action ExecutionAction
			checks ExecutionPolicyChecks
			want   PolicyOutcome
		}{"readonly protected", ModeReadOnly, protected, human, PolicyDeny},
	)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := EvaluateExecutionPolicy(tt.mode, tt.action, tt.checks)
			if got.Outcome != tt.want || got.Reason == "" {
				t.Fatalf("got %+v, want %s", got, tt.want)
			}
		})
	}
}

func TestExecutionModeRejectsMisclassifiedEffects(t *testing.T) {
	checks := ExecutionPolicyChecks{OwnerAuthorized: true, ContentValid: true, EconomicIntentConfirmed: true, Automatic: RulePassed}
	for _, action := range []ExecutionAction{
		{Ref: commercetool.ToolRef{ID: "listing.publish", Version: "v1.0.0"}, Class: ActionObservation, Risk: commercetool.RiskPublish, SideEffect: commercetool.SideEffectExternalMutation},
		{Ref: commercetool.ToolRef{ID: "product.image.generate", Version: "v1.0.0"}, Class: ActionImageGeneration, Risk: commercetool.RiskCompute, SideEffect: commercetool.SideEffectBusinessMutation},
	} {
		if got := EvaluateExecutionPolicy(ModeFullAccess, action, checks); got.Outcome != PolicyDeny {
			t.Fatalf("misclassified effect admitted: %+v", got)
		}
	}
}

func TestModeSnapshotRefCanonicalIdentity(t *testing.T) {
	ref := ModeSnapshotRef{Kind: ExecutionModeSnapshotKind, ID: "d545b0de-10f5-43d4-b2a2-885a77d93463", Digest: strings.Repeat("ab", 32)}
	if !ref.Valid() || ref.Absent() {
		t.Fatal("canonical ref rejected")
	}
	for _, bad := range []ModeSnapshotRef{
		{}, {Kind: "agent-configuration-v1", ID: ref.ID, Digest: ref.Digest},
		{Kind: ref.Kind, ID: strings.ToUpper(ref.ID), Digest: ref.Digest},
		{Kind: ref.Kind, ID: ref.ID, Digest: strings.ToUpper(ref.Digest)},
		{Kind: ref.Kind, ID: ref.ID, Digest: strings.Repeat("a", 63)},
	} {
		if bad.Valid() {
			t.Fatalf("invalid ref admitted: %+v", bad)
		}
	}
}
