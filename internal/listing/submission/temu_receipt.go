package submission

import (
	"bytes"
	"encoding/json"
	temugoods "task-processor/internal/marketplace/temu/goods"
	"task-processor/internal/marketplace/temu/model"
	"task-processor/internal/product/collection"
	"task-processor/internal/storecenter"
	"time"
)

const TemuLocalImageLifetime = 7 * 24 * time.Hour

const TemuImageSendWindow = 10 * time.Second

type TemuImageReceipt struct {
	Image            temugoods.ImageObservation `json:"image"`
	AttemptCreatedAt time.Time                  `json:"attemptCreatedAt"`
	FenceEpoch       int64                      `json:"fenceEpoch"`
	UseBefore        time.Time                  `json:"useBefore"`
	SupplierArtifact *temugoods.ImageArtifact   `json:"supplierArtifact,omitempty"`
}

func validOfficialBindingForTarget(b storecenter.ProductMerchantBinding, t ExecutionTarget) bool {
	if b.StoreID != t.StoreID {
		return false
	}
	if t.Platform == "temu" {
		return b.ValidTemuApplication()
	}
	if t.Platform == "tiktokshop" {
		return b.ValidTikTokApplication()
	}
	return t.Platform == "shein" && b.Platform != storecenter.PlatformTemu && b.Site == "shein-us" && b.ValidApplication()
}

func NewTemuImageCompletion(owner collection.Scope, b storecenter.ProductMerchantBinding, a ExecutionAttempt, claim ExecutionClaim, input model.LocalImageUpload, observed temugoods.ImageObservation, result model.LocalImageUploaded, responseHash string) (OfficialCompletion, error) {
	if b.MerchantMode != storecenter.MerchantLocal || b.Platform != storecenter.PlatformTemu || !temugoods.ValidImageObservation(observed) || len(observed.RemoteURLs) != 0 || observed.ResponseHash != "" || observed.SourceURL != input.FileURL || observed.CatID != input.CatID || observed.Usage != input.Usage || !isExecutionDigest(responseHash) || len(result.Images) < 1 || len(result.Images) > 1000 {
		return OfficialCompletion{}, ErrExecutionEvidenceRequired
	}
	raw, err := json.Marshal(input)
	if err != nil {
		return OfficialCompletion{}, ErrExecutionEvidenceRequired
	}
	body, err := completionBase(owner, b, a, claim, raw, TemuLocalImageAction)
	if err != nil {
		return OfficialCompletion{}, err
	}
	subject, err := TemuLocalImageSubjectID(observed.ContentHash, input.CatID, input.Usage, b)
	if err != nil || subject != a.Target.SubjectID {
		return OfficialCompletion{}, ErrExecutionEvidenceRequired
	}
	seen := map[string]bool{}
	for _, image := range result.Images {
		if seen[image.URL] {
			return OfficialCompletion{}, ErrExecutionEvidenceRequired
		}
		seen[image.URL] = true
		observed.RemoteURLs = append(observed.RemoteURLs, image.URL)
	}
	observed.ResponseHash = responseHash
	body.Kind = "image"
	body.ResponseHash = responseHash
	body.TemuImage = &TemuImageReceipt{Image: observed, AttemptCreatedAt: a.CreatedAt, FenceEpoch: a.FenceEpoch, UseBefore: a.CreatedAt.Add(TemuLocalImageLifetime)}
	return sealOfficialCompletion(claim, body)
}

