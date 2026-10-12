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
func closeGenerationCapacity(ctx context.Context, w storagecapacity.Writer, f GenerationFact) error {
	noGeneration := f.State == GenerationNoEffect
	invalidOutput := f.State == GenerationSucceeded && f.Success.ResultUnavailable == "invalid_result" && f.Settlement.State == "committed"
	if w == nil || !noGeneration && !invalidOutput {
		return nil
	}
	if f.Validate() != nil {
		return ErrCorruptPersistedEffect
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
	if noGeneration && reservation.State == "RESERVED" {
		return w.AbortUnstarted(ctx, id.TenantID, reservation.ID)
	}
	receipt := "no-generation:" + f.Fingerprint
	if invalidOutput {
		// Native invalid_result is immutable and publication rejects it. Its
		// settled point charge remains committed; only unused bytes can close.
		receipt = "invalid-generation:" + f.TerminalProofDigest()
	}
	return w.CloseWrite(ctx, id.TenantID, reservation.ID, storagecapacity.CloseProof{Owner: "image_agent", OperationID: operation, Fingerprint: f.Fingerprint, TerminalReceipt: receipt})
}
