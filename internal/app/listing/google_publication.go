package listingapp

import (
	"context"
	"encoding/json"
	"errors"
	"strconv"
	"strings"
	"time"

	storeapp "github.com/henryxu1008/shuomi-ai-commerce/internal/app/storecenter"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/listing/preparation"
	record "github.com/henryxu1008/shuomi-ai-commerce/internal/listing/record/target"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/listing/submission"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/product/asset"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/product/catalog"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/product/collection"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/storecenter"
	market "github.com/henryxu1008/shuomi-ai-commerce/internal/storecenter/googlemerchant"
)

type GooglePublicationStore interface {
	PublicationBinding(context.Context, string, string) (storeapp.GooglePublicationBinding, error)
	Publish(context.Context, string, string, storeapp.GooglePublicationBinding, market.Offer, func(context.Context) error) (market.Receipt, error)
	ReadProduct(context.Context, string, string, string) (market.Observation, error)
}

type GooglePublicationKernel interface {
	UploadKernel
	PrepareGoogleExecution(context.Context, submission.AcquireExecutionCommand) (submission.PreparedAcquisition, error)
	BeginGoogleSend(context.Context, *submission.PreparedSend) (*submission.SendPermit, error)
	StopGoogleBeforeSend(context.Context, *submission.PreparedSend, string) (submission.ExecutionAttempt, error)
}

type GooglePublication struct {
	Authorization preparation.Authorizer
	Sources       record.TargetSourceSelector
	Assets        record.ApprovedAssetReader
	Stores        GooglePublicationStore
	Plans         submission.GooglePlanRepository
	Kernel        GooglePublicationKernel
}

type GoogleProductInput struct {
	VariantID    string `json:"variantId"`
	Link         string `json:"link"`
	Price        string `json:"price"`
	Currency     string `json:"currency"`
	Availability string `json:"availability"`
	Condition    string `json:"condition"`
	Color        string `json:"color"`
	Size         string `json:"size"`
}

type GooglePrepareInput struct {
	SourceID string               `json:"sourceId"`
	StoreID  string               `json:"storeId"`
	ImageID  string               `json:"imageId"`
	Variants []GoogleProductInput `json:"variants"`
}

type GooglePublicationResult struct {
	ID    string             `json:"id"`
	Items []GoogleItemResult `json:"items"`
}

type GoogleItemResult struct {
	OfferID       string                     `json:"offerId"`
	Status        submission.ExecutionStatus `json:"status"`
	RemoteProduct string                     `json:"remoteProduct,omitempty"`
	Observation   *market.Observation        `json:"observation,omitempty"`
}

func (s *GooglePublication) available() bool {
	return s != nil && s.Authorization != nil && s.Sources != nil && s.Assets != nil &&
		s.Stores != nil && s.Plans != nil && s.Kernel != nil
}

func (s *GooglePublication) source(ctx context.Context, id string, write, requireAssets bool) (collection.Scope, catalog.PublishedSnapshot, asset.ApprovedAssetInventory, error) {
	if !s.available() || !collection.ValidID(id) {
		return collection.Scope{}, catalog.PublishedSnapshot{}, asset.ApprovedAssetInventory{}, preparation.ErrInvalid
	}
	required := []string{preparation.PermissionRead}
	if write {
		required = append(required, preparation.PermissionManage, preparation.PermissionSubmit)
	}
	var scope collection.Scope
	for _, permission := range required {
		current, err := s.Authorization.Authorize(ctx, permission)
		if err != nil || current.Validate() != nil || scope.OrganizationID != "" && current != scope {
			return scope, catalog.PublishedSnapshot{}, asset.ApprovedAssetInventory{}, preparation.ErrForbidden
		}
		scope = current
	}
	selected, err := s.Sources.Select(ctx, id)
	if err != nil {
		return scope, catalog.PublishedSnapshot{}, asset.ApprovedAssetInventory{}, err
	}
	owner, item, product, err := selected.Read(ctx)
	if err != nil || owner != scope || item.Source.Kind != "own" || collection.IsResearchFixtureProduct(product.Identity.ProductKey) {
		return scope, product, asset.ApprovedAssetInventory{}, preparation.ErrForbidden
	}
	if !requireAssets {
		return scope, product, asset.ApprovedAssetInventory{}, nil
	}
	inventory, err := s.approvedAssets(ctx, scope, product)
	return scope, product, inventory, err
}

