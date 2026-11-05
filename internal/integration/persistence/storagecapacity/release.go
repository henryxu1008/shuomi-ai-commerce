package storagecapacitystore

import (
	"context"
	"database/sql"
	"time"

	"gorm.io/gorm"
	"task-processor/internal/ledger/storagecapacity"
)

func sameObject(row objectRow, object storagecapacity.Object) bool {
	return row.Namespace == object.Namespace && row.Key == object.Key && row.SHA256 == object.SHA256 &&
		row.SizeBytes == object.SizeBytes && row.Category == object.Category && row.ExclusiveBinding == object.ExclusiveBinding
}

func VerifyStoredReleaseRuntime(ctx context.Context, db *gorm.DB) error {
	return db.WithContext(ctx).Exec("SELECT exclusive_binding, release_operation_id, release_proof_digest, released_at FROM public.storage_capacity_objects LIMIT 0").Error
}

func readStoredTarget(tx *gorm.DB, target storagecapacity.StoredObjectTarget) (objectRow, error) {
	var row objectRow
	if err := tx.First(&row, "identity=? AND organization_id=?", objectIdentity(target.Object.Namespace, target.Object.Key), target.OrganizationID).Error; err != nil {
		return row, err
	}
	if !sameObject(row, target.Object) {
		return row, storagecapacity.ErrConflict
	}
	var bindings []bindingRow
	if err := tx.Where("object_identity=?", row.Identity).Limit(2).Find(&bindings).Error; err != nil {
		return row, err
	}
	if len(bindings) != 1 || !bindings[0].Confirmed || bindings[0].ReservationID != target.ReservationID {
		return row, storagecapacity.ErrConflict
	}
	write, err := findReservation(tx, target.OrganizationID, target.ReservationID)
	if err != nil {
		return row, err
	}
	if write.Owner != "knowledge" || write.Remaining != 0 || write.MaxBytes != row.SizeBytes || write.MaxObjects != 1 || (write.State != "DISPATCHED" && write.State != "CLOSED") {
		return row, storagecapacity.ErrConflict
	}
	return row, nil
}

func (r *Repository) CheckStoredReleasable(ctx context.Context, target storagecapacity.StoredObjectTarget) error {
	if err := target.Validate(); err != nil {
		return err
	}
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var account accountRow
		if err := tx.First(&account, "organization_id = ?", target.OrganizationID).Error; err != nil {
			return err
		}
		row, err := readStoredTarget(tx, target)
		if err != nil {
			return err
		}
		if !row.Stored || row.ReleaseOperationID != "" || account.Occupied < row.SizeBytes {
			return storagecapacity.ErrConflict
		}
		return nil
	}, &sql.TxOptions{Isolation: sql.LevelRepeatableRead, ReadOnly: true})
}

func (r *Repository) ReleaseStored(ctx context.Context, proof storagecapacity.StoredDeletion) (receipt storagecapacity.ReleaseReceipt, err error) {
	if proof.Target.Validate() != nil || !storagecapacity.ValidID(proof.OperationID, 256) || !storagecapacity.ValidDigest(proof.ProofDigest) {
		return receipt, storagecapacity.ErrInvalid
	}
	err = r.locked(ctx, proof.Target.OrganizationID, func(tx *gorm.DB, account *accountRow, now time.Time) error {
		row, err := readStoredTarget(tx, proof.Target)
		if err != nil {
			return err
		}
		if row.ReleaseOperationID != "" {
			if row.Stored || row.ReleasedAt == nil || row.ReleaseOperationID != proof.OperationID || row.ReleaseProofDigest != proof.ProofDigest {
				return storagecapacity.ErrConflict
			}
			receipt = storagecapacity.ReleaseReceipt{OperationID: row.ReleaseOperationID, Bytes: row.SizeBytes, ReleasedAt: row.ReleasedAt.UTC()}
			return nil
		}
		if !row.Stored || account.Occupied < row.SizeBytes {
			return storagecapacity.ErrConflict
		}
		// Keep the original object identity and binding as the replay fence.
		result := tx.Model(&row).Where("stored = ? AND release_operation_id = ?", true, "").Updates(map[string]any{
			"stored": false, "release_operation_id": proof.OperationID, "release_proof_digest": proof.ProofDigest, "released_at": now,
		})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected != 1 {
			return storagecapacity.ErrConflict
		}
		account.Occupied -= row.SizeBytes
		if err := saveAccount(tx, account); err != nil {
			return err
		}
		receipt = storagecapacity.ReleaseReceipt{OperationID: proof.OperationID, Bytes: row.SizeBytes, ReleasedAt: now.UTC()}
		return nil
	})
	if err != nil {
		return storagecapacity.ReleaseReceipt{}, err
	}
	return receipt, nil
}
