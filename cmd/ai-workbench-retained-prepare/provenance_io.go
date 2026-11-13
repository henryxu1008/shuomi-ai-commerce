package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"strings"
	"time"

	coreconfig "task-processor/internal/core/config"
)

type sourceRecord struct {
	Kind        string `json:"kind"`
	ContainerID string `json:"source_container_id"`
	ImageID     string `json:"source_image_id"`
	Network     string `json:"source_network_mode"`
	Databases   map[string]struct {
		SystemID string `json:"system_identifier"`
		Database string `json:"database"`
		OID      string `json:"database_oid"`
		User     string `json:"session_user"`
		DumpSHA  string `json:"dump_sha256"`
	} `json:"databases"`
}
type restoreRecord struct {
	RestoredAt  time.Time `json:"observed_at"`
	Kind        string    `json:"kind"`
	SourceSHA   string    `json:"source_observation_sha256"`
	SourceDB    string    `json:"source_database"`
	DumpSHA     string    `json:"dump_sha256"`
	ContainerID string    `json:"target_container_id"`
	ImageID     string    `json:"target_image_id"`
	Network     string    `json:"target_network"`
	SystemID    string    `json:"target_system_identifier"`
	ExitOnError bool      `json:"exit_on_error"`
	Target      struct {
		Database string `json:"database"`
		OID      string `json:"database_oid"`
		User     string `json:"session_user"`
	} `json:"target"`
}
type pinRecord struct {
	ContainerID string `json:"container_id"`
	SystemID    string `json:"system_identifier"`
}

func loadProtected(ctx context.Context, path string, into any) ([]byte, error) {
	if err := coreconfig.VerifyPrivateFiles(ctx, []string{path}); err != nil {
		return nil, errors.New("protected provenance unavailable")
	}
	data, err := os.ReadFile(path)
	if err != nil || len(data) == 0 || len(data) > 1<<20 || json.Unmarshal(data, into) != nil {
		return nil, errors.New("protected provenance invalid")
	}
	return data, nil
}

func inspectDocker(ctx context.Context, id string) ([]string, error) {
	if len(id) != 64 {
		return nil, errors.New("invalid container identity")
	}
	if _, err := hex.DecodeString(id); err != nil {
		return nil, errors.New("invalid container identity")
	}
	out, err := exec.CommandContext(ctx, "docker", "inspect", "--format", `{{.Id}}|{{.Image}}|{{.HostConfig.NetworkMode}}|{{index .Config.Labels "io.listingkit.task"}}|{{.State.Running}}`, id).Output()
	if err != nil {
		return nil, errors.New("container inspection unavailable")
	}
	parts := strings.Split(strings.TrimSpace(string(out)), "|")
	if len(parts) != 5 || parts[4] != "true" {
		return nil, errors.New("container inspection invalid")
	}
	return parts, nil
}

func inspectPG(ctx context.Context, id, database, user string) (string, string, string, string, error) {
	if database == "" || user == "" {
		return "", "", "", "", errors.New("invalid database target")
	}
	q := `select (select system_identifier::text from pg_control_system())||'|'||(select oid::text from pg_database where datname=current_database())||'|'||current_database()||'|'||session_user`
	out, err := exec.CommandContext(ctx, "docker", "exec", id, "psql", "-X", "-qAt", "-v", "ON_ERROR_STOP=1", "-p", "5434", "-U", user, "-d", database, "-c", q).Output()
	if err != nil {
		return "", "", "", "", errors.New("database identity unavailable")
	}
	parts := strings.Split(strings.TrimSpace(string(out)), "|")
	if len(parts) != 4 {
		return "", "", "", "", errors.New("database identity invalid")
	}
	return parts[0], parts[1], parts[2], parts[3], nil
}

type validatedProvenance struct {
	Source  sourceObservation
	Receipt isolatedRestore
}

// checkActualProvenance validates separately generated private records against
// live containers and their PostgreSQL identities. It is read-only and cannot
// prove historical or cross-owner consistency.
func checkActualProvenance(ctx context.Context, sourcePath, receiptPath, pinPath, dumpPath string) (validatedProvenance, error) {
	var s sourceRecord
	var r restoreRecord
	var p pinRecord
	sourceBytes, err := loadProtected(ctx, sourcePath, &s)
	if err != nil {
		return validatedProvenance{}, err
	}
	if _, err = loadProtected(ctx, receiptPath, &r); err != nil {
		return validatedProvenance{}, err
	}
	if _, err = loadProtected(ctx, pinPath, &p); err != nil {
		return validatedProvenance{}, err
	}
	if err = coreconfig.VerifyPrivateFiles(ctx, []string{dumpPath}); err != nil {
		return validatedProvenance{}, errors.New("private dump unavailable")
	}
	dump, err := os.ReadFile(dumpPath)
	if err != nil || len(dump) == 0 {
		return validatedProvenance{}, errors.New("dump unavailable")
	}
	sh := sha256.Sum256(sourceBytes)
	dh := sha256.Sum256(dump)
	sc, err := inspectDocker(ctx, s.ContainerID)
	if err != nil {
		return validatedProvenance{}, err
	}
	if sc[2] != s.Network {
		return validatedProvenance{}, errors.New("source network changed")
	}
	tc, err := inspectDocker(ctx, r.ContainerID)
	if err != nil {
		return validatedProvenance{}, err
	}
	ss, so, sd, su, err := inspectPG(ctx, sc[0], "product_agent", "business_cluster_admin")
	if err != nil {
		return validatedProvenance{}, err
	}
	if sd != "product_agent" || su != "business_cluster_admin" {
		return validatedProvenance{}, errors.New("source session identity mismatch")
	}
	ts, to, td, tu, err := inspectPG(ctx, tc[0], r.Target.Database, r.Target.User)
	if err != nil {
		return validatedProvenance{}, err
	}
	d := s.Databases["product_agent"]
	obs := sourceObservation{Kind: s.Kind, ContainerID: s.ContainerID, ImageID: s.ImageID, Databases: map[string]sourceDatabase{"product_agent": {SystemID: d.SystemID, Database: d.Database, OID: d.OID, SessionUser: d.User, DumpSHA256: d.DumpSHA}}}
	rec := isolatedRestore{Kind: r.Kind, SourceObservationSHA256: r.SourceSHA, SourceDatabase: r.SourceDB, DumpSHA256: r.DumpSHA, ContainerID: r.ContainerID, ImageID: r.ImageID, Network: r.Network, SystemID: r.SystemID, Database: r.Target.Database, DatabaseOID: r.Target.OID, SessionUser: r.Target.User, ExitOnError: r.ExitOnError, RestoredAt: r.RestoredAt}
	live := liveProvenance{SourceContainer: sc[0], SourceImage: sc[1], SourceSystemID: ss, SourceDBOID: so, TargetContainer: tc[0], TargetImage: tc[1], TargetNetwork: tc[2], TargetLabel: tc[3], TargetSystemID: ts, TargetDBOID: to, TargetDatabase: td, TargetSessionUser: tu}
	if err := validateProvenance(obs, rec, isolatedPin{ContainerID: p.ContainerID, SystemID: p.SystemID}, live, hex.EncodeToString(sh[:]), hex.EncodeToString(dh[:])); err != nil {
		return validatedProvenance{}, err
	}
	return validatedProvenance{Source: obs, Receipt: rec}, nil
}
