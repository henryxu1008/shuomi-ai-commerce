package currentapplication

import (
	"context"
	"encoding/json"
	"errors"
	"io/fs"
	"net/http"
	"testing"
	"testing/fstest"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
	core "task-processor/internal/core/config"
	"task-processor/internal/integration/acquisition/zenrows"
	"task-processor/internal/product/dataacquisition"
)

const trialTestID = "aaaaaaaa-0000-4000-8000-000000000001"

func trialRuntimeTestConfig(t *testing.T) (*Config, Dependencies) {
	t.Helper()
	cfg := dataServicesRuntimeConfig(t)
	cfg.DataServices = &DataServicesConfig{Database: cfg.DataServices.Database}
	research := *cfg.ProductAcquisitionDatabase
	research.User = "product_research_runtime"
	cfg.MarketResearch = &ResearchConfig{Database: research}
	project := research
	project.User, project.Database = "ai_projects_runtime", "projects"
	cfg.ProjectCenter = &ProjectCenterConfig{Database: project}
	open := func(context.Context, DatabaseConfig) (*gorm.DB, error) { return &gorm.DB{}, nil }
	resource := emptyStorageCommercialOwner(t)
	deps := Dependencies{
		IdentityPreflight:      func(context.Context, IdentityConfig) error { return nil },
		OpenSourceAccount:      open,
		OpenCommercialOwner:    func(context.Context, DatabaseConfig) (*gorm.DB, error) { return resource, nil },
		OpenProductAcquisition: open,
		OpenDataServices:       open,
		OpenMarketResearch:     open,
		OpenProjectCenter:      open,
		CloseDatabase:          func(*gorm.DB) error { return nil },
	}
	return cfg, deps
}

func TestSHEINTrialRuntimeDefaultAndInvalidOptIn(t *testing.T) {
	cfg, deps := trialRuntimeTestConfig(t)
	stop := errors.New("fixture reached application assembly")
	deps.NewApplicationWithFeatures = func(_ context.Context, _ *gorm.DB, f ApplicationFeatures, _ *core.Config, _ *logrus.Logger) (*http.Server, error) {
		require.Nil(t, f.SHEINUSTrial)
		require.NotNil(t, f.MarketResearchDB)
		require.False(t, f.DataServices.ExecutionEnabled)
		require.Nil(t, f.DataServicesWorkflow)
		return nil, stop
	}
	require.ErrorIs(t, Run(context.Background(), cfg, logrus.New(), deps), stop)
	deps.IdentityPreflight = func(context.Context, IdentityConfig) error {
		t.Fatal("invalid trial configuration reached identity preflight")
		return nil
	}
	for _, dir := range []string{"", "relative", "/", "/run/../inputs", "/missing-shein-trial-fixture"} {
		cfg.SheinUSTrialInputDir = &dir
		require.ErrorContains(t, Run(context.Background(), cfg, logrus.New(), deps), "SHEIN US trial")
	}
}

func trialTestFile(t *testing.T, v any) *fstest.MapFile {
	t.Helper()
	body, err := json.Marshal(v)
	require.NoError(t, err)
	return &fstest.MapFile{Data: body}
}

func trialTestFiles(t *testing.T) (fstest.MapFS, map[string]any) {
	t.Helper()
	actor := map[string]string{"organizationId": trialTestID, "actorId": trialTestID, "memberId": trialTestID}
	return fstest.MapFS{"approval.json": trialTestFile(t, map[string]any{
			"version": "shein-us-trial-approval@v1", "id": trialTestID, "projectId": trialTestID,
			"accountRef": trialTestID, "keyRef": trialTestID, "funding": "enterprise_unallocated",
			"scope": actor, "reconciler": actor, "expiresAt": time.Now().Add(time.Hour),
		})}, map[string]any{
			"version": "shein-us-trial-gate@v1", "approvalEvidenceRef": trialTestID,
			"checkedAt": time.Now().Add(-time.Second), "validUntil": time.Now().Add(time.Hour),
			"allowedActions": []string{"send", "reconcile"}, "budget": map[string]any{
				"accountEvidenceRef": trialTestID, "validUntil": time.Now().Add(time.Hour),
				"availableCredits": 25, "maximumRequestCredits": 25, "zeroIncrementalCash": true, "exclusiveKeyWindow": true,
			},
		}
}

type trialTestRefresh struct {
	fs.FS
	afterKeyOpen func()
}

