package module

import (
	"task-processor/internal/core/config"
)

type Module interface {
	Name() string
	Enabled(cfg *config.Config) bool
	Register(reg *Registry) error
}
