package productsourcing

import (
	"context"
	"encoding/json"
	"reflect"

	"task-processor/internal/agent"
	"task-processor/internal/agentconfig"
	"task-processor/internal/imageagent"
	"task-processor/internal/product/collection"
	"task-processor/internal/product/sourcing"
)

const artworkOriginKey = "pod_artwork_origin_v1"

type ArtworkOrigin struct {
	Protocol, ContentType                                    string
	Identity                                                 imageagent.ExecutionIdentity
	PlanRevision                                             int64
	SlotID, ArtifactHash, InputDigest, IntentID, Fingerprint string
	Configuration                                            agent.ConfigurationSnapshotRef
	ModeSnapshot                                             agent.ModeSnapshotRef
}

// The existing image execution, materialization, Collection and Source owners
// supply every fact. This receiver has no model, remote mutation or new ledger.
type ArtworkReceiver struct {
	Images        imageagent.Repository
	Facts         imageagent.GenerationFactRepository
	Artifacts     imageagent.SlotExternalEffectV3Repository
	URLs          imageagent.DurableAssetPublicURLResolver
	Configuration agentconfig.ArtworkConfigurationRepository
	Collections   collection.Repository
	Authorize     collection.Authorizer
}

func (r *ArtworkReceiver) PublishArtwork(ctx context.Context, in imageagent.PublishArtworkInput) (imageagent.ArtworkAdoptionReceipt, error) {
	var empty imageagent.ArtworkAdoptionReceipt
	if r == nil || r.Images == nil || r.Facts == nil || r.Artifacts == nil || r.URLs == nil || r.Configuration == nil || r.Collections == nil || r.Authorize == nil || imageagent.ValidateOrganizationExecution(in.Identity, in.Identity.RunID) != nil {
		return empty, collection.ErrUnavailable
	}
	scope := collection.Scope{OrganizationID: in.Identity.TenantID, ActorID: in.Identity.UserID, MemberID: in.Identity.MemberID}
	p, err := r.Images.GetProjection(ctx, imageagent.RunScope{TenantID: scope.OrganizationID, OwnerUserID: scope.ActorID, RunID: in.Identity.RunID})
	if err != nil {
		return empty, err
	}
	if imageagent.ValidateArtworkAdmission(p.Run, p.Plan) != nil || p.Run.MemberID != scope.MemberID || p.Run.BusinessTaskID != in.Identity.BusinessTaskID || p.Plan.Revision != in.PlanRevision || p.ResultDigest != in.ResultDigest || p.PendingCommand == nil || p.PendingCommand.ActionID != in.ActionID || p.PendingCommand.Kind != "approve_results" || p.PendingCommand.Phase != imageagent.ImageSetApprovalPublicationStarted {
		return empty, imageagent.ErrRevisionConflict
	}
	digest, err := imageagent.ArtworkResultDigest(p.Plan, p.Slots, p.RecoverableEffects)
	if err != nil || digest != in.ResultDigest {
		return empty, imageagent.ErrRevisionConflict
	}
	slot := p.Slots[0]
	id := imageagent.SlotExternalEffectIdentity{RunScope: imageagent.ScopeForRun(p.Run), PlanRevision: in.PlanRevision, SlotID: slot.Slot.ID, Attempt: slot.Attempt}
	fact, err := r.Facts.ReadGenerationFact(ctx, id)
	if err != nil {
		return empty, err
	}
	effect, err := r.Artifacts.GetSlotExternalEffectV3(ctx, id)
	if err != nil {
		return empty, err
	}
	execution := imageagent.SlotExecutionInput{Artwork: imageagent.CloneArtworkPlan(p.Plan.Artwork), OrganizationIdentity: in.Identity, RunID: p.Run.ID, TenantID: scope.OrganizationID, UserID: scope.ActorID, TargetPlatform: "sds", PlanRevision: in.PlanRevision, Slot: p.Plan.Slots[0], Attempt: slot.Attempt, IdempotencyKey: effect.IdempotencyKey, AssetCatalog: p.AssetCatalog}
	provenance, err := imageagent.NewArtworkProvenance(execution, fact)
	if err != nil || fact.State != imageagent.GenerationSucceeded || fact.Intent.MemberID != scope.MemberID || fact.Success.ResultUnavailable != "" || effect.Identity != id || effect.InputFingerprint != imageagent.SlotExecutionFingerprint(execution) || effect.Phase != imageagent.SlotEffectV3PublicationComplete || len(effect.FinalManifest.Assets) != 1 || imageagent.ValidateSlotEffectV3Completion(effect.Published, effect.FinalManifest, effect.ResultFingerprint) != nil {
		return empty, imageagent.ErrRevisionConflict
	}
	closure, err := imageagent.ImageGenerationClosure(fact)
	if err != nil || closure.Kind != "settled" || !reflect.DeepEqual(slot.Closure, closure) {
		return empty, imageagent.ErrRevisionConflict
	}
	candidate, ref := slot.Candidates[0], effect.FinalManifest.Assets[0]
	proof := &imageagent.ImageGenerationProof{IntentID: fact.IntentID, Fingerprint: fact.Fingerprint, SettlementProofDigest: fact.TerminalProofDigest(), Points: fact.Intent.Points}
	if !reflect.DeepEqual(candidate.Artwork, &provenance) || !reflect.DeepEqual(candidate.GenerationProof, proof) || imageagent.ValidatePublishedAssetRefForSlot(execution, ref, 0) != nil || candidate.DurableAsset.ObjectKey != ref.ObjectKey || candidate.DurableAsset.SHA256 != ref.SHA256 || candidate.AssetID != effect.Published.Candidates[0].AssetID {
		return empty, imageagent.ErrRevisionConflict
	}
	url, err := imageagent.ResolvePublishedAssetURL(execution, candidate.DurableAsset, 0, r.URLs, nil)
	if err != nil {
		return empty, err
	}
	origin := artworkOriginFromRun(p, ref)
	key := collection.StableID(scope.OrganizationID, scope.ActorID, p.Run.ID, slot.Slot.ID, ref.SHA256)
	op := collection.StableID(scope.OrganizationID, scope.ActorID, key)
	envelope, err := artworkEnvelope(op, p.Plan.Artwork, origin, ref, url)
	if err != nil {
		return empty, err
	}
	command := collection.Command{Scope: scope, Key: key, OperationID: op, InputHash: collection.Digest(envelope), Mutation: collection.Mutation{Action: "adopt_artwork"}, Envelope: &envelope, ArtworkWriteGuard: func(ctx context.Context) error {
		fresh, err := r.Authorize.Authorize(ctx, collection.PermissionManage)
		if err != nil || fresh != scope {
			return collection.ErrForbidden
		}
		mode, err := r.Configuration.CheckArtworkExecutionMode(ctx, agent.ExecutionIdentity{OrganizationID: scope.OrganizationID, ActorID: scope.ActorID, MemberID: scope.MemberID}, origin.Configuration, origin.ModeSnapshot)
		if err != nil {
			return err
		}
		if mode.Mode == agent.ModeReadOnly {
			return collection.ErrForbidden
		}
		return nil
	}}
	receipt, err := r.Collections.Execute(ctx, command)
	if err != nil {
		return empty, err
	}
	item, err := r.Collections.ReadItem(ctx, scope, receipt.ItemID)
	if err != nil {
		return empty, err
	}
	if item.Source.Kind != "own" || item.Source.PublicationID != op || item.Source.Version == 0 {
		return empty, collection.ErrUnavailable
	}
	return imageagent.ArtworkAdoptionReceipt{RunID: p.Run.ID, PlanRevision: in.PlanRevision, SlotID: slot.Slot.ID, ArtifactHash: ref.SHA256, OperationID: op, BatchID: receipt.BatchID, ItemID: receipt.ItemID, ProductKey: item.Source.ProductKey, PublicationID: item.Source.PublicationID, SourceVersion: item.Source.Version}, nil
}

