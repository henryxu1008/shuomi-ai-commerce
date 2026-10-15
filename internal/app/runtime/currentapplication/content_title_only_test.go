package currentapplication

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
	"task-processor/internal/agentconfig"
	coreconfig "task-processor/internal/core/config"
	governed "task-processor/internal/integration/aicapability/einomodel"
)

func contentTitleRuntimeConfig(t *testing.T) *Config {
	t.Helper()
	cfg := agentRuntimeConfig()
	policy := cfg.ProductAgent.TextPolicies["org"]
	policy.Profile.ClientName = "content"
	policy.Profile.PromptVersion, policy.Profile.OutputSchemaVersion = agentconfig.ContentPromptVersion, agentconfig.ContentActionVersion
	cfg.ProductAgent.ContentTextPolicies = map[string]governed.RoutePolicy{"org": policy}
	cfg.ProductAgent.TextPolicies = nil
	cfg.ProductAgent.AssetDatabase = DatabaseConfig{}
	cfg.ProductAgent.Database.User = "product_agent_runtime"
	// Exercise the trusted deployment input without depending on the new field
	// existing before the regression is first run.
	require.NoError(t, json.Unmarshal([]byte(`{"contentTitleOnly":true}`), cfg.ProductAgent))
	chatDB := cfg.ProductAgent.Database
	chatDB.User = "ai_workbench_runtime"
	cfg.AIWorkbench = &AIWorkbenchConfig{Enabled: true, ConversationOnly: true, Database: chatDB, AllowedOrganizationIDs: []string{"org"}}
	return cfg
}

func TestContentTitleOnlyConfigCoexistsWithStorageOnlyChat(t *testing.T) {
	cfg := contentTitleRuntimeConfig(t)
	require.NoError(t, cfg.validate())
	for name, change := range map[string]func(*Config){
		"explicit opt-in required": func(c *Config) {
			require.NoError(t, json.Unmarshal([]byte(`{"contentTitleOnly":false}`), c.ProductAgent))
		},
		"asset connection forbidden":        func(c *Config) { c.ProductAgent.AssetDatabase = agentRuntimeConfig().ProductAgent.AssetDatabase },
		"title policy forbidden":            func(c *Config) { c.ProductAgent.TextPolicies = agentRuntimeConfig().ProductAgent.TextPolicies },
		"materials policy forbidden":        func(c *Config) { c.ProductAgent.MaterialsTextPolicies = c.ProductAgent.ContentTextPolicies },
		"selection policy forbidden":        func(c *Config) { c.ProductAgent.SelectionTextPolicies = c.ProductAgent.ContentTextPolicies },
		"diagnosis policy forbidden":        func(c *Config) { c.ProductAgent.DiagnosisTextPolicies = c.ProductAgent.ContentTextPolicies },
		"content policy required":           func(c *Config) { c.ProductAgent.ContentTextPolicies = nil },
		"run role must be prepared runtime": func(c *Config) { c.ProductAgent.Database.User = "agent_runtime" },
		"run owner forbidden":               func(c *Config) { c.ProductAgent.Database.User = "product_agent_owner" },
		"review acquisition role forbidden": func(c *Config) { c.ProductAgent.ReviewDatabase.User = "source_acquisition_runtime" },
		"review owner forbidden":            func(c *Config) { c.ProductAgent.ReviewDatabase.User = "acquisition_owner" },
		"planner policy forbidden":          func(c *Config) { c.AIWorkbench.PlanningTextPolicies = c.ProductAgent.ContentTextPolicies },
		"full chat forbidden":               func(c *Config) { c.AIWorkbench.ConversationOnly = false },
		"chat owner unchanged":              func(c *Config) { c.AIWorkbench.Database.Database = "another-owner" },
		"chat role remains separate":        func(c *Config) { c.AIWorkbench.Database.User = c.ProductAgent.Database.User },
	} {
		t.Run(name, func(t *testing.T) {
			c := contentTitleRuntimeConfig(t)
			change(c)
			require.Error(t, c.validate())
		})
	}
}

