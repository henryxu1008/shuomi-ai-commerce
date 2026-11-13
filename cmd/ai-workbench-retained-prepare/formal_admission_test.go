package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	recovery "task-processor/internal/app/modelinvocationrecovery"
	resourceadapter "task-processor/internal/integration/orgresource"
	store "task-processor/internal/integration/persistence/aiworkbench"
)

func formalFixture(now time.Time) (formalAnchor, formalAdmission, store.RetainedTarget) {
	hash := strings.Repeat("a", 64)
	target := store.RetainedTarget{Database: "product_agent", Address: "", Port: 0, SessionUser: "business_cluster_admin", SystemIdentifier: "123", DatabaseOID: "456", MaintenanceWindow: "window", BackupEvidence: hash, FactsDigest: hash, PrivilegesDigest: hash, NotAfter: now.Add(time.Hour)}
	anchor := formalAnchor{Version: "chat-35444-original-anchor-v1", Instance: "35444", DBContainerID: strings.Repeat("b", 64), AppContainerID: strings.Repeat("c", 64), DBImageID: "sha256:" + hash, AppImageID: "sha256:" + hash, ComposeProject: "trial", Network: "container:shared", Target: target, ReservationBindingSHA256: hash, SocketPort: 5434, OwnerResourceIdentities: map[string]string{"business": "resource-business", "identity": "resource-identity", "images": "resource-images", "temporal": "resource-temporal", "commercial": "resource-commercial"}}
	anchor.OriginalBuild = formalBuildIdentity{Commit: "787207b03ec4e462a59e51b2d76cffa4ea52d9a2", Tree: strings.Repeat("d", 40), ImageID: anchor.AppImageID, ReceiptSHA256: hash, Target: "data-services-runtime"}
	anchor.CandidateBuild = formalBuildIdentity{Commit: strings.Repeat("e", 40), Tree: strings.Repeat("f", 40), ImageID: "sha256:" + hash, ReceiptSHA256: hash, Target: "data-services-runtime"}
	anchor.RecoveryConfigSHA256 = hash
	anchor.InvocationBindingSHA256 = hash
	anchor.OwnerAnchorPaths = map[string]string{}
	anchor.OwnerAnchorSHA256 = map[string]string{}
	for _, owner := range []string{"business", "identity", "images", "temporal", "commercial"} {
		anchor.OwnerAnchorPaths[owner] = "private-" + owner
		anchor.OwnerAnchorSHA256[owner] = hash
	}
	a := formalAdmission{Version: "chat-35444-original-admission-v1", Instance: "35444", EvidenceKind: "owner-confirmed", Window: "window", AnchorSHA256: hash, ReservationBindingSHA256: hash, ReservationQuantityMicros: 15445282, ReservationState: "reserved", HelperSHA256: hash, ReservationUnsettled: true, NoRelatedRecoveryInFlight: true, ExpiresAt: target.NotAfter, RestoreSystemID: "789", RestoreDatabaseOID: "987", RestoreContainerID: strings.Repeat("d", 64), FreezeRef: "freeze", FreezeConfirmed: true, FreezeStartedAt: now.Add(-4 * time.Minute), FreezeExpiresAt: target.NotAfter}
	for _, owner := range []string{"business", "identity", "images", "temporal", "commercial"} {
		a.Proofs = append(a.Proofs, formalOwnerProof{Proof: ownerProof{Owner: owner, Window: a.Window, SourceSystemID: target.SystemIdentifier, TargetSystemID: target.SystemIdentifier, TargetDatabaseOID: target.DatabaseOID, ResourceIdentity: "resource-" + owner, BackupRef: "backup", BackupSHA256: hash, RestoreRef: "restore", RestoreSHA256: hash, VerifiedBy: "owner", FreezeRef: "freeze", Frozen: true, VerifiedAt: now.Add(-time.Minute), ExpiresAt: a.ExpiresAt}, RestoreResourceID: "isolated-" + owner, ReservationBindingSHA256: hash, BackupAt: now.Add(-3 * time.Minute), RestoredAt: now.Add(-2 * time.Minute)})
	}
	return anchor, a, target
}

