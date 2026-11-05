package storagecapacity

import (
	"context"
	"time"
)

// StoredObjectTarget identifies the original confirmed Knowledge upload.
// The browser cannot construct this internal target or a deletion proof.
type StoredObjectTarget struct {
	OrganizationID, ReservationID string
	Object                        Object
}

func (t StoredObjectTarget) Validate() error {
	if !ValidID(t.OrganizationID, 128) || !ValidID(t.ReservationID, 36) || !t.Object.ExclusiveBinding || t.Object.Validate() != nil {
		return ErrInvalid
	}
	return nil
}

type StoredDeletion struct {
	Target                   StoredObjectTarget
	OperationID, ProofDigest string
}

type ReleaseReceipt struct {
	OperationID string
	Bytes       int64
	ReleasedAt  time.Time
}

type StoredReleaser interface {
	// CheckStoredReleasable reports missing facts as ErrNotFound and a
	// present but ineligible target as ErrConflict.
	CheckStoredReleasable(context.Context, StoredObjectTarget) error
	ReleaseStored(context.Context, StoredDeletion) (ReleaseReceipt, error)
}
