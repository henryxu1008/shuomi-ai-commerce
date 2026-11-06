package orgresourceadapter

import (
	"context"
	"fmt"
	"time"

	"gorm.io/gorm"

	"task-processor/internal/ledger/orgresource"
)

type imageSettlementDecision string

const (
	imageSettlementCommit  imageSettlementDecision = "commit"
	imageSettlementRelease imageSettlementDecision = "release"
)

type calculatedImageSettlement struct {
	availableAfter int64
	reservedAfter  int64
	consumedAfter  int64
	grossCredit    int64
	debtRepaid     int64
	netCredit      int64
}

// The caller owns the existing transaction and bucket lock. Release repays
// debt in that same transaction before any credit can become available.
func calculateImageGenerationSettlement(ctx context.Context, tx *gorm.DB, reservation organizationResourceReservationRow, bucket organizationResourceBucketRow, decision imageSettlementDecision, actual int64) (calculatedImageSettlement, error) {
	result := calculatedImageSettlement{
		availableAfter: bucket.Available,
		reservedAfter:  bucket.Reserved - reservation.Quantity, consumedAfter: bucket.Consumed,
	}
	if actual < 0 || actual > reservation.Quantity || decision == imageSettlementRelease && actual != 0 {
		return calculatedImageSettlement{}, orgresource.ErrInvalidOwnerProof
	}
	if decision == imageSettlementCommit {
		result.consumedAfter = bucket.Consumed + actual
		if result.consumedAfter < bucket.Consumed {
			return calculatedImageSettlement{}, fmt.Errorf("%w: consumed balance overflow", orgresource.ErrInvalidInput)
		}
	}
	delta := reservation.Quantity - actual
	if delta == 0 {
		return result, nil
	}
	credit, err := applyPositiveCredit(ctx, tx, reservation.OrganizationID, reservation.ResourceType, delta, bucket.Available, time.Now().UTC())
	if err != nil {
		return calculatedImageSettlement{}, err
	}
	result.grossCredit = credit.gross
	result.debtRepaid = credit.debtRepaid
	result.netCredit = credit.net
	result.availableAfter = credit.availableAfter
	return result, nil
}
