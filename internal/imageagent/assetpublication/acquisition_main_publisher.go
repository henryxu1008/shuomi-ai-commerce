package assetpublication

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"task-processor/internal/imageagent"
	productasset "task-processor/internal/product/asset"
)

type AcquisitionMainPublisher struct {
	projections ProjectionSource
	assets      productasset.Repository
	publicURLs  imageagent.DurableAssetPublicURLResolver
	trialURLs   *imageagent.IsolatedTrialGeneratedURLPolicy
}

func NewAcquisitionMainPublisher(projections ProjectionSource, assets productasset.Repository, publicURLs imageagent.DurableAssetPublicURLResolver, trial ...*imageagent.IsolatedTrialGeneratedURLPolicy) (*AcquisitionMainPublisher, error) {
	if nilValue(projections) || nilValue(assets) || nilValue(publicURLs) || len(trial) > 1 {
		return nil, fmt.Errorf("image agent projection source, product asset repository, and public URL resolver are required")
	}
	publisher := &AcquisitionMainPublisher{projections: projections, assets: assets, publicURLs: publicURLs}
	if len(trial) == 1 {
		publisher.trialURLs = trial[0]
	}
	return publisher, nil
}

// PublishAcquisitionMain commits the approved capture main through the current
// Product Asset owner and its immutable approval receipt.
func (p *AcquisitionMainPublisher) PublishAcquisitionMain(ctx context.Context, input imageagent.PublishAcquisitionMainInput) (imageagent.PublicationAcknowledgement, error) {
	if p == nil || p.projections == nil || p.assets == nil || p.publicURLs == nil {
		return imageagent.PublicationAcknowledgement{}, imageagent.ErrValidation
	}
	if ctx == nil {
		return imageagent.PublicationAcknowledgement{}, imageagent.ErrValidation
	}
	if err := ctx.Err(); err != nil {
		return imageagent.PublicationAcknowledgement{}, err
	}
	if !canonical(input.RunID) || !canonical(input.TenantID) || !canonical(input.UserID) ||
		!canonical(input.IdempotencyKey) || input.PlanRevision <= 0 || len(input.CandidateAssetIDs) == 0 {
		return imageagent.PublicationAcknowledgement{}, imageagent.ErrValidation
	}
	scope := imageagent.RunScope{TenantID: input.TenantID, OwnerUserID: input.UserID, RunID: input.RunID}
	projection, err := p.projections.GetProjection(ctx, scope)
	if err != nil {
		return imageagent.PublicationAcknowledgement{}, fmt.Errorf("load approved image agent projection: %w", err)
	}
	commit, digest, err := p.approvalCommit(projection, input)
	if err != nil {
		return imageagent.PublicationAcknowledgement{}, err
	}
	receipt, err := p.assets.CommitApproval(ctx, commit)
	if err != nil {
		return imageagent.PublicationAcknowledgement{}, fmt.Errorf("commit approved product assets: %w", err)
	}
	return publicationAcknowledgement(commit, input.RunID, input.PlanRevision, digest, receipt), nil
}
func (p *AcquisitionMainPublisher) approvalCommit(projection imageagent.RunProjection, input imageagent.PublishAcquisitionMainInput) (productasset.ApprovalCommit, string, error) {
	if projection.Run.TenantID != input.TenantID || projection.Run.UserID != input.UserID || projection.Run.ID != input.RunID ||
		projection.Run.Status != imageagent.RunStatusAwaitingFinalApproval || projection.Run.ActivePlanRevision != input.PlanRevision ||
		projection.Plan.Revision != input.PlanRevision || len(projection.Plan.Slots) != len(projection.Slots) {
		return productasset.ApprovalCommit{}, "", imageagent.ErrRevisionConflict
	}
	if err := imageagent.ValidateAcquisitionMainPlan(projection.Run.TargetPlatform, projection.Run.ImagePolicyContext, projection.Plan); err != nil {
		return productasset.ApprovalCommit{}, "", err
	}
	product, err := imageagent.NormalizeProductContextRef(projection.AssetCatalog.ProductContext)
	if err != nil || product.ProductID == "" {
		return productasset.ApprovalCommit{}, "", imageagent.ErrRevisionConflict
	}
	digest, err := imageagent.ResultDigestV3(projection.Plan, projection.Slots)
	if err != nil || projection.ResultDigest == "" || projection.ResultDigest != digest {
		return productasset.ApprovalCommit{}, "", imageagent.ErrRevisionConflict
	}
	requested := make([]string, len(input.CandidateAssetIDs))
	for index, candidateID := range input.CandidateAssetIDs {
		if !canonical(candidateID) {
			return productasset.ApprovalCommit{}, "", imageagent.ErrValidation
		}
		requested[index] = candidateID
	}
	approved := make([]productasset.ApprovedAsset, 0, len(requested))
	observed := make([]string, 0, len(requested))
	mainAssets := 0
	for slotIndex, declared := range projection.Plan.Slots {
		slot := projection.Slots[slotIndex]
		role := productasset.RoleMain
		if declared.Role != imageagent.SlotRoleMain || len(projection.Plan.Slots) != 1 || len(slot.Candidates) != 1 {
			return productasset.ApprovalCommit{}, "", imageagent.ErrRevisionConflict
		}
		if slot.Slot.ID != declared.ID || slot.Slot.Role != declared.Role ||
			slot.Slot.Status != imageagent.SlotStatusAccepted || slot.Attempt <= 0 || len(slot.Candidates) == 0 {
			return productasset.ApprovalCommit{}, "", imageagent.ErrRevisionConflict
		}
		for candidateIndex, candidate := range slot.Candidates {
			if !canonical(candidate.AssetID) || !canonical(candidate.SourceAssetID) {
				return productasset.ApprovalCommit{}, "", imageagent.ErrRevisionConflict
			}
			execution := imageagent.SlotExecutionInput{
				RunID: input.RunID, TenantID: input.TenantID, UserID: input.UserID,
				PlanRevision: input.PlanRevision, Slot: declared, Attempt: slot.Attempt,
			}
			if err := imageagent.ValidatePublishedAssetIdentityForSlot(execution, candidate.DurableAsset, candidateIndex); err != nil {
				return productasset.ApprovalCommit{}, "", err
			}
			identity, err := imageagent.NormalizeDurableAssetIdentity(candidate.DurableAsset)
			if err != nil {
				return productasset.ApprovalCommit{}, "", err
			}
			url, err := imageagent.ResolvePublishedAssetURL(execution, identity, candidateIndex, p.publicURLs, p.trialURLs)
			if err != nil {
				return productasset.ApprovalCommit{}, "", imageagent.ErrValidation
			}
			if role == productasset.RoleMain {
				mainAssets++
			}
			observed = append(observed, candidate.AssetID)
			approved = append(approved, productasset.ApprovedAsset{
				ID: candidate.AssetID, RunID: input.RunID, PlanRevision: input.PlanRevision,
				SlotID: declared.ID, Attempt: slot.Attempt, Role: role, URL: url,
				SourceAssetID: candidate.SourceAssetID, Width: candidate.Width, Height: candidate.Height,
				Operations: append([]string(nil), candidate.Operations...),
			})
		}
	}
	if mainAssets != 1 || !slices.Equal(observed, requested) {
		return productasset.ApprovalCommit{}, "", imageagent.ErrRevisionConflict
	}
	commit := productasset.ApprovalCommit{
		TenantID: input.TenantID, ProductKey: product.ProductID, TargetPlatform: strings.TrimSpace(projection.Run.TargetPlatform), ActionID: input.IdempotencyKey,
		SourceSnapshotVersion: sourceSnapshotVersion(projection.AssetCatalog), Assets: approved,
	}
	if err := productasset.ValidateApprovalCommit(commit); err != nil {
		return productasset.ApprovalCommit{}, "", fmt.Errorf("validate approved product assets: %w", err)
	}
	return commit, digest, nil
}
