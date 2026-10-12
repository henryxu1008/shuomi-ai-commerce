package agentconfigpersistence

import (
	"context"
	"task-processor/internal/agent"
	"task-processor/internal/agentconfig"
)

func (s *Store) CheckImageExecutionMode(ctx context.Context, identity agent.ExecutionIdentity, config agent.ConfigurationSnapshotRef, ref agent.ModeSnapshotRef) (agentconfig.ModeSnapshot, error) {
	snapshot, err := s.LoadImageConfiguration(ctx, agent.Scope{OrganizationID: identity.OrganizationID, ActorID: identity.ActorID}, config)
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
	if mode.Subject.Subject.Kind != agentconfig.RegisteredAgentSubject || mode.Subject.Subject.ID != agentconfig.ImageAgentID || mode.Subject.Definition.Version != snapshot.AgentVersion || mode.Binding.ContextKind != agentconfig.ImageSnapshotContext || mode.Binding.ContextID != snapshot.ContextID || mode.Binding.RequestKey != snapshot.RequestKey {
		return agentconfig.ModeSnapshot{}, agentconfig.ErrConflict
	}
	current, err := s.ReadAgent(ctx, snapshot.Scope, agentconfig.ImageAgentID)
	if err != nil {
		return agentconfig.ModeSnapshot{}, err
	}
	if current.Activation != "ENABLED" || current.ActivationEpoch != snapshot.Epoch {
		return agentconfig.ModeSnapshot{}, agentconfig.ErrChanged
	}
	return mode, nil
}
