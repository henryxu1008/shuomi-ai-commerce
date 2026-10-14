package agentconfig

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestArtworkConfigurationIsExactAndDoesNotBorrowImageTemplates(t *testing.T) {
	input := TemplateInput{Name: "原创图案", TargetPlatform: "sds", Artwork: &ArtworkTemplate{Schema: ArtworkParameterSchema, Style: "简洁手绘", Constraints: "独立印花，避免商标与文字"}}
	require.True(t, input.ValidForAgent(ArtworkAgentID))
	require.Equal(t, ArtworkParameterSchema, ParameterSchemaForAgent(ArtworkAgentID))
	for _, other := range []string{ImageAgentID, ContentAgentID, MaterialsAgentID, "product.title.agent"} {
		require.False(t, input.ValidForAgent(other))
	}
	input.Image = &SetTemplate{}
	require.False(t, input.ValidForAgent(ArtworkAgentID), "a creative template cannot carry a parallel product-image configuration")
	input.Image = nil
	input.DefaultKnowledgeBaseID = "00000000-0000-4000-8000-000000000001"
	require.False(t, input.ValidForAgent(ArtworkAgentID))
	input.DefaultKnowledgeBaseID = ""
	input.TargetPlatform = "shein"
	require.False(t, input.ValidForAgent(ArtworkAgentID))
}
