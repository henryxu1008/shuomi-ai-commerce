//go:build integration

package knowledge_test

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	capacitystore "task-processor/internal/integration/persistence/storagecapacity"
	k "task-processor/internal/knowledge"
	"task-processor/internal/ledger/storagecapacity"
)

type rejectedKnowledgeObjects struct {
	k.KnowledgeObjectStore
	calls int
}

func (o *rejectedKnowledgeObjects) Inspect(context.Context, k.Object) (k.Inspection, error) {
	o.calls++
	return k.Inspection{}, nil
}

func (o *rejectedKnowledgeObjects) PutImmutable(context.Context, k.Object) error {
	o.calls++
	return nil
}

func TestKnowledgeRejectedUploadDoesNotRetainCapacityPostgres(t *testing.T) {
	f := newContextFixture(t)
	ctx := context.Background()
	require.NoError(t, capacitystore.AutoMigrate(f.db))
	capacity, err := capacitystore.New(f.db)
	require.NoError(t, err)
	objects := &rejectedKnowledgeObjects{}
	service, err := k.NewService(f.repo, objects)
	require.NoError(t, err)
	service.ConfigureStorageCapacity(&storagecapacity.Meter{Writer: capacity, Namespace: "knowledge-test"})

	for _, scenario := range []string{"missing_base", "missing_source", "inactive_base", "inactive_source", "stale_version", "source_limit", "busy_revision"} {
		t.Run(scenario, func(t *testing.T) {
			base := f.apply(k.Command{Kind: "base_create", Name: "Brand"}).Base
			command := k.Command{Scope: f.scope, Kind: "source_create", BaseID: base.ID, Name: "Guide"}
			want := k.ErrNotFound
			switch scenario {
			case "missing_base":
				command.BaseID = uuid.NewString()
			case "missing_source":
				command.Kind, command.SourceID, command.Version = "revision_create", uuid.NewString(), 1
			case "inactive_base":
				require.NoError(t, f.db.Exec("UPDATE knowledge_bases SET state=? WHERE id=?", k.Disabled, base.ID).Error)
				want = k.ErrInactive
			case "source_limit":
				for range k.MaxActiveSources {
					f.apply(k.Command{Kind: "source_create", BaseID: base.ID, Name: "Guide", Upload: &k.Revision{Filename: "guide.txt", ContentType: "text/plain", SizeBytes: 4, SHA256: k.Digest([]byte("test"))}})
				}
				want = k.ErrSourceLimit
			default:
				source := f.apply(k.Command{Kind: "source_create", BaseID: base.ID, Name: "Guide", Upload: &k.Revision{Filename: "guide.txt", ContentType: "text/plain", SizeBytes: 4, SHA256: k.Digest([]byte("test"))}}).Source
				command.Kind, command.SourceID, command.Version = "revision_create", source.ID, source.Version
				switch scenario {
				case "inactive_source":
					require.NoError(t, f.db.Exec("UPDATE knowledge_sources SET state=? WHERE id=?", k.Disabled, source.ID).Error)
					want = k.ErrInactive
				case "stale_version":
					command.Version++
					want = k.ErrConflict
				case "busy_revision":
					want = k.ErrRevisionBusy
				}
			}
			counts := func() []int64 {
				var out []int64
				for _, table := range []string{"knowledge_sources", "knowledge_revisions", "knowledge_ingest_operations"} {
					var n int64
					require.NoError(t, f.db.Table(table).Count(&n).Error)
					out = append(out, n)
				}
				return out
			}
			before := counts()
			for range 3 {
				command.Key = uuid.NewString()
				_, err := service.Upload(ctx, command, "guide.txt", []byte("test"))
				require.ErrorIs(t, err, want)
				var proof *k.UploadCommandRejection
				require.ErrorAs(t, err, &proof)
				require.Equal(t, command.Scope.OrganizationID, proof.OrganizationID)
				require.Equal(t, command.Kind, proof.Kind)
				require.Equal(t, command.Key, proof.Key)
				reservation, err := capacity.FindWrite(ctx, f.scope.OrganizationID, "knowledge", command.Kind+":"+command.Key)
				require.NoError(t, err)
				require.Equal(t, proof.Fingerprint, reservation.Intent.Fingerprint)
				require.ErrorIs(t, capacity.BindObjects(ctx, f.scope.OrganizationID, reservation.ID, []storagecapacity.Object{{Namespace: "knowledge-test", Key: "late/" + command.Key, SHA256: k.Digest([]byte("test")), SizeBytes: 4, Category: "knowledge_document"}}), storagecapacity.ErrConflict)
				require.ErrorIs(t, capacity.ClaimDispatch(ctx, f.scope.OrganizationID, reservation.ID), storagecapacity.ErrConflict)
			}
			summary, err := capacity.ReadSummary(ctx, f.scope.OrganizationID)
			require.NoError(t, err)
			require.Equal(t, "0", summary.ReservedBytes, "deterministic rejection must not charge unowned bytes")
			require.Equal(t, "0", summary.OccupiedBytes)
			require.Equal(t, before, counts(), "rejection must not create an owner operation or revision")
			require.Zero(t, objects.calls)
		})
	}
}

