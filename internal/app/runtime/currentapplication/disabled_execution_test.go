package currentapplication

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"task-processor/internal/agentconfig"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
	coreconfig "task-processor/internal/core/config"
)

func TestDisabledExecutionKeepsModelManagementAndOriginalValidation(t *testing.T) {
	c := agentRuntimeConfig()
	c.ProductAgent.Enabled = false
	writer := c.ProductAgent.Database
	writer.User, writer.MaxConnections = "model_route_config_writer", 2
	c.ModelRoutePricing = &ModelRoutePricingConfig{Writer: &writer}
	require.NoError(t, c.validate())
	defs, err := c.modelRouteDefinitions()
	require.NoError(t, err)
	require.Len(t, defs, 1)
	policy := c.ProductAgent.TextPolicies["org"]
	policy.Profile.MaximumPromptTokens = c.ProductAgent.Tokens + 1
	c.ProductAgent.TextPolicies["org"] = policy
	require.Error(t, c.validate(), "closing execution must not admit an invalid declared policy")
	c.ProductAgent.TextPolicies = nil
	require.NoError(t, c.validate(), "management can start without inventing a text consumer")
	defs, err = c.modelRouteDefinitions()
	require.NoError(t, err)
	require.Empty(t, defs)
	writer.Database = "another-owner"
	require.Error(t, c.validate(), "management retains the original owner boundary")
}

func TestDisabledFullImageKeepsReadPoolsWithoutWorkerDependencies(t *testing.T) {
	cfg := fullImageGenericRuntimeConfig(t)
	cfg.ImageAgent.Generation.Enabled = false
	require.NoError(t, cfg.validate())
	source, product, configuration, points, images, assets := &gorm.DB{}, &gorm.DB{}, &gorm.DB{}, emptyStorageCommercialOwner(t), &gorm.DB{}, &gorm.DB{}
	stop := errors.New("disabled full image application construction boundary")
	closed := map[*gorm.DB]int{}
	err := Run(context.Background(), cfg, logrus.New(), Dependencies{
		IdentityPreflight:      func(context.Context, IdentityConfig) error { return nil },
		OpenSourceAccount:      func(context.Context, DatabaseConfig) (*gorm.DB, error) { return source, nil },
		OpenProductAcquisition: func(context.Context, DatabaseConfig) (*gorm.DB, error) { return product, nil },
		OpenProductAgent:       func(context.Context, DatabaseConfig) (*gorm.DB, error) { return configuration, nil },
		OpenCommercialOwner:    func(context.Context, DatabaseConfig) (*gorm.DB, error) { return points, nil },
		OpenImageAgent:         func(context.Context, DatabaseConfig) (*gorm.DB, error) { return images, nil },
		OpenSupplyAssets:       func(context.Context, DatabaseConfig) (*gorm.DB, error) { return assets, nil },
		// No worker config loader or Temporal dependency is supplied. The referenced
		// private-worker.yaml does not exist and must never be read in this mode.
		NewApplicationWithFeatures: func(_ context.Context, _ *gorm.DB, f ApplicationFeatures, _ *coreconfig.Config, _ *logrus.Logger) (*http.Server, error) {
			require.True(t, f.FullImageSet, "closed full mode must never fall back to the single-image consumer")
			require.Same(t, images, f.ImageAgentDB)
			require.Same(t, assets, f.ImageSetAssetDB)
			require.Same(t, configuration, f.ProductAgentDB)
			require.Nil(t, f.ImageSetWorkerDB)
			require.Nil(t, f.ImageSetWorkerConfig)
			require.Nil(t, f.ImageSetTemporal)
			require.Nil(t, f.ImageAgentWorkflow)
			return nil, stop
		},
		CloseDatabase: func(db *gorm.DB) error { closed[db]++; return nil },
	})
	require.ErrorIs(t, err, stop)
	for _, db := range []*gorm.DB{source, product, configuration, points, images, assets} {
		require.Equal(t, 1, closed[db])
	}
	cfg.ImageAgent.Generation.Enabled = true
	require.Error(t, Run(context.Background(), cfg, logrus.New(), Dependencies{}), "enabled execution still requires its dependencies")
}

func TestDisabledManagementRetainsContentOnlyConsumerBoundary(t *testing.T) {
	for _, titleOnly := range []bool{false, true} {
		c := agentRuntimeConfig()
		c.ProductAgent.Enabled = false
		c.ProductAgent.ContentOnly, c.ProductAgent.ContentTitleOnly = !titleOnly, titleOnly
		writer := c.ProductAgent.Database
		writer.User, writer.MaxConnections = "model_route_config_writer", 2
		c.ModelRoutePricing = &ModelRoutePricingConfig{Writer: &writer}
		require.Error(t, c.validate(), "content-only metadata must not admit a title consumer while execution is closed")
		c.ProductAgent.ContentTextPolicies, c.ProductAgent.TextPolicies = c.ProductAgent.TextPolicies, nil
		policy := c.ProductAgent.ContentTextPolicies["org"]
		policy.Profile.PromptVersion, policy.Profile.OutputSchemaVersion = agentconfig.ContentPromptVersion, agentconfig.ContentActionVersion
		c.ProductAgent.ContentTextPolicies["org"] = policy
		require.NoError(t, c.validate(), "the original content-only metadata remains legal without starting execution")
	}
}
