package agentconfigpersistence

import (
	"slices"
	"strings"

	"github.com/jackc/pgx/v5"
)

// RuntimeTableGrant is the current owner's SQL grant definition, also exposed
// read-only to native preparation. It is not an installed privilege registry.
type RuntimeTableGrant struct {
	Relation               string
	Select, Insert, Update bool
	UpdateColumns          []string
}

func (g RuntimeTableGrant) Allows(privilege, column string) bool {
	switch privilege {
	case "SELECT":
		return g.Select
	case "INSERT":
		return g.Insert
	case "UPDATE":
		return g.Update || column != "" && slices.Contains(g.UpdateColumns, column)
	}
	return false
}

// SQL is used only by the existing explicit owner initializers. Qualifiers use
// Allows and never execute these GRANT statements.
func (g RuntimeTableGrant) SQL(role string) string {
	var privileges []string
	if g.Select {
		privileges = append(privileges, "SELECT")
	}
	if g.Insert {
		privileges = append(privileges, "INSERT")
	}
	if g.Update {
		privileges = append(privileges, "UPDATE")
	} else if len(g.UpdateColumns) > 0 {
		var columns []string
		for _, column := range g.UpdateColumns {
			columns = append(columns, pgx.Identifier{column}.Sanitize())
		}
		privileges = append(privileges, "UPDATE("+strings.Join(columns, ",")+")")
	}
	return "GRANT " + strings.Join(privileges, ",") + " ON " + pgx.Identifier(strings.Split(g.Relation, ".")).Sanitize() + " TO " + pgx.Identifier{role}.Sanitize()
}

func ExecutionModeRuntimeTableGrants() []RuntimeTableGrant {
	return []RuntimeTableGrant{
		{Relation: "agent_configuration.execution_modes", Select: true, Insert: true, UpdateColumns: []string{"mode", "revision", "updated_by", "updated_at"}},
		{Relation: "agent_configuration.execution_mode_snapshots", Select: true, Insert: true},
	}
}

func RuntimeTableGrants() []RuntimeTableGrant {
	return append([]RuntimeTableGrant{
		{Relation: "agent_configuration.organization_agents", Select: true, Insert: true, Update: true},
		{Relation: "agent_configuration.templates", Select: true, Insert: true, Update: true},
		{Relation: "agent_configuration.template_revisions", Select: true, Insert: true},
		{Relation: "agent_configuration.start_snapshots", Select: true, Insert: true},
		{Relation: "agent_configuration.commands", Select: true, Insert: true, UpdateColumns: []string{"before_revision", "after_revision", "receipt", "template_id"}},
		{Relation: "agent_configuration.image_run_admissions", Select: true, Insert: true},
	}, ExecutionModeRuntimeTableGrants()...)
}

func SkillAdmissionRuntimeTableGrant() RuntimeTableGrant {
	return RuntimeTableGrant{Relation: "agent_configuration.skill_admissions", Select: true, Insert: true, UpdateColumns: []string{"allow_personal", "revision", "updated_by", "updated_at"}}
}
