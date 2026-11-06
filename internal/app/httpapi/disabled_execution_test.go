package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
	"task-processor/internal/agent"
	"task-processor/internal/agentconfig"
	"task-processor/internal/authz"
	"task-processor/internal/core/config"
	"task-processor/internal/imageagent"
	"task-processor/internal/ledger/orgresource"
	"task-processor/internal/storecenter"
)

func TestDisabledFullImageAssemblyDoesNotRequireExecutionWorkflow(t *testing.T) {
	cfg := currentApplicationTestConfig()
	cfg.StorageCapacity = true // this test stops before owner/schema construction
	stop := errors.New("workbench assembly reached after dependency validation")
	factories := defaultCurrentApplicationFactories(context.Background())
	factories.buildWorkbench = func(*config.Config, *logrus.Logger) (workbenchContextBuildResult, error) {
		return workbenchContextBuildResult{}, stop
	}
	factories.buildResourceCharges = func(context.Context, *gorm.DB, *gorm.DB, *gorm.DB, *authz.ListingKitAuthorizer, storecenter.RuntimeCapabilities) (*orgresource.ConsumerChargeService, error) {
		return &orgresource.ConsumerChargeService{}, nil
	}
	source, images, product, assets, configuration, resources := &gorm.DB{}, &gorm.DB{}, &gorm.DB{}, &gorm.DB{}, &gorm.DB{}, &gorm.DB{}
	options := []CurrentApplicationOption{
		WithFullImageSet(images, nil, FullImageSetDependencies{AssetDB: assets}),
		WithProductAcquisition(product), WithAgentConfiguration(configuration), WithCommercialOwnerDatabase(resources),
		WithAccountAuditUsageSources(&gorm.DB{}, &gorm.DB{}),
	}
	_, err := buildCurrentApplication(context.Background(), source, cfg, logrus.New(), factories, options...)
	require.ErrorIs(t, err, stop)
	cfg.ImageAgent.Generation.Enabled = true
	_, err = buildCurrentApplication(context.Background(), source, cfg, logrus.New(), factories, options...)
	require.Error(t, err)
	require.NotErrorIs(t, err, stop, "enabled mode must not get past missing execution dependencies")
}

func TestDisabledFullImageHTTPKeepsRecentHistoryAndRejectsExecutionCommands(t *testing.T) {
	a, ctx, items, projections := recentImageSetFixture(t, 1)
	const contextID = "d1abe8da-b381-4924-8d15-d79bdbfacf70"
	p := projections[items[0].RunID]
	p.Run.BusinessTaskID, p.Plan.Set.Source.OperationID = contextID, contextID
	projections[p.Run.ID] = p
	items[0].ContextID = contextID
	reader, err := imageagent.NewRunReader(recentImageSetRepository{projections: projections})
	require.NoError(t, err)
	a.history, a.service, a.executionDisabled = reader, nil, true
	reads := 0
	a.readSources = recentImageSetSources(func(context.Context, imageagent.ExecutionIdentity, imageagent.PrepareImageSetInput) (imageagent.ImageSetPreparation, error) {
		reads++
		return imageagent.ImageSetPreparation{Source: p.Plan.Set.Source}, nil
	})
	a.recent = recentImageSetReader(func(context.Context, imageagent.ExecutionIdentity, string, string, int) ([]imageagent.ImageSetRunSummary, string, int, error) {
		return items, "", 0, nil
	})
	m := fullImageModule{application: a, bind: func(c context.Context, _ string) (context.Context, error) { return c, nil }}
	router := gin.New()
	for _, r := range m.routes() {
		router.Handle(r.Method, r.Path, r.Handler)
	}
	base := "/api/v1/workbench/sourcing/1688/acquisitions/" + contextID + "/images"
	w := httptest.NewRecorder()
	router.ServeHTTP(w, httptest.NewRequest(http.MethodGet, base+"/runs", nil).WithContext(ctx))
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	var page struct {
		Items []fullImageRecentRun `json:"items"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &page))
	require.Len(t, page.Items, 1)
	require.Equal(t, p.Run.ID, page.Items[0].RunID)
	require.Equal(t, p.Run.Status, page.Items[0].Status)
	require.Equal(t, 1, reads)
	for _, route := range m.routes() {
		if route.Method == http.MethodGet || strings.HasSuffix(route.Path, "/requirements") {
			continue
		}
		path := route.Path
		for _, key := range []string{":operation_id", ":source_id", ":run_id", ":context_id", ":request_id"} {
			path = strings.ReplaceAll(path, key, contextID)
		}
		w = httptest.NewRecorder()
		router.ServeHTTP(w, httptest.NewRequest(route.Method, path, strings.NewReader("{}")).WithContext(ctx))
		require.Equal(t, http.StatusConflict, w.Code, "%s: %s", path, w.Body.String())
	}
	require.Equal(t, 1, reads, "commands must not even re-read the source to prepare new state")
	for _, id := range []string{agentconfig.ImageAgentID, agentconfig.ArtworkAgentID} {
		caps := configuredImageAgentCapabilities(ctx, id, func(context.Context, ...string) (agent.Scope, error) { return agent.Scope{}, nil }, a)
		require.Equal(t, "UNAVAILABLE", caps[0].Readiness, "nil quote must never be consulted in closed mode")
	}
}
