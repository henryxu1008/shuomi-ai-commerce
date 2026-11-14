package main

import (
	"encoding/hex"
	"errors"
	"strings"
	"time"

	store "task-processor/internal/integration/persistence/aiworkbench"
)

// The original database, its independently restored backup and every other
// owner's resources are separate identities. These records are signed off by
// the trusted local operator; hashes cannot themselves establish a freeze.
type formalAnchor struct {
	Version, Instance, DBContainerID, AppContainerID, DBImageID, AppImageID, ComposeProject, Network string
	Target                                                                                           store.RetainedTarget
	ReservationBindingSHA256                                                                         string
	InvocationBindingSHA256, RecoveryConfigSHA256                                                    string
	SocketPort                                                                                       int
	OwnerResourceIdentities                                                                          map[string]string
	OwnerAnchorPaths, OwnerAnchorSHA256                                                              map[string]string
	OriginalBuild, CandidateBuild                                                                    formalBuildIdentity
	CandidateStartupProofPath, CandidateStartupProofSHA256                                           string
}

type formalOwnerAnchor struct {
	Version, Instance, Owner, OriginalObservationSHA256, SourceResourceID, ObservedBy string
	InventoryComplete                                                                 bool
	Components                                                                        []formalComponent
}
type formalComponent struct{ ID, Kind, Digest string }
type formalRestoredComponent struct{ ID, Kind, SourceDigest, RestoredDigest string }

// Digests cover each owner's canonical content and reference inventory, not
// passwords printed in records. Completeness is a trusted owner attestation.
func validateFormalOwnerCoverage(owner formalOwnerAnchor, restored []formalRestoredComponent) error {
	required := map[string][]string{
		"business":   {"database", "roles_privileges"},
		"commercial": {"database", "roles_privileges", "reservation_binding"},
		"identity":   {"database", "master_key", "identity_config"},
		"images":     {"object_inventory", "object_data", "object_credentials"},
		"temporal":   {"persistent_data", "workflow_metadata"},
	}
	kinds, ok := required[owner.Owner]
	if !ok || !owner.InventoryComplete || len(owner.Components) == 0 || len(owner.Components) != len(restored) {
		return errors.New("owner recovery inventory incomplete")
	}
	items := map[string]formalComponent{}
	seenKinds := map[string]bool{}
	for _, c := range owner.Components {
		if c.ID == "" || c.Kind == "" || !validDigest(c.Digest) {
			return errors.New("invalid owner inventory component")
		}
		if _, exists := items[c.ID]; exists {
			return errors.New("duplicate owner inventory component")
		}
		items[c.ID] = c
		seenKinds[c.Kind] = true
	}
	for _, kind := range kinds {
		if !seenKinds[kind] {
			return errors.New("required owner component missing")
		}
	}
	seen := map[string]bool{}
	for _, r := range restored {
		c, exists := items[r.ID]
		if !exists || seen[r.ID] || r.Kind != c.Kind || r.SourceDigest != c.Digest || r.RestoredDigest != c.Digest {
			return errors.New("owner restored component does not match inventory")
		}
		seen[r.ID] = true
	}
	return nil
}

func validateFormalOwnerAnchor(anchor formalAnchor, r formalOwnerAnchor, originalPin string) error {
	if r.Version != "chat-owner-source-v1" || r.Instance != "35444" || r.Owner == "" || r.OriginalObservationSHA256 != originalPin || r.SourceResourceID == "" || r.SourceResourceID != anchor.OwnerResourceIdentities[r.Owner] || strings.TrimSpace(r.ObservedBy) == "" {
		return errors.New("independent owner source record mismatch")
	}
	return nil
}

func validateProofOwnerAnchor(anchor formalAnchor, p ownerProof, r formalOwnerAnchor, originalPin string) error {
	if err := validateFormalOwnerAnchor(anchor, r, originalPin); err != nil {
		return err
	}
	if r.Owner != p.Owner || r.SourceResourceID != p.ResourceIdentity {
		return errors.New("owner proof and source record mismatch")
	}
	return nil
}

type formalContainer struct {
	ID, ImageID, Project, Network string
	Running, Paused               bool
}

