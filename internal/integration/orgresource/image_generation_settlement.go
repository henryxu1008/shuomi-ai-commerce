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
func calculateImageGenerationSettlement(ctx context.Context, tx *gorm.DB, reservation organizationResourceReservationRow, bucket organizationResourceBucketRow, decision imageSettlementDecision) (calculatedImageSettlement, error) {
	result := calculatedImageSettlement{
		availableAfter: bucket.Available,
		reservedAfter:  bucket.Reserved - reservation.Quantity, consumedAfter: bucket.Consumed,
	}
	if decision == imageSettlementCommit {
		result.consumedAfter = bucket.Consumed + reservation.Quantity
		if result.consumedAfter < bucket.Consumed {
			return calculatedImageSettlement{}, fmt.Errorf("%w: consumed balance overflow", orgresource.ErrInvalidInput)
		}
		return result, nil
	}

	credit, err := applyPositiveCredit(ctx, tx, reservation.OrganizationID, reservation.ResourceType, reservation.Quantity, bucket.Available, time.Now().UTC())
	if err != nil {
		return calculatedImageSettlement{}, err
	}
	result.grossCredit = credit.gross
	result.debtRepaid = credit.debtRepaid
	result.netCredit = credit.net
	result.availableAfter = credit.availableAfter
	return result, nil
}
