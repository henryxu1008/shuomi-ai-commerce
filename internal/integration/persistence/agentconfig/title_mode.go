package agentconfigpersistence

import (
	"context"
	"task-processor/internal/agent"
	"task-processor/internal/agentconfig"
)

// Every new model dispatch or local adoption checks this exact original
// configuration. History and committed receipts use their original read ports.
func (s *Store) CheckRegisteredTextExecutionMode(ctx context.Context, identity agent.ExecutionIdentity, config agent.ConfigurationSnapshotRef, ref agent.ModeSnapshotRef) (agentconfig.ModeSnapshot, error) {
	snapshot, err := s.LoadSnapshot(ctx, agent.Scope{OrganizationID: identity.OrganizationID, ActorID: identity.ActorID}, config)
	if err != nil {
		return agentconfig.ModeSnapshot{}, err
	}
	if snapshot.MemberID != identity.MemberID || snapshot.Request.ModeSnapshotRef != ref {
		return agentconfig.ModeSnapshot{}, agentconfig.ErrConflict
	}
	mode, err := s.CheckModeSnapshot(ctx, identity, ref)
	if err != nil {
		return agentconfig.ModeSnapshot{}, err
	}
	if mode.Subject.Subject.Kind != agentconfig.RegisteredAgentSubject || mode.Subject.Subject.ID != snapshot.AgentID || mode.Subject.Definition.Version != snapshot.AgentVersion || mode.Binding.ContextKind != snapshot.Request.Binding.ContextKind || mode.Binding.ContextID != snapshot.Request.Binding.ContextID || mode.Binding.RequestKey != snapshot.Request.Key {
		return agentconfig.ModeSnapshot{}, agentconfig.ErrConflict
	}
	current, err := s.ReadAgent(ctx, snapshot.Scope, snapshot.AgentID)
	if err != nil {
		return agentconfig.ModeSnapshot{}, err
	}
	if current.Activation != "ENABLED" || current.ActivationEpoch != snapshot.Epoch {
		return agentconfig.ModeSnapshot{}, agentconfig.ErrChanged
	}
	return mode, nil
}
