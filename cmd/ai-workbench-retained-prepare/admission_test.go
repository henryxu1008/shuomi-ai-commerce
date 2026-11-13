package main

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"testing"
	"time"

	store "task-processor/internal/integration/persistence/aiworkbench"
)

func validAdmissionFixture(now time.Time) admission {
	a := admission{
		Version: "chat-retained-admission-v1", Window: "fixture-window", Instance: "35444-isolated",
		Source:           targetIdentity{SystemID: "source-system", DatabaseOID: "100", Database: "product_agent", ContainerID: "source-container", ImageID: "image"},
		Target:           targetIdentity{SystemID: "target-system", DatabaseOID: "200", Database: "retained_fixture", ContainerID: "target-container", ImageID: "image", Network: "none"},
		DumpSHA256:       "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		FactsDigest:      "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
		PrivilegesDigest: "cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc",
		ReceiptSHA256:    "dddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddd",
		HelperSHA256:     "eeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee",
		RestoredAt:       now.Add(-time.Minute), ExpiresAt: now.Add(time.Hour), EvidenceKind: "isolated",
	}
	for _, owner := range []string{"business", "identity", "images", "temporal", "freeze"} {
		a.Proofs = append(a.Proofs, ownerProof{Owner: owner, Window: a.Window, SourceSystemID: a.Source.SystemID, TargetSystemID: a.Target.SystemID, TargetDatabaseOID: a.Target.DatabaseOID, ResourceIdentity: "isolated-" + owner, BackupRef: "backup-" + owner, BackupSHA256: a.DumpSHA256, RestoreRef: "restore-" + owner, RestoreSHA256: a.PrivilegesDigest, VerifiedBy: "fixture-" + owner, FreezeRef: "freeze-" + owner, Frozen: true, VerifiedAt: now.Add(-time.Minute), ExpiresAt: a.ExpiresAt})
	}
	return a
}

func TestAdmissionMustMatchTransactionTarget(t *testing.T) {
	now := time.Date(2026, 11, 13, 12, 0, 0, 0, time.UTC)
	a := validAdmissionFixture(now)
	r := isolatedRestore{ContainerID: a.Target.ContainerID, ImageID: a.Target.ImageID, Network: a.Target.Network, SystemID: a.Target.SystemID, Database: a.Target.Database, DatabaseOID: a.Target.DatabaseOID, SessionUser: "chat1260_restore_admin", RestoredAt: a.RestoredAt}
	target := store.RetainedTarget{Database: a.Target.Database, SessionUser: r.SessionUser, SystemIdentifier: a.Target.SystemID, DatabaseOID: a.Target.DatabaseOID, MaintenanceWindow: a.Window, BackupEvidence: a.ReceiptSHA256, FactsDigest: a.FactsDigest, PrivilegesDigest: a.PrivilegesDigest, NotAfter: a.ExpiresAt}
	if err := validateTransactionBinding(a, r, target); err != nil {
		t.Fatal(err)
	}
	target.FactsDigest = a.PrivilegesDigest
	if err := validateTransactionBinding(a, r, target); err == nil {
		t.Fatal("different snapshot accepted")
	}
	target.FactsDigest = a.FactsDigest
	a.Proofs[0].ExpiresAt = now.Add(10 * time.Minute)
	if err := validateTransactionBinding(a, r, target); err == nil {
		t.Fatal("an earlier owner deadline must limit the transaction")
	}
	target.NotAfter = a.Proofs[0].ExpiresAt
	if err := validateTransactionBinding(a, r, target); err != nil {
		t.Fatal(err)
	}
}

func TestAdmissionMustBindVerifiedSourceAndDump(t *testing.T) {
	a := validAdmissionFixture(time.Date(2026, 11, 13, 12, 0, 0, 0, time.UTC))
	s := sourceObservation{ContainerID: a.Source.ContainerID, ImageID: a.Source.ImageID, Databases: map[string]sourceDatabase{"product_agent": {SystemID: a.Source.SystemID, OID: a.Source.DatabaseOID, Database: a.Source.Database, DumpSHA256: a.DumpSHA256}}}
	r := isolatedRestore{DumpSHA256: a.DumpSHA256}
	if err := validateSourceBinding(a, s, r); err != nil {
		t.Fatal(err)
	}
	a.Source.DatabaseOID = "wrong"
	if err := validateSourceBinding(a, s, r); err == nil {
		t.Fatal("different source accepted")
	}
	a.Source.DatabaseOID = s.Databases["product_agent"].OID
	a.DumpSHA256 = a.FactsDigest
	if err := validateSourceBinding(a, s, r); err == nil {
		t.Fatal("different dump accepted")
	}
}

