package httpapi

import (
	"context"
	"fmt"
	"github.com/google/uuid"
	"task-processor/internal/agent"
	"task-processor/internal/imageagent"
	configstore "task-processor/internal/integration/persistence/agentconfig"
	"task-processor/internal/product/asset"
)

type imageSetPolicyAdopter struct {
	repository    imageagent.Repository
	configuration *configstore.Store
	selections    *asset.ImageSetService
	inventories   asset.ImageSetInventoryReader
}

func (a imageSetPolicyAdopter) PreparePolicyAdoption(ctx context.Context, identity imageagent.ExecutionIdentity) (*asset.ImageSetCommand, error) {
	p, err := a.repository.GetProjection(ctx, imageagent.RunScope{TenantID: identity.TenantID, OwnerUserID: identity.UserID, RunID: identity.RunID})
	if err != nil {
		return nil, err
	}
	mode, err := loadImageExecutionMode(ctx, a.configuration, identity, p)
	if err != nil {
		return nil, err
	}
	if mode.Mode != agent.ModeFullAccess {
		return nil, nil
	}
	if p.Run.Status != imageagent.RunStatusAwaitingFinalApproval || p.PendingCommand != nil || p.ResultDigest == "" || len(p.RecoverableEffects) > 0 || len(p.Slots) != len(p.Plan.Slots) {
		return nil, nil
	}
	choices := make([]asset.ImageSetChoice, 0, len(p.Plan.Slots))
	for index, declared := range p.Plan.Slots {
		slot := p.Slots[index]
		if slot.Slot.ID != declared.ID || slot.Slot.Status != imageagent.SlotStatusAccepted || len(slot.Candidates) != 1 || declared.Recipe == nil {
			return nil, nil
		}
		recipe := declared.Recipe
		choice := asset.ImageSetChoice{Kind: "generated", RunID: p.Run.ID, PlanRevision: p.Plan.Revision, SlotID: declared.ID, Attempt: slot.Attempt, ResultDigest: p.ResultDigest, AssetID: slot.Candidates[0].AssetID, Presentation: asset.ImagePresentation{Group: recipe.Placement.Group, Order: recipe.Placement.Order, VariantID: recipe.Placement.VariantID}}
		if position := recipe.OfficialPlacement; position != nil {
			choice.OfficialPlacement = &asset.ImageOfficialPlacement{Group: position.Group, SKC: position.SKC, SKU: position.SKU, Type: position.Type, Sort: position.Sort, Site: position.Site}
		}
		choices = append(choices, choice)
	}
	inventory, err := a.inventories.ReadImageSetInventory(ctx, asset.InventoryScope{TenantID: identity.TenantID, ProductKey: p.Plan.Set.Source.ProductID, TargetPlatform: p.Plan.Set.Target.Platform, SourceSnapshotVersion: p.Plan.Set.Source.EffectiveVersion})
	if err != nil {
		return nil, err
	}
	actionID := uuid.NewSHA1(uuid.NameSpaceOID, []byte(fmt.Sprintf("image-policy:%s:%d:%s:%s", p.Run.ID, p.Plan.Revision, p.ResultDigest, mode.Ref().Digest))).String()
	command, err := fullImageSelection(p, fullImageSelectionBody{ActionID: actionID, PlanRevision: p.Plan.Revision, ResultDigest: p.ResultDigest, ExpectedHead: inventory.Head, Choices: choices})
	if err != nil {
		return nil, err
	}
	proof := imagePolicyProvenance(mode, p)
	command.PolicyAdoption = &proof
	if !proof.Valid() || !completeImagePolicySelection(p, command) {
		return nil, imageagent.ErrCommandBlocked
	}
	preview, err := a.selections.Preview(ctx, command)
	if err != nil {
		return nil, err
	}
	command.SelectionDigest = preview.Digest
	return &command, nil
}
