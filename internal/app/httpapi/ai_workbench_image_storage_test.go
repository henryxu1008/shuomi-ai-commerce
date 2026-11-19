package httpapi

import (
	"bytes"
	"context"
	"errors"
	"image"
	"image/png"
	"strconv"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"task-processor/internal/aiworkbench"
	storagecapacitystore "task-processor/internal/integration/persistence/storagecapacity"
	"task-processor/internal/integration/s3"
	"task-processor/internal/ledger/storagecapacity"
)

type interruptedChatImageObjects struct {
	objects     map[string]s3.ImmutableObjectPut
	writes      map[string]int
	interrupted string
}

func (s *interruptedChatImageObjects) PutImmutable(_ context.Context, put s3.ImmutableObjectPut) error {
	s.writes[put.Key]++
	if put.Key == s.interrupted && s.writes[put.Key] == 1 {
		return errors.New("interrupted before storage")
	}
	s.objects[put.Key] = put
	return nil
}

func (s *interruptedChatImageObjects) ReadObject(_ context.Context, key string, limit int64) ([]byte, s3.ObjectInspection, error) {
	put, ok := s.objects[key]
	if put.SizeBytes > limit {
		return nil, s3.ObjectInspection{}, errors.New("object exceeds limit")
	}
	return put.Data, s3.ObjectInspection{Exists: ok, ContentLength: put.SizeBytes, ContentType: put.ContentType}, nil
}

func chatImageStorageCapacityFixture(t *testing.T) *storagecapacitystore.Repository {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:"+uuid.NewString()+"?mode=memory&cache=shared"), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	require.NoError(t, err)
	pool, err := db.DB()
	require.NoError(t, err)
	t.Cleanup(func() { _ = pool.Close() })
	require.NoError(t, storagecapacitystore.AutoMigrate(db))
	capacity, err := storagecapacitystore.New(db)
	require.NoError(t, err)
	return capacity
}

func TestChatImageStorageRetriesOnlyMissingImageAndCountsEachOnce(t *testing.T) {
	ctx := context.Background()
	capacity := chatImageStorageCapacityFixture(t)
	group := make([][]byte, 2)
	for n := range group {
		var raw bytes.Buffer
		require.NoError(t, png.Encode(&raw, image.NewRGBA(image.Rect(0, 0, n+1, 1))))
		group[n] = raw.Bytes()
	}
	pictures, err := inspectChatImages(group)
	require.NoError(t, err)
	operation := storagecapacity.Digest("original operation")
	first := aiworkbench.ChatImageObject(operation, 0)
	second := aiworkbench.ChatImageObject(operation, 1)
	objects := &interruptedChatImageObjects{objects: make(map[string]s3.ImmutableObjectPut), writes: make(map[string]int), interrupted: second}
	store := &ChatImageStorage{Objects: objects, Capacity: capacity, Namespace: "private-test"}
	require.ErrorIs(t, store.store(ctx, "org", operation, "compare", pictures, group), errChatImageStorageUnknown)
	summary, err := capacity.ReadSummary(ctx, "org")
	require.NoError(t, err)
	require.Equal(t, strconv.Itoa(len(group[0])), summary.OccupiedBytes)
	require.Equal(t, strconv.Itoa(len(group[1])), summary.ReservedBytes)

	require.ErrorIs(t, store.store(ctx, "org", operation, "changed question", pictures, group), storagecapacity.ErrConflict)
	require.ErrorIs(t, store.store(ctx, "org", operation, "compare", pictures, [][]byte{group[1], group[0]}), aiworkbench.ErrInvalid)
	require.NoError(t, store.store(ctx, "org", operation, "compare", pictures, group))
	require.NoError(t, store.store(ctx, "org", operation, "compare", pictures, group))
	require.Equal(t, map[string]int{first: 1, second: 2}, objects.writes)
	summary, err = capacity.ReadSummary(ctx, "org")
	require.NoError(t, err)
	require.Equal(t, strconv.Itoa(len(group[0])+len(group[1])), summary.OccupiedBytes)
	require.Equal(t, "0", summary.ReservedBytes)
	for n, picture := range pictures {
		body, found, err := store.read(ctx, operation, n, picture)
		require.NoError(t, err)
		require.True(t, found)
		require.Equal(t, group[n], body)
	}
}

