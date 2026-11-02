package agentconfigpersistence

import (
	"context"

	"task-processor/internal/agent"
	"task-processor/internal/agentconfig"
)

// FindMaterialsPreparation returns an immutable original receipt, never an
// execution permit. The application must authorize its complete frozen binding
// and compare public intent before disclosing its snapshot ref.
func (s *Store) FindMaterialsPreparation(ctx context.Context, scope agent.Scope, key string) (agentconfig.Snapshot, bool, error) {
	if s == nil || s.db == nil || ctx == nil || !scopeOK(scope) || !agentconfig.UUID(key) {
		return agentconfig.Snapshot{}, false, agentconfig.ErrInvalid
	}
	var rows []snapshotRow
	err := s.db.WithContext(ctx).Where("organization_id=? AND actor_id=? AND agent_id=? AND request_key=?", scope.OrganizationID, scope.ActorID, agentconfig.MaterialsAgentID, key).Limit(2).Find(&rows).Error
	if err != nil {
		return agentconfig.Snapshot{}, false, err
	}
	if len(rows) == 0 {
		return agentconfig.Snapshot{}, false, nil
	}
	if len(rows) != 1 {
		return agentconfig.Snapshot{}, false, agentconfig.ErrConflict
	}
	v, err := snapshotView(rows[0])
	return v, err == nil, err
}