func TestFormalTargetBindsOriginalSnapshotEvenAfterWindow(t *testing.T) {
	anchor, _, target := formalFixture(time.Now().Add(-2 * time.Hour))
	if err := validateFormalAnchor(anchor, target); err != nil {
		t.Fatal(err)
	}
	for _, mutate := range []func(*store.RetainedTarget){
		func(v *store.RetainedTarget) { v.FactsDigest = strings.Repeat("0", 64) },
		func(v *store.RetainedTarget) { v.PrivilegesDigest = strings.Repeat("0", 64) },
		func(v *store.RetainedTarget) { v.BackupEvidence = strings.Repeat("0", 64) },
		func(v *store.RetainedTarget) { v.MaintenanceWindow = "different" },
		func(v *store.RetainedTarget) { v.NotAfter = v.NotAfter.Add(time.Hour) },
	} {
		changed := target
		mutate(&changed)
		if validateFormalAnchor(anchor, changed) == nil {
			t.Fatal("replacement operation accepted")
		}
	}
}

func TestFormalCaptureWindowBoundaries(t *testing.T) {
	now := time.Now()
	_, a, target := formalFixture(now)
	if !formalCaptureWindowValid(a, target, now) {
		t.Fatal("current freeze rejected")
	}
	a.FreezeStartedAt = now.Add(time.Second)
	if formalCaptureWindowValid(a, target, now) {
		t.Fatal("future freeze accepted")
	}
	a.FreezeStartedAt = now.Add(-time.Second)
	a.FreezeExpiresAt = now
	if formalCaptureWindowValid(a, target, now) {
		t.Fatal("expired freeze accepted")
	}
	a.FreezeExpiresAt = now.Add(time.Hour)
	a.ExpiresAt = now
	if formalCaptureWindowValid(a, target, now) {
		t.Fatal("expired admission accepted")
	}
	a.ExpiresAt = now.Add(time.Hour)
	a.FreezeRef = "  "
	if formalCaptureWindowValid(a, target, now) {
		t.Fatal("empty freeze reference accepted")
	}
}

func TestRecoveryInputRequiresExactBytesAndReadOnlyEndpoints(t *testing.T) {
	anchor, _, _ := formalFixture(time.Now())
	cfg := recovery.Config{InvocationDatabase: recovery.DatabaseConfig{Host: "127.0.0.1", Port: anchor.SocketPort, Database: "product_agent", User: "product_agent_runtime", Password: "synthetic-secret"}, ResourceDatabase: recovery.DatabaseConfig{Host: "127.0.0.1", Port: anchor.SocketPort, Database: "commercial", User: "commercial_owner_runtime"}}
	raw, err := json.Marshal(cfg)
	if err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(raw)
	anchor.RecoveryConfigSHA256 = hex.EncodeToString(sum[:])
	if _, err := decodeFormalRecoveryInput(raw, anchor); err != nil {
		t.Fatal(err)
	}
	if _, err := decodeFormalRecoveryInput(append(raw, ' '), anchor); err == nil {
		t.Fatal("different secret input accepted")
	}
}

func TestFormalChildErrorNeverLeaksUntrustedStderr(t *testing.T) {
	if got := formalFailureCategory("secret-password"); got != "child failure" {
		t.Fatal("untrusted error was exposed")
	}
	if got := formalFailureCategory("original invocation inspection unavailable"); got != "read-only invocation inspection" {
		t.Fatalf("unexpected category %q", got)
	}
}

func TestRecoveryFailureCategoryUsesOnlyKnownSafeLabels(t *testing.T) {
	if got := formalRecoveryFailure(errors.New("password=secret")); got != "original invocation inspection unavailable" {
		t.Fatal("untrusted error exposed")
	}
	if got := formalRecoveryFailure(fmt.Errorf("%w: invocation runtime qualification", recovery.ErrRecoveryUnavailable)); got != "invocation runtime qualification unavailable" {
		t.Fatalf("unexpected category %q", got)
	}
}

func TestFormalCapabilityDoesNotSerializeRecoveryCredentials(t *testing.T) {
	anchor, a, target := formalFixture(time.Now())
	data, err := json.Marshal(formalChildRequest{Target: target, Anchor: anchor, Admission: a})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), "\"Password\"") || strings.Contains(string(data), "\"RecoveryConfig\":") {
		t.Fatal("formal capability contains recovery credentials")
	}
}

