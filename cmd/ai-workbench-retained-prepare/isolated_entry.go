package main

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"reflect"
	"regexp"
	"strings"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	coreconfig "task-processor/internal/core/config"
	store "task-processor/internal/integration/persistence/aiworkbench"
)

type isolatedPaths struct{ Source, Receipt, Pin, Dump, Admission, Helper string }
type isolatedRequest struct {
	Target      store.RetainedTarget
	ContainerID string
}

func unsafeSocketMount(mount string) bool {
	return mount == "/" || mount == "/var" || mount == "/var/run" || mount == "/run" || mount == "/run/postgresql" || mount == "/var/run/postgresql" || strings.HasPrefix(mount, "/run/postgresql/") || strings.HasPrefix(mount, "/var/run/postgresql/")
}

// The host is the trusted local operator. Protected records make accidental
// substitution observable; they do not replace actual owner restoration or
// continuously enforce a freeze. This path only accepts a network-none copy.
func runIsolatedHost(ctx context.Context, m manifest, paths isolatedPaths) error {
	if m.Instance != "35444-isolated" {
		return errors.New("only the isolated instance is supported")
	}
	var a admission
	if _, err := loadProtected(ctx, paths.Admission, &a); err != nil {
		return err
	}
	if err := validateAdmission(a, time.Now()); err != nil {
		return err
	}
	provenance, err := checkActualProvenance(ctx, paths.Source, paths.Receipt, paths.Pin, paths.Dump)
	if err != nil {
		return err
	}
	if err := validateSourceBinding(a, provenance.Source, provenance.Receipt); err != nil {
		return err
	}
	if err := validateTransactionBinding(a, provenance.Receipt, m.Target); err != nil {
		return err
	}
	raw, err := os.ReadFile(paths.Receipt)
	if err != nil {
		return errors.New("receipt unavailable")
	}
	hash := sha256.Sum256(raw)
	if hex.EncodeToString(hash[:]) != a.ReceiptSHA256 {
		return errors.New("receipt digest mismatch")
	}
	for _, proof := range a.Proofs {
		if err := coreconfig.VerifyPrivateFiles(ctx, []string{proof.BackupRef, proof.RestoreRef}); err != nil {
			return errors.New("owner evidence is not private")
		}
		if err := verifyOwnerFiles(proof); err != nil {
			return err
		}
	}
	helper, err := os.ReadFile(paths.Helper)
	if err != nil || len(helper) == 0 {
		return errors.New("isolated helper unavailable")
	}
	hash = sha256.Sum256(helper)
	if hex.EncodeToString(hash[:]) != a.HelperSHA256 {
		return errors.New("helper digest mismatch")
	}
	// A bind mount containing a PostgreSQL socket would undermine the isolated
	// namespace boundary. The named container's own data volume is allowed.
	mounts, err := exec.CommandContext(ctx, "docker", "inspect", "--format", `{{range .Mounts}}{{.Destination}}{{println}}{{end}}`, provenance.Receipt.ContainerID).Output()
	if err != nil {
		return errors.New("isolated mounts unavailable")
	}
	for _, mount := range strings.Fields(string(mounts)) {
		if unsafeSocketMount(mount) {
			return errors.New("external socket mount is not allowed")
		}
	}
	if !time.Now().Before(m.Target.NotAfter) {
		return errors.New("maintenance window expired")
	}
	var nonce [16]byte
	if _, err := rand.Read(nonce[:]); err != nil {
		return errors.New("temporary identity unavailable")
	}
	remote := "/tmp/ai-workbench-retained-" + hex.EncodeToString(nonce[:])
	if err := exec.CommandContext(ctx, "docker", "cp", paths.Helper, provenance.Receipt.ContainerID+":"+remote).Run(); err != nil {
		return errors.New("isolated helper copy failed")
	}
	if err := exec.CommandContext(ctx, "docker", "exec", provenance.Receipt.ContainerID, "chmod", "755", remote).Run(); err != nil {
		return errors.New("isolated helper preparation failed")
	}
	remoteHash, err := exec.CommandContext(ctx, "docker", "exec", provenance.Receipt.ContainerID, "sha256sum", remote).Output()
	if err != nil || len(strings.Fields(string(remoteHash))) < 1 || strings.Fields(string(remoteHash))[0] != a.HelperSHA256 {
		return errors.New("isolated helper digest mismatch")
	}
	// Recheck the protected chain and owner evidence immediately before launch.
	current, err := checkActualProvenance(ctx, paths.Source, paths.Receipt, paths.Pin, paths.Dump)
	if err != nil || !reflect.DeepEqual(provenance, current) {
		return errors.New("provenance changed before launch")
	}
	var refreshed admission
	if _, err := loadProtected(ctx, paths.Admission, &refreshed); err != nil || !reflect.DeepEqual(a, refreshed) {
		return errors.New("owner admission changed before launch")
	}
	if err := validateAdmission(refreshed, time.Now()); err != nil {
		return err
	}
	if err := validateSourceBinding(refreshed, current.Source, current.Receipt); err != nil {
		return err
	}
	if err := validateTransactionBinding(refreshed, current.Receipt, m.Target); err != nil {
		return err
	}
	raw, err = os.ReadFile(paths.Receipt)
	if err != nil {
		return errors.New("receipt unavailable")
	}
	hash = sha256.Sum256(raw)
	if hex.EncodeToString(hash[:]) != a.ReceiptSHA256 {
		return errors.New("receipt changed before launch")
	}
	for _, proof := range a.Proofs {
		if err := coreconfig.VerifyPrivateFiles(ctx, []string{proof.BackupRef, proof.RestoreRef}); err != nil {
			return errors.New("owner evidence is not private")
		}
		if err := verifyOwnerFiles(proof); err != nil {
			return err
		}
	}
	if !time.Now().Before(m.Target.NotAfter) {
		return errors.New("maintenance window expired")
	}
	deadlineCtx, cancel := context.WithDeadline(ctx, m.Target.NotAfter)
	defer cancel()
	req, _ := json.Marshal(isolatedRequest{Target: m.Target, ContainerID: provenance.Receipt.ContainerID})
	capPath := "/tmp/ai-workbench-cap-" + hex.EncodeToString(nonce[:])
	capCmd := exec.CommandContext(deadlineCtx, "docker", "exec", "-i", provenance.Receipt.ContainerID, "sh", "-c", `set -eC; umask 077; cat > "$1"`, "sh", capPath)
	capCmd.Stdin = bytes.NewReader(req)
	if err := capCmd.Run(); err != nil {
		return errors.New("isolated capability unavailable")
	}
	cmd := exec.CommandContext(deadlineCtx, "docker", "exec", provenance.Receipt.ContainerID, remote, "--internal-isolated-child", capPath)
	out, err := cmd.Output()
	if err != nil || strings.TrimSpace(string(out)) != "ISOLATED_READY" {
		return errors.New("isolated operation not confirmed; inspect the same target before retry")
	}
	return nil
}

