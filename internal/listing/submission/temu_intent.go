package submission

import (
	"bytes"
	"context"
	"encoding/json"
	"task-processor/internal/listing/preparation"
	record "task-processor/internal/listing/record/target"
	temugoods "task-processor/internal/marketplace/temu/goods"
	"task-processor/internal/marketplace/temu/model"
	"task-processor/internal/product/asset"
	"task-processor/internal/product/collection"
	"task-processor/internal/storecenter"
	"time"
)

type TemuImageReference struct {
	AssetID     string `json:"assetId"`
	Usage       int64  `json:"usage"`
	ReceiptID   string `json:"receiptId"`
	ReceiptHash string `json:"receiptHash"`
}

// ResolveTemuPublicationPayload changes only selected image positions using
// immutable, correlated, unexpired image receipts. The saved native body stays
// the only owner of all product fields.
func ResolveTemuPublicationPayload(saved record.TargetRecord, receipts []OfficialReceipt, now time.Time) ([]byte, []TemuImageReference, error) {
	if !record.PublicationReady(saved) || saved.Input.Temu == nil || now.IsZero() {
		return nil, nil, ErrExecutionEvidenceRequired
	}
	images := []temugoods.ImageObservation{}
	refs := []TemuImageReference{}
	inventory := asset.ApprovedAssetInventory{Scope: asset.InventoryScope{TenantID: saved.Merchant.OrganizationID, ProductKey: saved.Source.Source.ProductKey, TargetPlatform: "temu", SourceSnapshotVersion: saved.EffectiveVersion}}
	for _, selected := range saved.TemuImageObservations {
		var found *OfficialReceipt
		for i := range receipts {
			r := &receipts[i]
			if ValidateOfficialReceipt(*r) != nil || r.Kind != "image" || r.TemuImage == nil || r.Target.Platform != "temu" || r.Binding.OrganizationID != saved.Merchant.OrganizationID || r.Binding.StoreID != saved.Merchant.StoreID || r.Binding.SupplierIdentityHash != saved.Merchant.SupplierIdentityHash || saved.Merchant.MerchantMode == storecenter.MerchantLocal && !now.Before(r.TemuImage.UseBefore) || saved.Merchant.MerchantMode == storecenter.MerchantLocal != (r.Binding.MerchantMode == storecenter.MerchantLocal) {
				continue
			}
			remote := r.TemuImage.Image
			expectedSubject, subjectErr := TemuImageSubjectID(selected, saved.Merchant)
			if subjectErr != nil || r.Target.SubjectID != expectedSubject || remote.ContentHash != selected.ContentHash || remote.CatID != selected.CatID || saved.Merchant.MerchantMode == storecenter.MerchantLocal && remote.Usage != selected.Usage || remote.Width != selected.Width || remote.Height != selected.Height || remote.Bytes != selected.Bytes || remote.MediaType != selected.MediaType {
				continue
			}
			if found != nil {
				return nil, nil, ErrExecutionEvidenceRequired
			}
			found = r
		}
		if found == nil {
			return nil, nil, ErrExecutionEvidenceRequired
		}
		selected.RemoteURLs = append([]string(nil), found.TemuImage.Image.RemoteURLs...)
		selected.ResponseHash = found.ResponseHash
		images = append(images, selected)
		inventory.Assets = append(inventory.Assets, asset.ApprovedAsset{ID: selected.AssetID, URL: selected.SourceURL, Width: selected.Width, Height: selected.Height})
		refs = append(refs, TemuImageReference{AssetID: selected.AssetID, Usage: selected.Usage, ReceiptID: found.ID, ReceiptHash: collection.Digest(*found)})
	}
	result := temugoods.Build(*saved.Input.Temu, *saved.TemuRuleSnapshot, inventory, images)
	if !result.ReadyForUpload || len(result.SubmissionPayload) == 0 {
		return nil, nil, ErrExecutionEvidenceRequired
	}
	return append([]byte(nil), result.SubmissionPayload...), refs, nil
}

func PrepareTemuPublishIntent(ctx context.Context, sourceProof preparation.AuthorizedSource, saved record.TargetRecord, key string, payload []byte, receipts []OfficialReceipt) (OfficialIntentCommit, error) {
	body, err := temuIntentBase(ctx, sourceProof, saved, key)
	if err != nil {
		return OfficialIntentCommit{}, err
	}
	resolved, refs, err := ResolveTemuPublicationPayload(saved, receipts, body.CreatedAt)
	if err != nil || !bytes.Equal(payload, resolved) {
		return OfficialIntentCommit{}, ErrExecutionEvidenceRequired
	}
	body.TemuImages = refs
	body.Target.SubjectID, err = TemuProductSubjectID(saved.Source.Source.ProductKey, saved.Merchant)
	if err != nil {
		return OfficialIntentCommit{}, err
	}
	body.PayloadFingerprint = digestExecutionParts("temu", body.Target.StoreID, body.Target.SubjectID, TemuPublishAction, string(payload))
	return sealTemuIntent(sourceProof, body)
}