func TestFormalReservationInspectionBindsOriginalInvocationAndReservation(t *testing.T) {
	anchor, a, _ := formalFixture(time.Now())
	cfg := recovery.Config{OrganizationID: "fixture-org", MemberID: "fixture-member", InvocationID: "fixture-invocation", ReservationID: "fixture-reservation", ExpectedDispatchFingerprint: strings.Repeat("b", 64)}
	identity := reservationIdentityDigest(cfg)
	anchor.ReservationBindingSHA256 = identity
	a.ReservationBindingSHA256 = identity
	anchor.InvocationBindingSHA256 = strings.Repeat("c", 64)
	out := recovery.Output{InvocationID: cfg.InvocationID, BindingDigest: anchor.InvocationBindingSHA256, Reservation: resourceadapter.ModelInvocationReservation{OrganizationID: cfg.OrganizationID, MemberID: cfg.MemberID, InvocationID: cfg.InvocationID, ReservationID: cfg.ReservationID, DispatchFingerprint: cfg.ExpectedDispatchFingerprint, QuantityUnit: "ai-micropoint-v1", Quantity: 15445282, State: "reserved"}}
	if err := validateReservationInspection(anchor, a, cfg, out, identity); err != nil {
		t.Fatal(err)
	}
	out.Reservation.ReservationID = "different"
	if err := validateReservationInspection(anchor, a, cfg, out, identity); err == nil {
		t.Fatal("different reservation accepted")
	}
	out.Reservation.ReservationID = cfg.ReservationID
	operation := "unexpected-settlement"
	out.Reservation.SettlementOperationID = &operation
	if err := validateReservationInspection(anchor, a, cfg, out, identity); err == nil {
		t.Fatal("reservation with settlement operation accepted")
	}
}

func TestFormalReservationCoordinatorOnlyInspects(t *testing.T) {
	anchor, a, _ := formalFixture(time.Now())
	cfg := recovery.Config{OrganizationID: "fixture-org", MemberID: "fixture-member", InvocationID: "fixture-invocation", ReservationID: "fixture-reservation", ExpectedDispatchFingerprint: strings.Repeat("b", 64)}
	identity := reservationIdentityDigest(cfg)
	anchor.ReservationBindingSHA256 = identity
	a.ReservationBindingSHA256 = identity
	anchor.InvocationBindingSHA256 = strings.Repeat("c", 64)
	out := recovery.Output{InvocationID: cfg.InvocationID, BindingDigest: anchor.InvocationBindingSHA256, Reservation: resourceadapter.ModelInvocationReservation{OrganizationID: cfg.OrganizationID, MemberID: cfg.MemberID, InvocationID: cfg.InvocationID, ReservationID: cfg.ReservationID, DispatchFingerprint: cfg.ExpectedDispatchFingerprint, QuantityUnit: "ai-micropoint-v1", Quantity: 15445282, State: "reserved"}}
	called := ""
	err := inspectFormalReservation(context.Background(), anchor, a, cfg, identity, func(_ context.Context, _ recovery.Config, action string) (recovery.Output, error) {
		called = action
		return out, nil
	})
	if err != nil || called != "inspect" {
		t.Fatalf("expected read-only inspect: %v", err)
	}
}

func TestFormalRecoveryConfigMustStayOnOriginalLocalCluster(t *testing.T) {
	anchor, _, _ := formalFixture(time.Now())
	cfg := recovery.Config{InvocationDatabase: recovery.DatabaseConfig{Host: "127.0.0.1", Port: anchor.SocketPort, Database: "product_agent", User: "product_agent_runtime"}, ResourceDatabase: recovery.DatabaseConfig{Host: "127.0.0.1", Port: anchor.SocketPort, Database: "commercial", User: "commercial_owner_runtime"}}
	if err := validateFormalRecoveryEndpoints(anchor, cfg); err != nil {
		t.Fatal(err)
	}
	cfg.ResourceDatabase.Host = "remote.example"
	if err := validateFormalRecoveryEndpoints(anchor, cfg); err == nil {
		t.Fatal("remote ledger endpoint accepted")
	}
}