func runIsolatedChild(capPath string, probe bool) error {
	data, err := consumeIsolatedCapability(capPath)
	if err != nil {
		return err
	}
	if err := dropToPostgres(); err != nil {
		return err
	}
	var req isolatedRequest
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&req); err != nil || !regexp.MustCompile(`^[a-z][a-z0-9_]{0,62}$`).MatchString(req.Target.Database) {
		return errors.New("invalid isolated request")
	}
	var trailing any
	if err := dec.Decode(&trailing); err != io.EOF {
		return errors.New("invalid isolated request")
	}
	dsn := fmt.Sprintf("host=/var/run/postgresql port=5434 user=chat1260_restore_admin dbname=%s sslmode=disable", req.Target.Database)
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		return errors.New("isolated database unavailable")
	}
	raw, err := db.DB()
	if err != nil {
		return errors.New("isolated database unavailable")
	}
	defer raw.Close()
	ctx, cancel := context.WithDeadline(context.Background(), req.Target.NotAfter)
	defer cancel()
	if probe {
		if err := store.CheckRetainedIsolatedSession(ctx, db, req.Target, req.ContainerID); err != nil {
			return errors.New("isolated session check failed")
		}
		fmt.Println("ISOLATED_SESSION_VERIFIED")
		return nil
	}
	if _, err := store.ApplyRetainedIsolated(ctx, db, req.Target, req.ContainerID); err != nil {
		return errors.New("isolated preparation unconfirmed")
	}
	runtimeDSN := fmt.Sprintf("host=/var/run/postgresql port=5434 user=ai_workbench_runtime dbname=%s sslmode=disable", req.Target.Database)
	runtimeDB, err := gorm.Open(postgres.Open(runtimeDSN), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		return errors.New("committed or unknown; runtime verification unavailable")
	}
	runtimeRaw, err := runtimeDB.DB()
	if err != nil {
		return errors.New("committed or unknown; runtime verification unavailable")
	}
	defer runtimeRaw.Close()
	if err := store.VerifySchema(ctx, runtimeDB); err != nil {
		return errors.New("committed or unknown; runtime verification failed")
	}
	fmt.Println("ISOLATED_READY")
	return nil
}
