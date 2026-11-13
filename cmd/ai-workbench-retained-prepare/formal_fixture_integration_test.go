//go:build integration

package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

// This opt-in exercise uses only separately created network-none Docker
// fixtures. Owner and build/startup evidence are synthetic test records;
// they must never be used as evidence for the retained 35444 instance.
func TestFormalCoordinatorAgainstIsolatedCopy(t *testing.T) {
	dir, helper, recoveryConfig := os.Getenv("RETAINED_FORMAL_FIXTURE"), os.Getenv("RETAINED_FORMAL_HELPER"), os.Getenv("RETAINED_FORMAL_RECOVERY")
	if dir == "" || helper == "" || recoveryConfig == "" {
		t.Skip("explicit isolated fixture not supplied")
	}
	read := func(name string, v any) {
		t.Helper()
		b, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			t.Fatal(err)
		}
		if err := json.Unmarshal(b, v); err != nil {
			t.Fatal(err)
		}
	}
	var ids struct{ Net, Source, App, Restore string }
	read("containers.json", &ids)
	for _, id := range []string{ids.Net, ids.Source, ids.App, ids.Restore} {
		if !validContainerID(id) {
			t.Fatal("invalid fixture identity")
		}
		out, err := exec.Command("docker", "inspect", "--format", `{{index .Config.Labels "io.listingkit.task"}}`, id).Output()
		if err != nil || strings.TrimSpace(string(out)) != "chat1261-formal-fixture" {
			t.Fatal("not the authorized fixture")
		}
	}
	out, err := exec.Command("docker", "inspect", "--format", `{{.HostConfig.NetworkMode}}`, ids.Net).Output()
	if err != nil || strings.TrimSpace(string(out)) != "none" {
		t.Fatal("fixture has external network")
	}
	var m manifest
	read("formal-manifest.json", &m)
	var anchor formalAnchor
	read("formal-anchor.json", &anchor)
	if anchor.DBContainerID != ids.Source || anchor.AppContainerID != ids.App {
		t.Fatal("fixture target mismatch")
	}
	sourceRaw, err := os.ReadFile(filepath.Join(dir, "formal-source.json"))
	if err != nil {
		t.Fatal(err)
	}
	h := sha256.Sum256(sourceRaw)
	pin := hex.EncodeToString(h[:])
	paths := formalPaths{Source: filepath.Join(dir, "formal-source.json"), Anchor: filepath.Join(dir, "formal-anchor.json"), Admission: filepath.Join(dir, "formal-admission.json"), RecoveryConfig: recoveryConfig, Helper: helper}
	if _, err := runFormalReadOnlyHost(context.Background(), m, paths, strings.Repeat("0", 64), "state"); err == nil {
		t.Fatal("read-only inspection accepted the wrong source pin")
	}
	imageFixture := func(_ context.Context, id string) (formalImage, error) {
		for _, b := range []formalBuildIdentity{anchor.OriginalBuild, anchor.CandidateBuild} {
			if id == b.ImageID {
				return formalImage{ID: id, Revision: b.Commit, Tree: b.Tree, Target: b.Target}, nil
			}
		}
		return formalImage{}, os.ErrNotExist
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()
	bad := m
	bad.Target.DatabaseOID = "0"
	if err := runFormalHostWithSourcePin(ctx, bad, paths, pin, imageFixture); err == nil {
		t.Fatal("wrong DB accepted")
	}
	if err := runFormalHostWithSourcePin(ctx, m, paths, strings.Repeat("0", 64), imageFixture); err == nil {
		t.Fatal("wrong source pin accepted")
	}
	bad = m
	bad.ApprovedBinaryRevision = strings.Repeat("9", 40)
	if err := runFormalHostWithSourcePin(ctx, bad, paths, pin, imageFixture); err == nil {
		t.Fatal("wrong candidate accepted")
	}
	if err := runFormalHostWithSourcePin(ctx, m, paths, pin, imageFixture); err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	errs := make(chan error, 2)
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() { defer wg.Done(); errs <- runFormalHostWithSourcePin(ctx, m, paths, pin, imageFixture) }()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatal(err)
		}
	}
	state, err := runFormalReadOnlyHost(ctx, m, paths, pin, "state")
	if err != nil || state != "ORIGINAL_WORKBENCH_STATE_READY" {
		t.Fatalf("same-target read-only inspection unavailable: %s, %v", state, err)
	}
	wrong := m
	wrong.Target.SystemIdentifier = "0"
	if _, err := runFormalReadOnlyHost(ctx, wrong, paths, pin, "state"); err == nil {
		t.Fatal("read-only inspection accepted another cluster")
	}
	wrong = m
	wrong.Target.FactsDigest = strings.Repeat("0", 64)
	if _, err := runFormalReadOnlyHost(ctx, wrong, paths, pin, "state"); err == nil {
		t.Fatal("read-only inspection accepted a replacement snapshot")
	}
	// A completed operation must remain inspectable after its original window.
	var admission formalAdmission
	read("formal-admission.json", &admission)
	expired := m
	expired.Target.NotAfter = time.Now().Add(-time.Minute)
	anchor.Target.NotAfter = expired.Target.NotAfter
	anchorRaw, err := json.Marshal(anchor)
	if err != nil {
		t.Fatal(err)
	}
	expiredAnchor := filepath.Join(dir, "test-expired-anchor.json")
	expiredAdmission := filepath.Join(dir, "test-expired-admission.json")
	defer os.Remove(expiredAnchor)
	defer os.Remove(expiredAdmission)
	if err := os.WriteFile(expiredAnchor, anchorRaw, 0600); err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(anchorRaw)
	admission.AnchorSHA256 = hex.EncodeToString(sum[:])
	admissionRaw, err := json.Marshal(admission)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(expiredAdmission, admissionRaw, 0600); err != nil {
		t.Fatal(err)
	}
	expiredPaths := paths
	expiredPaths.Anchor, expiredPaths.Admission = expiredAnchor, expiredAdmission
	state, err = runFormalReadOnlyHost(ctx, expired, expiredPaths, pin, "state")
	if err != nil || state != "ORIGINAL_WORKBENCH_STATE_READY" {
		t.Fatalf("same-operation inspection after expiry unavailable: %s, %v", state, err)
	}
}

