package orgresource

import "errors"

const OperationReserve = "reserve"

type ReservationState string

const (
	ReservationReserved               ReservationState = "reserved"
	ReservationCommitted              ReservationState = "committed"
	ReservationReleased               ReservationState = "released"
	ReservationReconciliationRequired ReservationState = "reconciliation_required"
)

var (
	ErrReservationOwnerNotRegistered = errors.New("organization resource reservation owner is not registered")
	ErrOwnerScopeMismatch            = errors.New("organization resource reservation owner scope mismatch")
	ErrOwnerNotReservable            = errors.New("organization resource reservation owner is not reservable")
	ErrInsufficientBalance           = errors.New("organization resource balance is insufficient")
)

var (
	ErrOwnerNotTerminal    = errors.New("organization resource reservation owner is not terminal")
	ErrInvalidOwnerProof   = errors.New("organization resource reservation owner proof is invalid")
	ErrReservationNotFound = errors.New("organization resource reservation was not found")
)
