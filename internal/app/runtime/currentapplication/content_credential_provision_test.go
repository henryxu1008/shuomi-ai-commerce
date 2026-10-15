package currentapplication

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"task-processor/internal/agentconfig"
	"task-processor/internal/aicapability"
	governed "task-processor/internal/integration/aicapability/einomodel"
	"task-processor/internal/integration/googleinteractions"
	"task-processor/internal/integration/openai"
)

func contentProvisionFixture() (*Config, TitleCredentialProvision) {
	cfg := agentRuntimeConfig()
	cfg.ProductAgent.Enabled = false
	policy := cfg.ProductAgent.TextPolicies["org"]
	policy.Profile.ClientName, policy.Profile.ProviderID, policy.Profile.ModelID = "content", "content-provider", "content-model"
	policy.Profile.PromptVersion, policy.Profile.OutputSchemaVersion = agentconfig.ContentPromptVersion, agentconfig.ContentActionVersion
	policy.AdmittedCredentialVersion, policy.AdmittedEndpointIdentityDigest = "", ""
	cfg.ProductAgent.ContentTextPolicies = map[string]governed.RoutePolicy{"org": policy}
	writer := cfg.ProductAgent.Database
	writer.User = "credential_writer"
	return cfg, TitleCredentialProvision{Action: "upsert", Consumer: "content", OrganizationID: "org", ClientName: "content",
		APIKey: "synthetic-content-key", BaseURL: "https://content.example.test/v1", Model: "content-model",
		APIStyle: "openai-compatible", TimeoutSecond: 3, WriterDatabase: writer}
}

func contentProvisionDB(t *testing.T) *gorm.DB {
	t.Helper()
	// Keep separate writes on distinct clock ticks without timing assumptions
	// about the host's clock resolution.
	now := time.Date(2026, 10, 15, 0, 0, 0, 0, time.UTC)
	db, err := gorm.Open(sqlite.Dialector{DriverName: "sqlite", DSN: ":memory:"}, &gorm.Config{NowFunc: func() time.Time {
		now = now.Add(time.Second)
		return now
	}})
	require.NoError(t, err)
	sqlDB, err := db.DB()
	require.NoError(t, err)
	t.Cleanup(func() { _ = sqlDB.Close() })
	require.NoError(t, db.AutoMigrate(&openai.AIClientCredential{}))
	return db
}

