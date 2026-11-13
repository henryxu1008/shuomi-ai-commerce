// ai-workbench-retained-prepare is an explicit maintenance command; it is
// never run by the service or the fresh-install initializer.
package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	coreconfig "task-processor/internal/core/config"
	store "task-processor/internal/integration/persistence/aiworkbench"
)

type evidence struct{ Owner, Path, SHA256 string }
type manifest struct {
	Scope          string               `json:"scope"`
	Instance       string               `json:"instance"`
	OldRevision    string               `json:"old_revision"`
	TargetRevision string               `json:"target_revision"`
	Target         store.RetainedTarget `json:"target"`
	Evidence       []evidence           `json:"evidence"`
}

// Evidence files are operator records, not backups. Hashing prevents accidental
// substitution; the named owner must actually verify the backup and freeze.
func verifyEvidence(entries []evidence) error {
	needed := map[string]bool{"business": false, "identity": false, "images": false, "temporal": false, "freeze": false}
	for _, entry := range entries {
		if _, ok := needed[entry.Owner]; !ok || needed[entry.Owner] || !filepath.IsAbs(entry.Path) || len(entry.SHA256) != 64 {
			return errors.New("invalid owner evidence")
		}
		f, err := os.Open(entry.Path)
		if err != nil {
			return errors.New("owner evidence unavailable")
		}
		h := sha256.New()
		n, err := io.Copy(h, io.LimitReader(f, 1048577))
		_ = f.Close()
		if err != nil || n == 0 || n > 1048576 || hex.EncodeToString(h.Sum(nil)) != strings.ToLower(entry.SHA256) {
			return errors.New("owner evidence does not match manifest")
		}
		needed[entry.Owner] = true
	}
	for _, present := range needed {
		if !present {
			return errors.New("missing owner evidence")
		}
	}
	return nil
}

func run() error {
	manifestPath := flag.String("manifest", "", "absolute path to the approved, non-secret target manifest")
	mode := flag.String("mode", "inspect", "inspect, apply, or apply-isolated")
	source := flag.String("isolated-source", "", "protected read-only source observation")
	receipt := flag.String("isolated-receipt", "", "protected isolated restore receipt")
	pin := flag.String("isolated-pin", "", "protected prior isolated container identity")
	dump := flag.String("isolated-dump", "", "protected logical dump")
	admissionPath := flag.String("isolated-admission", "", "protected isolated owner evidence")
	helper := flag.String("isolated-helper", "", "prebuilt Linux helper")
	flag.Parse()
	if flag.NArg() != 0 || !filepath.IsAbs(*manifestPath) || (*mode != "inspect" && *mode != "apply" && *mode != "apply-isolated") {
		return errors.New("requires an absolute --manifest and an explicit supported mode")
	}
	if err := coreconfig.VerifyPrivateFiles(context.Background(), []string{*manifestPath}); err != nil {
		return errors.New("manifest must be private")
	}
	f, err := os.Open(*manifestPath)
	if err != nil {
		return errors.New("manifest unavailable")
	}
	defer f.Close()
	dec := json.NewDecoder(io.LimitReader(f, 1048577))
	dec.DisallowUnknownFields()
	var m manifest
	if err := dec.Decode(&m); err != nil {
		return errors.New("invalid manifest")
	}
	var trailing any
	if err := dec.Decode(&trailing); err != io.EOF {
		return errors.New("invalid manifest trailing data")
	}
	if m.Scope != "chat-35444-retained-v1" || (m.Instance != "35444" && m.Instance != "35444-isolated") || m.OldRevision != "787207b03ec4e462a59e51b2d76cffa4ea52d9a2" || m.TargetRevision != "a72b31ebec2dbc2d1b1b37321ae55b0ebd17a348" {
		return errors.New("unapproved scope or revision")
	}
	if *mode == "apply" {
		// Disabled until the independently reviewed target provenance and
		// owner attestations are bound to a trusted approval record. Hashing
		// self-selected files alone is not sufficient to authorize DDL.
		return errors.New("apply is not available: target provenance and owner evidence are not admitted")
	}
	if *mode == "apply-isolated" {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
		defer cancel()
		if err := runIsolatedHost(ctx, m, isolatedPaths{Source: *source, Receipt: *receipt, Pin: *pin, Dump: *dump, Admission: *admissionPath, Helper: *helper}); err != nil {
			return err
		}
		fmt.Println("ISOLATED_READY")
		return nil
	}
	dsn := os.Getenv("AI_WORKBENCH_RETAINED_DSN")
	if dsn == "" {
		return errors.New("AI_WORKBENCH_RETAINED_DSN required")
	}
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		return errors.New("database unavailable")
	}
	raw, err := db.DB()
	if err != nil {
		return errors.New("database unavailable")
	}
	defer raw.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()
	state, err := store.InspectRetainedWorkbench(ctx, db, m.Target)
	if err != nil {
		return errors.New("not confirmed; stop and inspect the same target read-only; check maintenance evidence and database logs")
	}
	fmt.Printf("%s_STRUCTURE_ONLY %s (target provenance and owner evidence not verified)\n", strings.ToUpper(*mode), state)
	return nil
}
func main() {
	if len(os.Args) == 3 && (os.Args[1] == "--internal-isolated-child" || os.Args[1] == "--internal-isolated-probe") {
		if err := runIsolatedChild(os.Args[2], os.Args[1] == "--internal-isolated-probe"); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		return
	}
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
