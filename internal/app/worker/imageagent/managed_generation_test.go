package imageagentworker

import (
	"context"
	"errors"
	"github.com/stretchr/testify/require"
	"task-processor/internal/core/config"
	"task-processor/internal/imageagent"
	"task-processor/internal/integration/openai"
	"testing"
	"time"
)

func TestSharedImageFactoryRejectsDriftAndPerPageOverrideWithoutFallback(t *testing.T) {
	resolver := &generationConfigTestResolver{resolved: &openai.ResolvedClientConfig{CacheKey: "synthetic-credential", Config: &openai.ClientConfig{APIKey: "synthetic", APIStyle: "dashscope", Model: "qwen-image-3.0", BaseURL: "https://workspace.cn-beijing.maas.aliyuncs.com", Timeout: time.Second}}}
	current := syntheticFactoryCurrent(resolver, 17)
	f := generationProviderFactory{resolver: resolver, current: current, price: config.ImageAgentGenerationConfig{Enabled: true, UnitVersion: "ai-micropoint-v1", MaximumMicropoints: 17}}
	ctx := syntheticGenerationContext()
	_, first, err := f.resolve(ctx)
	require.NoError(t, err)
	require.EqualValues(t, 17, first.Points)
	require.Equal(t, current.q.Image.Billing, first.Billing)
	current.digest = "replaced"
	require.ErrorIs(t, f.revalidate(ctx, imageagent.SlotExecutionInput{}, first), imageagent.ErrRevisionConflict)
	current.err = errors.New("unavailable")
	_, _, err = f.resolve(ctx)
	require.Error(t, err)
	current.err = nil
	resolver.resolved.CacheKey = "rotated"
	_, _, err = f.resolve(ctx)
	require.Error(t, err)
	_, _, err = f.resolve(ctx, &imageagent.ImageRouteSelection{BindingID: "other", RevisionID: "old"})
	require.ErrorIs(t, err, imageagent.ErrImageRouteChanged)
	f.current = nil
	_, _, err = f.resolve(ctx)
	require.Error(t, err, "the admission ceiling cannot substitute a shared current")
	_, _, err = f.resolve(context.Background())
	require.Error(t, err)
}
