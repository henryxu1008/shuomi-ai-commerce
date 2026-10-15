package tests

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLegacyPreviewAdapterStaysRetired(t *testing.T) {
	sources := trackedProductionTextSources(t, []string{"internal/compatibility/listingkit"}, func(path string) bool {
		return strings.HasSuffix(path, ".go")
	})
	violations, err := legacyPreviewAdapterViolations(sources)
	require.NoError(t, err)
	require.Empty(t, violations, "legacy preview adapter must remain retired under every filename")
}

func TestLegacyPreviewAdapterGuardRejectsRenamedRevival(t *testing.T) {
	sources := []listingKitImageBoundarySource{
		{path: "internal/compatibility/listingkit/legacy_preview.go", text: "package listingkit\nfunc AdaptLegacyPreviewShell() {}\n"},
		{path: "internal/compatibility/listingkit/revived_bridge.go", text: "package listingkit\nimport _ \"task-processor/internal/listing/preview\"\n"},
		{path: "internal/compatibility/listingkit/sourcehandoff/allowed.go", text: "package sourcehandoff\nimport _ \"task-processor/internal/product/sourcing\"\n"},
		{path: "internal/compatibility/listingkit/similar.go", text: "package listingkit\nimport _ \"task-processor/internal/listing/previewcache\"\n"},
	}
	violations, err := legacyPreviewAdapterViolations(sources)
	require.NoError(t, err)
	require.Equal(t, []string{
		"internal/compatibility/listingkit/legacy_preview.go -> AdaptLegacyPreviewShell",
		"internal/compatibility/listingkit/revived_bridge.go -> task-processor/internal/listing/preview",
	}, violations)
}

func legacyPreviewAdapterViolations(sources []listingKitImageBoundarySource) ([]string, error) {
	var violations []string
	for _, source := range sources {
		file, err := parser.ParseFile(token.NewFileSet(), source.path, source.text, 0)
		if err != nil {
			return nil, err
		}
		for _, declaration := range file.Decls {
			function, ok := declaration.(*ast.FuncDecl)
			if ok && function.Name.Name == "AdaptLegacyPreviewShell" {
				violations = append(violations, filepath.ToSlash(source.path)+" -> AdaptLegacyPreviewShell")
			}
		}
		for _, imported := range file.Imports {
			importPath, err := decodeGoImportPath(imported.Path.Value)
			if err != nil {
				return nil, err
			}
			if importMatchesPrefix(importPath, "task-processor/internal/listing/preview") {
				violations = append(violations, filepath.ToSlash(source.path)+" -> "+importPath)
			}
		}
	}
	sort.Strings(violations)
	return violations, nil
}

func TestLegacyConsumerFreezeRejectsNewEdges(t *testing.T) {
	const old = "internal/app/httpapi/composition_builder.go"
	const legacy = "task-processor/internal/compatibility/listingkit/sourcehandoff/a1688"
	baseline := map[string]struct{}{}
	for _, tc := range []struct {
		name, path, imported string
		blocked              bool
	}{
		{"retired source root is blocked", old, legacy, true},
		{"new file same package", "internal/app/httpapi/new.go", legacy, true},
		{"retired source HTTP is blocked", old, legacy + "/httpapi", true},
		{"new compatibility root", old, "task-processor/internal/compatibility/new", true},
		{"new tenant caller", "cmd/new/main.go", "task-processor/internal/tenantbridge", true},
		{"new tenant child", "tools/new/main.go", "task-processor/internal/tenantbridge/bootstrap", true},
		{"test consumer", "internal/product/new_test.go", legacy, true},
		{"current owner", old, "task-processor/internal/product/catalog", false},
		{"similar name", old, "task-processor/internal/tenantbridgex", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			for _, literal := range []string{`"` + tc.imported + `"`, "`" + tc.imported + "`", `"` + strings.Replace(tc.imported, "internal", `\x69nternal`, 1) + `"`} {
				sources := []listingKitImageBoundarySource{{path: tc.path, text: "//go:build guard_fixture\n\npackage fixture\nimport alias " + literal + "\n"}}
				violations, err := legacyConsumerViolations(sources, baseline)
				require.NoError(t, err)
				require.Equal(t, tc.blocked, len(violations) > 0, "%v", violations)
			}
		})
	}
}

