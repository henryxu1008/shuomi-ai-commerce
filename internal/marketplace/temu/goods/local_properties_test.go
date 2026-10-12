package goods

import (
	"github.com/stretchr/testify/require"
	"task-processor/internal/marketplace/temu/model"
	"testing"
)

func TestLocalAttributeControlAndExactComposition(t *testing.T) {
	template := model.LocalAttributeTemplate{Attributes: []model.LocalAttribute{{RefPID: draftPointer(int64(10)), Required: draftPointer(true), Rules: &model.LocalAttributeRules{ControlType: draftPointer(int64(16)), ChooseMaxNum: draftPointer(int64(2)), Values: &model.LocalAttributeValueRules{Precision: draftPointer(int64(0)), ValueRule: draftPointer(int64(1)), Minimum: draftPointer("0"), Maximum: draftPointer("100")}}, ValueGroups: []model.LocalAttributeValueGroup{{Values: []model.LocalAttributeValue{{VID: draftPointer(int64(1)), Value: "cotton"}, {VID: draftPointer(int64(2)), Value: "wool"}}}}}}}
	input := []model.LocalProperty{{RefPID: 10, VID: draftPointer(int64(1)), NumberInputValue: "40"}, {RefPID: 10, VID: draftPointer(int64(2)), NumberInputValue: "60"}}
	template.Attributes[0].Rules.Transnational = draftPointer(false)
	require.Empty(t, CheckLocalProperties(input, template))
	input[1].NumberInputValue = "60.0"
	require.NotEmpty(t, CheckLocalProperties(input, template), "explicit zero precision cannot accept decimals")
	input[1].NumberInputValue = "59"
	require.NotEmpty(t, CheckLocalProperties(input, template), "composition uses exact arithmetic")
	input[1].NumberInputValue = "60"
	input[1].VID = draftPointer(int64(1))
	require.NotEmpty(t, CheckLocalProperties(input, template), "repeated selected value is invalid")
}

func TestLocalAttributeDateAndParentChildGroups(t *testing.T) {
	parent := model.LocalAttribute{RefPID: draftPointer(int64(10)), Required: draftPointer(true), Rules: &model.LocalAttributeRules{ControlType: draftPointer(int64(1)), ChooseMaxNum: draftPointer(int64(1))}, ValueGroups: []model.LocalAttributeValueGroup{{Values: []model.LocalAttributeValue{{VID: draftPointer(int64(1)), Value: "parent", ChildAttributes: []model.LocalChildAttribute{{RefPID: draftPointer(int64(20)), ValueGroupID: "a"}}}}}}}
	child := model.LocalAttribute{RefPID: draftPointer(int64(20)), Required: draftPointer(true), Rules: &model.LocalAttributeRules{ControlType: draftPointer(int64(1)), ChooseMaxNum: draftPointer(int64(1))}, ValueGroups: []model.LocalAttributeValueGroup{{ID: "a", Values: []model.LocalAttributeValue{{VID: draftPointer(int64(2)), Value: "allowed"}}}, {ID: "b", Values: []model.LocalAttributeValue{{VID: draftPointer(int64(3)), Value: "other"}}}}}
	template := model.LocalAttributeTemplate{Attributes: []model.LocalAttribute{parent, child}}
	for i := range template.Attributes {
		template.Attributes[i].Rules.Transnational = draftPointer(false)
	}
	input := []model.LocalProperty{{RefPID: 10, VID: draftPointer(int64(1))}, {RefPID: 20, VID: draftPointer(int64(2))}}
	require.Empty(t, CheckLocalProperties(input, template))
	input[1].VID = draftPointer(int64(3))
	require.NotEmpty(t, CheckLocalProperties(input, template), "child group must come from the selected parent")
	template.Attributes = []model.LocalAttribute{{RefPID: draftPointer(int64(30)), Required: draftPointer(true), Rules: &model.LocalAttributeRules{ControlType: draftPointer(int64(6)), ChooseMaxNum: draftPointer(int64(1))}}}
	template.Attributes[0].Rules.Transnational = draftPointer(false)
	require.Empty(t, CheckLocalProperties([]model.LocalProperty{{RefPID: 30, Value: "2026-02-28,2026-03-01"}}, template))
	require.NotEmpty(t, CheckLocalProperties([]model.LocalProperty{{RefPID: 30, Value: "2026-02-30,2026-03-01"}}, template))
	require.NotEmpty(t, CheckLocalProperties([]model.LocalProperty{{RefPID: 30, Value: "2026-03-02,2026-03-01"}}, template))
}
