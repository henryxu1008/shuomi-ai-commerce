package storagecapacitystore

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"task-processor/internal/ledger/storagecapacity"
)

func TestStoredKnowledgeReleaseIsBoundAndCannotBeReactivated(t *testing.T) {
	repo, _ := storageFixture(t)
	ctx := context.Background()
	intent := reserveInput("org-a", "source_create:original", 35)
	intent.Owner = "knowledge"
	write, err := repo.ReserveWrite(ctx, intent)
	require.NoError(t, err)
	object := storagecapacity.Object{Namespace: "knowledge-deployment", Key: "knowledge/rev-a/digest", SHA256: storagecapacity.Digest("file"), SizeBytes: 35, Category: "knowledge_document", ExclusiveBinding: true}
	target := storagecapacity.StoredObjectTarget{OrganizationID: "org-a", ReservationID: write.ID, Object: object}
	require.NoError(t, repo.BindObjects(ctx, "org-a", write.ID, []storagecapacity.Object{object}))
	another := intent
	another.OperationID = "source_create:second"
	other, err := repo.ReserveWrite(ctx, another)
	require.NoError(t, err)
	require.ErrorIs(t, repo.BindObjects(ctx, "org-a", other.ID, []storagecapacity.Object{object}), storagecapacity.ErrConflict)
	require.NoError(t, repo.AbortUnstarted(ctx, "org-a", other.ID))
	missing := target
	missing.Object.Key += "-absent"
	require.ErrorIs(t, repo.CheckStoredReleasable(ctx, missing), storagecapacity.ErrNotFound)
	require.ErrorIs(t, repo.CheckStoredReleasable(ctx, target), storagecapacity.ErrConflict)
	require.NoError(t, repo.ClaimDispatch(ctx, "org-a", write.ID))
	require.NoError(t, repo.ConfirmStored(ctx, "org-a", write.ID, object.Namespace, object.Key, object.SHA256, object.SizeBytes))
	require.NoError(t, repo.CheckStoredReleasable(ctx, target))

	proof := storagecapacity.StoredDeletion{Target: target, OperationID: "delete-a", ProofDigest: storagecapacity.Digest("known-absence")}
	wrong := proof
	wrong.Target.OrganizationID = "org-b"
	_, err = repo.ReleaseStored(ctx, wrong)
	require.Error(t, err)
	receipt, err := repo.ReleaseStored(ctx, proof)
	require.NoError(t, err)
	require.Equal(t, int64(35), receipt.Bytes)
	replay, err := repo.ReleaseStored(ctx, proof)
	require.NoError(t, err)
	require.Equal(t, receipt, replay)
	wrong = proof
	wrong.ProofDigest = storagecapacity.Digest("different")
	_, err = repo.ReleaseStored(ctx, wrong)
	require.ErrorIs(t, err, storagecapacity.ErrConflict)
	summary, err := repo.ReadSummary(ctx, "org-a")
	require.NoError(t, err)
	require.Equal(t, "0", summary.OccupiedBytes)
	require.Equal(t, "0", summary.ReservedBytes)
	require.Empty(t, summary.Categories)
	require.Equal(t, "5368709120", summary.RemainingBytes)
	require.ErrorIs(t, repo.BindObjects(ctx, "org-a", write.ID, []storagecapacity.Object{object}), storagecapacity.ErrConflict)
	require.ErrorIs(t, repo.ClaimDispatch(ctx, "org-a", write.ID), storagecapacity.ErrConflict)
	require.ErrorIs(t, repo.ConfirmStored(ctx, "org-a", write.ID, object.Namespace, object.Key, object.SHA256, object.SizeBytes), storagecapacity.ErrConflict)
	_, err = repo.FindObjectWrite(ctx, "org-a", "knowledge", object)
	require.ErrorIs(t, err, storagecapacity.ErrConflict)
}
