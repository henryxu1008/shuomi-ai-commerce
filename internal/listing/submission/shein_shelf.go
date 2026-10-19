package submission

import (
	"context"
	"encoding/json"
	"task-processor/internal/authidentity"
	"task-processor/internal/listing/preparation"
	record "task-processor/internal/listing/record/target"
	"task-processor/internal/marketplace/shein/goods"
	"task-processor/internal/marketplace/shein/model"
	"task-processor/internal/product/collection"
	"task-processor/internal/storecenter"
	"time"
)

const SheinShelfAction = "shelf_on"

type SheinShelfIntent struct {
	OriginalReceiptID   string    `json:"originalReceiptId"`
	OriginalReceiptHash string    `json:"originalReceiptHash"`
	SKC                 string    `json:"skc"`
	Version             string    `json:"version"`
	ObservationHash     string    `json:"observationHash"`
	ConfirmationToken   string    `json:"confirmationToken"`
	ExpiresAt           time.Time `json:"expiresAt"`
}
type ShelfIntentCommit struct {
	source preparation.AuthorizedSource
	intent OfficialIntent
	hash   string
}

func (p ShelfIntentCommit) Read(ctx context.Context) (OfficialIntent, error) {
	owner, source, _, e := p.source.Read(ctx)
	if e != nil || owner != p.intent.Owner || source != p.intent.Source || ValidateOfficialIntent(p.intent) != nil || p.hash != collection.Digest(p.intent) {
		return OfficialIntent{}, ErrExecutionEvidenceRequired
	}
	return cloneOfficialIntent(p.intent)
}

type ShelfIntentRepository interface {
	PrepareShelfIntent(context.Context, ShelfIntentCommit) (OfficialIntent, error)
	ReadOfficialIntent(context.Context, collection.Scope, string) (OfficialIntent, error)
}

