// Package setupcontrol coordinates explicit local installation operations.
// Domain installation and readiness remain with their original owners.
package setupcontrol

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	"task-processor/internal/platform/privatefile"
)

var (
	ErrBusy        = errors.New("INSTANCE_BUSY")
	ErrDrift       = errors.New("CONFIGURATION_CHANGED")
	ErrUnresolved  = errors.New("ORIGINAL_OPERATION_UNRESOLVED")
	ErrInvalid     = errors.New("INVALID_REQUEST")
	ErrUnavailable = errors.New("CHECK_UNAVAILABLE")
	ErrExpired     = errors.New("RECOVERY_PREVIEW_EXPIRED")
)

type Request struct {
	Mode                 string          `json:"mode"`
	InstanceID           string          `json:"instanceId,omitempty"`
	Project              string          `json:"project,omitempty"`
	Profile              string          `json:"profile,omitempty"`
	ApplicationPort      int             `json:"applicationPort,omitempty"`
	IdentityPort         int             `json:"identityPort,omitempty"`
	MailPort             int             `json:"mailPort,omitempty"`
	ConfigurationFile    string          `json:"configurationFile,omitempty"`
	ToolReleaseDirectory string          `json:"toolReleaseDirectory,omitempty"`
	NetworkSubnet        string          `json:"networkSubnet,omitempty"`
	Enabled              map[string]bool `json:"enabled,omitempty"`
}
type Change struct {
	Label  string `json:"label"`
	Before string `json:"before"`
	After  string `json:"after"`
}
type Prepared struct {
	Request    Request         `json:"request"`
	InstanceID string          `json:"instanceId"`
	Before     string          `json:"before"`
	Candidate  string          `json:"candidate"`
	Services   []string        `json:"services"`
	Changes    []Change        `json:"changes"`
	Blocks     []string        `json:"blocks"`
	Data       json.RawMessage `json:"data"`
}
type Plan struct {
	Name       string        `json:"name"`
	CreatedAt  time.Time     `json:"createdAt"`
	Revision   int           `json:"revision"`
	Archived   bool          `json:"archived"`
	Stage      string        `json:"stage"`
	Reason     string        `json:"reason"`
	Attempts   []AttemptView `json:"attempts"`
	ID         string        `json:"id"`
	InstanceID string        `json:"instanceId"`
	Mode       string        `json:"mode"`
	Status     string        `json:"status"`
	Changes    []Change      `json:"changes"`
	Blocks     []string      `json:"blocks"`
	Services   []string      `json:"services"`
	UpdatedAt  time.Time     `json:"updatedAt"`
}
type record struct {
	Version        int       `json:"version"`
	Plan           Plan      `json:"plan"`
	Prepared       Prepared  `json:"prepared"`
	Attempts       []attempt `json:"attempts,omitempty"`
	OriginalHash   string    `json:"originalHash,omitempty"`
	PreviousGuard  string    `json:"previousGuard,omitempty"`
	ArchivePreview string    `json:"archivePreview,omitempty"`
}
type Driver interface {
	Identity(context.Context, Request) (string, error)
	Prepare(context.Context, Request, string) (Prepared, error)
	Fingerprint(context.Context, Prepared) (string, error)
	Apply(context.Context, Prepared, Report) error
	Verify(context.Context, Prepared, Report) error
}
type Control struct {
	root   string
	driver Driver
}

