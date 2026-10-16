package submission

import (
	"context"
	"reflect"
	"sort"
	"time"

	"github.com/google/uuid"
	"task-processor/internal/authidentity"
	"task-processor/internal/listing/preparation"
	record "task-processor/internal/listing/record/target"
	model "task-processor/internal/marketplace/shein/model"
	marketplace "task-processor/internal/marketplace/walmart"
	"task-processor/internal/product/collection"
	"task-processor/internal/storecenter"
)

const WalmartSubmitAction = "walmart_submit_feed"

func WalmartCorrelationID(intent OfficialIntent) string {
	return uuid.NewSHA1(uuid.NameSpaceURL, []byte("walmart-feed-v1:"+intent.Owner.OrganizationID+":"+intent.Key+":"+intent.PayloadFingerprint)).String()
}

type WalmartFeedIntent struct {
	Selection  marketplace.RuleSelection `json:"selection"`
	SKUs       []marketplace.SKUBinding  `json:"skus"`
	WireHash   string                    `json:"wireHash"`
	RepairFrom string                    `json:"repairFrom,omitempty"`
}
type WalmartFeedReceipt struct {
	FeedID        string                    `json:"feedId"`
	CorrelationID string                    `json:"correlationId"`
	WireHash      string                    `json:"wireHash"`
	Selection     marketplace.RuleSelection `json:"selection"`
	SKUs          []marketplace.SKUBinding  `json:"skus"`
	RepairFrom    string                    `json:"repairFrom,omitempty"`
}

