package imageagent

import (
	"context"
	"reflect"
	"slices"
	"strings"

	"github.com/henryxu1008/shuomi-ai-commerce/internal/agent"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/agentconfig"
	productasset "github.com/henryxu1008/shuomi-ai-commerce/internal/product/asset"
)

type ColorImageReworkRequest struct {
	SlotID      string                          `json:"slotId"`
	Head        productasset.ImageInventoryHead `json:"head"`
	Composition string                          `json:"composition,omitempty"`
}

type confirmedColorImageReworkReader interface {
	FindConfirmedColorImageRework(context.Context, RunScope, string) (RunProjection, error)
}

func (s *Service) confirmedColorImageRework(ctx context.Context, identity ExecutionIdentity, current RunProjection) (RunProjection, error) {
	reader, ok := s.repository.(confirmedColorImageReworkReader)
	if !ok {
		return current, ErrColorImageReworkConfirmed
	}
	key := ConfirmedColorImageReworkKey(current.Plan)
	winner, err := reader.FindConfirmedColorImageRework(ctx, ScopeForRun(current.Run), key)
	if err != nil || ValidateImageSetAdmission(winner.Run, winner.Plan) != nil || key != ConfirmedColorImageReworkKey(winner.Plan) {
		return current, ErrColorImageReworkConfirmed
	}
	if identity, err = s.identityForRun(identity, winner.Run); err != nil {
		return current, ErrColorImageReworkConfirmed
	}
	if err = s.imageSets.Contexts.RevalidateImageSet(ctx, identity, winner); err != nil {
		return current, ErrColorImageReworkConfirmed
	}
	return winner, ErrColorImageReworkConfirmed
}

type ColorImageReworkBinding struct {
	ParentRevision  int64
	SlotID          string
	Attempt         int
	CandidateID     string
	ArtifactHash    string
	ReplacedAssetID string
	Count           int
	Head            productasset.ImageInventoryHead
}

func (b ColorImageReworkBinding) valid() bool {
	return b.ParentRevision > 0 && canonicalImageValue(b.SlotID) && b.Attempt > 0 &&
		canonicalImageValue(b.CandidateID) && agentconfig.ImageDigest(b.ArtifactHash) &&
		canonicalImageValue(b.ReplacedAssetID) && b.Count >= 2 && b.Count <= MaxPlanSlots &&
		agentconfig.UUID(b.Head.ActionID) && b.Head.Valid()
}

func (s *ImageSetPlan) ColorImageRework() *ColorImageReworkBinding {
	if s == nil || s.Regeneration == nil {
		return nil
	}
	return s.Regeneration.ColorRework
}

// ConfirmedColorImageReworkKey depends only on the original business position.
// Input variations and new request keys cannot obtain another confirmation.
func ConfirmedColorImageReworkKey(plan Plan) string {
	binding := plan.Set.ColorImageRework()
	if binding == nil {
		return ""
	}
	return generationHash(struct {
		Source       ImageSourceBinding
		Target       ImageTarget
		Action, Slot string
	}{colorReworkSourceIdentity(plan.Set.Source), plan.Set.Target, binding.Head.ActionID, binding.SlotID})
}

func colorReworkSourceIdentity(source ImageSourceBinding) ImageSourceBinding {
	source.CatalogHash = ""
	return source
}

func validImageComposition(value string) bool {
	return agentconfig.ValidSetText(value, agentconfig.MaxSetBriefBytes) &&
		!strings.ContainsAny(value, "\n\t\u2028\u2029")
}

func imageSetReworkResultBound(set *ImageSetPlan) bool {
	return set.Regeneration.ResultDigest != "" && set.Regeneration.ResultDigest == set.Regeneration.ClosedEffectsDigest
}

type colorImageReworkPreparation struct {
	parent       RunProjection
	slot         Slot
	regeneration *ImageSetRegeneration
}

func ColorImageReworkParent(projection RunProjection) bool {
	set := projection.Plan.Set
	return set != nil && set.ColorImages != nil && set.Regeneration == nil &&
		projection.Run.Status == RunStatusCompleted && projection.PendingCommand == nil &&
		len(projection.RecoverableEffects) == 0 && projection.ResultDigest != "" &&
		len(projection.Plan.Slots) > 0 && projection.Plan.Slots[0].Recipe != nil &&
		projection.Plan.Slots[0].Recipe.Quote.Provider == "grsai"
}

