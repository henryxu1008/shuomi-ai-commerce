package setup

import (
	"context"
	"encoding/json"
	"path/filepath"
	"sort"
	"strings"
	"task-processor/internal/app/runtime/currentapplication"
	"task-processor/internal/app/setupcontrol"
	coreconfig "task-processor/internal/core/config"
	"task-processor/internal/platform/privatefile"
)

func (d *Driver) matches(ctx context.Context, p string, files []string, s string, c container) bool {
	b, e := d.compose(ctx, p, files, "config", "--hash", s)
	if e != nil {
		return false
	}
	parts := strings.Fields(string(b))
	if len(parts) == 2 && parts[0] == s && parts[1] == c.Config.Labels["com.docker.compose.config-hash"] {
		return true
	}
	return d.runningHashMatches(ctx, p, files, s, c)
}

// Require one complete combination that reproduces BOTH live services. The UI
// may have been updated after the backend; selecting its older label is unsafe.
func (d *Driver) effective(ctx context.Context, p string, cs []container) (map[string]any, error) {
	a, aok := service(cs, "current-application")
	u, uok := service(cs, "listingkit-ui")
	if !aok || !uok {
		return nil, setupcontrol.ErrUnavailable
	}
	for _, owner := range []container{u, a} {
		label := owner.Config.Labels["com.docker.compose.project.config_files"]
		if label == "" {
			continue
		}
		files := strings.Split(label, ",")
		valid := true
		for i, f := range files {
			files[i] = hostPath(f)
			if _, e := readPrivate(files[i], 2*1024*1024); e != nil {
				valid = false
				break
			}
		}
		if !valid {
			continue
		}
		if !d.matches(ctx, p, files, "current-application", a) || !d.matches(ctx, p, files, "listingkit-ui", u) {
			continue
		}
		b, e := d.compose(ctx, p, files, "config", "--format", "json")
		if e != nil {
			continue
		}
		var v map[string]any
		if json.Unmarshal(b, &v) == nil && v["name"] == p {
			return v, nil
		}
	}
	return nil, setupcontrol.ErrUnavailable
}
func (d *Driver) retainedFingerprint(ctx context.Context, cs []container) (string, error) {
	a, ok := service(cs, "current-application")
	if !ok {
		return "", setupcontrol.ErrUnavailable
	}
	p := a.Config.Labels["com.docker.compose.project"]
	effective, e := d.effective(ctx, p, cs)
	if e != nil {
		return "", e
	}
	manifest, e := d.manifest(ctx, a)
	if e != nil {
		return "", e
	}
	type running struct {
		ID, Image  string
		Config     any
		HostConfig any
		Mounts     []mount
	}
	values := []running{}
	for _, s := range []string{"current-application", "listingkit-ui", "runtime-network"} {
		c, ok := service(cs, s)
		if !ok {
			return "", setupcontrol.ErrUnavailable
		}
		values = append(values, running{c.ID, c.Image, c.Config, c.HostConfig, c.Mounts})
	}
	return hash(marshal([]any{effective, hash(manifest), values, bindings(cs)})), nil
}
func bindings(cs []container) string {
	rows := []string{}
	for _, c := range cs {
		s := c.Config.Labels["com.docker.compose.service"]
		config, _ := configPath(c)
		for _, m := range c.Mounts {
			if s == "current-application" && m.Destination == config {
				continue
			}
			rows = append(rows, s+"|"+m.Type+"|"+m.Name+"|"+m.Source+"|"+m.Destination)
		}
	}
	sort.Strings(rows)
	return hash(marshal(rows))
}
func (d *Driver) Prepare(ctx context.Context, r setupcontrol.Request, dir string) (setupcontrol.Prepared, error) {
	if d.images(ctx) != nil {
		return setupcontrol.Prepared{}, setupcontrol.ErrUnavailable
	}
	if r.Mode == "fresh" {
		return d.prepareFresh(ctx, r, dir)
	}
	if r.Mode != "existing" || r.Project != "" || r.Profile != "" || r.ApplicationPort != 0 || r.IdentityPort != 0 || r.MailPort != 0 || r.ToolReleaseDirectory != "" || r.NetworkSubnet != "" {
		return setupcontrol.Prepared{}, setupcontrol.ErrInvalid
	}
	v, cs, e := d.find(ctx, r.InstanceID)
	if e != nil {
		return setupcontrol.Prepared{}, e
	}
	a, ok := service(cs, "current-application")
	if !ok || !a.State.Running {
		return setupcontrol.Prepared{}, setupcontrol.ErrUnavailable
	}
	effective, e := d.effective(ctx, v.Project, cs)
	if e != nil {
		return setupcontrol.Prepared{}, e
	}
	before, e := d.retainedFingerprint(ctx, cs)
	if e != nil {
		return setupcontrol.Prepared{}, e
	}
	original, e := d.manifest(ctx, a)
	if e != nil {
		return setupcontrol.Prepared{}, e
	}
	fields, e := documentMap(original)
	if e != nil {
		return setupcontrol.Prepared{}, setupcontrol.ErrInvalid
	}
	if _, e = currentapplication.DecodeConfigDocument(original); e != nil {
		return setupcontrol.Prepared{}, setupcontrol.ErrInvalid
	}
	var provided map[string]any
	if r.ConfigurationFile != "" {
		b, e := readPrivate(r.ConfigurationFile, 65536)
		if e != nil {
			return setupcontrol.Prepared{}, setupcontrol.ErrInvalid
		}
		provided, e = documentMap(b)
		if e != nil {
			return setupcontrol.Prepared{}, e
		}
		if coreconfig.VerifyPrivateFiles(ctx, []string{r.ConfigurationFile}) != nil {
			return setupcontrol.Prepared{}, setupcontrol.ErrInvalid
		}
		if _, e = currentapplication.DecodeConfigDocument(b); e != nil {
			return setupcontrol.Prepared{}, setupcontrol.ErrInvalid
		}
		if !sameDeployment(fields, provided) {
			return setupcontrol.Prepared{}, setupcontrol.ErrInvalid
		}
	}
	next, e := changeModules(fields, provided, r.Enabled)
	if e != nil {
		return setupcontrol.Prepared{}, e
	}
	configFile := filepath.Join(dir, "current-application.json")
	candidate := marshal(next)
	if e = privatefile.Write(configFile, candidate); e != nil {
		return setupcontrol.Prepared{}, e
	}
	if e = privatefile.Write(filepath.Join(dir, "original-current-application.json"), original); e != nil {
		return setupcontrol.Prepared{}, e
	}
	blocks := []string{}
	if e = d.protectRuntimeFile(ctx, configFile); e != nil {
		blocks = append(blocks, "候选文件的Docker私密权限未完成核对；原配置未改变")
	}
	if coreconfig.VerifyPrivateFiles(ctx, []string{configFile}) != nil {
		blocks = append(blocks, "候选配置未通过本机私密文件检查，原实例未改变")
	}
	mods, e := d.inspectModules(ctx, a, configFile)
	if e != nil {
		blocks = append(blocks, "候选配置的原生只读检测未完成；不会以启动业务进程替代检测")
	}
	blocks = append(blocks, candidatePermissionBlocks(mods, r.Enabled)...)
	definitions := map[string]currentapplication.InstallModule{}
	for _, m := range currentapplication.InstallationModules() {
		definitions[m.ID] = m
	}
	changes := []setupcontrol.Change{}
	ids := []string{}
	for id := range r.Enabled {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	services, ok := effective["services"].(map[string]any)
	if !ok {
		return setupcontrol.Prepared{}, setupcontrol.ErrUnavailable
	}
	app, ok := services["current-application"].(map[string]any)
	if !ok {
		return setupcontrol.Prepared{}, setupcontrol.ErrUnavailable
	}
	ui, ok := services["listingkit-ui"].(map[string]any)
	if !ok {
		return setupcontrol.Prepared{}, setupcontrol.ErrUnavailable
	}
	environment, ok := ui["environment"].(map[string]any)
	if !ok {
		return setupcontrol.Prepared{}, setupcontrol.ErrUnavailable
	}
	u, _ := service(cs, "listingkit-ui")
	app["image"] = a.Image
	ui["image"] = u.Image
	delete(app, "build")
	delete(ui, "build")
	if e = writeCompose(filepath.Join(dir, "original.compose.json"), effective); e != nil {
		return setupcontrol.Prepared{}, e
	}
	for _, id := range ids {
		want := r.Enabled[id]
		definition := definitions[id]
		previous := "未启用"
		current := fields[id]
		if current != nil && current != false {
			previous = "已配置"
		}
		after := "停用入口（保留数据）"
		if want {
			after = "启用入口（保留原权限）"
			verified := false
			for _, m := range mods {
				if m.ID == id && m.Enabled && m.Installed == "verified" {
					verified = true
				}
			}
			if !verified {
				blocks = append(blocks, definition.Label+"：缺少合格的schema/运行权限核对；须由原owner现有维护入口处理，不能重跑新装")
			}
		}
		changes = append(changes, setupcontrol.Change{Label: definition.Label, Before: previous, After: after})
		if definition.UIFlag != "" {
			environment[definition.UIFlag] = map[bool]string{true: "true", false: "false"}[want]
		}
	}
	target, e := configPath(a)
	if e != nil {
		return setupcontrol.Prepared{}, e
	}
	volumes, ok := app["volumes"].([]any)
	if !ok {
		return setupcontrol.Prepared{}, setupcontrol.ErrUnavailable
	}
	filtered := []any{}
	for _, raw := range volumes {
		m, ok := raw.(map[string]any)
		if !ok {
			return setupcontrol.Prepared{}, setupcontrol.ErrUnavailable
		}
		if m["target"] != target {
			filtered = append(filtered, m)
		}
	}
	app["volumes"] = append(filtered, map[string]any{"type": "bind", "source": strings.ReplaceAll(configFile, "$", "$$"), "target": target, "read_only": true, "bind": map[string]any{"create_host_path": false}})
	// Pin the CURRENT image identities, including a separately previewed UI.
	app["image"] = a.Image
	ui["image"] = u.Image
	delete(app, "build")
	delete(ui, "build")
	composeFile := filepath.Join(dir, "compose.json")
	if e = writeCompose(composeFile, effective); e != nil {
		return setupcontrol.Prepared{}, e
	}
	b, e := readPrivate(composeFile, 2*1024*1024)
	if e != nil {
		return setupcontrol.Prepared{}, e
	}
	n := nativePlan{Project: v.Project, Mode: r.Mode, ComposeFile: composeFile, ComposeHash: hash(b), ManifestFile: configFile, ManifestHash: hash(candidate), Expected: r.Enabled, RuntimeBindings: bindings(cs), Origin: v.Origin}
	return setupcontrol.Prepared{Request: r, InstanceID: r.InstanceID, Before: before, Candidate: hash(marshal(n)), Services: []string{"current-application", "listingkit-ui"}, Changes: changes, Blocks: blocks, Data: marshal(n)}, nil
}

// A private file reference cannot silently point a module at another identity
// installation or another database host. The local runtime namespace remains
// the boundary of this host-only installation tool.
func sameDeployment(original, provided map[string]any) bool {
	for _, key := range []string{"identity", "sourceAccountDatabase"} {
		if original[key] == nil || hash(marshal(original[key])) != hash(marshal(provided[key])) {
			return false
		}
	}
	base, ok := original["sourceAccountDatabase"].(map[string]any)
	if !ok {
		return false
	}
	var walk func(any) bool
	walk = func(v any) bool {
		switch m := v.(type) {
		case map[string]any:
			if _, db := m["password"]; db && m["database"] != nil && m["user"] != nil {
				if m["host"] != base["host"] || m["port"] != base["port"] {
					return false
				}
			}
			for _, value := range m {
				if !walk(value) {
					return false
				}
			}
		case []any:
			for _, value := range m {
				if !walk(value) {
					return false
				}
			}
		}
		return true
	}
	return walk(provided)
}