func validateOriginalObservation(anchor formalAnchor, s sourceRecord) error {
	d, ok := s.Databases["product_agent"]
	if !ok || s.Kind != "current-read-only-observation-not-historical-proof" || s.ContainerID != anchor.DBContainerID || s.ImageID != anchor.DBImageID || s.ApplicationID != anchor.AppContainerID || s.ApplicationImageID != anchor.AppImageID || s.Network != anchor.Network || d.Database != anchor.Target.Database || d.SystemID != anchor.Target.SystemIdentifier || d.OID != anchor.Target.DatabaseOID || d.User != anchor.Target.SessionUser {
		return errors.New("original anchor differs from independently pinned observation")
	}
	return nil
}

func validateFormalContainers(anchor formalAnchor, db, app formalContainer) error {
	if !db.Running || (app.Running && !app.Paused) || db.ID != anchor.DBContainerID || app.ID != anchor.AppContainerID || db.ImageID != anchor.DBImageID || app.ImageID != anchor.AppImageID || db.Project != anchor.ComposeProject || app.Project != anchor.ComposeProject || db.Network != anchor.Network || app.Network != anchor.Network {
		return errors.New("original application and database container identity drift")
	}
	return nil
}

type formalOwnerProof struct {
	Proof                    ownerProof
	RestoreResourceID        string
	ReservationBindingSHA256 string
	BackupAt, RestoredAt     time.Time
}
type formalAdmission struct {
	Version, Instance, EvidenceKind, Window, AnchorSHA256, ReservationBindingSHA256 string
	ReservationQuantityMicros                                                       int64
	ReservationState                                                                string
	HelperSHA256                                                                    string
	ReservationUnsettled, NoRelatedRecoveryInFlight                                 bool
	RestoreSystemID, RestoreDatabaseOID, RestoreContainerID                         string
	FreezeRef                                                                       string
	FreezeConfirmed                                                                 bool
	FreezeStartedAt, FreezeExpiresAt, ExpiresAt                                     time.Time
	Proofs                                                                          []formalOwnerProof
}

type formalRestoreReceipt struct {
	Version, AnchorSHA256, Window, SourceSystemID, SourceDatabaseOID, BackupSHA256 string
	RestoreContainerID, RestoreSystemID, RestoreDatabaseOID, RestoreNetwork        string
	RestoredAt                                                                     time.Time
	Successful                                                                     bool
	Components                                                                     []formalRestoredComponent
}

type formalOwnerRestoreReceipt struct {
	Version, AnchorSHA256, Window, Owner, SourceResourceID, RestoreResourceID, BackupSHA256 string
	BackupAt, RestoredAt                                                                    time.Time
	Successful                                                                              bool
	Components                                                                              []formalRestoredComponent
}

// This is evidence that an earlier snapshot was independently restored, not
// evidence of a current, cross-owner or frozen backup.
func validateMinimalRestore(anchor formalAnchor, p formalOwnerProof, r restoreRecord, sourcePin string) error {
	if !validDigest(sourcePin) || r.Kind != "isolated-restore-observation-v1" || r.SourceSHA != sourcePin || r.SourceDB != anchor.Target.Database || r.DumpSHA != p.Proof.BackupSHA256 || !validContainerID(r.ContainerID) || r.ContainerID == anchor.DBContainerID || r.ImageID != anchor.DBImageID || r.Network != "none" || r.SystemID == "" || r.SystemID == anchor.Target.SystemIdentifier || r.Target.Database == "" || r.Target.OID == "" || !r.ExitOnError {
		return errors.New("historical isolated business restore mismatch")
	}
	return nil
}

func validateFormalOwnerRestore(anchor formalAnchor, a formalAdmission, p formalOwnerProof, r formalOwnerRestoreReceipt) error {
	if r.Version != "chat-owner-restore-v1" || r.AnchorSHA256 != a.AnchorSHA256 || r.Window != a.Window || r.Owner != p.Proof.Owner || r.SourceResourceID != anchor.OwnerResourceIdentities[r.Owner] || r.SourceResourceID != p.Proof.ResourceIdentity || r.RestoreResourceID != p.RestoreResourceID || r.BackupSHA256 != p.Proof.BackupSHA256 || !r.BackupAt.Equal(p.BackupAt) || !r.RestoredAt.Equal(p.RestoredAt) || !r.Successful {
		return errors.New("owner restoration receipt mismatch")
	}
	return nil
}

