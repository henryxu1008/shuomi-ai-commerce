package skillpersistence

import configstore "task-processor/internal/integration/persistence/agentconfig"

// RuntimeTableGrants is consumed by this owner's initializer and read-only
// qualification. Keep the Skill bounds with the Skill owner.
func RuntimeTableGrants() []configstore.RuntimeTableGrant {
	return []configstore.RuntimeTableGrant{
		{Relation: "skill_capability.skills", Select: true, Insert: true, UpdateColumns: []string{"head_version", "revision", "updated_at"}},
		{Relation: "skill_capability.versions", Select: true, Insert: true, UpdateColumns: []string{"state", "visibility"}},
		{Relation: "skill_capability.commands", Select: true, Insert: true},
		{Relation: "skill_capability.share_requests", Select: true, Insert: true, UpdateColumns: []string{"state", "result_skill_id", "updated_at"}},
	}
}
