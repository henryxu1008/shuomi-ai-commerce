package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"os/exec"
	"strings"
	"time"

	coreconfig "task-processor/internal/core/config"
	store "task-processor/internal/integration/persistence/aiworkbench"
)

type formalInspector func(context.Context, string) (formalContainer, error)

// This digest identifies the separately captured 35444 observation used only
// as a target identity anchor. It is not a backup or a freeze attestation.
const originalObservationSHA256 = "7a1e2dba19acb999447ad5805fe3d888e30ab9ca36a6b1c6c83a84c8434bb3c2"

func verifyPinnedOriginalObservation(ctx context.Context, path, pin string, anchor formalAnchor) error {
	var observed sourceRecord
	raw, err := loadProtected(ctx, path, &observed)
	if err != nil {
		return err
	}
	hash := sha256.Sum256(raw)
	if !validDigest(pin) || hex.EncodeToString(hash[:]) != pin {
		return errors.New("original observation does not match the fixed target anchor")
	}
	return validateOriginalObservation(anchor, observed)
}

func inspectFormalContainer(ctx context.Context, id string) (formalContainer, error) {
	if !validContainerID(id) {
		return formalContainer{}, errors.New("invalid original container identity")
	}
	out, err := exec.CommandContext(ctx, "docker", "inspect", "--format", `{{.Id}}|{{.Image}}|{{index .Config.Labels "com.docker.compose.project"}}|{{.HostConfig.NetworkMode}}|{{.State.Running}}|{{.State.Paused}}`, id).Output()
	if err != nil {
		return formalContainer{}, errors.New("original container unavailable")
	}
	p := strings.Split(strings.TrimSpace(string(out)), "|")
	if len(p) != 6 {
		return formalContainer{}, errors.New("invalid original container observation")
	}
	return formalContainer{ID: p[0], ImageID: p[1], Project: p[2], Network: p[3], Running: p[4] == "true", Paused: p[5] == "true"}, nil
}

func verifyFormalAnchor(ctx context.Context, path string, target store.RetainedTarget, inspect formalInspector) error {
	var anchor formalAnchor
	if _, err := loadProtected(ctx, path, &anchor); err != nil {
		return err
	}
	if err := validateFormalAnchor(anchor, target); err != nil {
		return err
	}
	db, err := inspect(ctx, anchor.DBContainerID)
	if err != nil {
		return err
	}
	app, err := inspect(ctx, anchor.AppContainerID)
	if err != nil {
		return err
	}
	return validateFormalContainers(anchor, db, app)
}

func readFormalAdmission(ctx context.Context, anchorPath, admissionPath string, target store.RetainedTarget, originalPin string, inspectImage func(context.Context, string) (formalImage, error)) (formalAnchor, formalAdmission, error) {
	var anchor formalAnchor
	var a formalAdmission
	anchorRaw, err := loadProtected(ctx, anchorPath, &anchor)
	if err != nil {
		return anchor, a, err
	}
	if _, err = loadProtected(ctx, admissionPath, &a); err != nil {
		return anchor, a, err
	}
	sum := sha256.Sum256(anchorRaw)
	if err := validateFormalAdmission(anchor, a, target, hex.EncodeToString(sum[:]), time.Now()); err != nil {
		return anchor, a, err
	}
	if err := verifyFormalBuildReceipt(ctx, anchor.OriginalBuild, inspectImage); err != nil {
		return anchor, a, err
	}
	if err := verifyFormalBuildReceipt(ctx, anchor.CandidateBuild, inspectImage); err != nil {
		return anchor, a, err
	}
	for _, item := range a.Proofs {
		p := item.Proof
		var ownerAnchor formalOwnerAnchor
		ownerRaw, err := loadProtected(ctx, anchor.OwnerAnchorPaths[p.Owner], &ownerAnchor)
		if err != nil {
			return anchor, a, err
		}
		ownerSHA := sha256.Sum256(ownerRaw)
		if !validDigest(anchor.OwnerAnchorSHA256[p.Owner]) || hex.EncodeToString(ownerSHA[:]) != anchor.OwnerAnchorSHA256[p.Owner] {
			return anchor, a, errors.New("owner source record digest mismatch")
		}
		if err := validateProofOwnerAnchor(anchor, p, ownerAnchor, originalPin); err != nil {
			return anchor, a, err
		}
		if err := coreconfig.VerifyPrivateFiles(ctx, []string{p.BackupRef, p.RestoreRef}); err != nil {
			return anchor, a, errors.New("owner evidence unavailable or not private")
		}
		if err := verifyOwnerFiles(p); err != nil {
			return anchor, a, err
		}
		if p.Owner == "business" {
			var receipt formalRestoreReceipt
			if _, err := loadProtected(ctx, p.RestoreRef, &receipt); err != nil {
				return anchor, a, err
			}
			if err := validateFormalRestore(anchor, a, receipt); err != nil {
				return anchor, a, err
			}
			if err := validateFormalOwnerCoverage(ownerAnchor, receipt.Components); err != nil {
				return anchor, a, err
			}
		} else {
			var receipt formalOwnerRestoreReceipt
			if _, err := loadProtected(ctx, p.RestoreRef, &receipt); err != nil {
				return anchor, a, err
			}
			if err := validateFormalOwnerRestore(anchor, a, item, receipt); err != nil {
				return anchor, a, err
			}
			if err := validateFormalOwnerCoverage(ownerAnchor, receipt.Components); err != nil {
				return anchor, a, err
			}
		}
	}
	return anchor, a, nil
}
