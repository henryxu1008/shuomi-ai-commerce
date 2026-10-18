package currentapplication

import (
	"encoding/json"
	"github.com/stretchr/testify/require"
	governed "task-processor/internal/integration/aicapability/einomodel"
	"testing"
)

func contentTasksConfig(t *testing.T) *Config {
	c := contentTitleRuntimeConfig(t)
	c.ProductCollections = true
	require.NoError(t, json.Unmarshal([]byte(`{"contentTitleOnly":false,"contentOnly":true}`), c.ProductAgent))
	policy := c.ProductAgent.ContentTextPolicies["org"]
	policy.Profile.ClientName = "chat"
	policy.Profile.PromptVersion = "ai-workbench-chat-plan-v2"
	policy.Profile.OutputSchemaVersion = "ai-workbench-plan-decision-v1"
	policy.Profile.MaximumCompletionTokens = 4096
	policy.Profile.MaximumInputBytes = 128 << 10
	c.AIWorkbench.ConversationOnly = false
	require.NoError(t, json.Unmarshal([]byte(`{"contentTasks":true}`), c.AIWorkbench))
	c.AIWorkbench.PlanningTextPolicies = map[string]governed.RoutePolicy{"org": policy}
	return c
}
func TestContentTasksRequireExplicitBoundedContentConsumer(t *testing.T) {
	require.NoError(t, contentTasksConfig(t).validate())
	for name, change := range map[string]func(*Config){
		"both profiles": func(c *Config) { c.ProductAgent.ContentTitleOnly = true },
		"title policy":  func(c *Config) { c.ProductAgent.TextPolicies = agentRuntimeConfig().ProductAgent.TextPolicies },
		"asset":         func(c *Config) { c.ProductAgent.AssetDatabase = agentRuntimeConfig().ProductAgent.AssetDatabase },
		"storage only":  func(c *Config) { c.AIWorkbench.ConversationOnly = true },
		"no allowlist":  func(c *Config) { c.AIWorkbench.AllowedOrganizationIDs = nil },
		"wrong planning version": func(c *Config) {
			p := c.AIWorkbench.PlanningTextPolicies["org"]
			p.Profile.PromptVersion = "ai-workbench-chat-plan-v1"
			c.AIWorkbench.PlanningTextPolicies["org"] = p
		},
	} {
		t.Run(name, func(t *testing.T) { c := contentTasksConfig(t); change(c); require.Error(t, c.validate()) })
	}
}
