package productsourcing

import (
	"context"
	"encoding/json"
	"reflect"
	"task-processor/internal/agent"
	"task-processor/internal/agentconfig"
	"task-processor/internal/imageagent"
	"task-processor/internal/product/asset"
	"task-processor/internal/product/catalog"
	"task-processor/internal/product/collection"
	"task-processor/internal/product/pod"
)

type ArtworkOriginGuard struct {
	URLs               imageagent.DurableAssetPublicURLResolver
	ReadGeneratedBytes func(context.Context, asset.SourceSelection, asset.ApprovedAsset, int64) ([]byte, error)
	Snapshots          catalog.VersionedSnapshotReader
	Images             imageagent.Repository
	Facts              imageagent.GenerationFactRepository
	Configuration      agentconfig.ArtworkConfigurationRepository
}

// Reuse the immutable artifact reader after the original Source/run/paid-mode
// proof. A generated source is never fetched through its browser URL.
func (g ArtworkOriginGuard) ReadArtworkSourceBytes(ctx context.Context, scope collection.Scope, ref pod.ArtworkReference, approved asset.ApprovedAsset, maximum int64) ([]byte, bool, error) {
	source := ref.Input.Source
	if err := g.AuthorizeArtworkSource(ctx, scope, source.ProductKey, source.PublicationID, source.Version); err != nil {
		return nil, false, err
	}
	snapshot, err := g.Snapshots.GetSnapshot(ctx, catalog.SnapshotIdentity{TenantID: scope.OrganizationID, ProductKey: source.ProductKey}, source.Version)
	if err != nil {
		return nil, false, err
	}
	for _, entry := range snapshot.Snapshot.Sources {
		if entry.Type != "agent_generated" && entry.Platform != "pod_artwork" && entry.ReferenceType != imageagent.ArtworkSchema {
			continue
		}
		var origin ArtworkOrigin
		if g.URLs == nil || g.ReadGeneratedBytes == nil || json.Unmarshal([]byte(entry.Metadata[artworkOriginKey]), &origin) != nil {
			return nil, true, asset.ErrSourceApprovalForbidden
		}
		p, err := g.Images.GetProjection(ctx, imageagent.RunScope{TenantID: scope.OrganizationID, OwnerUserID: scope.ActorID, RunID: origin.Identity.RunID})
		if err != nil {
			return nil, true, err
		}
		if len(p.Slots) != 1 || len(p.Slots[0].Candidates) != 1 {
			return nil, true, asset.ErrSourceApprovalForbidden
		}
		slot := p.Slots[0]
		candidate := slot.Candidates[0]
		if candidate.GenerationProof == nil || candidate.DurableAsset.SHA256 != origin.ArtifactHash || g.URLs.PublicURL(candidate.DurableAsset.ObjectKey) != approved.URL {
			return nil, true, asset.ErrSourceApprovalForbidden
		}
		generated := asset.ApprovedAsset{ID: candidate.AssetID, RunID: p.Run.ID, PlanRevision: p.Plan.Revision, SlotID: slot.Slot.ID, Attempt: slot.Attempt, URL: approved.URL, Width: candidate.Width, Height: candidate.Height, GenerationEvidence: &asset.GenerationEvidence{ArtifactHash: origin.ArtifactHash, IntentID: candidate.GenerationProof.IntentID, Fingerprint: candidate.GenerationProof.Fingerprint, SettlementProofDigest: candidate.GenerationProof.SettlementProofDigest}}
		raw, err := g.ReadGeneratedBytes(ctx, asset.SourceSelection{TenantID: scope.OrganizationID, ActorID: scope.ActorID, MemberID: scope.MemberID, ProductKey: source.ProductKey, OriginalPublicationID: source.PublicationID, OriginalSnapshotVersion: source.Version}, generated, maximum)
		return raw, true, err
	}
	return nil, false, nil
}

func (g ArtworkOriginGuard) AuthorizeSourceApproval(ctx context.Context, selection asset.SourceSelection, commit asset.ApprovalCommit) error {
	if commit.TenantID != selection.TenantID || commit.ProductKey != selection.ProductKey || commit.TargetPlatform != selection.TargetPlatform || commit.SourceSnapshotVersion != selection.EffectiveCatalogVersion {
		return asset.ErrSourceApprovalForbidden
	}
	return g.AuthorizeArtworkSource(ctx, collection.Scope{OrganizationID: selection.TenantID, ActorID: selection.ActorID, MemberID: selection.MemberID}, selection.ProductKey, selection.OriginalPublicationID, selection.OriginalSnapshotVersion)
}

