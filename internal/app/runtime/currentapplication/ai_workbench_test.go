package currentapplication

import (
	"encoding/json"
	"testing"

	governed "task-processor/internal/integration/aicapability/einomodel"
	"task-processor/internal/integration/aiworkbench/einorouter"
)

func TestAIWorkbenchConversationOnlyNeedsNoModel(t *testing.T) {
	cfg := agentRuntimeConfig()
	cfg.ProductAgent.Enabled = false
	cfg.CommercialOwnerDatabase = nil
	var w AIWorkbenchConfig
	if err := json.Unmarshal([]byte(`{"enabled":true,"conversationOnly":true,"allowedOrganizationIds":["org"]}`), &w); err != nil {
		t.Fatal(err)
	}
	w.Database = cfg.ProductAgent.Database
	w.Database.User = "ai_workbench_runtime"
	if err := w.validate(cfg); err != nil {
		t.Fatalf("model-free conversation management rejected: %v", err)
	}
	for _, test := range []struct {
		name   string
		change func(*AIWorkbenchConfig)
	}{
		{"implicit-full", func(v *AIWorkbenchConfig) { v.ConversationOnly = false }},
		{"empty-organizations", func(v *AIWorkbenchConfig) { v.AllowedOrganizationIDs = nil }},
		{"duplicate-organizations", func(v *AIWorkbenchConfig) { v.AllowedOrganizationIDs = []string{"org", "org"} }},
		{"invalid-organization", func(v *AIWorkbenchConfig) { v.AllowedOrganizationIDs = []string{" org"} }},
		{"wrong-owner", func(v *AIWorkbenchConfig) { v.Database.Database = "other" }},
		{"wrong-role", func(v *AIWorkbenchConfig) { v.Database.User = "product_agent_runtime" }},
		{"unbounded-pool", func(v *AIWorkbenchConfig) { v.Database.MaxConnections = 9 }},
		{"model-policy", func(v *AIWorkbenchConfig) { v.PlanningTextPolicies = map[string]governed.RoutePolicy{"org": {}} }},
		{"chat-intent-policy", func(v *AIWorkbenchConfig) { v.ChatIntentTextPolicies = map[string]governed.RoutePolicy{"org": {}} }},
		{"chat-turns-without-execution",func(v *AIWorkbenchConfig){v.ChatTurnsEnabled=true}},
	} {
		t.Run(test.name, func(t *testing.T) {
			copy := w
			test.change(&copy)
			if copy.validate(cfg) == nil {
				t.Fatal("unsafe conversation configuration admitted")
			}
		})
	}
	cfg.ProductAgent.Enabled = true
	if w.validate(cfg) == nil {
		t.Fatal("conversation-only mode admitted alongside execution")
	}
}

func TestStoreRoutePolicyNeedsItsOwnExplicitContract(t *testing.T) {
	cfg := agentRuntimeConfig()
	policy := cfg.ProductAgent.TextPolicies["org"]
	policy.Profile.ClientName = "chat"
	policy.Profile.PromptVersion = "ai-workbench-chat-plan-v1"
	policy.Profile.OutputSchemaVersion = "ai-workbench-plan-decision-v1"
	policy.Profile.MaximumCompletionTokens = 1024
	policy.Profile.MaximumInputBytes = 128 << 10
	policy.Profile.MaximumOutputBytes = 16 << 10
	w := &AIWorkbenchConfig{Enabled: true, Database: cfg.ProductAgent.Database, PlanningTextPolicies: map[string]governed.RoutePolicy{"org": policy}}
	w.Database.User = "ai_workbench_runtime"
	cfg.AIWorkbench = w
	route := policy
	route.Profile.PromptVersion = einorouter.PromptVersion
	route.Profile.OutputSchemaVersion = einorouter.SchemaVersion
	w.StoreRouteTextPolicies = map[string]governed.RoutePolicy{"org": route}
	if err := w.validate(cfg); err != nil {
		t.Fatal(err)
	}
	route.Profile.PromptVersion = policy.Profile.PromptVersion
	w.StoreRouteTextPolicies["org"] = route
	if err := w.validate(cfg); err == nil {
		t.Fatal("planning prompt cannot qualify as route contract")
	}
}

