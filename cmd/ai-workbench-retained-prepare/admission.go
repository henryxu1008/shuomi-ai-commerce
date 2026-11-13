package main

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	store "task-processor/internal/integration/persistence/aiworkbench"
)

func verifyOwnerFiles(p ownerProof) error {
	for _, item := range []struct{ path, digest string }{{p.BackupRef, p.BackupSHA256}, {p.RestoreRef, p.RestoreSHA256}} {
		if !filepath.IsAbs(item.path) || !validDigest(item.digest) {
			return errors.New("owner evidence path or digest invalid")
		}
		info, err := os.Lstat(item.path)
		if err != nil || !info.Mode().IsRegular() || info.Size() == 0 {
			return errors.New("owner evidence unavailable")
		}
		f, err := os.Open(item.path)
		if err != nil {
			return errors.New("owner evidence unavailable")
		}
		h := sha256.New()
		_, err = io.Copy(h, f)
		_ = f.Close()
		if err != nil || hex.EncodeToString(h.Sum(nil)) != item.digest {
			return errors.New("owner evidence digest mismatch")
		}
	}
	return nil
}

func validateTransactionBinding(a admission, r isolatedRestore, target store.RetainedTarget) error {
	if !a.RestoredAt.Equal(r.RestoredAt) {
		return errors.New("restore time does not match receipt")
	}
	deadline := a.ExpiresAt
	for _, p := range a.Proofs {
		if p.ExpiresAt.Before(deadline) {
			deadline = p.ExpiresAt
		}
	}
	if target.BackupEvidence != a.ReceiptSHA256 {
		return errors.New("target restore receipt mismatch")
	}
	if a.Target.ContainerID != r.ContainerID || a.Target.ImageID != r.ImageID || a.Target.Network != r.Network || a.Target.SystemID != r.SystemID || a.Target.DatabaseOID != r.DatabaseOID || a.Target.Database != r.Database || target.Database != r.Database || target.SystemIdentifier != r.SystemID || target.DatabaseOID != r.DatabaseOID || target.SessionUser != r.SessionUser || target.Address != "" || target.Port != 0 || target.MaintenanceWindow != a.Window || target.FactsDigest != a.FactsDigest || target.PrivilegesDigest != a.PrivilegesDigest || !target.NotAfter.Equal(deadline) {
		return errors.New("approved window and same-session database target do not match")
	}
	return nil
}

func validateSourceBinding(a admission, s sourceObservation, r isolatedRestore) error {
	d, ok := s.Databases["product_agent"]
	if !ok || a.Source.ContainerID != s.ContainerID || a.Source.ImageID != s.ImageID || a.Source.Database != d.Database || a.Source.SystemID != d.SystemID || a.Source.DatabaseOID != d.OID || a.DumpSHA256 != d.DumpSHA256 || a.DumpSHA256 != r.DumpSHA256 {
		return errors.New("approved source does not match verified source and dump")
	}
	return nil
}

// These records bind facts supplied by trusted local owners; validation does
// not by itself prove that a backup was restored or a freeze remains in force.
type targetIdentity struct {
	SystemID, DatabaseOID, Database, ContainerID, ImageID, Network string
}
type ownerProof struct {
	Owner, Window, SourceSystemID, TargetSystemID, TargetDatabaseOID                            string
	ResourceIdentity, BackupRef, BackupSHA256, RestoreRef, RestoreSHA256, VerifiedBy, FreezeRef string
	Frozen                                                                                      bool
	VerifiedAt, ExpiresAt                                                                       time.Time
}
type admission struct {
	Version, Window, Instance, EvidenceKind                                string
	Source, Target                                                         targetIdentity
	DumpSHA256, FactsDigest, PrivilegesDigest, ReceiptSHA256, HelperSHA256 string
	RestoredAt, ExpiresAt                                                  time.Time
	Proofs                                                                 []ownerProof
}

// A restore receipt must be created by the controlled copy operation and read
// independently from the separately retained source identity. Merely accepting
// both records from the same untrusted manifest would not establish provenance.
type restoreReceipt struct {
	Source, Target     targetIdentity
	DumpSHA256, Window string
}

func validateBindings(a admission, sourceAnchor targetIdentity, receipt restoreReceipt, liveSource, liveTarget targetIdentity) error {
	if sourceAnchor != a.Source || sourceAnchor != liveSource || receipt.Source != sourceAnchor || receipt.Target != a.Target || receipt.Target != liveTarget || receipt.DumpSHA256 != a.DumpSHA256 || receipt.Window != a.Window {
		return errors.New("source anchor, restore receipt and live target do not match")
	}
	return nil
}

func validDigest(s string) bool {
	if len(s) != 64 || s != strings.ToLower(s) {
		return false
	}
	_, err := hex.DecodeString(s)
	return err == nil
}
func validateAdmission(a admission, now time.Time) error {
	if !validDigest(a.ReceiptSHA256) || !validDigest(a.HelperSHA256) {
		return errors.New("missing receipt or helper digest")
	}
	if a.Source.SystemID == a.Target.SystemID {
		return errors.New("isolated copy must be a different database cluster")
	}
	if a.Version != "chat-retained-admission-v1" || strings.TrimSpace(a.Window) == "" || a.Instance != "35444-isolated" || a.EvidenceKind != "isolated" || a.Source.Database != "product_agent" || a.Target.Database == "" || a.Source.ContainerID == "" || a.Target.ContainerID == "" || a.Source.ContainerID == a.Target.ContainerID || a.Source.ImageID == "" || a.Source.ImageID != a.Target.ImageID || a.Source.SystemID == "" || a.Source.DatabaseOID == "" || a.Target.SystemID == "" || a.Target.DatabaseOID == "" || a.Target.Network != "none" || !validDigest(a.DumpSHA256) || !validDigest(a.FactsDigest) || !validDigest(a.PrivilegesDigest) || a.RestoredAt.IsZero() || a.RestoredAt.After(now) || !a.ExpiresAt.After(now) {
		return errors.New("invalid isolated target, provenance or window")
	}
	needed := map[string]bool{"business": false, "identity": false, "images": false, "temporal": false, "freeze": false}
	for _, p := range a.Proofs {
		if _, ok := needed[p.Owner]; !ok || needed[p.Owner] || p.Window != a.Window || p.SourceSystemID != a.Source.SystemID || p.TargetSystemID != a.Target.SystemID || p.TargetDatabaseOID != a.Target.DatabaseOID || strings.TrimSpace(p.ResourceIdentity) == "" || strings.TrimSpace(p.BackupRef) == "" || !validDigest(p.BackupSHA256) || strings.TrimSpace(p.RestoreRef) == "" || !validDigest(p.RestoreSHA256) || strings.TrimSpace(p.VerifiedBy) == "" || strings.TrimSpace(p.FreezeRef) == "" || !p.Frozen || p.VerifiedAt.IsZero() || p.VerifiedAt.After(now) || !p.ExpiresAt.After(now) || p.ExpiresAt.After(a.ExpiresAt) {
			return errors.New("owner proof missing, expired, revoked or mismatched")
		}
		needed[p.Owner] = true
	}
	for _, ok := range needed {
		if !ok {
			return errors.New("missing owner proof")
		}
	}
	return nil
}
