package submission

import (
	"context"
	"encoding/json"
	"reflect"
	"time"

	"task-processor/internal/authidentity"
	"task-processor/internal/listing/preparation"
	record "task-processor/internal/listing/record/target"
	marketplace "task-processor/internal/marketplace/tiktokshop"
	"task-processor/internal/product/collection"
	"task-processor/internal/storecenter"
)

const (
	TikTokImageStage         = "image"
	TikTokGlobalCreateStage  = "global_create"
	TikTokPublishStage       = "publish"
	TikTokImageAction        = "tiktok_upload_image"
	TikTokGlobalCreateAction = "tiktok_create_global"
	TikTokPublishAction      = "tiktok_publish_us"
)

type TikTokReceiptReference struct {
	ID   string `json:"id"`
	Hash string `json:"hash"`
}
type TikTokStageIntent struct {
	DraftHash string                        `json:"draftHash"`
	Image     *marketplace.ImageObservation `json:"image,omitempty"`
	Images    []TikTokReceiptReference      `json:"images,omitempty"`
	Global    *TikTokReceiptReference       `json:"global,omitempty"`
}
type TikTokEffectReceipt struct {
	DraftHash     string                            `json:"draftHash"`
	ResponseHash  string                            `json:"responseHash,omitempty"`
	Image         *marketplace.ImageObservation     `json:"image,omitempty"`
	Uploaded      *marketplace.UploadedImage        `json:"uploaded,omitempty"`
	Global        *marketplace.CreatedGlobalProduct `json:"global,omitempty"`
	Local         *marketplace.CreatedLocalProduct  `json:"local,omitempty"`
	LocalSaveMode string                            `json:"localSaveMode,omitempty"`
	Publication   *marketplace.GlobalPublishResult  `json:"publication,omitempty"`
}

func TikTokStageAction(stage string) string {
	switch stage {
	case TikTokImageStage:
		return TikTokImageAction
	case TikTokGlobalCreateStage:
		return TikTokGlobalCreateAction
	case TikTokPublishStage:
		return TikTokPublishAction
	}
	return ""
}
func TikTokProductSubjectID(productKey, stage string) (string, error) {
	if !authidentity.IsBoundedIdentifier(productKey) || stage != TikTokGlobalCreateStage && stage != TikTokPublishStage {
		return "", ErrExecutionInvalid
	}
	return "tiktok-" + stage + "-us-" + digestExecutionParts(productKey, "US", stage), nil
}
func TikTokImageSubjectID(image marketplace.ImageObservation, b storecenter.ProductMerchantBinding) (string, error) {
	if !isExecutionDigest(image.ContentHash) || !isExecutionDigest(b.SupplierIdentityHash) || !authidentity.IsBoundedIdentifier(b.ApplicationID) || !marketplace.ValidImageDimensions(image.UseCase, 1024, 1024) {
		return "", ErrExecutionInvalid
	}
	return "tiktok-image-us-" + digestExecutionParts(image.ContentHash, image.UseCase, b.SupplierIdentityHash, b.ApplicationID), nil
}
func TikTokCompatibleBinding(a, b storecenter.ProductMerchantBinding) bool {
	return a.ValidTikTokApplication() && b.ValidTikTokApplication() && a.OrganizationID == b.OrganizationID && a.StoreID == b.StoreID && a.ApplicationID == b.ApplicationID && a.SupplierIdentityHash == b.SupplierIdentityHash && a.TikTok.ShopID == b.TikTok.ShopID
}
func TikTokDraftHash(saved record.TargetRecord) string {
	return collection.Digest([]any{saved.Input.TikTok, saved.Source.Source.ProductKey, saved.EffectiveVersion, saved.ProductHash, saved.InventoryHash, saved.RulesHash, saved.TikTokImageObservations})
}
func TikTokReceiptRef(receipt OfficialReceipt) TikTokReceiptReference {
	return TikTokReceiptReference{ID: receipt.ID, Hash: collection.Digest(receipt)}
}
func TikTokReceiptMatchesIntent(receipt OfficialReceipt, intent OfficialIntent) bool {
	return receipt.TikTok != nil && intent.TikTok != nil && receipt.RecordID == intent.RecordID && receipt.ProductKey == intent.Source.Source.ProductKey && receipt.TikTok.DraftHash == intent.TikTok.DraftHash && (intent.Kind != TikTokImageStage || reflect.DeepEqual(receipt.TikTok.Image, intent.TikTok.Image))
}
func TikTokPayloadMatchesIntent(intent OfficialIntent, payload []byte) bool {
	return ValidateOfficialIntent(intent) == nil && intent.PayloadFingerprint == digestExecutionParts("tiktokshop", intent.Target.StoreID, intent.Target.SubjectID, TikTokStageAction(intent.Kind), string(payload))
}
func validTikTokRef(ref TikTokReceiptReference) bool {
	return validExecutionAttemptID(ref.ID) && isExecutionDigest(ref.Hash)
}

