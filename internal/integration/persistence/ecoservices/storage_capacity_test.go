package ecoservices

import (
	"context"
	"errors"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	e "task-processor/internal/ecoservices"
	capstore "task-processor/internal/integration/persistence/storagecapacity"
	"task-processor/internal/ledger/storagecapacity"
	"testing"
)

type requestCapacityObjects struct {
	e.PrivateObjectStore
	file e.File
	puts int
}

func (o *requestCapacityObjects) Inspect(_ context.Context, f e.File) (e.ObjectInspection, error) {
	return e.ObjectInspection{Exists: o.file.ID == f.ID, ContentType: o.file.ContentType, SHA256: o.file.SHA256, SizeBytes: o.file.SizeBytes}, nil
}
func (o *requestCapacityObjects) PutImmutable(_ context.Context, f e.File, _ []byte) error {
	o.file = f
	o.puts++
	return nil
}

type lostFileIntentACK struct {
	e.FileRepository
	lose bool
}

type proofFileIntentRepo struct {
	e.FileRepository
	create func(context.Context, e.FileIntent) (e.File, error)
}

type failedRequestCapacityAbort struct {
	storagecapacity.Writer
	fail bool
}

func (w *failedRequestCapacityAbort) AbortUnstarted(ctx context.Context, org, id string) error {
	if w.fail {
		return storagecapacity.ErrUnavailable
	}
	return w.Writer.AbortUnstarted(ctx, org, id)
}

func TestRequestFileFailedAbortKeepsOriginalIntentAndReturnsUnknown(t *testing.T) {
	r, _ := fixture(t)
	service, capacity, objects := requestCapacityFixture(t, r)
	writer := &failedRequestCapacityAbort{Writer: capacity, fail: true}
	service.ConfigureStorageCapacity(&storagecapacity.Meter{Writer: writer, Namespace: "request-files"})
	ctx := context.Background()
	scope := e.Scope{OrganizationID: "buyer", ActorID: "actor"}
	key, parent := uuid.NewString(), uuid.NewString()
	_, err := service.Upload(ctx, scope, key, "REQUEST", parent, "brief.txt", []byte("test"))
	require.ErrorIs(t, err, storagecapacity.ErrUnknown)
	require.NotErrorIs(t, err, e.ErrNotFound)
	summary, err := capacity.ReadSummary(ctx, "buyer")
	require.NoError(t, err)
	require.Equal(t, "4", summary.ReservedBytes)
	writer.fail = false
	_, err = service.Upload(ctx, scope, key, "REQUEST", parent, "brief.txt", []byte("test"))
	require.ErrorIs(t, err, e.ErrNotFound)
	summary, err = capacity.ReadSummary(ctx, "buyer")
	require.NoError(t, err)
	require.Equal(t, "0", summary.ReservedBytes)
	require.Zero(t, objects.puts)
}

func (r *proofFileIntentRepo) CreateFileIntent(ctx context.Context, in e.FileIntent) (e.File, error) {
	return r.create(ctx, in)
}

