package store

import (
	"context"
	"strings"

	"testing"

	"github.com/stretchr/testify/require"
	credential "task-processor/internal/aicapability/channelcredential"
	"task-processor/internal/imageagent"
	resourceadapter "task-processor/internal/integration/orgresource"
)

func TestChannelRebindingCannotReplaceOriginalUnknownOrRecovery(t *testing.T) {
	owner, commercial, limits, intent := generationPointDatabases(t, true)
	binding := (credential.Metadata{OrganizationID: intent.Identity.TenantID, ConnectionID: "shared-openai", Provider: "openai", Channel: "openai-direct", Account: "fixture", Region: "global", BaseURL: "https://api.openai.com/v1", Revision: 1}).WithIdentity()
	intent.CredentialReference = binding.Reference
	_, err := owner.PrepareGenerationIntent(context.Background(), intent)
	require.NoError(t, err)

	resources, err := resourceadapter.NewGormImageGenerationRepository(commercial, resourceadapter.TransactionConfig{}, owner, generationPointTestAuth{})
	require.NoError(t, err)
	ctx := context.Background()
	receipt, err := resources.ReserveImageGeneration(ctx, intent.Identity)
	require.NoError(t, err)
	_, err = owner.BindGenerationReservation(ctx, intent, receipt)
	require.NoError(t, err)
	_, won, err := owner.BeginGenerationDispatch(ctx, intent)
	require.NoError(t, err)
	require.True(t, won)
	changed := intent
	binding.Revision++
	binding = binding.WithIdentity()
	changed.CredentialReference = binding.Reference
	_, err = owner.PrepareGenerationIntent(ctx, changed)
	require.Error(t, err, "a new connection version cannot rewrite the original durable effect")
	recovery, err := imageagent.NewGenerationRecovery(owner, resources)
	require.NoError(t, err)
	state, err := recovery.ReconcileExistingGeneration(ctx, intent.Identity)
	require.NoError(t, err)
	require.Equal(t, imageagent.GenerationUnknown, state)
	read, err := limits.ReadMonthlyLimit(ctx, intent.Identity.TenantID, intent.MemberID)
	require.NoError(t, err)
	require.EqualValues(t, intent.Points, read.Reserved)
	_, err = owner.RecordGenerationSuccess(ctx, intent, imageagent.GenerationSuccess{BillableUsage: syntheticImageUsage(100, 100), UsageKnown: true, InputTokens: 100, OutputTokens: 100, TotalTokens: 200, ResponseID: "original-response", ResultDigest: strings.Repeat("c", 64), ResultUnavailable: "invalid_result"})
	require.NoError(t, err)
	for range 2 {
		state, err = recovery.ReconcileExistingGeneration(ctx, intent.Identity)
		require.NoError(t, err)
		require.Equal(t, imageagent.GenerationSucceeded, state)
	}
	fact, err := owner.ReadGenerationFact(ctx, intent.Identity)
	require.NoError(t, err)
	require.Equal(t, intent.CredentialReference, fact.Intent.CredentialReference)
	read, err = limits.ReadMonthlyLimit(ctx, intent.Identity.TenantID, intent.MemberID)
	require.NoError(t, err)
	require.Zero(t, read.Reserved)
	require.EqualValues(t, intent.Points, read.Consumed)
}
