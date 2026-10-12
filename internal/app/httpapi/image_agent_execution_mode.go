package httpapi

import (
	"context"
	"reflect"
	"task-processor/internal/agent"
	"task-processor/internal/agentconfig"
	"task-processor/internal/commercetool"
	"task-processor/internal/imageagent"
	configstore "task-processor/internal/integration/persistence/agentconfig"
	"task-processor/internal/product/asset"
)

type modeBoundImageContexts struct {
	imageagent.ImageSetContextReader
	source        imageagent.ImageSetSourceGuard
	configuration *configstore.Store
}

func (c modeBoundImageContexts) RevalidateImageSet(ctx context.Context, identity imageagent.ExecutionIdentity, p imageagent.RunProjection) error {
	if err := c.ImageSetContextReader.RevalidateImageSet(ctx, identity, p); err != nil {
		return err
	}
	_, err := checkImageGenerationMode(ctx, c.configuration, identity, p, false)
	return err
}
func (c modeBoundImageContexts) AuthorizeImageSetSource(ctx context.Context, identity imageagent.ExecutionIdentity, p imageagent.RunProjection) error {
	if c.source == nil {
		return imageagent.ErrCommandBlocked
	}
	if err := c.source.AuthorizeImageSetSource(ctx, identity, p); err != nil {
		return err
	}
	_, err := checkImageGenerationMode(ctx, c.configuration, identity, p, true)
	return err
}
func checkImageGenerationMode(ctx context.Context, configuration *configstore.Store, identity imageagent.ExecutionIdentity, p imageagent.RunProjection, admitted bool) (agentconfig.ModeSnapshot, error) {
	mode, err := loadImageExecutionMode(ctx, configuration, identity, p)
	if err != nil {
		return mode, err
	}
	if admitted {
		if err := imageagent.ValidateImageSetAdmission(p.Run, p.Plan); err != nil {
			return agentconfig.ModeSnapshot{}, err
		}
	}
	action := agent.ExecutionAction{Ref: commercetool.ToolRef{ID: "product.image.generate", Version: "v1.0.0"}, Class: agent.ActionImageGeneration, Risk: commercetool.RiskCompute, SideEffect: commercetool.SideEffectBusinessMutation, RequiresEconomicIntent: true}
	if agent.EvaluateExecutionPolicy(mode.Mode, action, agent.ExecutionPolicyChecks{OwnerAuthorized: true, ContentValid: true, EconomicIntentConfirmed: true}).Outcome != agent.PolicyAllow {
		return agentconfig.ModeSnapshot{}, imageagent.ErrCommandBlocked
	}
	return mode, nil
}

func loadImageExecutionMode(ctx context.Context, configuration *configstore.Store, identity imageagent.ExecutionIdentity, p imageagent.RunProjection) (agentconfig.ModeSnapshot, error) {
	var empty agentconfig.ModeSnapshot
	if configuration == nil || p.Plan.Set == nil || !p.Plan.Set.ModeSnapshotRef.Valid() || p.Run.ID != identity.RunID || p.Run.TenantID != identity.TenantID || p.Run.UserID != identity.UserID || p.Run.MemberID != identity.MemberID {
		return empty, imageagent.ErrCommandBlocked
	}
	snapshot, err := configuration.LoadImageConfiguration(ctx, agent.Scope{OrganizationID: identity.TenantID, ActorID: identity.UserID}, p.Plan.Set.Configuration)
	if err != nil {
		return empty, err
	}
	if snapshot.RunID != p.Run.ID || snapshot.ContextID != p.Run.BusinessTaskID || snapshot.ModeSnapshotRef != p.Plan.Set.ModeSnapshotRef || snapshot.InputDigest != p.Plan.Set.InputDigest {
		return empty, imageagent.ErrRevisionConflict
	}
	mode, err := configuration.CheckImageExecutionMode(ctx, agent.ExecutionIdentity{OrganizationID: identity.TenantID, ActorID: identity.UserID, MemberID: identity.MemberID}, p.Plan.Set.Configuration, p.Plan.Set.ModeSnapshotRef)
	if err != nil {
		return empty, err
	}
	return mode, nil
}