func (s *Service) prepareColorImageRework(ctx context.Context, identity ExecutionIdentity, input *PrepareImageSetInput) (colorImageReworkPreparation, error) {
	var result colorImageReworkPreparation
	parent, err := s.repository.GetProjection(ctx, RunScope{TenantID: identity.TenantID, OwnerUserID: identity.UserID, RunID: input.RegenerateFromRunID})
	if err != nil {
		return result, err
	}
	if _, err = s.identityForRun(identity, parent.Run); err != nil {
		return result, err
	}
	if !ColorImageReworkParent(parent) || parent.Run.BusinessTaskID != input.ContextID ||
		ValidateImageSetAdmission(parent.Run, parent.Plan) != nil {
		return result, ErrCommandBlocked
	}
	set := parent.Plan.Set
	if set.Source.ContextKind != input.ContextKind || set.Source.EffectiveVersion != input.EffectiveCatalogVersion ||
		set.Source.ApplyReceiptID != input.ApplyReceiptID || !reflect.DeepEqual(set.Route, input.Route) {
		return result, ErrRevisionConflict
	}
	var skill *agent.SkillVersionRef
	if set.Method != nil {
		skill = &set.Method.Skill
	}
	if !reflect.DeepEqual(skill, input.Method) {
		return result, ErrRevisionConflict
	}
	closed, err := ImageSetClosedEffectsDigest(parent.Plan, parent.Slots, parent.RecoverableEffects)
	if err != nil || closed != parent.ResultDigest {
		return result, ErrCommandBlocked
	}
	index := slices.IndexFunc(parent.Slots, func(slot SlotProjection) bool { return slot.Slot.ID == input.ColorRework.SlotID })
	if index < 0 || parent.Slots[index].Slot.Status != SlotStatusAccepted {
		return result, ErrValidation
	}
	target := parent.Slots[index]
	inventory, err := s.readColorImageReworkBaseline(ctx, identity.TenantID, set.Source, input.ColorRework.Head)
	if err != nil {
		return result, err
	}
	commit, err := s.imageSets.ColorReworkApprovals.ReadApprovalCommit(ctx, identity.TenantID, input.ColorRework.Head.ActionID)
	if err != nil {
		return result, err
	}
	expectedResult := productasset.ImageSetResultBinding{RunID: parent.Run.ID, PlanRevision: parent.Plan.Revision, ResultDigest: closed}
	if commit.ImageSet == nil || commit.ImageSet.PolicyAdoption != nil || commit.TenantID != identity.TenantID ||
		commit.ActionID != inventory.ApprovalActionID || commit.ProductKey != set.Source.ProductID ||
		commit.TargetPlatform != set.Target.Platform || commit.SourceSnapshotVersion != set.Source.EffectiveVersion ||
		commit.ImageSet.Source != imageSetAssetSource(set) || !slices.Equal(commit.ImageSet.Results, []productasset.ImageSetResultBinding{expectedResult}) ||
		len(commit.Assets) != len(parent.Slots) || !reflect.DeepEqual(commit.Assets, inventory.Assets) {
		return result, ErrRevisionConflict
	}
	slots := make(map[string]SlotProjection, len(parent.Slots))
	for _, slot := range parent.Slots {
		slots[slot.Slot.ID] = slot
	}
	var replacedAssetID string
	for _, approved := range commit.Assets {
		slot, ok := slots[approved.SlotID]
		if !ok || slot.Slot.Status != SlotStatusAccepted || !matchesColorReworkBaselineAsset(approved, expectedResult, slot) {
			return result, ErrRevisionConflict
		}
		delete(slots, approved.SlotID)
		if approved.SlotID == target.Slot.ID {
			replacedAssetID = approved.ID
		}
	}
	if replacedAssetID == "" {
		return result, ErrRevisionConflict
	}
	candidate := target.Candidates[0]
	input.ColorImages = CloneImageSetPlan(set).ColorImages
	input.SharedOriginalIDs = slices.Clone(target.Slot.SourceAssetIDs)
	return colorImageReworkPreparation{
		parent: parent, slot: parent.Plan.Slots[index],
		regeneration: &ImageSetRegeneration{RunID: parent.Run.ID, ClosedEffectsDigest: closed, ResultDigest: parent.ResultDigest,
			ColorRework: &ColorImageReworkBinding{ParentRevision: parent.Plan.Revision, SlotID: target.Slot.ID, Attempt: target.Attempt,
				CandidateID: candidate.AssetID, ArtifactHash: candidate.DurableAsset.SHA256, ReplacedAssetID: replacedAssetID,
				Count: len(commit.Assets), Head: inventory.Head}},
	}, nil
}

func matchesColorReworkBaselineAsset(approved productasset.ApprovedAsset, result productasset.ImageSetResultBinding, slot SlotProjection) bool {
	if len(slot.Candidates) != 1 || approved.GenerationEvidence == nil || approved.Presentation == nil {
		return false
	}
	candidate := slot.Candidates[0]
	proof := candidate.GenerationProof
	return approved.RunID == result.RunID && approved.PlanRevision == result.PlanRevision &&
		approved.Attempt == slot.Attempt && approved.SourceAssetID == candidate.SourceAssetID &&
		*approved.Presentation == imageSetPresentation(slot.Slot.Recipe.Placement) &&
		approved.GenerationEvidence.ArtifactHash == candidate.DurableAsset.SHA256 &&
		approved.GenerationEvidence.IntentID == proof.IntentID && approved.GenerationEvidence.Fingerprint == proof.Fingerprint &&
		approved.GenerationEvidence.SettlementProofDigest == proof.SettlementProofDigest &&
		approved.GenerationEvidence.ProviderUsageProofDigest == proof.ProviderUsageProofDigest
}

