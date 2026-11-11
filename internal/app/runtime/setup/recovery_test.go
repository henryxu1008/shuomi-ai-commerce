package setup

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"task-processor/internal/app/setupcontrol"
)

type recoveryBuildRunner struct {
	auditRunner
	afterBuild func()
	up         int
	revision   string
}

func (r *recoveryBuildRunner) Run(ctx context.Context, dir string, args []string) ([]byte, error) {
	if len(args) > 1 && args[0] == "--context" {
		args = args[2:]
	}
	if args[0] == "context" {
		return []byte("desktop-linux"), nil
	}
	if args[0] == "image" && args[2] != "alpine:3.22" {
		return marshal([]any{map[string]any{"Id": args[2], "Config": map[string]any{"Labels": map[string]string{"org.opencontainers.image.revision": r.revision}}}}), nil
	}
	if args[0] == "compose" {
		for _, a := range args {
			if a == "build" && r.afterBuild != nil {
				r.afterBuild()
			}
			if a == "up" {
				r.up++
			}
		}
		return nil, nil
	}
	return r.auditRunner.Run(ctx, dir, args)
}
func TestRecoveryRechecksConditionsAfterBuildBeforeStarting(t *testing.T) {
	for _, mode := range []string{"nonempty", "reference", "source", "unchanged"} {
		t.Run(mode, func(t *testing.T) {
			r := &recoveryBuildRunner{revision: strings.Repeat("a", 40)}
			d, p := auditFixture(t, &r.auditRunner)
			d.runner = r
			d.dockerContext = "desktop-linux"
			root := t.TempDir()
			source := filepath.Join(root, "source")
			require.NoError(t, os.MkdirAll(source, 0700))
			sentinel := filepath.Join(source, "sentinel")
			require.NoError(t, os.WriteFile(sentinel, []byte("source"), 0600))
			release := Release{SchemaVersion: 1, SourceCommit: r.revision, BackendImage: "sha256:backend", UIImage: "sha256:ui", Files: map[string]string{"sentinel": hash([]byte("source"))}}
			require.NoError(t, os.WriteFile(filepath.Join(root, "release.json"), marshal(release), 0600))
			d.source, d.release = source, release
			var n nativePlan
			require.NoError(t, json.Unmarshal(p.Data, &n))
			doc, e := readPlanDocument(n)
			require.NoError(t, e)
			services := map[string]any{"current-application": map[string]any{"image": release.BackendImage}, "listingkit-ui": map[string]any{"image": release.UIImage}, "builder": map[string]any{"build": map[string]any{"context": filepath.ToSlash(source)}}}
			doc["services"] = services
			raw := marshal(doc)
			require.NoError(t, os.WriteFile(n.ComposeFile, raw, 0600))
			n.ComposeHash = hash(raw)
			n.BuildServices = []string{"builder"}
			n.RequiredServices = requiredServices(services)
			p.Data = marshal(n)
			a, e := d.AuditRecovery(context.Background(), p, true)
			require.NoError(t, e)
			require.True(t, a.Empty)
			n.RecoveryFingerprint = a.Fingerprint
			p.Data = marshal(n)
			r.afterBuild = func() {
				switch mode {
				case "nonempty":
					r.nonempty = true
				case "reference":
					r.foreign = true
				case "source":
					require.NoError(t, os.WriteFile(sentinel, []byte("changed"), 0600))
				}
			}
			e = d.Apply(context.Background(), p, func(string) error { return nil })
			if mode == "unchanged" {
				require.NoError(t, e)
				require.Equal(t, 1, r.up)
			} else {
				require.Error(t, e)
				require.Zero(t, r.up, "must not initialise after build-time drift")
			}
		})
	}
}

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
