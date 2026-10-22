package module

import (
	"task-processor/internal/httproute"
)

type Registry struct {
	routes []httproute.Descriptor
}

func NewRegistry() *Registry {
	return &Registry{}
}

func (r *Registry) AddRoutes(routes ...httproute.Descriptor) {
	r.routes = append(r.routes, routes...)
}

func (r *Registry) Routes() []httproute.Descriptor {
	out := make([]httproute.Descriptor, len(r.routes))
	copy(out, r.routes)
	return out
}
