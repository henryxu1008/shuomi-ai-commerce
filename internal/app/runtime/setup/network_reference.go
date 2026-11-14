package setup

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"task-processor/internal/platform/privatefile"
)

// Compose hashes service network references after resolving their exact running
// container. Its declarative `config --hash` cannot reproduce that hash until
// this reference is resolved. Other differences continue to fail closed.
func resolveNetworkReference(document map[string]any, name string, current container, peers []container) bool {
	services, ok := document["services"].(map[string]any)
	if !ok {
		return false
	}
	serviceConfig, ok := services[name].(map[string]any)
	if !ok {
		return false
	}
	mode, ok := serviceConfig["network_mode"].(string)
	if !ok || !strings.HasPrefix(mode, "service:") {
		return false
	}
	referenced, ok := service(peers, strings.TrimPrefix(mode, "service:"))
	if !ok || !referenced.State.Running {
		return false
	}
	expected := "container:" + referenced.ID
	if current.HostConfig.NetworkMode != expected {
		return false
	}
	serviceConfig["network_mode"] = expected
	return true
}
func (d *Driver) runningHashMatches(ctx context.Context, project string, files []string, name string, current container) bool {
	raw, e := d.compose(ctx, project, files, "config", "--format", "json")
	if e != nil {
		return false
	}
	var document map[string]any
	if json.Unmarshal(raw, &document) != nil {
		return false
	}
	peers, e := d.containers(ctx, project)
	if e != nil || !resolveNetworkReference(document, name, current, peers) {
		return false
	}
	dir, e := os.MkdirTemp("", "shuomi-compose-check-")
	if e != nil {
		return false
	}
	file := filepath.Join(dir, "resolved.json")
	defer os.Remove(dir)
	defer os.Remove(file)
	if privatefile.RejectPath(dir) != nil || writeCompose(file, document) != nil {
		return false
	}
	raw, e = d.compose(ctx, project, []string{file}, "config", "--hash", name)
	if e != nil {
		return false
	}
	parts := strings.Fields(string(raw))
	return len(parts) == 2 && parts[0] == name && parts[1] == current.Config.Labels["com.docker.compose.config-hash"]
}
