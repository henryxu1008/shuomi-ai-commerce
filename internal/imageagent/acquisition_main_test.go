package imageagent

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAcquisitionMainRejectsRetiredPlans(t *testing.T) {
	base := Plan{SourceAssetIDs: []string{"source"}, Slots: []Slot{{ID: "main", Role: SlotRoleMain, SourceAssetIDs: []string{"source"}}}}
	policy := ImagePolicyContext{Country: "zz", Family: "default", SceneCategory: "general"}
	require.NoError(t, ValidateAcquisitionMainPlan("product", policy, base))
	for _, mutate := range []func(*Plan){func(p *Plan) { p.Slots = append(p.Slots, p.Slots[0]) }, func(p *Plan) { p.Slots[0].Role = SlotRoleScene }, func(p *Plan) { p.SourceAssetIDs = []string{"other"} }, func(p *Plan) { p.StyleReferenceIDs = []string{"style"} }, func(p *Plan) { p.Slots[0].Recipe = &ImageSlotRecipe{} }} {
		p := base
		p.Slots = append([]Slot(nil), base.Slots...)
		mutate(&p)
		require.ErrorIs(t, ValidateAcquisitionMainPlan("product", policy, p), ErrValidation)
	}
	require.ErrorIs(t, ValidateAcquisitionMainPlan("shein", policy, base), ErrValidation)
}
