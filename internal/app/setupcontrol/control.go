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
	ID         string    `json:"id"`
	InstanceID string    `json:"instanceId"`
	Mode       string    `json:"mode"`
	Status     string    `json:"status"`
	Changes    []Change  `json:"changes"`
	Blocks     []string  `json:"blocks"`
	Services   []string  `json:"services"`
	UpdatedAt  time.Time `json:"updatedAt"`
}
type record struct {
	Version  int      `json:"version"`
	Plan     Plan     `json:"plan"`
	Prepared Prepared `json:"prepared"`
}
type Driver interface {
	Identity(context.Context, Request) (string, error)
	Prepare(context.Context, Request, string) (Prepared, error)
	Fingerprint(context.Context, Prepared) (string, error)
	Apply(context.Context, Prepared) error
	Verify(context.Context, Prepared) error
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
	if err != nil || r.Version != 1 || r.Plan.ID != id || r.Prepared.InstanceID != r.Plan.InstanceID || !strings.HasPrefix(id, r.Plan.InstanceID+".") {
		return record{}, ErrUnavailable
	}
	return r, nil
}
func (c *Control) save(r record) error {
	dir, err := c.directory(r.Plan.ID)
	if err != nil {
		return err
	}
	r.Plan.UpdatedAt = time.Now().UTC()
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
func (c *Control) clear(instance string) error {
	return os.Remove(filepath.Join(c.root, instance, "active.json"))
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
	if active != "" {
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
	r := record{Version: 1, Plan: Plan{ID: id, InstanceID: instance, Mode: req.Mode, Status: "prepared", Changes: prepared.Changes, Blocks: prepared.Blocks, Services: prepared.Services, UpdatedAt: time.Now().UTC()}, Prepared: prepared}
	if err = c.save(r); err != nil {
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
	if r.Plan.Status == "succeeded" {
		if active, e := c.active(r.Plan.InstanceID); e == nil && active == id {
			_ = c.clear(r.Plan.InstanceID)
		}
		return r.Plan, nil
	}
	if r.Plan.Status != "prepared" {
		return r.Plan, ErrUnresolved
	}
	active, err := c.active(r.Plan.InstanceID)
	if err != nil {
		return r.Plan, err
	}
	if active != "" {
		return r.Plan, ErrUnresolved
	}
	if len(r.Plan.Blocks) > 0 {
		return r.Plan, ErrInvalid
	}
	current, err := c.driver.Fingerprint(ctx, r.Prepared)
	if err != nil {
		return r.Plan, err
	}
	if current != r.Prepared.Before {
		return r.Plan, ErrDrift
	}
	r.Plan.Status = "running"
	if err = c.save(r); err != nil {
		return r.Plan, err
	}
	key, _ := json.Marshal(id)
	if err = privatefile.Write(filepath.Join(c.root, r.Plan.InstanceID, "active.json"), key); err != nil {
		return r.Plan, err
	}
	err = c.driver.Apply(ctx, r.Prepared)
	if err == nil {
		err = c.driver.Verify(ctx, r.Prepared)
	}
	if err != nil {
		r.Plan.Status = "unknown"
		if e := c.save(r); e != nil {
			return r.Plan, e
		}
		return r.Plan, ErrUnresolved
	}
	r.Plan.Status = "succeeded"
	if err = c.save(r); err != nil {
		return r.Plan, err
	}
	if err = c.clear(r.Plan.InstanceID); err != nil {
		return r.Plan, err
	}
	return r.Plan, nil
}
func (c *Control) Reconcile(ctx context.Context, id string) (Plan, error) {
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
	if r.Plan.Status == "succeeded" {
		if active, e := c.active(r.Plan.InstanceID); e == nil && active == id {
			_ = c.clear(r.Plan.InstanceID)
		}
		return r.Plan, nil
	}
	if r.Plan.Status != "unknown" && r.Plan.Status != "running" {
		return r.Plan, ErrInvalid
	}
	active, err := c.active(r.Plan.InstanceID)
	if err != nil || active != id {
		return r.Plan, ErrUnresolved
	}
	if c.driver.Verify(ctx, r.Prepared) != nil {
		return r.Plan, ErrUnresolved
	}
	r.Plan.Status = "succeeded"
	if err = c.save(r); err != nil {
		return r.Plan, err
	}
	if err = c.clear(r.Plan.InstanceID); err != nil {
		return r.Plan, err
	}
	return r.Plan, nil
}
