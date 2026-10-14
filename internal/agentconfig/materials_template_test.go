package agentconfig

import (
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestMaterialsTemplateUsesExactRegistrationWithoutTitleConfiguration(t *testing.T) {
	require.Equal(t, MaterialsParameterSchema, ParameterSchemaForAgent(MaterialsAgentID))
	require.True(t, (TemplateInput{Name: "平台资料", TargetPlatform: "shein"}).ValidForAgent(MaterialsAgentID))
	for _, in := range []TemplateInput{
		{Name: "other platform", TargetPlatform: "temu"},
		{Name: "canonical content", TargetPlatform: "product"},
		{Name: "knowledge is not platform evidence", TargetPlatform: "shein", DefaultKnowledgeBaseID: uuid.NewString()},
	} {
		require.False(t, in.ValidForAgent(MaterialsAgentID))
	}
	require.Equal(t, ParameterSchema, ParameterSchemaForAgent("product.title.agent"))
	require.True(t, (TemplateInput{Name: "old exact title", TargetPlatform: "temu", DefaultKnowledgeBaseID: uuid.NewString()}).ValidForAgent("product.title.agent"))
}
