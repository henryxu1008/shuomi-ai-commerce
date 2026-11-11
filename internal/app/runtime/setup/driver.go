package setup

import (
	"context"
	"encoding/json"
	"net"
	"net/url"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"task-processor/internal/app/runtime/currentapplication"
	"task-processor/internal/app/setupcontrol"
	"time"
)

type Release struct {
	SchemaVersion int               `json:"schemaVersion"`
	SourceCommit  string            `json:"sourceCommit"`
	BackendImage  string            `json:"backendImage"`
	UIImage       string            `json:"uiImage"`
	Files         map[string]string `json:"files"`
}
type Driver struct {
	dockerContext string
	source        string
	release       Release
	runner        Runner
}

func New(source string, release Release, runner Runner) (*Driver, error) {
	if runner == nil {
		runner = Docker{}
	}
	d := &Driver{source: source, release: release, runner: runner}
	if !filepath.IsAbs(source) || release.SchemaVersion != 1 || !regexp.MustCompile(`^[0-9a-f]{40}$`).MatchString(release.SourceCommit) || !strings.HasPrefix(release.BackendImage, "sha256:") || !strings.HasPrefix(release.UIImage, "sha256:") {
		return nil, setupcontrol.ErrInvalid
	}
	if e := d.verifySource(); e != nil {
		return nil, e
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	contextName, e := runner.Run(ctx, source, []string{"context", "show"})
	if e != nil || strings.TrimSpace(string(contextName)) == "" {
		return nil, setupcontrol.ErrUnavailable
	}
	d.dockerContext = strings.TrimSpace(string(contextName))
	return d, nil
}
func (d *Driver) verifySource() error {
	if len(d.release.Files) == 0 {
		return setupcontrol.ErrInvalid
	}
	for path, want := range d.release.Files {
		if !filepath.IsLocal(path) || filepath.ToSlash(filepath.Clean(filepath.FromSlash(path))) != path || strings.ContainsAny(path, ":\\") {
			return setupcontrol.ErrInvalid
		}
		b, e := readPrivate(filepath.Join(d.source, filepath.FromSlash(path)), 20*1024*1024)
		if e != nil || hash(b) != want {
			return setupcontrol.ErrDrift
		}
	}
	return nil
}
func (d *Driver) images(ctx context.Context) error {
	for _, id := range []string{d.release.BackendImage, d.release.UIImage} {
		b, e := d.run(ctx, "image", "inspect", id)
		var values []struct {
			ID     string `json:"Id"`
			Config struct{ Labels map[string]string }
		}
		if e != nil || json.Unmarshal(b, &values) != nil || len(values) != 1 || values[0].ID != id || values[0].Config.Labels["org.opencontainers.image.revision"] != d.release.SourceCommit {
			return setupcontrol.ErrUnavailable
		}
	}
	return nil
}
func (d *Driver) identity(ctx context.Context, project string) (string, error) {
	b, e := d.run(ctx, "info", "--format", "{{.ID}}")
	if e != nil || strings.TrimSpace(string(b)) == "" {
		return "", setupcontrol.ErrUnavailable
	}
	// Aliases of the same daemon resolve to the same ID and lock directory.
	return "docker-" + hash([]byte(strings.TrimSpace(string(b)) + "\n" + project))[:40], nil
}
func (d *Driver) Identity(ctx context.Context, r setupcontrol.Request) (string, error) {
	if r.Mode == "fresh" {
		if e := validateNew(r, false); e != nil {
			return "", e
		}
		return d.identity(ctx, r.Project)
	}
	if r.Mode != "existing" {
		return "", setupcontrol.ErrInvalid
	}
	cs, e := d.containers(ctx, "")
	if e != nil {
		return "", e
	}
	for _, c := range cs {
		if c.Config.Labels["com.docker.compose.service"] != "current-application" {
			continue
		}
		id, e := d.identity(ctx, c.Config.Labels["com.docker.compose.project"])
		if e == nil && id == r.InstanceID {
			return id, nil
		}
	}
	return "", setupcontrol.ErrInvalid
}
func (d *Driver) Snapshot(ctx context.Context) (setupcontrol.Snapshot, error) {
	s := setupcontrol.Snapshot{Version: d.release.SourceCommit, Instances: []setupcontrol.Instance{}, Profiles: profiles(), Blocks: []string{}}
	s.HostFreeBytes = availableSpace(d.source)
	if d.images(ctx) != nil {
		s.Blocks = append(s.Blocks, "此版本的准确后端或界面镜像尚未准备好")
	}
	cs, e := d.containers(ctx, "")
	if e != nil {
		s.Blocks = append(s.Blocks, "Docker未连接；请先在本机启动Docker引擎")
		return s, nil
	}
	s.DockerReady = true
	seen := map[string]bool{}
	for _, c := range cs {
		p := c.Config.Labels["com.docker.compose.project"]
		if c.Config.Labels["com.docker.compose.service"] != "current-application" || seen[p] {
			continue
		}
		seen[p] = true
		id, e := d.identity(ctx, p)
		if e != nil {
			return s, e
		}
		group := []container{}
		for _, v := range cs {
			if v.Config.Labels["com.docker.compose.project"] == p {
				group = append(group, v)
			}
		}
		s.Instances = append(s.Instances, describe(id, p, group))
	}
	sort.Slice(s.Instances, func(i, j int) bool { return s.Instances[i].Project < s.Instances[j].Project })
	return s, nil
}
func describe(id, p string, cs []container) setupcontrol.Instance {
	a, _ := service(cs, "current-application")
	u, _ := service(cs, "listingkit-ui")
	origin := envValue(u, "LISTINGKIT_PUBLIC_BASE_URL")
	parsed, e := url.Parse(origin)
	if e != nil || parsed.Scheme != "https" || parsed.Hostname() != "localhost" || parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" {
		origin = ""
	}
	return setupcontrol.Instance{ID: id, Project: p, Origin: origin, BackendImage: a.Image, UIImage: u.Image, Blocks: []string{}}
}
func (d *Driver) find(ctx context.Context, id string) (setupcontrol.Instance, []container, error) {
	cs, e := d.containers(ctx, "")
	if e != nil {
		return setupcontrol.Instance{}, nil, e
	}
	for _, c := range cs {
		if c.Config.Labels["com.docker.compose.service"] != "current-application" {
			continue
		}
		p := c.Config.Labels["com.docker.compose.project"]
		v, e := d.identity(ctx, p)
		if e == nil && v == id {
			group := []container{}
			for _, v := range cs {
				if v.Config.Labels["com.docker.compose.project"] == p {
					group = append(group, v)
				}
			}
			return describe(id, p, group), group, nil
		}
	}
	return setupcontrol.Instance{}, nil, setupcontrol.ErrInvalid
}
func configPath(c container) (string, error) {
	for i, v := range c.Config.Cmd {
		if (v == "-config" || v == "--config") && i+1 < len(c.Config.Cmd) {
			p := c.Config.Cmd[i+1]
			if strings.HasPrefix(p, "/private/") && !strings.Contains(p, "..") {
				return p, nil
			}
		}
	}
	return "", setupcontrol.ErrUnavailable
}
func (d *Driver) manifest(ctx context.Context, a container) ([]byte, error) {
	p, e := configPath(a)
	if e != nil {
		return nil, e
	}
	b, e := d.run(ctx, "exec", a.ID, "cat", p)
	if e != nil || len(b) > 65536 {
		return nil, setupcontrol.ErrUnavailable
	}
	return b, nil
}
func (d *Driver) inspectModules(ctx context.Context, a container, file string) ([]currentapplication.ModuleInspection, error) {
	args := []string{"run", "--rm", "--read-only", "--cap-drop", "ALL", "--security-opt", "no-new-privileges", "--pids-limit", "128", "--memory", "512m", "--cpus", "1", "--network", "container:" + a.ID, "--volumes-from", a.ID + ":ro"}
	path, e := configPath(a)
	if e != nil {
		return nil, e
	}
	if file != "" {
		path = "/setup-config.json"
		args = append(args, "--mount", "type=bind,src="+file+",dst="+path+",readonly")
	}
	for _, key := range []string{"SSL_CERT_FILE", "SSL_CERT_DIR"} {
		if v := envValue(a, key); v != "" {
			args = append(args, "-e", key+"="+v)
		}
	}
	args = append(args, "--entrypoint", "/usr/local/bin/current-application", d.release.BackendImage, "-config", path, "-inspect-modules")
	b, e := d.run(ctx, args...)
	if e != nil {
		return nil, e
	}
	var modules []currentapplication.ModuleInspection
	if json.Unmarshal(b, &modules) != nil {
		return nil, setupcontrol.ErrUnavailable
	}
	return modules, nil
}
func (d *Driver) Inspect(ctx context.Context, id string) (setupcontrol.Instance, error) {
	v, cs, e := d.find(ctx, id)
	if e != nil {
		return v, e
	}
	a, aok := service(cs, "current-application")
	u, uok := service(cs, "listingkit-ui")
	if !aok || !uok || !a.State.Running {
		v.Blocks = append(v.Blocks, "后端或界面不是唯一运行服务；无法安全管理")
		return v, nil
	}
	if _, e = d.effective(ctx, v.Project, cs); e != nil {
		v.Blocks = append(v.Blocks, "无法核实当前服务与完整Compose组合一致；请先保留并恢复准确的部署文件")
	}
	mods, e := d.inspectModules(ctx, a, "")
	if e != nil {
		v.Blocks = append(v.Blocks, "原配置或原生只读检查无法完成；未启动业务运行流程")
		return v, nil
	}
	for _, m := range mods {
		var ui *bool
		if m.UIFlag != "" {
			b := envValue(u, m.UIFlag) == "true"
			ui = &b
		}
		v.Modules = append(v.Modules, setupcontrol.Module{ID: m.ID, Label: m.Label, Configured: m.Configured, Enabled: m.Enabled, Installed: m.Installed, UIEnabled: ui, Mutable: m.Installed == "verified" && len(v.Blocks) == 0, Note: m.Note})
	}
	return v, nil
}
func changeModules(original, provided map[string]any, selected map[string]bool) (map[string]any, error) {
	var next map[string]any
	_ = json.Unmarshal(marshal(original), &next)
	definitions := map[string]currentapplication.InstallModule{}
	for _, m := range currentapplication.InstallationModules() {
		definitions[m.ID] = m
	}
	if len(selected) == 0 {
		return nil, setupcontrol.ErrInvalid
	}
	for id, enabled := range selected {
		m, ok := definitions[id]
		if !ok {
			return nil, setupcontrol.ErrInvalid
		}
		v := next[id]
		if v == nil && enabled && provided != nil {
			v = provided[id]
			next[id] = v
		}
		switch m.Toggle {
		case "boolean":
			next[id] = enabled
		case "enabled":
			obj, ok := v.(map[string]any)
			if !ok {
				return nil, setupcontrol.ErrInvalid
			}
			obj["enabled"] = enabled
		case "presence":
			if enabled {
				if v == nil {
					return nil, setupcontrol.ErrInvalid
				}
			} else {
				delete(next, id)
			}
		}
	}
	return next, nil
}

