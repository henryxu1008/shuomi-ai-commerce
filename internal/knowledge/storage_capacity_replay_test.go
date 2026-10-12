package knowledge

import (
	"context"
	"errors"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	store "task-processor/internal/integration/persistence/storagecapacity"
	"task-processor/internal/ledger/storagecapacity"
	"testing"
)

type replayKnowledgeRepo struct {
	Repository
	revision Revision
	lost     bool
	confirms int
}

func (r *replayKnowledgeRepo) Apply(_ context.Context, c Command) (Result, error) {
	if r.revision.ID == "" {
		r.revision = *c.Upload
		r.revision.ID = uuid.NewString()
		r.revision.OrganizationID = c.Scope.OrganizationID
		r.revision.ObjectKey = "knowledge/" + r.revision.ID + "/" + r.revision.SHA256
		r.revision.State = Admitted
	}
	if r.lost {
		r.lost = false
		return Result{}, errors.New("original apply commit ACK lost")
	}
	return Result{Source: &Source{ID: "source"}, Revision: &r.revision}, nil
}
func (r *replayKnowledgeRepo) ClaimUpload(context.Context, string, string, string) (Revision, bool, error) {
	return r.revision, true, nil
}
func (r *replayKnowledgeRepo) ConfirmObject(context.Context, Revision) error {
	r.confirms++
	return nil
}
func (r *replayKnowledgeRepo) GetSource(context.Context, string, string) (Source, error) {
	return Source{ID: "source", LatestRevision: &r.revision}, nil
}

type replayKnowledgeObjects struct {
	KnowledgeObjectStore
	object Object
}

func (o *replayKnowledgeObjects) Inspect(_ context.Context, _ Object) (Inspection, error) {
	return Inspection{Exists: o.object.Key != "", SHA256: o.object.SHA256, SizeBytes: o.object.SizeBytes}, nil
}
func (o *replayKnowledgeObjects) PutImmutable(_ context.Context, object Object) error {
	o.object = object
	return nil
}
func TestKnowledgeOriginalApplyReplayAndProcessorCannotReadmit(t *testing.T) {
	ctx := context.Background()
	db, e := gorm.Open(sqlite.Open("file:"+uuid.NewString()+"?mode=memory&cache=shared"), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	require.NoError(t, e)
	pool, _ := db.DB()
	t.Cleanup(func() { _ = pool.Close() })
	require.NoError(t, store.AutoMigrate(db))
	capacity, e := store.New(db)
	require.NoError(t, e)
	meter := &storagecapacity.Meter{Writer: capacity, Namespace: "knowledge-physical"}
	native := &replayKnowledgeRepo{lost: true}
	objects := &replayKnowledgeObjects{}
	service := &Service{repo: native, objects: objects, capacity: meter}
	command := Command{Scope: Scope{OrganizationID: "org", ActorID: "actor"}, Kind: "source_create", Key: uuid.NewString(), BaseID: uuid.NewString(), Name: "文档"}
	_, e = service.Upload(ctx, command, "document.txt", []byte("test"))
	require.Error(t, e)
	originalID := native.revision.ID
	s, e := capacity.ReadSummary(ctx, "org")
	require.NoError(t, e)
	require.Equal(t, "4", s.ReservedBytes)
	_, e = service.Upload(ctx, command, "document.txt", []byte("changed"))
	require.ErrorIs(t, e, storagecapacity.ErrConflict)
	s, e = capacity.ReadSummary(ctx, "org")
	require.NoError(t, e)
	require.Equal(t, "4", s.ReservedBytes)
	_, e = service.Upload(ctx, command, "document.txt", []byte("test"))
	require.NoError(t, e)
	require.Equal(t, originalID, native.revision.ID)
	s, e = capacity.ReadSummary(ctx, "org")
	require.NoError(t, e)
	require.Equal(t, "4", s.OccupiedBytes)
	require.Equal(t, "0", s.ReservedBytes)
	missing := native.revision
	missing.ObjectKey = "knowledge/unbound"
	processor := &Processor{repo: native, objects: objects, capacity: meter}
	before := native.confirms
	processor.process(ctx, missing)
	require.Equal(t, before, native.confirms)
	s, e = capacity.ReadSummary(ctx, "org")
	require.NoError(t, e)
	require.Equal(t, "0", s.ReservedBytes)
}