func TestContentCredentialProvisionUsesOnlyItsOwnRouteAndOrganizationRow(t *testing.T) {
	for _, adapter := range []string{"openai-compatible", "claude-native"} {
		t.Run(adapter, func(t *testing.T) {
			cfg, input := contentProvisionFixture()
			policy := cfg.ProductAgent.ContentTextPolicies["org"]
			policy.Profile.AdapterKind, input.APIStyle = adapter, adapter
			cfg.ProductAgent.ContentTextPolicies["org"] = policy
			// First provisioning must not require title/planning policies or enabled execution.
			cfg.ProductAgent.TextPolicies = nil
			db := contentProvisionDB(t)
			store := openai.NewGormCredentialResolver(db)
			untouched := []openai.AIClientCredential{
				{TenantID: "org", ClientName: "text", APIKey: "synthetic-title", Enabled: true},
				{TenantID: "org", ClientName: "chat", APIKey: "synthetic-planning", Enabled: true},
				{TenantID: "org", UserID: "member", ClientName: "content", APIKey: "synthetic-member", Enabled: true},
				{TenantID: "other-org", ClientName: "content", APIKey: "synthetic-other-org", Enabled: true},
			}
			for _, row := range untouched {
				row.BaseURL, row.Model, row.APIStyle = "https://existing.example.test/v1", "existing-model", "openai-compatible"
				require.NoError(t, store.SaveCredential(context.Background(), row))
			}
			result, err := SaveTitleCredential(context.Background(), cfg, input, db)
			require.NoError(t, err)
			require.Equal(t, "content-provider", result.ProviderID)
			require.True(t, result.Enabled)
			require.NotEmpty(t, result.CredentialVersion)
			require.Equal(t, governed.EndpointIdentityDigest(input.BaseURL), result.EndpointIdentityDigest)
			encoded, err := json.Marshal(result)
			require.NoError(t, err)
			require.NotContains(t, string(encoded), input.APIKey)
			require.NotContains(t, string(encoded), input.WriterDatabase.Password)
			stored, err := store.GetCredential(context.Background(), "org", "", "content")
			require.NoError(t, err)
			require.NotNil(t, stored)
			require.Equal(t, input.APIKey, stored.APIKey)
			require.Equal(t, adapter, stored.APIStyle)
			require.Equal(t, governed.CredentialVersion(*stored), result.CredentialVersion)
			policy.AdmittedCredentialVersion, policy.AdmittedEndpointIdentityDigest = result.CredentialVersion, result.EndpointIdentityDigest
			cfg.ProductAgent.ContentTextPolicies["org"] = policy
			resolver, err := governed.NewOrganizationRouteResolver(openai.NewOrganizationOnlyCredentialResolver(db),
				map[governed.RouteKey]governed.RoutePolicy{{OrganizationID: "org", Operation: aicapability.OperationProductContentDecision}: policy})
			require.NoError(t, err)
			identity := aicapability.TextInputIdentity{OrganizationID: "org", Operation: aicapability.OperationProductContentDecision}
			_, err = resolver.Resolve(context.Background(), identity)
			require.NoError(t, err, "returned binding must resolve through the existing content owner")
			input.APIKey = "synthetic-rotated-key"
			rotated, err := SaveTitleCredential(context.Background(), cfg, input, db)
			require.NoError(t, err)
			require.NotEqual(t, result.CredentialVersion, rotated.CredentialVersion)
			_, err = resolver.Resolve(context.Background(), identity)
			require.Error(t, err, "rotation must invalidate the old admitted binding")
			disable := TitleCredentialProvision{Action: "disable", Consumer: "content", OrganizationID: "org", ClientName: "content", WriterDatabase: input.WriterDatabase}
			result, err = SaveTitleCredential(context.Background(), cfg, disable, db)
			require.NoError(t, err)
			require.False(t, result.Enabled)
			stored, err = store.GetCredential(context.Background(), "org", "", "content")
			require.NoError(t, err)
			require.NotNil(t, stored)
			require.False(t, stored.Enabled)
			var count int64
			require.NoError(t, db.Model(&openai.AIClientCredential{}).Count(&count).Error)
			require.EqualValues(t, len(untouched)+1, count, "rotation/disable must reuse the same organization credential row")
			_, err = resolver.Resolve(context.Background(), identity)
			require.Error(t, err, "disabled content must not borrow member or other route credentials")
			for _, original := range untouched {
				row, err := store.GetCredential(context.Background(), original.TenantID, original.UserID, original.ClientName)
				require.NoError(t, err)
				require.NotNil(t, row)
				require.Equal(t, original.APIKey, row.APIKey)
				require.True(t, row.Enabled)
			}
		})
	}
}