func PrepareTemuImageIntent(ctx context.Context, sourceProof preparation.AuthorizedSource, saved record.TargetRecord, observed temugoods.ImageObservation, prior *OfficialReceipt) (OfficialIntentCommit, model.LocalImageUpload, error) {
	var input model.LocalImageUpload
	if !temugoods.ValidImageObservation(observed) || len(observed.RemoteURLs) > 0 || observed.ResponseHash != "" || saved.Merchant.MerchantMode != storecenter.MerchantLocal {
		return OfficialIntentCommit{}, input, ErrExecutionEvidenceRequired
	}
	selected := false
	for _, v := range saved.TemuImageObservations {
		if collection.Digest(v) == collection.Digest(observed) {
			selected = true
		}
	}
	if !selected {
		return OfficialIntentCommit{}, input, ErrExecutionEvidenceRequired
	}
	subject, err := TemuLocalImageSubjectID(observed.ContentHash, observed.CatID, observed.Usage, saved.Merchant)
	if err != nil {
		return OfficialIntentCommit{}, input, err
	}
	renewal := ""
	if prior != nil {
		if ValidateOfficialReceipt(*prior) != nil || prior.TemuImage == nil || prior.Target.SubjectID != subject || prior.Target.StoreID != saved.Merchant.StoreID || prior.Binding.OrganizationID != saved.Merchant.OrganizationID || prior.Binding.SupplierIdentityHash != saved.Merchant.SupplierIdentityHash {
			return OfficialIntentCommit{}, input, ErrExecutionEvidenceRequired
		}
		renewal = prior.ID
	}
	input = model.LocalImageUpload{FileURL: observed.SourceURL, CatID: observed.CatID, Usage: observed.Usage}
	raw, err := json.Marshal(input)
	if err != nil {
		return OfficialIntentCommit{}, input, ErrExecutionEvidenceRequired
	}
	key := "temu-image-" + digestExecutionParts(subject, renewal, saved.ID, string(raw), saved.Merchant.ApplicationRevision)
	body, err := temuIntentBase(ctx, sourceProof, saved, key)
	if err != nil {
		return OfficialIntentCommit{}, input, err
	}
	body.Kind = "image"
	body.Target.SubjectID = subject
	body.TemuImage = &observed
	body.TemuRenewalFrom = renewal
	body.PayloadFingerprint = digestExecutionParts("temu", body.Target.StoreID, subject, TemuLocalImageAction, string(raw))
	proof, err := sealTemuIntent(sourceProof, body)
	return proof, input, err
}

func temuIntentBase(ctx context.Context, sourceProof preparation.AuthorizedSource, saved record.TargetRecord, key string) (OfficialIntent, error) {
	scope, source, _, err := sourceProof.Read(ctx)
	if err != nil || source != saved.Source || scope.OrganizationID != saved.Merchant.OrganizationID || !record.PublicationReady(saved) || saved.Input.Temu == nil || !collection.ValidID(saved.ID) {
		return OfficialIntent{}, ErrExecutionEvidenceRequired
	}
	return OfficialIntent{Key: key, Owner: scope, Kind: "publish", RecordID: saved.ID, RecordHash: collection.Digest(saved), Source: source, Target: ExecutionTarget{Platform: "temu", StoreID: saved.Merchant.StoreID}, Binding: saved.Merchant, CreatedAt: time.Now().UTC().Truncate(time.Microsecond)}, nil
}

func sealTemuIntent(source preparation.AuthorizedSource, body OfficialIntent) (OfficialIntentCommit, error) {
	body.InputHash = OfficialIntentInputHash(body)
	copy, err := cloneOfficialIntent(body)
	if err != nil || ValidateOfficialIntent(copy) != nil {
		return OfficialIntentCommit{}, ErrExecutionEvidenceRequired
	}
	return OfficialIntentCommit{source: source, intent: copy, hash: collection.Digest(copy)}, nil
}

func validateTemuIntent(v OfficialIntent) error {
	if v.Image != nil || v.TikTok != nil {
		return ErrExecutionEvidenceRequired
	}
	if v.Kind == "publish" {
		subject, err := TemuProductSubjectID(v.Source.Source.ProductKey, v.Binding)
		if err != nil || subject != v.Target.SubjectID || v.TemuImage != nil || v.TemuRenewalFrom != "" || len(v.TemuImages) < 1 || len(v.TemuImages) > 1000 {
			return ErrExecutionEvidenceRequired
		}
		seen := map[struct {
			asset string
			usage int64
		}]bool{}
		for _, r := range v.TemuImages {
			key := struct {
				asset string
				usage int64
			}{r.AssetID, r.Usage}
			if r.AssetID == "" || r.Usage < 1 || r.Usage > 7 || !validExecutionAttemptID(r.ReceiptID) || !isExecutionDigest(r.ReceiptHash) || seen[key] {
				return ErrExecutionEvidenceRequired
			}
			seen[key] = true
		}
		return nil
	}
	if v.Kind != "image" || v.TemuImage == nil || len(v.TemuImages) > 0 || !temugoods.ValidImageObservation(*v.TemuImage) || len(v.TemuImage.RemoteURLs) > 0 || v.TemuImage.ResponseHash != "" || v.TemuRenewalFrom != "" && (!validExecutionAttemptID(v.TemuRenewalFrom) || v.Binding.MerchantMode != storecenter.MerchantLocal) {
		return ErrExecutionEvidenceRequired
	}
	subject, err := TemuImageSubjectID(*v.TemuImage, v.Binding)
	if err != nil || subject != v.Target.SubjectID {
		return ErrExecutionEvidenceRequired
	}
	return nil
}