func TestFormalAdmissionRejectsMissingOrMisboundProof(t *testing.T) {
	now := time.Date(2026, 11, 13, 12, 0, 0, 0, time.UTC)
	anchor, a, target := formalFixture(now)
	if err := validateFormalAdmission(anchor, a, target, strings.Repeat("a", 64), now); err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name   string
		change func(*formalAnchor, *formalAdmission, *store.RetainedTarget)
	}{
		{"synthetic", func(_ *formalAnchor, a *formalAdmission, _ *store.RetainedTarget) { a.EvidenceKind = "isolated" }},
		{"wrong_runtime_target", func(anchor *formalAnchor, _ *formalAdmission, _ *store.RetainedTarget) {
			anchor.CandidateBuild.Target = "runtime"
		}},
		{"wrong_target", func(_ *formalAnchor, _ *formalAdmission, t *store.RetainedTarget) { t.DatabaseOID = "other" }},
		{"restore_is_source", func(_ *formalAnchor, a *formalAdmission, t *store.RetainedTarget) {
			a.RestoreSystemID = t.SystemIdentifier
		}},
		{"missing_owner", func(_ *formalAnchor, a *formalAdmission, _ *store.RetainedTarget) { a.Proofs = a.Proofs[:4] }},
		{"wrong_window", func(_ *formalAnchor, a *formalAdmission, _ *store.RetainedTarget) { a.Proofs[0].Proof.Window = "other" }},
		{"wrong_reservation", func(_ *formalAnchor, a *formalAdmission, _ *store.RetainedTarget) { a.ReservationQuantityMicros++ }},
		{"inflight", func(_ *formalAnchor, a *formalAdmission, _ *store.RetainedTarget) {
			a.NoRelatedRecoveryInFlight = false
		}},
		{"freeze_revoked", func(_ *formalAnchor, a *formalAdmission, _ *store.RetainedTarget) { a.FreezeConfirmed = false }},
		{"owner_expired", func(_ *formalAnchor, a *formalAdmission, _ *store.RetainedTarget) {
			a.Proofs[0].Proof.ExpiresAt = now.Add(-time.Second)
		}},
		{"same_restore_resource", func(_ *formalAnchor, a *formalAdmission, _ *store.RetainedTarget) {
			a.Proofs[0].RestoreResourceID = a.Proofs[0].Proof.ResourceIdentity
		}},
		{"backup_before_freeze", func(_ *formalAnchor, a *formalAdmission, _ *store.RetainedTarget) {
			a.Proofs[0].BackupAt = now.Add(-5 * time.Minute)
		}},
		{"wrong_owner_resource", func(_ *formalAnchor, a *formalAdmission, _ *store.RetainedTarget) {
			a.Proofs[1].Proof.ResourceIdentity = "another-resource"
		}},
		{"wrong_owner_resource", func(_ *formalAnchor, a *formalAdmission, _ *store.RetainedTarget) {
			a.Proofs[1].Proof.ResourceIdentity = "another-resource"
		}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			anchor, a, target := formalFixture(now)
			tc.change(&anchor, &a, &target)
			if err := validateFormalAdmission(anchor, a, target, strings.Repeat("a", 64), now); err == nil {
				t.Fatal("invalid formal admission accepted")
			}
		})
	}
}

func TestFormalOwnerRestoreReceiptBindsActualResource(t *testing.T) {
	now := time.Date(2026, 11, 13, 12, 0, 0, 0, time.UTC)
	anchor, a, _ := formalFixture(now)
	p := a.Proofs[1]
	r := formalOwnerRestoreReceipt{Version: "chat-owner-restore-v1", AnchorSHA256: a.AnchorSHA256, Window: a.Window, Owner: p.Proof.Owner, SourceResourceID: p.Proof.ResourceIdentity, RestoreResourceID: p.RestoreResourceID, BackupSHA256: p.Proof.BackupSHA256, BackupAt: p.BackupAt, RestoredAt: p.RestoredAt, Successful: true}
	if err := validateFormalOwnerRestore(anchor, a, p, r); err != nil {
		t.Fatal(err)
	}
	r.SourceResourceID = "wrong"
	if err := validateFormalOwnerRestore(anchor, a, p, r); err == nil {
		t.Fatal("another owner resource accepted")
	}
}

