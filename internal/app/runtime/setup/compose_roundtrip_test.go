package setup

import (
	"context"
	"encoding/json"
	"github.com/stretchr/testify/require"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

func TestResolvedComposeRoundTripPreservesPrivateFileSubstitutions(t *testing.T) {
	if _, e := exec.LookPath("docker"); e != nil {
		t.Skip("Docker Compose CLI required; no daemon or containers are used")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	dir := t.TempDir()
	source := filepath.Join(dir, "source.json")
	candidate := filepath.Join(dir, "candidate.json")
	document := map[string]any{"services": map[string]any{"probe": map[string]any{"image": "alpine:3.22", "command": []any{"sh", "-ec", "export AUTH_SECRET=\"$$(cat /private/auth-secret)\"; exec node server.js"}, "environment": map[string]any{"LITERAL": "literal$$part"}}}}
	require.NoError(t, os.WriteFile(source, marshal(document), 0600))
	runner := Docker{}
	first, e := runner.Run(ctx, dir, []string{"compose", "-p", "setup-roundtrip", "-f", source, "config", "--format", "json"})
	if e != nil {
		t.Skip("Docker Compose plugin unavailable")
	}
	var resolved map[string]any
	require.NoError(t, json.Unmarshal(first, &resolved))
	require.NoError(t, writeCompose(candidate, resolved))
	second, e := runner.Run(ctx, dir, []string{"compose", "-p", "setup-roundtrip", "-f", candidate, "config", "--format", "json"})
	require.NoError(t, e)
	var replay map[string]any
	require.NoError(t, json.Unmarshal(second, &replay))
	require.Equal(t, resolved, replay, "normalised Compose output already preserves literal dollar escaping")
}
