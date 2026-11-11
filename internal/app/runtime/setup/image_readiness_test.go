package setup

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"task-processor/internal/app/setupcontrol"
)

type imageReadinessRunner struct {
	images map[string][]byte
	calls  []string
}

func (r *imageReadinessRunner) Run(_ context.Context, _ string, args []string) ([]byte, error) {
	r.calls = append(r.calls, strings.Join(args, " "))
	if len(args) == 3 && args[0] == "image" && args[1] == "inspect" {
		if b, ok := r.images[args[2]]; ok {
			return b, nil
		}
		return nil, setupcontrol.ErrUnavailable
	}
	if len(args) > 0 && args[0] == "ps" {
		return nil, nil
	}
	return nil, setupcontrol.ErrInvalid
}

func TestImageReadinessNamesEachUnavailableComponentWithoutRelaxingAdmission(t *testing.T) {
	const version = "approved-version"
	image := func(id, revision string) []byte {
		return marshal([]any{map[string]any{"Id": id, "Config": map[string]any{"Labels": map[string]string{"org.opencontainers.image.revision": revision}}}})
	}
	for _, tc := range []struct {
		name    string
		images  map[string][]byte
		missing []string
	}{
		{"both absent", nil, []string{"应用服务安装文件", "网页界面安装文件"}},
		{"only UI absent", map[string][]byte{"backend": image("backend", version)}, []string{"网页界面安装文件"}},
		{"wrong revision", map[string][]byte{"backend": image("backend", version), "ui": image("ui", "another-version")}, []string{"网页界面安装文件"}},
		{"wrong identity", map[string][]byte{"backend": image("backend", version), "ui": image("another-image", version)}, []string{"网页界面安装文件"}},
		{"ready", map[string][]byte{"backend": image("backend", version), "ui": image("ui", version)}, nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			runner := &imageReadinessRunner{images: tc.images}
			driver := &Driver{source: t.TempDir(), release: Release{SourceCommit: version, BackendImage: "backend", UIImage: "ui"}, runner: runner}
			snapshot, err := driver.Snapshot(context.Background())
			require.NoError(t, err)
			require.True(t, snapshot.DockerReady)
			require.Len(t, snapshot.Blocks, len(tc.missing))
			for i, name := range tc.missing {
				require.Contains(t, snapshot.Blocks[i], name)
				require.Contains(t, snapshot.Blocks[i], "重新检查")
			}
			if len(tc.missing) > 0 {
				require.ErrorIs(t, driver.images(context.Background()), setupcontrol.ErrUnavailable)
			} else {
				require.NoError(t, driver.images(context.Background()))
			}
			for _, call := range runner.calls {
				require.True(t, strings.HasPrefix(call, "image inspect ") || strings.HasPrefix(call, "ps "), call)
			}
		})
	}
}