var projectPattern = regexp.MustCompile(`^[a-z][a-z0-9-]{2,47}$`)

func validateNew(r setupcontrol.Request, exists bool) error {
	if e := validateSubnet(r.NetworkSubnet); e != nil {
		return e
	}
	if exists || !projectPattern.MatchString(r.Project) || r.Mode != "fresh" || r.InstanceID != "" || r.ConfigurationFile != "" || len(r.Enabled) != 0 {
		return setupcontrol.ErrInvalid
	}
	validProfile := false
	for _, p := range profiles() {
		if r.Profile == p.ID {
			validProfile = true
		}
	}
	if !validProfile {
		return setupcontrol.ErrInvalid
	}
	seen := map[int]bool{}
	for _, port := range []int{r.ApplicationPort, r.IdentityPort, r.MailPort} {
		if port < 1024 || port > 65535 || seen[port] {
			return setupcontrol.ErrInvalid
		}
		seen[port] = true
	}
	for _, p := range []int{r.ApplicationPort, r.IdentityPort} {
		switch p {
		case 1025, 3000, 3001, 5432, 5433, 7233, 8025, 8080, 8085, 9000, 18080:
			return setupcontrol.ErrInvalid
		}
	}
	return nil
}
func freePorts(r setupcontrol.Request) error {
	ls := []net.Listener{}
	defer func() {
		for _, l := range ls {
			_ = l.Close()
		}
	}()
	for _, p := range []int{r.ApplicationPort, r.IdentityPort, r.MailPort} {
		l, e := net.Listen("tcp", "127.0.0.1:"+strconv.Itoa(p))
		if e != nil {
			return setupcontrol.ErrInvalid
		}
		ls = append(ls, l)
	}
	return nil
}
func profiles() []setupcontrol.Profile {
	return []setupcontrol.Profile{{ID: "base", Label: "基础账户与店铺中心", Note: "身份、企业、权限、资源、账户与店铺；外部服务需另行配置"}, {ID: "unified", Label: "现有原生模块组合", Note: "复用知识库、数据服务、商品资料、通知、工具市场、生态及存储安装组合；Chat等后置安装仍需原owner前置"}}
}