func TestContentTitleOnlyStartupDoesNotOpenAssetPool(t *testing.T) {
	cfg := contentTitleRuntimeConfig(t)
	// Stop at the next owner open, after ProductAgent pool selection and before
	// unrelated storage-capacity/schema construction on these fake pools.
	pools := []*gorm.DB{{}, {}, {}, {}, {}}
	opened, closed := 0, 0
	stop := errors.New("fixture construction stop")
	called := false
	deps := Dependencies{
		IdentityPreflight:      func(context.Context, IdentityConfig) error { return nil },
		OpenSourceAccount:      func(context.Context, DatabaseConfig) (*gorm.DB, error) { return pools[0], nil },
		OpenCommercialOwner:    func(context.Context, DatabaseConfig) (*gorm.DB, error) { return pools[1], nil },
		OpenProductAcquisition: func(context.Context, DatabaseConfig) (*gorm.DB, error) { return pools[2], nil },
		OpenProductAgent: func(_ context.Context, db DatabaseConfig) (*gorm.DB, error) {
			require.NotEmpty(t, db.Database)
			opened++
			require.LessOrEqual(t, opened, 2, "content-only must not open an unused Asset owner")
			return pools[2+opened], nil
		},
		NewApplicationWithFeatures: func(_ context.Context, _ *gorm.DB, f ApplicationFeatures, _ *coreconfig.Config, _ *logrus.Logger) (*http.Server, error) {
			return nil, stop
		},
		OpenAIWorkbench: func(context.Context, DatabaseConfig) (*gorm.DB, error) {
			called = true
			require.Equal(t, 2, opened)
			return nil, stop
		},
		CloseDatabase: func(*gorm.DB) error { closed++; return nil },
	}
	require.Error(t, Run(context.Background(), cfg, logrus.New(), deps))
	require.True(t, called)
	require.Equal(t, 2, opened)
	require.Equal(t, 5, closed)
}

func TestContentTitleOnlyWrongRolesStopBeforeOpeningPools(t *testing.T) {
	t.Run("disabled management keeps its existing contract", func(t *testing.T) {
		cfg := contentTitleRuntimeConfig(t)
		cfg.ProductAgent.Enabled = false
		cfg.ProductAgent.Database.User = "retained_management_runtime"
		require.NoError(t, cfg.validate())
	})
	for _, target := range []string{"run", "review"} {
		t.Run(target, func(t *testing.T) {
			cfg := contentTitleRuntimeConfig(t)
			if target == "run" {
				cfg.ProductAgent.Database.User = "product_agent_owner"
			} else {
				cfg.ProductAgent.ReviewDatabase.User = "source_acquisition_runtime"
			}
			opened := 0
			open := func(context.Context, DatabaseConfig) (*gorm.DB, error) {
				opened++
				return nil, errors.New("unexpected open")
			}
			err := Run(context.Background(), cfg, logrus.New(), Dependencies{OpenSourceAccount: open, OpenProductAgent: open, OpenProductAcquisition: open, IdentityPreflight: func(context.Context, IdentityConfig) error { t.Fatal("identity preflight must not begin"); return nil }})
			require.ErrorContains(t, err, "prepared Run and Review runtime identities")
			require.Zero(t, opened)
		})
	}
}

func TestContentTitleOnlyDoesNotTakeOverOtherAssetConsumers(t *testing.T) {
	for name, factory := range map[string]func(*testing.T) *Config{"image": fullImageGenericRuntimeConfig, "supply": supplyRuntimeConfig} {
		t.Run(name, func(t *testing.T) {
			cfg := factory(t)
			require.NoError(t, cfg.validate())
			content := contentTitleRuntimeConfig(t).ProductAgent
			if cfg.ProductAgent != nil {
				content.Database = cfg.ProductAgent.Database
			}
			content.Database.User = "product_agent_runtime"
			content.ReviewDatabase = *cfg.ProductAcquisitionDatabase
			content.ReviewDatabase.User = "product_review_runtime"
			cfg.ProductAgent = content
			require.NoError(t, cfg.validate(), "content consumes no Asset pool and must not replace another module's Asset owner")
		})
	}
}
