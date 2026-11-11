package setupcontrol

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/google/uuid"
	"task-processor/internal/platform/privatefile"
)

// Report must persist the step intent before the driver dispatches its command.
type Report func(string) error

// CommandError retains only a closed category and numeric exit status.
type CommandError struct {
	Code     string
	ExitCode int
}

func (e *CommandError) Error() string { return "INSTALL_COMMAND_UNAVAILABLE" }
func (e *CommandError) Unwrap() error { return ErrUnavailable }

type StepResult struct {
	Stage      string    `json:"stage"`
	Status     string    `json:"status"`
	StartedAt  time.Time `json:"startedAt"`
	FinishedAt time.Time `json:"finishedAt,omitempty"`
	Code       string    `json:"code,omitempty"`
	ExitCode   int       `json:"exitCode,omitempty"`
}

var stepNames = map[string]string{"check": "检查安装条件", "download": "准备安装文件", "build": "准备安装组件", "start": "启动安装服务", "verify": "检查运行结果", "handoff": "提供登录资料"}

type AttemptView struct {
	Number    int          `json:"number"`
	Status    string       `json:"status"`
	Stage     string       `json:"stage"`
	Reason    string       `json:"reason"`
	UpdatedAt time.Time    `json:"updatedAt"`
	Steps     []StepResult `json:"steps,omitempty"`
}
type attempt struct {
	AttemptView
	Key           string   `json:"key"`
	PreviewID     string   `json:"previewId,omitempty"`
	PayloadDigest string   `json:"payloadDigest"`
	Prepared      Prepared `json:"prepared"`
}
type RecoveryAudit struct {
	Fingerprint    string
	Removed, Empty bool
	Changes        []Change
	Blocks         []string
}
type RecoveryDriver interface {
	AuditRecovery(context.Context, Prepared, bool) (RecoveryAudit, error)
	PrepareRetry(context.Context, Prepared, string, string) (Prepared, error)
}
type RecoveryPreview struct {
	ID          string    `json:"id"`
	OperationID string    `json:"operationId"`
	Action      string    `json:"action"`
	Changes     []Change  `json:"changes"`
	Blocks      []string  `json:"blocks"`
	ExpiresAt   time.Time `json:"expiresAt"`
}
type sealedRecovery struct {
	Preview        RecoveryPreview `json:"preview"`
	Revision       int             `json:"revision"`
	OriginalDigest string          `json:"originalDigest"`
	Fingerprint    string          `json:"fingerprint"`
	Prepared       Prepared        `json:"prepared"`
}