// Files and manifests stay private; only a fixed public summary reaches HTTP.
type nativePlan struct {
	Project, Mode, ComposeFile, ComposeHash, ManifestFile, ManifestHash, Origin string
	Expected                                                                    map[string]bool
	RuntimeBindings                                                             string
	BuildServices                                                               []string
	RequiredServices                                                            map[string]string
	Handoff                                                                     string
}

func loadNative(p setupcontrol.Prepared) (nativePlan, error) {
	var n nativePlan
	if json.Unmarshal(p.Data, &n) != nil || n.Project == "" || n.Mode != p.Request.Mode {
		return n, setupcontrol.ErrInvalid
	}
	return n, nil
}
func (d *Driver) Fingerprint(ctx context.Context, p setupcontrol.Prepared) (string, error) {
	n, e := loadNative(p)
	if e != nil {
		return "", e
	}
	if e = d.candidateIntact(n); e != nil {
		return "", e
	}
	if n.Mode == "fresh" {
		return d.freshFingerprint(ctx, p.Request)
	}
	_, cs, e := d.find(ctx, p.InstanceID)
	if e != nil {
		return "", e
	}
	return d.retainedFingerprint(ctx, cs)
}
func (d *Driver) candidateIntact(n nativePlan) error {
	b, e := readPrivate(n.ComposeFile, 2*1024*1024)
	if e != nil || hash(b) != n.ComposeHash {
		return setupcontrol.ErrDrift
	}
	if n.ManifestFile != "" {
		b, e = readPrivate(n.ManifestFile, 65536)
		if e != nil || hash(b) != n.ManifestHash {
			return setupcontrol.ErrDrift
		}
	}
	return d.verifySource()
}
func (d *Driver) Apply(ctx context.Context, p setupcontrol.Prepared) error {
	n, e := loadNative(p)
	if e != nil {
		return e
	}
	if e = d.candidateIntact(n); e != nil {
		return e
	}
	if e = d.images(ctx); e != nil {
		return e
	}
	if n.Mode == "fresh" {
		if _, e = d.freshFingerprint(ctx, p.Request); e != nil {
			return e
		}
		if _, e = d.compose(ctx, n.Project, []string{n.ComposeFile}, "pull", "--ignore-buildable", "--policy", "missing"); e != nil {
			return e
		}
		if len(n.BuildServices) > 0 {
			_, e = d.compose(ctx, n.Project, []string{n.ComposeFile}, append([]string{"build"}, n.BuildServices...)...)
			if e != nil {
				return e
			}
		}
		_, e = d.compose(ctx, n.Project, []string{n.ComposeFile}, "up", "-d", "--no-build", "--pull", "never", "--wait", "--wait-timeout", "1200")
		return e
	}
	_, e = d.compose(ctx, n.Project, []string{n.ComposeFile}, "up", "-d", "--no-deps", "--no-build", "--pull", "never", "--wait", "--wait-timeout", "180", "current-application", "listingkit-ui")
	return e
}
func (d *Driver) Verify(ctx context.Context, p setupcontrol.Prepared) error {
	n, e := loadNative(p)
	if e != nil {
		return e
	}
	instance, e := d.identity(ctx, n.Project)
	if e != nil || instance != p.InstanceID {
		return setupcontrol.ErrDrift
	}
	if e = d.candidateIntact(n); e != nil {
		return e
	}
	cs, e := d.containers(ctx, n.Project)
	if e != nil {
		return e
	}
	a, ok := service(cs, "current-application")
	if !ok || !healthy(a) {
		return setupcontrol.ErrUnavailable
	}
	u, ok := service(cs, "listingkit-ui")
	if !ok || !healthy(u) {
		return setupcontrol.ErrUnavailable
	}
	for _, s := range []string{"current-application", "listingkit-ui"} {
		c, _ := service(cs, s)
		if !d.matches(ctx, n.Project, []string{n.ComposeFile}, s, c) {
			return setupcontrol.ErrDrift
		}
	}
	if n.Mode == "existing" && bindings(cs) != n.RuntimeBindings {
		return setupcontrol.ErrDrift
	}
	if n.Mode == "fresh" && !requiredHealthy(cs, n.RequiredServices) {
		return setupcontrol.ErrUnavailable
	}
	if n.ManifestFile != "" {
		b, e := d.manifest(ctx, a)
		if e != nil || hash(b) != n.ManifestHash {
			return setupcontrol.ErrDrift
		}
	}
	mods, e := d.inspectModules(ctx, a, "")
	if e != nil {
		return e
	}
	for id, want := range n.Expected {
		found := false
		for _, m := range mods {
			if m.ID == id {
				found = true
				if m.Enabled != want || (want && m.Installed != "verified") {
					return setupcontrol.ErrUnavailable
				}
			}
		}
		if !found {
			return setupcontrol.ErrUnavailable
		}
	}
	if n.Mode == "fresh" {
		return d.handoff(ctx, n.Project, n.Handoff)
	}
	return nil
}
func healthy(c container) bool {
	return c.State.Running && (c.State.Health == nil || c.State.Health.Status == "healthy")
}
