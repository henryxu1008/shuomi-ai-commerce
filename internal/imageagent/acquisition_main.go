package imageagent

// ValidateAcquisitionMainPlan accepts the current capture-main plan only. Image
// sets and Artwork have their own current admission owners.
func ValidateAcquisitionMainPlan(target string, policy ImagePolicyContext, plan Plan) error {
	if target != "product" || policy != (ImagePolicyContext{Country: "zz", Family: "default", SceneCategory: "general"}) || plan.Set != nil || plan.Artwork != nil || len(plan.Slots) != 1 || len(plan.SourceAssetIDs) != 1 || len(plan.StyleReferenceIDs) != 0 {
		return ErrValidation
	}
	slot := plan.Slots[0]
	if slot.Role != SlotRoleMain || slot.Recipe != nil || len(slot.SourceAssetIDs) != 1 || slot.SourceAssetIDs[0] != plan.SourceAssetIDs[0] || len(slot.StyleReferenceIDs) != 0 {
		return ErrValidation
	}
	return nil
}
func ValidateAcquisitionMainExecution(input SlotExecutionInput) error {
	if input.Artwork != nil || input.ImageSet != nil || input.ImagePolicyContext == nil {
		return ErrValidation
	}
	return ValidateAcquisitionMainPlan(input.TargetPlatform, *input.ImagePolicyContext, Plan{SourceAssetIDs: input.Slot.SourceAssetIDs, Slots: []Slot{input.Slot}})
}
