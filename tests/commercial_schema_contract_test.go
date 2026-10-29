package tests

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestAccountComposeCommercialSchemaUsesGORMUniqueConstraintNames(t *testing.T) {
	path := filepath.Join("..", "deployments/docker/referrals-compose/terraform/commercial-schema.sql")
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("retired commercial DDL remains: %v", err)
	}
	for _, name := range []string{"acquisition-compose", "referrals-compose"} {
		source, err := os.ReadFile(filepath.Join("..", "deployments/docker", name, "init.sh"))
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(string(source), "commercial-schema.sql") {
			t.Errorf("%s still installs retired DDL", name)
		}
	}
	// Current Store/Resource constraints and repeatability are exercised by the
	// original internal/workbench/schema and owner migration tests, not legacy SQL.
	for _, owner := range []string{"internal/workbench/schema/runtime.go", "internal/integration/orgresource/schema.go"} {
		if _, err := os.Stat(filepath.Join("..", filepath.FromSlash(owner))); err != nil {
			t.Fatal(err)
		}
	}
}
