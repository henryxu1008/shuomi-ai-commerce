package objectstore

import (
	"context"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"strconv"
	"strings"
	"task-processor/internal/imageagent"
	capacitystore "task-processor/internal/integration/persistence/storagecapacity"
	"task-processor/internal/ledger/storagecapacity"
	"testing"
	"time"
)

type capacityGenerationReader struct{ fact imageagent.GenerationFact }

func (r capacityGenerationReader) ReadGenerationFactForStorage(context.Context, string, string, string, int64, string, int64) (imageagent.GenerationFact, error) {
	return r.fact, nil
}
func TestArtifactPublicationRequiresOriginalReservationAndCountsCopyOnce(t *testing.T) {
	ctx := context.Background()
	api := &fakeObjectStore{objects: map[string]fakeObject{}}
	artifacts := newTestStore(t, api)
	prepared := mustPrepare(t, artifacts)
	require.NoError(t, artifacts.EnsureStaged(ctx, prepared))
	id := testIdentity()
	intent := imageagent.GenerationIntent{Identity: id, MemberID: "member", CatalogHash: "catalog-v1:" + strings.Repeat("a", 64), SourceDigest: strings.Repeat("b", 64), PromptVersion: "prompt", RouteReference: "route", CredentialReference: "credential", ConfigurationVersion: "configuration", Provider: "grsai", Model: "gpt-image-2.5", Protocol: "grsai-json-sync-v1", Resolution: "1024x1024", Quality: "auto", PriceVersion: "price", Points: 12, LimitVersion: 1, MonthStart: time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)}
	f, e := imageagent.NewGenerationFact(intent)
	require.NoError(t, e)
	f, e = f.BindReservation(imageagent.GenerationReservationReceipt{IntentID: f.IntentID, Fingerprint: f.Fingerprint, OrganizationID: id.TenantID, MemberID: "member", OperationID: "image-reserve:" + f.IntentID, ReservationID: "points-reservation", ResourceType: "ai_point", Points: 12, PriceVersion: "price", LimitVersion: 1, MonthStart: intent.MonthStart})
	require.NoError(t, e)
	f, _, e = f.BeginDispatch()
	require.NoError(t, e)
	f, e = f.RecordSuccess(imageagent.GenerationSuccess{ResponseID: "response", ResultDigest: strings.Repeat("c", 64), ResultURL: "https://cdn.example.org/output.png"})
	require.NoError(t, e)
	f, e = f.BindSettlement(imageagent.GenerationSettlementReceipt{IntentID: f.IntentID, Fingerprint: f.Fingerprint, OperationID: "image-finalize:" + f.IntentID, ReservationID: "points-reservation", State: "committed", Points: 12, ProofDigest: f.TerminalProofDigest()})
	require.NoError(t, e)
	db, e := gorm.Open(sqlite.Open("file:"+uuid.NewString()+"?mode=memory&cache=shared"), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	require.NoError(t, e)
	pool, _ := db.DB()
	t.Cleanup(func() { _ = pool.Close() })
	require.NoError(t, capacitystore.AutoMigrate(db))
	capacity, e := capacitystore.New(db)
	require.NoError(t, e)
	artifacts.ConfigureStorageCapacity(capacity, "image-physical", capacityGenerationReader{f})
	_, e = artifacts.Finalize(ctx, prepared.Manifest)
	require.ErrorIs(t, e, storagecapacity.ErrNotFound)
	require.Zero(t, api.copyCalls)
	r, e := capacity.ReserveWrite(ctx, storagecapacity.WriteIntent{OrganizationID: id.TenantID, Owner: "image_agent", OperationID: imageagent.StorageGenerationOperation(id), Fingerprint: f.Fingerprint, MaxBytes: 1024, MaxObjects: 1})
	require.NoError(t, e)
	require.NoError(t, capacity.ClaimDispatch(ctx, id.TenantID, r.ID))
	first, e := artifacts.Finalize(ctx, prepared.Manifest)
	require.NoError(t, e)
	replay, e := artifacts.Finalize(ctx, prepared.Manifest)
	require.NoError(t, e)
	require.Equal(t, first, replay)
	require.Equal(t, 1, api.copyCalls)
	s, e := capacity.ReadSummary(ctx, id.TenantID)
	require.NoError(t, e)
	require.Equal(t, strconv.FormatInt(prepared.Manifest.Assets[0].SizeBytes, 10), s.OccupiedBytes)
	require.Equal(t, "0", s.ReservedBytes)
}
