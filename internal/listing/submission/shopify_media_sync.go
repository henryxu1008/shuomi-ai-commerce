package submission

import (
	"context"
	"encoding/hex"
	"errors"
	"regexp"
	"slices"
	"time"

	"github.com/google/uuid"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/authidentity"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/product/collection"
)

type ShopifyMediaPhase string

const (
	ShopifyMediaPrepare   ShopifyMediaPhase = "prepare_shopify_media"
	ShopifyMediaUpload    ShopifyMediaPhase = "upload_shopify_media"
	ShopifyMediaAppend    ShopifyMediaPhase = "append_shopify_media"
	ShopifyMediaAssociate ShopifyMediaPhase = "associate_shopify_media"
)

type ShopifyMediaStatus string

const (
	ShopifyMediaPending   ShopifyMediaStatus = "pending"
	ShopifyMediaComplete  ShopifyMediaStatus = "complete"
	ShopifyMediaUncertain ShopifyMediaStatus = "uncertain"
	ShopifyMediaExpired   ShopifyMediaStatus = "expired"
)

type ShopifyMediaPlan struct {
	ID            string
	Owner         collection.Scope
	ShopKey       string
	Generation    int64
	ProductID     string
	SourceKey     string
	SourceVersion uint64
	ApprovalHash  string
	BeforeHash    string
	ExpiresAt     time.Time
	GalleryIDs    []string
	Images        []string
	Covers        []ShopifyMediaCover
}

type ShopifyMediaCover struct{ VariantID, ImageHash, CurrentMediaID string }
type ShopifyMediaAssociation struct{ VariantID, MediaID string }

type ShopifyMediaOriginal struct {
	ID, Hash string
	Ready    bool
}

type ShopifyMediaRun struct {
	Plan       ShopifyMediaPlan
	Token      string
	Cycle      int
	Candidates []string
	Associated bool
	Status     ShopifyMediaStatus
}

type ShopifyMediaDispatch struct {
	AttemptID, Token string
	Phase            ShopifyMediaPhase
	Index            int
}

type ShopifyMediaReceipt struct {
	dispatch   ShopifyMediaDispatch
	digest     string
	candidates []string
	rejected   bool
}

func (r ShopifyMediaReceipt) Evidence(dispatch ShopifyMediaDispatch) (string, []string, bool, bool) {
	return r.digest, slices.Clone(r.candidates), r.rejected, r.dispatch == dispatch && r.digest != ""
}

type ShopifyMediaJournal interface {
	Begin(context.Context, ShopifyMediaPlan) (ShopifyMediaRun, error)
	Admit(context.Context, ShopifyMediaRun, ShopifyMediaPhase, int, string) (ShopifyMediaDispatch, error)
	Observe(context.Context, ShopifyMediaRun, ShopifyMediaDispatch, ShopifyMediaReceipt) error
	End(context.Context, ShopifyMediaRun, ShopifyMediaStatus) error
}

type ShopifyMediaRequest struct {
	Plan         ShopifyMediaPlan
	Phase        ShopifyMediaPhase
	Index        int
	Staged       []string
	Associations []ShopifyMediaAssociation
}

type ShopifyMediaResponse struct {
	OriginalResponse bool
	Rejected         bool
	Staged           []string
	Candidates       []string
}

type ShopifyMediaObservation struct {
	ProductID    string
	Draft        bool
	Complete     bool
	Gallery      []ShopifyMediaOriginal
	Associations map[string]string
}

// The provider port must resolve the current Store, original member and source
// inventory independently of the retained plan. BeforeAppend also compares
// the frozen pre-append remote snapshot.
type ShopifyMediaProvider interface {
	Verify(ctx context.Context, plan ShopifyMediaPlan, beforeAppend bool) error
	Read(context.Context, ShopifyMediaPlan) (ShopifyMediaObservation, error)
	Send(context.Context, ShopifyMediaRequest) (ShopifyMediaResponse, error)
}

type ShopifyMediaSync struct {
	Journal  ShopifyMediaJournal
	Provider ShopifyMediaProvider
}

