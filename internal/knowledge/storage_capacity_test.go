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
