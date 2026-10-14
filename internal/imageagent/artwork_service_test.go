package imageagent_test

import (
	"context"
	"errors"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"task-processor/internal/agent"
	"task-processor/internal/agentconfig"
	"task-processor/internal/imageagent"
	"task-processor/internal/imageagent/store"
	"testing"
)

type artworkConfigurationFixture struct{ setConfigurationFixture }

func (c *artworkConfigurationFixture) PrepareArtworkConfiguration(ctx context.Context, start agentconfig.ArtworkStartCommand) (agentconfig.ArtworkConfigurationSnapshot, error) {
	v, err := c.PrepareImageConfiguration(ctx, agentconfig.ImageStartCommand(start))
	v.AgentVersion = agentconfig.ArtworkAgentVersion
	return agentconfig.ArtworkConfigurationSnapshot{ImageConfigurationSnapshot: v, ArtworkParameters: agentconfig.ArtworkTemplate{Schema: agentconfig.ArtworkParameterSchema, Style: "扁平插画"}}, err
}
func (c *artworkConfigurationFixture) LoadArtworkConfiguration(context.Context, agent.Scope, agent.ConfigurationSnapshotRef) (agentconfig.ArtworkConfigurationSnapshot, error) {
	v := c.snapshot
	v.AgentVersion = agentconfig.ArtworkAgentVersion
	return agentconfig.ArtworkConfigurationSnapshot{ImageConfigurationSnapshot: v, ArtworkParameters: agentconfig.ArtworkTemplate{Schema: agentconfig.ArtworkParameterSchema, Style: "扁平插画"}}, nil
}
func (c *artworkConfigurationFixture) AdmitArtworkRun(ctx context.Context, cmd agentconfig.ArtworkRunAdmissionCommand, limit agentconfig.ImageRunLimits) (agentconfig.ArtworkRunAdmissionReceipt, error) {
	v, err := c.AdmitImageRun(ctx, agentconfig.ImageRunAdmissionCommand(cmd), limit)
	return agentconfig.ArtworkRunAdmissionReceipt(v), err
}
func (c *artworkConfigurationFixture) ReadArtworkRunAdmission(ctx context.Context, scope agent.Scope, ref agent.ConfigurationSnapshotRef) (agentconfig.ArtworkRunAdmissionReceipt, error) {
	v, err := c.ReadImageRunAdmission(ctx, scope, ref)
	return agentconfig.ArtworkRunAdmissionReceipt(v), err
}
func (c *artworkConfigurationFixture) CheckArtworkExecutionMode(context.Context, agent.ExecutionIdentity, agent.ConfigurationSnapshotRef, agent.ModeSnapshotRef) (agentconfig.ModeSnapshot, error) {
	if !c.enabled {
		return agentconfig.ModeSnapshot{}, agentconfig.ErrChanged
	}
	return agentconfig.ModeSnapshot{Mode: agent.ModeDefault}, nil
}

func TestArtworkPreparationHasNoOriginalAndConfirmationRestoresOneAdmission(t *testing.T) {
	_, _, _, _, _, quotes, ctx, _ := imageSetServiceFixture(t)
	config := &artworkConfigurationFixture{setConfigurationFixture: setConfigurationFixture{enabled: true}}
	workflows := &recordingWorkflowClient{}
	repo := store.NewMemoryRepository()
	service, err := imageagent.NewService(repo, workflows, staticCatalogResolver{}, imageagent.WithOrganizationScope(), imageagent.WithArtworkDependencies(imageagent.ArtworkDependencies{Configuration: config, Quotes: quotes, HardLimits: agentconfig.ImageRunLimits{Images: 1, Points: 100, ElapsedSeconds: 300}}))
	require.NoError(t, err)
	input := imageagent.PrepareArtworkInput{RequestID: uuid.NewString(), ContextID: "creative", ExpectedModeRevision: "0", Theme: "原创森林动物", Requirements: "不含商品照片和文字"}
	prepared, err := service.PrepareArtwork(ctx, input)
	require.NoError(t, err)
	byOriginalKey, readErr := service.GetPreparedArtworkByKey(ctx, input.ContextID, input.RequestID)
	require.NoError(t, readErr, "every accepted canonical preparation context supports exact original-key readback")
	require.Equal(t, prepared.Projection.Run.ID, byOriginalKey.Projection.Run.ID)
	require.Empty(t, prepared.Projection.Plan.SourceAssetIDs)
	require.Empty(t, prepared.Projection.AssetCatalog.Assets)
	require.Nil(t, prepared.Projection.Plan.Set)
	require.Len(t, prepared.Projection.Plan.Slots, 1)
	require.Empty(t, workflows.starts, "preparation and quoting must not dispatch a paid workflow")
	changed := input
	changed.Theme = "另一主题"
	_, err = service.PrepareArtwork(ctx, changed)
	require.ErrorIs(t, err, imageagent.ErrRevisionConflict)
	confirm := imageagent.ConfirmArtworkInput{RunID: prepared.Projection.Run.ID, ActionID: uuid.NewString(), ExpectedRevision: 1, PlanDigest: prepared.PlanDigest, QuoteDigest: prepared.QuoteDigest}
	quotes.quote.Points++
	_, err = service.ConfirmArtwork(ctx, confirm)
	require.ErrorIs(t, err, imageagent.ErrRevisionConflict)
	quotes.quote.Points--
	config.admitACK = errors.New("admission ACK lost")
	workflows.startErr = errors.New("workflow ACK lost")
	confirmed, err := service.ConfirmArtwork(ctx, confirm)
	require.ErrorContains(t, err, "workflow ACK lost")
	require.NoError(t, imageagent.ValidateArtworkAdmission(confirmed.Run, confirmed.Plan))
	config.enabled = false
	workflows.startErr = nil
	replay, err := service.ConfirmArtwork(ctx, confirm)
	require.NoError(t, err)
	require.Equal(t, confirmed.Run.ImageAdmission, replay.Run.ImageAdmission)
	require.Equal(t, 1, config.admissions)
	require.Len(t, workflows.starts, 2)
	require.Equal(t, workflows.starts[0], workflows.starts[1])
}