func artworkOriginFromRun(p imageagent.RunProjection, ref imageagent.PublishedAssetRef) ArtworkOrigin {
	return ArtworkOrigin{Protocol: imageagent.ArtworkSchema, ContentType: ref.ContentType, Identity: imageagent.ExecutionIdentity{ScopeProtocol: p.Run.ScopeProtocol, RunID: p.Run.ID, TenantID: p.Run.TenantID, UserID: p.Run.UserID, MemberID: p.Run.MemberID, BusinessTaskID: p.Run.BusinessTaskID}, PlanRevision: p.Plan.Revision, SlotID: ref.Artwork.SlotID, ArtifactHash: ref.SHA256, InputDigest: ref.Artwork.InputDigest, IntentID: ref.Artwork.IntentID, Fingerprint: ref.Artwork.Fingerprint, Configuration: p.Plan.Artwork.Configuration, ModeSnapshot: p.Plan.Artwork.ModeSnapshotRef}
}

func artworkEnvelope(id string, plan *imageagent.ArtworkPlan, origin ArtworkOrigin, ref imageagent.PublishedAssetRef, url string) (sourcing.SourceEnvelope, error) {
	if plan == nil || origin.Protocol != imageagent.ArtworkSchema || ref.Artwork == nil || ref.SHA256 != origin.ArtifactHash {
		return sourcing.SourceEnvelope{}, collection.ErrInvalid
	}
	raw, err := json.Marshal(origin)
	if err != nil {
		return sourcing.SourceEnvelope{}, err
	}
	e := sourcing.SourceEnvelope{Identity: sourcing.SourceIdentity{SourceType: "agent_generated", SourcePlatform: "pod_artwork", SourceID: id}, RawReference: sourcing.RawSourceReference{ReferenceType: imageagent.ArtworkSchema, ReferenceID: id, Checksum: sourcing.RawSnapshotChecksum(string(raw)), Metadata: map[string]string{artworkOriginKey: string(raw)}}, ProductCandidate: sourcing.ProductCandidate{Title: plan.Theme, Description: plan.Requirements}, AssetCandidates: []sourcing.AssetCandidate{{SourceID: collection.StableID(id, ref.SHA256), URL: url, MediaType: "image", Role: "main", Checksum: "sha256:" + ref.SHA256, Width: ref.Width, Height: ref.Height}}, Trace: sourcing.SourceTrace{SourceRunID: origin.Identity.RunID}}
	return sourcing.NormalizePublicationEnvelope(e)
}
