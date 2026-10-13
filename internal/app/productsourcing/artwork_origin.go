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
)

type ArtworkOriginGuard struct {
	Snapshots     catalog.VersionedSnapshotReader
	Images        imageagent.Repository
	Facts         imageagent.GenerationFactRepository
	Configuration agentconfig.ArtworkConfigurationRepository
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
