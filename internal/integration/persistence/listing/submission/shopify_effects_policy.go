package submissionpersistence

import (
	"context"
	"crypto/subtle"
	"errors"

	"github.com/henryxu1008/shuomi-ai-commerce/internal/integration/persistence/listing/shopifyguard"

	"gorm.io/gorm"

	"github.com/henryxu1008/shuomi-ai-commerce/internal/listing/submission"
)

func shopifyGuardError(err error) error {
	switch {
	case errors.Is(err, shopifyguard.ErrFrozen):
		return submission.ErrExecutionNotFound
	case errors.Is(err, shopifyguard.ErrRestricted):
		return submission.ErrExecutionTargetClaimed
	case err != nil:
		return submission.ErrExecutionUnavailable
	default:
		return nil
	}
}

func requireShopifyGuard(ctx context.Context, tx *gorm.DB, attempt submission.ExecutionAttempt, create bool) error {
	if attempt.Target.Platform != "shopify" {
		return nil
	}
	intent, err := readOfficialAttemptIntent(ctx, tx, attempt)
	if err != nil {
		return err
	}
	source := ""
	if create {
		source = intent.Source.Source.ProductKey
	}
	binding := intent.Binding
	return shopifyGuardError(shopifyguard.Check(tx, attempt.OrganizationID, binding.Shopify.ShopKey, binding.ConnectionRevision, source))
}

func visibleShopifyAttempt(ctx context.Context, db *gorm.DB, attempt submission.ExecutionAttempt) error {
	if attempt.Target.Platform != "shopify" {
		return nil
	}
	intent, err := readOfficialAttemptIntent(ctx, db, attempt)
	if err != nil {
		return err
	}
	binding := intent.Binding
	return shopifyGuardError(shopifyguard.Visible(db.WithContext(ctx), attempt.OrganizationID, binding.Shopify.ShopKey, binding.ConnectionRevision))
}

// VerifyShopifySend rechecks the original persisted claim and its protected
// target before the one-shot local preparation releases its send permit.
func (r *Repository) VerifyShopifySend(ctx context.Context, claim submission.ExecutionClaim) error {
	if r == nil || !r.immutableOfficialEffects || r.transactionFinalizer {
		return submission.ErrExecutionUnavailable
	}
	_, err := r.write(ctx, func(tx *gorm.DB) (mutation, error) {
		original, found, err := r.findByID(ctx, tx, claim.Scope.OrganizationID, claim.AttemptID, false)
		if err != nil {
			return mutation{}, err
		}
		if !found || original.Target.Platform != "shopify" {
			return mutation{}, submission.ErrExecutionClaimRejected
		}
		if err := requireShopifyGuard(ctx, tx, original, true); err != nil {
			return mutation{}, err
		}
		attempt, _, hash, err := r.lockCurrentAttempt(ctx, tx, claim.Scope.OrganizationID, claim.AttemptID)
		if err != nil {
			return mutation{}, err
		}
		if attempt.FenceEpoch != claim.FenceEpoch || attempt.ClaimOwnerID != claim.OwnerID ||
			subtle.ConstantTimeCompare([]byte(hash), []byte(submission.ExecutionClaimTokenHash(claim.Token))) != 1 ||
			attempt.Status != submission.ExecutionClaimed || !r.now().Before(attempt.LeaseExpiresAt) {
			return mutation{}, submission.ErrExecutionClaimRejected
		}
		if err := r.requireShopifyEffect(ctx, tx, attempt); err != nil {
			return mutation{}, err
		}
		intent, err := readOfficialAttemptIntent(ctx, tx, attempt)
		if err != nil {
			return mutation{}, err
		}
		err = shopifyguard.AdmitDispatch(tx, shopifyguard.Dispatch{Organization: attempt.OrganizationID, ShopKey: intent.Binding.Shopify.ShopKey,
			Generation: intent.Binding.ConnectionRevision, AttemptID: attempt.AttemptID, Epoch: attempt.FenceEpoch})
		if errors.Is(err, shopifyguard.ErrConflict) {
			return mutation{}, submission.ErrExecutionClaimRejected
		}
		if err != nil {
			return mutation{}, submission.ErrExecutionUnavailable
		}
		return mutation{attempt: attempt}, nil
	})
	return err
}

func (r *Repository) requireShopifyEffect(ctx context.Context, tx *gorm.DB, attempt submission.ExecutionAttempt) error {
	intent, saved, err := readCurrentOfficialEffect(ctx, tx, attempt)
	if err != nil {
		return err
	}
	if attempt.Target.Platform != "shopify" || attempt.Action != submission.ShopifyDraftAction {
		return submission.ErrExecutionEvidenceRequired
	}
	_, raw, err := submission.ShopifyDraftPayload(saved)
	if err != nil || !submission.ShopifyPayloadMatchesIntent(intent, raw) {
		return submission.ErrExecutionEvidenceRequired
	}
	return nil
}
