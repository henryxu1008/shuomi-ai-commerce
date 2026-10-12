package agentconfigpersistence

import (
	"gorm.io/gorm"
	"regexp"
	"task-processor/internal/agentconfig"
)

// InstallExecutionModes is an explicit additive update for the current owner.
// It does not touch existing facts, grants, roles, activations or snapshots.
func InstallExecutionModes(db *gorm.DB, role string) error {
	if db == nil || db.Dialector.Name() != "postgres" || !regexp.MustCompile(`^[a-z][a-z0-9_]{0,62}$`).MatchString(role) {
		return agentconfig.ErrInvalid
	}
	return db.Transaction(func(tx *gorm.DB) error {
		var allowed bool
		if err := tx.Raw(`SELECT EXISTS(SELECT 1 FROM pg_roles r JOIN pg_namespace n ON n.nspowner=r.oid WHERE r.rolname='agent_configuration_owner' AND n.nspname='agent_configuration' AND NOT(r.rolcanlogin OR r.rolsuper OR r.rolcreatedb OR r.rolcreaterole OR r.rolbypassrls))`).Scan(&allowed).Error; err != nil || !allowed {
			return agentconfig.ErrInvalid
		}
		if err := tx.Raw(`SELECT EXISTS(SELECT 1 FROM pg_roles WHERE rolname=? AND NOT(rolsuper OR rolcreatedb OR rolcreaterole OR rolreplication OR rolbypassrls) AND NOT pg_has_role(oid,'agent_configuration_owner','MEMBER'))`, role).Scan(&allowed).Error; err != nil || !allowed {
			return agentconfig.ErrInvalid
		}
		if err := tx.Raw(`SELECT has_schema_privilege(?,'agent_configuration','USAGE') AND NOT has_schema_privilege(?,'agent_configuration','CREATE')`, role, role).Scan(&allowed).Error; err != nil || !allowed {
			return agentconfig.ErrInvalid
		}
		if err := tx.Exec("SET LOCAL ROLE agent_configuration_owner").Error; err != nil {
			return err
		}
		if err := tx.Exec(executionModesSQL).Error; err != nil {
			return err
		}
		if err := tx.Exec("RESET ROLE").Error; err != nil {
			return err
		}
		quoted := `"` + role + `"`
		for _, sql := range []string{
			"GRANT SELECT, INSERT ON agent_configuration.execution_modes, agent_configuration.execution_mode_snapshots TO " + quoted,
			"GRANT UPDATE (mode, revision, updated_by, updated_at) ON agent_configuration.execution_modes TO " + quoted,
		} {
			if err := tx.Exec(sql).Error; err != nil {
				return err
			}
		}
		for _, table := range []string{"execution_modes", "execution_mode_snapshots"} {
			if err := tx.Raw(`SELECT c.relowner=(SELECT oid FROM pg_roles WHERE rolname='agent_configuration_owner') AND NOT has_table_privilege(?,c.oid,'DELETE,TRUNCATE,REFERENCES,TRIGGER') FROM pg_class c WHERE c.oid=?::regclass`, role, "agent_configuration."+table).Scan(&allowed).Error; err != nil || !allowed {
				return agentconfig.ErrInvalid
			}
		}
		var unsafe bool
		if err := tx.Raw(`SELECT has_any_column_privilege(?,'agent_configuration.execution_mode_snapshots','UPDATE') OR EXISTS(SELECT 1 FROM pg_attribute WHERE attrelid='agent_configuration.execution_modes'::regclass AND attnum>0 AND NOT attisdropped AND attname NOT IN ('mode','revision','updated_by','updated_at') AND has_column_privilege(?,attrelid,attname,'UPDATE'))`, role, role).Scan(&unsafe).Error; err != nil || unsafe {
			return agentconfig.ErrInvalid
		}
		return nil
	})
}