func (s ShopifyMediaSync) Confirm(ctx context.Context, plan ShopifyMediaPlan) (status ShopifyMediaStatus, err error) {
	if ctx == nil || s.Provider == nil || s.Journal == nil || !plan.Valid() {
		return "", ErrExecutionInvalid
	}
	ctx, cancel := context.WithTimeout(ctx, 45*time.Second)
	defer cancel()
	if err = s.Provider.Verify(ctx, plan, false); err != nil {
		return "", err
	}
	run, err := s.Journal.Begin(ctx, plan)
	if err != nil || run.Token == "" {
		return run.Status, err
	}
	status = ShopifyMediaPending
	defer func() {
		closeCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), ExecutionTimeout)
		defer cancel()
		endErr := s.Journal.End(closeCtx, run, status)
		if endErr != nil {
			status, err = ShopifyMediaUncertain, errors.Join(err, endErr)
		}
	}()
	if len(plan.Images) > 0 && len(run.Candidates) == 0 {
		candidates, dispatchErr := s.appendImages(ctx, run)
		if dispatchErr != nil {
			return dispatchStatus(dispatchErr)
		}
		run.Candidates = candidates
	}
	observation, err := s.Provider.Read(ctx, plan)
	if err != nil {
		return status, err
	}
	desired, ready := plan.correlate(observation, run.Candidates)
	if !ready {
		return status, nil
	}
	if coversMatch(observation, desired) {
		return ShopifyMediaComplete, nil
	}
	if run.Associated || !plan.coversAtConfirmation(observation, desired) {
		return status, nil
	}
	if !time.Now().Before(plan.ExpiresAt) {
		return ShopifyMediaExpired, nil
	}
	changes := slices.DeleteFunc(slices.Clone(desired), func(item ShopifyMediaAssociation) bool {
		return observation.Associations[item.VariantID] == item.MediaID
	})
	_, err = s.send(ctx, run, ShopifyMediaRequest{Plan: plan, Phase: ShopifyMediaAssociate, Associations: changes}, false)
	if err != nil {
		return dispatchStatus(err)
	}
	observation, err = s.Provider.Read(ctx, plan)
	if err != nil {
		return status, err
	}
	if after, verified := plan.correlate(observation, run.Candidates); verified && slices.Equal(after, desired) && coversMatch(observation, desired) {
		return ShopifyMediaComplete, nil
	}
	return status, nil
}

func coversMatch(observation ShopifyMediaObservation, desired []ShopifyMediaAssociation) bool {
	for _, association := range desired {
		if observation.Associations[association.VariantID] != association.MediaID {
			return false
		}
	}
	return true
}

func dispatchStatus(err error) (ShopifyMediaStatus, error) {
	if errors.Is(err, ErrExecutionOutcomeUnknown) {
		return ShopifyMediaUncertain, nil
	}
	return ShopifyMediaPending, err
}

func (s ShopifyMediaSync) appendImages(ctx context.Context, run ShopifyMediaRun) ([]string, error) {
	plan := run.Plan
	staging, err := s.send(ctx, run, ShopifyMediaRequest{Plan: plan, Phase: ShopifyMediaPrepare}, true)
	if err != nil {
		return nil, err
	}
	for i, resource := range staging.Staged {
		if _, err := s.send(ctx, run, ShopifyMediaRequest{Plan: plan, Phase: ShopifyMediaUpload, Index: i, Staged: []string{resource}}, true); err != nil {
			return nil, err
		}
	}
	appended, err := s.send(ctx, run, ShopifyMediaRequest{Plan: plan, Phase: ShopifyMediaAppend, Staged: staging.Staged}, true)
	return appended.Candidates, err
}

func (s ShopifyMediaSync) send(ctx context.Context, run ShopifyMediaRun, request ShopifyMediaRequest, beforeAppend bool) (ShopifyMediaResponse, error) {
	if err := s.Provider.Verify(ctx, run.Plan, beforeAppend); err != nil {
		return ShopifyMediaResponse{}, err
	}
	// Signed staging resources are sent directly from memory. The durable
	// fingerprint binds the approved plan and semantic payload only.
	fingerprint := collection.Digest(struct {
		Plan         string
		Associations []ShopifyMediaAssociation
	}{collection.Digest(run.Plan), request.Associations})
	permit, err := s.Journal.Admit(ctx, run, request.Phase, request.Index, fingerprint)
	if err != nil {
		return ShopifyMediaResponse{}, err
	}
	if ctx.Err() != nil {
		return ShopifyMediaResponse{}, ErrExecutionOutcomeUnknown
	}
	response, err := s.Provider.Send(ctx, request)
	if err != nil || !response.OriginalResponse || !validMediaReply(request, response) {
		return ShopifyMediaResponse{}, ErrExecutionOutcomeUnknown
	}
	returnCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), ExecutionTimeout)
	defer cancel()
	receipt := ShopifyMediaReceipt{dispatch: permit, digest: collection.Digest(response), candidates: slices.Clone(response.Candidates), rejected: response.Rejected}
	if s.Journal.Observe(returnCtx, run, permit, receipt) != nil {
		return ShopifyMediaResponse{}, ErrExecutionOutcomeUnknown
	}
	if response.Rejected {
		return ShopifyMediaResponse{}, ErrExecutionOutcomeUnknown
	}
	return response, nil
}

func validMediaReply(request ShopifyMediaRequest, response ShopifyMediaResponse) bool {
	if response.Rejected {
		return len(response.Staged) == 0 && len(response.Candidates) == 0
	}
	switch request.Phase {
	case ShopifyMediaPrepare:
		if len(response.Staged) != len(request.Plan.Images) || len(response.Candidates) != 0 {
			return false
		}
		seen := make(map[string]bool, len(response.Staged))
		for _, resource := range response.Staged {
			if resource == "" || len(resource) > 2048 || seen[resource] {
				return false
			}
			seen[resource] = true
		}
		return true
	case ShopifyMediaAppend:
		return len(response.Staged) == 0 && len(response.Candidates) >= len(request.Plan.Images) &&
			validShopifyMediaIDs(response.Candidates, shopifyMediaImageID, 60)
	case ShopifyMediaUpload, ShopifyMediaAssociate:
		return len(response.Staged) == 0 && len(response.Candidates) == 0
	default:
		return false
	}
}

