package einoruntime

import (
	"context"
	"testing"

	"github.com/henryxu1008/shuomi-ai-commerce/internal/agent"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/commercetool"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/product/catalog/tools/canonicalinspect"
	"github.com/stretchr/testify/require"
)

func TestTitlePilotSingleCallUsesTitleValidator(t *testing.T) {
	runtime, request, model, tools, validator, _, _ := fixture(t, proposal("supported title"))
	config := runtime.config
	config.SingleCallReadOnly = true
	config.PreReadTools = []commercetool.ToolRef{canonicalinspect.Definition().Ref}
	runtime, err := New(config)
	require.NoError(t, err)
	request.Limits.ModelCalls = 1
	result, err := runtime.Start(context.Background(), request)
	require.NoError(t, err)
	require.Equal(t, agent.HumanReviewRequired, result.State.Phase)
	require.Equal(t, 1, model.calls)
	require.Equal(t, 1, tools.calls)
	require.Equal(t, 1, validator.calls)
	require.Equal(t, canonicalinspect.Definition().Ref, model.lastInput.History[0].Tool)
}