func validateFormalRestore(anchor formalAnchor, a formalAdmission, r formalRestoreReceipt) error {
	var business *formalOwnerProof
	for i := range a.Proofs {
		if a.Proofs[i].Proof.Owner == "business" {
			if business != nil {
				return errors.New("duplicate business owner")
			}
			business = &a.Proofs[i]
		}
	}
	if business == nil || r.Version != "chat-35444-restore-v1" || r.AnchorSHA256 != a.AnchorSHA256 || r.Window != a.Window || r.SourceSystemID != anchor.Target.SystemIdentifier || r.SourceDatabaseOID != anchor.Target.DatabaseOID || r.BackupSHA256 != business.Proof.BackupSHA256 || r.RestoreContainerID != a.RestoreContainerID || r.RestoreSystemID != a.RestoreSystemID || r.RestoreDatabaseOID != a.RestoreDatabaseOID || r.RestoreNetwork != "none" || !r.RestoredAt.Equal(business.RestoredAt) || !r.Successful {
		return errors.New("business restore receipt mismatched")
	}
	return nil
}

func validContainerID(id string) bool {
	if len(id) != 64 {
		return false
	}
	_, err := hex.DecodeString(id)
	return err == nil
}

func validateFormalAnchor(anchor formalAnchor, target store.RetainedTarget) error {
	if anchor.OriginalBuild.Target != "data-services-runtime" || anchor.CandidateBuild.Target != anchor.OriginalBuild.Target {
		return errors.New("candidate backend target differs from the approved runtime topology")
	}
	if !validDigest(anchor.RecoveryConfigSHA256) || !validDigest(anchor.InvocationBindingSHA256) {
		return errors.New("original read-only inspection binding missing")
	}
	if anchor.OriginalBuild.Commit != "787207b03ec4e462a59e51b2d76cffa4ea52d9a2" || anchor.OriginalBuild.ImageID != anchor.AppImageID || !validGitID(anchor.OriginalBuild.Tree) || !validGitID(anchor.CandidateBuild.Commit) || !validGitID(anchor.CandidateBuild.Tree) || !validDigest(anchor.OriginalBuild.ReceiptSHA256) || !validDigest(anchor.CandidateBuild.ReceiptSHA256) {
		return errors.New("approved build identities missing")
	}
	if anchor.Version != "chat-35444-original-anchor-v1" || anchor.Instance != "35444" || !validContainerID(anchor.DBContainerID) || !validContainerID(anchor.AppContainerID) || anchor.DBContainerID == anchor.AppContainerID || !strings.HasPrefix(anchor.DBImageID, "sha256:") || !validDigest(strings.TrimPrefix(anchor.DBImageID, "sha256:")) || !strings.HasPrefix(anchor.AppImageID, "sha256:") || !validDigest(strings.TrimPrefix(anchor.AppImageID, "sha256:")) || anchor.ComposeProject == "" || anchor.Network == "" || !validDigest(anchor.ReservationBindingSHA256) {
		return errors.New("invalid original target anchor")
	}
	original := anchor.Target
	if original.Database != "product_agent" || original.SessionUser != "business_cluster_admin" || original.SystemIdentifier == "" || original.DatabaseOID == "" || target.Database != original.Database || target.Address != original.Address || target.Port != original.Port || target.SessionUser != original.SessionUser || target.SystemIdentifier != original.SystemIdentifier || target.DatabaseOID != original.DatabaseOID {
		return errors.New("original target does not match anchor")
	}
	if target.FactsDigest != original.FactsDigest || target.PrivilegesDigest != original.PrivilegesDigest || target.BackupEvidence != original.BackupEvidence || target.MaintenanceWindow != original.MaintenanceWindow || !target.NotAfter.Equal(original.NotAfter) {
		return errors.New("original operation snapshot does not match anchor")
	}
	if target.Address != "" || target.Port != 0 || anchor.SocketPort <= 0 || anchor.SocketPort > 65535 {
		return errors.New("original target requires its approved container Unix socket")
	}
	return nil
}

