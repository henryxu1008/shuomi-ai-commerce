package setup

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"task-processor/internal/app/setupcontrol"
)

type runningImageRunner struct{ calls []string }

func (r *runningImageRunner) Run(_ context.Context, _ string, args []string) ([]byte, error) {
	call := strings.Join(args, " ")
	r.calls = append(r.calls, call)
	if strings.Contains(call, "sha256:running") {
		return nil, setupcontrol.ErrUnavailable
	}
	return []byte(`[]`), nil
}

func TestInspectionRejectsWhenRunningImageCannotVerifyCandidate(t *testing.T) {
	runner := &runningImageRunner{}
	d := &Driver{release: Release{BackendImage: "sha256:installer"}, runner: runner}
	a := container{ID: "running-container", Image: "sha256:running"}
	a.Config.Cmd = []string{"-config", "/private/config.json"}
	_, err := d.inspectModules(context.Background(), a, "candidate.json")
	require.Error(t, err)
	require.Contains(t, strings.Join(runner.calls, "\n"), "sha256:running")
}
