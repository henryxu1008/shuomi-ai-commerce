package httpapi

import (
	"context"
	"encoding/json"
	"github.com/google/uuid"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/app/productsourcing"
	acquisitionstore "github.com/henryxu1008/shuomi-ai-commerce/internal/integration/persistence/product/acquisition"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/ledger/orgresource"
	"gorm.io/gorm"
)

// Existing route tests explicitly substitute only the Resource boundary; the
// Product intent and terminal proof still come from the real owner repository.
type testHTTPResourceCharges struct{ db *gorm.DB }

func (c testHTTPResourceCharges) Reserve(ctx context.Context, identity orgresource.ConsumerChargeIdentity) (orgresource.ConsumerChargeReceipt, error) {
	repo, err := acquisitionstore.NewRepository(ctx, c.db)
	if err != nil {
		return orgresource.ConsumerChargeReceipt{}, err
	}
	owner, err := productsourcing.NewAcquisitionChargeOwner(repo)
	if err != nil {
		return orgresource.ConsumerChargeReceipt{}, err
	}
	intent, err := owner.ReadChargeIntent(ctx, identity)
	if err != nil {
		return orgresource.ConsumerChargeReceipt{}, err
	}
	proof, err := repo.ReadChargeProof(ctx, identity.OrganizationID, identity.OperationID)
	if err != nil {
		return orgresource.ConsumerChargeReceipt{}, err
	}
	reservation := proof.ReservationID
	if reservation == "" {
		encoded, _ := json.Marshal(identity)
		reservation = uuid.NewSHA1(uuid.NameSpaceOID, encoded).String()
	}
	return orgresource.ConsumerChargeReceipt{Intent: intent, ReservationID: reservation, State: orgresource.ReservationReserved}, nil
}
func (c testHTTPResourceCharges) Reconcile(ctx context.Context, identity orgresource.ConsumerChargeIdentity) (orgresource.ConsumerChargeReceipt, error) {
	receipt, err := c.Reserve(ctx, identity)
	if err != nil {
		return orgresource.ConsumerChargeReceipt{}, err
	}
	repo, err := acquisitionstore.NewRepository(ctx, c.db)
	if err != nil {
		return orgresource.ConsumerChargeReceipt{}, err
	}
	owner, err := productsourcing.NewAcquisitionChargeOwner(repo)
	if err != nil {
		return orgresource.ConsumerChargeReceipt{}, err
	}
	proof, err := owner.ReadChargeProof(ctx, receipt)
	if err != nil {
		return orgresource.ConsumerChargeReceipt{}, err
	}
	if proof.State == orgresource.ConsumerEffectSucceeded {
		receipt.State = orgresource.ReservationCommitted
	} else if proof.State == orgresource.ConsumerEffectFailed {
		receipt.State = orgresource.ReservationReleased
	} else {
		return receipt, orgresource.ErrConsumerChargeUnknown
	}
	return receipt, nil
}