func TestContentCredentialProvisionRejectsUnadmittedRoutesBeforeWriting(t *testing.T) {
	for name, change := range map[string]func(*Config, *TitleCredentialProvision){
		"missing content cannot borrow title or planning": func(c *Config, in *TitleCredentialProvision) {
			policy := c.ProductAgent.ContentTextPolicies["org"]
			c.ProductAgent.TextPolicies["org"] = policy
			c.AIWorkbench = &AIWorkbenchConfig{PlanningTextPolicies: map[string]governed.RoutePolicy{"org": policy}}
			c.ProductAgent.ContentTextPolicies = nil
		},
		"wrong organization": func(c *Config, in *TitleCredentialProvision) { in.OrganizationID = "other-org" },
		"wrong client":       func(c *Config, in *TitleCredentialProvision) { in.ClientName = "text" },
		"wrong model":        func(c *Config, in *TitleCredentialProvision) { in.Model = "title-model" },
		"wrong protocol":     func(c *Config, in *TitleCredentialProvision) { in.APIStyle = "claude-native" },
		"unknown consumer":   func(c *Config, in *TitleCredentialProvision) { in.Consumer = "unknown" },
		"wrong endpoint": func(c *Config, in *TitleCredentialProvision) {
			p := c.ProductAgent.ContentTextPolicies["org"]
			p.AdmittedEndpointIdentityDigest = governed.EndpointIdentityDigest("https://admitted.example.test/v1")
			c.ProductAgent.ContentTextPolicies["org"] = p
		},
		"wrong prompt": func(c *Config, in *TitleCredentialProvision) {
			p := c.ProductAgent.ContentTextPolicies["org"]
			p.Profile.PromptVersion = "product-title-agent-v1"
			c.ProductAgent.ContentTextPolicies["org"] = p
		},
		"wrong output": func(c *Config, in *TitleCredentialProvision) {
			p := c.ProductAgent.ContentTextPolicies["org"]
			p.Profile.OutputSchemaVersion = "product-title-action-v1"
			c.ProductAgent.ContentTextPolicies["org"] = p
		},
		"title-only Google adapter": func(c *Config, in *TitleCredentialProvision) {
			p := c.ProductAgent.ContentTextPolicies["org"]
			p.Profile.ProviderID, p.Profile.AdapterKind, p.Profile.ModelID = "google", "google-interactions", googleinteractions.Model
			p.Profile.AdapterPolicyVersion, p.Profile.UsageMappingVersion = googleinteractions.AdapterPolicyVersion, googleinteractions.UsageMappingVersion
			p.Profile.MaximumInputBytes, p.Profile.MaximumOutputBytes = 128<<10, 256<<10
			c.ProductAgent.ContentTextPolicies["org"] = p
			in.BaseURL, in.APIStyle, in.Model = googleinteractions.Origin, "google-interactions", googleinteractions.Model
		},
	} {
		t.Run(name, func(t *testing.T) {
			cfg, input := contentProvisionFixture()
			change(cfg, &input)
			db := contentProvisionDB(t)
			_, err := SaveTitleCredential(context.Background(), cfg, input, db)
			require.Error(t, err)
			var count int64
			require.NoError(t, db.Model(&openai.AIClientCredential{}).Count(&count).Error)
			require.Zero(t, count, "rejected input must not write any credential")
		})
	}
}

func TestCredentialProvisionRejectsCrossConsumerClientCollision(t *testing.T) {
	for _, other := range []string{"title", "planning", "materials", "selection", "diagnosis"} {
		for _, action := range []string{"upsert", "disable"} {
			t.Run(other+"/"+action, func(t *testing.T) {
				cfg, input := contentProvisionFixture()
				policy := cfg.ProductAgent.ContentTextPolicies["org"]
				// Compare the owner's canonical client key even for an inactive,
				// malformed sibling policy; do not rely on its runtime validation.
				policy.Profile.ClientName = " \tcontent\n"
				switch other {
				case "title":
					cfg.ProductAgent.TextPolicies["org"] = policy
				case "planning":
					cfg.AIWorkbench = &AIWorkbenchConfig{PlanningTextPolicies: map[string]governed.RoutePolicy{"org": policy}}
				case "materials":
					cfg.ProductAgent.MaterialsTextPolicies = map[string]governed.RoutePolicy{"org": policy}
				case "selection":
					cfg.ProductAgent.SelectionTextPolicies = map[string]governed.RoutePolicy{"org": policy}
				case "diagnosis":
					cfg.ProductAgent.DiagnosisTextPolicies = map[string]governed.RoutePolicy{"org": policy}
				}
				assertCredentialCollisionPreservesRow(t, cfg, input, action)
			})
		}
	}
	for _, consumer := range []string{"title", "planning"} {
		for _, action := range []string{"upsert", "disable"} {
			t.Run("reverse/"+consumer+"/"+action, func(t *testing.T) {
				cfg, input := contentProvisionFixture()
				policy := cfg.ProductAgent.ContentTextPolicies["org"]
				input.Consumer = consumer
				if consumer == "title" {
					policy.Profile.PromptVersion, policy.Profile.OutputSchemaVersion = "product-title-agent-v1", "product-title-action-v1"
					cfg.ProductAgent.TextPolicies["org"] = policy
				} else {
					policy.Profile.PromptVersion, policy.Profile.OutputSchemaVersion = "ai-workbench-chat-plan-v1", "ai-workbench-plan-decision-v1"
					policy.Profile.MaximumCompletionTokens, policy.Profile.MaximumInputBytes = 4096, 128<<10
					cfg.AIWorkbench = &AIWorkbenchConfig{PlanningTextPolicies: map[string]governed.RoutePolicy{"org": policy}}
				}
				assertCredentialCollisionPreservesRow(t, cfg, input, action)
			})
		}
	}
}

