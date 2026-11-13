//go:build integration

package main

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// Opt-in read-only exercise of the explicitly prepared, network-isolated copy.
func TestRecordedIsolatedProvenance(t *testing.T) {
	dir := os.Getenv("RETAINED_1261_PROOF_DIR")
	if dir == "" {
		t.Skip("isolated receipt not supplied")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	receipt, err := checkActualProvenance(ctx, filepath.Join(dir, "source-observation.json"), filepath.Join(dir, "retained_1261_agent_v2-restore-receipt.json"), filepath.Join(dir, "prior-isolated-pin.json"), filepath.Join(dir, "product_agent.dump"))
	if err != nil {
		t.Fatal(err)
	}
	if receipt.Receipt.Database != "retained_1261_agent_v2" {
		t.Fatal("unexpected isolated database")
	}
}
