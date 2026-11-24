package submission

import (
	"context"
	"encoding/json"
	"time"

	"github.com/henryxu1008/shuomi-ai-commerce/internal/authidentity"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/listing/preparation"
	record "github.com/henryxu1008/shuomi-ai-commerce/internal/listing/record/target"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/marketplace/shopify"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/product/collection"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/storecenter"
)

const ShopifyDraftAction = "create_shopify_draft"

func ShopifyDraftTarget(binding storecenter.ProductMerchantBinding, productKey string) (ExecutionTarget, error) {
	if !binding.ValidShopifyApplication() || !authidentity.IsBoundedIdentifier(productKey) {
		return ExecutionTarget{}, ErrExecutionEvidenceRequired
	}
	// StoreID in this effect is the Store owner's stable shop reference, not
	// the replaceable Workbench store record. The normal kernel fence is reused.
	return ExecutionTarget{Platform: "shopify", StoreID: binding.Shopify.ShopKey,
		SubjectID: "shopify-draft-" + digestExecutionParts(binding.OrganizationID, binding.Shopify.ShopKey, productKey)}, nil
}

func ShopifyDraftPayload(saved record.TargetRecord) (shopify.Draft, []byte, error) {
	if saved.Merchant.Platform != storecenter.PlatformShopify || !record.PublicationReady(saved) {
		return shopify.Draft{}, nil, ErrExecutionEvidenceRequired
	}
	draft := *saved.Shopify
	draft.SourceKey = digestExecutionParts("shopify-draft", saved.Merchant.OrganizationID, saved.Merchant.Shopify.ShopKey, saved.Source.Source.ProductKey)
	raw, err := json.Marshal(draft)
	if err != nil || len(raw) > MaxExecutionPayloadBytes {
		return shopify.Draft{}, nil, ErrExecutionEvidenceRequired
	}
	return draft, raw, nil
}

func PrepareShopifyDraftIntent(ctx context.Context, sourceProof preparation.AuthorizedSource, saved record.TargetRecord, key string) (OfficialIntentCommit, []byte, error) {
	scope, source, _, err := sourceProof.Read(ctx)
	if err != nil || source != saved.Source || scope.OrganizationID != saved.Merchant.OrganizationID {
		return OfficialIntentCommit{}, nil, ErrExecutionEvidenceRequired
	}
	target, err := ShopifyDraftTarget(saved.Merchant, source.Source.ProductKey)
	if err != nil {
		return OfficialIntentCommit{}, nil, err
	}
	_, raw, err := ShopifyDraftPayload(saved)
	if err != nil {
		return OfficialIntentCommit{}, nil, err
	}
	intent := OfficialIntent{Key: key, Owner: scope, Kind: "publish", RecordID: saved.ID, RecordHash: collection.Digest(saved),
		Source: source, Binding: saved.Merchant, Target: target, CreatedAt: saved.CreatedAt,
		PayloadFingerprint: digestExecutionParts(target.Platform, target.StoreID, target.SubjectID, ShopifyDraftAction, string(raw))}
	intent.InputHash = OfficialIntentInputHash(intent)
	if ValidateOfficialIntent(intent) != nil {
		return OfficialIntentCommit{}, nil, ErrExecutionEvidenceRequired
	}
	return OfficialIntentCommit{source: sourceProof, intent: intent, hash: collection.Digest(intent)}, raw, nil
}

func ShopifyPayloadMatchesIntent(intent OfficialIntent, raw []byte) bool {
	return intent.PayloadFingerprint == digestExecutionParts(intent.Target.Platform, intent.Target.StoreID, intent.Target.SubjectID, ShopifyDraftAction, string(raw))
}

func validateShopifyIntent(intent OfficialIntent) error {
	target, err := ShopifyDraftTarget(intent.Binding, intent.Source.Source.ProductKey)
	if err != nil || target != intent.Target || intent.Kind != "publish" || intent.Image != nil || intent.Shelf != nil || intent.Walmart != nil ||
		intent.TikTok != nil || intent.TemuImage != nil || len(intent.TemuImages) != 0 || intent.TemuRenewalFrom != "" {
		return ErrExecutionEvidenceRequired
	}
	return nil
}

