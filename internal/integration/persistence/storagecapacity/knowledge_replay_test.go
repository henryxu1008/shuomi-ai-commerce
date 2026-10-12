package storagecapacitystore_test

import (
	"context"
	"errors"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	store "task-processor/internal/integration/persistence/storagecapacity"
	"task-processor/internal/knowledge"
	"task-processor/internal/ledger/storagecapacity"
	"testing"
)

type replayKnowledgeRepo struct {
	knowledge.Repository
	revision knowledge.Revision
	lost     bool
	confirms int
	claimed  knowledge.Revision
}

func (r *replayKnowledgeRepo) Apply(_ context.Context, c knowledge.Command) (knowledge.Result, error) {
	if r.revision.ID == "" {
		r.revision = *c.Upload
		r.revision.ID = uuid.NewString()
		r.revision.OrganizationID = c.Scope.OrganizationID
		r.revision.ObjectKey = "knowledge/" + r.revision.ID + "/" + r.revision.SHA256
		r.revision.State = knowledge.Admitted
	}
	if r.lost {
		r.lost = false
		return knowledge.Result{}, errors.New("original apply commit ACK lost")
	}
	return knowledge.Result{Source: &knowledge.Source{ID: "source"}, Revision: &r.revision}, nil
}
func (r *replayKnowledgeRepo) ClaimUpload(context.Context, string, string, string) (knowledge.Revision, bool, error) {
	return r.revision, true, nil
}
func (r *replayKnowledgeRepo) ConfirmObject(context.Context, knowledge.Revision) error {
	r.confirms++
	return nil
}
func (r *replayKnowledgeRepo) GetSource(context.Context, string, string) (knowledge.Source, error) {
	return knowledge.Source{ID: "source", LatestRevision: &r.revision}, nil
}

func (r *replayKnowledgeRepo) ClaimProcessing(context.Context, string, int) ([]knowledge.Revision, error) {
	return []knowledge.Revision{r.claimed}, nil
}

type replayKnowledgeParser struct{}

func (replayKnowledgeParser) Parse(context.Context, string, []byte) knowledge.ParseResult {
	return knowledge.ParseResult{Failure: "unexpected parser"}
}

type replayKnowledgeObjects struct {
	knowledge.KnowledgeObjectStore
	object knowledge.Object
}

func (o *replayKnowledgeObjects) Inspect(_ context.Context, _ knowledge.Object) (knowledge.Inspection, error) {
	return knowledge.Inspection{Exists: o.object.Key != "", SHA256: o.object.SHA256, SizeBytes: o.object.SizeBytes}, nil
}
func (o *replayKnowledgeObjects) PutImmutable(_ context.Context, object knowledge.Object) error {
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
	service, e := knowledge.NewService(native, objects)
	require.NoError(t, e)
	service.ConfigureStorageCapacity(meter)
	command := knowledge.Command{Scope: knowledge.Scope{OrganizationID: "org", ActorID: "actor"}, Kind: "source_create", Key: uuid.NewString(), BaseID: uuid.NewString(), Name: "文档"}
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
	native.claimed = missing
	processor, e := knowledge.NewProcessor(native, objects, replayKnowledgeParser{})
	require.NoError(t, e)
	processor.ConfigureStorageCapacity(meter)
	before := native.confirms
	require.NoError(t, processor.Sweep(ctx))
	require.Equal(t, before, native.confirms)
	s, e = capacity.ReadSummary(ctx, "org")
	require.NoError(t, e)
	require.Equal(t, "0", s.ReservedBytes)
}
