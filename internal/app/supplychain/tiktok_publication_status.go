package supplychainapp

import (
	"context"
	"time"

	"task-processor/internal/listing/preparation"
	record "task-processor/internal/listing/record/target"
	"task-processor/internal/listing/submission"
	marketplace "task-processor/internal/marketplace/tiktokshop"
	"task-processor/internal/storecenter"
)

type TikTokStatusObservation struct {
	marketplace.ProductStatus
	ObservedAt time.Time `json:"observedAt"`
}

// Reads only an original confirmed publication. It never resolves an UNKNOWN
// claim, changes a durable receipt, or grants another send.
func (a *Application) TikTokPublicationStatus(ctx context.Context, sourceID, storeID string) (PublicationView, error) {
	view, err := a.Publication(ctx, sourceID, storeID)
	if err != nil {
		return view, err
	}
	if view.TikTok == nil || a.TikTokStatusStores == nil {
		return view, record.ErrNotReady
	}
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	scope, err := a.Authorization.Authorize(ctx, preparation.PermissionRead)
	if err != nil {
		return view, err
	}
	proof, err := a.Sources.Select(ctx, sourceID)
	if err != nil {
		return view, err
	}
	owner, source, _, err := proof.Read(ctx)
	if err != nil || owner != scope {
		return view, preparation.ErrForbidden
	}
	binding, err := a.TargetMerchants.ReadTargetMerchant(ctx, scope, storeID)
	if err != nil || binding != view.binding || binding.Platform != storecenter.PlatformTikTokShop || !validTargetMerchant(scope, storeID, binding) {
		return view, record.ErrForbidden
	}
	expected := marketplace.ProductStatusExpectation{}
	if view.TikTok.Local != nil {
		expected.ProductID = view.TikTok.Local.ID
		for _, sku := range view.TikTok.Local.SKUs {
			expected.LocalSKUIDs = append(expected.LocalSKUIDs, sku.ID)
		}
	} else if view.TikTok.Publication != nil && view.TikTok.Publication.ProductID != "" {
		expected.ProductID = view.TikTok.Publication.ProductID
		expected.Mappings = view.TikTok.Publication.SKUs
		for _, sku := range expected.Mappings {
			expected.LocalSKUIDs = append(expected.LocalSKUIDs, sku.LocalSKUID)
		}
		subject, e := submission.TikTokProductSubjectID(source.Source.ProductKey, submission.TikTokGlobalCreateStage)
		if e != nil {
			return view, e
		}
		global, e := a.PublicationReceipts.FindOfficialTarget(ctx, scope.OrganizationID, submission.ExecutionTarget{Platform: "tiktokshop", StoreID: storeID, SubjectID: subject})
		if e != nil {
			return view, e
		}
		if submission.ValidateOfficialReceipt(global) != nil || global.Kind != submission.TikTokGlobalCreateStage || global.ProductKey != source.Source.ProductKey || !submission.TikTokCompatibleBinding(global.Binding, binding) || global.TikTok == nil || global.TikTok.Global == nil || global.TikTok.DraftHash != view.TikTok.DraftHash {
			return view, record.ErrConflict
		}
		expected.GlobalProductID = global.TikTok.Global.ID
	}
	if expected.ProductID == "" {
		return view, record.ErrNotReady
	}
	merchant, err := a.TikTokStatusStores.TikTokMerchant(ctx, scope, storeID, &binding)
	if err != nil {
		return view, err
	}
	reader, ok := merchant.(interface {
		ProductStatus(context.Context, marketplace.ProductStatusExpectation) (marketplace.ProductStatus, error)
	})
	if !ok || merchant.Binding() != binding {
		return view, record.ErrUnavailable
	}
	status, err := reader.ProductStatus(ctx, expected)
	if err != nil {
		return view, err
	}
	if status.ProductID != expected.ProductID {
		return view, record.ErrConflict
	}
	if _, _, _, err = proof.Read(ctx); err != nil {
		return view, err
	}
	fresh, err := a.TargetMerchants.ReadTargetMerchant(ctx, scope, storeID)
	if err != nil || fresh != binding {
		return view, record.ErrForbidden
	}
	view.TikTokStatus = &TikTokStatusObservation{ProductStatus: status, ObservedAt: time.Now().UTC()}
	return view, nil
}
