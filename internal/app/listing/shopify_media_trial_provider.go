package listingapp

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"slices"

	"github.com/google/uuid"
	wire "github.com/henryxu1008/shuomi-ai-commerce/internal/integration/shopify"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/listing/preparation"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/listing/submission"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/product/asset"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/storecenter"
)

type shopifyMediaTrialWriter interface {
	Prepare(context.Context, [][32]byte) ([]*wire.StagedDraftMedia, error)
	Upload(context.Context, *wire.StagedDraftMedia, []byte) error
	Append(context.Context, []*wire.StagedDraftMedia) ([]string, error)
	Associate(context.Context, []wire.DraftMediaAssociation) error
}

type shopifyTrialDispatchEvidence struct {
	source shopifyMediaSourceEvidence
	remote wire.DraftMediaEvidence
}

// Each confirmation creates a new provider. Signed forms are held only in
// this invocation, and durable operation proofs never contain the forms.
type shopifyMediaTrialProvider struct {
	operation submission.ShopifyMediaOperation
	read      func(context.Context, submission.ShopifyMediaPlan, bool) (shopifyTrialDispatchEvidence, error)
	image     func(context.Context, shopifyMediaSourceEvidence, string) ([]byte, error)
	writer    func(context.Context) (shopifyMediaTrialWriter, error)

	verified   string
	readback   string
	last       shopifyTrialDispatchEvidence
	phase      submission.ShopifyMediaPhase
	stagedKeys []string
	staged     []*wire.StagedDraftMedia
}

func (a *Application) newShopifyMediaTrialProvider(operation submission.ShopifyMediaOperation, binding storecenter.ShopifyBinding, sourceID string) *shopifyMediaTrialProvider {
	service := a.ShopifyMedia
	subject := shopifyProductSubject(operation.Owner, storecenter.ProductPurposePublish)
	return &shopifyMediaTrialProvider{
		operation: operation,
		read: func(ctx context.Context, plan submission.ShopifyMediaPlan, gallery bool) (shopifyTrialDispatchEvidence, error) {
			if service == nil || service.Store == nil || service.Inventory == nil || service.ReadImage == nil {
				return shopifyTrialDispatchEvidence{}, preparation.ErrUnavailable
			}
			active, err := a.Authorization.Authorize(ctx, preparation.PermissionSubmit)
			if err != nil || active != operation.Owner {
				return shopifyTrialDispatchEvidence{}, preparation.ErrForbidden
			}
			source, err := a.Sources.Select(ctx, sourceID)
			if err != nil {
				return shopifyTrialDispatchEvidence{}, err
			}
			scope, item, original, err := source.Read(ctx)
			if err != nil || scope != operation.Owner || item.Source.Kind != "own" ||
				item.Source.ProductKey != plan.SourceKey || original.Version != plan.SourceVersion {
				return shopifyTrialDispatchEvidence{}, preparation.ErrConflict
			}
			selected, err := a.readShopifyMediaSource(ctx, scope, item, original)
			if err != nil {
				return shopifyTrialDispatchEvidence{}, err
			}
			evidence := shopifyTrialDispatchEvidence{source: selected}
			if gallery {
				evidence.remote, err = service.Store.ReadPairedShopifyTrialMedia(ctx, subject, binding, operation.ProductID)
			} else {
				err = service.Store.CheckPairedShopifyMediaBinding(ctx, subject, binding)
			}
			if err != nil {
				return shopifyTrialDispatchEvidence{}, shopifyMediaPreviewError(err)
			}
			if err := a.recheckShopifyMediaSource(ctx, selected); err != nil {
				return shopifyTrialDispatchEvidence{}, err
			}
			return evidence, nil
		},
		image: func(ctx context.Context, evidence shopifyMediaSourceEvidence, hash string) ([]byte, error) {
			for i, proof := range evidence.proofs {
				if hex.EncodeToString(proof.Hash[:]) != hash {
					continue
				}
				imageID := evidence.images[evidence.positions[i]].ID
				index := slices.IndexFunc(evidence.inventory.Assets, func(approved asset.ApprovedAsset) bool { return approved.ID == imageID })
				if index >= 0 {
					return service.ReadImage(ctx, evidence.source, evidence.inventory.Assets[index], wire.MaxStagedDraftImageBytes)
				}
			}
			return nil, submission.ErrExecutionEvidenceRequired
		},
		writer: func(ctx context.Context) (shopifyMediaTrialWriter, error) {
			active, err := a.Authorization.Authorize(ctx, preparation.PermissionSubmit)
			if err != nil || active != operation.Owner {
				return nil, preparation.ErrForbidden
			}
			return service.Store.VerifyPairedShopifyMediaWrite(ctx, subject, binding, operation.ProductID)
		},
	}
}