type knowledgeApplyACKLoss struct {
	k.Repository
	loseResponse bool
	last         k.Command
}

func (r *knowledgeApplyACKLoss) Apply(ctx context.Context, command k.Command) (k.Result, error) {
	r.last = command
	result, err := r.Repository.Apply(ctx, command)
	if err == nil && r.loseResponse {
		r.loseResponse = false
		return k.Result{}, k.ErrUnavailable
	}
	return result, err
}

func TestKnowledgeUploadRejectionUnknownAndExistingOutcomeRetainCapacityPostgres(t *testing.T) {
	f := newContextFixture(t)
	ctx := context.Background()
	require.NoError(t, capacitystore.AutoMigrate(f.db))
	capacity, err := capacitystore.New(f.db)
	require.NoError(t, err)
	native := &knowledgeApplyACKLoss{Repository: f.repo}
	objects := &ambiguousKnowledgeObjects{data: map[string]k.Object{}}
	service, err := k.NewService(native, objects)
	require.NoError(t, err)
	service.ConfigureStorageCapacity(&storagecapacity.Meter{Writer: capacity, Namespace: "knowledge-test"})
	command := k.Command{Scope: f.scope, Kind: "source_create", Key: uuid.NewString(), BaseID: f.base.ID, Name: "Guide"}
	assertUnknown := func(err error) {
		t.Helper()
		require.ErrorIs(t, err, k.ErrUnavailable)
		var proof *k.UploadCommandRejection
		require.False(t, errors.As(err, &proof), "SQL/ACK uncertainty cannot authorize release")
		summary, err := capacity.ReadSummary(ctx, f.scope.OrganizationID)
		require.NoError(t, err)
		require.Equal(t, "4", summary.ReservedBytes)
	}
	// The exact-outcome read fails before admission. Keep the original bytes,
	// then recover by the same command once the owner is readable again.
	require.NoError(t, f.db.Exec("ALTER TABLE knowledge_ingest_operations RENAME TO knowledge_ingest_operations_offline").Error)
	_, err = service.Upload(ctx, command, "guide.txt", []byte("test"))
	require.NoError(t, f.db.Exec("ALTER TABLE knowledge_ingest_operations_offline RENAME TO knowledge_ingest_operations").Error)
	assertUnknown(err)
	// A parent SQL failure is also not the zero-row business NotFound result.
	require.NoError(t, f.db.Exec("ALTER TABLE knowledge_bases RENAME TO knowledge_bases_offline").Error)
	_, err = service.Upload(ctx, command, "guide.txt", []byte("test"))
	require.NoError(t, f.db.Exec("ALTER TABLE knowledge_bases_offline RENAME TO knowledge_bases").Error)
	assertUnknown(err)
	_, err = service.Upload(ctx, command, "guide.txt", []byte("test"))
	require.NoError(t, err)
	require.Equal(t, 1, objects.puts)

	command.Key = uuid.NewString()
	native.loseResponse = true
	_, err = service.Upload(ctx, command, "guide.txt", []byte("test"))
	assertUnknown(err)
	original := native.last
	changed := original
	changed.Fingerprint = k.Digest([]byte("changed payload"))
	_, err = f.repo.Apply(ctx, changed)
	require.ErrorIs(t, err, k.ErrConflict)
	var proof *k.UploadCommandRejection
	require.False(t, errors.As(err, &proof), "an existing mismatched command is not a no-create proof")

	// The original outcome exists, but its revision read now fails. This must
	// retain bytes, even though a later replay can restore the same operation.
	require.NoError(t, f.db.Exec("ALTER TABLE knowledge_revisions RENAME TO knowledge_revisions_offline").Error)
	_, err = service.Upload(ctx, command, "guide.txt", []byte("test"))
	require.NoError(t, f.db.Exec("ALTER TABLE knowledge_revisions_offline RENAME TO knowledge_revisions").Error)
	assertUnknown(err)
	_, err = service.Upload(ctx, command, "guide.txt", []byte("test"))
	require.NoError(t, err)
	require.Equal(t, 2, objects.puts, "one PUT per original revision")
	_, err = service.Upload(ctx, command, "guide.txt", []byte("test"))
	require.NoError(t, err)
	require.Equal(t, 2, objects.puts)
	summary, err := capacity.ReadSummary(ctx, f.scope.OrganizationID)
	require.NoError(t, err)
	require.Equal(t, "0", summary.ReservedBytes)
	require.Equal(t, "8", summary.OccupiedBytes)

	// An invalid upload discovered after the source insert has begun remains
	// an ordinary error, even though this transaction subsequently rolls back.
	_, err = f.repo.Apply(ctx, k.Command{Scope: f.scope, Kind: "source_create", Key: uuid.NewString(), Fingerprint: k.Digest([]byte("no-upload")), BaseID: f.base.ID, Name: "Guide"})
	require.ErrorIs(t, err, k.ErrInvalid)
	require.False(t, errors.As(err, &proof))
}
