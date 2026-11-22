package module

import (
	"github.com/henryxu1008/shuomi-ai-commerce/internal/core/config"
)

type Module interface {
	Name() string
	Enabled(cfg *config.Config) bool
	Register(reg *Registry) error
}
