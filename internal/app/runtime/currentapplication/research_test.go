package currentapplication

import (
	"context"
	"errors"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/client"
	"gorm.io/gorm"
	"net/http"
	core "task-processor/internal/core/config"
	"testing"
)

func TestResearchRequiresProjectsDataServicesAndItsRestrictedProductPool(t *testing.T) {
	cfg := runtimeTestConfig()
	product := DatabaseConfig{Host: "127.0.0.1", Port: 15432, User: "product_research_runtime", Password: "fixture-only", Database: "product", MaxConnections: 4}
	cfg.MarketResearch = &ResearchConfig{Database: product}
	require.Error(t, cfg.validateResearch())
	cfg.ProductCollections = true
	cfg.ProductAcquisitionDatabase = &product
	cfg.DataServices = &DataServicesConfig{}
	cfg.ProjectCenter = &ProjectCenterConfig{Database: DatabaseConfig{Host: "127.0.0.1", Port: 15432, Database: "projects"}}
	require.NoError(t, cfg.validateResearch())
	cfg.MarketResearch.Database.User = "data_services_runtime"
	require.Error(t, cfg.validateResearch())
	cfg.MarketResearch.Database = product
	cfg.MarketResearch.Database.Database = "other"
	require.Error(t, cfg.validateResearch())
}

func TestResearchPoolLifecycleConsumesItsOwnPoolAndClosesOnFailure(t *testing.T) {
	for _, stage := range []string{"open", "construct", "alias"} {
		t.Run(stage, func(t *testing.T) {
			cfg := dataServicesRuntimeConfig(t)
			cfg.ProjectCenter = &ProjectCenterConfig{Database: DatabaseConfig{Host: "127.0.0.1", Port: 15432, User: "ai_projects_runtime", Password: "fixture-only", Database: "projects", MaxConnections: 4}}
			d := *cfg.ProductAcquisitionDatabase
			d.User = "product_research_runtime"
			cfg.MarketResearch = &ResearchConfig{Database: d}
			source, product, project, data, researchDB := &gorm.DB{}, &gorm.DB{}, &gorm.DB{}, &gorm.DB{}, &gorm.DB{}
			resource := emptyStorageCommercialOwner(t)
			var closed []*gorm.DB
			stop := errors.New("bounded constructor stop")
			constructed := false
			deps := Dependencies{IdentityPreflight: func(context.Context, IdentityConfig) error { return nil }, OpenSourceAccount: func(context.Context, DatabaseConfig) (*gorm.DB, error) { return source, nil }, OpenCommercialOwner: func(context.Context, DatabaseConfig) (*gorm.DB, error) { return resource, nil }, OpenProductAcquisition: func(context.Context, DatabaseConfig) (*gorm.DB, error) { return product, nil }, OpenProjectCenter: func(context.Context, DatabaseConfig) (*gorm.DB, error) { return project, nil }, OpenDataServices: func(context.Context, DatabaseConfig) (*gorm.DB, error) { return data, nil }, DialDataServicesWorkflow: func(context.Context, string, string) (client.Client, func() error, error) {
				return &dataClientFixture{}, func() error { return nil }, nil
			}, OpenMarketResearch: func(context.Context, DatabaseConfig) (*gorm.DB, error) {
				if stage == "alias" {
					return product, nil
				}
				if stage == "open" {
					return researchDB, stop
				}
				return researchDB, nil
			}, CloseDatabase: func(db *gorm.DB) error { closed = append(closed, db); return nil }, NewApplicationWithFeatures: func(_ context.Context, _ *gorm.DB, f ApplicationFeatures, _ *core.Config, _ *logrus.Logger) (*http.Server, error) {
				constructed = true
				require.Same(t, researchDB, f.MarketResearchDB)
				require.Same(t, project, f.ProjectCenterDB)
				require.Nil(t, f.ProductAgent)
				return nil, stop
			}}
			require.Error(t, Run(context.Background(), cfg, logrus.New(), deps))
			require.Equal(t, stage == "construct", constructed)
			want := []*gorm.DB{data, project, product, resource, source}
			if stage != "alias" {
				want = append([]*gorm.DB{researchDB}, want...)
			}
			require.Equal(t, want, closed)
		})
	}
}
