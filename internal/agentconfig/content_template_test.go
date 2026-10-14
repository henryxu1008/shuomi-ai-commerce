package agentconfig

import (
	"testing"

	"github.com/stretchr/testify/require"
	"task-processor/internal/product/enrichment"
)

func TestContentTemplateKeepsLanguageAndFieldsInItsOwnAgentSchema(t *testing.T) {
	selection, err := enrichment.NewContentSelection("en-US", []string{"title", "description"})
	require.NoError(t, err)
	parameters := ContentTemplate{Schema: ContentParameterSchema, Selection: selection}
	input := TemplateInput{Name: "English content", TargetPlatform: "product", Content: &parameters}
	require.True(t, input.ValidForAgent(ContentAgentID))
	require.Equal(t, ContentParameterSchema, ParameterSchemaForAgent(ContentAgentID))
	require.False(t, input.ValidForAgent("product.title.agent"))
	require.False(t, input.ValidForAgent(ImageAgentID))
	bad := input
	bad.Content = nil
	require.False(t, bad.ValidForAgent(ContentAgentID))
	parameters.Schema = ParameterSchema
	require.False(t, input.ValidForAgent(ContentAgentID))
	parameters.Schema = ContentParameterSchema
	parameters.Selection.TargetLanguage = "en-us"
	require.False(t, input.ValidForAgent(ContentAgentID))
	parameters.Selection.TargetLanguage = "en-US"
	parameters.Selection.Title, parameters.Selection.Description = false, false
	require.False(t, input.ValidForAgent(ContentAgentID))
}