func (s *GooglePublication) approvedAssets(ctx context.Context, scope collection.Scope, product catalog.PublishedSnapshot) (asset.ApprovedAssetInventory, error) {
	return s.Assets.GetApprovedInventory(ctx, asset.InventoryScope{
		TenantID: scope.OrganizationID, ProductKey: product.Identity.ProductKey,
		SourceSnapshotVersion: product.Version, TargetPlatform: "product",
	})
}

type GoogleSourceView struct {
	Title    string                    `json:"title"`
	Variants []GoogleVariantChoice     `json:"variants"`
	Records  []submission.GoogleRecord `json:"records"`
	Images   []GoogleImageChoice       `json:"images"`
}

type GoogleVariantChoice struct {
	ID    string `json:"id"`
	Label string `json:"label"`
}

type GoogleImageChoice struct {
	ID  string `json:"id"`
	URL string `json:"url"`
}

func (s *GooglePublication) Options(ctx context.Context, sourceID string) (GoogleSourceView, error) {
	scope, product, _, err := s.source(ctx, sourceID, false, false)
	if err != nil {
		return GoogleSourceView{}, err
	}
	records, err := s.Plans.List(ctx, scope, sourceID)
	if err != nil {
		return GoogleSourceView{}, err
	}
	view := GoogleSourceView{Title: product.Snapshot.Title, Variants: googleVariants(product), Images: []GoogleImageChoice{}, Records: records}
	inventory, err := s.approvedAssets(ctx, scope, product)
	if err != nil {
		return view, nil
	}
	for _, image := range inventory.Assets {
		view.Images = append(view.Images, GoogleImageChoice{ID: image.ID, URL: image.URL})
	}
	return view, nil
}

func googleVariants(product catalog.PublishedSnapshot) []GoogleVariantChoice {
	if len(product.Snapshot.Variants) == 0 {
		return []GoogleVariantChoice{{ID: "default", Label: product.Snapshot.Title}}
	}
	result := make([]GoogleVariantChoice, 0, len(product.Snapshot.Variants))
	seen := map[string]bool{}
	for _, variant := range product.Snapshot.Variants {
		key := variant.SourceID
		if key == "" {
			key = variant.SKU
		}
		if key == "" || seen[key] {
			return nil
		}
		seen[key] = true
		label := variant.Title
		if label == "" {
			label = variant.SKU
		}
		if label == "" {
			label = key
		}
		result = append(result, GoogleVariantChoice{ID: key, Label: label})
	}
	return result
}

func (s *GooglePublication) Prepare(ctx context.Context, key string, input GooglePrepareInput) (submission.GooglePlan, error) {
	if !collection.ValidID(key) || !collection.ValidID(input.StoreID) || len(input.Variants) < 1 || len(input.Variants) > 10 {
		return submission.GooglePlan{}, preparation.ErrInvalid
	}
	scope, product, inventory, err := s.source(ctx, input.SourceID, true, true)
	if err != nil {
		return submission.GooglePlan{}, err
	}
	binding, err := s.Stores.PublicationBinding(ctx, scope.OrganizationID, input.StoreID)
	if err != nil {
		return submission.GooglePlan{}, err
	}
	var imageURL string
	for _, image := range inventory.Assets {
		if image.ID == input.ImageID {
			imageURL = image.URL
			break
		}
	}
	if imageURL == "" {
		return submission.GooglePlan{}, record.ErrNotReady
	}
	choices := googleVariants(product)
	allowed := map[string]bool{}
	for _, choice := range choices {
		allowed[choice.ID] = true
	}
	plan := submission.GooglePlan{ID: key, Owner: scope, SourceID: input.SourceID, StoreID: input.StoreID, Version: product.Version,
		ProductHash: collection.Digest(product.Snapshot), InventoryHash: collection.Digest(inventory), Target: binding.Target,
		Generation: binding.Generation, CreatedAt: time.Now().UTC().Truncate(time.Microsecond)}
	for _, item := range input.Variants {
		if !allowed[item.VariantID] {
			return submission.GooglePlan{}, preparation.ErrInvalid
		}
		amount, ok := googlePrice(item.Price)
		if !ok {
			return submission.GooglePlan{}, preparation.ErrInvalid
		}
		offerID, err := plan.Target.OfferID(collection.StableID(product.Identity.ProductKey, item.VariantID))
		if err != nil {
			return submission.GooglePlan{}, err
		}
		offer := market.Offer{ID: offerID, Title: product.Snapshot.Title, Description: product.Snapshot.PlainDescription(),
			Brand: product.Snapshot.Brand, Link: item.Link, ImageLink: imageURL, Currency: item.Currency,
			AmountMicros: amount, Availability: item.Availability, Condition: item.Condition, Color: item.Color, Size: item.Size}
		if len(choices) > 1 {
			offer.ItemGroupID, _ = plan.Target.OfferID(collection.StableID(product.Identity.ProductKey, "group"))
		}
		plan.Offers = append(plan.Offers, offer)
	}
	return s.Plans.Save(ctx, plan)
}