func (r *lostFileIntentACK) CreateFileIntent(ctx context.Context, in e.FileIntent) (e.File, error) {
	f, err := r.FileRepository.CreateFileIntent(ctx, in)
	if err == nil && r.lose {
		r.lose = false
		return e.File{}, errors.New("commit ACK lost")
	}
	return f, err
}
func requestCapacityFixture(t *testing.T, r e.FileRepository) (*e.FileService, *capstore.Repository, *requestCapacityObjects) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:"+uuid.NewString()+"?mode=memory&cache=shared"), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	require.NoError(t, err)
	pool, _ := db.DB()
	t.Cleanup(func() { _ = pool.Close() })
	require.NoError(t, capstore.AutoMigrate(db))
	capacity, err := capstore.New(db)
	require.NoError(t, err)
	objects := &requestCapacityObjects{}
	service, err := e.NewFileService(r, objects)
	require.NoError(t, err)
	service.ConfigureStorageCapacity(&storagecapacity.Meter{Writer: capacity, Namespace: "request-files"})
	return service, capacity, objects
}
func TestRequestFileRejectionDoesNotConsumeCapacity(t *testing.T) {
	r, _ := fixture(t)
	service, capacity, objects := requestCapacityFixture(t, r)
	ctx := context.Background()
	scope := e.Scope{OrganizationID: "buyer", ActorID: "actor"}
	for i := 0; i < 3; i++ {
		_, err := service.Upload(ctx, scope, uuid.NewString(), "REQUEST", uuid.NewString(), "brief.txt", []byte("test"))
		require.ErrorIs(t, err, e.ErrNotFound)
	}
	s, err := capacity.ReadSummary(ctx, "buyer")
	require.NoError(t, err)
	require.Equal(t, "0", s.ReservedBytes)
	require.Equal(t, 0, objects.puts)
	var count int64
	require.NoError(t, r.db.Model(&fileRow{}).Count(&count).Error)
	require.Zero(t, count)
}
func TestRequestFileDatabaseErrorIsUnknownRatherThanNoCreateProof(t *testing.T) {
	r, _ := fixture(t)
	service, capacity, objects := requestCapacityFixture(t, r)
	require.NoError(t, r.db.Migrator().DropTable(&requestRow{}))
	ctx := context.Background()
	_, err := service.Upload(ctx, e.Scope{OrganizationID: "buyer", ActorID: "actor"}, uuid.NewString(), "REQUEST", uuid.NewString(), "brief.txt", []byte("test"))
	require.Error(t, err)
	require.False(t, errors.Is(err, e.ErrNotFound), "database failure is not definitive parent absence")
	s, err := capacity.ReadSummary(ctx, "buyer")
	require.NoError(t, err)
	require.Equal(t, "4", s.ReservedBytes)
	require.Equal(t, 0, objects.puts)
}
func TestRequestFileLostIntentACKThenAuthorizationLossPreservesOriginalWrite(t *testing.T) {
	r, _ := fixture(t)
	ctx := context.Background()
	parent := uuid.NewString()
	require.NoError(t, r.db.Create(requestRecord(e.Request{ID: parent, BuyerOrganizationID: "buyer", ProviderOrganizationID: "provider", State: "AWAITING_ACCEPTANCE", Version: 1})).Error)
	lost := &lostFileIntentACK{FileRepository: r, lose: true}
	service, capacity, objects := requestCapacityFixture(t, lost)
	scope := e.Scope{OrganizationID: "buyer", ActorID: "actor"}
	key := uuid.NewString()
	_, err := service.Upload(ctx, scope, key, "REQUEST", parent, "brief.txt", []byte("test"))
	require.Error(t, err)
	require.NoError(t, r.db.Model(&requestRow{}).Where("id=?", parent).Update("buyer_organization_id", "other").Error)
	_, err = service.Upload(ctx, scope, key, "REQUEST", parent, "brief.txt", []byte("test"))
	require.ErrorIs(t, err, e.ErrNotFound)
	s, err := capacity.ReadSummary(ctx, "buyer")
	require.NoError(t, err)
	require.Equal(t, "4", s.ReservedBytes, "existing original intent cannot be aborted from a later authorization error")
	require.NoError(t, r.db.Model(&requestRow{}).Where("id=?", parent).Update("buyer_organization_id", "buyer").Error)
	_, err = service.Upload(ctx, scope, key, "REQUEST", parent, "brief.txt", []byte("test"))
	require.NoError(t, err)
	s, err = capacity.ReadSummary(ctx, "buyer")
	require.NoError(t, err)
	require.Equal(t, "4", s.OccupiedBytes)
	require.Equal(t, "0", s.ReservedBytes)
	require.Equal(t, 1, objects.puts)
}

func TestRequestFileNoCreateProofCannotCloseAnotherIdentityOrDispatchedWrite(t *testing.T) {
	for _, mode := range []string{"organization", "file", "fingerprint", "dispatched", "abort_fence"} {
		t.Run(mode, func(t *testing.T) {
			ctx := context.Background()
			repo := &proofFileIntentRepo{}
			service, capacity, objects := requestCapacityFixture(t, repo)
			repo.create = func(ctx context.Context, in e.FileIntent) (e.File, error) {
				proof := &e.FileIntentRejection{OrganizationID: in.Scope.OrganizationID, FileID: in.File.ID, Fingerprint: in.Fingerprint, Cause: e.ErrNotFound}
				switch mode {
				case "organization":
					proof.OrganizationID = "other"
				case "file":
					proof.FileID = uuid.NewString()
				case "fingerprint":
					proof.Fingerprint = "other"
				case "dispatched":
					reservation, err := capacity.FindWrite(ctx, in.Scope.OrganizationID, "ecoservices_request", in.File.ID)
					require.NoError(t, err)
					require.NoError(t, capacity.ClaimDispatch(ctx, in.Scope.OrganizationID, reservation.ID))
				}
				return e.File{}, proof
			}
			key := uuid.NewString()
			_, err := service.Upload(ctx, e.Scope{OrganizationID: "buyer", ActorID: "actor"}, key, "REQUEST", uuid.NewString(), "brief.txt", []byte("test"))
			if mode == "dispatched" {
				require.ErrorIs(t, err, storagecapacity.ErrUnknown)
			} else {
				require.ErrorIs(t, err, e.ErrNotFound)
			}
			s, err := capacity.ReadSummary(ctx, "buyer")
			require.NoError(t, err)
			if mode == "abort_fence" {
				require.Equal(t, "0", s.ReservedBytes)
				id := uuid.NewSHA1(uuid.NameSpaceOID, []byte("ecoservices-file:buyer:"+key)).String()
				reservation, err := capacity.FindWrite(ctx, "buyer", "ecoservices_request", id)
				require.NoError(t, err)
				require.ErrorIs(t, capacity.ClaimDispatch(ctx, "buyer", reservation.ID), storagecapacity.ErrConflict)
			} else {
				require.Equal(t, "4", s.ReservedBytes)
			}
			require.Zero(t, objects.puts)
		})
	}
}
