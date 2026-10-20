package currentapplication

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"task-processor/internal/aicapability"
	d "task-processor/internal/aicapability/routeconfig"
	governed "task-processor/internal/integration/aicapability/einomodel"
	qualified "task-processor/internal/integration/aicapability/routeconfig"
	"task-processor/internal/integration/openai"
)

func planningV2ProvisionFixture(t *testing.T) (*Config, TitleCredentialProvision) {
	t.Helper()
	cfg := contentTasksConfig(t)
	require.NoError(t, cfg.validate(), "use the actual content-task planning contract")
	policy := cfg.AIWorkbench.PlanningTextPolicies["org"]
	writer := cfg.ProductAgent.Database
	writer.User = "title_credential_writer"
	return cfg, TitleCredentialProvision{Action: "upsert", Consumer: "planning", OrganizationID: "org", ClientName: policy.Profile.ClientName,
		APIKey: "synthetic-planning-key", BaseURL: "https://grsaiapi.com/v1", Model: policy.Profile.ModelID,
		APIStyle: policy.Profile.AdapterKind, TimeoutSecond: 3, WriterDatabase: writer}
}

func TestPlanningV2CredentialProvisionCreatesAndRotatesOnlyAdmittedBinding(t *testing.T) {
	for _, managed := range []bool{false, true} {
		t.Run(map[bool]string{false: "initial route", true: "managed candidate"}[managed], func(t *testing.T) {
			cfg, input := planningV2ProvisionFixture(t)
			policy := cfg.AIWorkbench.PlanningTextPolicies["org"]
			policy.AdmittedCredentialVersion = ""
			if managed {
				policy.Profile.ClientName, input.ClientName = "chat-candidate", "chat-candidate"
				writer := cfg.ProductAgent.Database
				writer.User, writer.MaxConnections = "model_route_config_writer", 2
				cfg.ModelRoutePricing = &ModelRoutePricingConfig{TextWriter: &writer, Bindings: []qualified.Definition{{
					Scope: d.Scope{OrganizationID: "org", Capability: string(aicapability.OperationAIWorkbenchChatPlan)}, Text: &policy,
				}}}
			} else {
				cfg.AIWorkbench.PlanningTextPolicies["org"] = policy
				cfg.ProductAgent.Enabled, cfg.AIWorkbench.Enabled = false, false
			}
			db := contentProvisionDB(t)
			store := openai.NewGormCredentialResolver(db)
			require.NoError(t, store.SaveCredential(context.Background(), openai.AIClientCredential{
				TenantID: "org", UserID: "member", ClientName: input.ClientName, APIKey: "synthetic-member-key", Enabled: true,
				BaseURL: "https://member.example.test/v1", Model: "member-model", APIStyle: "openai-compatible",
			}))
			before, err := store.GetCredential(context.Background(), "org", "member", input.ClientName)
			require.NoError(t, err)
			require.NoError(t, ValidateTitleCredentialProvision(cfg, input), "CLI validation must admit v2 before opening the DB")
			first, err := SaveTitleCredential(context.Background(), cfg, input, db)
			require.NoError(t, err)
			require.NotEmpty(t, first.CredentialVersion)
			require.Equal(t, governed.EndpointIdentityDigest(input.BaseURL), first.EndpointIdentityDigest)
			policy.AdmittedCredentialVersion, policy.AdmittedEndpointIdentityDigest = first.CredentialVersion, first.EndpointIdentityDigest
			if !managed {
				cfg.ProductAgent.Enabled, cfg.AIWorkbench.Enabled = true, true
				cfg.AIWorkbench.PlanningTextPolicies["org"] = policy
			}
			require.NoError(t, cfg.validate(), "returned binding must satisfy the actual consumer configuration")
			resolver := func(p governed.RoutePolicy) *governed.OrganizationRouteResolver {
				r, err := governed.NewOrganizationRouteResolver(openai.NewOrganizationOnlyCredentialResolver(db),
					map[governed.RouteKey]governed.RoutePolicy{{OrganizationID: "org", Operation: aicapability.OperationAIWorkbenchChatPlan}: p})
				require.NoError(t, err)
				return r
			}
			identity := aicapability.TextInputIdentity{OrganizationID: "org", Operation: aicapability.OperationAIWorkbenchChatPlan}
			oldResolver := resolver(policy)
			_, err = oldResolver.Resolve(context.Background(), identity)
			require.NoError(t, err, "resolve checks metadata only; it does not call a model")
			input.APIKey = "synthetic-rotated-key"
			rotated, err := SaveTitleCredential(context.Background(), cfg, input, db)
			require.NoError(t, err)
			require.NotEqual(t, first.CredentialVersion, rotated.CredentialVersion)
			_, err = oldResolver.Resolve(context.Background(), identity)
			require.Error(t, err, "rotation must invalidate the exact old binding")
			policy.AdmittedCredentialVersion = rotated.CredentialVersion
			_, err = resolver(policy).Resolve(context.Background(), identity)
			require.NoError(t, err)
			stored, err := store.GetCredential(context.Background(), "org", "", input.ClientName)
			require.NoError(t, err)
			require.Equal(t, input.APIKey, stored.APIKey)
			var count int64
			require.NoError(t, db.Model(&openai.AIClientCredential{}).Count(&count).Error)
			require.EqualValues(t, 2, count, "rotation must reuse the organization row")
			after, err := store.GetCredential(context.Background(), "org", "member", input.ClientName)
			require.NoError(t, err)
			require.Equal(t, before, after)
		})
	}
}