func (s *Service) readColorImageReworkBaseline(ctx context.Context, tenant string, source ImageSourceBinding, head productasset.ImageInventoryHead) (productasset.ImageSetInventory, error) {
	if s.imageSets.ColorReworkApprovals == nil {
		return productasset.ImageSetInventory{}, ErrCommandBlocked
	}
	return readColorReworkInventory(ctx, s.imageSets.ColorReworkInventory, tenant, source, head)
}

func readColorReworkInventory(ctx context.Context, reader productasset.ImageSetInventoryReader, tenant string, source ImageSourceBinding, head productasset.ImageInventoryHead) (productasset.ImageSetInventory, error) {
	if reader == nil {
		return productasset.ImageSetInventory{}, ErrCommandBlocked
	}
	inventory, err := reader.ReadImageSetInventory(ctx, productasset.InventoryScope{
		TenantID: tenant, ProductKey: source.ProductID, SourceSnapshotVersion: source.EffectiveVersion, TargetPlatform: "product",
	})
	if err != nil {
		return productasset.ImageSetInventory{}, err
	}
	if inventory.Head != head || inventory.ApprovalActionID != head.ActionID {
		return productasset.ImageSetInventory{}, ErrRevisionConflict
	}
	return inventory, nil
}

func (r colorImageReworkPreparation) build(input PrepareImageSetInput, resolved ImageSetPreparation, snapshot agentconfig.ImageConfigurationSnapshot, quote ImageGenerationQuote, method *ImageMethodBinding, text *ImageTextBinding) (Plan, error) {
	parent := r.parent.Plan.Set
	if !SameImageSetRegenerationContext(parent.Source, resolved.Source, parent.Target, resolved.Target) ||
		!reflect.DeepEqual(r.parent.AssetCatalog.ProductContext, resolved.Catalog.ProductContext) ||
		!slices.Equal(r.slot.Recipe.References, resolved.Observations) ||
		snapshot.ParametersDigest != parent.ParametersDigest || !reflect.DeepEqual(parent.Method, method) ||
		quote.Provider != "grsai" || !quote.UsesProviderUsage() {
		return Plan{}, ErrRevisionConflict
	}
	slot := r.slot
	slot.Recipe = CloneImageSlotRecipe(slot.Recipe)
	recipe := slot.Recipe
	key := agentconfig.RequiredTaskEvidence(recipe.Purpose)
	if key != "" {
		if resolved.Evidence[key] != recipe.TaskEvidence {
			return Plan{}, ErrRevisionConflict
		}
		recipe.EvidenceDigest = generationHash(struct {
			Source     ImageSourceBinding
			Key, Value string
		}{resolved.Source, key, recipe.TaskEvidence})
	}
	recipe.Quote, recipe.Composition = quote, input.ColorRework.Composition
	recipe.Prompt = imageTextPrompt(resolved.Catalog.ProductContext, slot.Brief, recipe)
	if recipe.Composition == "" && recipe.Prompt != r.slot.Recipe.Prompt {
		return Plan{}, ErrRevisionConflict
	}
	return Plan{Revision: 1, IdempotencyKey: input.RequestID, CreatedBy: snapshot.Scope.ActorID,
		SourceAssetIDs: slices.Clone(slot.SourceAssetIDs), Slots: []Slot{slot},
		Set: &ImageSetPlan{Schema: ImageSetSchema, Source: resolved.Source, Target: resolved.Target,
			Configuration: snapshot.Ref(), ConfigurationEpoch: snapshot.Epoch, ParametersDigest: snapshot.ParametersDigest,
			InputDigest: snapshot.InputDigest, Route: CloneImageRoute(input.Route), Method: method, Text: text,
			ColorImages: input.ColorImages, Regeneration: r.regeneration}}, nil
}

func imageSetPresentation(p ImagePlacement) productasset.ImagePresentation {
	return productasset.ImagePresentation{Color: p.Color, Group: p.Group, Order: p.Order, VariantID: p.VariantID}
}

func imageSetAssetSource(set *ImageSetPlan) productasset.SourceSelectionRequest {
	source := set.Source
	return productasset.SourceSelectionRequest{ContextKind: string(source.ContextKind), ItemID: source.OperationID,
		OriginalPublicationID: source.OriginalPublicationID, OriginalSnapshotVersion: source.OriginalVersion,
		EffectiveCatalogVersion: source.EffectiveVersion, ApplyReceiptID: source.ApplyReceiptID, TargetPlatform: set.Target.Platform}
}
