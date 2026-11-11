package setup

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"task-processor/internal/app/setupcontrol"
)

// Opt-in uses only newly-created synthetic volumes, never the retained instance.
func TestRecoveryDockerReadonlyVolumeAdmission(t *testing.T) {
	if os.Getenv("LISTINGKIT_SETUP_RECOVERY_DOCKER_TEST") != "1" {
		t.Skip("explicit isolated Docker fixture required")
	}
	ctx := context.Background()
	project := "install1240-recovery-check-" + uuid.NewString()[:8]
	d := &Driver{source: t.TempDir(), runner: Docker{}}
	d.clientsIdle = d.desktopClientsIdle
	run := func(args ...string) string {
		b, e := d.run(ctx, args...)
		require.NoError(t, e)
		return strings.TrimSpace(string(b))
	}
	run("image", "inspect", "alpine:3.22")
	volumes := map[string]any{}
	names := []string{}
	for i := 0; i < 35; i++ {
		logical := fmt.Sprintf("v%d", i)
		name := project + "-" + logical
		names = append(names, name)
		volumes[logical] = map[string]any{"name": name}
	}
	foreign := "install1240-foreign-" + uuid.NewString()[:8]
	t.Cleanup(func() {
		cleanup, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		if raw, e := d.run(cleanup, "inspect", foreign); e == nil {
			var cs []container
			if json.Unmarshal(raw, &cs) == nil && len(cs) == 1 && cs[0].Config.Labels["io.shuomi.recovery-fixture"] == project {
				_, _ = d.run(cleanup, "rm", "-f", cs[0].ID)
			}
		}
		b, e := d.run(cleanup, "volume", "ls", "-q", "--filter", "label=io.shuomi.recovery-fixture="+project)
		if e != nil {
			t.Error("fixture cleanup could not enumerate its dedicated volumes")
			return
		}
		for _, name := range strings.Fields(string(b)) {
			if !strings.HasPrefix(name, project+"-") {
				t.Error("unexpected cleanup scope")
				return
			}
			if _, e = d.run(cleanup, "volume", "rm", name); e != nil {
				t.Error("dedicated fixture volume cleanup failed")
			}
		}
	})
	for logical, raw := range volumes {
		name := raw.(map[string]any)["name"].(string)
		run("volume", "create", "--label", "io.shuomi.recovery-fixture="+project, "--label", "com.docker.compose.project="+project, "--label", "com.docker.compose.volume="+logical, name)
	}
	raw := marshal(map[string]any{"name": project, "volumes": volumes, "networks": map[string]any{}})
	file := filepath.Join(d.source, "compose.json")
	require.NoError(t, os.WriteFile(file, raw, 0600))
	id, e := d.identity(ctx, project)
	require.NoError(t, e)
	p := setupcontrol.Prepared{InstanceID: id, Request: setupcontrol.Request{Mode: "fresh", Project: project}, Data: marshal(nativePlan{Mode: "fresh", Project: project, ComposeFile: file, ComposeHash: hash(raw)})}
	a, e := d.AuditRecovery(ctx, p, true)
	require.NoError(t, e)
	require.True(t, a.Empty)
	require.Empty(t, a.Blocks)
	run("create", "--pull", "never", "--network", "none", "--name", foreign, "--label", "io.shuomi.recovery-fixture="+project, "--mount", "type=volume,src="+names[0]+",dst=/source,readonly,volume-nocopy", "alpine:3.22", "true")
	a, e = d.AuditRecovery(ctx, p, true)
	require.NoError(t, e)
	require.False(t, a.Empty)
	require.NotEmpty(t, a.Blocks)
	run("rm", foreign)
	run("run", "--rm", "--pull", "never", "--network", "none", "--read-only", "--cap-drop", "ALL", "--mount", "type=volume,src="+names[0]+",dst=/data,volume-nocopy", "alpine:3.22", "sh", "-eu", "-c", `printf '%s' 'synthetic-fixture-state' > /data/.test-state`)
	fileHash := func() string {
		return run("run", "--rm", "--pull", "never", "--network", "none", "--read-only", "--cap-drop", "ALL", "--mount", "type=volume,src="+names[0]+",dst=/data,readonly,volume-nocopy", "alpine:3.22", "sha256sum", "/data/.test-state")
	}
	before := fileHash()
	a, e = d.AuditRecovery(ctx, p, true)
	require.NoError(t, e)
	require.False(t, a.Empty)
	require.NotEmpty(t, a.Blocks)
	require.Equal(t, before, fileHash())
	run("volume", "rm", names[34])
	a, e = d.AuditRecovery(ctx, p, true)
	require.NoError(t, e)
	require.False(t, a.Empty)
	remaining := strings.Fields(run("volume", "ls", "-q", "--filter", "label=io.shuomi.recovery-fixture="+project))
	require.Len(t, remaining, 34)
	t.Log("35 isolated volumes: empty admitted, unlabelled reference refused, hidden state preserved, missing volume not recreated")
}
