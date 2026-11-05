package imageagent_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"task-processor/internal/agentconfig"
	"task-processor/internal/authidentity"
	"task-processor/internal/imageagent"
	"task-processor/internal/imageagent/store"
)

func TestRunReaderKeepsOriginalImageKeyAndIdentityWithoutExecution(t *testing.T) {
	s, repo, workflows, configuration, source, quotes, ctx, input := imageSetServiceFixture(t)
	input.ContextID = uuid.NewString()
	source.preparation.Source.OperationID = input.ContextID
	prepared, err := s.PrepareImageSet(ctx, input)
	require.NoError(t, err)
	configuration.enabled = false
	quotes.err = imageagent.ErrBudgetQuoteUnavailable
	reader, err := imageagent.NewRunReader(repo)
	require.NoError(t, err)
	got, err := reader.GetPreparedImageSet(ctx, input.ContextKind, input.ContextID, input.RequestID)
	require.NoError(t, err)
	require.Equal(t, prepared.Projection, got)
	got, err = reader.Get(ctx, got.Run.ID)
	require.NoError(t, err)
	require.Equal(t, prepared.Projection, got)
	for _, id := range []authidentity.AuthenticatedIdentity{
		{TenantID: "other", EffectiveOrganizationID: "other", UserID: "actor", EffectiveMemberID: "member"},
		{TenantID: "org", EffectiveOrganizationID: "org", UserID: "other", EffectiveMemberID: "member"},
		{TenantID: "org", EffectiveOrganizationID: "org", UserID: "actor", EffectiveMemberID: "other"},
		{TenantID: "org", EffectiveOrganizationID: "other", UserID: "actor", EffectiveMemberID: "member"},
	} {
		_, err = reader.Get(authidentity.WithAuthenticatedIdentity(context.Background(), id), got.Run.ID)
		require.Error(t, err)
	}
	_, err = reader.GetPreparedImageSet(ctx, input.ContextKind, uuid.NewString(), input.RequestID)
	require.Error(t, err)
	require.Empty(t, workflows.starts)
	require.Zero(t, configuration.admissions)
}

func TestRunReaderKeepsOriginalArtworkKeyWithoutExecutionDependencies(t *testing.T) {
	_, _, _, _, _, quotes, ctx, _ := imageSetServiceFixture(t)
	configuration := &artworkConfigurationFixture{setConfigurationFixture: setConfigurationFixture{enabled: true}}
	workflows, repo := &recordingWorkflowClient{}, store.NewMemoryRepository()
	s, err := imageagent.NewService(repo, workflows, staticCatalogResolver{}, imageagent.WithOrganizationScope(), imageagent.WithArtworkDependencies(imageagent.ArtworkDependencies{Configuration: configuration, Quotes: quotes, HardLimits: agentconfig.ImageRunLimits{UnitVersion: "ai-micropoint-v1", Images: 1, Points: 100, ElapsedSeconds: 300}}))
	require.NoError(t, err)
	input := imageagent.PrepareArtworkInput{RequestID: uuid.NewString(), ContextID: "creative", ExpectedModeRevision: "0", Theme: "原创森林动物", Requirements: "不含商品照片和文字"}
	prepared, err := s.PrepareArtwork(ctx, input)
	require.NoError(t, err)
	configuration.enabled = false
	reader, err := imageagent.NewRunReader(repo)
	require.NoError(t, err)
	got, err := reader.GetPreparedArtworkByKey(ctx, input.ContextID, input.RequestID)
	require.NoError(t, err)
	require.Equal(t, prepared, got)
	got, err = reader.GetPreparedArtwork(ctx, got.Projection.Run.ID)
	require.NoError(t, err)
	require.Equal(t, prepared, got)
	_, err = reader.GetPreparedArtworkByKey(ctx, "another-context", input.RequestID)
	require.Error(t, err)
	require.Empty(t, workflows.starts)
	require.Zero(t, configuration.admissions)
}
