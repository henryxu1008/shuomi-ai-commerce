package goods

import (
	"github.com/stretchr/testify/require"
	"task-processor/internal/marketplace/temu/model"
	"testing"
)

func TestOptionalLocalAttributeNeedsExplicitCrossBorderRequirementFlag(t *testing.T) {
	template := model.LocalAttributeTemplate{Attributes: []model.LocalAttribute{{RefPID: draftPointer(int64(12)), Required: draftPointer(false), Rules: &model.LocalAttributeRules{ControlType: draftPointer(int64(0)), ChooseMaxNum: draftPointer(int64(1))}}}}
	require.NotEmpty(t, CheckLocalProperties(nil, template), "missing transnational flag is not false")
	template.Attributes[0].Rules.Transnational = draftPointer(false)
	require.Empty(t, CheckLocalProperties(nil, template))
	template.Attributes[0].Rules.Transnational = draftPointer(true)
	require.NotEmpty(t, CheckLocalProperties(nil, template), "cross-border required attribute cannot be omitted")
	template.Attributes[0].Rules = nil
	require.NotEmpty(t, CheckLocalProperties(nil, template), "missing rule object cannot prove an optional field")
}
