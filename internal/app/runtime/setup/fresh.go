package setup

import (
	"context"
	"encoding/json"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"task-processor/internal/app/setupcontrol"
	"task-processor/internal/platform/privatefile"
)

func (d *Driver) freshFingerprint(ctx context.Context, r setupcontrol.Request) (string, error) {
	if e := validateNew(r, false); e != nil {
		return "", e
	}
	cs, e := d.containers(ctx, r.Project)
	if e != nil {
		return "", e
	}
	if len(cs) != 0 {
		return "", setupcontrol.ErrInvalid
	}
	for _, kind := range []string{"volume", "network"} {
		b, e := d.run(ctx, kind, "ls", "--format", "{{.Name}}")
		if e != nil {
			return "", e
		}
		for _, name := range strings.Fields(string(b)) {
			if name == r.Project || strings.HasPrefix(name, r.Project+"-") || strings.HasPrefix(name, r.Project+"_") {
				return "", setupcontrol.ErrInvalid
			}
		}
	}
	if e = freePorts(r); e != nil {
		return "", e
	}
	if e = d.checkSubnet(ctx, r.NetworkSubnet); e != nil {
		return "", e
	}
	id, e := d.identity(ctx, r.Project)
	if e != nil {
		return "", e
	}
	return hash(marshal([]any{id, r, d.release.SourceCommit})), nil
}
func (d *Driver) prepareFresh(ctx context.Context, r setupcontrol.Request, dir string) (setupcontrol.Prepared, error) {
	before, e := d.freshFingerprint(ctx, r)
	if e != nil {
		return setupcontrol.Prepared{}, e
	}
	id, e := d.identity(ctx, r.Project)
	if e != nil {
		return setupcontrol.Prepared{}, e
	}
	origin := "https://localhost:" + strconv.Itoa(r.ApplicationPort)
	files := []string{filepath.Join(d.source, "deployments/docker/account-compose/docker-compose.yml")}
	names := []string{}
	if r.Profile != "base" {
		names = append(names, "knowledge", "data-services", "modules", "ecoservices", "storage-capacity")
	}
	for _, name := range names {
		files = append(files, filepath.Join(d.source, "deployments/docker/account-compose/docker-compose."+name+".yml"))
	}
	blocks := []string{}
	pluginDirectory := r.ToolReleaseDirectory
	if r.Profile != "base" {
		b, e := readPrivate(filepath.Join(pluginDirectory, "release.json"), 8192)
		var release struct {
			SchemaVersion int    `json:"schemaVersion"`
			CaptureAppURL string `json:"captureAppUrl"`
			SHA256        string `json:"sha256"`
			Filename      string `json:"filename"`
		}
		if e != nil || json.Unmarshal(b, &release) != nil || release.SchemaVersion != 1 || release.CaptureAppURL != origin+"/capture/1688" || release.Filename != "shuomi-product-capture.zip" {
			return setupcontrol.Prepared{}, setupcontrol.ErrInvalid
		}
		zip, e := readPrivate(filepath.Join(pluginDirectory, release.Filename), 20*1024*1024)
		if e != nil || hash(zip) != release.SHA256 {
			return setupcontrol.Prepared{}, setupcontrol.ErrInvalid
		}
	}
	envFile := filepath.Join(dir, "install.env")
	env := "COMPOSE_PROJECT_NAME=" + r.Project + "\nACCOUNT_APPLICATION_PORT=" + strconv.Itoa(r.ApplicationPort) + "\nACCOUNT_IDENTITY_PORT=" + strconv.Itoa(r.IdentityPort) + "\nACCOUNT_MAIL_PORT=" + strconv.Itoa(r.MailPort) + "\n"
	if pluginDirectory != "" {
		if !filepath.IsAbs(pluginDirectory) || strings.ContainsAny(pluginDirectory, "\r\n'\"") {
			return setupcontrol.Prepared{}, setupcontrol.ErrInvalid
		}
		env += "ACCOUNT_TOOL_RELEASE_DIRECTORY='" + filepath.ToSlash(pluginDirectory) + "'\n"
	}
	if e = privatefile.Write(envFile, []byte(env)); e != nil {
		return setupcontrol.Prepared{}, e
	}
	args := append([]string{"compose", "--env-file", envFile}, composeArgs(r.Project, files)[1:]...)
	args = append(args, "config", "--format", "json")
	raw, e := d.run(ctx, args...)
	if e != nil {
		return setupcontrol.Prepared{}, e
	}
	var effective map[string]any
	if json.Unmarshal(raw, &effective) != nil {
		return setupcontrol.Prepared{}, setupcontrol.ErrUnavailable
	}
	if e = configureNetworks(effective, r.NetworkSubnet); e != nil {
		return setupcontrol.Prepared{}, e
	}
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
	app["image"] = d.release.BackendImage
	ui["image"] = d.release.UIImage
	delete(app, "build")
	delete(ui, "build")
	build := []string{}
	for name, v := range services {
		obj, ok := v.(map[string]any)
		if !ok {
			return setupcontrol.Prepared{}, setupcontrol.ErrUnavailable
		}
		if _, has := obj["build"]; has {
			if p, ok := obj["profiles"].([]any); !ok || len(p) == 0 {
				build = append(build, name)
			}
		}
	}
	sort.Strings(build)
	composeFile := filepath.Join(dir, "compose.json")
	if e = writeCompose(composeFile, effective); e != nil {
		return setupcontrol.Prepared{}, e
	}
	b, e := readPrivate(composeFile, 2*1024*1024)
	if e != nil {
		return setupcontrol.Prepared{}, e
	}
	handoff := filepath.Join(dir, "handoff")
	n := nativePlan{Project: r.Project, Mode: r.Mode, ComposeFile: composeFile, ComposeHash: hash(b), BuildServices: build, Origin: origin, Handoff: handoff, ReleaseFile: filepath.Join(filepath.Dir(d.source), "release.json")}
	n.RequiredServices = requiredServices(services)
	if r.Profile == "unified" {
		n.Expected = map[string]bool{"knowledge": true, "dataServices": true, "productCollections": true, "notificationCenterDatabase": true, "reportCenter": true, "toolMarket": true, "ecoservices": true, "projectCenter": true}
	}
	installationLabel := ""
	for _, profile := range profiles() {
		if profile.ID == r.Profile {
			installationLabel = profile.Label
		}
	}
	changes := []setupcontrol.Change{{Label: "安装名称", Before: "不存在", After: r.Project}, {Label: "安装哪些功能", Before: "未安装", After: installationLabel}, {Label: "安装版本", Before: "未安装", After: d.release.SourceCommit}, {Label: "应用入口", Before: "无", After: origin}, {Label: "私密登录资料", Before: "无", After: handoff}, {Label: "数据与外部服务", Before: "空项目", After: "建立新的空系统；不生成示例商品，不调用外部平台或AI"}}
	if r.NetworkSubnet != "" {
		changes = append(changes, setupcontrol.Change{Label: "网络地址范围", Before: "未分配", After: r.NetworkSubnet + "（仅用于本次安装，不更改其他Docker网络设置）"})
	}
	return setupcontrol.Prepared{Request: r, InstanceID: id, Before: before, Candidate: hash(marshal(n)), Services: []string{"登录、数据存储、应用及所选功能"}, Changes: changes, Blocks: blocks, Data: marshal(n)}, nil
}
func (d *Driver) handoff(ctx context.Context, project, dir string) error {
	// The original bootstrap owns these credentials. Copy by a fixed private
	// channel; neither the public DTO nor process arguments contains their bytes.
	for _, file := range []struct{ volume, name, out string }{{"trusted-ca", "root-ca.pem", "root-ca.pem"}, {"tofu-inputs", "operator-password", "operator-password.txt"}} {
		raw, e := d.run(ctx, "run", "--rm", "--network", "none", "--read-only", "--cap-drop", "ALL", "--security-opt", "no-new-privileges", "--mount", "type=volume,src="+project+"-"+file.volume+",dst=/source,readonly", "alpine:3.22", "cat", "/source/"+file.name)
		if e != nil || len(raw) == 0 || len(raw) > 16384 {
			return setupcontrol.ErrUnavailable
		}
		if e = privatefile.Write(filepath.Join(dir, file.out), raw); e != nil {
			return e
		}
	}
	return nil
}