func (p *shopifyMediaTrialProvider) Verify(ctx context.Context, plan submission.ShopifyMediaPlan, phase submission.ShopifyMediaPhase) error {
	p.verified = ""
	gallery := phase == submission.ShopifyMediaPrepare || phase == submission.ShopifyMediaAppend || phase == submission.ShopifyMediaAssociate
	before := phase == submission.ShopifyMediaPrepare || phase == submission.ShopifyMediaAppend
	evidence, err := p.current(ctx, plan, gallery, before)
	if err != nil {
		return err
	}
	if phase == submission.ShopifyMediaAssociate && p.readback != "" && p.readback != shopifyTrialBeforeHash(evidence.remote) {
		return submission.ErrExecutionEvidenceRequired
	}
	p.last, p.phase, p.verified = evidence, phase, plan.Fingerprint()
	return nil
}

func (p *shopifyMediaTrialProvider) Read(ctx context.Context, plan submission.ShopifyMediaPlan) (submission.ShopifyMediaObservation, error) {
	evidence, err := p.current(ctx, plan, true, false)
	if err != nil {
		return submission.ShopifyMediaObservation{}, err
	}
	p.readback = shopifyTrialBeforeHash(evidence.remote)
	return evidence.observation(), nil
}

func (p *shopifyMediaTrialProvider) current(ctx context.Context, plan submission.ShopifyMediaPlan, gallery, beforeAppend bool) (shopifyTrialDispatchEvidence, error) {
	if p.operation != plan.Operation() || !plan.Valid() {
		return shopifyTrialDispatchEvidence{}, submission.ErrExecutionEvidenceRequired
	}
	evidence, err := p.read(ctx, plan, gallery)
	if err != nil {
		return shopifyTrialDispatchEvidence{}, err
	}
	if !evidence.matches(plan, gallery, beforeAppend) {
		return shopifyTrialDispatchEvidence{}, submission.ErrExecutionEvidenceRequired
	}
	return evidence, nil
}

func (e shopifyTrialDispatchEvidence) matches(plan submission.ShopifyMediaPlan, gallery, beforeAppend bool) bool {
	if e.source.selection.OriginalSnapshotVersion != plan.SourceVersion || len(plan.Covers) != 6 ||
		len(e.source.images) != len(e.source.proofs) ||
		len(e.source.positions) != len(e.source.proofs) {
		return false
	}
	approved := make(map[string]string, len(e.source.proofs))
	for i, proof := range e.source.proofs {
		if proof.Size < 1 || proof.Hash == ([32]byte{}) {
			return false
		}
		approved[e.source.images[e.source.positions[i]].ID] = hex.EncodeToString(proof.Hash[:])
	}
	if len(approved) != len(e.source.images) || shopifyTrialApprovalHash(e.source, approved) != plan.ApprovalHash {
		return false
	}
	if !gallery {
		return true
	}
	if e.remote.ID != plan.ProductID || len(e.remote.Variants) != len(plan.Covers) {
		return false
	}
	if beforeAppend {
		return plan.BeforeHash == shopifyTrialBeforeHash(e.remote)
	}
	original := e.remote
	original.Gallery = nil
	for _, image := range e.remote.Gallery {
		if slices.Contains(plan.GalleryIDs, image.ID) {
			original.Gallery = append(original.Gallery, image)
		}
	}
	original.Variants = slices.Clone(e.remote.Variants)
	for i, variant := range original.Variants {
		index := slices.IndexFunc(plan.Covers, func(cover submission.ShopifyMediaCover) bool { return cover.VariantID == variant.ID })
		if index < 0 || len(variant.MediaIDs) > 1 {
			return false
		}
		original.Variants[i].MediaIDs = nil
		if current := plan.Covers[index].CurrentMediaID; current != "" {
			original.Variants[i].MediaIDs = []string{current}
		}
	}
	return plan.BeforeHash == shopifyTrialBeforeHash(original)
}

func (e shopifyTrialDispatchEvidence) observation() submission.ShopifyMediaObservation {
	result := submission.ShopifyMediaObservation{ProductID: e.remote.ID, Draft: true, Complete: true,
		Gallery: make([]submission.ShopifyMediaOriginal, 0, len(e.remote.Gallery)), Associations: make(map[string]string, len(e.remote.Variants))}
	for _, image := range e.remote.Gallery {
		ready := image.Type == "IMAGE" && image.Status == "READY" && image.Hash != ([32]byte{})
		result.Gallery = append(result.Gallery, submission.ShopifyMediaOriginal{ID: image.ID, Hash: hex.EncodeToString(image.Hash[:]), Ready: ready})
	}
	for _, variant := range e.remote.Variants {
		result.Associations[variant.ID] = ""
		if len(variant.MediaIDs) == 1 {
			result.Associations[variant.ID] = variant.MediaIDs[0]
		}
	}
	return result
}

