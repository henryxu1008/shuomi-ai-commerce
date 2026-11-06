package agentconfigpersistence

import (
	"context"
	"errors"
	"gorm.io/gorm"
	"task-processor/internal/agent"
	"task-processor/internal/agentconfig"
)

// LoadResearchPreparationByKey retrieves the original owner receipt without
// current model routing or evidence hydration. The consumer must authorize all
// original sources before disclosing any binding or snapshot reference.
func (s *Store) LoadResearchPreparationByKey(ctx context.Context, scope agent.Scope, member, project, key string) (agentconfig.Snapshot, bool, error) {
	if !scopeOK(scope) || !agent.ValidID(member) || !agentconfig.UUID(project) || !agentconfig.UUID(key) {
		return agentconfig.Snapshot{}, false, agentconfig.ErrInvalid
	}
	var row snapshotRow
	err := s.db.WithContext(ctx).Where("organization_id=? AND actor_id=? AND agent_id=? AND context_kind=? AND context_id=? AND request_key=?", scope.OrganizationID, scope.ActorID, agentconfig.ResearchAgentID, agent.ResearchSelectionContextKind, project, key).Take(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return agentconfig.Snapshot{}, false, nil
	}
	if err != nil {
		return agentconfig.Snapshot{}, true, err
	}
	snap, err := snapshotView(row)
	if err != nil {
		return agentconfig.Snapshot{}, true, err
	}
	if snap.MemberID != member {
		return agentconfig.Snapshot{}, true, agentconfig.ErrForbidden
	}
	return snap, true, nil
}
