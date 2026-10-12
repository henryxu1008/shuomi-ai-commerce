package imageagent

import (
	"context"
	"task-processor/internal/ledger/storagecapacity"
)

type StorageGenerationFactReader interface {
	ReadGenerationFactForStorage(context.Context, string, string, string, int64, string, int64) (GenerationFact, error)
}

func StorageGenerationOperationKey(org, ownerHash, run string, revision int64, slot string, attempt int64) string {
	return generationHash([]any{org, ownerHash, run, revision, slot, attempt})
}
func StorageGenerationOperation(id SlotExternalEffectIdentity) string {
	ownerKey, _ := ArtifactOwnerKey(id.OwnerUserID)
	return StorageGenerationOperationKey(id.TenantID, ownerKey, id.RunID, id.PlanRevision, id.SlotID, int64(id.Attempt))
}
func (e *GenerationExecution) ConfigureStorageCapacity(w storagecapacity.Writer, maxBytes int64) {
	e.dependencies.StorageCapacity = w
	e.dependencies.MaxStorageBytes = maxBytes
}
func ConfigureGenerationRecoveryStorage(recovery GenerationRecovery, w storagecapacity.Writer) {
	if r, ok := recovery.(*generationRecovery); ok {
		r.capacity = w
	}
}
func closeNoGenerationCapacity(ctx context.Context, w storagecapacity.Writer, f GenerationFact) error {
	if w == nil || f.State != GenerationNoEffect {
		return nil
	}
	id := f.Intent.Identity
	operation := StorageGenerationOperation(id)
	reservation, e := w.FindWrite(ctx, id.TenantID, "image_agent", operation)
	if e == storagecapacity.ErrNotFound {
		return nil
	}
	if e != nil {
		return e
	}
	if reservation.State == "RESERVED" {
		return w.AbortUnstarted(ctx, id.TenantID, reservation.ID)
	}
	return w.CloseWrite(ctx, id.TenantID, reservation.ID, storagecapacity.CloseProof{Owner: "image_agent", OperationID: operation, Fingerprint: f.Fingerprint, TerminalReceipt: "no-generation:" + f.Fingerprint})
}