func New(root string, driver Driver) *Control { return &Control{root: root, driver: driver} }
func digest(data []byte) string               { h := sha256.Sum256(data); return hex.EncodeToString(h[:]) }
func safeID(s string) bool {
	if len(s) < 1 || len(s) > 96 {
		return false
	}
	for _, c := range s {
		if !(c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' || c == '-') {
			return false
		}
	}
	return true
}
func (c *Control) directory(id string) (string, error) {
	parts := strings.Split(id, ".")
	if len(parts) != 2 || !safeID(parts[0]) || uuid.Validate(parts[1]) != nil {
		return "", ErrInvalid
	}
	p := filepath.Join(c.root, parts[0], "plans", parts[1])
	return p, privatefile.RejectPath(p)
}
func (c *Control) read(id string) (record, error) {
	dir, err := c.directory(id)
	if err != nil {
		return record{}, err
	}
	raw, err := os.ReadFile(filepath.Join(dir, "receipt.json"))
	if err != nil {
		return record{}, ErrUnavailable
	}
	var r record
	err = json.Unmarshal(raw, &r)
	if err != nil || (r.Version != 1 && r.Version != 2) || r.Plan.ID != id || r.Prepared.InstanceID != r.Plan.InstanceID || !strings.HasPrefix(id, r.Plan.InstanceID+".") {
		return record{}, ErrUnavailable
	}
	if r.Version == 2 && (len(r.Attempts) == 0 || r.OriginalHash != "" && !c.originalIntact(dir, r.OriginalHash)) {
		return record{}, ErrUnavailable
	}
	for i, a := range r.Attempts {
		if a.Number != i || !strings.HasPrefix(a.Key, id+":") || a.PayloadDigest != digest(mustJSON(a.Prepared)) || a.Prepared.InstanceID != r.Plan.InstanceID || !sameRetryInput(r.Prepared.Request, a.Prepared.Request) {
			return record{}, ErrUnavailable
		}
	}
	r.Plan.Name = r.Prepared.Request.Project
	if r.Plan.Name == "" {
		r.Plan.Name = r.Plan.InstanceID
	}
	if r.Plan.CreatedAt.IsZero() {
		r.Plan.CreatedAt = r.Plan.UpdatedAt
	}
	r.project()
	return r, nil
}
func (c *Control) save(r *record) error {
	dir, err := c.directory(r.Plan.ID)
	if err != nil {
		return err
	}
	r.Plan.UpdatedAt = time.Now().UTC()
	r.Plan.Revision++
	r.project()
	raw, err := json.Marshal(r)
	if err != nil {
		return err
	}
	return privatefile.Write(filepath.Join(dir, "receipt.json"), raw)
}
func (c *Control) active(instance string) (string, error) {
	p := filepath.Join(c.root, instance, "active.json")
	if privatefile.RejectPath(p) != nil {
		return "", ErrUnavailable
	}
	raw, err := os.ReadFile(p)
	if os.IsNotExist(err) {
		return "", nil
	}
	if err != nil {
		return "", ErrUnavailable
	}
	var id string
	if json.Unmarshal(raw, &id) != nil {
		return "", ErrUnavailable
	}
	return id, nil
}
func (c *Control) Plan(ctx context.Context, req Request) (Plan, error) {
	instance, err := c.driver.Identity(ctx, req)
	if err != nil || !safeID(instance) {
		return Plan{}, ErrInvalid
	}
	lock, err := lockInstance(filepath.Join(c.root, instance))
	if err != nil {
		return Plan{}, err
	}
	defer lock.Close()
	active, err := c.active(instance)
	if err != nil {
		return Plan{}, err
	}
	if !c.guardEnded(active) {
		return Plan{}, ErrUnresolved
	}
	id := instance + "." + uuid.NewString()
	dir, _ := c.directory(id)
	if err = os.MkdirAll(dir, 0700); err != nil {
		return Plan{}, err
	}
	prepared, err := c.driver.Prepare(ctx, req, dir)
	if err != nil {
		return Plan{}, err
	}
	if prepared.InstanceID != instance {
		return Plan{}, ErrDrift
	}
	r := record{Version: 2, Plan: Plan{ID: id, InstanceID: instance, Name: req.Project, CreatedAt: time.Now().UTC(), Mode: req.Mode, Status: "prepared", Changes: prepared.Changes, Blocks: prepared.Blocks, Services: prepared.Services}, Prepared: prepared, PreviousGuard: active, Attempts: []attempt{{AttemptView: AttemptView{Number: 0, Status: "prepared", Stage: "check"}, Key: id + ":0", Prepared: prepared}}}
	r.Attempts[0].PayloadDigest = digest(mustJSON(prepared))
	if err = c.save(&r); err != nil {
		return Plan{}, err
	}
	return r.Plan, nil
}
func (c *Control) Read(id string) (Plan, error) { r, err := c.read(id); return r.Plan, err }
func (c *Control) Pending(instance string) (*Plan, error) {
	if !safeID(instance) {
		return nil, ErrInvalid
	}
	id, err := c.active(instance)
	if err != nil || id == "" {
		return nil, err
	}
	p, err := c.Read(id)
	if err == nil && (p.Archived || p.Status == "succeeded") {
		return nil, nil
	}
	return &p, err
}
func (c *Control) Outstanding() ([]Plan, error) {
	entries, err := os.ReadDir(c.root)
	if os.IsNotExist(err) {
		return []Plan{}, nil
	}
	if err != nil {
		return nil, ErrUnavailable
	}
	plans := []Plan{}
	for _, entry := range entries {
		if !entry.IsDir() || entry.Name() == "controller" || !safeID(entry.Name()) {
			continue
		}
		p, e := c.Pending(entry.Name())
		if e != nil {
			return nil, e
		}
		if p != nil {
			plans = append(plans, *p)
		}
	}
	return plans, nil
}
func (c *Control) Apply(ctx context.Context, id string) (Plan, error) {
	return c.withOperation(id, func(r *record) (Plan, error) {
		if r.Plan.Status == "succeeded" {
			return r.Plan, nil
		}
		if r.Version != 2 || r.Plan.Archived || r.Plan.Status != "prepared" {
			return r.Plan, ErrUnresolved
		}
		active, err := c.active(r.Plan.InstanceID)
		if err != nil {
			return r.Plan, err
		}
		if active != r.PreviousGuard || !c.guardEnded(active) {
			return r.Plan, ErrUnresolved
		}
		if len(r.Plan.Blocks) > 0 {
			return r.Plan, ErrInvalid
		}
		before, err := c.driver.Fingerprint(ctx, r.Prepared)
		if err != nil {
			return r.Plan, err
		}
		if before != r.Prepared.Before {
			return r.Plan, ErrDrift
		}
		r.Plan.Status = "running"
		r.Attempts[0].Status = "running"
		if err = c.save(r); err != nil {
			return r.Plan, err
		}
		if err = c.bindGuard(r, active); err != nil {
			return r.Plan, err
		}
		return c.execute(ctx, r)
	})
}
func (c *Control) Reconcile(ctx context.Context, id string) (Plan, error) {
	return c.withOperation(id, func(r *record) (Plan, error) {
		if r.Plan.Status == "succeeded" || r.Plan.Archived {
			return r.Plan, nil
		}
		if r.Plan.Status == "failed" {
			return r.Plan, ErrUnresolved
		}
		if r.Plan.Status != "unknown" && r.Plan.Status != "running" {
			return r.Plan, ErrInvalid
		}
		active, err := c.active(r.Plan.InstanceID)
		if err != nil {
			return r.Plan, err
		}
		if active != id {
			if r.Version != 2 || active != r.PreviousGuard || !c.guardEnded(active) {
				return r.Plan, ErrUnresolved
			}
			if err = c.bindGuard(r, active); err != nil {
				return r.Plan, err
			}
		}
		if err = c.driver.Verify(ctx, r.current(), func(stage string) error {
			if _, ok := stepNames[stage]; !ok {
				return ErrInvalid
			}
			return nil
		}); err != nil {
			if parkInterrupted(r) {
				if e := c.save(r); e != nil {
					return r.Plan, e
				}
			}
			return r.Plan, ErrUnresolved
		}
		// A failed observation of a v1 operation must remain read-only. Only a
		// verified completion (or explicit recovery confirmation) may adopt it.
		if err = c.adopt(r); err != nil {
			return r.Plan, err
		}
		return c.finish(r, nil)
	})
}
