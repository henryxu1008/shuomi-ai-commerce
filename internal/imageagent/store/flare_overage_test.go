package store

import (
	"bytes"
	"context"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"os"
	"strings"
	"sync"
	"task-processor/internal/aicapability"
	credential "task-processor/internal/aicapability/channelcredential"
	"task-processor/internal/imageagent"
	"testing"
	"time"
)

func TestFlareOverageStopsAnotherRunButPreservesOriginalKey(t *testing.T) {
	db := generationTestDatabase(t)
	assertFlareOverageBinding(t, db)
}

func TestFlareOverageDispatchSerializationPostgreSQL(t *testing.T) {
	if os.Getenv("CI") == "" {
		t.Skip("requires task-owned CI PostgreSQL; no retained database access")
	}
	if os.Getenv("ISSUE487_TEST_DSN") != "" && os.Getenv("ISSUE487_EXCLUSIVE_POSTGRES") != "ISOLATED_TRIAL_ONLY" {
		t.Skip("external PostgreSQL has no isolated-trial authorization")
	}
	dsn := imageRuntimePermissionTestDSN(t)
	root, err := gorm.Open(postgres.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	require.NoError(t, err)
	sqlRoot, err := root.DB()
	require.NoError(t, err)
	defer sqlRoot.Close()
	name := "flare_reference_" + strings.ReplaceAll(uuid.NewString(), "-", "")
	require.NoError(t, root.Exec(`CREATE DATABASE "`+name+`"`).Error)
	defer func() { require.NoError(t, root.Exec(`DROP DATABASE "`+name+`" WITH (FORCE)`).Error) }()
	db, err := gorm.Open(postgres.Open(dsn+" dbname="+name), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	require.NoError(t, err)
	sqlDB, err := db.DB()
	require.NoError(t, err)
	defer sqlDB.Close()
	require.NoError(t, AutoMigrate(db))
	assertFlareOverageBinding(t, db)
}

func assertFlareOverageBinding(t *testing.T, db *gorm.DB) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	repo := NewOrganizationRepository(db).(*gormRepository)
	makeIntent := func(suffix, config string) imageagent.GenerationIntent {
		res := v3Reservation("flare-" + suffix)
		initializeSlotEffectRun(t, repo, res.Identity.RunID)
		require.NoError(t, db.Model(&runRecord{}).Where("id = ?", res.Identity.RunID).Updates(map[string]any{"scope_protocol": imageagent.OrganizationScopeProtocol, "member_id": "grant-1"}).Error)
		_, won, err := repo.ReserveSlotProviderV3(ctx, res)
		require.NoError(t, err)
		require.True(t, won)
		cat, err := repo.GetAssetCatalog(ctx, res.Identity.RunScope)
		require.NoError(t, err)
		b := flareStoreBilling()
		in := imageagent.GenerationIntent{Billing: b, Identity: res.Identity, MemberID: "grant-1", CatalogHash: cat.Manifest.Hash, SourceDigest: strings.Repeat("b", 64), PromptVersion: "white-v1", RouteReference: "route-1", CredentialReference: (credential.Metadata{OrganizationID: res.Identity.TenantID, ConnectionID: "shared-openai", Provider: "openai", Channel: "openai-direct", Account: "fixture", Region: "global", BaseURL: "https://api.openai.com/v1", Revision: 1}).WithIdentity().Reference, ConfigurationVersion: config, Provider: "openai", Model: aicapability.FlareImageModel, Protocol: aicapability.FlareImageProtocol, Resolution: "1024x1024", Quality: "auto", PriceVersion: "reference-v1", Points: 300, LimitVersion: 1, MonthStart: time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)}
		fact, err := repo.PrepareGenerationIntent(ctx, in)
		require.NoError(t, err)
		_, err = repo.BindGenerationReservation(ctx, in, imageagent.GenerationReservationReceipt{UnitVersion: aicapability.AIPointMicrounitV1, IntentID: fact.IntentID, Fingerprint: fact.Fingerprint, OrganizationID: in.Identity.TenantID, MemberID: in.MemberID, OperationID: "image-reserve:" + fact.IntentID, ReservationID: "reserve-" + suffix, ResourceType: "ai_point", Points: in.Points, PriceVersion: in.PriceVersion, LimitVersion: 1, MonthStart: in.MonthStart})
		require.NoError(t, err)
		return in
	}
	original := makeIntent("original", "config-1")
	other := makeIntent("new", "config-1")
	unrelated := makeIntent("different", "config-2")
	_, won, err := repo.BeginGenerationDispatch(ctx, original)
	require.NoError(t, err)
	require.True(t, won)
	proof := imageagent.GenerationSuccess{ResponseID: "response-1", ResultDigest: strings.Repeat("a", 64), ResultUnavailable: "invalid_result", UsageKnown: true, InputTokens: 101, OutputTokens: 100, TotalTokens: 201, BillableUsage: aicapability.BillableModelUsage{MappingVersion: aicapability.ReferenceImagesTotalsV1, Complete: true, OutputImage: 100, ReferenceImageTotals: aicapability.ReferenceImageTotals{Known: true, Input: 101, Output: 100, Total: 201}}}
	fact, err := repo.RecordGenerationSuccess(ctx, original, proof)
	require.NoError(t, err)
	require.Equal(t, imageagent.GenerationUnknown, fact.State)
	_, won, err = repo.BeginGenerationDispatch(ctx, other)
	require.ErrorIs(t, err, imageagent.ErrReferenceImageBindingStopped)
	require.False(t, won)
	_, won, err = repo.BeginGenerationDispatch(ctx, unrelated)
	require.NoError(t, err)
	require.True(t, won)
	reopened := NewOrganizationRepository(db).(*gormRepository)
	got, err := reopened.ReadGenerationFact(ctx, original.Identity)
	require.NoError(t, err)
	require.Equal(t, fact, got)
	_, won, err = reopened.BeginGenerationDispatch(ctx, original)
	require.NoError(t, err)
	require.False(t, won)
	replay, err := reopened.RecordGenerationSuccess(ctx, original, proof)
	require.NoError(t, err)
	require.Equal(t, fact, replay)
	if db.Dialector.Name() != "postgres" {
		return
	}
	// A different run must see the first observation after acquiring the
	// same-binding transaction lock, even if it started before that commit.
	raceOwner, raceNext := makeIntent("race-owner", "config-race"), makeIntent("race-next", "config-race")
	_, won, err = repo.BeginGenerationDispatch(ctx, raceOwner)
	require.NoError(t, err)
	require.True(t, won)
	paused, release := make(chan struct{}), make(chan struct{})
	var entered, releaseOnce sync.Once
	defer releaseOnce.Do(func() { close(release) })
	require.NoError(t, db.Callback().Update().Before("gorm:update").Register("hold_flare_overage", func(tx *gorm.DB) {
		values, ok := tx.Statement.Dest.(map[string]any)
		if !ok {
			return
		}
		raw, _ := values["generation_fact_json"].([]byte)
		if bytes.Contains(raw, []byte(`"ObservedOverage"`)) {
			entered.Do(func() { close(paused); <-release })
		}
	}))
	defer db.Callback().Update().Remove("hold_flare_overage")
	observed := make(chan error, 1)
	go func() { _, e := repo.RecordGenerationSuccess(ctx, raceOwner, proof); observed <- e }()
	select {
	case <-paused:
	case <-time.After(5 * time.Second):
		t.Fatal("observation did not reach controlled persistence")
	}
	dispatched := make(chan error, 1)
	go func() {
		_, won, e := repo.BeginGenerationDispatch(ctx, raceNext)
		if e == nil && won {
			e = imageagent.ErrProviderContractViolation
		}
		dispatched <- e
	}()
	select {
	case e := <-dispatched:
		t.Fatalf("dispatch crossed an uncommitted same-binding observation: %v", e)
	case <-time.After(100 * time.Millisecond):
	}
	releaseOnce.Do(func() { close(release) })
	require.NoError(t, <-observed)
	require.ErrorIs(t, <-dispatched, imageagent.ErrReferenceImageBindingStopped)
}

func flareStoreBilling() imageagent.ImageBilling {
	return aicapability.OfficialModelBilling{OfficialTariff: aicapability.OfficialModelTariff{UnitVersion: aicapability.AIPointMicrounitV1, PriceVersion: "reference-v1", FXVersion: "identity-v1", Currency: "CNY", CNYMicrosPerCurrencyUnit: 1000000, Model: aicapability.FlareImageModel, Channel: "openai-direct", UsageVersion: aicapability.ReferenceImagesTotalsV1, Rates: aicapability.OfficialModelRates{InputText: 10000, InputImage: 20000, OutputImage: 10000}}, ReferenceEstimate: aicapability.ReferenceImageEstimate{InputTokens: 100, OutputTokens: 100, OperationScope: aicapability.ReferenceImageOperationScope, Size: "1024x1024", Quality: "auto", AllowAutoSize: true, MaximumPromptBytes: 65536, MaximumReferenceImages: 3, MaximumReferenceBytes: 16 << 20, MaximumReferencePixels: 64000000}}
}
