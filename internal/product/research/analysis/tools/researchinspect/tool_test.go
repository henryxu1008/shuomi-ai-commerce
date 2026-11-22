package researchinspect

import (
	"context"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/agent"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/commercetool"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/product/research/analysis"
	"github.com/stretchr/testify/require"
	"testing"
)

type reader struct{}

func (reader) Inspect(context.Context, agent.Binding) (analysis.Inspection, error) {
	return analysis.Inspection{}, agent.ErrUnavailable
}
func TestResearchRegistryAcceptsClosedEvidenceAndExactDefinition(t *testing.T) {
	_, err := commercetool.NewRegistry(commercetool.Tool{Definition: Definition(), Executor: executor{reader{}}})
	require.NoError(t, err)
	require.Equal(t, "product.research.agent", AgentDefinition().ID)
}
