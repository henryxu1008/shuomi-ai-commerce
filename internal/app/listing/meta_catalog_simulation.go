package listingapp

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/henryxu1008/shuomi-ai-commerce/internal/listing/preparation"
	record "github.com/henryxu1008/shuomi-ai-commerce/internal/listing/record/target"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/listing/submission"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/product/asset"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/product/catalog"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/product/collection"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/storecenter"
	meta "github.com/henryxu1008/shuomi-ai-commerce/internal/storecenter/metacatalog"
)

type MetaPublication struct {
	Authorization preparation.Authorizer
	Sources       record.TargetSourceSelector
	Assets        record.ApprovedAssetReader
	Stores        *storecenter.MetaCatalogSimulation
	Plans         submission.MetaPlanRepository
	Kernel        UploadKernel
}

type MetaPrepareInput struct {
	SourceID   string `json:"sourceId"`
	StoreID    string `json:"storeId"`
	ImageID    string `json:"imageId"`
	VariantID  string `json:"variantId"`
	ProductURL string `json:"productUrl"`
	Price      string `json:"price"`
	Brand      string `json:"brand"`
}

type MetaPublicationResult struct {
	ID          string                       `json:"id"`
	Status      submission.ExecutionStatus   `json:"status"`
	RemoteID    string                       `json:"remoteId,omitempty"`
	Observation *storecenter.MetaObservation `json:"observation,omitempty"`
}

type MetaSourceView struct {
	Title    string                  `json:"title"`
	Brand    string                  `json:"brand"`
	Variants []GoogleVariantChoice   `json:"variants"`
	Images   []GoogleImageChoice     `json:"images"`
	Records  []submission.MetaRecord `json:"records"`
}

func (s *MetaPublication) source(ctx context.Context, id string, write bool) (collection.Scope, catalog.PublishedSnapshot, error) {
	var scope collection.Scope
	var product catalog.PublishedSnapshot
	if s == nil || s.Authorization == nil || s.Sources == nil || s.Assets == nil || s.Stores == nil || s.Plans == nil || s.Kernel == nil || !collection.ValidID(id) {
		return scope, product, preparation.ErrInvalid
	}
	permissions := []string{preparation.PermissionRead}
	if write {
		permissions = append(permissions, preparation.PermissionManage, preparation.PermissionSubmit)
	}
	for _, permission := range permissions {
		current, err := s.Authorization.Authorize(ctx, permission)
		if err != nil || current.Validate() != nil || scope.OrganizationID != "" && scope != current {
			return scope, product, preparation.ErrForbidden
		}
		scope = current
	}
	selection, err := s.Sources.Select(ctx, id)
	if err != nil {
		return scope, product, err
	}
	owner, item, product, err := selection.Read(ctx)
	if err != nil {
		return scope, product, err
	}
	if owner != scope || item.Source.Kind != "own" || collection.IsResearchFixtureProduct(product.Identity.ProductKey) {
		return scope, product, preparation.ErrForbidden
	}
	return scope, product, nil
}

func (s *MetaPublication) approvedAssets(ctx context.Context, scope collection.Scope, p catalog.PublishedSnapshot) (asset.ApprovedAssetInventory, error) {
	return s.Assets.GetApprovedInventory(ctx, asset.InventoryScope{
		TenantID: scope.OrganizationID, ProductKey: p.Identity.ProductKey, SourceSnapshotVersion: p.Version, TargetPlatform: "product",
	})
}

func (s *MetaPublication) Options(ctx context.Context, sourceID string) (MetaSourceView, error) {
	scope, p, err := s.source(ctx, sourceID, false)
	if err != nil {
		return MetaSourceView{}, err
	}
	records, err := s.Plans.List(ctx, scope, sourceID)
	if err != nil {
		return MetaSourceView{}, err
	}
	view := MetaSourceView{Title: p.Snapshot.Title, Brand: p.Snapshot.Brand, Variants: googleVariants(p), Images: []GoogleImageChoice{}, Records: records}
	if inventory, err := s.approvedAssets(ctx, scope, p); err == nil {
		for _, image := range inventory.Assets {
			if image.Role == asset.RoleMain {
				view.Images = append(view.Images, GoogleImageChoice{ID: image.ID, URL: image.URL})
			}
		}
	}
	return view, nil
}

