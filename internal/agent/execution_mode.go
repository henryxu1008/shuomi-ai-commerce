package agent

import (
	"encoding/hex"

	"github.com/google/uuid"
	"task-processor/internal/commercetool"
)

type ExecutionMode string

const (
	ModeDefault                ExecutionMode = "default"
	ModeAutoApprove            ExecutionMode = "auto_approve"
	ModeReadOnly               ExecutionMode = "read_only"
	ModeFullAccess             ExecutionMode = "full_access"
	ExecutionModeSnapshotKind                = "agent-execution-mode-v1"
	ExecutionModePolicyVersion               = "agent-execution-policy-v1"
)

func (m ExecutionMode) Valid() bool {
	return m == ModeDefault || m == ModeAutoApprove || m == ModeReadOnly || m == ModeFullAccess
}

// ModeSnapshotRef carries an immutable, owner-resolved execution intent. Its
// validity proves only canonical metadata; it grants no identity or permission.
type ModeSnapshotRef struct{ Kind, ID, Digest string }

func (r ModeSnapshotRef) Absent() bool { return r == ModeSnapshotRef{} }
func (r ModeSnapshotRef) Valid() bool {
	id, err := uuid.Parse(r.ID)
	d, decodeErr := hex.DecodeString(r.Digest)
	return r.Kind == ExecutionModeSnapshotKind && err == nil && id != uuid.Nil && id.String() == r.ID && decodeErr == nil && len(d) == 32 && hex.EncodeToString(d) == r.Digest
}

type ExecutionActionClass string

const (
	ActionObservation     ExecutionActionClass = "observation"
	ActionProposal        ExecutionActionClass = "proposal"
	ActionLocalWrite      ExecutionActionClass = "local_write"
	ActionImageGeneration ExecutionActionClass = "image_generation"
	ActionProtected       ExecutionActionClass = "protected"
)

// ExecutionAction is code-owned adapter metadata, never accepted from a model
// or HTTP payload. In particular, read/propose metadata cannot hide a publish.
type ExecutionAction struct {
	Ref                                       commercetool.ToolRef
	Class                                     ExecutionActionClass
	Risk                                      commercetool.RiskLevel
	SideEffect                                commercetool.SideEffectMode
	AutomaticApproval, RequiresEconomicIntent bool
}

func (a ExecutionAction) Valid() bool {
	if a.Ref.Validate() != nil {
		return false
	}
	switch a.Class {
	case ActionObservation:
		return (a.Risk == commercetool.RiskRead || a.Risk == commercetool.RiskCompute) && a.SideEffect == commercetool.SideEffectNone && !a.AutomaticApproval
	case ActionProposal:
		return a.Risk == commercetool.RiskPropose && a.SideEffect == commercetool.SideEffectNone && !a.AutomaticApproval
	case ActionLocalWrite:
		return a.Risk == commercetool.RiskWrite && a.SideEffect == commercetool.SideEffectBusinessMutation
	case ActionImageGeneration:
		return (a.Risk == commercetool.RiskCompute || a.Risk == commercetool.RiskWrite) && a.SideEffect == commercetool.SideEffectBusinessMutation && a.RequiresEconomicIntent && !a.AutomaticApproval
	case ActionProtected:
		return (a.Risk == commercetool.RiskWrite || a.Risk == commercetool.RiskPublish) && (a.SideEffect == commercetool.SideEffectBusinessMutation || a.SideEffect == commercetool.SideEffectExternalMutation) && !a.AutomaticApproval
	default:
		return false
	}
}

type AutomaticRuleOutcome string

const (
	RuleNotEvaluated AutomaticRuleOutcome = ""
	RulePassed       AutomaticRuleOutcome = "passed"
	RuleNeedsHuman   AutomaticRuleOutcome = "needs_human"
	RuleRejected     AutomaticRuleOutcome = "rejected"
	RuleUnavailable  AutomaticRuleOutcome = "unavailable"
)

// These are results of the real owner's fresh checks, including exact content,
// source/version, grants and quote. They are not a browser approval interface.
type ExecutionPolicyChecks struct {
	OwnerAuthorized, ContentValid                  bool
	EconomicIntentConfirmed, HumanContentConfirmed bool
	Automatic                                      AutomaticRuleOutcome
}

type PolicyOutcome string

const (
	PolicyAllow                 PolicyOutcome = "allow"
	PolicyHumanRequired         PolicyOutcome = "human_required"
	PolicySeparateAuthorization PolicyOutcome = "separate_authorization_required"
	PolicyDeny                  PolicyOutcome = "deny"
)

type ExecutionPolicyDecision struct {
	Outcome PolicyOutcome
	Reason  string
}

// EvaluateExecutionPolicy is pure classification, not an adoption proof. The
// application must resolve the frozen snapshot, current revision and identity;
// the domain owner still owns admission, commit, receipts and UNKNOWN recovery.
func EvaluateExecutionPolicy(mode ExecutionMode, action ExecutionAction, checks ExecutionPolicyChecks) ExecutionPolicyDecision {
	decision := func(outcome PolicyOutcome, reason string) ExecutionPolicyDecision {
		return ExecutionPolicyDecision{Outcome: outcome, Reason: reason}
	}
	if !mode.Valid() || !action.Valid() {
		return decision(PolicyDeny, "INVALID_EXECUTION_POLICY")
	}
	if !checks.OwnerAuthorized || !checks.ContentValid {
		return decision(PolicyDeny, "OWNER_VALIDATION_FAILED")
	}
	if checks.Automatic == RuleRejected || checks.Automatic == RuleUnavailable {
		return decision(PolicyDeny, "AUTOMATIC_RULE_"+string(checks.Automatic))
	}
	if checks.Automatic != RuleNotEvaluated && checks.Automatic != RulePassed && checks.Automatic != RuleNeedsHuman {
		return decision(PolicyDeny, "INVALID_AUTOMATIC_RULE_RESULT")
	}
	if action.Class == ActionObservation || action.Class == ActionProposal {
		if action.RequiresEconomicIntent && !checks.EconomicIntentConfirmed {
			return decision(PolicyHumanRequired, "ECONOMIC_CONFIRMATION_REQUIRED")
		}
		return decision(PolicyAllow, "READ_OR_PROPOSAL_PERMITTED")
	}
	if mode == ModeReadOnly {
		return decision(PolicyDeny, "READ_ONLY_MODE")
	}
	if action.Class == ActionProtected {
		return decision(PolicySeparateAuthorization, "SEPARATE_AUTHORIZATION_REQUIRED")
	}
	if action.RequiresEconomicIntent && !checks.EconomicIntentConfirmed {
		return decision(PolicyHumanRequired, "ECONOMIC_CONFIRMATION_REQUIRED")
	}
	if action.Class == ActionImageGeneration {
		return decision(PolicyAllow, "CONFIRMED_IMAGE_PLAN")
	}
	if checks.HumanContentConfirmed {
		return decision(PolicyAllow, "HUMAN_CONTENT_CONFIRMED")
	}
	if mode == ModeFullAccess {
		return decision(PolicyAllow, "POLICY_PREAUTHORIZED_LOCAL_ACTION")
	}
	if mode == ModeAutoApprove && action.AutomaticApproval && checks.Automatic == RulePassed {
		return decision(PolicyAllow, "AUTOMATIC_RULE_PASSED")
	}
	return decision(PolicyHumanRequired, "HUMAN_CONTENT_CONFIRMATION_REQUIRED")
}
