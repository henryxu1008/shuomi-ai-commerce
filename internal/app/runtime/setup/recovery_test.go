package setup

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"task-processor/internal/app/setupcontrol"
)

type auditRunner struct {
	foreign, nonempty, missingSource bool
	calls                            []string
}

func (r *auditRunner) Run(_ context.Context, _ string, args []string) ([]byte, error) {
	r.calls = append(r.calls, strings.Join(args, " "))
	if len(args) > 1 && args[0] == "info" {
		if args[len(args)-1] == "{{.DockerRootDir}}" {
			return []byte("/var/lib/docker"), nil
		}
		return []byte("daemon"), nil
	}
	if args[0] == "ps" {
		if r.foreign {
			return []byte("unlabelled"), nil
		}
		return nil, nil
	}
	if args[0] == "inspect" {
		return []byte(`[{"Id":"unlabelled","Name":"/other-work","Config":{"Labels":{}},"Mounts":[{"Type":"volume","Name":"test-install-v0"}]}]`), nil
	}
	if args[0] == "volume" && args[1] == "ls" {
		names := []string{}
		for i := 0; i < 35; i++ {
			names = append(names, fmt.Sprintf("test-install-v%d", i))
		}
		return []byte(strings.Join(names, "\n")), nil
	}
	if args[0] == "volume" && args[1] == "inspect" {
		rows := []any{}
		for i := 0; i < 35; i++ {
			rows = append(rows, map[string]any{"Name": fmt.Sprintf("test-install-v%d", i), "Driver": "local", "Mountpoint": fmt.Sprintf("/var/lib/docker/volumes/test-install-v%d/_data", i), "CreatedAt": "2026-11-11T12:06:33Z", "Labels": map[string]string{"com.docker.compose.project": "test-install", "com.docker.compose.volume": fmt.Sprintf("v%d", i)}})
		}
		return marshal(rows), nil
	}
	if args[0] == "network" {
		return nil, nil
	}
	if args[0] == "image" {
		return []byte(`[{"Id":"sha256:helper"}]`), nil
	}
	if args[0] == "run" {
		if r.missingSource {
			return nil, setupcontrol.ErrUnavailable
		}
		if r.nonempty {
			return []byte("NONEMPTY\n"), nil
		}
		return []byte("EMPTY\n"), nil
	}
	return nil, setupcontrol.ErrUnavailable
}
func auditFixture(t *testing.T, r *auditRunner) (*Driver, setupcontrol.Prepared) {
	t.Helper()
	root := t.TempDir()
	volumes := map[string]any{}
	for i := 0; i < 35; i++ {
		volumes[fmt.Sprintf("v%d", i)] = map[string]any{"name": fmt.Sprintf("test-install-v%d", i)}
	}
	raw := marshal(map[string]any{"name": "test-install", "volumes": volumes, "networks": map[string]any{"default": map[string]any{"name": "test-install_default"}}})
	file := filepath.Join(root, "compose.json")
	require.NoError(t, os.WriteFile(file, raw, 0600))
	d := &Driver{source: root, runner: r}
	d.clientsIdle = func(context.Context) (bool, error) { return true, nil }
	id, e := d.identity(context.Background(), "test-install")
	require.NoError(t, e)
	p := setupcontrol.Prepared{InstanceID: id, Request: setupcontrol.Request{Mode: "fresh", Project: "test-install"}, Data: marshal(nativePlan{Mode: "fresh", Project: "test-install", ComposeFile: file, ComposeHash: hash(raw)})}
	return d, p
}
func TestRecoveryAuditSeesUnlabelledReferencesAndNeverReadsTheirVolumes(t *testing.T) {
	r := &auditRunner{foreign: true}
	d, p := auditFixture(t, r)
	a, e := d.AuditRecovery(context.Background(), p, true)
	require.NoError(t, e)
	require.False(t, a.Empty)
	require.NotEmpty(t, a.Blocks)
	for _, c := range r.calls {
		require.NotContains(t, c, "label=com.docker.compose.project")
		require.False(t, strings.HasPrefix(c, "run "), c)
	}
}
func TestRecoveryAuditUsesOnlyReadonlyBindAndFailsClosedForContents(t *testing.T) {
	for _, tc := range []struct {
		name              string
		nonempty, missing bool
	}{{"empty", false, false}, {"nonempty", true, false}, {"missing source", false, true}} {
		t.Run(tc.name, func(t *testing.T) {
			r := &auditRunner{nonempty: tc.nonempty, missingSource: tc.missing}
			d, p := auditFixture(t, r)
			a, e := d.AuditRecovery(context.Background(), p, true)
			if tc.missing {
				require.Error(t, e)
			} else {
				require.NoError(t, e)
				require.Equal(t, !tc.nonempty, a.Empty)
			}
			reads := 0
			for _, c := range r.calls {
				if strings.HasPrefix(c, "run ") {
					reads++
					require.Contains(t, c, "type=bind,src=/var/lib/docker/volumes/")
					require.Contains(t, c, ",readonly")
					require.Contains(t, c, "--network none")
					require.Contains(t, c, "--read-only")
					require.NotContains(t, c, "type=volume")
				}
				require.False(t, strings.Contains(c, "volume create"), c)
			}
			require.Equal(t, 1, reads)
		})
	}
}

func TestRecoveryAuditRejectsClientAppearingDuringHelper(t *testing.T) {
	r := &auditRunner{}
	d, p := auditFixture(t, r)
	observations := 0
	d.clientsIdle = func(context.Context) (bool, error) { observations++; return observations == 1, nil }
	a, e := d.AuditRecovery(context.Background(), p, true)
	require.NoError(t, e)
	require.False(t, a.Empty)
	require.NotEmpty(t, a.Blocks)
	require.Equal(t, 2, observations)
}
