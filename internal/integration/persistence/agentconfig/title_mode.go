package agentconfigpersistence

import (
	"context"
	"task-processor/internal/agent"
	"task-processor/internal/agentconfig"
)

// CheckRegisteredTextExecutionMode binds execution and automatic adoption to
// the original configuration and the live activation epoch.
func (s *Store) CheckRegisteredTextExecutionMode(ctx context.Context, identity agent.ExecutionIdentity, config agent.ConfigurationSnapshotRef, ref agent.ModeSnapshotRef) (agentconfig.ModeSnapshot, error) {
	snapshot, mode, err := s.checkRegisteredTextMode(ctx, identity, config, ref)
	if err != nil {
		return agentconfig.ModeSnapshot{}, err
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

// CheckCompletedTitleScriptReviewMode checks the original binding and the
// current mode for a completed own-product script result. Its caller must
// establish the completed Run and current Review/Skill authorization.
func (s *Store) CheckCompletedTitleScriptReviewMode(ctx context.Context, identity agent.ExecutionIdentity, config agent.ConfigurationSnapshotRef, ref agent.ModeSnapshotRef) (agentconfig.ModeSnapshot, error) {
	snapshot, mode, err := s.checkRegisteredTextMode(ctx, identity, config, ref)
	if err != nil {
		return agentconfig.ModeSnapshot{}, err
	}
	request := snapshot.Request
	if snapshot.AgentID != agentconfig.ContentAgentID || request.Binding.ContextKind != "own_product" || request.Binding.TargetPlatform != "product" || !request.ScriptSelected() || !request.ValidScript() {
		return agentconfig.ModeSnapshot{}, agentconfig.ErrConflict
	}
	if mode.Mode != agent.ModeDefault {
		return agentconfig.ModeSnapshot{}, agentconfig.ErrForbidden
	}
	return mode, nil
}

func (s *Store) checkRegisteredTextMode(ctx context.Context, identity agent.ExecutionIdentity, config agent.ConfigurationSnapshotRef, ref agent.ModeSnapshotRef) (agentconfig.Snapshot, agentconfig.ModeSnapshot, error) {
	snapshot, err := s.LoadSnapshot(ctx, agent.Scope{OrganizationID: identity.OrganizationID, ActorID: identity.ActorID}, config)
	if err != nil {
		return agentconfig.Snapshot{}, agentconfig.ModeSnapshot{}, err
	}
	if snapshot.MemberID != identity.MemberID || snapshot.Request.ModeSnapshotRef != ref {
		return agentconfig.Snapshot{}, agentconfig.ModeSnapshot{}, agentconfig.ErrConflict
	}
	if snapshot.AgentID == agentconfig.MaterialsAgentID && !validMaterialsTextConfiguration(snapshot.AgentID, snapshot.AgentVersion, snapshot.KnowledgeBaseID, snapshot.ExecutionModelProfile) {
		return agentconfig.Snapshot{}, agentconfig.ModeSnapshot{}, agentconfig.ErrConflict
	}
	mode, err := s.CheckModeSnapshot(ctx, identity, ref)
	if err != nil {
		return agentconfig.Snapshot{}, agentconfig.ModeSnapshot{}, err
	}
	if mode.Subject.Subject.Kind != agentconfig.RegisteredAgentSubject || mode.Subject.Subject.ID != snapshot.AgentID || mode.Subject.Definition.Version != snapshot.AgentVersion || mode.Binding.ContextKind != snapshot.Request.Binding.ContextKind || mode.Binding.ContextID != snapshot.Request.Binding.ContextID || mode.Binding.RequestKey != snapshot.Request.Key {
		return agentconfig.Snapshot{}, agentconfig.ModeSnapshot{}, agentconfig.ErrConflict
	}
	return snapshot, mode, nil
}
