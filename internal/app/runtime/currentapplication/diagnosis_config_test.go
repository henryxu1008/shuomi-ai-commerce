package currentapplication

import (
	"github.com/stretchr/testify/require"
	"task-processor/internal/agentconfig"
	governed "task-processor/internal/integration/aicapability/einomodel"
	"testing"
)

func TestDiagnosisPolicyCannotBorrowAnotherAgentOrExpandCurrentBudget(t *testing.T) {
	cfg := agentRuntimeConfig()
	policy := cfg.ProductAgent.TextPolicies["org"]
	cfg.ProductAgent.DiagnosisTextPolicies = map[string]governed.RoutePolicy{"org": policy}
	require.Error(t, cfg.validate())
	policy.Profile.PromptVersion = agentconfig.DiagnosisPromptVersion
	policy.Profile.OutputSchemaVersion = agentconfig.DiagnosisActionVersion
	cfg.ProductAgent.DiagnosisTextPolicies["org"] = policy
	require.NoError(t, cfg.validate())
	cfg.ProductAgent.DiagnosisTextPolicies["foreign"] = policy
	require.Error(t, cfg.validate())
	delete(cfg.ProductAgent.DiagnosisTextPolicies, "foreign")
	policy.Profile.MaximumCompletionTokens = cfg.ProductAgent.Tokens
	cfg.ProductAgent.DiagnosisTextPolicies["org"] = policy
	require.Error(t, cfg.validate())
}

func TestDiagnosisPolicyRequiresTheMandatoryFlowBudget(t *testing.T) {
	for _, limits := range [][2]int{{1, 2}, {2, 2}, {3, 2}, {4, 1}, {4, 2}} {
		cfg := agentRuntimeConfig()
		policy := cfg.ProductAgent.TextPolicies["org"]
		policy.Profile.PromptVersion, policy.Profile.OutputSchemaVersion = agentconfig.DiagnosisPromptVersion, agentconfig.DiagnosisActionVersion
		cfg.ProductAgent.DiagnosisTextPolicies = map[string]governed.RoutePolicy{"org": policy}
		cfg.ProductAgent.TextPolicies = nil
		cfg.ProductAgent.Steps, cfg.ProductAgent.ModelCalls = limits[0], limits[1]
		if limits == [2]int{4, 2} {
			require.NoError(t, cfg.validate())
		} else {
			require.Error(t, cfg.validate(), "insufficient mandatory flow budget must fail before serving")
		}
	}
}

func TestDiagnosisOnlyPolicyDoesNotRequireASecondTitleModel(t *testing.T) {
	cfg := agentRuntimeConfig()
	policy := cfg.ProductAgent.TextPolicies["org"]
	policy.Profile.PromptVersion = agentconfig.DiagnosisPromptVersion
	policy.Profile.OutputSchemaVersion = agentconfig.DiagnosisActionVersion
	cfg.ProductAgent.DiagnosisTextPolicies = map[string]governed.RoutePolicy{"org": policy}
	cfg.ProductAgent.TextPolicies = nil
	require.NoError(t, cfg.validate())
	profile := policy.ShapeProfile()
	cost, err := profile.MaximumCost()
	require.NoError(t, err)
	for _, short := range []string{"tokens", "cost"} {
		limited := *cfg.ProductAgent
		limited.Tokens = 2 * (profile.MaximumPromptTokens + profile.MaximumCompletionTokens)
		limited.CostMicros = 2 * cost
		if short == "tokens" {
			limited.Tokens--
		} else {
			limited.CostMicros--
		}
		cfg.ProductAgent = &limited
		require.Error(t, cfg.validate())
	}
	cfg.ProductAgent.Tokens = 2 * (profile.MaximumPromptTokens + profile.MaximumCompletionTokens)
	cfg.ProductAgent.CostMicros = 2 * cost
	require.NoError(t, cfg.validate())
	require.Empty(t, cfg.ProductAgent.TextPolicies, "diagnosis policy is never copied to title")
	cfg.ProductAgent.DiagnosisTextPolicies = nil
	require.Error(t, cfg.validate(), "enabled execution still needs an explicitly admitted consumer")
}