func googlePrice(value string) (int64, bool) {
	whole, cents, found := strings.Cut(value, ".")
	if whole == "" || len(whole) > 10 || strings.Trim(whole, "0123456789") != "" ||
		found && (cents == "" || len(cents) > 6 || strings.Trim(cents, "0123456789") != "") {
		return 0, false
	}
	amount, err := strconv.ParseInt(whole+cents+strings.Repeat("0", 6-len(cents)), 10, 64)
	return amount, err == nil && amount > 0
}

func (s *GooglePublication) load(ctx context.Context, sourceID, id string, write bool) (submission.GooglePlan, error) {
	scope, product, inventory, err := s.source(ctx, sourceID, write, write)
	if err != nil {
		return submission.GooglePlan{}, err
	}
	plan, err := s.Plans.Read(ctx, scope, id)
	if err != nil {
		return submission.GooglePlan{}, err
	}
	if plan.SourceID != sourceID || write && (plan.Version != product.Version ||
		plan.ProductHash != collection.Digest(product.Snapshot) || plan.InventoryHash != collection.Digest(inventory)) {
		return submission.GooglePlan{}, record.ErrConflict
	}
	return plan, nil
}

func (s *GooglePublication) Confirm(ctx context.Context, sourceID, id string) (GooglePublicationResult, error) {
	plan, err := s.load(ctx, sourceID, id, true)
	if err != nil {
		return GooglePublicationResult{}, err
	}
	current, err := s.Stores.PublicationBinding(ctx, plan.Owner.OrganizationID, plan.StoreID)
	if err != nil {
		return GooglePublicationResult{}, err
	}
	if current.Target != plan.Target || current.Generation != plan.Generation {
		return GooglePublicationResult{}, record.ErrConflict
	}
	result := GooglePublicationResult{ID: id, Items: make([]GoogleItemResult, 0, len(plan.Offers))}
	for _, offer := range plan.Offers {
		if err := ctx.Err(); err != nil {
			return result, err
		}
		item, err := s.send(ctx, plan, current, offer)
		if err != nil {
			return result, err
		}
		result.Items = append(result.Items, item)
	}
	return result, nil
}

func googleIntent(plan submission.GooglePlan, offer market.Offer) string {
	return collection.StableID(plan.Owner.OrganizationID, "google-product-insert", plan.ID, offer.ID)
}

