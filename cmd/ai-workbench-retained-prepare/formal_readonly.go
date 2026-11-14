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
	"strings"
	"time"

	store "task-processor/internal/integration/persistence/aiworkbench"
)

// Database inspection remains available after a maintenance window expires.
// Docker copies a helper and creates a temporary capability inside the target
// container, so running this on the retained instance requires authorization
// for those temporary filesystem writes even though SQL is read-only.
func runFormalReadOnlyHost(ctx context.Context, m manifest, paths formalPaths, pin, mode string) (string, error) {
	if mode != "state" && mode != "capture" {
		return "", errors.New("unsupported inspection mode")
	}
	var anchor formalAnchor
	anchorRaw, err := loadProtected(ctx, paths.Anchor, &anchor)
	if err != nil {
		return "", err
	}
	var admission formalAdmission
	if _, err = loadProtected(ctx, paths.Admission, &admission); err != nil {
		return "", err
	}
	h := sha256.Sum256(anchorRaw)
	if admission.AnchorSHA256 != hex.EncodeToString(h[:]) || !validDigest(admission.HelperSHA256) {
		return "", errors.New("inspection binding unavailable")
	}
	if err = validateFormalAnchor(anchor, m.Target); err != nil {
		return "", err
	}
	if mode == "capture" && !formalCaptureWindowValid(admission, m.Target, time.Now()) {
		return "", errors.New("frozen maintenance window required")
	}
	if err = verifyPinnedOriginalObservation(ctx, paths.Source, pin, anchor); err != nil {
		return "", err
	}
	if err = verifyOriginalLiveContainers(ctx, anchor); err != nil {
		return "", err
	}
	binary, err := os.ReadFile(paths.Helper)
	if err != nil || len(binary) == 0 {
		return "", errors.New("inspection helper unavailable")
	}
	digest := sha256.Sum256(binary)
	if hex.EncodeToString(digest[:]) != admission.HelperSHA256 {
		return "", errors.New("inspection helper digest mismatch")
	}
	var nonce [16]byte
	if _, err = rand.Read(nonce[:]); err != nil {
		return "", errors.New("temporary identity unavailable")
	}
	name := hex.EncodeToString(nonce[:])
	remote := "/tmp/ai-workbench-formal-" + name
	capPath := "/tmp/ai-workbench-cap-" + name
	if exec.CommandContext(ctx, "docker", "cp", paths.Helper, anchor.DBContainerID+":"+remote).Run() != nil || exec.CommandContext(ctx, "docker", "exec", anchor.DBContainerID, "chmod", "755", remote).Run() != nil {
		return "", errors.New("inspection helper preparation failed")
	}
	remoteHash, err := exec.CommandContext(ctx, "docker", "exec", anchor.DBContainerID, "sha256sum", remote).Output()
	if err != nil || len(strings.Fields(string(remoteHash))) == 0 || strings.Fields(string(remoteHash))[0] != admission.HelperSHA256 {
		return "", errors.New("inspection helper digest mismatch")
	}
	if err = verifyPinnedOriginalObservation(ctx, paths.Source, pin, anchor); err != nil {
		return "", err
	}
	if err = verifyOriginalLiveContainers(ctx, anchor); err != nil {
		return "", err
	}
	req, err := json.Marshal(formalChildRequest{Target: m.Target, Anchor: anchor, Admission: admission})
	if err != nil {
		return "", errors.New("inspection capability unavailable")
	}
	defer clear(req)
	capCmd := exec.CommandContext(ctx, "docker", "exec", "-i", anchor.DBContainerID, "sh", "-c", `set -eC; umask 077; cat > "$1"`, "sh", capPath)
	capCmd.Stdin = bytes.NewReader(req)
	if capCmd.Run() != nil {
		return "", errors.New("inspection capability unavailable")
	}
	childMode := "--internal-original-state"
	if mode == "capture" {
		childMode = "--internal-original-capture"
	}
	out, err := exec.CommandContext(ctx, "docker", "exec", anchor.DBContainerID, remote, childMode, capPath).Output()
	state := strings.TrimSpace(string(out))
	if mode == "capture" {
		if !formalCaptureWindowValid(admission, m.Target, time.Now()) {
			return "", errors.New("frozen maintenance window expired during capture")
		}
		parts := strings.Fields(state)
		if err != nil || len(parts) != 3 || parts[0] != "ORIGINAL_WORKBENCH_BASELINE" || !validDigest(parts[1]) || !validDigest(parts[2]) {
			return "", errors.New("original baseline unavailable")
		}
		return state, nil
	}
	if err != nil || (state != "ORIGINAL_WORKBENCH_STATE_OLD" && state != "ORIGINAL_WORKBENCH_STATE_READY") {
		return "", errors.New("original target state unknown; stop writes")
	}
	return state, nil
}

func formalCaptureWindowValid(a formalAdmission, target store.RetainedTarget, now time.Time) bool {
	return a.FreezeConfirmed && strings.TrimSpace(a.Window) != "" && a.Window == target.MaintenanceWindow && strings.TrimSpace(a.FreezeRef) != "" && !a.FreezeStartedAt.IsZero() && !now.Before(a.FreezeStartedAt) && now.Before(a.FreezeExpiresAt) && now.Before(a.ExpiresAt) && now.Before(target.NotAfter)
}
