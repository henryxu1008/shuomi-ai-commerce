package skillsandbox

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"io"
	"net/http"
	"strings"
	"time"
	"unicode/utf8"

	osb "github.com/alibaba/OpenSandbox/sdks/sandbox/go"
	"github.com/google/uuid"
	strict "sigs.k8s.io/json"
	"task-processor/internal/agent"
	"task-processor/internal/skill"
)

// Linux O_NOFOLLOW/O_NONBLOCK + fstat bind the read to one ordinary file.
// The trusted collector uses isolated stdlib, no user parameters or imports,
// and copies bounded bytes to a root-owned directory before SDK download.
const collectResult = `import os,stat
fd=os.open('/workspace/output/result.json',os.O_RDONLY|os.O_NOFOLLOW|os.O_NONBLOCK)
try:
 assert stat.S_ISREG(os.fstat(fd).st_mode)
 with os.fdopen(fd,'rb',closefd=False) as src:
  data=src.read(8193)
 assert len(data)<=8192
 with open('/workspace/collected/result.json','xb') as dst: dst.write(data)
finally: os.close(fd)
`

func (e *Executor) Execute(parent context.Context, task agent.ScriptTask) (result agent.ScriptResult, err error) {
	result.Receipt = agent.ScriptReceipt{Outcome: "not_dispatched", Cleanup: "not_needed", Reason: "profile_unavailable", InputDigest: agent.ScriptDigest(task.Input), DescriptorDigest: task.Descriptor.Digest()}
	if e == nil || task.Descriptor != e.descriptor || !task.Descriptor.Valid() || !agent.ValidID(task.RunID) || !agent.ValidID(task.CallID) || len(task.Input) > 16<<10 {
		return result, nil
	}
	files := make([]skill.File, 0, len(task.Files))
	for _, f := range task.Files {
		files = append(files, skill.File{Path: f.Path, Text: f.Text})
	}
	pkg, validationErr := skill.ValidatePackage(files)
	if validationErr != nil || pkg.Execution == nil || pkg.Execution.Entrypoint != task.Entrypoint || pkg.Digest != task.PackageDigest {
		result.Receipt.Reason = "invalid_package"
		return result, nil
	}
	ctx, cancel := context.WithTimeout(parent, 30*time.Second)
	defer cancel()
	select {
	case e.gate <- struct{}{}:
		defer func() { <-e.gate }()
	case <-ctx.Done():
		result.Receipt.Reason = "capacity_deadline"
		return result, nil
	}
	if time.Now().Before(e.unknownUntil) {
		result.Receipt.Reason = "creation_pending"
		return result, nil
	}
	listed, listErr := e.lifecycle.ListSandboxes(ctx, osb.ListOptions{Page: 1, PageSize: 3})
	if listErr != nil || listed == nil || listed.Pagination.TotalItems >= 2 || listed.Pagination.HasNextPage || len(listed.Items) >= 2 {
		result.Receipt.Reason = "capacity_unavailable"
		return result, nil
	}
	if deadline, ok := ctx.Deadline(); !ok || time.Until(deadline) < 10*time.Second {
		result.Receipt.Reason = "deadline"
		return result, nil
	}
	ttl := 120
	var nonce [32]byte
	if _, randomErr := rand.Read(nonce[:]); randomErr != nil {
		result.Receipt.Reason = "credential_unavailable"
		return result, nil
	}
	execdToken := hex.EncodeToString(nonce[:])
	result.Receipt.Outcome = "outcome_unknown"
	result.Receipt.Cleanup = "pending"
	result.Receipt.Reason = "create_unconfirmed"
	sandbox, createErr := e.lifecycle.CreateSandbox(ctx, osb.CreateSandboxRequest{Image: &osb.ImageSpec{URI: e.descriptor.ImageDigest}, Timeout: &ttl, Env: map[string]string{"EXECD_ACCESS_TOKEN": execdToken}, ResourceLimits: osb.ResourceLimits{"cpu": "1", "memory": "256Mi", "disk": "128Mi"}, Entrypoint: []string{"tail", "-f", "/dev/null"}, Metadata: map[string]string{"skill_profile": e.descriptor.Digest()}})
	if createErr != nil || sandbox == nil {
		e.unknownUntil = time.Now().Add(120 * time.Second)
		return result, nil
	}
	id, parseErr := uuid.Parse(sandbox.ID)
	if parseErr != nil || id == uuid.Nil || id.String() != sandbox.ID {
		e.unknownUntil = time.Now().Add(120 * time.Second)
		return result, nil
	}
	var commandID string
	execdURL := e.origin + "/v1/sandboxes/" + sandbox.ID + "/proxy/44772"
	execd := osb.NewExecdClient(execdURL, execdToken, osb.WithHTTPClient(e.client), osb.WithHeaders(e.headers))
	defer func() {
		cleanup, stop := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
		defer stop()
		if commandID != "" {
			_ = execd.InterruptCommand(cleanup, commandID)
		}
		if e.lifecycle.DeleteSandbox(cleanup, sandbox.ID) != nil {
			return
		}
		_, readErr := e.lifecycle.GetSandbox(cleanup, sandbox.ID)
		var api *osb.APIError
		if errors.As(readErr, &api) && api.StatusCode == 404 {
			result.Receipt.Cleanup = "completed"
		}
	}()
	// Lifecycle readback ties private input to the returned instance and exact
	// immutable image/profile. Runtime/firewall enforcement stays deployment-owned.
	info, infoErr := e.lifecycle.GetSandbox(ctx, sandbox.ID)
	if infoErr != nil || info.ID != sandbox.ID || info.Image == nil || info.Image.URI != e.descriptor.ImageDigest || info.Metadata["skill_profile"] != e.descriptor.Digest() {
		result.Receipt.Reason = "instance_mismatch"
		return result, nil
	}
	for i := 0; i < 30; i++ {
		if execd.Ping(ctx) == nil {
			break
		}
		select {
		case <-ctx.Done():
			return result, nil
		case <-time.After(100 * time.Millisecond):
		}
	}
	// A healthy but unprotected daemon is not an admitted execution target.
	anonymous := osb.NewExecdClient(execdURL, "", osb.WithHTTPClient(e.client), osb.WithHeaders(e.headers))
	_, unauthenticatedErr := anonymous.GetFileInfo(ctx, "/")
	var denied *osb.APIError
	if !errors.As(unauthenticatedErr, &denied) || denied.StatusCode != http.StatusUnauthorized {
		result.Receipt.Reason = "execd_auth_unavailable"
		return result, nil
	}
	for _, dir := range []string{"/workspace", "/workspace/input", "/workspace/scripts", "/workspace/output", "/workspace/collected"} {
		if execd.CreateDirectory(ctx, dir, 755) != nil {
			result.Receipt.Reason = "setup_failed"
			return result, nil
		}
	}
	if execd.SetPermissions(ctx, osb.PermissionsRequest{"/workspace/output": osb.Permission{Owner: "nobody", Group: "nogroup", Mode: 755}, "/workspace/collected": osb.Permission{Owner: "root", Group: "root", Mode: 700}}) != nil {
		result.Receipt.Reason = "setup_failed"
		return result, nil
	}
	for _, f := range pkg.Files {
		if !strings.HasPrefix(f.Path, "scripts/") {
			continue
		}
		if execd.UploadFile(ctx, strings.NewReader(f.Text), osb.UploadFileOptions{Metadata: osb.FileMetadata{Path: "/workspace/" + f.Path, Mode: 444}}) != nil {
			result.Receipt.Reason = "upload_failed"
			return result, nil
		}
	}
	if execd.UploadFile(ctx, bytes.NewReader(task.Input), osb.UploadFileOptions{Metadata: osb.FileMetadata{Path: "/workspace/input/input.json", Mode: 444}}) != nil {
		result.Receipt.Reason = "upload_failed"
		return result, nil
	}
	uid := int32(65534)
	command, cancelCommand := context.WithTimeout(ctx, 10*time.Second)
	var observed observation
	runErr := execd.RunCommand(command, osb.RunCommandRequest{Argv: []string{"/usr/local/bin/python3", "-I", "-S", "/workspace/" + task.Entrypoint}, Cwd: "/workspace", UID: &uid, GID: &uid, Timeout: 10000, Envs: workloadEnvironment()}, observed.observe)
	cancelCommand()
	commandID = observed.id
	result.Receipt.Outcome = observed.outcome(runErr)
	result.Receipt.Reason = "execution_unconfirmed"
	if result.Receipt.Outcome == "failed" {
		result.Receipt.Reason = "script_failed"
		return result, nil
	}
	if result.Receipt.Outcome != "succeeded" {
		return result, nil
	}
	// Script exit alone is not success. The collector/result/schema must also
	// complete; an ambiguous collection never becomes a successful title.
	result.Receipt.Outcome = "outcome_unknown"
	result.Receipt.Reason = "result_unconfirmed"
	root := int32(0)
	var collected observation
	collectErr := execd.RunCommand(ctx, osb.RunCommandRequest{Argv: []string{"/usr/local/bin/python3", "-I", "-S", "-c", collectResult}, Cwd: "/", UID: &root, GID: &root, Timeout: 2000, Envs: workloadEnvironment()}, collected.observe)
	if collected.outcome(collectErr) != "succeeded" {
		return result, nil
	}
	data, dataErr := execd.DownloadFile(ctx, "/workspace/collected/result.json", "")
	if dataErr != nil {
		return result, nil
	}
	raw, readErr := io.ReadAll(data)
	_ = data.Close()
	if readErr != nil {
		return result, nil
	}
	var output struct {
		SchemaVersion string   `json:"schemaVersion"`
		Title         string   `json:"title"`
		Diagnostics   []string `json:"diagnostics"`
	}
	violations, decodeErr := strict.UnmarshalStrict(raw, &output)
	if decodeErr != nil || len(violations) != 0 || !utf8.Valid(raw) || output.SchemaVersion != "title-result-v1" || output.Title == "" || !utf8.ValidString(output.Title) || output.Diagnostics == nil || len(output.Diagnostics) > 8 {
		result.Receipt.Reason = "invalid_result"
		result.Receipt.Outcome = "failed"
		return result, nil
	}
	for _, message := range output.Diagnostics {
		if len(message) > 256 || !utf8.ValidString(message) {
			result.Receipt.Reason = "invalid_result"
			result.Receipt.Outcome = "failed"
			return result, nil
		}
	}
	result.Title = output.Title
	result.Receipt.OutputDigest = agent.ScriptDigest(raw)
	result.Receipt.Outcome = "succeeded"
	result.Receipt.Reason = "completed"
	return result, nil
}

// The official command env overlay is applied before setuid/execve. Do not
// launch an env-clearing wrapper with a secret briefly inherited by that UID.
func workloadEnvironment() map[string]string {
	return map[string]string{"EXECD_ACCESS_TOKEN": "", "EXECD_ENVS": "", "JUPYTER_TOKEN": ""}
}