func (s *MetaPublication) Prepare(ctx context.Context, key string, input MetaPrepareInput) (submission.MetaPlan, error) {
	micros, valid := googlePrice(input.Price)
	price := micros / 10000
	_, fraction, _ := strings.Cut(input.Price, ".")
	valid = valid && len(fraction) <= 2
	if !collection.ValidID(key) || !collection.ValidID(input.StoreID) || !valid || input.VariantID == "" {
		return submission.MetaPlan{}, preparation.ErrInvalid
	}
	scope, p, err := s.source(ctx, input.SourceID, true)
	if err != nil {
		return submission.MetaPlan{}, err
	}
	var variantLabel string
	for _, choice := range googleVariants(p) {
		if choice.ID == input.VariantID {
			variantLabel = choice.Label
			break
		}
	}
	if variantLabel == "" {
		return submission.MetaPlan{}, preparation.ErrInvalid
	}
	inventory, err := s.approvedAssets(ctx, scope, p)
	if err != nil {
		return submission.MetaPlan{}, err
	}
	var imageURL string
	for _, image := range inventory.Assets {
		if image.ID == input.ImageID && image.Role == asset.RoleMain {
			imageURL = image.URL
			break
		}
	}
	if imageURL == "" {
		return submission.MetaPlan{}, record.ErrNotReady
	}
	binding, err := s.Stores.PublicationBinding(ctx, scope.OrganizationID, input.StoreID)
	if err != nil {
		return submission.MetaPlan{}, metaStoreError(err)
	}
	brand := strings.TrimSpace(p.Snapshot.Brand)
	if brand == "" {
		brand = strings.TrimSpace(input.Brand)
	}
	title := strings.TrimSpace(p.Snapshot.Title)
	if len(p.Snapshot.Variants) > 1 && title != variantLabel {
		title += " · " + variantLabel
	}
	plan := submission.MetaPlan{
		ID: key, Owner: scope, SourceID: input.SourceID, StoreID: input.StoreID, Version: p.Version, Generation: binding.Generation,
		ProductHash: collection.Digest(p.Snapshot), InventoryHash: collection.Digest(inventory), CatalogID: binding.CatalogID,
		BusinessID: binding.BusinessID, WhatsAppID: binding.WhatsAppID, CreatedAt: time.Now().UTC().Truncate(time.Microsecond),
		VariantLabel: variantLabel,
		Product: meta.Product{
			RetailerID: "sm-" + collection.Digest([]string{binding.CatalogID, scope.OrganizationID, p.Identity.ProductKey, input.VariantID}),
			Title:      title, Description: p.Snapshot.Description, ProductURL: strings.TrimSpace(input.ProductURL),
			ImageURL: imageURL, Brand: brand, PriceCents: price,
		},
	}
	if !plan.Valid() {
		return submission.MetaPlan{}, preparation.ErrInvalid
	}
	return s.Plans.Save(ctx, plan)
}

func (s *MetaPublication) load(ctx context.Context, sourceID, id string, write bool) (submission.MetaPlan, error) {
	if !collection.ValidID(id) {
		return submission.MetaPlan{}, preparation.ErrInvalid
	}
	scope, _, err := s.source(ctx, sourceID, write)
	if err != nil {
		return submission.MetaPlan{}, err
	}
	plan, err := s.Plans.Read(ctx, scope, id)
	if err != nil {
		return plan, err
	}
	if plan.SourceID != sourceID {
		return submission.MetaPlan{}, submission.ErrExecutionNotFound
	}
	return plan, nil
}

func metaIntent(p submission.MetaPlan) string {
	return collection.StableID(p.Owner.OrganizationID, submission.MetaCatalogCreateAction, p.ID)
}