// The immutable saved input owns every product field. Only proven image URIs
// and exact created global SKU identities may be substituted for dispatch.
func ResolveTikTokPayload(saved record.TargetRecord, stage, key string, image *marketplace.ImageObservation, receipts []OfficialReceipt, global *OfficialReceipt) ([]byte, *TikTokStageIntent, error) {
	if !record.PublicationReady(saved) || saved.Input.TikTok == nil || TikTokStageAction(stage) == "" {
		return nil, nil, ErrExecutionEvidenceRequired
	}
	effect := &TikTokStageIntent{DraftHash: TikTokDraftHash(saved)}
	var payload any
	if stage == TikTokImageStage {
		if image == nil || !marketplace.ValidImageObservation(*image) || len(receipts) != 0 || global != nil {
			return nil, nil, ErrExecutionEvidenceRequired
		}
		found := false
		for _, selected := range saved.TikTokImageObservations {
			if selected == *image {
				found = true
			}
		}
		if !found {
			return nil, nil, ErrExecutionEvidenceRequired
		}
		copy := *image
		effect.Image = &copy
		payload = copy
	} else if stage == TikTokPublishStage && saved.Input.TikTok.Selection.Method == marketplace.GlobalPublishing {
		if image != nil || len(receipts) != 0 || global == nil || ValidateOfficialReceipt(*global) != nil || global.Kind != TikTokGlobalCreateStage || global.TikTok == nil || global.TikTok.Global == nil || !TikTokCompatibleBinding(global.Binding, saved.Merchant) || global.TikTok.DraftHash != effect.DraftHash {
			return nil, nil, ErrExecutionEvidenceRequired
		}
		ref := TikTokReceiptRef(*global)
		effect.Global = &ref
		built, err := marketplace.BuildGlobalPublish(*saved.Input.TikTok, *global.TikTok.Global)
		if err != nil {
			return nil, nil, ErrExecutionEvidenceRequired
		}
		payload = built
	} else {
		if image != nil || global != nil || len(receipts) != len(saved.TikTokImageObservations) {
			return nil, nil, ErrExecutionEvidenceRequired
		}
		refs := []marketplace.ImageReference{}
		used := map[string]bool{}
		for _, selected := range saved.TikTokImageObservations {
			subject, err := TikTokImageSubjectID(selected, saved.Merchant)
			if err != nil {
				return nil, nil, err
			}
			var found *OfficialReceipt
			for i := range receipts {
				r := &receipts[i]
				if ValidateOfficialReceipt(*r) == nil && r.Kind == TikTokImageStage && r.Target.SubjectID == subject && TikTokCompatibleBinding(r.Binding, saved.Merchant) && r.TikTok != nil && r.TikTok.Image != nil && *r.TikTok.Image == selected {
					if found != nil {
						return nil, nil, ErrExecutionEvidenceRequired
					}
					found = r
				}
			}
			if found == nil || used[found.ID] {
				return nil, nil, ErrExecutionEvidenceRequired
			}
			used[found.ID] = true
			effect.Images = append(effect.Images, TikTokReceiptRef(*found))
			refs = append(refs, marketplace.ImageReference{AssetID: selected.AssetID, Image: *found.TikTok.Uploaded})
		}
		external, err := TikTokProductSubjectID(saved.Source.Source.ProductKey, stage)
		if err != nil {
			return nil, nil, err
		}
		if stage == TikTokGlobalCreateStage {
			built, e := marketplace.BuildGlobalCreate(*saved.Input.TikTok, refs, external)
			if e != nil {
				return nil, nil, ErrExecutionEvidenceRequired
			}
			payload = built
		} else {
			built, e := marketplace.BuildLocalCreate(*saved.Input.TikTok, refs, external, key)
			if e != nil {
				return nil, nil, ErrExecutionEvidenceRequired
			}
			payload = built
		}
	}
	raw, err := json.Marshal(payload)
	if err != nil || len(raw) > MaxExecutionPayloadBytes {
		return nil, nil, ErrExecutionEvidenceRequired
	}
	return raw, effect, nil
}
func PrepareTikTokIntent(ctx context.Context, sourceProof preparation.AuthorizedSource, saved record.TargetRecord, key, stage string, image *marketplace.ImageObservation, receipts []OfficialReceipt, global *OfficialReceipt) (OfficialIntentCommit, []byte, error) {
	scope, source, _, err := sourceProof.Read(ctx)
	if err != nil || saved.Source != source || saved.Merchant.OrganizationID != scope.OrganizationID || !authidentity.IsBoundedIdentifier(key) {
		return OfficialIntentCommit{}, nil, ErrExecutionEvidenceRequired
	}
	payload, effect, err := ResolveTikTokPayload(saved, stage, key, image, receipts, global)
	if err != nil {
		return OfficialIntentCommit{}, nil, err
	}
	subject, err := TikTokProductSubjectID(source.Source.ProductKey, stage)
	if stage == TikTokImageStage {
		subject, err = TikTokImageSubjectID(*image, saved.Merchant)
	}
	if err != nil {
		return OfficialIntentCommit{}, nil, err
	}
	body := OfficialIntent{Key: key, Owner: scope, Kind: stage, RecordID: saved.ID, RecordHash: collection.Digest(saved), Source: source, Binding: saved.Merchant, Target: ExecutionTarget{Platform: "tiktokshop", StoreID: saved.Merchant.StoreID, SubjectID: subject}, TikTok: effect, CreatedAt: time.Now().UTC().Truncate(time.Microsecond)}
	body.PayloadFingerprint = digestExecutionParts(body.Target.Platform, body.Target.StoreID, subject, TikTokStageAction(stage), string(payload))
	body.InputHash = OfficialIntentInputHash(body)
	copied, err := cloneOfficialIntent(body)
	if err != nil || ValidateOfficialIntent(copied) != nil {
		return OfficialIntentCommit{}, nil, ErrExecutionEvidenceRequired
	}
	return OfficialIntentCommit{source: sourceProof, intent: copied, hash: collection.Digest(copied)}, payload, nil
}
func validateTikTokIntent(i OfficialIntent) error {
	if i.TikTok == nil || !isExecutionDigest(i.TikTok.DraftHash) || i.Image != nil || i.TemuImage != nil || len(i.TemuImages) > 0 || i.TemuRenewalFrom != "" {
		return ErrExecutionEvidenceRequired
	}
	var subject string
	var err error
	switch i.Kind {
	case TikTokImageStage:
		if i.TikTok.Image == nil || !marketplace.ValidImageObservation(*i.TikTok.Image) || len(i.TikTok.Images) != 0 || i.TikTok.Global != nil {
			return ErrExecutionEvidenceRequired
		}
		subject, err = TikTokImageSubjectID(*i.TikTok.Image, i.Binding)
	case TikTokGlobalCreateStage, TikTokPublishStage:
		if i.TikTok.Image != nil || len(i.TikTok.Images) > 1000 || len(i.TikTok.Images) == 0 && i.TikTok.Global == nil || len(i.TikTok.Images) > 0 && i.TikTok.Global != nil || i.Kind == TikTokGlobalCreateStage && i.TikTok.Global != nil {
			return ErrExecutionEvidenceRequired
		}
		seen := map[string]bool{}
		for _, ref := range i.TikTok.Images {
			if !validTikTokRef(ref) || seen[ref.ID] {
				return ErrExecutionEvidenceRequired
			}
			seen[ref.ID] = true
		}
		if i.TikTok.Global != nil && !validTikTokRef(*i.TikTok.Global) {
			return ErrExecutionEvidenceRequired
		}
		subject, err = TikTokProductSubjectID(i.Source.Source.ProductKey, i.Kind)
	default:
		return ErrExecutionEvidenceRequired
	}
	if err != nil || i.Target.SubjectID != subject {
		return ErrExecutionEvidenceRequired
	}
	return nil
}
func NewTikTokCompletion(owner collection.Scope, saved record.TargetRecord, intent OfficialIntent, attempt ExecutionAttempt, claim ExecutionClaim, payload []byte, effect TikTokEffectReceipt) (OfficialCompletion, error) {
	if ValidateOfficialIntent(intent) != nil || intent.Owner != owner || intent.RecordID != saved.ID || intent.RecordHash != collection.Digest(saved) || effect.DraftHash != intent.TikTok.DraftHash {
		return OfficialCompletion{}, ErrExecutionEvidenceRequired
	}
	body, err := completionBase(owner, intent.Binding, attempt, claim, payload, TikTokStageAction(intent.Kind))
	if err != nil {
		return OfficialCompletion{}, err
	}
	if body.Target != intent.Target || body.IntentKey != intent.Key || body.PayloadFingerprint != intent.PayloadFingerprint {
		return OfficialCompletion{}, ErrExecutionEvidenceRequired
	}
	body.Kind, body.RecordID, body.ProductKey, body.TikTok = intent.Kind, saved.ID, saved.Source.Source.ProductKey, &effect
	switch intent.Kind {
	case TikTokImageStage:
		if effect.Image == nil || !reflect.DeepEqual(effect.Image, intent.TikTok.Image) {
			return OfficialCompletion{}, ErrExecutionEvidenceRequired
		}
	case TikTokGlobalCreateStage:
		if effect.Global == nil || !marketplace.CorrelatesCreatedSKUs(effect.Global.SKUs, *saved.Input.TikTok) {
			return OfficialCompletion{}, ErrExecutionEvidenceRequired
		}
		body.ResponseHash = effect.Global.ResponseHash
	case TikTokPublishStage:
		if effect.Local != nil {
			if !marketplace.CorrelatesCreatedSKUs(effect.Local.SKUs, *saved.Input.TikTok) {
				return OfficialCompletion{}, ErrExecutionEvidenceRequired
			}
			body.ResponseHash = effect.Local.ResponseHash
			effect.LocalSaveMode = saved.Input.TikTok.SaveMode
		} else if effect.Publication != nil {
			var sent marketplace.PublishGlobalRequest
			if json.Unmarshal(payload, &sent) != nil || len(sent.Targets) != 1 || sent.Targets[0].Region != "US" || !marketplace.CorrelatesGlobalPublication(*effect.Publication, sent, intent.Binding.TikTok.ShopID) {
				return OfficialCompletion{}, ErrExecutionEvidenceRequired
			}
			body.ResponseHash = effect.Publication.Fingerprint
		}
	}
	if intent.Kind == TikTokImageStage { // Image transport supplies the exact raw response hash separately.
		if effect.Uploaded == nil {
			return OfficialCompletion{}, ErrExecutionEvidenceRequired
		}
		effect.ResponseHash = effect.Uploaded.ResponseHash
		body.ResponseHash = effect.ResponseHash
	}
	return sealOfficialCompletion(claim, body)
}
func validateTikTokReceipt(r OfficialReceipt) error {
	e := r.TikTok
	if e == nil || !collection.ValidID(r.RecordID) || !authidentity.IsBoundedIdentifier(r.ProductKey) || !isExecutionDigest(e.DraftHash) || r.Image != nil || r.TemuProduct != nil || r.TemuImage != nil || r.Product.SPUName != "" || len(r.Product.SKCs) > 0 {
		return ErrExecutionEvidenceRequired
	}
	var subject string
	var err error
	switch r.Kind {
	case TikTokImageStage:
		if e.Image == nil || e.Uploaded == nil || !marketplace.ValidImageObservation(*e.Image) || !marketplace.ValidUploadedImage(*e.Uploaded) || e.Image.UseCase != e.Uploaded.UseCase || e.Global != nil || e.Local != nil || e.Publication != nil || e.ResponseHash != r.ResponseHash {
			return ErrExecutionEvidenceRequired
		}
		subject, err = TikTokImageSubjectID(*e.Image, r.Binding)
	case TikTokGlobalCreateStage:
		if e.Image != nil || e.Uploaded != nil || e.Global == nil || !marketplace.ValidCreatedGlobal(*e.Global) || e.Local != nil || e.Publication != nil || e.Global.ResponseHash != r.ResponseHash {
			return ErrExecutionEvidenceRequired
		}
		subject, err = TikTokProductSubjectID(r.ProductKey, r.Kind)
	case TikTokPublishStage:
		if e.Image != nil || e.Uploaded != nil || e.Global != nil || (e.Local == nil) == (e.Publication == nil) {
			return ErrExecutionEvidenceRequired
		}
		if e.Local != nil && (!marketplace.ValidCreatedLocal(*e.Local) || e.Local.ResponseHash != r.ResponseHash || e.LocalSaveMode != "AS_DRAFT" && e.LocalSaveMode != "LISTING") {
			return ErrExecutionEvidenceRequired
		}
		if e.Publication != nil && (!marketplace.ValidGlobalPublishResult(*e.Publication, r.Binding.TikTok.ShopID) || e.Publication.Fingerprint != r.ResponseHash) {
			return ErrExecutionEvidenceRequired
		}
		subject, err = TikTokProductSubjectID(r.ProductKey, r.Kind)
	default:
		return ErrExecutionEvidenceRequired
	}
	if err != nil || subject != r.Target.SubjectID {
		return ErrExecutionEvidenceRequired
	}
	return nil
}
