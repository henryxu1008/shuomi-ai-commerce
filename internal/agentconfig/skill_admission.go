package agentconfig

import (
	"context"
	"task-processor/internal/agent"
)

type SkillAdmission struct {
	AllowPersonal bool   `json:"allowPersonal"`
	Revision      string `json:"revision"`
}
type SkillAdmissionCommand struct {
	Scope         agent.Scope
	Key           string
	Expected      uint64
	AllowPersonal bool
}
type SkillAdmissionRepository interface {
	ReadSkillAdmission(context.Context, agent.Scope) (SkillAdmission, error)
	SetSkillAdmission(context.Context, SkillAdmissionCommand, func(context.Context) error) (SkillAdmission, error)
}
