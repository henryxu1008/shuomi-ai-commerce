package httpapi

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
	"task-processor/internal/authz"
	"task-processor/internal/core/config"
	governed "task-processor/internal/integration/aicapability/einomodel"
	commercialstore "task-processor/internal/integration/persistence/commercialbilling"
	acquisitionstore "task-processor/internal/integration/persistence/product/acquisition"
	capacitystore "task-processor/internal/integration/persistence/storagecapacity"
	kernelmodule "task-processor/internal/kernel/module"
	"task-processor/internal/ledger/orgresource"
	"task-processor/internal/storecenter"
)

func testFullChatApplicationComposition(t *testing.T, f *acquisitionHTTPFixture, agent ProductAgentDependencies, deps routeAuthDependencies, chatDB *gorm.DB, policy governed.RoutePolicy, operationID string) {
	t.Helper()
	ctx := context.Background()
	require.NoError(t, commercialstore.AutoMigrate(f.owner))
	require.NoError(t, capacitystore.AutoMigrate(f.owner))
	require.NoError(t, acquisitionstore.GrantRuntimePermissions(ctx, f.owner, acquisitionstore.RuntimeCapabilities{Collections: true}))
	factories := currentApplicationFactories{
		buildWorkbench: func(*config.Config, *logrus.Logger) (workbenchContextBuildResult, error) {
			return workbenchContextBuildResult{module: currentApplicationTestModule{name: "workbench", routes: currentWorkbenchApplicationRoutes[:4]}, authDependencies: &deps}, nil
		},
		buildSourceAccount: func(_ *gorm.DB, a *authz.ListingKitAuthorizer) (kernelmodule.Module, error) {
			a.SetRolePolicyReader(acquisitionCurrentPolicies{})
			return currentApplicationTestModule{name: "source", routes: currentWorkbenchApplicationRoutes[5:]}, nil
		},
		buildCommercial: func(*gorm.DB, *gorm.DB, *authz.ListingKitAuthorizer) (kernelmodule.Module, error) {
			return currentApplicationTestModule{name: "commercial", routes: currentWorkbenchApplicationRoutes[4:5]}, nil
		},
		buildResourceCharges: func(ctx context.Context, db, _, _ *gorm.DB, _ *authz.ListingKitAuthorizer, _ storecenter.RuntimeCapabilities) (*orgresource.ConsumerChargeService, error) {
			owner, err := buildProductChargeOwner(ctx, db)
			if err != nil {
				return nil, err
			}
			resource := &sourceMaterialChargeFixture{port: testHTTPResourceCharges{db: f.db}, receipts: map[orgresource.ConsumerChargeIdentity]orgresource.ConsumerChargeReceipt{}}
			return orgresource.NewConsumerChargeService(resource, map[orgresource.ResourceConsumer]orgresource.ConsumerChargeOwner{orgresource.ConsumerProductAcquisition: owner})
		},
	}
	cfg := currentApplicationTestConfig()
	cfg.ListingKit.Zitadel.TenantDirectoryToken = "synthetic"
	app, err := buildCurrentApplication(ctx, &gorm.DB{}, cfg, logrus.New(), factories, WithProductAcquisition(f.db), WithCommercialOwnerDatabase(f.owner), WithProductCollections(), WithProductAgent(agent), WithAIWorkbench(AIWorkbenchDependencies{DB: chatDB, ContentTasks: true, AllowedOrganizationIDs: []string{"B"}, PlanningTextPolicies: map[string]governed.RoutePolicy{"B": policy}}))
	require.NoError(t, err)
	server := httptest.NewServer(app.Handler)
	defer server.Close()
	code, _, err := acquisitionHTTPRequest(server, "GET", workbenchChatBase, "admin", "B", "", "")
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, code)
	for _, kind := range []string{"product-agent", "content-agent"} {
		path := productAcquisitionBase + "/" + operationID + "/" + kind + "/runs"
		code, _, err := acquisitionHTTPRequest(server, "POST", path, "admin", "B", "", "{}")
		require.NoError(t, err)
		require.Equal(t, http.StatusBadRequest, code, path+" rejects a missing idempotency key before execution")
	}
}
