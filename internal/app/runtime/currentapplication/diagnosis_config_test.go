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

func TestDiagnosisOnlyPolicyDoesNotRequireASecondTitleModel(t *testing.T) {
	cfg := agentRuntimeConfig()
	policy := cfg.ProductAgent.TextPolicies["org"]
	policy.Profile.PromptVersion = agentconfig.DiagnosisPromptVersion
	policy.Profile.OutputSchemaVersion = agentconfig.DiagnosisActionVersion
	cfg.ProductAgent.DiagnosisTextPolicies = map[string]governed.RoutePolicy{"org": policy}
	cfg.ProductAgent.TextPolicies = nil
	require.NoError(t, cfg.validate())
	require.Empty(t, cfg.ProductAgent.TextPolicies, "diagnosis policy is never copied to title")
	cfg.ProductAgent.DiagnosisTextPolicies = nil
	require.Error(t, cfg.validate(), "enabled execution still needs an explicitly admitted consumer")
}
