package tests

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"slices"
	"testing"
)

func TestLegacyConfigProjectionDubClosureIsRetired(t *testing.T) {
	path := filepath.Join("..", "internal", "integration", "dub")
	if _, err := os.Lstat(path); !os.IsNotExist(err) {
		t.Errorf("retired Dub package must be absent: %v", err)
	}
	violations, err := findBannedImportViolations(filepath.Join("..", "internal"), []string{`"task-processor/internal/integration/dub"`}, nil, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(violations) != 0 {
		t.Errorf("retired Dub consumers: %v", violations)
	}
	for path, banned := range map[string][]string{
		"internal/core/config/config.go":                         {"Deprecated", "deprecatedEnvWarnings", "logDeprecatedEnvUsage"},
		"internal/core/config/loader_builder.go":                 {"Deprecated"},
		"internal/imageagent/projection.go":                      {"SlotMutation"},
		"internal/imageagent/store/projection.go":                {"SlotMutation", "orderedSlotProjectionMutations"},
		"internal/imageagent/temporal/activities_persistence.go": {"SlotMutation"},
		"internal/imageagent/temporal/activities_recovery_v3.go": {"SlotMutation"},
	} {
		file, err := parser.ParseFile(token.NewFileSet(), filepath.Join("..", filepath.FromSlash(path)), nil, 0)
		if err != nil {
			t.Fatal(err)
		}
		ast.Inspect(file, func(n ast.Node) bool {
			if id, ok := n.(*ast.Ident); ok && slices.Contains(banned, id.Name) {
				t.Errorf("%s retains retired identifier %s", path, id.Name)
			}
			return true
		})
	}
	for path, kept := range map[string][]string{
		"internal/core/config/config.go":          {"newViper", "AutomaticEnv", "BindEnv", "LoadConfig", "LoadConfigFromFile", "LoadConfigFromFileWithoutValidation"},
		"internal/core/config/loader_builder.go":  {"BuildConfig", "legacyListingKitUsernameAllowlistConfigured"},
		"internal/imageagent/projection.go":       {"ProjectionCommit", "SlotProjectionMutation", "SlotMutations"},
		"internal/imageagent/store/projection.go": {"projectionCommitFingerprint", "hashJSON", "withProjectionTransaction", "captureMemoryProjectionState", "restoreMemoryProjectionState", "validateSlotProjectionMutationIdentity"},
	} {
		file, err := parser.ParseFile(token.NewFileSet(), filepath.Join("..", filepath.FromSlash(path)), nil, 0)
		if err != nil {
			t.Fatal(err)
		}
		seen := map[string]bool{}
		ast.Inspect(file, func(n ast.Node) bool {
			if id, ok := n.(*ast.Ident); ok {
				seen[id.Name] = true
			}
			return true
		})
		for _, name := range kept {
			if !seen[name] {
				t.Errorf("%s must retain current %s", path, name)
			}
		}
	}
}
