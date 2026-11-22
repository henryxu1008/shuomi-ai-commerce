package skillsandbox

import (
	"context"
	"time"

	osb "github.com/alibaba/OpenSandbox/sdks/sandbox/go"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/agent"
)

// CheckControl uses the same authenticated, bounded SDK/TLS transport as
// execution. This is a read-only reachability check, not a capacity reservation
// or proof that a later create/exec will succeed. Execute still checks capacity.
func (e *Executor) CheckControl(ctx context.Context) error {
	_, err := e.listControl(ctx)
	return err
}

func (e *Executor) listControl(ctx context.Context) (*osb.ListSandboxesResponse, error) {
	if e == nil || e.lifecycle == nil {
		return nil, agent.ErrUnavailable
	}
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	listed, err := e.lifecycle.ListSandboxes(ctx, osb.ListOptions{Page: 1, PageSize: 3})
	if err != nil || listed == nil {
		// Neither transport errors (including TLS/authentication) nor SDK decode
		// errors establish capacity. Never return raw endpoint/SDK errors.
		return nil, agent.ErrUnavailable
	}
	return listed, nil
}