func TestFormalOwnerAnchorBindsOriginalResourceToPinnedTarget(t *testing.T) {
	anchor, _, _ := formalFixture(time.Now())
	r := formalOwnerAnchor{Version: "chat-owner-source-v1", Instance: "35444", Owner: "identity", OriginalObservationSHA256: originalObservationSHA256, SourceResourceID: anchor.OwnerResourceIdentities["identity"], ObservedBy: "fixture-owner"}
	if err := validateFormalOwnerAnchor(anchor, r, originalObservationSHA256); err != nil {
		t.Fatal(err)
	}
	r.SourceResourceID = "other"
	if err := validateFormalOwnerAnchor(anchor, r, originalObservationSHA256); err == nil {
		t.Fatal("incorrect owner source accepted")
	}
}

func TestFormalOwnerProofRejectsAnotherValidOwnersAnchor(t *testing.T) {
	anchor, a, _ := formalFixture(time.Now())
	business := a.Proofs[0].Proof
	identity := formalOwnerAnchor{Version: "chat-owner-source-v1", Instance: "35444", Owner: "identity", OriginalObservationSHA256: originalObservationSHA256, SourceResourceID: anchor.OwnerResourceIdentities["identity"], ObservedBy: "fixture"}
	if err := validateProofOwnerAnchor(anchor, business, identity, originalObservationSHA256); err == nil {
		t.Fatal("different owner's otherwise valid source record accepted")
	}
}

func TestFormalOwnerCoverageRejectsMissingIdentityKey(t *testing.T) {
	h := strings.Repeat("a", 64)
	owner := formalOwnerAnchor{Owner: "identity", InventoryComplete: true, Components: []formalComponent{{ID: "db", Kind: "database", Digest: h}, {ID: "master", Kind: "master_key", Digest: h}, {ID: "config", Kind: "identity_config", Digest: h}}}
	restored := []formalRestoredComponent{{ID: "db", Kind: "database", SourceDigest: h, RestoredDigest: h}, {ID: "master", Kind: "master_key", SourceDigest: h, RestoredDigest: h}, {ID: "config", Kind: "identity_config", SourceDigest: h, RestoredDigest: h}}
	if err := validateFormalOwnerCoverage(owner, restored); err != nil {
		t.Fatal(err)
	}
	if err := validateFormalOwnerCoverage(owner, restored[:1]); err == nil {
		t.Fatal("missing identity key and configuration accepted")
	}
}

func TestFormalRestoreReceiptBindsSourceBackupAndIndependentRestore(t *testing.T) {
	now := time.Date(2026, 11, 13, 12, 0, 0, 0, time.UTC)
	anchor, a, _ := formalFixture(now)
	r := formalRestoreReceipt{Version: "chat-35444-restore-v1", AnchorSHA256: a.AnchorSHA256, Window: a.Window, SourceSystemID: anchor.Target.SystemIdentifier, SourceDatabaseOID: anchor.Target.DatabaseOID, BackupSHA256: a.Proofs[0].Proof.BackupSHA256, RestoreContainerID: a.RestoreContainerID, RestoreSystemID: a.RestoreSystemID, RestoreDatabaseOID: a.RestoreDatabaseOID, RestoreNetwork: "none", RestoredAt: a.Proofs[0].RestoredAt, Successful: true}
	if err := validateFormalRestore(anchor, a, r); err != nil {
		t.Fatal(err)
	}
	r.SourceDatabaseOID = "other"
	if err := validateFormalRestore(anchor, a, r); err == nil {
		t.Fatal("unrelated restore accepted")
	}
}

func TestFormalContainersMustMatchSeparateTrustedAnchor(t *testing.T) {
	anchor, _, _ := formalFixture(time.Now())
	db := formalContainer{ID: anchor.DBContainerID, ImageID: anchor.DBImageID, Project: anchor.ComposeProject, Network: anchor.Network, Running: true}
	app := formalContainer{ID: anchor.AppContainerID, ImageID: anchor.AppImageID, Project: anchor.ComposeProject, Network: anchor.Network, Running: false}
	if err := validateFormalContainers(anchor, db, app); err != nil {
		t.Fatal(err)
	}
	db.ID = strings.Repeat("f", 64)
	if err := validateFormalContainers(anchor, db, app); err == nil {
		t.Fatal("different live DB container accepted")
	}
}

