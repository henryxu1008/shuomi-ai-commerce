package storagecapacitystore

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"task-processor/internal/ledger/storagecapacity"
)

func storageFixture(t *testing.T) (*Repository, *gorm.DB) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:"+uuid.NewString()+"?mode=memory&cache=shared"), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	require.NoError(t, err)
	require.NoError(t, AutoMigrate(db))
	r, err := New(db)
	require.NoError(t, err)
	pool, _ := db.DB()
	t.Cleanup(func() { _ = pool.Close() })
	return r, db
}

func reserveInput(org, operation string, size int64) storagecapacity.WriteIntent {
	return storagecapacity.WriteIntent{OrganizationID: org, Owner: "source_media", OperationID: operation, Fingerprint: storagecapacity.Digest(operation), MaxBytes: size, MaxObjects: 1}
}

func TestStorageCapacityUnknownAndReplayCannotFreeOrDoubleCount(t *testing.T) {
	r, _ := storageFixture(t)
	ctx := context.Background()
	input := reserveInput("org-a", "upload-original", storagecapacity.BaseBytes)
	reserved, err := r.ReserveWrite(ctx, input)
	require.NoError(t, err)
	_, err = r.ReserveWrite(ctx, reserveInput("org-a", "other", 1))
	require.ErrorIs(t, err, storagecapacity.ErrExceeded)
	replay, err := r.ReserveWrite(ctx, input)
	require.NoError(t, err)
	require.Equal(t, reserved.ID, replay.ID)
	input.MaxBytes--
	_, err = r.ReserveWrite(ctx, input)
	require.ErrorIs(t, err, storagecapacity.ErrConflict)
	require.NoError(t, r.BindObjects(ctx, "org-a", reserved.ID, []storagecapacity.Object{{Namespace: "products", Key: "original", SHA256: storagecapacity.Digest("bytes"), SizeBytes: storagecapacity.BaseBytes, Category: "product_media"}}))
	require.NoError(t, r.ClaimDispatch(ctx, "org-a", reserved.ID))
	require.ErrorIs(t, r.AbortUnstarted(ctx, "org-a", reserved.ID), storagecapacity.ErrUnknown)
	summary, err := r.ReadSummary(ctx, "org-a")
	require.NoError(t, err)
	require.Equal(t, "0", summary.OccupiedBytes)
	require.Equal(t, "5368709120", summary.ReservedBytes)
	require.NoError(t, r.ConfirmStored(ctx, "org-a", reserved.ID, "products", "original", storagecapacity.Digest("bytes"), storagecapacity.BaseBytes))
	require.NoError(t, r.ConfirmStored(ctx, "org-a", reserved.ID, "products", "original", storagecapacity.Digest("bytes"), storagecapacity.BaseBytes))
	summary, err = r.ReadSummary(ctx, "org-a")
	require.NoError(t, err)
	require.Equal(t, "5368709120", summary.OccupiedBytes)
	require.Equal(t, "0", summary.ReservedBytes)
	other, err := r.ReadSummary(ctx, "org-b")
	require.NoError(t, err)
	require.Equal(t, "0", other.OccupiedBytes)
	require.ErrorIs(t, r.ConfirmStored(ctx, "org-b", reserved.ID, "products", "original", storagecapacity.Digest("bytes"), storagecapacity.BaseBytes), storagecapacity.ErrNotFound)
}

func TestStorageCapacityGenerationReservesBeforeKeyExists(t *testing.T) {
	r, _ := storageFixture(t)
	ctx := context.Background()
	input := reserveInput("org", "run-slot-attempt", 100)
	input.Owner = "image_agent"
	reservation, err := r.ReserveWrite(ctx, input)
	require.NoError(t, err)
	require.NoError(t, r.ClaimDispatch(ctx, "org", reservation.ID))
	object := storagecapacity.Object{Namespace: "image", Key: "published-after-generation", SHA256: storagecapacity.Digest("image"), SizeBytes: 80, Category: "ai_image"}
	require.NoError(t, r.BindObjects(ctx, "org", reservation.ID, []storagecapacity.Object{object}))
	require.NoError(t, r.BindObjects(ctx, "org", reservation.ID, []storagecapacity.Object{object}))
	require.NoError(t, r.ConfirmStored(ctx, "org", reservation.ID, object.Namespace, object.Key, object.SHA256, object.SizeBytes))
	s, err := r.ReadSummary(ctx, "org")
	require.NoError(t, err)
	require.Equal(t, "80", s.OccupiedBytes)
	require.Equal(t, "20", s.ReservedBytes, "unbound output budget stays reserved until the native owner closes it")
	object.SizeBytes = 101
	require.True(t, errors.Is(r.BindObjects(ctx, "org", reservation.ID, []storagecapacity.Object{object}), storagecapacity.ErrConflict))
}

func TestStorageCapacityPreparedLotCannotBeUsedAndExpiryKeepsData(t *testing.T) {
	r, _ := storageFixture(t)
	ctx := context.Background()
	now := time.Date(2026, 10, 12, 0, 0, 0, 0, time.UTC)
	r.now = func() time.Time { return now }
	lot := storagecapacity.PurchaseLot{OrganizationID: "org", OrderID: "order", ItemID: "item", Bytes: storagecapacity.PackBytes, DurationSeconds: storagecapacity.PeriodSeconds, TermsFingerprint: storagecapacity.Digest("terms")}
	require.NoError(t, r.PrepareLot(ctx, lot))
	s, err := r.ReadSummary(ctx, "org")
	require.NoError(t, err)
	require.Equal(t, "5368709120", s.QuotaBytes)
	require.NoError(t, r.ActivateLotInTransaction(ctx, r.db, lot, "committed-money-receipt"))
	s, err = r.ReadSummary(ctx, "org")
	require.NoError(t, err)
	require.Equal(t, "59055800320", s.QuotaBytes)
	now = now.Add(time.Hour)
	require.NoError(t, r.ActivateLotInTransaction(ctx, r.db, lot, "committed-money-receipt"))
	reservation, err := r.ReserveWrite(ctx, reserveInput("org", "file", 20*storagecapacity.GiB))
	require.NoError(t, err)
	object := storagecapacity.Object{Namespace: "products", Key: "file", SHA256: storagecapacity.Digest("file"), SizeBytes: 20 * storagecapacity.GiB, Category: "product_media"}
	require.NoError(t, r.BindObjects(ctx, "org", reservation.ID, []storagecapacity.Object{object}))
	require.NoError(t, r.ClaimDispatch(ctx, "org", reservation.ID))
	now = now.Add(30 * 24 * time.Hour)
	require.NoError(t, r.ConfirmStored(ctx, "org", reservation.ID, object.Namespace, object.Key, object.SHA256, object.SizeBytes), "an admitted reservation may finish after capacity expiry")
	s, err = r.ReadSummary(ctx, "org")
	require.NoError(t, err)
	require.Equal(t, "21474836480", s.OccupiedBytes)
	require.Equal(t, "16106127360", s.OverageBytes)
	_, err = r.ReserveWrite(ctx, reserveInput("org", "new-file", 1))
	require.ErrorIs(t, err, storagecapacity.ErrExceeded)
}