func (s *GooglePublication) send(ctx context.Context, plan submission.GooglePlan, binding storeapp.GooglePublicationBinding, offer market.Offer) (GoogleItemResult, error) {
	scope := submission.ExecutionScope{OrganizationID: plan.Owner.OrganizationID}
	key := googleIntent(plan, offer)
	payload, err := json.Marshal([]any{plan.Version, plan.ProductHash, plan.InventoryHash, binding, offer})
	if err != nil {
		return GoogleItemResult{}, err
	}
	subject := collection.Digest([]string{plan.Target.ContentLanguage, plan.Target.FeedLabel, offer.ID})
	a, err := s.Kernel.PrepareGoogleExecution(ctx, submission.AcquireExecutionCommand{
		Scope: scope, IntentKey: key, Target: submission.ExecutionTarget{Platform: string(storecenter.PlatformGoogleMerchant),
			StoreID: plan.StoreID, SubjectID: subject}, Action: submission.GoogleInsertAction, Payload: payload, ClaimOwnerID: plan.Owner.ActorID,
		Lease: 30 * time.Second})
	if err != nil {
		return GoogleItemResult{}, err
	}
	if a.Preparation == nil {
		return googleResult(offer.ID, a.Attempt), nil
	}
	var permit *submission.SendPermit
	remote, sendErr := s.Stores.Publish(ctx, scope.OrganizationID, plan.StoreID, binding, offer, func(sendCtx context.Context) error {
		current, err := s.Authorization.Authorize(sendCtx, preparation.PermissionSubmit)
		if err != nil || current != plan.Owner {
			return preparation.ErrForbidden
		}
		permit, err = s.Kernel.BeginGoogleSend(sendCtx, a.Preparation)
		return err
	})
	finish, cancel := context.WithTimeout(context.WithoutCancel(ctx), 4*time.Second)
	defer cancel()
	if permit == nil {
		reason := submission.NoSendPreflightUnavailable
		if errors.Is(sendErr, preparation.ErrForbidden) {
			reason = submission.NoSendAccessRecheckFailed
		}
		stopped, err := s.Kernel.StopGoogleBeforeSend(finish, a.Preparation, reason)
		if err != nil {
			return GoogleItemResult{OfferID: offer.ID, Status: submission.ExecutionOutcomeUnknown}, nil
		}
		return googleResult(offer.ID, stopped), nil
	}
	claim := submission.ExecutionClaim{Scope: scope, AttemptID: permit.AttemptID, FenceEpoch: permit.FenceEpoch,
		OwnerID: permit.ClaimOwnerID, Token: permit.ClaimToken}
	if sendErr != nil && !errors.Is(sendErr, market.ErrOfferExists) && !errors.Is(sendErr, market.ErrRejected) &&
		!errors.Is(sendErr, market.ErrUnauthorized) && !errors.Is(sendErr, market.ErrSourceMismatch) {
		saved, e := s.Kernel.MarkUnknown(finish, claim, submission.UnknownResponseLost)
		if e != nil {
			return GoogleItemResult{OfferID: offer.ID, Status: submission.ExecutionOutcomeUnknown}, nil
		}
		return googleResult(offer.ID, saved), nil
	}
	evidence := submission.ExecutionEvidence{Kind: submission.EvidenceProviderResponse, Outcome: submission.ExecutionSucceeded,
		Reference: offer.ID, Fingerprint: collection.Digest(remote), Reason: remote.Product, ObservedAt: time.Now().UTC()}
	if sendErr != nil {
		evidence.Outcome, evidence.Fingerprint, evidence.Reason = submission.ExecutionFailedDefinitive, collection.Digest(sendErr.Error()), sendErr.Error()
	}
	saved, err := s.Kernel.Complete(finish, claim, evidence)
	if err != nil {
		_, _ = s.Kernel.MarkUnknown(finish, claim, submission.UnknownResponseLost)
		return GoogleItemResult{OfferID: offer.ID, Status: submission.ExecutionOutcomeUnknown}, nil
	}
	return googleResult(offer.ID, saved), nil
}

func googleResult(offer string, attempt submission.ExecutionAttempt) GoogleItemResult {
	result := GoogleItemResult{OfferID: offer, Status: attempt.Status}
	if attempt.Status == submission.ExecutionSucceeded && attempt.Evidence != nil {
		result.RemoteProduct = attempt.Evidence.Reason
	}
	return result
}

func (s *GooglePublication) Result(ctx context.Context, sourceID, id string) (GooglePublicationResult, error) {
	plan, err := s.load(ctx, sourceID, id, false)
	if err != nil {
		return GooglePublicationResult{}, err
	}
	result := GooglePublicationResult{ID: id, Items: make([]GoogleItemResult, 0, len(plan.Offers))}
	scope := submission.ExecutionScope{OrganizationID: plan.Owner.OrganizationID}
	for _, offer := range plan.Offers {
		attempt, err := s.Kernel.ReadIntent(ctx, scope, googleIntent(plan, offer))
		if errors.Is(err, submission.ErrExecutionNotFound) {
			result.Items = append(result.Items, GoogleItemResult{OfferID: offer.ID, Status: submission.ExecutionNotSent})
			continue
		}
		if err != nil {
			return result, err
		}
		item := googleResult(offer.ID, attempt)
		if item.Status == submission.ExecutionClaimed && !time.Now().Before(attempt.LeaseExpiresAt) {
			if expired, err := s.Kernel.Expire(ctx, scope, attempt.AttemptID); err == nil {
				item = googleResult(offer.ID, expired)
			}
		}
		if item.Status == submission.ExecutionSucceeded || item.Status == submission.ExecutionOutcomeUnknown {
			if observation, err := s.Stores.ReadProduct(ctx, plan.Owner.OrganizationID, plan.StoreID, offer.ID); err == nil {
				item.Observation = &observation
			}
		}
		result.Items = append(result.Items, item)
	}
	return result, nil
}
