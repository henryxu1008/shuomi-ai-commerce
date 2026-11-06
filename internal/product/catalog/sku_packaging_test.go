package catalog

import (
	"crypto/sha256"
	"encoding/json"
	"testing"
)

func TestSKUPackagingLeavesOriginalSnapshotBytesAndDigestUnchanged(t *testing.T) {
	const old = `{"title":"Original","variants":[{"source_id":"stable","sku":"OLD","stock":0,"trace":{},"own_stock_declared":true}]}`
	var snapshot ProductSnapshot
	if err := json.Unmarshal([]byte(old), &snapshot); err != nil { t.Fatal(err) }
	cloned, err := CloneProductSnapshot(snapshot)
	if err != nil { t.Fatal(err) }
	raw, err := json.Marshal(cloned)
	if err != nil || string(raw) != old { t.Fatalf("old snapshot encoding changed: %s (%v)", raw, err) }
	if sha256.Sum256(raw) != sha256.Sum256([]byte(old)) { t.Fatal("old digest changed") }
}
