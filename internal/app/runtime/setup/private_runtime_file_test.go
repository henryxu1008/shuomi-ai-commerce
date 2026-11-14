package setup

import (
	"context"
	"github.com/stretchr/testify/require"
	"os"
	"path/filepath"
	"task-processor/internal/app/setupcontrol"
	"testing"
)

type fileRunner struct {
	file   string
	change bool
	args   []string
}

func (r *fileRunner) Run(_ context.Context, _ string, args []string) ([]byte, error) {
	r.args = append([]string{}, args...)
	if r.change {
		_ = os.WriteFile(r.file, []byte("unexpected"), 0600)
	}
	return nil, nil
}
func TestRuntimeFilePermissionPreparationBindsOnlyCandidateAndRejectsByteDrift(t *testing.T) {
	file := filepath.Join(t.TempDir(), "candidate.json")
	require.NoError(t, os.WriteFile(file, []byte("private-test-value"), 0600))
	runner := &fileRunner{file: file}
	d := &Driver{runner: runner}
	require.NoError(t, d.protectRuntimeFile(context.Background(), file))
	require.Contains(t, runner.args, "none")
	require.Contains(t, runner.args, "type=bind,src="+file+",dst=/runtime-manifest")
	require.NotContains(t, runner.args, "private-test-value")
	runner.change = true
	require.ErrorIs(t, d.protectRuntimeFile(context.Background(), file), setupcontrol.ErrDrift)
}