func (r *record) current() Prepared {
	if len(r.Attempts) > 0 {
		return r.Attempts[len(r.Attempts)-1].Prepared
	}
	return r.Prepared
}
func parkInterrupted(r *record) bool {
	if len(r.Attempts) == 0 {
		return false
	}
	a := &r.Attempts[len(r.Attempts)-1]
	if a.Status != "running" {
		return false
	}
	a.Status = "unknown"
	if a.Reason == "" {
		a.Reason = "上次执行已中断，结果尚未确认"
	}
	if len(a.Steps) > 0 && a.Steps[len(a.Steps)-1].Status == "running" {
		a.Steps[len(a.Steps)-1].Status = "unknown"
	}
	return true
}
func (r *record) project() {
	if len(r.Attempts) == 0 {
		return
	}
	r.Plan.Attempts = make([]AttemptView, len(r.Attempts))
	for i := range r.Attempts {
		r.Plan.Attempts[i] = r.Attempts[i].AttemptView
	}
	last := r.Attempts[len(r.Attempts)-1]
	r.Plan.Status, r.Plan.Stage, r.Plan.Reason = last.Status, last.Stage, last.Reason
}
func (c *Control) originalIntact(dir, want string) bool {
	p := filepath.Join(dir, "original-receipt.json")
	if privatefile.RejectPath(p) != nil {
		return false
	}
	b, err := os.ReadFile(p)
	return err == nil && digest(b) == want
}
func (c *Control) adopt(r *record) error {
	if r.Version == 2 {
		return nil
	}
	dir, err := c.directory(r.Plan.ID)
	if err != nil {
		return err
	}
	b, err := os.ReadFile(filepath.Join(dir, "receipt.json"))
	if err != nil {
		return ErrUnavailable
	}
	p := filepath.Join(dir, "original-receipt.json")
	if privatefile.RejectPath(p) != nil {
		return ErrInvalid
	}
	old, err := os.ReadFile(p)
	if err == nil && digest(old) != digest(b) {
		return ErrDrift
	}
	if err != nil && !os.IsNotExist(err) {
		return ErrUnavailable
	}
	if os.IsNotExist(err) {
		if err = privatefile.Write(p, b); err != nil {
			return err
		}
	}
	if !c.originalIntact(dir, digest(b)) {
		return ErrUnavailable
	}
	r.Version, r.OriginalHash = 2, digest(b)
	r.Attempts = []attempt{{AttemptView: AttemptView{Number: 0, Status: r.Plan.Status, Reason: "当时未记录失败步骤和原因", UpdatedAt: r.Plan.UpdatedAt}, Key: r.Plan.ID + ":0", Prepared: r.Prepared, PayloadDigest: digest(mustJSON(r.Prepared))}}
	return c.save(r)
}
func mustJSON(v any) []byte { b, _ := json.Marshal(v); return b }
func sameRetryInput(original, next Request) bool {
	original.NetworkSubnet = next.NetworkSubnet
	return digest(mustJSON(original)) == digest(mustJSON(next))
}
func retryAuditAllowed(r *record, a RecoveryAudit) bool {
	if !a.Empty {
		return false
	}
	if !a.Removed {
		return true
	}
	return r.Plan.Status == "failed" && (r.Plan.Stage == "check" || r.Plan.Stage == "download" || r.Plan.Stage == "build")
}
func (c *Control) withOperation(id string, fn func(*record) (Plan, error)) (Plan, error) {
	r, err := c.read(id)
	if err != nil {
		return Plan{}, err
	}
	global, err := lockInstance(filepath.Join(c.root, "controller"))
	if err != nil {
		return r.Plan, err
	}
	defer global.Close()
	lock, err := lockInstance(filepath.Join(c.root, r.Plan.InstanceID))
	if err != nil {
		return r.Plan, err
	}
	defer lock.Close()
	r, err = c.read(id)
	if err != nil {
		return Plan{}, err
	}
	return fn(&r)
}
func (c *Control) guardEnded(id string) bool {
	if id == "" {
		return true
	}
	r, err := c.read(id)
	return err == nil && (r.Plan.Archived || r.Plan.Status == "succeeded")
}
func (c *Control) bindGuard(r *record, previous string) error {
	active, err := c.active(r.Plan.InstanceID)
	if err != nil || active != previous {
		return ErrDrift
	}
	return privatefile.Write(filepath.Join(c.root, r.Plan.InstanceID, "active.json"), mustJSON(r.Plan.ID))
}
func (c *Control) report(r *record) Report {
	return func(stage string) error {
		if _, ok := stepNames[stage]; !ok {
			return ErrInvalid
		}
		last := &r.Attempts[len(r.Attempts)-1]
		now := time.Now().UTC()
		if len(last.Steps) > 0 {
			previous := &last.Steps[len(last.Steps)-1]
			if previous.Status == "running" {
				previous.Status = "succeeded"
				previous.FinishedAt = now
			}
		}
		last.Steps = append(last.Steps, StepResult{Stage: stage, Status: "running", StartedAt: now})
		last.Stage, last.Reason, last.UpdatedAt = stage, "", time.Now().UTC()
		return c.save(r)
	}
}
func (c *Control) execute(ctx context.Context, r *record) (Plan, error) {
	report := c.report(r)
	err := c.driver.Apply(ctx, r.current(), report)
	if err == nil {
		err = c.driver.Verify(ctx, r.current(), report)
	}
	return c.finish(r, err)
}
func (c *Control) finish(r *record, err error) (Plan, error) {
	last := &r.Attempts[len(r.Attempts)-1]
	last.Status, last.Reason = "succeeded", ""
	if err != nil {
		last.Status = "unknown"
		if last.Stage == "check" || last.Stage == "download" || last.Stage == "build" {
			last.Status = "failed"
		}
		name := stepNames[last.Stage]
		if name == "" {
			name = "安装操作"
		}
		code, detail, exit := failureDetail(err)
		last.Reason = name + "未完成；" + detail + "。请检查原操作。"
		if len(last.Steps) > 0 {
			step := &last.Steps[len(last.Steps)-1]
			step.Code, step.ExitCode = code, exit
		}
	}
	last.UpdatedAt = time.Now().UTC()
	if len(last.Steps) > 0 {
		step := &last.Steps[len(last.Steps)-1]
		if step.Status == "running" {
			step.Status = last.Status
			step.FinishedAt = last.UpdatedAt
		}
	}
	if e := c.save(r); e != nil {
		return r.Plan, e
	}
	if err != nil {
		return r.Plan, ErrUnresolved
	}
	return r.Plan, nil
}
func failureDetail(err error) (string, string, int) {
	var command *CommandError
	if errors.As(err, &command) {
		messages := map[string]string{"timeout": "等待超时", "interrupted": "命令已中断", "command_failed": "本机安装命令未完成", "command_unavailable": "无法启动本机安装命令", "output_limit": "检查信息超过安全读取限制"}
		if text, ok := messages[command.Code]; ok {
			return command.Code, text, command.ExitCode
		}
	}
	if errors.Is(err, ErrDrift) {
		return "changed", "安装条件已发生变化", 0
	}
	if errors.Is(err, ErrInvalid) {
		return "invalid", "设置未通过检查", 0
	}
	if errors.Is(err, ErrUnavailable) {
		return "unavailable", "所需安装条件尚未就绪", 0
	}
	return "unknown", "具体原因尚未定位", 0
}
func (c *Control) History() ([]Plan, error) {
	entries, err := os.ReadDir(c.root)
	if os.IsNotExist(err) {
		return []Plan{}, nil
	}
	if err != nil {
		return nil, ErrUnavailable
	}
	result := []Plan{}
	for _, entry := range entries {
		if !entry.IsDir() || entry.Name() == "controller" || !safeID(entry.Name()) {
			continue
		}
		active, err := c.active(entry.Name())
		if err != nil {
			return nil, err
		}
		dirs, err := os.ReadDir(filepath.Join(c.root, entry.Name(), "plans"))
		if os.IsNotExist(err) {
			continue
		}
		if err != nil || len(dirs) > 1024 {
			return nil, ErrUnavailable
		}
		for _, dir := range dirs {
			if !dir.IsDir() || uuid.Validate(dir.Name()) != nil {
				continue
			}
			p, err := c.Read(entry.Name() + "." + dir.Name())
			if err != nil {
				return nil, err
			}
			if p.Status == "prepared" || p.ID == active && !p.Archived && p.Status != "succeeded" {
				continue
			}
			result = append(result, p)
		}
	}
	sort.Slice(result, func(i, j int) bool { return result[i].UpdatedAt.After(result[j].UpdatedAt) })
	if len(result) > 50 {
		result = result[:50]
	}
	return result, nil
}
func (c *Control) previewPath(id, preview string) (string, error) {
	if uuid.Validate(preview) != nil {
		return "", ErrInvalid
	}
	dir, err := c.directory(id)
	if err != nil {
		return "", err
	}
	p := filepath.Join(dir, "recovery-previews", preview+".json")
	return p, privatefile.RejectPath(p)
}
func (c *Control) PrepareRecovery(ctx context.Context, id, action, subnet string) (RecoveryPreview, error) {
	var result RecoveryPreview
	_, err := c.withOperation(id, func(r *record) (Plan, error) {
		if (action != "archive" && action != "retry") || action == "archive" && subnet != "" || r.Plan.Mode != "fresh" || r.Plan.Archived || r.Plan.Status == "prepared" || r.Plan.Status == "succeeded" {
			return r.Plan, ErrInvalid
		}
		active, err := c.active(r.Plan.InstanceID)
		if err != nil || active != id {
			return r.Plan, ErrUnresolved
		}
		driver, ok := c.driver.(RecoveryDriver)
		if !ok {
			return r.Plan, ErrUnavailable
		}
		audit, err := driver.AuditRecovery(ctx, r.current(), action == "retry")
		if err != nil {
			return r.Plan, err
		}
		result = RecoveryPreview{ID: uuid.NewString(), OperationID: id, Action: action, Changes: audit.Changes, Blocks: append([]string{}, audit.Blocks...), ExpiresAt: time.Now().UTC().Add(5 * time.Minute)}
		if action == "archive" && !audit.Removed {
			result.Blocks = append(result.Blocks, "仍有相关资源或无法确认已清理，不能归档")
		}
		if action == "retry" && !retryAuditAllowed(r, audit) {
			result.Blocks = append(result.Blocks, "尚未证明全部卷为空且未初始化，不能重新尝试")
		}
		path, err := c.previewPath(id, result.ID)
		if err != nil {
			return r.Plan, err
		}
		prepared := r.current()
		if action == "retry" && len(result.Blocks) == 0 {
			prepared, err = driver.PrepareRetry(ctx, r.current(), subnet, filepath.Join(filepath.Dir(path), result.ID))
			if err != nil {
				return r.Plan, err
			}
			if prepared.InstanceID != r.Plan.InstanceID || prepared.Before != audit.Fingerprint || !sameRetryInput(r.Prepared.Request, prepared.Request) {
				return r.Plan, ErrDrift
			}
			result.Changes = append(result.Changes, prepared.Changes...)
			result.Blocks = append(result.Blocks, prepared.Blocks...)
		}
		dir, _ := c.directory(id)
		raw, err := os.ReadFile(filepath.Join(dir, "receipt.json"))
		if err != nil {
			return r.Plan, ErrUnavailable
		}
		sealed := sealedRecovery{Preview: result, Revision: r.Plan.Revision, OriginalDigest: digest(raw), Fingerprint: audit.Fingerprint, Prepared: prepared}
		return r.Plan, privatefile.Write(path, mustJSON(sealed))
	})
	return result, err
}
func (c *Control) ConfirmRecovery(ctx context.Context, id, preview string) (Plan, error) {
	return c.withOperation(id, func(r *record) (Plan, error) {
		path, err := c.previewPath(id, preview)
		if err != nil {
			return r.Plan, err
		}
		raw, err := os.ReadFile(path)
		if err != nil || len(raw) > 2*1024*1024 {
			return r.Plan, ErrInvalid
		}
		var seal sealedRecovery
		if json.Unmarshal(raw, &seal) != nil || seal.Preview.ID != preview || seal.Preview.OperationID != id {
			return r.Plan, ErrInvalid
		}
		if r.ArchivePreview == preview {
			return r.Plan, nil
		}
		for _, a := range r.Attempts {
			if a.PreviewID == preview {
				return r.Plan, nil
			}
		}
		if time.Now().After(seal.Preview.ExpiresAt) {
			return r.Plan, ErrExpired
		}
		if len(seal.Preview.Blocks) > 0 || (seal.Preview.Action != "archive" && seal.Preview.Action != "retry") || r.Plan.Archived || r.Plan.Status == "succeeded" {
			return r.Plan, ErrInvalid
		}
		active, err := c.active(r.Plan.InstanceID)
		if err != nil || active != id {
			return r.Plan, ErrUnresolved
		}
		dir, _ := c.directory(id)
		before, err := os.ReadFile(filepath.Join(dir, "receipt.json"))
		if err != nil || digest(before) != seal.OriginalDigest || r.Plan.Revision != seal.Revision {
			return r.Plan, ErrDrift
		}
		driver, ok := c.driver.(RecoveryDriver)
		if !ok {
			return r.Plan, ErrUnavailable
		}
		audit, err := driver.AuditRecovery(ctx, r.current(), seal.Preview.Action == "retry")
		if err != nil {
			return r.Plan, err
		}
		if audit.Fingerprint != seal.Fingerprint || len(audit.Blocks) > 0 || seal.Preview.Action == "archive" && !audit.Removed || seal.Preview.Action == "retry" && !retryAuditAllowed(r, audit) {
			return r.Plan, ErrDrift
		}
		if err = c.adopt(r); err != nil {
			return r.Plan, err
		}
		parkInterrupted(r)
		if seal.Preview.Action == "archive" {
			r.Plan.Archived, r.ArchivePreview = true, preview
			if err = c.save(r); err != nil {
				return r.Plan, err
			}
			return r.Plan, nil
		}
		number := len(r.Attempts)
		r.Attempts = append(r.Attempts, attempt{AttemptView: AttemptView{Number: number, Status: "running", Stage: "check", UpdatedAt: time.Now().UTC()}, Key: id + ":" + preview, PreviewID: preview, PayloadDigest: digest(mustJSON(seal.Prepared)), Prepared: seal.Prepared})
		r.Plan.Changes, r.Plan.Services = seal.Prepared.Changes, seal.Prepared.Services
		if err = c.save(r); err != nil {
			return r.Plan, err
		}
		return c.execute(ctx, r)
	})
}
