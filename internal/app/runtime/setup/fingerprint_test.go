package setup

import (
	"context"
	"github.com/stretchr/testify/require"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type fingerprintRunner struct{}

func (fingerprintRunner) Run(_ context.Context, _ string, args []string) ([]byte, error) {
	for i, v := range args {
		if v == "--hash" {
			return []byte(args[i+1] + " stable\n"), nil
		}
	}
	if len(args) > 0 && args[0] == "exec" {
		return []byte(`{"schemaVersion":1}`), nil
	}
	return []byte(`{"name":"same-project","services":{}}`), nil
}
func TestBeforeFingerprintIncludesActualDatabaseVolumeBindings(t *testing.T) {
	file := filepath.Join(t.TempDir(), "compose.json")
	require.NoError(t, os.WriteFile(file, []byte("{}"), 0600))
	containers := []container{}
	for _, name := range []string{"current-application", "listingkit-ui", "runtime-network", "business-db"} {
		c := container{ID: name}
		c.Config.Labels = map[string]string{"com.docker.compose.project": "same-project", "com.docker.compose.service": name, "com.docker.compose.config-hash": "stable", "com.docker.compose.project.config_files": file}
		c.Config.Cmd = []string{"-config", "/private/runtime/current-application.json"}
		containers = append(containers, c)
	}
	containers[3].Mounts = []mount{{Type: "volume", Name: "original-data", Source: "/volumes/original", Destination: "/var/lib/postgresql/data"}}
	d := &Driver{runner: fingerprintRunner{}}
	before, e := d.retainedFingerprint(context.Background(), containers)
	require.NoError(t, e)
	containers[3].Mounts[0].Name = "replacement-data"
	containers[3].Mounts[0].Source = strings.ReplaceAll(containers[3].Mounts[0].Source, "original", "replacement")
	after, e := d.retainedFingerprint(context.Background(), containers)
	require.NoError(t, e)
	require.NotEqual(t, before, after, "a database binding change must reject the old plan before dispatch")
}
