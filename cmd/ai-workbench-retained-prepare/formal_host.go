package main

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"reflect"
	"strings"
	"time"

	recovery "task-processor/internal/app/modelinvocationrecovery"
	coreconfig "task-processor/internal/core/config"
	store "task-processor/internal/integration/persistence/aiworkbench"
)

type formalPaths struct{ Source, Anchor, Admission, RecoveryConfig, Helper string }
type formalChildRequest struct {
	Target    store.RetainedTarget
	Anchor    formalAnchor
	Admission formalAdmission
}

func formalFailureCategory(stderr string) string {
	switch strings.TrimSpace(stderr) {
	case "original invocation inspection unavailable":
		return "read-only invocation inspection"
	case "invocation runtime qualification unavailable", "resource runtime qualification unavailable":
		return "read-only runtime qualification"
	case "original invocation or reservation inspection mismatch":
		return "invocation or reservation binding"
	case "original preparation not confirmed; inspect the same target before retry":
		return "owner transaction"
	case "committed or unknown; runtime verification failed", "committed or unknown; runtime connection unavailable":
		return "runtime verification after commit"
	case "isolated capability unavailable", "isolated capability invalid", "isolated capability cannot be consumed":
		return "one-time capability"
	case "cannot lock process privileges", "cannot set current-thread privilege restriction", "cannot drop supplementary groups", "cannot drop group identity", "cannot drop user identity", "process privilege drop unverified", "saved identities not dropped", "root identity remains recoverable", "thread credentials unavailable", "thread privilege drop unverified":
		return "privilege drop"
	default:
		return "child failure"
	}
}

func readFormalRecoveryConfig(ctx context.Context, path string, anchor formalAnchor) (recovery.Config, []byte, error) {
	if err := coreconfig.VerifyPrivateFiles(ctx, []string{path}); err != nil {
		return recovery.Config{}, nil, errors.New("protected recovery configuration unavailable")
	}
	raw, err := os.ReadFile(path)
	if err != nil || len(raw) == 0 || len(raw) > 16<<10 {
		return recovery.Config{}, nil, errors.New("recovery configuration unavailable")
	}
	cfg, err := decodeFormalRecoveryInput(raw, anchor)
	if err != nil {
		clear(raw)
		return recovery.Config{}, nil, errors.New("recovery configuration invalid")
	}
	if reservationIdentityDigest(cfg) != originalReservationIdentitySHA256 || anchor.ReservationBindingSHA256 != originalReservationIdentitySHA256 || anchor.InvocationBindingSHA256 != originalInvocationBindingSHA256 {
		clear(raw)
		return recovery.Config{}, nil, errors.New("original invocation identity differs from the pinned record")
	}
	return cfg, raw, nil
}

func noExternalSocketMounts(ctx context.Context, id string) error {
	out, err := exec.CommandContext(ctx, "docker", "inspect", "--format", `{{range .Mounts}}{{.Destination}}{{println}}{{end}}`, id).Output()
	if err != nil {
		return errors.New("database container mounts unavailable")
	}
	for _, mount := range strings.Fields(string(out)) {
		if unsafeSocketMount(mount) {
			return errors.New("external PostgreSQL socket mount is not allowed")
		}
	}
	return nil
}

func verifyOriginalLiveContainers(ctx context.Context, anchor formalAnchor) error {
	db, err := inspectFormalContainer(ctx, anchor.DBContainerID)
	if err != nil {
		return err
	}
	app, err := inspectFormalContainer(ctx, anchor.AppContainerID)
	if err != nil {
		return err
	}
	if err := validateFormalContainers(anchor, db, app); err != nil {
		return err
	}
	return noExternalSocketMounts(ctx, anchor.DBContainerID)
}

// runFormalHost is explicit maintenance only. The protected records are owner
// attestations; operators must actually maintain the freeze throughout the run.
func runFormalHost(ctx context.Context, m manifest, paths formalPaths) error {
	return runFormalHostWithSourcePin(ctx, m, paths, originalObservationSHA256, inspectFormalImage)
}

