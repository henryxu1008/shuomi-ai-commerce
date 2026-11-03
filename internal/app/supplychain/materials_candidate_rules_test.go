package supplychainapp

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
	"task-processor/internal/agent"
	"task-processor/internal/listing/materials"
	record "task-processor/internal/listing/record/target"
	"task-processor/internal/marketplace/shein/goods"
	model "task-processor/internal/marketplace/shein/model"
	"task-processor/internal/product/collection"
	"task-processor/internal/storecenter"
)

type candidateRulesFixtureRules struct {
	uploadRules
	calls int
}

func (r *candidateRulesFixtureRules) ReadCandidateCategoryRules(_ context.Context, _ collection.Scope, _ string, id int64) (storecenter.ProductMerchantBinding, goods.OfficialRuleSnapshot, error) {
	r.calls++
	if id != 124 {
		return r.binding, goods.OfficialRuleSnapshot{}, record.ErrInvalid
	}
	v := r.snapshot
	v.Linked = nil
	v.Categories = []model.Category{{ID: 124, ProductTypeID: 457, Leaf: uploadPointer(true)}}
	v.Attributes.ProductTypeID = 457
	return r.binding, v, nil
}

func TestCandidateRulesSupplyReadPreservesSavedDraftAndFrozenSelector(t *testing.T) {
	a, records, _ := materialsApplication(t)
	rules := &candidateRulesFixtureRules{uploadRules: a.Rules.(uploadRules)}
	a.Rules = rules
	reader, ok := any(a).(interface {
		InspectCandidateRulesRecord(context.Context, string, string) (materials.CandidateRulesDisclosure, error)
	})
	require.True(t, ok, "query must use current native facts without preparing/saving a new Target")
	if !ok {
		return
	}
	before, err := json.Marshal(records.saved)
	require.NoError(t, err)
	v, err := reader.InspectCandidateRulesRecord(materialsReadContext(), records.saved.ID, "124")
	require.NoError(t, err)
	require.Equal(t, "124", v.Binding.CandidateRules.CandidateCategoryID)
	require.Equal(t, "listing-candidate-rules-v1", v.Binding.ContextKind)
	require.EqualValues(t, 123, records.saved.Input.Draft.Product.CategoryID)
	after, err := json.Marshal(records.saved)
	require.NoError(t, err)
	require.Equal(t, before, after)
	require.Equal(t, "not_evaluated", v.Coverage[4].Status)
	bound, ok := any(a).(materials.CandidateRulesReader)
	require.True(t, ok)
	_, err = bound.ReadCandidateRules(materialsReadContext(), v.Binding)
	require.NoError(t, err)
	wrong := v.Binding
	wrong.CandidateRules.CandidateCategoryID = "125"
	_, err = bound.ReadCandidateRules(materialsReadContext(), wrong)
	require.Error(t, err)
	records.saved.Revision++
	_, err = bound.ReadCandidateRules(materialsReadContext(), v.Binding)
	require.Error(t, err)
	_, err = bound.ReadCandidateRules(materialsReadContext(), agent.Binding{})
	require.Error(t, err)
}
