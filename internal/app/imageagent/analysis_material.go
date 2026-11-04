package imageagentapp

import (
	"context"
	"reflect"
	"strconv"
	"task-processor/internal/agent"
	"task-processor/internal/agentconfig"
	"task-processor/internal/aicapability"
	"task-processor/internal/authidentity"
	"task-processor/internal/imageagent"
	"task-processor/internal/product/asset"
	"task-processor/internal/product/catalog"
)

// Ordinary source/approved bytes use existing owners. Caller AnalysisSources
// freshly authorizes the exact product and inventory before reaching this port.
type AnalysisMaterialReader struct {
	Snapshots     catalog.VersionedSnapshotReader
	ReadPublic    SourceByteReader
	ReadGenerated GeneratedMaterialByteReader
	ReadOwned     func(context.Context, agent.Binding, string, string, int64) ([]byte, bool, error)
	ReadOrigin    func(context.Context, agent.Binding, catalog.IdentifiedImage, int64) ([]byte, error)
}

func (r AnalysisMaterialReader) ReadAnalysisSource(ctx context.Context, b agent.Binding, img catalog.IdentifiedImage, maximum int64) ([]byte, error) {
	i, ok := authidentity.AuthenticatedIdentityFromContext(ctx)
	if !ok || r.Snapshots == nil || !agentconfig.ImageAnalysisBinding(b) || maximum < 1 || maximum > aicapability.MaximumSingleImageBytes {
		return nil, agent.ErrInvalid
	}
	version, err := strconv.ParseUint(b.CatalogVersion, 10, 63)
	if err != nil {
		return nil, err
	}
	s, err := r.Snapshots.GetSnapshot(ctx, catalog.SnapshotIdentity{TenantID: i.TenantID, ProductKey: b.ProductKey}, version)
	if err != nil {
		return nil, err
	}
	if s.PublicationID != b.PublicationID || s.Version != version || s.Identity != (catalog.SnapshotIdentity{TenantID: i.TenantID, ProductKey: b.ProductKey}) {
		return nil, agent.ErrConflict
	}
	found := false
	for _, original := range catalog.IdentifyImages(s.Snapshot.Images) {
		if reflect.DeepEqual(original, img) {
			found = true
			break
		}
	}
	if !found {
		return nil, agent.ErrConflict
	}
	for _, origin := range s.Snapshot.Sources {
		if origin.Type == "agent_generated" || origin.Platform == "pod_artwork" || origin.ReferenceType == imageagent.ArtworkSchema {
			if r.ReadOrigin == nil {
				return nil, agent.ErrUnavailable
			}
			raw, e := r.ReadOrigin(ctx, b, img, maximum)
			return boundedAnalysisMaterial(raw, maximum, e)
		}
	}
	return r.readSourceURL(ctx, b, img.ID, img.URL, i.UserID, maximum, false)
}

func (r AnalysisMaterialReader) ReadAnalysisApproved(ctx context.Context, b agent.Binding, img asset.ApprovedAsset, maximum int64) ([]byte, error) {
	i, ok := authidentity.AuthenticatedIdentityFromContext(ctx)
	if !ok || !agentconfig.ImageAnalysisBinding(b) || maximum < 1 || maximum > aicapability.MaximumSingleImageBytes {
		return nil, agent.ErrInvalid
	}
	if img.GenerationEvidence != nil {
		if img.SourceApproval != nil || r.ReadGenerated == nil {
			return nil, agent.ErrUnavailable
		}
		version, _ := strconv.ParseUint(b.CatalogVersion, 10, 63)
		raw, err := r.ReadGenerated(ctx, asset.SourceSelection{TenantID: i.TenantID, ActorID: i.UserID, MemberID: i.EffectiveMemberID, ProductKey: b.ProductKey, OriginalPublicationID: b.PublicationID, OriginalSnapshotVersion: version, EffectiveCatalogVersion: version, TargetPlatform: b.TargetPlatform}, img, maximum)
		return boundedAnalysisMaterial(raw, maximum, err)
	}
	if img.SourceApproval == nil || img.SourceApproval.ReferenceHash != asset.ReferenceHash(img.SourceAssetID, img.URL) {
		return nil, agent.ErrUnavailable
	}
	raw, err := r.readSourceURL(ctx, b, img.SourceAssetID, img.URL, img.SourceApproval.ActorID, maximum, img.SourceApproval.ManualMedia != nil)
	if err == nil && img.SourceApproval.ManualMedia != nil && (int64(len(raw)) != img.SourceApproval.ManualMedia.Bytes || analysisEvidenceHash(raw) != img.SourceApproval.ManualMedia.Hash) {
		return nil, agent.ErrConflict
	}
	return raw, err
}

func (r AnalysisMaterialReader) readSourceURL(ctx context.Context, b agent.Binding, id, url, owner string, maximum int64, manual bool) ([]byte, error) {
	if r.ReadOwned != nil {
		raw, handled, err := r.ReadOwned(ctx, b, url, owner, maximum)
		if handled || err != nil {
			return boundedAnalysisMaterial(raw, maximum, err)
		}
	}
	if manual || r.ReadPublic == nil {
		return nil, agent.ErrUnavailable
	}
	if _, err := imageagent.ValidateSafeImageURL(url); err != nil {
		return nil, err
	}
	raw, err := r.ReadPublic(ctx, imageagent.AuthorizedAsset{ID: id, URL: url, Type: imageagent.AuthorizedAssetSource}, maximum)
	return boundedAnalysisMaterial(raw, maximum, err)
}

func boundedAnalysisMaterial(raw []byte, maximum int64, err error) ([]byte, error) {
	if err != nil {
		return nil, err
	}
	if len(raw) == 0 || int64(len(raw)) > maximum {
		return nil, agent.ErrInvalid
	}
	return append([]byte(nil), raw...), nil
}
