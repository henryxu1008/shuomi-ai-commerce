package ecoservices

import (
	"context"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	store "task-processor/internal/integration/persistence/storagecapacity"
	"task-processor/internal/ledger/storagecapacity"
	"testing"
)

type capacityFilesRepo struct {
	FileRepository
	intents int
}

func (r *capacityFilesRepo) CreateFileIntent(_ context.Context, i FileIntent) (File, error) {
	r.intents++
	return i.File, nil
}
func (r *capacityFilesRepo) ConfirmFile(_ context.Context, i FileIntent) (File, error) {
	f := i.File
	f.State = "CONFIRMED"
	return f, nil
}

type capacityFilesObjects struct {
	PrivateObjectStore
	file File
	puts int
}

func (o *capacityFilesObjects) Inspect(_ context.Context, f File) (ObjectInspection, error) {
	return ObjectInspection{Exists: o.file.ID == f.ID, ContentType: o.file.ContentType, SHA256: o.file.SHA256, SizeBytes: o.file.SizeBytes}, nil
}
func (o *capacityFilesObjects) PutImmutable(_ context.Context, f File, _ []byte) error {
	o.puts++
	o.file = f
	return nil
}
func TestRequestAttachmentsCountButQualificationEvidenceDoesNot(t *testing.T) {
	ctx := context.Background()
	db, e := gorm.Open(sqlite.Open("file:"+uuid.NewString()+"?mode=memory&cache=shared"), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	require.NoError(t, e)
	pool, _ := db.DB()
	t.Cleanup(func() { _ = pool.Close() })
	require.NoError(t, store.AutoMigrate(db))
	capacity, e := store.New(db)
	require.NoError(t, e)
	repo := &capacityFilesRepo{}
	objects := &capacityFilesObjects{}
	service, e := NewFileService(repo, objects)
	require.NoError(t, e)
	service.ConfigureStorageCapacity(&storagecapacity.Meter{Writer: capacity, Namespace: "ecoservices-physical"})
	scope := Scope{OrganizationID: "org", ActorID: "actor"}
	key := uuid.NewString()
	_, e = service.Upload(ctx, scope, key, "REQUEST", "", "brief.txt", []byte("brief"))
	require.NoError(t, e)
	_, e = service.Upload(ctx, scope, key, "REQUEST", "", "brief.txt", []byte("brief"))
	require.NoError(t, e)
	s, e := capacity.ReadSummary(ctx, "org")
	require.NoError(t, e)
	require.Equal(t, "5", s.OccupiedBytes)
	require.Equal(t, 1, objects.puts)
	_, e = capacity.ReserveWrite(ctx, storagecapacity.WriteIntent{OrganizationID: "org", Owner: "fixture", OperationID: "fill", Fingerprint: storagecapacity.Digest("fill"), MaxBytes: storagecapacity.BaseBytes - 5, MaxObjects: 1})
	require.NoError(t, e)
	before := repo.intents
	_, e = service.Upload(ctx, scope, uuid.NewString(), "REQUEST", "", "new.txt", []byte("more"))
	require.ErrorIs(t, e, storagecapacity.ErrExceeded)
	require.Equal(t, before, repo.intents)
	require.Equal(t, 1, objects.puts)
	_, e = service.Upload(ctx, scope, uuid.NewString(), "APPLICATION", "", "qualification.txt", []byte("qualification"))
	require.NoError(t, e)
	s, e = capacity.ReadSummary(ctx, "org")
	require.NoError(t, e)
	require.Equal(t, "5", s.OccupiedBytes, "mandatory qualification evidence must remain savable at capacity limit")
}