// Re-read immutable Source evidence from the original Catalog publication;
// browser flags and edited product text never determine the creative origin.
func (g ArtworkOriginGuard) AuthorizeArtworkSource(ctx context.Context, scope collection.Scope, product, publication string, version uint64) error {
	if g.Snapshots == nil || scope.Validate() != nil {
		return asset.ErrSourceApprovalForbidden
	}
	snapshot, err := g.Snapshots.GetSnapshot(ctx, catalog.SnapshotIdentity{TenantID: scope.OrganizationID, ProductKey: product}, version)
	if err != nil {
		return err
	}
	if snapshot.Identity != (catalog.SnapshotIdentity{TenantID: scope.OrganizationID, ProductKey: product}) || snapshot.PublicationID != publication || snapshot.Version != version {
		return asset.ErrSourceApprovalForbidden
	}
	var origin *ArtworkOrigin
	for _, source := range snapshot.Snapshot.Sources {
		if source.Type != "agent_generated" && source.Platform != "pod_artwork" && source.ReferenceType != imageagent.ArtworkSchema {
			continue
		}
		var parsed ArtworkOrigin
		raw := source.Metadata[artworkOriginKey]
		if origin != nil || source.Type != "agent_generated" || source.Platform != "pod_artwork" || source.ReferenceType != imageagent.ArtworkSchema || len(raw) == 0 || len(raw) > 8192 || json.Unmarshal([]byte(raw), &parsed) != nil {
			return asset.ErrSourceApprovalForbidden
		}
		origin = &parsed
	}
	if origin == nil {
		return nil
	}
	if g.Images == nil || g.Facts == nil || g.Configuration == nil || origin.Protocol != imageagent.ArtworkSchema || imageagent.ValidateOrganizationExecution(origin.Identity, origin.Identity.RunID) != nil || origin.Identity.TenantID != scope.OrganizationID || origin.Identity.UserID != scope.ActorID || origin.Identity.MemberID != scope.MemberID || !agentconfig.ImageDigest(origin.ArtifactHash) {
		return asset.ErrSourceApprovalForbidden
	}
	p, err := g.Images.GetProjection(ctx, imageagent.RunScope{TenantID: scope.OrganizationID, OwnerUserID: scope.ActorID, RunID: origin.Identity.RunID})
	if err != nil {
		return err
	}
	if imageagent.ValidateArtworkAdmission(p.Run, p.Plan) != nil || p.Run.MemberID != scope.MemberID || p.Plan.Revision != origin.PlanRevision || p.Plan.Artwork.Configuration != origin.Configuration || p.Plan.Artwork.ModeSnapshotRef != origin.ModeSnapshot || len(p.Slots) != 1 || len(p.Slots[0].Candidates) != 1 {
		return asset.ErrSourceApprovalForbidden
	}
	slot := p.Slots[0]
	if slot.Slot.ID != origin.SlotID || slot.Slot.Status != imageagent.SlotStatusAccepted || slot.Candidates[0].DurableAsset.SHA256 != origin.ArtifactHash || slot.Candidates[0].Artwork == nil {
		return asset.ErrSourceApprovalForbidden
	}
	id := imageagent.SlotExternalEffectIdentity{RunScope: imageagent.ScopeForRun(p.Run), PlanRevision: p.Plan.Revision, SlotID: slot.Slot.ID, Attempt: slot.Attempt}
	fact, err := g.Facts.ReadGenerationFact(ctx, id)
	if err != nil {
		return err
	}
	closure, err := imageagent.ImageGenerationClosure(fact)
	if err != nil || fact.State != imageagent.GenerationSucceeded || closure.Kind != "settled" || !reflect.DeepEqual(closure, slot.Closure) || fact.IntentID != origin.IntentID || fact.Fingerprint != origin.Fingerprint || fact.Intent.InputDigest != origin.InputDigest || fact.Intent.MemberID != scope.MemberID {
		return asset.ErrSourceApprovalForbidden
	}
	mode, err := g.Configuration.CheckArtworkExecutionMode(ctx, agent.ExecutionIdentity{OrganizationID: scope.OrganizationID, ActorID: scope.ActorID, MemberID: scope.MemberID}, origin.Configuration, origin.ModeSnapshot)
	if err != nil {
		return err
	}
	if mode.Mode == agent.ModeReadOnly {
		return asset.ErrSourceApprovalForbidden
	}
	return nil
}
