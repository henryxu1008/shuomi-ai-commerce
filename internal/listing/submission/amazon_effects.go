package submission

import (
	"context"
	"sync/atomic"
	"time"

	"github.com/henryxu1008/shuomi-ai-commerce/internal/authidentity"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/listing/preparation"
	record "github.com/henryxu1008/shuomi-ai-commerce/internal/listing/record/target"
	amazonmarket "github.com/henryxu1008/shuomi-ai-commerce/internal/marketplace/amazonus"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/product/collection"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/storecenter"
)

const AmazonSubmitAction = "amazon_create_listing"

type AmazonIntent struct {
	SKU      string `json:"sku"`
	WireHash string `json:"wireHash"`
}

// AmazonSendAuthorization is issued inside the submit service and consumed
// once by the Product owner transaction. The seller partition stays outside
// the immutable official intent.
type AmazonSendAuthorization struct {
	intent    OfficialIntent
	source    preparation.AuthorizedSource
	partition string
}

// AmazonWritePermit can only be issued after a committed kernel acquisition.
// The outbound call consumes it once for the exact current Store and wire.
type AmazonWritePermit struct {
	intent    OfficialIntent
	expiresAt time.Time
	sent      *atomic.Bool
}

func (p *AmazonWritePermit) Consume(scope collection.Scope, binding storecenter.ProductMerchantBinding, draft amazonmarket.Draft, now time.Time) error {
	if p == nil || p.sent == nil || p.intent.Amazon == nil || p.intent.Owner != scope || p.intent.Binding != binding ||
		p.intent.Amazon.SKU != draft.SKU || p.intent.Amazon.WireHash != draft.WireHash || draft.Validate() != nil ||
		p.intent.PayloadFingerprint != digestExecutionParts("amazon", binding.StoreID, draft.SKU, AmazonSubmitAction, string(draft.Attributes)) ||
		now.IsZero() || !now.Before(p.expiresAt) || !p.sent.CompareAndSwap(false, true) {
		return ErrExecutionClaimRejected
	}
	return nil
}

func PrepareAmazonSend(ctx context.Context, source preparation.AuthorizedSource, saved record.TargetRecord, key, partition string) (*AmazonSendAuthorization, error) {
	scope, selected, _, err := source.Read(ctx)
	if err != nil || selected != saved.Source || saved.Amazon == nil || !record.PublicationReady(saved) ||
		!collection.ValidID(key) || !isExecutionDigest(partition) || scope.OrganizationID != saved.Merchant.OrganizationID {
		return nil, ErrExecutionEvidenceRequired
	}
	target := AmazonTarget(saved.Merchant.StoreID, selected.Source.ProductKey)
	body := OfficialIntent{
		Key: key, Owner: scope, Kind: "publish", RecordID: saved.ID, RecordHash: collection.Digest(saved),
		Source: selected, Target: target, Binding: saved.Merchant, CreatedAt: time.Now().UTC().Truncate(time.Microsecond),
		Amazon: &AmazonIntent{SKU: target.SubjectID, WireHash: saved.Amazon.Draft.WireHash},
	}
	body.PayloadFingerprint = digestExecutionParts(target.Platform, target.StoreID, target.SubjectID, AmazonSubmitAction, string(saved.Amazon.Draft.Attributes))
	body.InputHash = OfficialIntentInputHash(body)
	if ValidateOfficialIntent(body) != nil {
		return nil, ErrExecutionEvidenceRequired
	}
	return &AmazonSendAuthorization{intent: body, source: source, partition: partition}, nil
}

func (a *AmazonSendAuthorization) Read(ctx context.Context) (OfficialIntent, string, error) {
	if a == nil {
		return OfficialIntent{}, "", ErrExecutionEvidenceRequired
	}
	scope, source, _, err := a.source.Read(ctx)
	if err != nil || scope != a.intent.Owner || source != a.intent.Source || ValidateOfficialIntent(a.intent) != nil {
		return OfficialIntent{}, "", ErrExecutionEvidenceRequired
	}
	intent, err := cloneOfficialIntent(a.intent)
	return intent, a.partition, err
}

func AmazonTarget(storeID, productKey string) ExecutionTarget {
	return ExecutionTarget{Platform: "amazon", StoreID: storeID, SubjectID: amazonmarket.SellerSKU(storeID, productKey)}
}

func ValidAmazonTarget(t ExecutionTarget) bool {
	return t.Platform == "amazon" && collection.ValidID(t.StoreID) && amazonmarket.ValidSellerSKU(t.SubjectID)
}

func validateAmazonIntent(v OfficialIntent) error {
	if v.Amazon == nil || v.Kind != "publish" || v.Shelf != nil || v.Image != nil || v.Walmart != nil ||
		v.TikTok != nil || v.TemuImage != nil || len(v.TemuImages) != 0 || v.TemuRenewalFrom != "" ||
		!authidentity.IsBoundedIdentifier(v.Source.Source.ProductKey) || !isExecutionDigest(v.Amazon.WireHash) ||
		v.Amazon.SKU != v.Target.SubjectID || v.Target != AmazonTarget(v.Binding.StoreID, v.Source.Source.ProductKey) {
		return ErrExecutionEvidenceRequired
	}
	return nil
}

func AmazonIntentMatchesRecord(i OfficialIntent, saved record.TargetRecord) bool {
	return ValidateOfficialIntent(i) == nil && i.Amazon != nil && saved.Amazon != nil && record.PublicationReady(saved) &&
		i.RecordID == saved.ID && i.RecordHash == collection.Digest(saved) && i.Source == saved.Source &&
		i.Binding == saved.Merchant && i.Amazon.WireHash == saved.Amazon.Draft.WireHash &&
		i.PayloadFingerprint == digestExecutionParts("amazon", i.Target.StoreID, i.Target.SubjectID, AmazonSubmitAction, string(saved.Amazon.Draft.Attributes))
}