func TestChatImageStorageLimitRejectionPermanentlyFencesOriginalRequest(t *testing.T) {
	ctx := context.Background()
	capacity := chatImageStorageCapacityFixture(t)
	filler, err := capacity.ReserveWrite(ctx, storagecapacity.WriteIntent{
		OrganizationID: "org", Owner: "product_media", OperationID: "filler",
		Fingerprint: storagecapacity.Digest("full"), MaxBytes: storagecapacity.BaseBytes, MaxObjects: 1,
	})
	require.NoError(t, err)

	var raw bytes.Buffer
	require.NoError(t, png.Encode(&raw, image.NewRGBA(image.Rect(0, 0, 1, 1))))
	group := [][]byte{raw.Bytes()}
	pictures, err := inspectChatImages(group)
	require.NoError(t, err)
	objects := &interruptedChatImageObjects{objects: make(map[string]s3.ImmutableObjectPut), writes: make(map[string]int)}
	store := &ChatImageStorage{Objects: objects, Capacity: capacity, Namespace: "private-test"}

	rejected := storagecapacity.Digest("original-limit")
	err = store.store(ctx, "org", rejected, "compare", pictures, group)
	var proven *aiworkbench.MessageNotAcceptedError
	require.ErrorAs(t, err, &proven)
	require.ErrorIs(t, proven.Cause, storagecapacity.ErrExceeded)
	require.Empty(t, objects.writes)
	require.NoError(t, capacity.AbortUnstarted(ctx, "org", filler.ID))

	// Free space may now be available, but a delayed retry of the already
	// rejected request cannot claim capacity, store files or reach the model.
	err = store.store(ctx, "org", rejected, "compare", pictures, group)
	require.ErrorAs(t, err, &proven)
	require.ErrorIs(t, proven.Cause, storagecapacity.ErrExceeded)
	require.ErrorIs(t, store.store(ctx, "org", rejected, "changed question", pictures, group), storagecapacity.ErrConflict)
	require.Empty(t, objects.writes)
	summary, err := capacity.ReadSummary(ctx, "org")
	require.NoError(t, err)
	require.Equal(t, "0", summary.OccupiedBytes)
	require.Equal(t, "0", summary.ReservedBytes)

	next := storagecapacity.Digest("new-question")
	require.NoError(t, store.store(ctx, "org", next, "compare", pictures, group))
	require.Equal(t, map[string]int{aiworkbench.ChatImageObject(next, 0): 1}, objects.writes)
}

func TestChatImageBudgetRejectionOnlyFencesUnacceptedWrite(t *testing.T) {
	ctx := context.Background()
	capacity := chatImageStorageCapacityFixture(t)
	var raw bytes.Buffer
	require.NoError(t, png.Encode(&raw, image.NewRGBA(image.Rect(0, 0, 1, 1))))
	group := [][]byte{raw.Bytes()}
	pictures, err := inspectChatImages(group)
	require.NoError(t, err)
	objects := &interruptedChatImageObjects{objects: make(map[string]s3.ImmutableObjectPut), writes: make(map[string]int)}
	store := &ChatImageStorage{Objects: objects, Capacity: capacity, Namespace: "private-test"}
	operation := storagecapacity.Digest("budget-rejection")

	var proof *aiworkbench.MessageNotAcceptedError
	err = store.rejectBudget(ctx, "org", operation, "compare", pictures)
	require.ErrorAs(t, err, &proof)
	require.ErrorIs(t, proof.Cause, storagecapacity.ErrRejected)
	// Changing the model budget later cannot turn the old rejection into a write.
	err = store.store(ctx, "org", operation, "compare", pictures, group)
	require.ErrorAs(t, err, &proof)
	require.ErrorIs(t, proof.Cause, storagecapacity.ErrRejected)
	require.Empty(t, objects.writes)

	next := storagecapacity.Digest("admitted-before-budget-changed")
	require.NoError(t, store.store(ctx, "org", next, "compare", pictures, group))
	require.ErrorIs(t, store.rejectBudget(ctx, "org", next, "compare", pictures), errChatImageStorageUnknown)
	require.NoError(t, store.store(ctx, "org", next, "compare", pictures, group))
	require.Equal(t, map[string]int{aiworkbench.ChatImageObject(next, 0): 1}, objects.writes)
}
