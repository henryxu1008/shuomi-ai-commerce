package knowledge

import (
	"context"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"task-processor/internal/ledger/storagecapacity"
	"testing"
)

type quotaDeniedKnowledgeWriter struct{ storagecapacity.Writer }

func (quotaDeniedKnowledgeWriter) ReserveWrite(context.Context, storagecapacity.WriteIntent) (storagecapacity.Reservation, error) {
	return storagecapacity.Reservation{}, storagecapacity.ErrExceeded
}

type capacityAdmissionRepo struct {
	Repository
	applied  int
	revision Revision
}

type rejectionKnowledgeWriter struct {
	storagecapacity.Writer
	aborts int
}

func (w *rejectionKnowledgeWriter) ReserveWrite(_ context.Context, in storagecapacity.WriteIntent) (storagecapacity.Reservation, error) {
	return storagecapacity.Reservation{ID: "reservation", Intent: in}, nil
}
func (w *rejectionKnowledgeWriter) AbortUnstarted(context.Context, string, string) error {
	w.aborts++
	return nil
}

type rejectionKnowledgeRepo struct {
	Repository
	alter func(*UploadCommandRejection)
	plain error
}

func (r rejectionKnowledgeRepo) Apply(_ context.Context, c Command) (Result, error) {
	if r.plain != nil {
		return Result{}, r.plain
	}
	proof := &UploadCommandRejection{OrganizationID: c.Scope.OrganizationID, Kind: c.Kind, Key: c.Key, Fingerprint: c.Fingerprint, Cause: ErrNotFound}
	r.alter(proof)
	return Result{}, proof
}

func TestKnowledgeUploadOnlyAbortsExactNativeRejection(t *testing.T) {
	for _, field := range []string{"matching", "organization", "kind", "key", "fingerprint", "ordinary_error"} {
		t.Run(field, func(t *testing.T) {
			writer := &rejectionKnowledgeWriter{}
			repo := rejectionKnowledgeRepo{alter: func(p *UploadCommandRejection) {
				switch field {
				case "organization":
					p.OrganizationID = "foreign"
				case "kind":
					p.Kind = "revision_create"
				case "key":
					p.Key = uuid.NewString()
				case "fingerprint":
					p.Fingerprint = Digest([]byte("different"))
				}
			}}
			if field == "ordinary_error" {
				repo.plain = ErrNotFound
			}
			s := &Service{repo: repo, capacity: &storagecapacity.Meter{Writer: writer, Namespace: "namespace"}}
			_, err := s.Upload(context.Background(), Command{Scope: Scope{OrganizationID: "org", ActorID: "actor"}, Kind: "source_create", Key: uuid.NewString(), BaseID: uuid.NewString(), Name: "Guide"}, "guide.txt", []byte("test"))
			require.ErrorIs(t, err, ErrNotFound)
			want := 0
			if field == "matching" {
				want = 1
			}
			require.Equal(t, want, writer.aborts)
		})
	}
}

func (r *capacityAdmissionRepo) Apply(context.Context, Command) (Result, error) {
	r.applied++
	return Result{Revision: &r.revision}, nil
}
func (r *capacityAdmissionRepo) ClaimUpload(context.Context, string, string, string) (Revision, bool, error) {
	return r.revision, true, nil
}
func TestKnowledgeQuotaDenialDoesNotConsumeSourceAdmission(t *testing.T) {
	for _, kind := range []string{"source_create", "revision_create"} {
		t.Run(kind, func(t *testing.T) {
			r := &capacityAdmissionRepo{revision: Revision{ID: uuid.NewString(), OrganizationID: "org", ObjectKey: "knowledge/revision/hash", SizeBytes: 4, SHA256: Digest([]byte("test"))}}
			s := &Service{repo: r, capacity: &storagecapacity.Meter{Writer: quotaDeniedKnowledgeWriter{}, Namespace: "namespace"}}
			_, e := s.Upload(context.Background(), Command{Scope: Scope{OrganizationID: "org", ActorID: "actor"}, Kind: kind, Key: uuid.NewString(), BaseID: uuid.NewString(), SourceID: uuid.NewString(), Version: 1, Name: "文档"}, "document.txt", []byte("test"))
			require.ErrorIs(t, e, storagecapacity.ErrExceeded)
			require.Zero(t, r.applied, "capacity rejection must not consume a Knowledge source slot or revision version")
		})
	}
}