func TestLegacyDrainDepguardRules(t *testing.T) {
	rules := loadDepguardRules(t, filepath.Join("..", ".golangci.yml"))
	for _, root := range []string{"compatibility", "tenantbridge"} {
		rule := requireDepguardRule(t, rules, "legacy_"+root+"_consumer_freeze")
		wantFiles := []string{"**/*.go"}
		seen := map[string]bool{}
		for edge := range legacyConsumerBaseline {
			parts := strings.Split(edge, " -> ")
			if importMatchesPrefix(parts[1], "task-processor/internal/"+root) && !seen[parts[0]] {
				wantFiles = append(wantFiles, "!**/"+parts[0])
				seen[parts[0]] = true
			}
		}
		assertExactStringSet(t, root+" files", rule.Files, wantFiles)
		denied := depguardDenyPackageSet(rule)
		for _, suffix := range []string{"$", "/"} {
			_, ok := denied["task-processor/internal/"+root+suffix]
			require.True(t, ok)
		}
	}
}

// Drain-only baseline at main 2fd42cc06. Remove entries as callers drain; never regenerate upward.
// Includes test callers and intra-legacy edges so neither can hide new consumers.
var legacyConsumerBaseline = map[string]struct{}{}

func legacyConsumerViolations(sources []listingKitImageBoundarySource, baseline map[string]struct{}) ([]string, error) {
	edges, err := phase3BannedImportDeclarationViolations(sources, []string{
		"task-processor/internal/compatibility", "task-processor/internal/tenantbridge",
	})
	if err != nil {
		return nil, err
	}
	var violations []string
	for _, edge := range edges {
		if _, allowed := baseline[edge]; !allowed {
			violations = append(violations, edge)
		}
	}
	return violations, nil
}

func TestCurrentOwnerLegacyRootDepguardCoverage(t *testing.T) {
	rules := loadDepguardRules(t, filepath.Join("..", ".golangci.yml"))
	rule := requireDepguardRule(t, rules, "current_owners_legacy_listingkit_root")
	var files []string
	for _, owner := range currentOwnerLegacyListingKitRoots {
		files = append(files, "**/internal/"+owner+"/*.go", "**/internal/"+owner+"/**/*.go")
	}
	assertExactStringSet(t, "current owner coverage", rule.Files, files)
	_, denied := depguardDenyPackageSet(rule)["task-processor/internal/listingkit$"]
	require.True(t, denied, "mixed root service owner must be denied")
}

func TestCurrentOwnerLegacyRootGuardRejectsStoreCenterFixture(t *testing.T) {
	violations, err := currentOwnerLegacyListingKitRootViolations([]listingKitImageBoundarySource{
		{path: "internal/storecenter/revived.go", text: "package storecenter\nimport _ \"task-processor/internal/listingkit\"\n"},
		{path: "internal/storecenter/current_leaf.go", text: "package storecenter\nimport _ \"task-processor/internal/listingkit/core\"\n"},
	})
	require.NoError(t, err)
	require.Equal(t, []string{"internal/storecenter/revived.go -> task-processor/internal/listingkit"}, violations)
}

func TestLegacyImportParserRejectsMalformedSources(t *testing.T) {
	_, err := legacyConsumerViolations([]listingKitImageBoundarySource{{path: "broken.go", text: "package broken\nimport ("}}, nil)
	require.Error(t, err)
}