func (p *shopifyMediaTrialProvider) Send(ctx context.Context, request submission.ShopifyMediaRequest) (submission.ShopifyMediaResponse, error) {
	if p.verified == "" || p.verified != request.Plan.Fingerprint() || p.phase != request.Phase {
		return submission.ShopifyMediaResponse{}, submission.ErrExecutionEvidenceRequired
	}
	p.verified = ""
	var writer shopifyMediaTrialWriter
	var err error
	if request.Phase != submission.ShopifyMediaUpload {
		writer, err = p.writer(ctx)
		if err != nil {
			return submission.ShopifyMediaResponse{}, err
		}
	}
	reply := submission.ShopifyMediaResponse{OriginalResponse: true}
	switch request.Phase {
	case submission.ShopifyMediaPrepare:
		reply.Staged, err = p.prepare(ctx, writer, request.Plan.Images)
	case submission.ShopifyMediaUpload:
		err = p.upload(ctx, request)
	case submission.ShopifyMediaAppend:
		if !slices.Equal(p.stagedKeys, request.Staged) || len(p.staged) == 0 {
			return submission.ShopifyMediaResponse{}, submission.ErrExecutionEvidenceRequired
		}
		reply.Candidates, err = writer.Append(ctx, p.staged)
		p.staged, p.stagedKeys = nil, nil
	case submission.ShopifyMediaAssociate:
		if p.readback == "" {
			return submission.ShopifyMediaResponse{}, submission.ErrExecutionEvidenceRequired
		}
		err = p.associate(ctx, writer, request)
	default:
		return submission.ShopifyMediaResponse{}, submission.ErrExecutionInvalid
	}
	if errors.Is(err, wire.ErrDraftMediaRejected) {
		return submission.ShopifyMediaResponse{OriginalResponse: true, Rejected: true}, nil
	}
	if err != nil {
		return submission.ShopifyMediaResponse{}, err
	}
	return reply, nil
}

func (p *shopifyMediaTrialProvider) prepare(ctx context.Context, writer shopifyMediaTrialWriter, images []string) ([]string, error) {
	if len(p.staged) != 0 {
		return nil, submission.ErrExecutionEvidenceRequired
	}
	hashes := make([][32]byte, len(images))
	for i, hash := range images {
		raw, err := hex.DecodeString(hash)
		if err != nil || len(raw) != sha256.Size {
			return nil, submission.ErrExecutionEvidenceRequired
		}
		copy(hashes[i][:], raw)
	}
	staged, err := writer.Prepare(ctx, hashes)
	if err != nil {
		return nil, err
	}
	if len(staged) != len(hashes) {
		return nil, submission.ErrExecutionEvidenceRequired
	}
	keys := make([]string, len(staged))
	for i, image := range staged {
		if image == nil {
			return nil, submission.ErrExecutionEvidenceRequired
		}
		keys[i] = uuid.NewString()
	}
	p.staged, p.stagedKeys = staged, keys
	return slices.Clone(keys), nil
}

func (p *shopifyMediaTrialProvider) upload(ctx context.Context, request submission.ShopifyMediaRequest) error {
	i := request.Index
	if i < 0 || i >= len(p.staged) || i >= len(request.Plan.Images) || len(request.Staged) != 1 || request.Staged[0] != p.stagedKeys[i] {
		return submission.ErrExecutionEvidenceRequired
	}
	bytes, err := p.image(ctx, p.last.source, request.Plan.Images[i])
	if err != nil {
		return err
	}
	hash := sha256.Sum256(bytes)
	if hex.EncodeToString(hash[:]) != request.Plan.Images[i] {
		return submission.ErrExecutionEvidenceRequired
	}
	writer, err := p.writer(ctx)
	if err != nil {
		return err
	}
	return writer.Upload(ctx, p.staged[i], bytes)
}

func (p *shopifyMediaTrialProvider) associate(ctx context.Context, writer shopifyMediaTrialWriter, request submission.ShopifyMediaRequest) error {
	current := p.last.observation()
	changes := make([]wire.DraftMediaAssociation, 0, len(request.Associations))
	for _, change := range request.Associations {
		i := slices.IndexFunc(request.Plan.Covers, func(cover submission.ShopifyMediaCover) bool { return cover.VariantID == change.VariantID })
		j := slices.IndexFunc(current.Gallery, func(image submission.ShopifyMediaOriginal) bool { return image.ID == change.MediaID })
		if i < 0 || j < 0 || current.Associations[change.VariantID] != request.Plan.Covers[i].CurrentMediaID ||
			!current.Gallery[j].Ready || current.Gallery[j].Hash != request.Plan.Covers[i].ImageHash {
			return submission.ErrExecutionEvidenceRequired
		}
		changes = append(changes, wire.DraftMediaAssociation{VariantID: change.VariantID, MediaID: change.MediaID})
	}
	return writer.Associate(ctx, changes)
}
