package setup

import (
	"context"
	"encoding/json"
	"fmt"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	"task-processor/internal/app/setupcontrol"
	"task-processor/internal/platform/privatefile"
)

type recoveryVolume struct {
	Name, Driver, Mountpoint, CreatedAt string
	Labels, Options                     map[string]string
}

func inProject(name, project string) bool {
	return name == project || strings.HasPrefix(name, project+"-") || strings.HasPrefix(name, project+"_")
}
func readPlanDocument(n nativePlan) (map[string]any, error) {
	b, e := readPrivate(n.ComposeFile, 2*1024*1024)
	if e != nil || hash(b) != n.ComposeHash {
		return nil, setupcontrol.ErrDrift
	}
	var doc map[string]any
	if json.Unmarshal(b, &doc) != nil || doc["name"] != n.Project {
		return nil, setupcontrol.ErrInvalid
	}
	return doc, nil
}
func (d *Driver) recoveryMetadata(ctx context.Context, p setupcontrol.Prepared) (setupcontrol.RecoveryAudit, []recoveryVolume, error) {
	a := setupcontrol.RecoveryAudit{Blocks: []string{}, Changes: []setupcontrol.Change{}}
	n, e := loadNative(p)
	if e != nil || n.Mode != "fresh" || p.Request.Project != n.Project {
		return a, nil, setupcontrol.ErrInvalid
	}
	id, e := d.identity(ctx, n.Project)
	if e != nil || id != p.InstanceID {
		return a, nil, setupcontrol.ErrDrift
	}
	doc, e := readPlanDocument(n)
	if e != nil {
		return a, nil, e
	}
	definitions, ok := doc["volumes"].(map[string]any)
	if !ok || len(definitions) < 1 || len(definitions) > 128 {
		return a, nil, setupcontrol.ErrInvalid
	}
	expected := map[string]string{}
	for logical, raw := range definitions {
		v, ok := raw.(map[string]any)
		if !ok || v["external"] == true {
			return a, nil, setupcontrol.ErrInvalid
		}
		name, ok := v["name"].(string)
		if !ok || !inProject(name, n.Project) || strings.ContainsAny(name, "/\\,:") || expected[name] != "" {
			return a, nil, setupcontrol.ErrInvalid
		}
		expected[name] = logical
	}
	raw, e := d.run(ctx, "volume", "ls", "-q")
	if e != nil {
		return a, nil, e
	}
	names := strings.Fields(string(raw))
	if len(names) > 2048 {
		return a, nil, setupcontrol.ErrUnavailable
	}
	all := []recoveryVolume{}
	if len(names) > 0 {
		raw, e = d.run(ctx, append([]string{"volume", "inspect"}, names...)...)
		if e != nil || json.Unmarshal(raw, &all) != nil || len(all) != len(names) {
			return a, nil, setupcontrol.ErrUnavailable
		}
	}
	selected := []recoveryVolume{}
	for _, v := range all {
		if expected[v.Name] == "" && !inProject(v.Name, n.Project) && v.Labels["com.docker.compose.project"] != n.Project {
			continue
		}
		selected = append(selected, v)
		if expected[v.Name] == "" || v.Driver != "local" || len(v.Options) > 0 || v.Labels["com.docker.compose.project"] != n.Project || v.Labels["com.docker.compose.volume"] != expected[v.Name] || v.CreatedAt == "" {
			a.Blocks = append(a.Blocks, "存储归属或配置与原安装不一致，请保留现场并核对")
		}
	}
	sort.Slice(selected, func(i, j int) bool { return selected[i].Name < selected[j].Name })
	if len(selected) != 0 && len(selected) != len(expected) {
		a.Blocks = append(a.Blocks, "原安装的存储集合不完整，不能重新尝试")
	}
	rootRaw, e := d.run(ctx, "info", "--format", "{{.DockerRootDir}}")
	if e != nil {
		return a, nil, e
	}
	dataRoot := strings.TrimSpace(string(rootRaw))
	if !strings.HasPrefix(dataRoot, "/") || path.Clean(dataRoot) != dataRoot || dataRoot == "/" {
		return a, nil, setupcontrol.ErrUnavailable
	}
	cs, e := d.inspectContainers(ctx, []string{"ps", "-aq"})
	if e != nil {
		return a, nil, e
	}
	for _, c := range cs {
		if c.Config.Labels["com.docker.compose.project"] == n.Project || inProject(strings.TrimPrefix(c.Name, "/"), n.Project) {
			a.Blocks = append(a.Blocks, "仍有安装服务或容器，不能归档或重新尝试")
		}
		for _, m := range c.Mounts {
			for name := range expected {
				root := path.Join(dataRoot, "volumes", name, "_data")
				if m.Type == "volume" && m.Name == name || m.Type == "bind" && (m.Source == root || strings.HasPrefix(m.Source, root+"/") || strings.HasPrefix(root, strings.TrimSuffix(m.Source, "/")+"/")) {
					a.Blocks = append(a.Blocks, "安装存储仍被容器引用，不能归档或重新尝试")
				}
			}
		}
	}
	raw, e = d.run(ctx, "network", "ls", "-q")
	if e != nil {
		return a, nil, e
	}
	ids := strings.Fields(string(raw))
	if len(ids) > 512 {
		return a, nil, setupcontrol.ErrUnavailable
	}
	var networks []struct {
		Name   string
		Labels map[string]string
	}
	if len(ids) > 0 {
		raw, e = d.run(ctx, append([]string{"network", "inspect"}, ids...)...)
		if e != nil || json.Unmarshal(raw, &networks) != nil {
			return a, nil, setupcontrol.ErrUnavailable
		}
	}
	for _, network := range networks {
		if inProject(network.Name, n.Project) || network.Labels["com.docker.compose.project"] == n.Project {
			a.Blocks = append(a.Blocks, "仍有原安装网络，需先核对当前状态")
		}
	}
	a.Fingerprint = hash(marshal([]any{id, expected, selected}))
	a.Removed = len(selected) == 0 && len(a.Blocks) == 0
	a.Changes = append(a.Changes, setupcontrol.Change{Label: "专属存储", Before: fmt.Sprintf("原计划%d个", len(expected)), After: fmt.Sprintf("当前%d个；归属和引用已检查", len(selected))})
	return a, selected, nil
}
func (d *Driver) AuditRecovery(ctx context.Context, p setupcontrol.Prepared, contents bool) (setupcontrol.RecoveryAudit, error) {
	a, volumes, e := d.recoveryMetadata(ctx, p)
	if e != nil || len(a.Blocks) > 0 {
		return a, e
	}
	if !d.noRecoveryClients(ctx) {
		a.Removed = false
		a.Blocks = append(a.Blocks, "本机安装命令仍在运行，或当前环境无法完整核实执行进程；请稍后重新检查")
		return a, nil
	}
	if !contents || len(volumes) == 0 {
		after, _, err := d.recoveryMetadata(ctx, p)
		if err != nil || len(after.Blocks) > 0 || after.Fingerprint != a.Fingerprint {
			return a, setupcontrol.ErrDrift
		}
		if !d.noRecoveryClients(ctx) {
			a.Removed = false
			a.Blocks = append(a.Blocks, "仍有本机安装命令，不能结束原操作")
			return a, nil
		}
		a.Empty = contents && len(volumes) == 0
		return a, nil
	}
	raw, e := d.run(ctx, "info", "--format", "{{.DockerRootDir}}")
	if e != nil {
		return a, e
	}
	root := strings.TrimSpace(string(raw))
	if !strings.HasPrefix(root, "/") || path.Clean(root) != root || root == "/" {
		return a, setupcontrol.ErrUnavailable
	}
	raw, e = d.run(ctx, "image", "inspect", "alpine:3.22")
	var images []struct {
		ID string `json:"Id"`
	}
	if e != nil || json.Unmarshal(raw, &images) != nil || len(images) != 1 || !strings.HasPrefix(images[0].ID, "sha256:") {
		return a, setupcontrol.ErrUnavailable
	}
	name := "setup-read-" + uuid.NewString()
	args := []string{"run", "--rm", "--pull", "never", "--name", name, "--label", "io.shuomi.setup-read=" + name, "--network", "none", "--read-only", "--cap-drop", "ALL", "--security-opt", "no-new-privileges", "--pids-limit", "32", "--memory", "32m", "--cpus", "0.5"}
	for i, v := range volumes {
		if v.Mountpoint != path.Join(root, "volumes", v.Name, "_data") {
			return a, setupcontrol.ErrUnavailable
		}
		args = append(args, "--mount", fmt.Sprintf("type=bind,src=%s,dst=/inspect/v%d,readonly", v.Mountpoint, i))
	}
	args = append(args, images[0].ID, "sh", "-eu", "-c", `for p in /inspect/*; do test -d "$p"; v="$(find "$p" -mindepth 1 -maxdepth 1 -print -quit)"; if test -n "$v"; then echo NONEMPTY; exit 0; fi; done; echo EMPTY`)
	readCtx, cancel := context.WithTimeout(ctx, 45*time.Second)
	defer cancel()
	defer d.removeReadHelper(name)
	raw, e = d.run(readCtx, args...)
	if e != nil {
		return a, e
	}
	if string(raw) != "EMPTY\n" && string(raw) != "NONEMPTY\n" {
		return a, setupcontrol.ErrUnavailable
	}
	after, _, e := d.recoveryMetadata(ctx, p)
	if e != nil || len(after.Blocks) > 0 || after.Fingerprint != a.Fingerprint {
		return a, setupcontrol.ErrDrift
	}
	if !d.noRecoveryClients(ctx) {
		a.Removed = false
		a.Blocks = append(a.Blocks, "仍有本机安装命令，不能重新尝试")
		return a, nil
	}
	a.Empty = string(raw) == "EMPTY\n"
	if !a.Empty {
		a.Blocks = append(a.Blocks, "至少一个存储中已有内容，请保留数据并由维护人员处理")
	}
	a.Changes = append(a.Changes, setupcontrol.Change{Label: "存储内容", Before: "未检查", After: map[bool]string{true: "全部为空，未读取任何文件内容", false: "发现已有内容，保持原样"}[a.Empty]})
	return a, nil
}
func (d *Driver) noRecoveryClients(ctx context.Context) bool {
	if d.clientsIdle == nil {
		return false
	}
	idle, e := d.clientsIdle(ctx)
	return e == nil && idle
}
func (d *Driver) removeReadHelper(name string) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	raw, e := d.run(ctx, "inspect", name)
	var cs []container
	if e == nil && json.Unmarshal(raw, &cs) == nil && len(cs) == 1 && cs[0].Config.Labels["io.shuomi.setup-read"] == name {
		_, _ = d.run(ctx, "rm", "-f", cs[0].ID)
	}
}
func (d *Driver) installationDriver(ctx context.Context, n nativePlan) (*Driver, error) {
	doc, e := readPlanDocument(n)
	if e != nil {
		return nil, e
	}
	services, ok := doc["services"].(map[string]any)
	if !ok {
		return nil, setupcontrol.ErrInvalid
	}
	if len(n.RequiredServices) == 0 || hash(marshal(n.RequiredServices)) != hash(marshal(requiredServices(services))) {
		return nil, setupcontrol.ErrUnavailable
	}
	var source string
	if n.ReleaseFile != "" {
		if !filepath.IsAbs(n.ReleaseFile) || filepath.Base(n.ReleaseFile) != "release.json" {
			return nil, setupcontrol.ErrDrift
		}
		source = filepath.Join(filepath.Dir(n.ReleaseFile), "source")
	}
	for _, name := range n.BuildServices {
		service, ok := services[name].(map[string]any)
		if !ok {
			return nil, setupcontrol.ErrInvalid
		}
		build, ok := service["build"].(map[string]any)
		if !ok {
			return nil, setupcontrol.ErrInvalid
		}
		value, ok := build["context"].(string)
		if !ok {
			return nil, setupcontrol.ErrInvalid
		}
		value = hostPath(value)
		if !filepath.IsAbs(value) {
			return nil, setupcontrol.ErrDrift
		}
		if source == "" {
			source = value
		}
		relative, err := filepath.Rel(source, value)
		if err != nil || !filepath.IsLocal(relative) || privatefile.RejectPath(value) != nil {
			return nil, setupcontrol.ErrDrift
		}
	}
	if source == "" {
		return nil, setupcontrol.ErrUnavailable
	}
	releaseFile := filepath.Join(filepath.Dir(source), "release.json")
	if n.ReleaseFile != "" && filepath.Clean(n.ReleaseFile) != releaseFile {
		return nil, setupcontrol.ErrDrift
	}
	b, e := readPrivate(releaseFile, 2*1024*1024)
	var release Release
	if e != nil || json.Unmarshal(b, &release) != nil {
		return nil, setupcontrol.ErrUnavailable
	}
	for name, id := range map[string]string{"current-application": release.BackendImage, "listingkit-ui": release.UIImage} {
		s, ok := services[name].(map[string]any)
		if !ok || s["image"] != id {
			return nil, setupcontrol.ErrDrift
		}
	}
	installer, e := New(source, release, d.runner)
	if e != nil {
		return nil, e
	}
	if installer.dockerContext != d.dockerContext {
		return nil, setupcontrol.ErrDrift
	}
	if e = installer.images(ctx); e != nil {
		return nil, e
	}
	return installer, nil
}
func (d *Driver) PrepareRetry(ctx context.Context, p setupcontrol.Prepared, subnet, dir string) (setupcontrol.Prepared, error) {
	n, e := loadNative(p)
	if e != nil || n.Mode != "fresh" {
		return p, setupcontrol.ErrInvalid
	}
	a, e := d.AuditRecovery(ctx, p, true)
	if e != nil || !a.Empty || len(a.Blocks) > 0 {
		return p, setupcontrol.ErrUnresolved
	}
	installer, e := d.installationDriver(ctx, n)
	if e != nil {
		return p, e
	}
	r := p.Request
	if subnet != "" {
		r.NetworkSubnet = subnet
	}
	if e = validateNew(r, false); e != nil {
		return p, e
	}
	if e = freePorts(r); e != nil {
		return p, e
	}
	if e = d.checkSubnet(ctx, r.NetworkSubnet); e != nil {
		return p, e
	}
	doc, e := readPlanDocument(n)
	if e != nil {
		return p, e
	}
	if e = configureNetworks(doc, r.NetworkSubnet); e != nil {
		return p, e
	}
	file := filepath.Join(dir, "retry-compose.json")
	if e = writeCompose(file, doc); e != nil {
		return p, e
	}
	raw, e := readPrivate(file, 2*1024*1024)
	if e != nil {
		return p, e
	}
	n.ComposeFile, n.ComposeHash, n.RecoveryFingerprint = file, hash(raw), a.Fingerprint
	n.ReleaseFile = filepath.Join(filepath.Dir(installer.source), "release.json")
	// Credentials are handed off only after this same attempt fully verifies.
	n.Handoff = filepath.Join(dir, "handoff")
	if e = privatefile.RejectPath(n.Handoff); e != nil {
		return p, e
	}
	previousSubnet := p.Request.NetworkSubnet
	p.Request, p.Data, p.Before, p.Candidate = r, marshal(n), a.Fingerprint, hash(marshal(n))
	p.Changes = append([]setupcontrol.Change{}, p.Changes...)
	for i := range p.Changes {
		if p.Changes[i].Label == "私密登录资料" {
			p.Changes[i].After = n.Handoff
		}
	}
	p.Changes = append(p.Changes, setupcontrol.Change{Label: "重新尝试", Before: "原安装未确认完成", After: "保留全部空存储，使用原安装版本"}, setupcontrol.Change{Label: "网络地址范围", Before: previousSubnet, After: r.NetworkSubnet})
	return p, nil
}