func ShelfSubjectID(skc, site string) (string, error) {
	if !authidentity.IsBoundedIdentifier(skc) || site != "shein-us" {
		return "", ErrExecutionInvalid
	}
	return "shein-shelf-on-us-v1-" + digestExecutionParts(skc, site), nil
}
func CompatibleShelfBinding(original, current storecenter.ProductMerchantBinding) bool {
	return (current.Platform == "" || current.Platform == storecenter.PlatformShein) && original.OrganizationID == current.OrganizationID && original.StoreID == current.StoreID && original.Site == "shein-us" && current.Site == "shein-us" && original.SupplierIdentityHash == current.SupplierIdentityHash && original.ApplicationID == current.ApplicationID && original.ApplicationType == current.ApplicationType && current.ValidApplication()
}
func OriginalShelfPlan(saved record.TargetRecord, original OfficialReceipt, skc string) (model.ProductSKC, error) {
	if ValidateOfficialReceipt(original) != nil || original.Kind != "publish" || original.Target.Platform != "shein" || original.RecordID != saved.ID || original.ProductKey != saved.Source.Source.ProductKey || original.Binding != saved.Merchant || !authidentity.IsBoundedIdentifier(original.Product.Version) || !goods.CorrelatesPublishedMembership(saved.Result.Product, original.Product) {
		return model.ProductSKC{}, ErrExecutionEvidenceRequired
	}
	for _, remote := range original.Product.SKCs {
		if remote.SKCName == skc {
			for _, local := range saved.Result.Product.SKCs {
				if len(local.SKUs) == len(remote.SKUs) {
					match := true
					for _, r := range remote.SKUs {
						found := false
						for _, l := range local.SKUs {
							found = found || l.SupplierSKU == r.SupplierSKU
						}
						match = match && found
					}
					if match {
						return local, nil
					}
				}
			}
		}
	}
	return model.ProductSKC{}, ErrExecutionEvidenceRequired
}
func NewShelfIntent(ctx context.Context, source preparation.AuthorizedSource, saved record.TargetRecord, original OfficialReceipt, binding storecenter.ProductMerchantBinding, key, skc, confirmation, observation string, now time.Time) (ShelfIntentCommit, error) {
	owner, item, _, e := source.Read(ctx)
	if e != nil || owner != original.Owner || item != saved.Source || !CompatibleShelfBinding(original.Binding, binding) || !authidentity.IsBoundedIdentifier(key) || !collection.ValidID(confirmation) || !isExecutionDigest(observation) {
		return ShelfIntentCommit{}, ErrExecutionEvidenceRequired
	}
	if _, e = OriginalShelfPlan(saved, original, skc); e != nil {
		return ShelfIntentCommit{}, e
	}
	target, e := ShelfSubjectID(skc, "shein-us")
	if e != nil {
		return ShelfIntentCommit{}, e
	}
	body := OfficialIntent{Key: key, Owner: owner, Kind: "shelf_on", RecordID: saved.ID, RecordHash: collection.Digest(saved), Source: item, Target: ExecutionTarget{Platform: "shein", StoreID: binding.StoreID, SubjectID: target}, Binding: binding, CreatedAt: now.UTC().Truncate(time.Microsecond), Shelf: &SheinShelfIntent{OriginalReceiptID: original.ID, OriginalReceiptHash: collection.Digest(original), SKC: skc, Version: original.Product.Version, ObservationHash: observation, ConfirmationToken: confirmation, ExpiresAt: now.UTC().Truncate(time.Microsecond).Add(5 * time.Minute)}}
	raw := ShelfPayload(body)
	body.PayloadFingerprint = digestExecutionParts(body.Target.Platform, body.Target.StoreID, body.Target.SubjectID, SheinShelfAction, string(raw))
	body.InputHash = OfficialIntentInputHash(body)
	if ValidateOfficialIntent(body) != nil {
		return ShelfIntentCommit{}, ErrExecutionEvidenceRequired
	}
	return ShelfIntentCommit{source: source, intent: body, hash: collection.Digest(body)}, nil
}
func ShelfPayload(v OfficialIntent) []byte {
	raw, _ := json.Marshal([]any{v.Kind, v.Owner, v.RecordID, v.RecordHash, v.Source, v.Target, v.Binding, v.Shelf})
	return raw
}
func validateShelfIntent(v OfficialIntent) error {
	s := v.Shelf
	if v.Target.Platform != "shein" || v.Kind != "shelf_on" || s == nil || !collection.ValidID(s.OriginalReceiptID) || !isExecutionDigest(s.OriginalReceiptHash) || !authidentity.IsBoundedIdentifier(s.Version) || !collection.ValidID(s.ConfirmationToken) || !isExecutionDigest(s.ObservationHash) || !s.ExpiresAt.Equal(v.CreatedAt.Add(5*time.Minute)) || v.Binding.ApplicationType == storecenter.OfficialApplicationType(model.ModeFullyManaged) || v.Image != nil || v.TikTok != nil || v.Walmart != nil || v.TemuImage != nil || len(v.TemuImages) != 0 || v.TemuRenewalFrom != "" {
		return ErrExecutionEvidenceRequired
	}
	subject, e := ShelfSubjectID(s.SKC, v.Binding.Site)
	if e != nil || subject != v.Target.SubjectID || v.PayloadFingerprint != digestExecutionParts(v.Target.Platform, v.Target.StoreID, v.Target.SubjectID, SheinShelfAction, string(ShelfPayload(v))) {
		return ErrExecutionEvidenceRequired
	}
	return nil
}
func ShelfCooldownReady(prior ExecutionAttempt, now time.Time) bool {
	if prior.Status == ExecutionClaimed || prior.Status == ExecutionOutcomeUnknown || prior.Status == ExecutionSucceeded {
		return false
	}
	upper := prior.LeaseExpiresAt
	if prior.FinishedAt != nil {
		upper = *prior.FinishedAt
	}
	return !upper.IsZero() && now.After(upper.Add(20*time.Minute))
}
func NewShelfCompletion(intent OfficialIntent, attempt ExecutionAttempt, claim ExecutionClaim, result model.ShelfResult) (OfficialCompletion, error) {
	if ValidateOfficialIntent(intent) != nil || intent.Shelf == nil || result.Response != nil || result.SKC != intent.Shelf.SKC || result.Site != "shein-us" || !isExecutionDigest(result.ResponseHash) || attempt.IntentKey != intent.Key {
		return OfficialCompletion{}, ErrExecutionEvidenceRequired
	}
	body, e := completionBase(intent.Owner, intent.Binding, attempt, claim, ShelfPayload(intent), SheinShelfAction)
	if e != nil {
		return OfficialCompletion{}, e
	}
	body.Kind = "shelf_on"
	body.RecordID = intent.RecordID
	body.Shelf = &result
	body.ResponseHash = result.ResponseHash
	return sealOfficialCompletion(claim, body)
}
func validateShelfReceipt(v OfficialReceipt) error {
	if v.Shelf == nil || v.Shelf.Response != nil || v.TemuProduct != nil || v.TemuImage != nil || v.TikTok != nil || v.Walmart != nil || v.Kind != "shelf_on" || v.Target.Platform != "shein" || !collection.ValidID(v.RecordID) || v.ProductKey != "" || v.Image != nil || v.Product.SPUName != "" || len(v.Product.SKCs) > 0 || v.Shelf.ResponseHash != v.ResponseHash {
		return ErrExecutionEvidenceRequired
	}
	s, e := ShelfSubjectID(v.Shelf.SKC, v.Shelf.Site)
	if e != nil || s != v.Target.SubjectID {
		return ErrExecutionEvidenceRequired
	}
	return nil
}