func NewTemuPublishCompletion(owner collection.Scope, recordID, productKey string, b storecenter.ProductMerchantBinding, a ExecutionAttempt, claim ExecutionClaim, raw []byte, result temugoods.CreatedGoods) (OfficialCompletion, error) {
	if b.Platform != storecenter.PlatformTemu || !isExecutionDigest(result.ResponseHash) || !collection.ValidID(recordID) || len(raw) < 1 || len(raw) > MaxExecutionPayloadBytes {
		return OfficialCompletion{}, ErrExecutionEvidenceRequired
	}
	var canonical []byte
	if b.MerchantMode == storecenter.MerchantLocal {
		var sent model.LocalGoodsV2
		if result.Local == nil || result.Supplier != nil || json.Unmarshal(raw, &sent) != nil || !temugoods.CorrelatesLocalCreation(sent, *result.Local) {
			return OfficialCompletion{}, ErrExecutionEvidenceRequired
		}
		canonical, _ = json.Marshal(sent)
	} else {
		var sent model.SupplierGoods
		if result.Supplier == nil || result.Local != nil || json.Unmarshal(raw, &sent) != nil || !temugoods.CorrelatesSupplierCreation(sent, *result.Supplier) {
			return OfficialCompletion{}, ErrExecutionEvidenceRequired
		}
		canonical, _ = json.Marshal(sent)
	}
	if !bytes.Equal(raw, canonical) {
		return OfficialCompletion{}, ErrExecutionEvidenceRequired
	}
	body, err := completionBase(owner, b, a, claim, raw, TemuPublishAction)
	if err != nil {
		return OfficialCompletion{}, err
	}
	subject, err := TemuProductSubjectID(productKey, b)
	if err != nil || subject != a.Target.SubjectID {
		return OfficialCompletion{}, ErrExecutionEvidenceRequired
	}
	body.Kind, body.RecordID, body.ProductKey, body.TemuProduct, body.ResponseHash = "publish", recordID, productKey, &result, result.ResponseHash
	return sealOfficialCompletion(claim, body)
}

func validateTemuReceipt(v OfficialReceipt) error {
	if v.TikTok != nil || v.Image != nil || v.Product.SPUName != "" || len(v.Product.SKCs) > 0 || v.Product.ResponseHash != "" {
		return ErrExecutionEvidenceRequired
	}
	if v.Kind == "publish" {
		subject, err := TemuProductSubjectID(v.ProductKey, v.Binding)
		if err != nil || subject != v.Target.SubjectID || !collection.ValidID(v.RecordID) || v.TemuImage != nil || v.TemuProduct == nil || !temugoods.ValidCreatedGoods(*v.TemuProduct) || v.TemuProduct.ResponseHash != v.ResponseHash || v.Binding.MerchantMode == storecenter.MerchantLocal != (v.TemuProduct.Local != nil) {
			return ErrExecutionEvidenceRequired
		}
		return nil
	}
	if v.Kind != "image" || v.TemuImage == nil || v.TemuProduct != nil || v.RecordID != "" || v.ProductKey != "" {
		return ErrExecutionEvidenceRequired
	}
	fact := v.TemuImage
	if !temugoods.ValidImageObservation(fact.Image) || len(fact.Image.RemoteURLs) < 1 || fact.Image.ResponseHash != v.ResponseHash || fact.AttemptCreatedAt.IsZero() || fact.FenceEpoch < 1 || v.ObservedAt.Before(fact.AttemptCreatedAt) {
		return ErrExecutionEvidenceRequired
	}
	if v.Binding.MerchantMode == storecenter.MerchantLocal {
		if fact.SupplierArtifact != nil || !fact.UseBefore.Equal(fact.AttemptCreatedAt.Add(TemuLocalImageLifetime)) {
			return ErrExecutionEvidenceRequired
		}
	} else if !fact.UseBefore.IsZero() || fact.SupplierArtifact == nil || !temugoods.ValidImageArtifact(*fact.SupplierArtifact) || len(fact.Image.RemoteURLs) != 1 || fact.Image.RemoteURLs[0] != fact.SupplierArtifact.URL || fact.SupplierArtifact.Width != fact.Image.Width || fact.SupplierArtifact.Height != fact.Image.Height {
		return ErrExecutionEvidenceRequired
	}
	subject, err := TemuImageSubjectID(fact.Image, v.Binding)
	if err != nil || subject != v.Target.SubjectID {
		return ErrExecutionEvidenceRequired
	}
	return nil
}