func imageSelectionModeAdmission(repository imageagent.Repository, configuration *configstore.Store) asset.ImageSetMutationAdmission {
	return func(ctx context.Context, source asset.SourceSelection, command asset.ImageSetCommand) error {
		p, err := repository.GetProjection(ctx, imageagent.RunScope{TenantID: source.TenantID, OwnerUserID: source.ActorID, RunID: command.ApprovingResult.RunID})
		if err != nil {
			return err
		}
		identity := imageagent.ExecutionIdentity{ScopeProtocol: p.Run.ScopeProtocol, RunID: p.Run.ID, TenantID: source.TenantID, UserID: source.ActorID, MemberID: source.MemberID, BusinessTaskID: p.Run.BusinessTaskID}
		mode, err := loadImageExecutionMode(ctx, configuration, identity, p)
		if err != nil {
			return err
		}
		if command.ApprovingResult.PlanRevision != p.Plan.Revision || command.ApprovingResult.ResultDigest != p.ResultDigest {
			return imageagent.ErrRevisionConflict
		}
		if command.PolicyAdoption != nil {
			proof := imagePolicyProvenance(mode, p)
			if !reflect.DeepEqual(command.PolicyAdoption, &proof) || !completeImagePolicySelection(p, command) {
				return imageagent.ErrCommandBlocked
			}
		}
		action := agent.ExecutionAction{Ref: commercetool.ToolRef{ID: "product.image.policy.adopt", Version: "v1.0.0"}, Class: agent.ActionLocalWrite, Risk: commercetool.RiskWrite, SideEffect: commercetool.SideEffectBusinessMutation}
		if agent.EvaluateExecutionPolicy(mode.Mode, action, agent.ExecutionPolicyChecks{OwnerAuthorized: true, ContentValid: true, HumanContentConfirmed: command.PolicyAdoption == nil}).Outcome != agent.PolicyAllow {
			return imageagent.ErrCommandBlocked
		}
		return nil
	}
}
func imagePolicyProvenance(mode agentconfig.ModeSnapshot, p imageagent.RunProjection) asset.ImageSetPolicyProvenance {
	return asset.ImageSetPolicyProvenance{Identity: mode.Identity, Mode: mode.Mode, ModeRevision: mode.Revision, ModeSnapshotRef: mode.Ref(), ConfigurationSnapshotRef: p.Plan.Set.Configuration, RunID: p.Run.ID, PlanRevision: p.Plan.Revision, ResultDigest: p.ResultDigest, PolicyVersion: mode.PolicyVersion, ActionVersion: asset.ImageSetPolicyAdoptionVersion}
}
func completeImagePolicySelection(p imageagent.RunProjection, command asset.ImageSetCommand) bool {
	if len(p.RecoverableEffects) > 0 || len(p.Slots) != len(p.Plan.Slots) || len(command.Choices) != len(p.Plan.Slots) || len(p.Plan.Slots) == 0 {
		return false
	}
	for index, declared := range p.Plan.Slots {
		slot := p.Slots[index]
		if slot.Slot.ID != declared.ID || slot.Slot.Status != imageagent.SlotStatusAccepted || slot.Attempt < 1 || len(slot.Candidates) != 1 || declared.Recipe == nil {
			return false
		}
		choice := command.Choices[index]
		if declared.Recipe.OfficialPlacement == nil {
			if choice.OfficialPlacement != nil {
				return false
			}
		} else {
			position := declared.Recipe.OfficialPlacement
			expected := asset.ImageOfficialPlacement{Group: position.Group, SKC: position.SKC, SKU: position.SKU, Type: position.Type, Sort: position.Sort, Site: position.Site}
			if choice.OfficialPlacement == nil || *choice.OfficialPlacement != expected {
				return false
			}
		}
		if choice.Kind != "generated" || choice.RunID != p.Run.ID || choice.PlanRevision != p.Plan.Revision || choice.SlotID != declared.ID || choice.Attempt != slot.Attempt || choice.AssetID != slot.Candidates[0].AssetID || choice.ResultDigest != p.ResultDigest || choice.Presentation.Group != declared.Recipe.Placement.Group || choice.Presentation.Order != declared.Recipe.Placement.Order || choice.Presentation.VariantID != declared.Recipe.Placement.VariantID {
			return false
		}
	}
	return true
}