func TestFormalAnchorMatchesIndependentlyPinnedObservation(t *testing.T) {
	anchor, _, _ := formalFixture(time.Now())
	obs := sourceRecord{Kind: "current-read-only-observation-not-historical-proof", ContainerID: anchor.DBContainerID, ImageID: anchor.DBImageID, ApplicationID: anchor.AppContainerID, ApplicationImageID: anchor.AppImageID, Network: anchor.Network}
	obs.Databases = make(map[string]observedSourceDatabase)
	obs.Databases["product_agent"] = observedSourceDatabase{SystemID: anchor.Target.SystemIdentifier, OID: anchor.Target.DatabaseOID, Database: anchor.Target.Database, User: anchor.Target.SessionUser}
	if err := validateOriginalObservation(anchor, obs); err != nil {
		t.Fatal(err)
	}
	obs.Databases["product_agent"] = observedSourceDatabase{SystemID: "different", OID: anchor.Target.DatabaseOID, Database: anchor.Target.Database, User: anchor.Target.SessionUser}
	if err := validateOriginalObservation(anchor, obs); err == nil {
		t.Fatal("different original source accepted")
	}
}

func TestFormalBuildReceiptMustMatchActualImageAndApprovedSource(t *testing.T) {
	commit := strings.Repeat("a", 40)
	tree := strings.Repeat("b", 40)
	image := "sha256:" + strings.Repeat("c", 64)
	r := formalBuildReceipt{SchemaVersion: 1, SourceCommit: commit, SourceTree: tree, Images: []formalBuildImage{{Component: "Backend", ImageID: image, Target: "runtime"}}}
	live := formalImage{ID: image, Revision: commit, Tree: tree, Target: "runtime"}
	if err := validateFormalBuild(r, live, commit, tree, image, "runtime"); err != nil {
		t.Fatal(err)
	}
	live.Revision = strings.Repeat("d", 40)
	if err := validateFormalBuild(r, live, commit, tree, image, "runtime"); err == nil {
		t.Fatal("image from another source accepted")
	}
}

func TestFormalReleaseMustMatchExplicitlyApprovedCandidate(t *testing.T) {
	anchor, _, _ := formalFixture(time.Now())
	m := manifest{ApprovedBinaryRevision: anchor.CandidateBuild.Commit, ApprovedBinaryTree: anchor.CandidateBuild.Tree}
	if err := validateFormalReleaseBinding(m, anchor); err != nil {
		t.Fatal(err)
	}
	anchor.CandidateBuild.Commit = strings.Repeat("9", 40)
	if err := validateFormalReleaseBinding(m, anchor); err == nil {
		t.Fatal("other valid image source accepted")
	}
}

func TestFormalReleaseRejectsOldImageAndUnmatchedStartupProof(t *testing.T) {
	anchor, _, _ := formalFixture(time.Now())
	m := manifest{ApprovedBinaryRevision: anchor.CandidateBuild.Commit, ApprovedBinaryTree: anchor.CandidateBuild.Tree}
	p := formalStartupProof{Version: "chat-isolated-startup-v1", Commit: anchor.CandidateBuild.Commit, Tree: anchor.CandidateBuild.Tree, ImageID: anchor.CandidateBuild.ImageID, RuntimeVerified: true, Isolated: true, SourceSystemID: anchor.Target.SystemIdentifier, TestSystemID: "separate", SchemaFingerprint: "55110ab58b16be50277d212dd85c3360757ba9432ef55cab7637191ea259fbe8"}
	if err := validateFormalStartup(anchor, m, p); err != nil {
		t.Fatal(err)
	}
	p.Commit = anchor.OriginalBuild.Commit
	if err := validateFormalStartup(anchor, m, p); err == nil {
		t.Fatal("old image startup proof accepted")
	}
	m.ApprovedBinaryRevision = anchor.OriginalBuild.Commit
	anchor.CandidateBuild.Commit = anchor.OriginalBuild.Commit
	if err := validateFormalReleaseBinding(m, anchor); err == nil {
		t.Fatal("old binary accepted as candidate")
	}
}