func validateFormalAdmission(anchor formalAnchor, a formalAdmission, target store.RetainedTarget, anchorSHA string, now time.Time) error {
	if err := validateFormalAnchor(anchor, target); err != nil {
		return err
	}
	if !validDigest(a.HelperSHA256) {
		return errors.New("approved original helper digest missing")
	}
	original := anchor.Target
	minimal := a.Version == "chat-35444-original-admission-v2"
	if (!minimal && a.Version != "chat-35444-original-admission-v1") || a.Instance != "35444" || a.EvidenceKind != "owner-confirmed" || !validDigest(anchorSHA) || a.AnchorSHA256 != anchorSHA || a.ReservationBindingSHA256 != anchor.ReservationBindingSHA256 || a.ReservationQuantityMicros != 15445282 || a.ReservationState != "reserved" || !a.ReservationUnsettled || !a.NoRelatedRecoveryInFlight || a.Window == "" || a.Window != target.MaintenanceWindow || !validDigest(target.FactsDigest) || !validDigest(target.PrivilegesDigest) {
		return errors.New("original owner or reservation admission invalid")
	}
	if a.RestoreSystemID == "" || a.RestoreSystemID == original.SystemIdentifier || a.RestoreDatabaseOID == "" || !validContainerID(a.RestoreContainerID) || a.RestoreContainerID == anchor.DBContainerID || a.RestoreContainerID == anchor.AppContainerID {
		return errors.New("backup restore target is not independent")
	}
	if !a.FreezeConfirmed || strings.TrimSpace(a.FreezeRef) == "" || a.FreezeStartedAt.IsZero() || a.FreezeStartedAt.After(now) || !a.FreezeExpiresAt.After(now) || !a.ExpiresAt.After(now) {
		return errors.New("maintenance freeze absent or expired")
	}
	deadline := a.ExpiresAt
	if a.FreezeExpiresAt.Before(deadline) {
		deadline = a.FreezeExpiresAt
	}
	needed := map[string]bool{"business": false, "identity": false, "images": false, "temporal": false, "commercial": false}
	if minimal {
		needed = map[string]bool{"business": false}
	}
	for owner := range needed {
		if !minimal && (strings.TrimSpace(anchor.OwnerAnchorPaths[owner]) == "" || !validDigest(anchor.OwnerAnchorSHA256[owner])) {
			return errors.New("independent owner source anchor missing")
		}
	}
	resources := map[string]bool{}
	for _, item := range a.Proofs {
		p := item.Proof
		if anchor.OwnerResourceIdentities[p.Owner] == "" || anchor.OwnerResourceIdentities[p.Owner] != p.ResourceIdentity {
			return errors.New("owner source resource differs from trusted anchor")
		}
		if !minimal && item.BackupAt.Before(a.FreezeStartedAt) {
			return errors.New("owner backup predates maintenance freeze")
		}
		if _, ok := needed[p.Owner]; !ok || needed[p.Owner] || p.Window != a.Window || p.SourceSystemID != original.SystemIdentifier || p.TargetSystemID != original.SystemIdentifier || p.TargetDatabaseOID != original.DatabaseOID || strings.TrimSpace(p.ResourceIdentity) == "" || resources[p.ResourceIdentity] || strings.TrimSpace(item.RestoreResourceID) == "" || item.RestoreResourceID == p.ResourceIdentity || strings.TrimSpace(p.BackupRef) == "" || !validDigest(p.BackupSHA256) || strings.TrimSpace(p.RestoreRef) == "" || !validDigest(p.RestoreSHA256) || strings.TrimSpace(p.VerifiedBy) == "" || p.FreezeRef != a.FreezeRef || !p.Frozen || item.BackupAt.IsZero() || item.RestoredAt.Before(item.BackupAt) || p.VerifiedAt.Before(item.RestoredAt) || p.VerifiedAt.After(now) || !p.ExpiresAt.After(now) {
			return errors.New("owner proof invalid or mismatched")
		}
		if p.Owner == "business" && target.BackupEvidence != p.BackupSHA256 {
			return errors.New("business backup binding mismatch")
		}
		if p.Owner == "commercial" && item.ReservationBindingSHA256 != anchor.ReservationBindingSHA256 {
			return errors.New("commercial reservation binding mismatch")
		}
		needed[p.Owner] = true
		resources[p.ResourceIdentity] = true
		if p.ExpiresAt.Before(deadline) {
			deadline = p.ExpiresAt
		}
	}
	for _, ok := range needed {
		if !ok {
			return errors.New("missing original owner evidence")
		}
	}
	if !target.NotAfter.Equal(deadline) {
		return errors.New("target deadline does not match owner evidence")
	}
	return nil
}