// The old source-to-ListingKit task route and handoff are retired, not hidden.
// Protect the production composition without banning current Product Sourcing.
func TestLegacy1688TaskHandoffAndRouteStayRetired(t *testing.T) {
	for _, relative := range []string{
		"internal/sourceaccount",
		"internal/app/productsourcing/httpapi",
		"internal/infra/httpx/crawler_1688_handler.go",
		"internal/infra/httpx/crawler_1688_handler_test.go",
		"internal/crawler/alibaba1688/crawler_service.go",
		"internal/crawler/alibaba1688/crawler_service_test.go",
		"internal/crawler/alibaba1688/worker_processor.go",
		"internal/crawler/alibaba1688/worker_processor_test.go",
		"internal/crawler/alibaba1688/account_profile.go",
		"internal/crawler/alibaba1688/account_profile_test.go",
	} {
		_, err := os.Stat(filepath.Join("..", filepath.FromSlash(relative)))
		require.True(t, os.IsNotExist(err), "retired numeric source-account artifact must remain absent: %s (%v)", relative, err)
	}
	for _, relative := range []string{"internal/crawler/alibaba1688", "internal/infra/httpx"} {
		assertNoBannedImportPrefixes(t, filepath.Join("..", filepath.FromSlash(relative)), []string{"task-processor/internal/sourceaccount"}, nil)
	}
	assertNoBannedImportPrefixes(t, filepath.Join("..", "internal", "crawler", "alibaba1688"), []string{
		"task-processor/internal/platform/workerpool",
		"task-processor/internal/infra/httpx",
	}, nil)
	for _, source := range trackedProductionTextSources(t, []string{"internal/crawler/alibaba1688"}, func(path string) bool {
		return strings.HasSuffix(path, ".go")
	}) {
		file, err := parser.ParseFile(token.NewFileSet(), source.path, source.text, 0)
		require.NoError(t, err)
		ast.Inspect(file, func(node ast.Node) bool {
			switch declaration := node.(type) {
			case *ast.TypeSpec:
				require.NotContains(t, []string{"AccountProfile", "AccountProfileResolver", "Crawler1688Processor"}, declaration.Name.Name, source.path)
			case *ast.FuncDecl:
				require.NotContains(t, []string{"ProcessWithAccountProfile", "NewBrowserManagerForAccountProfile", "IsAccountFallbackEligible"}, declaration.Name.Name, source.path)
			}
			return true
		})
	}
	legacyRoot := filepath.Join("..", "internal", "compatibility", "listingkit", "sourcehandoff")
	_, err := os.Stat(legacyRoot)
	require.True(t, os.IsNotExist(err), "retired 1688 Task handoff package must remain absent: %v", err)
	for _, name := range []string{"1688-runtime-acceptance.ps1", "1688-runtime-acceptance.Tests.ps1"} {
		_, err := os.Stat(filepath.Join("..", "scripts", name))
		require.True(t, os.IsNotExist(err), "retired Task creation runner %s must remain absent: %v", name, err)
	}
	_, err = os.Stat(filepath.Join("..", "scripts", "lib", "listingkit-device-auth.ps1"))
	require.True(t, os.IsNotExist(err), "runner-only device-auth helper must remain absent: %v", err)

	for _, name := range []string{"composition_builder.go", "composition_modules.go", "types.go"} {
		path := filepath.Join("..", "internal", "app", "httpapi", name)
		_, err := os.Stat(path)
		require.True(t, os.IsNotExist(err), "retired Task assembly must remain absent: %s (%v)", name, err)
	}
}

func TestLegacy1688SharedAccountDTOStaysRetired(t *testing.T) {
	for _, source := range trackedProductionTextSources(t, []string{"internal/crawler/shared"}, func(path string) bool {
		return strings.HasSuffix(path, ".go")
	}) {
		file, err := parser.ParseFile(token.NewFileSet(), source.path, source.text, 0)
		require.NoError(t, err)
		ast.Inspect(file, func(node ast.Node) bool {
			switch declaration := node.(type) {
			case *ast.Field:
				for _, name := range declaration.Names {
					require.NotContains(t, []string{"SourceAccountID", "SourceAccessMode", "SourceFallbackReason"}, name.Name, source.path)
				}
			case *ast.FuncDecl:
				require.NotEqual(t, "EffectiveSourceAccessMode", declaration.Name.Name, source.path)
			}
			return true
		})
	}
}