func (s *MetaPublication) Confirm(ctx context.Context, sourceID, id string) (MetaPublicationResult, error) {
	p, err := s.load(ctx, sourceID, id, true)
	if err != nil {
		return MetaPublicationResult{}, err
	}
	scope := submission.ExecutionScope{OrganizationID: p.Owner.OrganizationID}
	existing, err := s.Kernel.ReadIntent(ctx, scope, metaIntent(p))
	if err == nil {
		return s.resultFromAttempt(ctx, p, existing, false), nil
	}
	if !errors.Is(err, submission.ErrExecutionNotFound) {
		return MetaPublicationResult{}, err
	}
	binding := storecenter.MetaBinding{CatalogID: p.CatalogID, BusinessID: p.BusinessID, WhatsAppID: p.WhatsAppID, Generation: p.Generation}
	var acquisition submission.ExecutionAcquisition
	remote, sendErr := s.Stores.WithPublication(ctx, scope.OrganizationID, p.StoreID, binding, p.Product, func(sendCtx context.Context) (bool, error) {
		current, product, err := s.source(sendCtx, p.SourceID, true)
		if err != nil {
			return false, err
		}
		inventory, err := s.approvedAssets(sendCtx, current, product)
		if err != nil {
			return false, err
		}
		if current != p.Owner || p.Version != product.Version || p.ProductHash != collection.Digest(product.Snapshot) || p.InventoryHash != collection.Digest(inventory) {
			return false, record.ErrConflict
		}
		payload, err := json.Marshal([]any{p.Version, p.ProductHash, p.InventoryHash, p.Generation, p.CatalogID, p.BusinessID, p.WhatsAppID, p.Product})
		if err != nil {
			return false, err
		}
		acquisition, err = s.Kernel.Acquire(sendCtx, submission.AcquireExecutionCommand{
			Scope: scope, IntentKey: metaIntent(p), Action: submission.MetaCatalogCreateAction, Payload: payload, ClaimOwnerID: p.Owner.ActorID,
			Target: submission.ExecutionTarget{Platform: string(storecenter.PlatformMetaCatalog), StoreID: p.StoreID, SubjectID: p.Product.RetailerID}, Lease: 30 * time.Second,
		})
		return acquisition.Permit != nil, err
	})
	if acquisition.Permit == nil {
		if sendErr != nil {
			return MetaPublicationResult{}, metaStoreError(sendErr)
		}
		return s.resultFromAttempt(ctx, p, acquisition.Attempt, false), nil
	}
	finish, cancel := context.WithTimeout(context.WithoutCancel(ctx), 4*time.Second)
	defer cancel()
	claim := submission.ExecutionClaim{Scope: scope, AttemptID: acquisition.Permit.AttemptID, FenceEpoch: acquisition.Permit.FenceEpoch,
		OwnerID: acquisition.Permit.ClaimOwnerID, Token: acquisition.Permit.ClaimToken}
	if sendErr == nil || errors.Is(sendErr, meta.ErrRejected) {
		evidence := submission.ExecutionEvidence{Kind: submission.EvidenceProviderResponse, Outcome: submission.ExecutionSucceeded,
			Reference: p.Product.RetailerID, Fingerprint: collection.Digest(remote), Reason: remote, ObservedAt: time.Now().UTC()}
		if sendErr != nil {
			evidence.Outcome, evidence.Reason = submission.ExecutionFailedDefinitive, "catalog_rejected"
			evidence.Fingerprint = collection.Digest(evidence.Reason)
		}
		if saved, err := s.Kernel.Complete(finish, claim, evidence); err == nil {
			return s.resultFromAttempt(ctx, p, saved, false), nil
		}
	}
	_, _ = s.Kernel.MarkUnknown(finish, claim, submission.UnknownResponseLost)
	return MetaPublicationResult{ID: id, Status: submission.ExecutionOutcomeUnknown}, nil
}

func (s *MetaPublication) resultFromAttempt(ctx context.Context, p submission.MetaPlan, attempt submission.ExecutionAttempt, observe bool) MetaPublicationResult {
	out := MetaPublicationResult{ID: p.ID, Status: attempt.Status}
	if attempt.Status == submission.ExecutionClaimed && !time.Now().Before(attempt.LeaseExpiresAt) {
		if expired, err := s.Kernel.Expire(ctx, submission.ExecutionScope{OrganizationID: p.Owner.OrganizationID}, attempt.AttemptID); err == nil {
			out.Status = expired.Status
		}
	}
	if attempt.Status == submission.ExecutionSucceeded && attempt.Evidence != nil && meta.ValidID(attempt.Evidence.Reason) {
		out.RemoteID = attempt.Evidence.Reason
		if observe {
			binding := storecenter.MetaBinding{CatalogID: p.CatalogID, BusinessID: p.BusinessID, WhatsAppID: p.WhatsAppID, Generation: p.Generation}
			observation := s.Stores.Observe(ctx, p.Owner.OrganizationID, p.StoreID, binding, p.Product.RetailerID, out.RemoteID)
			out.Observation = &observation
		}
	}
	return out
}

func (s *MetaPublication) Result(ctx context.Context, sourceID, id string) (MetaPublicationResult, error) {
	p, err := s.load(ctx, sourceID, id, false)
	if err != nil {
		return MetaPublicationResult{}, err
	}
	attempt, err := s.Kernel.ReadIntent(ctx, submission.ExecutionScope{OrganizationID: p.Owner.OrganizationID}, metaIntent(p))
	if errors.Is(err, submission.ErrExecutionNotFound) {
		return MetaPublicationResult{ID: id, Status: submission.ExecutionNotSent}, nil
	}
	if err != nil {
		return MetaPublicationResult{}, err
	}
	return s.resultFromAttempt(ctx, p, attempt, true), nil
}

func metaStoreError(err error) error {
	switch {
	case errors.Is(err, storecenter.ErrNotFound):
		return preparation.ErrForbidden
	case errors.Is(err, storecenter.ErrVersionConflict):
		return record.ErrConflict
	case errors.Is(err, storecenter.ErrOfficialConnectionUnavailable):
		return record.ErrNotReady
	default:
		return err
	}
}