func TestPlanningCredentialProvisionRequiresModeSpecificPrompt(t *testing.T) {
	for _, contentTasks := range []bool{false, true} {
		for _, prompt := range []string{"ai-workbench-chat-plan-v1", "ai-workbench-chat-plan-v2", "ai-workbench-chat-plan-v3"} {
			t.Run(map[bool]string{false: "title", true: "content"}[contentTasks]+"/"+prompt, func(t *testing.T) {
				cfg, input := planningV2ProvisionFixture(t)
				policy := cfg.AIWorkbench.PlanningTextPolicies["org"]
				if !contentTasks {
					cfg = agentRuntimeConfig()
					cfg.AIWorkbench = &AIWorkbenchConfig{Enabled: true, Database: cfg.ProductAgent.Database,
						PlanningTextPolicies: map[string]governed.RoutePolicy{"org": policy}}
					cfg.AIWorkbench.Database.User = "ai_workbench_runtime"
				}
				policy.Profile.PromptVersion = prompt
				cfg.AIWorkbench.PlanningTextPolicies["org"] = policy
				expected := "ai-workbench-chat-plan-v1"
				if contentTasks {
					expected = "ai-workbench-chat-plan-v2"
				}
				if prompt == expected {
					require.NoError(t, cfg.validate())
					require.NoError(t, ValidateTitleCredentialProvision(cfg, input))
				} else {
					require.Error(t, cfg.validate())
					require.Error(t, ValidateTitleCredentialProvision(cfg, input))
				}
			})
		}
	}
}

func TestPlanningV2CredentialProvisionKeepsExistingAdmissionLimits(t *testing.T) {
	for name, change := range map[string]func(*Config, *TitleCredentialProvision, *aicapability.ModelProfile){
		"schema": func(_ *Config, _ *TitleCredentialProvision, p *aicapability.ModelProfile) {
			p.OutputSchemaVersion = "ai-workbench-plan-decision-v2"
		},
		"completion": func(_ *Config, _ *TitleCredentialProvision, p *aicapability.ModelProfile) {
			p.MaximumCompletionTokens = 4097
		},
		"input": func(_ *Config, _ *TitleCredentialProvision, p *aicapability.ModelProfile) {
			p.MaximumInputBytes = (128 << 10) + 1
		},
		"output": func(_ *Config, _ *TitleCredentialProvision, p *aicapability.ModelProfile) {
			p.MaximumOutputBytes = (16 << 10) + 1
		},
		"cost": func(c *Config, _ *TitleCredentialProvision, _ *aicapability.ModelProfile) {
			c.ProductAgent.CostMicros = 1
		},
		"endpoint": func(_ *Config, in *TitleCredentialProvision, _ *aicapability.ModelProfile) {
			in.BaseURL = "https://other.example.test/v1"
		},
		"client": func(_ *Config, in *TitleCredentialProvision, _ *aicapability.ModelProfile) {
			in.ClientName = "unadmitted"
		},
		"writer": func(c *Config, in *TitleCredentialProvision, _ *aicapability.ModelProfile) {
			in.WriterDatabase.User = c.ProductAgent.Database.User
		},
		"timeout": func(_ *Config, in *TitleCredentialProvision, _ *aicapability.ModelProfile) { in.TimeoutSecond = 1 },
	} {
		t.Run(name, func(t *testing.T) {
			cfg, input := planningV2ProvisionFixture(t)
			policy := cfg.AIWorkbench.PlanningTextPolicies["org"]
			change(cfg, &input, &policy.Profile)
			cfg.AIWorkbench.PlanningTextPolicies["org"] = policy
			require.Error(t, ValidateTitleCredentialProvision(cfg, input))
			db := contentProvisionDB(t)
			_, err := SaveTitleCredential(context.Background(), cfg, input, db)
			require.Error(t, err)
			var count int64
			require.NoError(t, db.Model(&openai.AIClientCredential{}).Count(&count).Error)
			require.Zero(t, count, "rejected input must not write credentials")
		})
	}
}