func WalmartSubjectID(productKey, merchantHash string) (string, error) {
	if !authidentity.IsBoundedIdentifier(productKey) || !isExecutionDigest(merchantHash) {
		return "", ErrExecutionInvalid
	}
	return "walmart-offer-us-" + digestExecutionParts(productKey, merchantHash, "US", "walmart-offer-create"), nil
}
func WalmartCompatibleBinding(a, b storecenter.ProductMerchantBinding) bool {
	return a.ValidWalmartApplication() && b.ValidWalmartApplication() && a.OrganizationID == b.OrganizationID && a.StoreID == b.StoreID && a.SupplierIdentityHash == b.SupplierIdentityHash && a.Walmart.Environment == b.Walmart.Environment
}
func walmartSKUsValid(skus []marketplace.SKUBinding) bool {
	if len(skus) == 0 || len(skus) > marketplace.MaxSKUs {
		return false
	}
	sources, sellers := map[string]bool{}, map[string]bool{}
	for _, s := range skus {
		if s.SourceSKUKey == "" || len(s.SourceSKUKey) > 512 || s.SellerSKU == "" || len(s.SellerSKU) > 128 || sources[s.SourceSKUKey] || sellers[s.SellerSKU] {
			return false
		}
		sources[s.SourceSKUKey] = true
		sellers[s.SellerSKU] = true
	}
	return true
}
func WalmartExpectedSKUs(skus []marketplace.SKUBinding) []string {
	out := []string{}
	for _, s := range skus {
		out = append(out, s.SellerSKU)
	}
	sort.Strings(out)
	return out
}
func PrepareWalmartIntent(ctx context.Context, proof preparation.AuthorizedSource, saved record.TargetRecord, key string) (OfficialIntentCommit, error) {
	scope, source, _, e := proof.Read(ctx)
	if e != nil || source != saved.Source || !record.PublicationReady(saved) || saved.Walmart == nil || !authidentity.IsBoundedIdentifier(key) || scope.OrganizationID != saved.Merchant.OrganizationID {
		return OfficialIntentCommit{}, ErrExecutionEvidenceRequired
	}
	subject, e := WalmartSubjectID(source.Source.ProductKey, saved.Merchant.SupplierIdentityHash)
	if e != nil {
		return OfficialIntentCommit{}, e
	}
	d := saved.Walmart
	body := OfficialIntent{Key: key, Owner: scope, Kind: "publish", RecordID: saved.ID, RecordHash: collection.Digest(saved), Source: source, Binding: saved.Merchant, Target: ExecutionTarget{Platform: "walmart", StoreID: saved.Merchant.StoreID, SubjectID: subject}, Walmart: &WalmartFeedIntent{Selection: d.Selection, SKUs: d.SKUs, WireHash: d.WireHash, RepairFrom: saved.Input.Walmart.RepairFrom}, CreatedAt: time.Now().UTC().Truncate(time.Microsecond)}
	body.PayloadFingerprint = digestExecutionParts("walmart", body.Target.StoreID, subject, WalmartSubmitAction, string(d.Payload))
	body.InputHash = OfficialIntentInputHash(body)
	body, e = cloneOfficialIntent(body)
	if e != nil || ValidateOfficialIntent(body) != nil {
		return OfficialIntentCommit{}, ErrExecutionEvidenceRequired
	}
	return OfficialIntentCommit{source: proof, intent: body, hash: collection.Digest(body)}, nil
}
func validateWalmartIntent(v OfficialIntent) error {
	if v.Walmart == nil || v.Kind != "publish" || v.Image != nil || v.TikTok != nil || v.TemuImage != nil || len(v.TemuImages) > 0 || v.TemuRenewalFrom != "" || !v.Walmart.Selection.Valid() || !isExecutionDigest(v.Walmart.WireHash) || !walmartSKUsValid(v.Walmart.SKUs) || v.Walmart.RepairFrom != "" && !validExecutionAttemptID(v.Walmart.RepairFrom) {
		return ErrExecutionEvidenceRequired
	}
	subject, e := WalmartSubjectID(v.Source.Source.ProductKey, v.Binding.SupplierIdentityHash)
	if e != nil || v.Target.SubjectID != subject {
		return ErrExecutionEvidenceRequired
	}
	return nil
}
func WalmartIntentMatchesRecord(v OfficialIntent, saved record.TargetRecord) bool {
	return ValidateOfficialIntent(v) == nil && saved.Walmart != nil && record.PublicationReady(saved) && v.RecordID == saved.ID && v.RecordHash == collection.Digest(saved) && v.Source == saved.Source && v.Binding == saved.Merchant && v.Walmart.Selection == saved.Walmart.Selection && v.Walmart.WireHash == saved.Walmart.WireHash && reflect.DeepEqual(v.Walmart.SKUs, saved.Walmart.SKUs) && v.Walmart.RepairFrom == saved.Input.Walmart.RepairFrom && v.PayloadFingerprint == digestExecutionParts("walmart", v.Target.StoreID, v.Target.SubjectID, WalmartSubmitAction, string(saved.Walmart.Payload))
}
func NewWalmartCompletion(owner collection.Scope, saved record.TargetRecord, intent OfficialIntent, attempt ExecutionAttempt, claim ExecutionClaim, accepted marketplace.AcceptedFeed) (OfficialCompletion, error) {
	if !WalmartIntentMatchesRecord(intent, saved) || accepted.CorrelationID != WalmartCorrelationID(intent) || accepted.FeedID == "" || len(accepted.FeedID) > 256 || attempt.IntentKey != intent.Key {
		return OfficialCompletion{}, ErrExecutionEvidenceRequired
	}
	body, e := completionBase(owner, saved.Merchant, attempt, claim, saved.Walmart.Payload, WalmartSubmitAction)
	if e != nil {
		return OfficialCompletion{}, e
	}
	body.Kind = "publish"
	body.RecordID = saved.ID
	body.ProductKey = saved.Source.Source.ProductKey
	body.Walmart = &WalmartFeedReceipt{FeedID: accepted.FeedID, CorrelationID: accepted.CorrelationID, WireHash: intent.Walmart.WireHash, Selection: intent.Walmart.Selection, SKUs: intent.Walmart.SKUs, RepairFrom: intent.Walmart.RepairFrom}
	body.ResponseHash = collection.Digest(body.Walmart)
	return sealOfficialCompletion(claim, body)
}
func validateWalmartReceipt(v OfficialReceipt) error {
	w := v.Walmart
	if w == nil || v.Kind != "publish" || !collection.ValidID(v.RecordID) || !authidentity.IsBoundedIdentifier(v.ProductKey) || w.FeedID == "" || len(w.FeedID) > 256 || uuid.Validate(w.CorrelationID) != nil || !isExecutionDigest(w.WireHash) || !w.Selection.Valid() || !walmartSKUsValid(w.SKUs) || v.Image != nil || v.TemuImage != nil || v.TemuProduct != nil || v.TikTok != nil || !reflect.DeepEqual(v.Product, model.PublishResult{}) || v.ResponseHash != collection.Digest(w) {
		return ErrExecutionEvidenceRequired
	}
	subject, e := WalmartSubjectID(v.ProductKey, v.Binding.SupplierIdentityHash)
	if e != nil || subject != v.Target.SubjectID {
		return ErrExecutionEvidenceRequired
	}
	return nil
}
func WalmartReceiptMatchesIntent(r OfficialReceipt, i OfficialIntent) bool {
	if r.Walmart == nil || i.Walmart == nil {
		return false
	}
	return r.RecordID == i.RecordID && r.ProductKey == i.Source.Source.ProductKey && r.Walmart.Selection == i.Walmart.Selection && r.Walmart.WireHash == i.Walmart.WireHash && reflect.DeepEqual(r.Walmart.SKUs, i.Walmart.SKUs) && r.Walmart.RepairFrom == i.Walmart.RepairFrom && r.Walmart.CorrelationID == WalmartCorrelationID(i)
}

type WalmartResultRequest struct {
	Receipt     OfficialReceipt
	Generation  int64
	TrackedSKUs []string
}
type WalmartResultRecord struct {
	ID             string                      `json:"id"`
	ReceiptID      string                      `json:"receiptId"`
	Generation     int64                       `json:"generation"`
	Latest         bool                        `json:"latest"`
	RepairableSKUs []string                    `json:"repairableSKUs,omitempty"`
	Observation    marketplace.FeedObservation `json:"observation"`
}
type WalmartResultRepository interface {
	BeginWalmartRead(context.Context, collection.Scope, string) (WalmartResultRequest, error)
	SaveWalmartRead(context.Context, collection.Scope, WalmartResultRequest, marketplace.FeedObservation) (WalmartResultRecord, error)
	ReadWalmartResults(context.Context, collection.Scope, ExecutionTarget) ([]WalmartResultRecord, error)
}
