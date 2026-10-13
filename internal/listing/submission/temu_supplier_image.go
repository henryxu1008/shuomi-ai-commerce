package submission

import (
	"context"
	"encoding/json"
	"task-processor/internal/listing/preparation"
	record "task-processor/internal/listing/record/target"
	temugoods "task-processor/internal/marketplace/temu/goods"
	"task-processor/internal/marketplace/temu/model"
	"task-processor/internal/product/collection"
	"task-processor/internal/storecenter"
)

// The durable manifest contains only original approved facts and fixed E43
// options. The actual base64 exists only for one permitted provider send.
type TemuSupplierImageManifest struct {
	Image        temugoods.ImageObservation `json:"image"`
	BusinessType int64                      `json:"imageBizType"`
	Options      model.SupplierImageOptions `json:"options"`
}

func ValidTemuSupplierImageManifest(v TemuSupplierImageManifest) bool {
	biz, err := TemuSupplierImageBusinessType(v.Image.Usage)
	return err == nil && biz == v.BusinessType && temugoods.ValidImageObservation(v.Image) && v.Image.Bytes <= 2<<20 && len(v.Image.RemoteURLs) == 0 && v.Image.ResponseHash == "" && v.Options.CategoryID == v.Image.CatID && !v.Options.IntelligenceCrop && !v.Options.Boost && v.Options.SizeMode == 0
}

func PrepareTemuSupplierImageIntent(ctx context.Context, source preparation.AuthorizedSource, saved record.TargetRecord, observed temugoods.ImageObservation) (OfficialIntentCommit, TemuSupplierImageManifest, error) {
	var manifest TemuSupplierImageManifest
	biz, err := TemuSupplierImageBusinessType(observed.Usage)
	if err != nil || saved.Merchant.MerchantMode == storecenter.MerchantLocal {
		return OfficialIntentCommit{}, manifest, ErrExecutionEvidenceRequired
	}
	manifest = TemuSupplierImageManifest{Image: observed, BusinessType: biz, Options: model.SupplierImageOptions{CategoryID: observed.CatID}}
	if !ValidTemuSupplierImageManifest(manifest) {
		return OfficialIntentCommit{}, manifest, ErrExecutionEvidenceRequired
	}
	selected := false
	for _, v := range saved.TemuImageObservations {
		if collection.Digest(v) == collection.Digest(observed) {
			selected = true
		}
	}
	if !selected {
		return OfficialIntentCommit{}, manifest, ErrExecutionEvidenceRequired
	}
	subject, err := TemuImageSubjectID(observed, saved.Merchant)
	if err != nil {
		return OfficialIntentCommit{}, manifest, err
	}
	raw, err := json.Marshal(manifest)
	if err != nil {
		return OfficialIntentCommit{}, manifest, err
	}
	key := "temu-supplier-image-" + digestExecutionParts(subject, saved.ID, string(raw), saved.Merchant.ApplicationRevision)
	body, err := temuIntentBase(ctx, source, saved, key)
	if err != nil {
		return OfficialIntentCommit{}, manifest, err
	}
	body.Kind, body.Target.SubjectID, body.TemuImage = "image", subject, &observed
	body.PayloadFingerprint = digestExecutionParts("temu", body.Target.StoreID, subject, TemuSupplierImageAction, string(raw))
	proof, err := sealTemuIntent(source, body)
	return proof, manifest, err
}

func SupplierUploadedImageURL(v model.SupplierImageUploaded) (string, error) {
	urls := append(append([]string{}, v.URLs...), v.ImageURL, v.URL)
	definite := ""
	for _, url := range urls {
		if url == "" {
			continue
		}
		if definite != "" && definite != url {
			return "", ErrExecutionEvidenceRequired
		}
		definite = url
	}
	if definite == "" {
		return "", ErrExecutionEvidenceRequired
	}
	return definite, nil
}

func NewTemuSupplierImageCompletion(owner collection.Scope, b storecenter.ProductMerchantBinding, a ExecutionAttempt, claim ExecutionClaim, manifest TemuSupplierImageManifest, result model.SupplierImageUploaded, responseHash string, artifact temugoods.ImageArtifact) (OfficialCompletion, error) {
	if b.MerchantMode == storecenter.MerchantLocal || !ValidTemuSupplierImageManifest(manifest) || !isExecutionDigest(responseHash) || !temugoods.ValidImageArtifact(artifact) {
		return OfficialCompletion{}, ErrExecutionEvidenceRequired
	}
	url, err := SupplierUploadedImageURL(result)
	if err != nil || url != artifact.URL {
		return OfficialCompletion{}, ErrExecutionEvidenceRequired
	}
	raw, err := json.Marshal(manifest)
	if err != nil {
		return OfficialCompletion{}, err
	}
	body, err := completionBase(owner, b, a, claim, raw, TemuSupplierImageAction)
	if err != nil {
		return OfficialCompletion{}, err
	}
	subject, err := TemuImageSubjectID(manifest.Image, b)
	if err != nil || subject != a.Target.SubjectID {
		return OfficialCompletion{}, ErrExecutionEvidenceRequired
	}
	image := manifest.Image
	image.RemoteURLs, image.ResponseHash = []string{url}, responseHash
	body.Kind, body.ResponseHash = "image", responseHash
	body.TemuImage = &TemuImageReceipt{Image: image, AttemptCreatedAt: a.CreatedAt, FenceEpoch: a.FenceEpoch, SupplierArtifact: &artifact}
	return sealOfficialCompletion(claim, body)
}
