// Package setup assembles the local installer with the existing Compose and
// current-application owners. The serving application never imports this driver.
package setup

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"task-processor/internal/app/setupcontrol"
	"task-processor/internal/platform/privatefile"
)

type Runner interface {
	Run(context.Context, string, []string) ([]byte, error)
}
type Docker struct{}
type boundedBuffer struct {
	bytes.Buffer
	overflow bool
}

func (b *boundedBuffer) Write(v []byte) (int, error) {
	n := len(v)
	remaining := 2*1024*1024 - b.Len()
	if remaining < 0 {
		remaining = 0
	}
	if len(v) > remaining {
		b.overflow = true
		v = v[:remaining]
	}
	_, _ = b.Buffer.Write(v)
	return n, nil
}
func (Docker) Run(ctx context.Context, dir string, args []string) ([]byte, error) {
	c := exec.CommandContext(ctx, "docker", args...)
	c.Env = dockerEnvironment()
	c.Dir = dir
	c.Stderr = io.Discard
	var out boundedBuffer
	c.Stdout = &out
	if c.Run() != nil || out.overflow {
		return nil, setupcontrol.ErrUnavailable
	}
	return out.Bytes(), nil
}

// Compose interpolation and opt-in profiles come from the exact prepared
// input, never a shell left over from another instance. Context is explicit;
// unrelated provider secrets are not inherited by the Docker subprocess.
func dockerEnvironment() []string {
	result := []string{}
	for _, v := range os.Environ() {
		key, _, ok := strings.Cut(v, "=")
		if !ok {
			continue
		}
		switch strings.ToUpper(key) {
		case "PATH", "PATHEXT", "SYSTEMROOT", "WINDIR", "SYSTEMDRIVE", "COMSPEC", "TEMP", "TMP", "USERPROFILE", "LOCALAPPDATA", "APPDATA", "PROGRAMDATA", "PROGRAMFILES", "PROGRAMFILES(X86)", "COMMONPROGRAMFILES", "HOME", "HOMEDRIVE", "HOMEPATH", "USERNAME", "USERDOMAIN", "USER", "LOGNAME", "LANG", "LC_ALL", "DOCKER_CONFIG", "DOCKER_CLI_PLUGIN_EXTRA_DIRS", "TERM", "NO_COLOR":
			result = append(result, v)
		}
	}
	return result
}

type container struct {
	ID     string `json:"Id"`
	Image  string
	Config struct {
		Image                string
		Env, Cmd, Entrypoint []string
		Labels               map[string]string
	}
	HostConfig struct {
		NetworkMode    string
		Binds          []string
		Mounts         json.RawMessage
		ReadonlyRootfs bool
		CapDrop        []string
		SecurityOpt    []string
	}
	Mounts []mount
	State  struct {
		Running  bool
		Status   string
		ExitCode int
		Health   *struct{ Status string }
	}
}
type mount struct {
	Type, Name, Source, Destination string
	RW                              bool
}

func hash(v []byte) string { h := sha256.Sum256(v); return hex.EncodeToString(h[:]) }
func marshal(v any) []byte { b, _ := json.Marshal(v); return b }
func (d *Driver) run(ctx context.Context, args ...string) ([]byte, error) {
	if d.dockerContext != "" {
		args = append([]string{"--context", d.dockerContext}, args...)
	}
	return d.runner.Run(ctx, d.source, args)
}
func (d *Driver) containers(ctx context.Context, project string) ([]container, error) {
	args := []string{"ps", "-aq", "--filter", "label=com.docker.compose.project"}
	if project != "" {
		args = append(args, "--filter", "label=com.docker.compose.project="+project)
	}
	raw, e := d.run(ctx, args...)
	if e != nil {
		return nil, e
	}
	ids := strings.Fields(string(raw))
	if len(ids) == 0 {
		return []container{}, nil
	}
	if len(ids) > 512 {
		return nil, setupcontrol.ErrUnavailable
	}
	raw, e = d.run(ctx, append([]string{"inspect"}, ids...)...)
	if e != nil {
		return nil, e
	}
	var result []container
	if json.Unmarshal(raw, &result) != nil {
		return nil, setupcontrol.ErrUnavailable
	}
	return result, nil
}
func service(cs []container, name string) (container, bool) {
	var found container
	n := 0
	for _, c := range cs {
		if c.Config.Labels["com.docker.compose.service"] == name && c.Config.Labels["com.docker.compose.oneoff"] != "True" {
			found = c
			n++
		}
	}
	return found, n == 1
}
func envValue(c container, key string) string {
	for _, v := range c.Config.Env {
		if strings.HasPrefix(v, key+"=") {
			return strings.TrimPrefix(v, key+"=")
		}
	}
	return ""
}
func hostPath(v string) string {
	if strings.HasPrefix(v, "/run/desktop/mnt/host/") {
		v = strings.TrimPrefix(v, "/run/desktop/mnt/host/")
		if len(v) > 2 && v[1] == '/' {
			v = strings.ToUpper(v[:1]) + ":" + v[1:]
		}
	}
	return filepath.FromSlash(v)
}
func composeArgs(project string, files []string) []string {
	args := []string{"compose", "-p", project}
	for _, f := range files {
		args = append(args, "-f", f)
	}
	return args
}
func (d *Driver) compose(ctx context.Context, project string, files []string, extra ...string) ([]byte, error) {
	return d.run(ctx, append(composeArgs(project, files), extra...)...)
}
func readPrivate(p string, limit int) ([]byte, error) {
	if !filepath.IsAbs(p) || privatefile.RejectPath(p) != nil {
		return nil, setupcontrol.ErrInvalid
	}
	f, e := os.Open(p)
	if e != nil {
		return nil, setupcontrol.ErrUnavailable
	}
	defer f.Close()
	s, e := f.Stat()
	if e != nil || !s.Mode().IsRegular() || s.Size() > int64(limit) {
		return nil, setupcontrol.ErrInvalid
	}
	b, e := io.ReadAll(io.LimitReader(f, int64(limit+1)))
	if e != nil || len(b) > limit {
		return nil, setupcontrol.ErrInvalid
	}
	return b, nil
}

// docker compose config emits a reusable document with dollar escaping already
// preserved. Escaping that document again corrupts shell substitutions.
func writeCompose(path string, v map[string]any) error {
	return privatefile.Write(path, marshal(v))
}
