package orgresourceadapter

import (
	"context"
	"encoding/json"
	"time"

	"task-processor/internal/aicapability"
	"task-processor/internal/ledger/orgresource"
)

type ModelInvocationReservation struct {
	OrganizationID      string     `json:"organizationId"`
	InvocationID        string     `json:"invocationId"`
	MemberID            string     `json:"memberId"`
	ReservationID       string     `json:"reservationId"`
	DispatchFingerprint string     `json:"dispatchFingerprint"`
	QuantityUnit        string     `json:"quantityUnit"`
	Quantity            int64      `json:"quantity"`
	MemberMonth         time.Time  `json:"memberMonth"`
	State               string     `json:"state"`
	SettledAt           *time.Time `json:"settledAt,omitempty"`
}

// InspectModelInvocationReservation reads the immutable original reservation
// and receipt. It has no reservation, recovery-scan or settlement side effects.
func (r *GormModelInvocationRepository) InspectModelInvocationReservation(ctx context.Context, org, invocationID string) (ModelInvocationReservation, error) {
	fact, err := r.readFact(ctx, org, invocationID)
	if err != nil {
		return ModelInvocationReservation{}, err
	}
	fingerprint := modelPointFingerprint(fact)
	quantity, err := fact.OfficialTariff.ReserveTextMicropoints(fact.MaximumPromptTokens, fact.MaximumCompletionTokens)
	if err != nil {
		return ModelInvocationReservation{}, orgresource.ErrInvalidOwnerProof
	}
	var operation organizationResourceOperationRow
	if err = r.db.WithContext(ctx).Where("organization_id = ? AND operation_id = ?", org, modelPointKey("reserve", invocationID)).Take(&operation).Error; err != nil {
		return ModelInvocationReservation{}, err
	}
	var receipt modelPointReceipt
	if operation.State != "succeeded" || operation.OperationType != "model_point_reserve" || operation.RequestFingerprint != fingerprint || json.Unmarshal([]byte(operation.ImmutableResult), &receipt) != nil {
		return ModelInvocationReservation{}, orgresource.ErrInvalidOwnerProof
	}
	var row organizationResourceReservationRow
	if err = r.db.WithContext(ctx).Where("organization_id = ? AND reservation_id = ?", org, receipt.ReservationID).Take(&row).Error; err != nil {
		return ModelInvocationReservation{}, err
	}
	if !matchesModelPointReceipt(fact, fingerprint, quantity, receipt, row) {
		return ModelInvocationReservation{}, orgresource.ErrInvalidOwnerProof
	}
	return ModelInvocationReservation{
		OrganizationID: org, InvocationID: invocationID, MemberID: fact.MemberID, ReservationID: row.ReservationID,
		DispatchFingerprint: fingerprint, QuantityUnit: row.QuantityUnit, Quantity: row.Quantity,
		MemberMonth: *row.MemberMonthStart, State: row.State, SettledAt: row.SettledAt,
	}, nil
}

func (r *GormModelInvocationRepository) VerifyProviderRejectionReservation(ctx context.Context, fact aicapability.InvocationRecord) error {
	if aicapability.ValidateProviderRejection(fact) != nil {
		return orgresource.ErrInvalidOwnerProof
	}
	view, err := r.InspectModelInvocationReservation(ctx, fact.TenantID, fact.InvocationID)
	if err != nil {
		return err
	}
	p := fact.ProviderRejectionProof
	if view.MemberID != fact.MemberID || view.ReservationID != p.ReservationID || view.DispatchFingerprint != p.DispatchFingerprint || view.DispatchFingerprint != modelPointFingerprint(fact) ||
		view.Quantity != p.ReservedMicropoints || !view.MemberMonth.Equal(p.MemberMonth) || view.QuantityUnit != fact.OfficialTariff.UnitVersion || view.State != "reserved" && view.State != "reconciliation_required" && view.State != "committed" {
		return orgresource.ErrInvalidOwnerProof
	}
	return nil
}