func TestEvidenceFilesMustMatchTheApprovedDigest(t *testing.T) {
	dir := t.TempDir()
	backup := filepath.Join(dir, "backup")
	restore := filepath.Join(dir, "restore")
	if err := os.WriteFile(backup, []byte("fixture backup"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(restore, []byte("fixture restored"), 0600); err != nil {
		t.Fatal(err)
	}
	hash := func(s string) string { h := sha256.Sum256([]byte(s)); return hex.EncodeToString(h[:]) }
	p := ownerProof{BackupRef: backup, BackupSHA256: hash("fixture backup"), RestoreRef: restore, RestoreSHA256: hash("fixture restored")}
	if err := verifyOwnerFiles(p); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(restore, []byte("changed"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := verifyOwnerFiles(p); err == nil {
		t.Fatal("changed restore evidence accepted")
	}
}

func TestSocketMountAliasIsRejected(t *testing.T) {
	for _, mount := range []string{"/", "/var", "/var/run", "/var/run/postgresql", "/run", "/run/postgresql", "/run/postgresql/injected"} {
		if !unsafeSocketMount(mount) {
			t.Fatalf("unsafe mount accepted: %s", mount)
		}
	}
	if unsafeSocketMount("/var/lib/postgresql/data") {
		t.Fatal("unrelated data mount rejected")
	}
}

func TestAdmissionBindsTargetAndAllOwnerProofs(t *testing.T) {
	now := time.Date(2026, 11, 13, 12, 0, 0, 0, time.UTC)
	tests := []struct {
		name   string
		mutate func(*admission)
	}{
		{"wrong_target", func(a *admission) { a.Proofs[0].TargetDatabaseOID = "201" }},
		{"wrong_window", func(a *admission) { a.Proofs[1].Window = "another" }},
		{"missing_owner", func(a *admission) { a.Proofs = a.Proofs[:4] }},
		{"duplicate_owner", func(a *admission) { a.Proofs[1].Owner = "business" }},
		{"missing_restore", func(a *admission) { a.Proofs[2].RestoreRef = "" }},
		{"freeze_revoked", func(a *admission) { a.Proofs[4].Frozen = false }},
		{"expired", func(a *admission) { a.ExpiresAt = now.Add(-time.Second) }},
		{"unsafe_network", func(a *admission) { a.Target.Network = "bridge" }},
		{"same_cluster", func(a *admission) {
			a.Target.SystemID = a.Source.SystemID
			for i := range a.Proofs {
				a.Proofs[i].TargetSystemID = a.Target.SystemID
			}
		}},
		{"missing_resource", func(a *admission) { a.Proofs[1].ResourceIdentity = "" }},
		{"wrong_backup_digest", func(a *admission) { a.Proofs[1].BackupSHA256 = "oops" }},
		{"dump_missing", func(a *admission) { a.DumpSHA256 = "" }},
	}
	if err := validateAdmission(validAdmissionFixture(now), now); err != nil {
		t.Fatalf("valid fixture: %v", err)
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			a := validAdmissionFixture(now)
			tc.mutate(&a)
			if err := validateAdmission(a, now); err == nil {
				t.Fatal("invalid admission accepted")
			}
		})
	}
}

func TestAdmissionRejectsDifferentAnchoredSourceOrLiveTarget(t *testing.T) {
	now := time.Date(2026, 11, 13, 12, 0, 0, 0, time.UTC)
	a := validAdmissionFixture(now)
	anchor := a.Source
	restore := restoreReceipt{Source: a.Source, Target: a.Target, DumpSHA256: a.DumpSHA256, Window: a.Window}
	if err := validateBindings(a, anchor, restore, a.Source, a.Target); err != nil {
		t.Fatal(err)
	}
	badAnchor := anchor
	badAnchor.ContainerID = "different-source"
	if err := validateBindings(a, badAnchor, restore, a.Source, a.Target); err == nil {
		t.Fatal("mismatched source anchor accepted")
	}
	badReceipt := restore
	badReceipt.DumpSHA256 = a.FactsDigest
	if err := validateBindings(a, anchor, badReceipt, a.Source, a.Target); err == nil {
		t.Fatal("mismatched restore digest accepted")
	}
	live := a.Target
	live.DatabaseOID = "different-oid"
	if err := validateBindings(a, anchor, restore, a.Source, live); err == nil {
		t.Fatal("mismatched live target accepted")
	}
	live = a.Target
	live.ContainerID = "different-container"
	if err := validateBindings(a, anchor, restore, a.Source, live); err == nil {
		t.Fatal("mismatched live container accepted")
	}
}