func TestAIWorkbenchRejectsSharedProductAgentRuntimeRole(t *testing.T) {
	cfg := agentRuntimeConfig()
	planning := cfg.ProductAgent.TextPolicies["org"]
	planning.Profile.ClientName = "chat"
	planning.Profile.PromptVersion = "ai-workbench-chat-plan-v1"
	planning.Profile.OutputSchemaVersion = "ai-workbench-plan-decision-v1"
	planning.Profile.MaximumCompletionTokens = 4096
	planning.Profile.MaximumInputBytes = 128 << 10
	cfg.AIWorkbench = &AIWorkbenchConfig{Enabled: true, Database: cfg.ProductAgent.Database,
		PlanningTextPolicies: map[string]governed.RoutePolicy{"org": planning}}
	cfg.AIWorkbench.Database.User = "ai_workbench_runtime"
	if err := cfg.AIWorkbench.validate(cfg); err != nil {
		t.Fatalf("distinct runtime role rejected: %v", err)
	}
	cfg.ProductAgent.Database.User = cfg.AIWorkbench.Database.User
	if err := cfg.AIWorkbench.validate(cfg); err == nil {
		t.Fatal("shared Product Agent and AI Workbench runtime role admitted")
	}
}

func TestChatImageStorageRequiresHTTPSOutsidePrivateNetwork(t *testing.T) {
	cfg := agentRuntimeConfig()
	cfg.ProductCollections = true
	cfg.StorageCapacity = true
	cfg.ModelRoutePricing = &ModelRoutePricingConfig{}
	planning := cfg.ProductAgent.TextPolicies["org"]
	planning.Profile.PromptVersion = "ai-workbench-chat-plan-v3"
	planning.Profile.OutputSchemaVersion = "ai-workbench-plan-decision-v1"
	planning.Profile.MaximumCompletionTokens = 4096
	planning.Profile.MaximumInputBytes = 128 << 10
	planning.Profile.MaximumOutputBytes = 16 << 10
	cfg.ProductAgent.ContentTextPolicies = cfg.ProductAgent.TextPolicies
	w := &AIWorkbenchConfig{
		Enabled:                true,
		ContentTasks:           true,
		Database:               cfg.ProductAgent.Database,
		AllowedOrganizationIDs: []string{"org"},
		PlanningTextPolicies:   map[string]governed.RoutePolicy{"org": planning},
		ChatImageStorage: &KnowledgeStorageConfig{
			Region: "local", Bucket: "chat-images", Mode: "aws",
			AccessKeyID: "isolated-access", SecretAccessKey: "isolated-secret",
		},
	}
	w.Database.User = "ai_workbench_runtime"
	for _, tc := range []struct {
		endpoint string
		allowed  bool
	}{
		{"http://objects.example.com:9000", false},
		{"https://objects.example.com", true},
		{"http://127.0.0.1:9000", true},
		{"http://knowledge-objects:9000", true},
		{"", true},
	} {
		t.Run(tc.endpoint, func(t *testing.T) {
			w.ChatImageStorage.Endpoint = tc.endpoint
			if err := w.validate(cfg); (err == nil) != tc.allowed {
				t.Fatalf("endpoint allowed = %t, want %t (error: %v)", err == nil, tc.allowed, err)
			}
		})
	}
}

func TestScheduledWorkbenchRequiresExplicitExecutionAndTemporal(t *testing.T) {
	cfg := agentRuntimeConfig()
	cfg.Identity.TenantDirectoryToken = "fixture-service-token"
	w := AIWorkbenchConfig{Enabled: true, ContentTasks: true, Schedules: &AIWorkbenchSchedulesConfig{TemporalAddress: "127.0.0.1:7233", TemporalNamespace: "default"}}
	if err := w.validateSchedules(cfg); err != nil {
		t.Fatal(err)
	}
	for _, change := range []func(*AIWorkbenchConfig){
		func(c *AIWorkbenchConfig) { c.ConversationOnly = true },
		func(c *AIWorkbenchConfig) { c.ContentTasks = false },
		func(c *AIWorkbenchConfig) { c.Enabled = false },
		func(c *AIWorkbenchConfig) { s := *c.Schedules; s.TemporalAddress = "0.0.0.0:7233"; c.Schedules = &s },
		func(c *AIWorkbenchConfig) { s := *c.Schedules; s.TemporalNamespace = ""; c.Schedules = &s },
	} {
		copy := w
		change(&copy)
		if copy.validateSchedules(cfg) == nil {
			t.Fatal("incomplete scheduled execution admitted")
		}
	}
}