func NewShopifyDraftCompletion(scope collection.Scope, saved record.TargetRecord, attempt ExecutionAttempt, claim ExecutionClaim, raw []byte, outcome shopify.ProductObservation) (OfficialCompletion, error) {
	target, err := ShopifyDraftTarget(saved.Merchant, saved.Source.Source.ProductKey)
	if err != nil || target != attempt.Target || !outcome.Valid() || outcome.Status != "DRAFT" {
		return OfficialCompletion{}, ErrExecutionEvidenceRequired
	}
	body, err := completionBase(scope, saved.Merchant, attempt, claim, raw, ShopifyDraftAction)
	if err != nil {
		return OfficialCompletion{}, err
	}
	body.Kind, body.RecordID, body.ProductKey = "publish", saved.ID, saved.Source.Source.ProductKey
	body.Shopify = &outcome
	body.ResponseHash = collection.Digest(outcome)
	return sealOfficialCompletion(claim, body)
}

func validateShopifyReceipt(receipt OfficialReceipt) error {
	target, err := ShopifyDraftTarget(receipt.Binding, receipt.ProductKey)
	if err != nil || target != receipt.Target || receipt.Kind != "publish" || !collection.ValidID(receipt.RecordID) ||
		receipt.Shopify == nil || !receipt.Shopify.Valid() || receipt.ResponseHash != collection.Digest(*receipt.Shopify) ||
		receipt.TikTok != nil || receipt.Walmart != nil || receipt.TemuProduct != nil || receipt.TemuImage != nil || receipt.Shelf != nil || receipt.Image != nil {
		return ErrExecutionEvidenceRequired
	}
	return nil
}

func NewShopifyDraftResolution(intent OfficialIntent, saved record.TargetRecord, attempt ExecutionAttempt, outcome shopify.ProductObservation) (OfficialResolution, error) {
	if ValidateOfficialIntent(intent) != nil || ValidatePersistedExecutionAttempt(attempt) != nil || !outcome.Valid() ||
		attempt.Status != ExecutionOutcomeUnknown || attempt.Action != ShopifyDraftAction ||
		intent.Binding.Platform != storecenter.PlatformShopify || intent.Binding != saved.Merchant || intent.Source != saved.Source ||
		intent.RecordID != saved.ID || intent.RecordHash != collection.Digest(saved) ||
		attempt.OrganizationID != intent.Owner.OrganizationID || attempt.Target != intent.Target ||
		attempt.IntentKey != intent.Key || attempt.PayloadFingerprint != intent.PayloadFingerprint {
		return OfficialResolution{}, ErrExecutionEvidenceRequired
	}
	_, raw, err := ShopifyDraftPayload(saved)
	if err != nil || !ShopifyPayloadMatchesIntent(intent, raw) {
		return OfficialResolution{}, ErrExecutionEvidenceRequired
	}
	body := OfficialReceipt{
		ID: attempt.AttemptID, Owner: intent.Owner, Kind: intent.Kind, RecordID: saved.ID, ProductKey: intent.Source.Source.ProductKey,
		Target: intent.Target, IntentKey: intent.Key, PayloadFingerprint: intent.PayloadFingerprint, Binding: intent.Binding,
		Shopify: &outcome, ResponseHash: collection.Digest(outcome), ObservedAt: time.Now().UTC().Truncate(time.Microsecond),
	}
	body, err = cloneOfficialReceipt(body)
	if err != nil || ValidateOfficialReceipt(body) != nil {
		return OfficialResolution{}, ErrExecutionEvidenceRequired
	}
	data := OfficialResolutionData{Receipt: body, FenceEpoch: attempt.FenceEpoch, IntentHash: collection.Digest(intent), RecordHash: intent.RecordHash}
	return OfficialResolution{data: data, hash: collection.Digest(data), expiresAt: time.Now().Add(5 * time.Second)}, nil
}