func TestFormalCaptureAgainstIsolatedOldCopy(t *testing.T) {
	dir, helper := os.Getenv("RETAINED_FORMAL_FIXTURE"), os.Getenv("RETAINED_FORMAL_HELPER")
	if dir == "" || helper == "" {
		t.Skip("explicit isolated fixture not supplied")
	}
	read := func(name string, v any) {
		t.Helper()
		b, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			t.Fatal(err)
		}
		if err := json.Unmarshal(b, v); err != nil {
			t.Fatal(err)
		}
	}
	var ids struct{ Restore string }
	var app struct{ App string }
	read("containers.json", &ids)
	read("capture-app.json", &app)
	for _, id := range []string{ids.Restore, app.App} {
		if !validContainerID(id) {
			t.Fatal("invalid fixture identity")
		}
		out, err := exec.Command("docker", "inspect", "--format", `{{index .Config.Labels "io.listingkit.task"}}`, id).Output()
		if err != nil || strings.TrimSpace(string(out)) != "chat1261-formal-fixture" {
			t.Fatal("not the authorized fixture")
		}
		out, err = exec.Command("docker", "inspect", "--format", `{{.HostConfig.NetworkMode}}`, id).Output()
		if err != nil || strings.TrimSpace(string(out)) != "none" {
			t.Fatal("capture fixture is not network-none")
		}
	}
	var m manifest
	var anchor formalAnchor
	read("capture-manifest.json", &m)
	read("capture-anchor.json", &anchor)
	if anchor.DBContainerID != ids.Restore || anchor.AppContainerID != app.App {
		t.Fatal("capture fixture mismatch")
	}
	raw, err := os.ReadFile(filepath.Join(dir, "capture-source.json"))
	if err != nil {
		t.Fatal(err)
	}
	h := sha256.Sum256(raw)
	pin := hex.EncodeToString(h[:])
	paths := formalPaths{Source: filepath.Join(dir, "capture-source.json"), Anchor: filepath.Join(dir, "capture-anchor.json"), Admission: filepath.Join(dir, "capture-admission.json"), Helper: helper}
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	if _, err := runFormalReadOnlyHost(ctx, m, paths, strings.Repeat("0", 64), "capture"); err == nil {
		t.Fatal("capture accepted wrong source pin")
	}
	out, err := runFormalReadOnlyHost(ctx, m, paths, pin, "capture")
	if err != nil || !strings.HasPrefix(out, "ORIGINAL_WORKBENCH_BASELINE ") {
		t.Fatalf("capture failed: %v", err)
	}
	parts := strings.Fields(out)
	if len(parts) != 3 || parts[1] == m.Target.FactsDigest || parts[2] == m.Target.PrivilegesDigest {
		t.Fatal("capture did not read the old target")
	}
	var admission formalAdmission
	read("capture-admission.json", &admission)
	admission.FreezeExpiresAt = time.Now().Add(-time.Minute)
	expiredPath := filepath.Join(dir, "test-expired-capture-admission.json")
	defer os.Remove(expiredPath)
	b, err := json.Marshal(admission)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(expiredPath, b, 0600); err != nil {
		t.Fatal(err)
	}
	paths.Admission = expiredPath
	if _, err := runFormalReadOnlyHost(ctx, m, paths, pin, "capture"); err == nil {
		t.Fatal("capture accepted expired freeze")
	}
	// Block one old table until the synthetic window ends: a partial snapshot
	// must never be reported as a valid frozen baseline.
	lockCtx, lockCancel := context.WithCancel(context.Background())
	defer lockCancel()
	locker := exec.CommandContext(lockCtx, "docker", "exec", ids.Restore, "psql", "-h", "/var/run/postgresql", "-p", "5434", "-U", "business_cluster_admin", "-d", "product_agent", "-c", "BEGIN; LOCK TABLE ai_workbench.messages IN ACCESS EXCLUSIVE MODE; SELECT pg_sleep(15)")
	if err := locker.Start(); err != nil {
		t.Fatal(err)
	}
	defer func() { lockCancel(); _ = locker.Wait() }()
	locked := false
	for i := 0; i < 30; i++ {
		q := exec.Command("docker", "exec", ids.Restore, "psql", "-h", "/var/run/postgresql", "-p", "5434", "-U", "business_cluster_admin", "-d", "product_agent", "-At", "-c", `SELECT EXISTS(SELECT 1 FROM pg_locks WHERE relation='ai_workbench.messages'::regclass AND mode='AccessExclusiveLock' AND granted)`)
		out, _ := q.Output()
		if strings.TrimSpace(string(out)) == "t" {
			locked = true
			break
		}
		time.Sleep(100 * time.Millisecond)
	}
	if !locked {
		t.Fatal("isolated lock unavailable")
	}
	deadline := time.Now().Add(30 * time.Second)
	admissionDeadline := time.Now().Add(6 * time.Second)
	short := m
	short.Target.NotAfter = deadline
	anchor.Target.NotAfter = deadline
	anchorRaw, err := json.Marshal(anchor)
	if err != nil {
		t.Fatal(err)
	}
	shortAnchor := filepath.Join(dir, "test-short-capture-anchor.json")
	shortAdmission := filepath.Join(dir, "test-short-capture-admission.json")
	defer os.Remove(shortAnchor)
	defer os.Remove(shortAdmission)
	if err := os.WriteFile(shortAnchor, anchorRaw, 0600); err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(anchorRaw)
	admission.AnchorSHA256 = hex.EncodeToString(sum[:])
	admission.FreezeStartedAt = time.Now().Add(-time.Minute)
	admission.FreezeExpiresAt = deadline
	admission.ExpiresAt = admissionDeadline
	b, err = json.Marshal(admission)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(shortAdmission, b, 0600); err != nil {
		t.Fatal(err)
	}
	paths.Anchor, paths.Admission = shortAnchor, shortAdmission
	if _, err := runFormalReadOnlyHost(ctx, short, paths, pin, "capture"); err == nil {
		t.Fatal("capture accepted a snapshot after expiry")
	}
}
