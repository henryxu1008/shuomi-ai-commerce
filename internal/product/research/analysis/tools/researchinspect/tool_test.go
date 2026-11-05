package researchinspect

import (
	"context"
	"github.com/stretchr/testify/require"
	"task-processor/internal/agent"
	"task-processor/internal/commercetool"
	"task-processor/internal/product/research/analysis"
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