func assertCredentialCollisionPreservesRow(t *testing.T, cfg *Config, input TitleCredentialProvision, action string) {
	t.Helper()
	db := contentProvisionDB(t)
	store := openai.NewGormCredentialResolver(db)
	require.NoError(t, store.SaveCredential(context.Background(), openai.AIClientCredential{
		TenantID: "org", ClientName: input.ClientName, APIKey: "synthetic-preserved-key", BaseURL: "https://original.example.test/v1",
		Model: "preserved-model", APIStyle: "openai-compatible", Enabled: true,
	}))
	before, err := store.GetCredential(context.Background(), "org", "", input.ClientName)
	require.NoError(t, err)
	require.NotNil(t, before)
	input.Action = action
	if action == "disable" {
		input.APIKey, input.BaseURL, input.Model, input.APIStyle, input.TimeoutSecond = "", "", "", "", 0
	}
	_, err = SaveTitleCredential(context.Background(), cfg, input, db)
	require.Error(t, err, "ambiguous credential targets must fail before any write")
	after, err := store.GetCredential(context.Background(), "org", "", input.ClientName)
	require.NoError(t, err)
	require.Equal(t, before, after, "the entire existing credential and its version must be preserved")
}

func TestCredentialProvisionRejectsDefaultClientCollision(t *testing.T) {
	for _, action := range []string{"upsert", "disable"} {
		t.Run(action, func(t *testing.T) {
			cfg, input := contentProvisionFixture()
			policy := cfg.ProductAgent.ContentTextPolicies["org"]
			input.ClientName, policy.Profile.ClientName = "default", "default"
			cfg.ProductAgent.ContentTextPolicies["org"] = policy
			policy.Profile.ClientName = " \t"
			cfg.ProductAgent.TextPolicies["org"] = policy
			assertCredentialCollisionPreservesRow(t, cfg, input, action)
		})
	}
}

func TestContentCredentialProvisionKeepsDistinctOwnerKeysAvailable(t *testing.T) {
	for name, key := range map[string][2]string{"other organization": {"other-org", "content"}, "case-sensitive client": {"org", "Content"}} {
		t.Run(name, func(t *testing.T) {
			cfg, input := contentProvisionFixture()
			policy := cfg.ProductAgent.ContentTextPolicies["org"]
			policy.Profile.ClientName = key[1]
			cfg.ProductAgent.TextPolicies = map[string]governed.RoutePolicy{key[0]: policy}
			db := contentProvisionDB(t)
			store := openai.NewGormCredentialResolver(db)
			require.NoError(t, store.SaveCredential(context.Background(), openai.AIClientCredential{
				TenantID: key[0], ClientName: key[1], APIKey: "synthetic-other-key", BaseURL: "https://original.example.test/v1",
				Model: "preserved-model", APIStyle: "openai-compatible", Enabled: true,
			}))
			before, err := store.GetCredential(context.Background(), key[0], "", key[1])
			require.NoError(t, err)
			require.NotNil(t, before)
			_, err = SaveTitleCredential(context.Background(), cfg, input, db)
			require.NoError(t, err)
			after, err := store.GetCredential(context.Background(), key[0], "", key[1])
			require.NoError(t, err)
			require.Equal(t, before, after)
		})
	}
}
