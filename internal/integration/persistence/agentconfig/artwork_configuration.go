package agentconfigpersistence

import (
	"context"
	"task-processor/internal/agent"
	"task-processor/internal/agentconfig"
)

func (s *Store) PrepareArtworkConfiguration(ctx context.Context, c agentconfig.ArtworkStartCommand) (agentconfig.ArtworkConfigurationSnapshot, error) {
	v, err := s.prepareVisualConfiguration(ctx, agentconfig.ImageStartCommand(c), artworkProfile)
	return agentconfig.ArtworkConfigurationSnapshot{ImageConfigurationSnapshot: v.ImageConfigurationSnapshot, ArtworkParameters: v.ArtworkParameters}, err
}
func (s *Store) LoadArtworkConfiguration(ctx context.Context, scope agent.Scope, ref agent.ConfigurationSnapshotRef) (agentconfig.ArtworkConfigurationSnapshot, error) {
	v, err := s.loadVisualConfiguration(ctx, scope, ref, artworkProfile)
	return agentconfig.ArtworkConfigurationSnapshot{ImageConfigurationSnapshot: v.ImageConfigurationSnapshot, ArtworkParameters: v.ArtworkParameters}, err
}
func (s *Store) AdmitArtworkRun(ctx context.Context, c agentconfig.ArtworkRunAdmissionCommand, limits agentconfig.ImageRunLimits) (agentconfig.ArtworkRunAdmissionReceipt, error) {
	v, err := s.admitVisualRun(ctx, agentconfig.ImageRunAdmissionCommand(c), limits, artworkProfile)
	return agentconfig.ArtworkRunAdmissionReceipt(v), err
}
func (s *Store) ReadArtworkRunAdmission(ctx context.Context, scope agent.Scope, ref agent.ConfigurationSnapshotRef) (agentconfig.ArtworkRunAdmissionReceipt, error) {
	v, err := s.readVisualAdmission(ctx, scope, ref, artworkProfile)
	return agentconfig.ArtworkRunAdmissionReceipt(v), err
}
func (s *Store) CheckArtworkExecutionMode(ctx context.Context, identity agent.ExecutionIdentity, config agent.ConfigurationSnapshotRef, ref agent.ModeSnapshotRef) (agentconfig.ModeSnapshot, error) {
	return s.checkVisualExecutionMode(ctx, identity, config, ref, artworkProfile)
}

var _ agentconfig.ArtworkConfigurationRepository = (*Store)(nil)