var (
	shopifyMediaProductID = regexp.MustCompile("^gid://shopify/Product/[1-9][0-9]{0,19}$")
	shopifyMediaImageID   = regexp.MustCompile("^gid://shopify/MediaImage/[1-9][0-9]{0,19}$")
	shopifyMediaVariantID = regexp.MustCompile("^gid://shopify/ProductVariant/[1-9][0-9]{0,19}$")
)

func (p ShopifyMediaPlan) Valid() bool {
	id, err := uuid.Parse(p.ID)
	shop, shopErr := uuid.Parse(p.ShopKey)
	if err != nil || shopErr != nil || id.String() != p.ID || shop.String() != p.ShopKey || p.Owner.Validate() != nil ||
		p.Generation < 1 || !shopifyMediaProductID.MatchString(p.ProductID) || !authidentity.IsBoundedIdentifier(p.SourceKey) ||
		p.SourceVersion < 1 || !validMediaHash(p.ApprovalHash) || !validMediaHash(p.BeforeHash) || p.ExpiresAt.IsZero() ||
		!validShopifyMediaIDs(p.GalleryIDs, shopifyMediaImageID, 60) || len(p.Images) > 10 || len(p.Covers) > 30 ||
		len(p.Images)+len(p.Covers) == 0 {
		return false
	}
	hashes := make(map[string]bool, len(p.Images))
	for _, hash := range p.Images {
		if !validMediaHash(hash) || hashes[hash] {
			return false
		}
		hashes[hash] = true
	}
	variants := make(map[string]bool, len(p.Covers))
	for _, cover := range p.Covers {
		if !shopifyMediaVariantID.MatchString(cover.VariantID) || variants[cover.VariantID] || !validMediaHash(cover.ImageHash) ||
			cover.CurrentMediaID != "" && !slices.Contains(p.GalleryIDs, cover.CurrentMediaID) {
			return false
		}
		variants[cover.VariantID] = true
	}
	return true
}

func validMediaHash(s string) bool {
	if len(s) != 64 {
		return false
	}
	b, err := hex.DecodeString(s)
	return err == nil && hex.EncodeToString(b) == s
}

func validShopifyMediaIDs(ids []string, pattern *regexp.Regexp, limit int) bool {
	if len(ids) > limit {
		return false
	}
	seen := make(map[string]bool, len(ids))
	for _, id := range ids {
		if !pattern.MatchString(id) || seen[id] {
			return false
		}
		seen[id] = true
	}
	return true
}

func (p ShopifyMediaPlan) correlate(obs ShopifyMediaObservation, candidates []string) ([]ShopifyMediaAssociation, bool) {
	if !obs.Draft || !obs.Complete || obs.ProductID != p.ProductID || len(obs.Gallery) > 60 {
		return nil, false
	}
	added := make(map[string]bool, len(candidates))
	for _, id := range candidates {
		if !slices.Contains(p.GalleryIDs, id) {
			added[id] = true
		}
	}
	if len(added) != len(p.Images) || len(obs.Gallery) != len(p.GalleryIDs)+len(added) {
		return nil, false
	}
	originals := make(map[string]string, len(obs.Gallery))
	found := make(map[string]bool, len(obs.Gallery))
	for _, image := range obs.Gallery {
		if !shopifyMediaImageID.MatchString(image.ID) || found[image.ID] || !image.Ready || !validMediaHash(image.Hash) {
			return nil, false
		}
		found[image.ID] = true
		if _, duplicate := originals[image.Hash]; duplicate {
			originals[image.Hash] = ""
		} else {
			originals[image.Hash] = image.ID
		}
	}
	for _, id := range p.GalleryIDs {
		if !found[id] {
			return nil, false
		}
	}
	for _, hash := range p.Images {
		if id := originals[hash]; id == "" || !added[id] {
			return nil, false
		}
	}
	covers := make([]ShopifyMediaAssociation, 0, len(p.Covers))
	for _, cover := range p.Covers {
		id := originals[cover.ImageHash]
		if _, exists := obs.Associations[cover.VariantID]; !exists || id == "" ||
			!slices.Contains(p.GalleryIDs, id) && !added[id] {
			return nil, false
		}
		covers = append(covers, ShopifyMediaAssociation{VariantID: cover.VariantID, MediaID: id})
	}
	return covers, true
}

func (p ShopifyMediaPlan) coversAtConfirmation(observation ShopifyMediaObservation, desired []ShopifyMediaAssociation) bool {
	for i, cover := range p.Covers {
		if current := observation.Associations[cover.VariantID]; current != cover.CurrentMediaID && current != desired[i].MediaID {
			return false
		}
	}
	return true
}
