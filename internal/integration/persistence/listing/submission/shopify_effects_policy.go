package submissionpersistence

import (
	"context"

	"gorm.io/gorm"

	"github.com/henryxu1008/shuomi-ai-commerce/internal/listing/submission"
)

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
