package agentconfigpersistence

import (
	"context"
	"task-processor/internal/agent"
	"task-processor/internal/agentconfig"
)

func (s *Store) CheckImageExecutionMode(ctx context.Context, identity agent.ExecutionIdentity, config agent.ConfigurationSnapshotRef, ref agent.ModeSnapshotRef) (agentconfig.ModeSnapshot, error) {
	return s.checkVisualExecutionMode(ctx, identity, config, ref, productImageProfile)
}
func (s *Store) checkVisualExecutionMode(ctx context.Context, identity agent.ExecutionIdentity, config agent.ConfigurationSnapshotRef, ref agent.ModeSnapshotRef, profile visualConfigurationProfile) (agentconfig.ModeSnapshot, error) {
	snapshot, err := s.loadVisualConfiguration(ctx, agent.Scope{OrganizationID: identity.OrganizationID, ActorID: identity.ActorID}, config, profile)
	if err != nil {
		return agentconfig.ModeSnapshot{}, err
	}
	if snapshot.MemberID != identity.MemberID || snapshot.ModeSnapshotRef != ref || !ref.Valid() {
		return agentconfig.ModeSnapshot{}, agentconfig.ErrConflict
	}
	mode, err := s.CheckModeSnapshot(ctx, identity, ref)
	if err != nil {
		return agentconfig.ModeSnapshot{}, err
	}
	if mode.Subject.Subject.Kind != agentconfig.RegisteredAgentSubject || mode.Subject.Subject.ID != profile.agentID || mode.Subject.Definition.Version != snapshot.AgentVersion || mode.Binding.ContextKind != profile.context || mode.Binding.ContextID != snapshot.ContextID || mode.Binding.RequestKey != snapshot.RequestKey {
		return agentconfig.ModeSnapshot{}, agentconfig.ErrConflict
	}
	current, err := s.ReadAgent(ctx, snapshot.Scope, profile.agentID)
	if err != nil {
		return agentconfig.ModeSnapshot{}, err
	}
	if current.Activation != "ENABLED" || current.ActivationEpoch != snapshot.Epoch {
		return agentconfig.ModeSnapshot{}, agentconfig.ErrChanged
	}
	return mode, nil
}