func (f trialTestRefresh) Open(name string) (fs.File, error) {
	file, err := f.FS.Open(name)
	if name == "key.json" && err == nil {
		f.afterKeyOpen()
	}
	return file, err
}

func TestSHEINTrialLatestGateAfterKeyRead(t *testing.T) {
	files, gate := trialTestFiles(t)
	inputs, err := loadSHEINUSTrialFiles(files)
	require.NoError(t, err) // No gate or key exists at startup.
	approval := inputs.Approval
	gate["binding"] = approval.Binding()
	files["gate.json"] = trialTestFile(t, gate)
	files["key.json"] = trialTestFile(t, map[string]any{
		"version": "shein-us-trial-key@v1", "binding": approval.Binding(),
		"accountRef": trialTestID, "keyRef": trialTestID, "secret": "synthetic-key",
	})
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_, err = inputs.Budget(ctx, approval)
	require.NoError(t, err)
	key, err := inputs.ReadKey(ctx, trialTestID, trialTestID)
	require.NoError(t, err)
	require.Equal(t, "synthetic-key", key)
	refreshed, err := loadSHEINUSTrialFiles(trialTestRefresh{files, func() {
		delete(gate, "budget") // Same Send permission, revoked budget while reading key.
		files["gate.json"] = trialTestFile(t, gate)
	}})
	require.NoError(t, err)
	key, err = refreshed.ReadKey(ctx, trialTestID, trialTestID)
	require.Error(t, err)
	require.Empty(t, key)
	require.NoError(t, refreshed.Access(ctx, dataacquisition.Principal{}, approval, dataacquisition.SHEINUSTrialReconcile))
}

func TestSHEINTrialFileEvidenceOnlySettlesOriginal(t *testing.T) {
	files, _ := trialTestFiles(t)
	inputs, err := loadSHEINUSTrialFiles(files)
	require.NoError(t, err)
	approval := inputs.Approval
	lookup, err := zenrows.NewLookup(inputs.ReadUsage)
	require.NoError(t, err)
	ctx := context.Background()
	request := dataacquisition.SHEINUSTrialRequest{
		URL: "https://us.shein.com/Women-Dresses-c-1727.html", Mode: "auto", Extract: "auto", ProxyCountry: "us", CardLimit: 20,
	}
	attempt := dataacquisition.SHEINUSTrialAttempt{
		SHEINUSTrialUsage: dataacquisition.SHEINUSTrialUsage{
			ApprovalID: approval.ID, Binding: approval.Binding(), RequestID: "original-request",
		},
		ClaimedAt: time.Now(),
	}
	record := map[string]any{"accountRef": trialTestID, "requestId": "different-request", "final": true, "credits": 0}
	evidence := map[string]any{
		"version": "shein-us-trial-usage@v1", "binding": approval.Binding(), "evidenceRef": trialTestID,
		"checkedAt": time.Now(), "records": []any{record},
	}
	files["usage.json"] = trialTestFile(t, evidence)
	_, err = lookup(ctx, approval, request, attempt)
	require.ErrorIs(t, err, dataacquisition.ErrUnknown)
	record["requestId"] = attempt.RequestID
	record["credits"] = nil
	files["usage.json"] = trialTestFile(t, evidence)
	_, err = lookup(ctx, approval, request, attempt)
	require.ErrorIs(t, err, dataacquisition.ErrUnknown)
	record["credits"] = 0
	files["usage.json"] = trialTestFile(t, evidence)
	usage, err := lookup(ctx, approval, request, attempt)
	require.NoError(t, err)
	require.Equal(t, attempt.RequestID, usage.RequestID)
	require.Equal(t, 0, *usage.Credits)
	attempt.RequestID = ""
	_, err = lookup(ctx, approval, request, attempt)
	require.ErrorIs(t, err, dataacquisition.ErrUnknown)
}

func TestSHEINTrialStrictPrivateFile(t *testing.T) {
	for _, body := range []string{
		`{"budget":{"availableCredits":25,"availableCredits":24}}`,
		`{"budget":{"availableCredits":25,"AvailableCredits":24}}`,
		`{"budget":{"availableCredits":25,"unknown":true}}`,
		`{} {}`,
		`null`,
	} {
		files := fstest.MapFS{"gate.json": {Data: []byte(body)}}
		_, err := readTrialFile[trialGateFile](files, "gate.json", 8192)
		require.Error(t, err, body)
	}
}
