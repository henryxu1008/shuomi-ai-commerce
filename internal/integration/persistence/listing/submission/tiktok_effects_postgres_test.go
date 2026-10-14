package submissionpersistence

import (
	"context"
	"github.com/stretchr/testify/require"
	"task-processor/internal/listing/submission"
	"testing"
	"time"
)

func TestPostgresTikTokCompletedStagesCannotBeRecreatedWithNewIntent(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	db, _ := openExecutionPostgres(t, ctx)
	require.NoError(t, InstallSchema(db))
	plain, err := NewRepository(db)
	require.NoError(t, err)
	seed, err := submission.NewExecutionKernel(plain)
	require.NoError(t, err)
	effects, err := NewOfficialEffectsRepository(db)
	require.NoError(t, err)
	kernel, err := submission.NewExecutionKernel(effects)
	require.NoError(t, err)
	for _, stage := range []string{submission.TikTokImageStage, submission.TikTokGlobalCreateStage, submission.TikTokPublishStage} {
		command := executionCommand("org-a", "first-"+stage, "tiktok-"+stage, "{\"version\":1}")
		command.Target.Platform = "tiktokshop"
		command.Action = submission.TikTokStageAction(stage)
		acquired, err := seed.Acquire(ctx, command)
		require.NoError(t, err)
		require.NotNil(t, acquired.Permit)
		_, err = seed.Complete(ctx, permitClaim("org-a", acquired.Permit), providerEvidence(submission.ExecutionSucceeded, "receipt-"+stage, time.Now().UTC()))
		require.NoError(t, err)
		command.IntentKey = "second-" + stage
		command.Payload = []byte("{\"version\":2}")
		_, err = kernel.Acquire(ctx, command)
		require.ErrorIs(t, err, submission.ErrExecutionTargetSucceeded, "new intent cannot recreate the completed %s stage", stage)
	}
}
