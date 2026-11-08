package routeconfigstore

import configstore "task-processor/internal/integration/persistence/agentconfig"

// RuntimeReadTableGrants is the original route reader's authority, shared by
// explicit installation and read-only qualification. It grants no new rights.
func RuntimeReadTableGrants() []configstore.RuntimeTableGrant {
	return []configstore.RuntimeTableGrant{
		{Relation: "public.ai_route_config_revisions", Select: true},
		{Relation: "public.ai_route_config_current", Select: true},
		{Relation: "public.ai_route_config_options", Select: true},
		{Relation: "public.ai_channel_credential_bindings", Select: true},
	}
}