// Production passes only the compiled original pin; tests substitute an
// independently created isolated observation without exposing a runtime flag.
func runFormalHostWithSourcePin(ctx context.Context, m manifest, paths formalPaths, sourcePin string, inspectImage func(context.Context, string) (formalImage, error)) error {
	target := m.Target
	anchor, a, err := readFormalAdmission(ctx, paths.Anchor, paths.Admission, target, sourcePin, inspectImage)
	if err != nil {
		return err
	}
	if err := validateFormalReleaseBinding(m, anchor); err != nil {
		return err
	}
	if err := verifyFormalStartupProof(ctx, anchor, m); err != nil {
		return err
	}
	if err := verifyPinnedOriginalObservation(ctx, paths.Source, sourcePin, anchor); err != nil {
		return err
	}
	if err := verifyOriginalLiveContainers(ctx, anchor); err != nil {
		return err
	}
	cfg, rawConfig, err := readFormalRecoveryConfig(ctx, paths.RecoveryConfig, anchor)
	if err != nil {
		return err
	}
	clear(rawConfig)
	helper, err := os.ReadFile(paths.Helper)
	if err != nil || len(helper) == 0 {
		return errors.New("approved original helper unavailable")
	}
	h := sha256.Sum256(helper)
	if hex.EncodeToString(h[:]) != a.HelperSHA256 {
		return errors.New("original helper digest mismatch")
	}
	var nonce [16]byte
	if _, err := rand.Read(nonce[:]); err != nil {
		return errors.New("temporary identity unavailable")
	}
	remote := "/tmp/ai-workbench-formal-" + hex.EncodeToString(nonce[:])
	if err := exec.CommandContext(ctx, "docker", "cp", paths.Helper, anchor.DBContainerID+":"+remote).Run(); err != nil {
		return errors.New("original helper preparation failed")
	}
	if err := exec.CommandContext(ctx, "docker", "exec", anchor.DBContainerID, "chmod", "755", remote).Run(); err != nil {
		return errors.New("original helper preparation failed")
	}
	remoteHash, err := exec.CommandContext(ctx, "docker", "exec", anchor.DBContainerID, "sha256sum", remote).Output()
	if err != nil || len(strings.Fields(string(remoteHash))) < 1 || strings.Fields(string(remoteHash))[0] != a.HelperSHA256 {
		return errors.New("original helper digest mismatch")
	}
	// Re-read independently protected inputs after copying and just before the
	// one-time capability is created. Do not automatically retry on failure.
	anchor2, a2, err := readFormalAdmission(ctx, paths.Anchor, paths.Admission, target, sourcePin, inspectImage)
	if err != nil {
		return err
	}
	if !reflect.DeepEqual(anchor, anchor2) || !reflect.DeepEqual(a, a2) {
		return errors.New("original admission changed before launch")
	}
	if err := validateFormalReleaseBinding(m, anchor2); err != nil {
		return err
	}
	if err := verifyFormalStartupProof(ctx, anchor2, m); err != nil {
		return err
	}
	if err := verifyPinnedOriginalObservation(ctx, paths.Source, sourcePin, anchor); err != nil {
		return err
	}
	if err := verifyOriginalLiveContainers(ctx, anchor); err != nil {
		return err
	}
	cfg2, rawConfig, err := readFormalRecoveryConfig(ctx, paths.RecoveryConfig, anchor)
	if err != nil {
		return err
	}
	defer clear(rawConfig)
	if !reflect.DeepEqual(cfg, cfg2) {
		return errors.New("recovery configuration changed before launch")
	}
	if !time.Now().Before(target.NotAfter) {
		return errors.New("maintenance window expired")
	}
	deadlineCtx, cancel := context.WithDeadline(ctx, target.NotAfter)
	defer cancel()
	req, err := json.Marshal(formalChildRequest{Target: target, Anchor: anchor, Admission: a})
	if err != nil {
		return errors.New("original capability unavailable")
	}
	defer clear(req)
	capPath := "/tmp/ai-workbench-cap-" + hex.EncodeToString(nonce[:])
	capCmd := exec.CommandContext(deadlineCtx, "docker", "exec", "-i", anchor.DBContainerID, "sh", "-c", `set -eC; umask 077; cat > "$1"`, "sh", capPath)
	capCmd.Stdin = bytes.NewReader(req)
	if err := capCmd.Run(); err != nil {
		return errors.New("original capability unavailable")
	}
	cmd := exec.CommandContext(deadlineCtx, "docker", "exec", "-i", anchor.DBContainerID, remote, "--internal-original-child", capPath)
	cmd.Stdin = bytes.NewReader(rawConfig)
	out, err := cmd.Output()
	if err != nil || strings.TrimSpace(string(out)) != "ORIGINAL_PREPARATION_VERIFIED" {
		category := "child failure"
		var exit *exec.ExitError
		if errors.As(err, &exit) {
			category = formalFailureCategory(string(exit.Stderr))
		}
		return errors.New("original preparation not confirmed (" + category + "); inspect the same target before retry")
	}
	return nil
}
