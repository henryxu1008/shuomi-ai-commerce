package tests

import (
	"errors"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"
)

func TestAlibaba1688CrawlerDoesNotImportListingKitRoot(t *testing.T) {
	assertNoProductionBannedImports(t, filepath.Join("..", "internal", "crawler", "alibaba1688"), []string{
		`"task-processor/internal/listingkit"`,
	}, nil)
}

func TestAlibaba1688CrawlerDoesNotConfigureLegacyTenantThroughListingKitHTTPAPI(t *testing.T) {
	root := filepath.Join("..", "internal", "crawler", "alibaba1688")
	index, err := loadGoFileIndex(root, "")
	if err != nil {
		t.Fatal(err)
	}
	fset := token.NewFileSet()
	for path, facts := range index.files {
		if strings.HasSuffix(filepath.Base(path), "_test.go") {
			continue
		}
		file, err := parser.ParseFile(fset, path, facts.source, 0)
		if err != nil {
			t.Fatal(err)
		}
		listingKitHTTPAPINames := make(map[string]struct{})
		dotImportedListingKitHTTPAPI := false
		for _, imported := range file.Imports {
			if strings.Trim(imported.Path.Value, `"`) != "task-processor/internal/listingkit/httpapi" {
				continue
			}
			if imported.Name == nil {
				listingKitHTTPAPINames["httpapi"] = struct{}{}
				continue
			}
			if imported.Name.Name == "." {
				dotImportedListingKitHTTPAPI = true
				continue
			}
			if imported.Name.Name != "_" {
				listingKitHTTPAPINames[imported.Name.Name] = struct{}{}
			}
		}
		ast.Inspect(file, func(node ast.Node) bool {
			call, ok := node.(*ast.CallExpr)
			if !ok {
				return true
			}
			if dotImportedListingKitHTTPAPI {
				ident, ok := call.Fun.(*ast.Ident)
				if ok && ident.Name == "ConfigureLegacyTenantResolver" {
					t.Errorf("%s dot-imports listingkit/httpapi to configure Legacy Tenant Bridge; current sourcing must use native Organization identity", path)
				}
			}
			selector, ok := call.Fun.(*ast.SelectorExpr)
			if !ok || selector.Sel == nil || selector.Sel.Name != "ConfigureLegacyTenantResolver" {
				return true
			}
			ident, ok := selector.X.(*ast.Ident)
			if ok {
				if _, imported := listingKitHTTPAPINames[ident.Name]; imported {
					t.Errorf("%s still configures Legacy Tenant Bridge through listingkit/httpapi; current sourcing must use native Organization identity", path)
				}
			}
			return true
		})
	}
}

func TestAlibaba1688CrawlerDoesNotImportListingKitHTTPAPIForSourceAccountBuilder(t *testing.T) {
	root := filepath.Join("..", "internal", "crawler", "alibaba1688")
	index, err := loadGoFileIndex(root, "")
	if err != nil {
		t.Fatal(err)
	}
	fset := token.NewFileSet()
	for path, facts := range index.files {
		if strings.HasSuffix(filepath.Base(path), "_test.go") {
			continue
		}
		file, err := parser.ParseFile(fset, path, facts.source, 0)
		if err != nil {
			t.Fatal(err)
		}
		listingKitHTTPAPINames := make(map[string]struct{})
		dotImportedListingKitHTTPAPI := false
		for _, imported := range file.Imports {
			if strings.Trim(imported.Path.Value, `"`) != "task-processor/internal/listingkit/httpapi" {
				continue
			}
			if imported.Name == nil {
				listingKitHTTPAPINames["httpapi"] = struct{}{}
				continue
			}
			if imported.Name.Name == "." {
				dotImportedListingKitHTTPAPI = true
				continue
			}
			if imported.Name.Name != "_" {
				listingKitHTTPAPINames[imported.Name.Name] = struct{}{}
			}
		}
		ast.Inspect(file, func(node ast.Node) bool {
			if dotImportedListingKitHTTPAPI {
				call, ok := node.(*ast.CallExpr)
				if ok {
					ident, ok := call.Fun.(*ast.Ident)
					if ok && ident.Name == "BuildSourceAccountRepository" {
						t.Errorf("%s dot-imports listingkit/httpapi to build source-account repositories; use the source-account bootstrap owner", path)
					}
				}
			}
			selector, ok := node.(*ast.SelectorExpr)
			if !ok || selector.Sel == nil || selector.Sel.Name != "BuildSourceAccountRepository" {
				return true
			}
			ident, ok := selector.X.(*ast.Ident)
			if ok {
				if _, imported := listingKitHTTPAPINames[ident.Name]; imported {
					t.Errorf("%s still builds source-account repositories through listingkit/httpapi; use the source-account bootstrap owner", path)
				}
			}
			return true
		})
	}
}

func TestSheinLoginReadsIdentityFromNeutralContext(t *testing.T) {
	root := filepath.Join("..", "internal", "sheinlogin")
	// Current execution authorization uses the neutral verified identity context.
	for _, name := range []string{"authorization.go"} {
		path := filepath.Join(root, name)
		source, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		file, err := parser.ParseFile(token.NewFileSet(), path, source, parser.ImportsOnly)
		if err != nil {
			t.Fatal(err)
		}
		importsAuthIdentity := false
		for _, imp := range file.Imports {
			if imp.Path.Value == `"task-processor/internal/authidentity"` {
				importsAuthIdentity = true
				break
			}
		}
		if !importsAuthIdentity {
			t.Errorf("%s must import internal/authidentity for verified identity context", path)
		}

		file, err = parser.ParseFile(token.NewFileSet(), path, source, 0)
		if err != nil {
			t.Fatal(err)
		}
		ast.Inspect(file, func(node ast.Node) bool {
			selector, ok := node.(*ast.SelectorExpr)
			if !ok || selector.Sel == nil ||
				(selector.Sel.Name != "AuthenticatedIdentityFromContext" && selector.Sel.Name != "AuthenticatedIdentity") {
				return true
			}
			ident, ok := selector.X.(*ast.Ident)
			if ok && ident.Name == "listingkit" {
				t.Errorf("%s reads verified identity through listingkit; use internal/authidentity", path)
			}
			return true
		})
	}
}

func TestListingKitIdentityReadersUseNeutralContext(t *testing.T) {
	paths := []string{
		filepath.Join("..", "internal", "authz", "httpapi", "route_authorization.go"),
		filepath.Join("..", "internal", "app", "httpapi", "server_auth.go"),
	}
	for _, path := range paths {
		source, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		file, err := parser.ParseFile(token.NewFileSet(), path, source, parser.ImportsOnly)
		if err != nil {
			t.Fatal(err)
		}
		importsAuthIdentity := false
		for _, imp := range file.Imports {
			if imp.Path.Value == `"task-processor/internal/authidentity"` {
				importsAuthIdentity = true
				break
			}
		}
		if !importsAuthIdentity {
			t.Errorf("%s must import internal/authidentity for verified identity context", path)
		}

		file, err = parser.ParseFile(token.NewFileSet(), path, source, 0)
		if err != nil {
			t.Fatal(err)
		}
		ast.Inspect(file, func(node ast.Node) bool {
			selector, ok := node.(*ast.SelectorExpr)
			if !ok || selector.Sel == nil ||
				(selector.Sel.Name != "AuthenticatedIdentityFromContext" && selector.Sel.Name != "AuthenticatedIdentity") {
				return true
			}
			ident, ok := selector.X.(*ast.Ident)
			if ok && ident.Name == "listingkit" {
				t.Errorf("%s reads verified identity through listingkit; use internal/authidentity", path)
			}
			return true
		})
	}
}

func TestZitadelAuthMiddlewareWritesNeutralIdentityContext(t *testing.T) {
	path := filepath.Join("..", "internal", "authruntime", "zitadel", "middleware.go")
	source, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	file, err := parser.ParseFile(token.NewFileSet(), path, source, parser.ImportsOnly)
	if err != nil {
		t.Fatal(err)
	}
	importsAuthIdentity := false
	for _, imp := range file.Imports {
		if imp.Path.Value == `"task-processor/internal/authidentity"` {
			importsAuthIdentity = true
			break
		}
	}
	if !importsAuthIdentity {
		t.Fatalf("%s must import internal/authidentity for verified identity projection", path)
	}

	file, err = parser.ParseFile(token.NewFileSet(), path, source, 0)
	if err != nil {
		t.Fatal(err)
	}
	ast.Inspect(file, func(node ast.Node) bool {
		selector, ok := node.(*ast.SelectorExpr)
		if !ok || selector.Sel == nil {
			return true
		}
		if selector.Sel.Name != "WithAuthenticatedIdentity" && selector.Sel.Name != "AuthenticatedIdentity" {
			return true
		}
		ident, ok := selector.X.(*ast.Ident)
		if ok && ident.Name == "listingkit" {
			t.Errorf("%s projects verified identity through listingkit; use internal/authidentity", path)
		}
		return true
	})
}

func TestZitadelAuthRuntimeDoesNotImportListingKit(t *testing.T) {
	assertNoBannedImportPrefixes(t, filepath.Join("..", "internal", "authruntime", "zitadel"), []string{
		"task-processor/internal/listingkit",
	}, nil)
}

func TestListingKitAuthenticatedIdentityCompatibilityFacadeIsRetired(t *testing.T) {
	index, err := loadGoFileIndex(filepath.Join("..", "internal", "listingkit"), "")
	if err != nil {
		t.Fatal(err)
	}
	for path, facts := range index.files {
		if strings.HasSuffix(filepath.Base(path), "_test.go") {
			continue
		}
		declarations, err := findListingKitAuthenticatedIdentityDeclarations(path, facts.source)
		if err != nil {
			t.Fatal(err)
		}
		for _, declaration := range declarations {
			t.Errorf("%s", declaration)
		}
	}

	index, err = loadGoFileIndex(filepath.Join("..", "internal"), "")
	if err != nil {
		t.Fatal(err)
	}
	for path, facts := range index.files {
		if strings.HasSuffix(filepath.Base(path), "_test.go") {
			continue
		}
		violations, err := findListingKitAuthenticatedIdentityImports(path, facts.source)
		if err != nil {
			t.Fatal(err)
		}
		for _, violation := range violations {
			t.Errorf("%s", violation)
		}
	}
}

func findListingKitAuthenticatedIdentityDeclarations(path string, source []byte) ([]string, error) {
	file, err := parser.ParseFile(token.NewFileSet(), path, source, 0)
	if err != nil {
		return nil, err
	}

	retiredNames := map[string]struct{}{
		"AuthenticatedIdentity":            {},
		"WithAuthenticatedIdentity":        {},
		"AuthenticatedIdentityFromContext": {},
	}
	var violations []string
	for _, declaration := range file.Decls {
		switch declaration := declaration.(type) {
		case *ast.FuncDecl:
			if declaration.Recv == nil {
				if _, retired := retiredNames[declaration.Name.Name]; retired {
					violations = append(violations, path+": declares retired identity API "+declaration.Name.Name)
				}
			}
		case *ast.GenDecl:
			for _, specification := range declaration.Specs {
				switch specification := specification.(type) {
				case *ast.TypeSpec:
					if _, retired := retiredNames[specification.Name.Name]; retired {
						violations = append(violations, path+": declares retired identity API "+specification.Name.Name)
					}
				case *ast.ValueSpec:
					for _, name := range specification.Names {
						if _, retired := retiredNames[name.Name]; retired {
							violations = append(violations, path+": declares retired identity API "+name.Name)
						}
					}
				}
			}
		}
	}
	return violations, nil
}

func TestAuthenticatedIdentityRootImportsStayRestricted(t *testing.T) {
	index, err := loadGoFileIndex(filepath.Join("..", "internal"), "")
	if err != nil {
		t.Fatal(err)
	}
	for path, facts := range index.files {
		if strings.HasSuffix(filepath.Base(path), "_test.go") {
			continue
		}
		violations, err := findListingKitAuthenticatedIdentityImports(path, facts.source)
		if err != nil {
			t.Fatal(err)
		}
		for _, violation := range violations {
			t.Errorf("%s", violation)
		}
	}
}

func TestAuthenticatedIdentityRootImportScannerResolvesAliases(t *testing.T) {
	violations, err := findListingKitAuthenticatedIdentityImports("synthetic.go", []byte(`package synthetic

import legacylistingkit "task-processor/internal/listingkit"

func readIdentity(ctx any) {
		legacylistingkit.AuthenticatedIdentityFromContext(ctx)
}
`))
	if err != nil {
		t.Fatal(err)
	}
	if len(violations) != 1 {
		t.Fatalf("violations = %v, want one alias violation", violations)
	}
}

func TestListingKitAuthenticatedIdentityDeclarationScannerRejectsMovedFacade(t *testing.T) {
	violations, err := findListingKitAuthenticatedIdentityDeclarations("synthetic.go", []byte(`package listingkit

type AuthenticatedIdentity struct{}

var WithAuthenticatedIdentity = func() {}

func AuthenticatedIdentityFromContext() {}
`))
	if err != nil {
		t.Fatal(err)
	}
	if len(violations) != 3 {
		t.Fatalf("violations = %v, want three retired identity declarations", violations)
	}
}

func findListingKitAuthenticatedIdentityImports(path string, source []byte) ([]string, error) {
	file, err := parser.ParseFile(token.NewFileSet(), path, source, 0)
	if err != nil {
		return nil, err
	}

	listingKitNames := make(map[string]struct{})
	var violations []string
	for _, imp := range file.Imports {
		importPath := strings.Trim(imp.Path.Value, `"`)
		if importPath != "task-processor/internal/listingkit" {
			continue
		}
		if imp.Name != nil {
			switch imp.Name.Name {
			case "_":
				continue
			case ".":
				violations = append(violations, path+": dot-imports internal/listingkit; identity context must use internal/authidentity")
				continue
			default:
				listingKitNames[imp.Name.Name] = struct{}{}
			}
			continue
		}
		listingKitNames["listingkit"] = struct{}{}
	}

	ast.Inspect(file, func(node ast.Node) bool {
		selector, ok := node.(*ast.SelectorExpr)
		if !ok || selector.Sel == nil {
			return true
		}
		switch selector.Sel.Name {
		case "AuthenticatedIdentity", "AuthenticatedIdentityFromContext", "WithAuthenticatedIdentity":
		default:
			return true
		}
		ident, ok := selector.X.(*ast.Ident)
		if !ok {
			return true
		}
		if _, ok := listingKitNames[ident.Name]; ok {
			violations = append(violations, path+": imports internal/listingkit for authenticated identity; use internal/authidentity")
		}
		return true
	})
	return violations, nil
}

func TestSourceHandoffLegacyHTTPImportsStayRetiredAcrossBuildTargets(t *testing.T) {
	t.Parallel()
	index, err := loadGoFileIndex(filepath.Join("..", "internal"), "")
	if err != nil {
		t.Fatal(err)
	}

	for path, facts := range index.files {
		if strings.HasSuffix(filepath.Base(path), "_test.go") {
			continue
		}
		if _, ok := facts.imports[`"task-processor/internal/productenrich/httpapi/sourcea1688"`]; ok {
			t.Errorf("%s imports retired source-handoff HTTP owner; use the canonical product source-handoff package", path)
		}
	}
}

func TestPortsManagementAPIPackageIsRetired(t *testing.T) {
	index, err := loadGoFileIndex(filepath.Join("..", "internal"), "")
	if err != nil {
		t.Fatal(err)
	}
	for path, facts := range index.files {
		if _, ok := facts.imports[`"task-processor/internal/ports/managementapi"`]; ok {
			t.Fatalf("%s imports retired managementapi compatibility package; use listingadmin/local runtime types instead", path)
		}
	}
}

func TestGlobalPortsFacadePackageStaysRetired(t *testing.T) {
	path := filepath.Join("..", "internal", "ports", "contracts.go")
	if _, err := os.Stat(path); err == nil {
		t.Fatalf("%s still exists; use current owner contracts; neither retired ports root may be revived", path)
	} else if !os.IsNotExist(err) {
		t.Fatalf("stat %s: %v", path, err)
	}

	if _, err := os.Stat(filepath.Join("..", "internal", "app", "ports")); !os.IsNotExist(err) {
		t.Fatalf("retired app/ports root must be absent: %v", err)
	}

	index, err := loadGoFileIndex(filepath.Join("..", "internal"), "")
	if err != nil {
		t.Fatal(err)
	}
	for path, facts := range index.files {
		for literal := range facts.imports {
			imported, err := decodeGoImportPath(literal)
			if err != nil {
				t.Fatalf("decode import in %s: %v", path, err)
			}
			if importMatchesPrefix(imported, "task-processor/internal/ports") ||
				importMatchesPrefix(imported, "task-processor/internal/app/ports") {
				t.Fatalf("%s imports retired ports facade %s; use current owner contracts; neither retired root may be revived", path, imported)
			}
		}
	}
}

func TestLegacyResiliencePackageStaysRetired(t *testing.T) {
	path := filepath.Join("..", "internal", "pkg", "resilience")
	if _, err := os.Stat(path); err == nil {
		t.Fatalf("%s still exists; use internal/shared/resilience for runtime-neutral resilience behavior", path)
	} else if !os.IsNotExist(err) {
		t.Fatalf("stat %s: %v", path, err)
	}
}

func TestListingAdminCompatibilityDoesNotExposeTaskStatusAdapters(t *testing.T) {
	path := filepath.Join("..", "internal", "listingadmin", "management_api_types.go")
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("retired numeric DTO production owner remains: %v", err)
	}
}

func TestListingAdminCompatibilityDoesNotExposeImportTaskUpdateDTO(t *testing.T) {
	path := filepath.Join("..", "internal", "listingadmin", "management_api_types.go")
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("retired numeric DTO production owner remains: %v", err)
	}
}

func TestListingAdminCompatibilityDoesNotExposeImportTaskResponseDTO(t *testing.T) {
	path := filepath.Join("..", "internal", "listingadmin", "management_api_types.go")
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("retired numeric DTO production owner remains: %v", err)
	}
}

func TestListingAdminDispatchEventCompatibilityAliasesStayRetired(t *testing.T) {
	root := filepath.Join("..", "internal", "listingadmin")
	entries, err := os.ReadDir(root)
	if err != nil {
		t.Fatalf("read %s: %v", root, err)
	}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".go") {
			continue
		}
		path := filepath.Join(root, entry.Name())
		content, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read %s: %v", path, err)
		}
		for _, phrase := range []string{
			"type DispatchEventFilter =",
			"type DispatchEventSummaryRow =",
			"type DispatchEventListRow =",
		} {
			if strings.Contains(string(content), phrase) {
				t.Fatalf("%s recreates retired dispatch event alias %q", path, phrase)
			}
		}
	}
}

func TestAppTaskStatusDTOAdapterIsRetired(t *testing.T) {
	path := filepath.Join("..", "internal", "app", "taskstatusdto")
	if _, err := os.Stat(path); err == nil {
		t.Fatalf("%s still exists; keep the retired task status DTO owner absent", path)
	} else if !os.IsNotExist(err) {
		t.Fatalf("stat %s: %v", path, err)
	}
}

func TestListingKitNonAPISheinImportsStayAllowlisted(t *testing.T) {
	root := filepath.Join("..", "internal", "listingkit")
	allowedImports := map[string]map[string]struct{}{
		`"task-processor/internal/shein/client"`: {
			filepath.Clean(filepath.Join(root, "service_shein_categories.go")):              {},
			filepath.Clean(filepath.Join(root, "service_shein_category_api_helpers.go")):    {},
			filepath.Clean(filepath.Join(root, "service_submit_remote_context_helpers.go")): {},
			filepath.Clean(filepath.Join(root, "service_submit_store_context.go")):          {},
			filepath.Clean(filepath.Join(root, "shein_runtime.go")):                         {},
			filepath.Clean(filepath.Join(root, "service_preview_test.go")):                  {},
			filepath.Clean(filepath.Join(root, "service_revision_test.go")):                 {},
			filepath.Clean(filepath.Join(root, "service_shein_store_client.go")):            {},
			filepath.Clean(filepath.Join(root, "service_shein_store_client_test.go")):       {},
			filepath.Clean(filepath.Join(root, "shein_admin_service.go")):                   {},
			filepath.Clean(filepath.Join(root, "shein_admin_service_support.go")):           {},
			filepath.Clean(filepath.Join(root, "service_admin_wiring_support.go")):          {},
			filepath.Clean(filepath.Join(root, "httpapi", "bootstrap_test.go")):             {},
			filepath.Clean(filepath.Join(root, "service_submit_context_resolver.go")):       {},
		},
		`"task-processor/internal/shein/store"`: {
			filepath.Clean(filepath.Join(root, "shein_submit_payload.go")):              {},
			filepath.Clean(filepath.Join(root, "shein_submit_payload_site_support.go")): {},
		},
		`"task-processor/internal/shein/submitprep"`: {
			filepath.Clean(filepath.Join(root, "shein_submit_test_helpers_test.go")): {},
		},
		`"task-processor/internal/shein/category"`: {
			filepath.Clean(filepath.Join(root, "httpapi", "shein_category_selector_adapter.go")): {},
		},
	}

	index, err := loadGoFileIndex(root, "")
	if err != nil {
		t.Fatal(err)
	}
	for path, facts := range index.files {
		for bannedImport, allowedFiles := range allowedImports {
			if _, ok := facts.imports[bannedImport]; !ok {
				continue
			}
			if _, ok := allowedFiles[path]; !ok {
				t.Errorf("%s imports %s; keep non-API SHEIN dependencies isolated to the current allowlisted adapter files", path, bannedImport)
			}
		}
	}
}

func TestListingKitAmazonListingImportsStayAllowlisted(t *testing.T) {
	root := filepath.Join("..", "internal", "listingkit")
	allowedFiles := map[string]struct{}{
		filepath.Clean(filepath.Join(root, "assembler_test.go")):                         {},
		filepath.Clean(filepath.Join(root, "export_model.go")):                           {},
		filepath.Clean(filepath.Join(root, "interfaces.go")):                             {},
		filepath.Clean(filepath.Join(root, "interfaces_dependencies.go")):                {},
		filepath.Clean(filepath.Join(root, "model_result.go")):                           {},
		filepath.Clean(filepath.Join(root, "model_result_support.go")):                   {},
		filepath.Clean(filepath.Join(root, "preview_model.go")):                          {},
		filepath.Clean(filepath.Join(root, "platform_payload_input_models.go")):          {},
		filepath.Clean(filepath.Join(root, "platform_payload_models_export.go")):         {},
		filepath.Clean(filepath.Join(root, "platform_payload_models_preview_amazon.go")): {},
		filepath.Clean(filepath.Join(root, "preview_platform_payload_test.go")):          {},
		filepath.Clean(filepath.Join(root, "service.go")):                                {},
		filepath.Clean(filepath.Join(root, "service_defaults.go")):                       {},
	}

	index, err := loadGoFileIndex(root, "")
	if err != nil {
		t.Fatal(err)
	}
	for path, facts := range index.files {
		if _, ok := facts.imports[`"task-processor/internal/amazonlisting"`]; !ok {
			continue
		}
		if _, ok := allowedFiles[path]; !ok {
			t.Errorf("%s imports task-processor/internal/amazonlisting; keep amazonlisting dependencies isolated to current facade/result bridge files", path)
		}
	}
}

func TestListingPreviewImportsStayPlatformNeutralAcrossBuildTargets(t *testing.T) {
	assertLegacyListingPreviewDownloadSourcesRetired(t)
}

func TestPublishingSheinNonAPISheinImportsStayAllowlisted(t *testing.T) {
	root := filepath.Join("..", "internal", "marketplace", "shein")
	allowedImports := map[string]map[string]struct{}{
		`"task-processor/internal/shein/client"`:     {},
		`"task-processor/internal/shein/content"`:    {},
		`"task-processor/internal/shein/context"`:    {},
		`"task-processor/internal/shein/product"`:    {},
		`"task-processor/internal/shein/store"`:      {},
		`"task-processor/internal/shein/category"`:   {},
		`"task-processor/internal/shein/submitprep"`: {},
		`"task-processor/internal/shein/publish"`:    {},
	}

	index, err := loadGoFileIndex(root, "")
	if err != nil {
		t.Fatal(err)
	}
	for path, facts := range index.files {
		for bannedImport, allowedFiles := range allowedImports {
			if _, ok := facts.imports[bannedImport]; !ok {
				continue
			}
			if _, ok := allowedFiles[path]; !ok {
				t.Errorf("%s imports %s; keep publishing/shein direct dependencies on legacy shein implementation packages isolated to current allowlisted adapter files", path, bannedImport)
			}
		}
	}

	managedRoot := filepath.Join("..", "internal", "publishing", "sheinmanaged")
	managedAllowedImports := map[string]map[string]struct{}{
		`"task-processor/internal/shein/category"`:      {},
		`"task-processor/internal/shein/client"`:        {},
		`"task-processor/internal/shein/managedclient"`: {},
	}
	managedIndex, err := loadGoFileIndex(filepath.Join("..", "internal"), "")
	if err != nil {
		t.Fatal(err)
	}
	for path, facts := range managedIndex.files {
		if !strings.HasPrefix(path, managedRoot+string(os.PathSeparator)) {
			continue
		}
		for bannedImport, allowedFiles := range managedAllowedImports {
			if _, ok := facts.imports[bannedImport]; !ok {
				continue
			}
			if _, ok := allowedFiles[path]; !ok {
				t.Errorf("%s imports %s; keep publishing/sheinmanaged direct dependencies on legacy shein category implementation isolated to adapter files", path, bannedImport)
			}
		}
	}
}

func TestPublishingSheinManagedAPIImportsStayAllowlisted(t *testing.T) {
	managedRoot := filepath.Join("..", "internal", "publishing", "sheinmanaged")
	allowedFiles := map[string]struct{}{}

	index, err := loadGoFileIndex(filepath.Join("..", "internal"), "")
	if err != nil {
		t.Fatal(err)
	}
	for path, facts := range index.files {
		if !strings.HasPrefix(path, managedRoot+string(os.PathSeparator)) {
			continue
		}
		if strings.HasSuffix(filepath.Base(path), "_test.go") || pathAllowed(path, allowedFiles) {
			continue
		}
		for quotedImport := range facts.imports {
			importPath := strings.Trim(quotedImport, `"`)
			if importMatchesPrefix(importPath, "task-processor/internal/shein/api") {
				t.Errorf("%s imports %s; keep publishing/sheinmanaged concrete SHEIN API clients isolated to current builder and API factory files", path, importPath)
			}
		}
	}
}

func TestPublishingSheinManagedRetiredManagementImportsStayBlocked(t *testing.T) {
	root := filepath.Join("..", "internal", "publishing", "sheinmanaged")
	allowedFiles := map[string]struct{}{}

	index, err := loadGoFileIndex(filepath.Join("..", "internal"), "")
	if err != nil {
		t.Fatal(err)
	}

	for path, facts := range index.files {
		if !strings.HasPrefix(path, root+string(os.PathSeparator)) {
			continue
		}
		if pathAllowed(path, allowedFiles) {
			continue
		}
		for quotedImport := range facts.imports {
			importPath := strings.Trim(quotedImport, `"`)
			if importMatchesPrefix(importPath, "task-processor/internal/infra/clients/management") {
				t.Errorf("%s imports %s; keep publishing/sheinmanaged management dependencies limited to current retirement seams and prefer in-repository database/repository access for new business data", path, importPath)
			}
		}
	}
}

func TestPublishingSheinRuntimeBaseAPIClientAliasStaysRetired(t *testing.T) {
	root := filepath.Join("..", "internal", "marketplace", "shein")
	index, err := loadGoFileIndex(root, "")
	if err != nil {
		t.Fatal(err)
	}
	for path := range index.files {
		content, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read %s: %v", path, err)
		}
		if strings.Contains(string(content), "RuntimeBaseAPIClient") {
			t.Fatalf("%s references RuntimeBaseAPIClient; use shein/client.BaseAPIClient directly in the runtime API factory", path)
		}
	}
}

func TestPublishingSheinOpenAIImportsStayAllowlisted(t *testing.T) {
	root := filepath.Join("..", "internal", "marketplace", "shein")
	allowedFiles := map[string]struct{}{}

	index, err := loadGoFileIndex(root, "")
	if err != nil {
		t.Fatal(err)
	}

	for path, facts := range index.files {
		if pathAllowed(path, allowedFiles) {
			continue
		}
		if _, ok := facts.imports[`"task-processor/internal/integration/openai"`]; ok {
			t.Errorf("%s imports OpenAI adapter directly; keep publishing/shein concrete OpenAI dependencies limited to current inference and runtime resolver seams", path)
		}
	}
}

func TestPublishingSheinSubmitPrepUsesOnlySensitiveWordAdapter(t *testing.T) {
	adapter := filepath.Join("..", "internal", "marketplace", "shein", "draft", "sensitive_word_adapter.go")
	if _, err := os.Lstat(adapter); !os.IsNotExist(err) {
		t.Fatalf("retired sensitive adapter must be absent: %v", err)
	}
	root := filepath.Join("..", "internal", "marketplace", "shein", "draft")
	file, err := parser.ParseFile(token.NewFileSet(), filepath.Join(root, "local_cleanup.go"), nil, 0)
	if err != nil {
		t.Fatal(err)
	}
	ownerFound, callFound := false, false
	for _, imp := range file.Imports {
		imported, err := decodeGoImportPath(imp.Path.Value)
		if err != nil {
			t.Fatal(err)
		}
		if imp.Name != nil {
			t.Fatal("cleanup imports must use their actual owner names")
		}
		switch imported {
		case "task-processor/internal/marketplace/shein/publishing":
			ownerFound = true
		case "strings", "unicode", "task-processor/internal/marketplace/shein/offlinepackage":
		default:
			t.Errorf("cleanup must not inject legacy/runtime/logger dependencies: %s", imported)
		}
	}
	ast.Inspect(file, func(node ast.Node) bool {
		switch n := node.(type) {
		case *ast.Ident:
			switch n.Name {
			case "ensureSanitizerContext", "newSheinSensitiveWordSanitizer", "sheinSensitiveWordSanitizer", "CleanupContext", "SensitiveWordService":
				t.Errorf("cleanup retains old service/context/factory: %s", n.Name)
			}
		case *ast.InterfaceType, *ast.FuncLit, *ast.GoStmt:
			t.Error("cleanup must not inject an interface/callback or start runtime work")
		case *ast.CallExpr:
			selector, ok := n.Fun.(*ast.SelectorExpr)
			if !ok || selector.Sel.Name != "SanitizeDisplayText" {
				break
			}
			pkg, ok := selector.X.(*ast.Ident)
			if !ok || pkg.Name != "publishing" || pkg.Obj != nil || len(n.Args) != 2 || n.Ellipsis.IsValid() {
				t.Fatal("cleanup must directly call its unshadowed publishing pure owner")
			}
			callFound = true
		}
		return true
	})
	if !ownerFound || !callFound {
		t.Fatal("current cleanup must directly call publishing.SanitizeDisplayText")
	}
	for relative, function := range map[string]string{"local_copy.go": "sanitizeSheinListingCopy", "local_package.go": "sanitizeLocalDraft"} {
		caller, err := parser.ParseFile(token.NewFileSet(), filepath.Join(root, relative), nil, 0)
		if err != nil {
			t.Fatal(err)
		}
		owner := ""
		for _, imp := range caller.Imports {
			imported, err := decodeGoImportPath(imp.Path.Value)
			if err != nil {
				t.Fatal(err)
			}
			if imported == "task-processor/internal/marketplace/shein/publishing" {
				owner = "publishing"
				if imp.Name != nil {
					owner = imp.Name.Name
				}
				if owner == "." || owner == "_" {
					t.Fatal("current pure owner must be explicit")
				}
			}
		}
		if owner == "" {
			t.Fatalf("%s lacks the current publishing owner", relative)
		}
		found := false
		ast.Inspect(caller, func(node ast.Node) bool {
			call, ok := node.(*ast.CallExpr)
			if !ok {
				return true
			}
			name, ok := call.Fun.(*ast.Ident)
			if !ok || name.Name != function {
				return true
			}
			if len(call.Args) != 2 || call.Ellipsis.IsValid() {
				t.Fatal("production cleanup must pass only its value input")
			}
			input, ok := call.Args[1].(*ast.CompositeLit)
			if !ok {
				t.Fatal("production cleanup input must be explicit")
			}
			typ, ok := input.Type.(*ast.SelectorExpr)
			if !ok {
				t.Fatal("production cleanup input must belong to publishing")
			}
			pkg, ok := typ.X.(*ast.Ident)
			if !ok || pkg.Name != owner || pkg.Obj != nil || typ.Sel.Name != "DisplayTextCleanupInput" || len(input.Elts) != 1 {
				t.Fatal("production cleanup must not infer brand or runtime inputs")
			}
			word, ok := input.Elts[0].(*ast.KeyValueExpr)
			if !ok {
				t.Fatal("production cleanup Words must be explicit")
			}
			key, ok := word.Key.(*ast.Ident)
			if !ok || key.Name != "Words" {
				t.Fatal("production cleanup must keep empty brand inputs")
			}
			defaults, ok := word.Value.(*ast.CallExpr)
			if !ok || len(defaults.Args) != 0 || defaults.Ellipsis.IsValid() {
				t.Fatal("production cleanup must explicitly request default word values")
			}
			fun, ok := defaults.Fun.(*ast.SelectorExpr)
			if !ok {
				t.Fatal("default word values must come from publishing")
			}
			pkg, ok = fun.X.(*ast.Ident)
			if !ok || pkg.Name != owner || pkg.Obj != nil || fun.Sel.Name != "DefaultDisplayTextWords" {
				t.Fatal("default word owner must not be aliased or shadowed")
			}
			found = true
			return true
		})
		if !found {
			t.Fatalf("%s must keep its actual explicit default cleanup call", relative)
		}
	}
}
func TestProductEnrichCanonicalImportsStayRetiredAcrossBuildTargets(t *testing.T) {
	for _, tc := range []struct {
		name string
		root string
	}{
		{name: "offlinepackage", root: filepath.Join("..", "internal", "marketplace", "shein", "offlinepackage")},
		{name: "draft", root: filepath.Join("..", "internal", "marketplace", "shein", "draft")},
		{name: "publishing", root: filepath.Join("..", "internal", "marketplace", "shein", "publishing")},
		{name: "catalog", root: filepath.Join("..", "internal", "product", "catalog")},
	} {
		t.Run(tc.name, func(t *testing.T) {
			index, err := loadGoFileIndex(tc.root, "")
			if err != nil {
				t.Fatal(err)
			}
			for path, facts := range index.files {
				if strings.HasSuffix(filepath.Base(path), "_test.go") {
					continue
				}
				if _, ok := facts.imports[`"task-processor/internal/productenrich"`]; ok {
					t.Errorf("%s imports task-processor/internal/productenrich; use canonical catalog/product contracts", path)
				}
			}
		})
	}
}

func TestListingKitImportDirectionStaysRetiredAcrossBuildTargets(t *testing.T) {
	root := filepath.Join("..", "internal", "listingkit")
	index, err := loadGoFileIndex(root, "")
	if err != nil {
		t.Fatal(err)
	}

	subdomains := map[string]struct{}{
		"generation": {},
		"submission": {},
		"workflow":   {},
		"workspace":  {},
	}
	for path, facts := range index.files {
		if strings.HasSuffix(filepath.Base(path), "_test.go") {
			continue
		}
		rel, err := filepath.Rel(filepath.Clean(root), path)
		if err != nil {
			t.Fatalf("rel %s to %s: %v", path, root, err)
		}
		parts := strings.Split(filepath.ToSlash(rel), "/")
		if len(parts) > 1 {
			if _, ok := subdomains[parts[0]]; ok {
				if _, ok := facts.imports[`"task-processor/internal/listingkit"`]; ok {
					t.Errorf("%s imports task-processor/internal/listingkit; ListingKit subdomains must stay independent of the root facade", path)
				}
			}
			continue
		}
		if _, ok := facts.imports[`"task-processor/internal/workspace/shein"`]; ok {
			t.Errorf("%s imports task-processor/internal/workspace/shein; root ListingKit files must use the compatibility bridge", path)
		}
	}
}

func TestListingKitRootSheinHelpersStayAllowlisted(t *testing.T) {
	root := filepath.Join("..", "internal", "listingkit")
	allowed := map[string]struct{}{
		"shein_build_validation.go":                           {},
		"shein_admin_service.go":                              {},
		"shein_admin_service_support.go":                      {},
		"shein_color_block_image.go":                          {},
		"shein_default_store_resolution.go":                   {},
		"shein_final_draft.go":                                {},
		"shein_final_draft_support.go":                        {},
		"shein_image_regeneration.go":                         {},
		"shein_image_regeneration_model.go":                   {},
		"shein_image_strategy.go":                             {},
		"shein_pricing.go":                                    {},
		"shein_repair_center.go":                              {},
		"shein_repair_clone_support.go":                       {},
		"shein_repair_revision_support.go":                    {},
		"shein_repair_support.go":                             {},
		"shein_repair_validation_support.go":                  {},
		"shein_resolution_cache.go":                           {},
		"shein_review_state.go":                               {},
		"shein_runtime.go":                                    {},
		"shein_sale_attribute_policy.go":                      {},
		"shein_size_reference_images.go":                      {},
		"shein_settings.go":                                   {},
		"shein_store_resolution_presentation.go":              {},
		"shein_studio_ai_product_images.go":                   {},
		"shein_studio_image_helpers.go":                       {},
		"shein_studio_images.go":                              {},
		"shein_studio_size_reference_images.go":               {},
		"shein_studio_variant_coverage.go":                    {},
		"shein_studio_variant_images.go":                      {},
		"shein_submission_events.go":                          {},
		"shein_submit_debug.go":                               {},
		"shein_submit_images.go":                              {},
		"shein_submit_payload.go":                             {},
		"shein_submit_image_upload_support.go":                {},
		"shein_submit_payload_image_support.go":               {},
		"shein_submit_payload_site_support.go":                {},
		"shein_submit_payload_supplier_validation_support.go": {},
		"shein_submit_readiness.go":                           {},
		"shein_submit_readiness_checks_support.go":            {},
		"shein_submit_readiness_guidance_support.go":          {},
		"shein_submit_readiness_status_support.go":            {},
		"shein_submit_readiness_types.go":                     {},
		"shein_submit_readiness_state.go":                     {},
		"shein_submit_retry.go":                               {},
		"shein_submit_sku_normalization.go":                   {},
		"shein_submit_sku_pricing_support.go":                 {},
		"shein_submit_sku_style_support.go":                   {},
		"shein_submit_sku_variant_support.go":                 {},
		"shein_submit_state.go":                               {},
		"shein_template_matcher.go":                           {},
		"shein_variant_image_coverage.go":                     {},
		"shein_workspace_editor_bridge.go":                    {},
		"shein_workspace_inspection_bridge.go":                {},
		"shein_workspace_readiness_support.go":                {},
		"shein_workspace_repair_bridge.go":                    {},
		"shein_workspace_revision_bridge.go":                  {},
		"shein_workspace_submit_bridge.go":                    {},
		"shein_workspace_types_bridge.go":                     {},
	}

	entries, err := os.ReadDir(root)
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if !strings.HasPrefix(name, "shein_") || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		if _, ok := allowed[name]; !ok {
			t.Errorf("%s is a new root-level SHEIN helper; move new domain logic into publishing/shein, workspace/shein, or a listingkit subpackage instead", filepath.Join(root, name))
		}
	}
}

func TestListingKitRootServiceSubmitFilesStayAllowlisted(t *testing.T) {
	root := filepath.Join("..", "internal", "listingkit")
	allowed := map[string]struct{}{
		"service_submit.go":                                {},
		"service_submit_action_normalization_helper.go":    {},
		"service_submit_collaborators.go":                  {},
		"service_submit_contracts.go":                      {},
		"service_submit_context_resolver.go":               {},
		"service_submit_bindings.go":                       {},
		"service_submit_collaborator_stages.go":            {},
		"service_submit_default_action.go":                 {},
		"service_submit_default_action_resolver_helper.go": {},
		"service_submit_direct.go":                         {},
		"service_submit_entrypoint.go":                     {},
		"service_submit_lease_helper.go":                   {},
		"service_submit_remote_context_helpers.go":         {},
		"service_submit_routing.go":                        {},
		"service_submit_runtime_context.go":                {},
		"service_submit_runtime_context_resolver.go":       {},
		"service_submit_managed_wiring_support.go":         {},
		"service_submit_settings_resolution.go":            {},
		"service_submit_settings_resolution_helpers.go":    {},
		"service_submit_shared.go":                         {},
		"service_submit_store_context.go":                  {},
		"service_submit_task_identity_helper.go":           {},
		"service_submit_recovery.go":                       {},
		"service_submit_temporal_adapter.go":               {},
		"service_submit_temporal_wiring_support.go":        {},
		"service_submit_temporal_task_loader_helper.go":    {},
		"service_submit_warehouse_selection_helper.go":     {},
		"service_submit_wiring.go":                         {},
		"service_submit_wiring_resolution_support.go":      {},
		"service_submit_wiring_support.go":                 {},
		"service_submit_workflow.go":                       {},
		"service_submit_workflow_entry_helpers.go":         {},
	}

	entries, err := os.ReadDir(root)
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if !strings.HasPrefix(name, "service_submit") || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		if _, ok := allowed[name]; !ok {
			t.Errorf("%s is a new root-level submit service file; move new submit logic into internal/listingkit/submission or an existing thin facade file instead", filepath.Join(root, name))
		}
	}
}

func TestListingKitRootTaskSubmissionFilesStayAllowlisted(t *testing.T) {
	root := filepath.Join("..", "internal", "listingkit")
	allowed := map[string]struct{}{
		"task_submission_service.go":                         {},
		"task_submission_execution_images.go":                {},
		"task_submission_execution_normalize.go":             {},
		"task_submission_execution_product.go":               {},
		"task_submission_execution_remote.go":                {},
		"task_submission_execution_service.go":               {},
		"task_submission_failure_persistence_support.go":     {},
		"task_submission_payload_stage_support.go":           {},
		"task_submission_recovery_lease.go":                  {},
		"task_submission_recovery_remote.go":                 {},
		"task_submission_recovery_service.go":                {},
		"task_submission_recovery_service_remote_support.go": {},
		"task_submission_recovery_service_route_support.go":  {},
		"task_submission_refresh_mutation.go":                {},
		"task_submission_refresh_selection.go":               {},
		"task_submission_refresh_service.go":                 {},
		"task_submission_remote_completion_support.go":       {},
		"task_submission_remote_refresh_support.go":          {},
		"task_submission_remote_support.go":                  {},
		"task_submission_state_service.go":                   {},
		"task_submission_state_persistence_support.go":       {},
		"task_submission_success_persistence_support.go":     {},
	}

	entries, err := os.ReadDir(root)
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if !strings.HasPrefix(name, "task_submission") || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		if _, ok := allowed[name]; !ok {
			t.Errorf("%s is a new root-level task submission file; move new submission task logic into internal/listingkit/submission or an existing task submission facade file instead", filepath.Join(root, name))
		}
	}
}

func TestListingKitRootServiceGenerationFilesStayAllowlisted(t *testing.T) {
	root := filepath.Join("..", "internal", "listingkit")
	allowed := map[string]struct{}{
		"service_generation.go": {},
	}

	entries, err := os.ReadDir(root)
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if !strings.HasPrefix(name, "service_generation") || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		if _, ok := allowed[name]; !ok {
			t.Errorf("%s is a new root-level generation service file; move new generation logic into internal/listingkit/generation or an existing thin facade file instead", filepath.Join(root, name))
		}
	}
}

func TestListingKitRootGenerationFilesStayAllowlisted(t *testing.T) {
	root := filepath.Join("..", "internal", "listingkit")
	allowed := map[string]struct{}{
		"generation_action_keys.go":                          {},
		"generation_action_render_previews.go":               {},
		"generation_conditional_descriptor_support.go":       {},
		"generation_conditional_panel_update_support.go":     {},
		"generation_conditional_state.go":                    {},
		"generation_navigation_dispatch_merge.go":            {},
		"generation_navigation_dispatch_rules.go":            {},
		"generation_navigation_target_conditional.go":        {},
		"generation_navigation_target_dispatch_support.go":   {},
		"generation_navigation_target_identity_support.go":   {},
		"generation_navigation_target_identity.go":           {},
		"generation_overview.go":                             {},
		"generation_overview_action_support.go":              {},
		"generation_overview_filter_support.go":              {},
		"generation_overview_quality_support.go":             {},
		"generation_platform_cards.go":                       {},
		"generation_queue.go":                                {},
		"generation_queue_bundle_support.go":                 {},
		"generation_queue_list.go":                           {},
		"generation_queue_summary_support.go":                {},
		"generation_queue_tasks.go":                          {},
		"generation_recovery_summary_support.go":             {},
		"generation_recovery_summary.go":                     {},
		"generation_render_preview_capabilities.go":          {},
		"generation_resolved_action_summary.go":              {},
		"generation_review_delta.go":                         {},
		"generation_review_focus.go":                         {},
		"generation_review_navigation_target_clone_shape.go": {},
		"generation_review_navigation_target.go":             {},
		"generation_review_patch_support.go":                 {},
		"generation_review_patch.go":                         {},
		"generation_review_persistence.go":                   {},
		"generation_review_section_config.go":                {},
		"generation_review_section_support.go":               {},
		"generation_review_session_sections.go":              {},
		"generation_review_session_slot_support.go":          {},
		"generation_review_state.go":                         {},
		"generation_review_toolbar.go":                       {},
		"generation_scene_preset_summary.go":                 {},
		"generation_task_list.go":                            {},
	}

	entries, err := os.ReadDir(root)
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if !strings.HasPrefix(name, "generation_") || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		if _, ok := allowed[name]; !ok {
			t.Errorf("%s is a new root-level generation file; move new generation logic into internal/listingkit/generation or another narrower subpackage instead", filepath.Join(root, name))
		}
	}
}

func TestListingKitRootSheinWorkspaceBridgesDoNotImportWorkspaceDomainDirectly(t *testing.T) {
	root := filepath.Join("..", "internal", "listingkit")
	targetFiles := map[string]struct{}{
		"preview_builder_shein.go":             {},
		"preview_model.go":                     {},
		"final_review_support.go":              {},
		"model_task.go":                        {},
		"revision_apply_result.go":             {},
		"revision_applied_changes.go":          {},
		"revision_diff_preview.go":             {},
		"revision_diff_compare.go":             {},
		"revision_history_compare.go":          {},
		"revision_history_detail.go":           {},
		"revision_history_restore_draft.go":    {},
		"revision_interaction_presentation.go": {},
		"revision_restore_result.go":           {},
		"revision_restore_request.go":          {},
		"revision_success_shared.go":           {},
		"revision_success_payload.go":          {},
		"revision_validation.go":               {},
		"revision_validate_model.go":           {},
		"revision_workspace_bridge.go":         {},
		"shein_submit_readiness.go":            {},
		"shein_workspace_editor_bridge.go":     {},
		"shein_workspace_inspection_bridge.go": {},
		"shein_repair_support.go":              {},
		"shein_repair_center.go":               {},
		"shein_review_state.go":                {},
		"shein_workspace_readiness_support.go": {},
		"shein_build_validation.go":            {},
		"shein_workspace_repair_bridge.go":     {},
		"shein_workspace_revision_bridge.go":   {},
		"shein_workspace_submit_bridge.go":     {},
		"shein_workspace_types_bridge.go":      {},
		"submit_readiness_guidance_shein.go":   {},
		"submit_freshness_shein.go":            {},
		"submit_readiness_projection_shein.go": {},
		"submit_readiness_summary_shein.go":    {},
		"task_state_support.go":                {},
	}
	allowedImports := map[string]map[string]struct{}{
		`"task-processor/internal/workspace/shein"`: {},
	}

	index, err := loadGoFileIndex(root, "")
	if err != nil {
		t.Fatal(err)
	}
	for path, facts := range index.files {
		name := filepath.Base(path)
		if _, ok := targetFiles[name]; !ok {
			continue
		}
		for bannedImport, allowedFiles := range allowedImports {
			if _, ok := facts.imports[bannedImport]; !ok {
				continue
			}
			if _, ok := allowedFiles[path]; !ok {
				t.Errorf("%s imports %s; keep root shein workspace bridges as thin compatibility wrappers and move direct workspace domain wiring into internal/listingkit/workspace/shein", path, bannedImport)
			}
		}
	}
}

func TestListingKitSheinWorkspaceBridgeDoesNotImportLegacyWorkspaceDomain(t *testing.T) {
	root := filepath.Join("..", "internal", "listingkit", "workspace", "shein")
	if _, err := os.Stat(root); err != nil {
		if os.IsNotExist(err) {
			return
		}
		t.Fatal(err)
	}
	assertNoBannedImports(t, root, []string{
		`"task-processor/internal/workspace/shein"`,
	}, nil)
}

func TestRootBootstrapListingRuntimeDependenciesFacadeStaysRetired(t *testing.T) {
	path := filepath.Join("..", "internal", "app", "bootstrap", "consumer_dependencies.go")
	if _, err := os.Stat(path); err == nil {
		t.Fatalf("%s still exists; import bootstrap/listingruntime directly instead of reviving the root listing-runtime dependency facade", path)
	} else if !os.IsNotExist(err) {
		t.Fatalf("stat %s: %v", path, err)
	}
}

func TestListingRuntimeLocalCompatibilityFacadeStaysRetired(t *testing.T) {
	path := filepath.Join("..", "internal", "listingruntime", "local", "runtime.go")
	if _, err := os.Stat(path); err == nil {
		t.Fatalf("%s still exists; use explicit LocalRuntime and LocalDataProvider constructors instead of the local runtime compatibility facade", path)
	} else if !os.IsNotExist(err) {
		t.Fatalf("stat %s: %v", path, err)
	}
}

func TestProductSourceCompatibilityAliasStaysRetired(t *testing.T) {
	path := filepath.Join("..", "internal", "app", "ports", "platform.go")
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("retired app/ports source must be absent: %v", err)
	}

	index, err := loadGoFileIndex(filepath.Join("..", "internal"), "")
	if err != nil {
		t.Fatal(err)
	}
	for path := range index.files {
		content, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read %s: %v", path, err)
		}
		if strings.Contains(string(content), "ProductSource") {
			t.Fatalf("%s mentions ProductSource; use the current source owner contracts", path)
		}
	}
}

func TestAppRuntimeLegacyAppConfigFlagsStayRetired(t *testing.T) {
	roots := []string{
		filepath.Join("..", "internal", "app", "runtime", "listingkitschemamigrate"),
	}
	for _, root := range roots {
		index, err := loadGoFileIndex(root, "")
		if err != nil {
			t.Fatal(err)
		}
		for path := range index.files {
			content, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("read %s: %v", path, err)
			}
			for _, phrase := range []string{"AppConfig", "app-config"} {
				if strings.Contains(string(content), phrase) {
					t.Fatalf("%s mentions %q; use the canonical --config runtime option instead of retaining legacy app-config compatibility", path, phrase)
				}
			}
		}
	}
}

func TestConsumerProductFetcherBuilderStaysRetired(t *testing.T) {
	path := filepath.Join("..", "internal", "app", "consumer", "product_fetcher_builder.go")
	content, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return
	}
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	for _, phrase := range []string{
		"func BuildPlatformProductFetcher(",
		"func ResolvePlatformFetcherType(",
		"func PlatformUsesLocalFetcher(",
	} {
		if strings.Contains(string(content), phrase) {
			t.Fatalf("%s exposes %s; use the current Product sourcing port, not retired fetcher factories", path, phrase)
		}
	}
}

func TestBootstrapResourceFacadeStaysRetired(t *testing.T) {
	path := filepath.Join("..", "internal", "app", "bootstrap", "resource_factories.go")
	if _, err := os.Stat(path); err == nil {
		t.Fatalf("%s still exists; import bootstrap/resources directly instead of reviving the top-level resource facade", path)
	} else if !os.IsNotExist(err) {
		t.Fatalf("stat %s: %v", path, err)
	}
}

func TestBootstrapProcessorAndSchedulerFacadesStayRetired(t *testing.T) {
	for _, name := range []string{"processor_factories.go", "scheduler_factories.go"} {
		path := filepath.Join("..", "internal", "app", "bootstrap", name)
		if _, err := os.Stat(path); err == nil {
			t.Fatalf("%s still exists; import the dedicated bootstrap subpackage directly instead of reviving top-level forwarding facades", path)
		} else if !os.IsNotExist(err) {
			t.Fatalf("stat %s: %v", path, err)
		}
	}
}

func TestTaskStatusAdapterCallersUseRuntimeNamedConstructor(t *testing.T) {
	root := filepath.Join("..", "internal")

	index, err := loadGoFileIndex(root, "")
	if err != nil {
		t.Fatal(err)
	}

	for path, facts := range index.files {
		file, err := parser.ParseFile(token.NewFileSet(), path, facts.source, 0)
		if err != nil {
			t.Fatalf("parse %s: %v", path, err)
		}
		ast.Inspect(file, func(node ast.Node) bool {
			selector, ok := node.(*ast.SelectorExpr)
			if !ok || selector.Sel == nil || selector.Sel.Name != "NewManagementClientAdapter" {
				return true
			}
			ident, ok := selector.X.(*ast.Ident)
			if ok && ident.Name == "taskstatus" {
				t.Fatalf("%s calls taskstatus.NewManagementClientAdapter; use NewRuntimeTaskStatusAdapter outside compatibility wrappers", path)
			}
			return true
		})
	}
}

func TestTaskStatusCompatibilityPackageStaysRetired(t *testing.T) {
	path := filepath.Join("..", "internal", "taskstatus")
	if _, err := os.Stat(path); err == nil {
		t.Fatalf("%s still exists; the retired numeric task status owner must remain absent", path)
	} else if !os.IsNotExist(err) {
		t.Fatalf("stat %s: %v", path, err)
	}

	root := filepath.Join("..", "internal")
	index, err := loadGoFileIndex(root, "")
	if err != nil {
		t.Fatal(err)
	}
	for path, facts := range index.files {
		for quotedImport := range facts.imports {
			if strings.Trim(quotedImport, `"`) == "task-processor/internal/taskstatus" {
				t.Fatalf("%s imports retired internal/taskstatus compatibility package; do not revive the numeric task status owner", path)
			}
		}
	}
}

func TestListingAdminUsesManagementAPIPortAliases(t *testing.T) {
	root := filepath.Join("..", "internal", "listingadmin")
	allowedFiles := map[string]struct{}{
		filepath.Clean(filepath.Join(root, "management_compat.go")): {},
	}
	index, err := loadGoFileIndex(root, "")
	if err != nil {
		t.Fatal(err)
	}
	for path, facts := range index.files {
		if pathAllowed(path, allowedFiles) {
			continue
		}
		if _, ok := facts.imports[`"task-processor/internal/infra/clients/management/api"`]; ok {
			t.Fatalf("%s imports concrete management API DTOs; use listingadmin/local runtime types for compatibility DTOs", path)
		}
	}
}

func TestStateAndTaskRPCUseManagementAPIPortAliases(t *testing.T) {
	root := filepath.Join("..", "internal")
	retired := filepath.Join(root, "taskrpcapi")
	if _, err := os.Stat(retired); !os.IsNotExist(err) {
		t.Fatalf("retired Task RPC assembly must stay absent: %v", err)
	}
	assertNoBannedImportPrefixesBelow(t, root, retired, []string{"task-processor/internal/infra/clients/management/api"}, nil)
}

func TestAmazonServicesUseStoreAPIPort(t *testing.T) {
	// The old Services owner has retired; it must not return as a store wrapper.
	path := filepath.Join("..", "internal", "amazon", "model")
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("retired Amazon Services/context owner must stay absent: %v", err)
	}
}

func TestAmazonUsesManagementAPIPortAliases(t *testing.T) {
	root := filepath.Join("..", "internal", "amazon")
	if _, err := os.Lstat(root); !os.IsNotExist(err) {
		t.Fatalf("retired Amazon owner must stay absent: %v", err)
	}
	index, err := loadGoFileSubtreeIndex(filepath.Join("..", "internal"), root)
	if err != nil {
		t.Fatal(err)
	}
	for path, facts := range index.files {
		if _, ok := facts.imports[`"task-processor/internal/infra/clients/management/api"`]; ok {
			t.Fatalf("%s imports concrete management API DTOs; use listingadmin/local runtime types", path)
		}
	}
}

func TestTemuRuntimeErrorsUseCapabilityNames(t *testing.T) {
	// The runtime and its diagnostic files retired together; none may return.
	paths := []string{
		filepath.Join("..", "internal", "temu"),
		filepath.Join("..", "internal", "temu", "api", "client", "client.go"),
		filepath.Join("..", "internal", "temu", "api", "client", "auth.go"),
		filepath.Join("..", "internal", "temu", "api", "client", "auth_retry_handler.go"),
	}
	for _, path := range paths {
		if _, err := os.Lstat(path); !os.IsNotExist(err) {
			t.Fatalf("retired TEMU runtime artifact %s must stay absent: %v", path, err)
		}
	}
}

func TestTemuContextUsesManagementAPIPortAliases(t *testing.T) {
	root := filepath.Join("..", "internal", "temu", "context")
	index, err := loadGoFileSubtreeIndex(filepath.Join("..", "internal"), root)
	if err != nil {
		t.Fatal(err)
	}
	for path, facts := range index.files {
		if !strings.HasPrefix(path, root+string(os.PathSeparator)) {
			continue
		}
		if _, ok := facts.imports[`"task-processor/internal/infra/clients/management/api"`]; ok {
			t.Fatalf("%s imports concrete management API DTOs; use listingadmin/local runtime types", path)
		}
	}
}

func TestTemuFilterUsesManagementAPIPortAliases(t *testing.T) {
	root := filepath.Join("..", "internal", "temu", "filter")
	index, err := loadGoFileSubtreeIndex(filepath.Join("..", "internal"), root)
	if err != nil {
		t.Fatal(err)
	}
	for path, facts := range index.files {
		if !strings.HasPrefix(path, root+string(os.PathSeparator)) {
			continue
		}
		if _, ok := facts.imports[`"task-processor/internal/infra/clients/management/api"`]; ok {
			t.Fatalf("%s imports concrete management API DTOs; use listingadmin/local runtime types", path)
		}
	}
}

func TestTemuRulesUsesManagementAPIPortAliases(t *testing.T) {
	root := filepath.Join("..", "internal", "temu", "rules")
	index, err := loadGoFileSubtreeIndex(filepath.Join("..", "internal"), root)
	if err != nil {
		t.Fatal(err)
	}
	for path, facts := range index.files {
		if !strings.HasPrefix(path, root+string(os.PathSeparator)) {
			continue
		}
		if _, ok := facts.imports[`"task-processor/internal/infra/clients/management/api"`]; ok {
			t.Fatalf("%s imports concrete management API DTOs; use listingadmin/local runtime types", path)
		}
	}
}

func TestTemuHandlerBaseUsesManagementAPIPortAliases(t *testing.T) {
	root := filepath.Join("..", "internal", "temu", "handlerbase")
	index, err := loadGoFileSubtreeIndex(filepath.Join("..", "internal"), root)
	if err != nil {
		t.Fatal(err)
	}
	for path, facts := range index.files {
		if !strings.HasPrefix(path, root+string(os.PathSeparator)) {
			continue
		}
		if _, ok := facts.imports[`"task-processor/internal/infra/clients/management/api"`]; ok {
			t.Fatalf("%s imports concrete management API DTOs; use listingadmin/local runtime types", path)
		}
	}
}

func TestTemuStoreUsesManagementAPIPortAliases(t *testing.T) {
	root := filepath.Join("..", "internal", "temu", "store")
	index, err := loadGoFileSubtreeIndex(filepath.Join("..", "internal"), root)
	if err != nil {
		t.Fatal(err)
	}
	for path, facts := range index.files {
		if !strings.HasPrefix(path, root+string(os.PathSeparator)) {
			continue
		}
		if _, ok := facts.imports[`"task-processor/internal/infra/clients/management/api"`]; ok {
			t.Fatalf("%s imports concrete management API DTOs; use listingadmin/local runtime types", path)
		}
	}
}

func TestTemuSkuUsesManagementAPIPortAliases(t *testing.T) {
	root := filepath.Join("..", "internal", "temu", "sku")
	index, err := loadGoFileSubtreeIndex(filepath.Join("..", "internal"), root)
	if err != nil {
		t.Fatal(err)
	}
	for path, facts := range index.files {
		if !strings.HasPrefix(path, root+string(os.PathSeparator)) {
			continue
		}
		if _, ok := facts.imports[`"task-processor/internal/infra/clients/management/api"`]; ok {
			t.Fatalf("%s imports concrete management API DTOs; use listingadmin/local runtime types", path)
		}
	}
}

func TestTemuProductUsesManagementAPIPortAliases(t *testing.T) {
	root := filepath.Join("..", "internal", "temu", "product")
	index, err := loadGoFileSubtreeIndex(filepath.Join("..", "internal"), root)
	if err != nil {
		t.Fatal(err)
	}
	for path, facts := range index.files {
		if !strings.HasPrefix(path, root+string(os.PathSeparator)) {
			continue
		}
		if _, ok := facts.imports[`"task-processor/internal/infra/clients/management/api"`]; ok {
			t.Fatalf("%s imports concrete management API DTOs; use listingadmin/local runtime types", path)
		}
	}
}

func TestTemuPricingUsesManagementAPIPortAliases(t *testing.T) {
	root := filepath.Join("..", "internal", "temu", "pricing")
	index, err := loadGoFileSubtreeIndex(filepath.Join("..", "internal"), root)
	if err != nil {
		t.Fatal(err)
	}
	for path, facts := range index.files {
		if !strings.HasPrefix(path, root+string(os.PathSeparator)) {
			continue
		}
		if _, ok := facts.imports[`"task-processor/internal/infra/clients/management/api"`]; ok {
			t.Fatalf("%s imports concrete management API DTOs; use listingadmin/local runtime types", path)
		}
	}
}

func TestTemuAPIClientUsesManagementAPIPortAliases(t *testing.T) {
	root := filepath.Join("..", "internal", "temu", "api", "client")
	index, err := loadGoFileSubtreeIndex(filepath.Join("..", "internal"), root)
	if err != nil {
		t.Fatal(err)
	}
	for path, facts := range index.files {
		if _, ok := facts.imports[`"task-processor/internal/infra/clients/management/api"`]; ok {
			t.Fatalf("%s imports concrete management API DTOs; use listingadmin/local runtime types", path)
		}
	}
}

func TestRetiredManagementServiceHotspotsUseCapabilityNames(t *testing.T) {
	for _, relative := range []string{"internal/sheinloginmanaged/bridge.go", "internal/shein/client/cookie_manager_local_test.go"} {
		if _, err := os.Stat(filepath.Join("..", filepath.FromSlash(relative))); !os.IsNotExist(err) {
			t.Fatalf("retired numeric bridge remains: %s: %v", relative, err)
		}
	}
}

func TestRetiredManagementRuntimeNamingHotspotsUseCapabilityNames(t *testing.T) {
	checks := map[string][]string{
		filepath.Join("..", "internal", "shein", "scheduler", "factory.go"): {
			"type managementRuntime",
			"managementClient",
			"platformbase.ManagementRuntime",
			"ManagementRuntime:",
			"GetManagementRuntime",
		},
		filepath.Join("..", "internal", "shein", "scheduler", "runtime_adapter.go"): {
			"ManagementRuntime",
			"NewManagementRuntime",
		},
		filepath.Join("..", "internal", "shein", "pipeline", "processor.go"): {
			"type managementRuntime",
		},
		filepath.Join("..", "internal", "shein", "pipeline", "dependencies_builder.go"): {
			"managementRuntime",
		},
		filepath.Join("..", "internal", "temu", "product", "submit_handler.go"): {
			"management_api",
		},
		filepath.Join("..", "internal", "app", "bootstrap", "resources", "shared_resources.go"): {
			"managementSchedulerFactoryRuntime",
			"managementProcessorRuntime",
		},
		filepath.Join("..", "internal", "platformbase", "base_factory.go"): {
			"ManagementRuntime",
			"GetManagementRuntime",
		},
		filepath.Join("..", "internal", "temu", "scheduler", "factory.go"): {
			"ManagementRuntime:",
			"GetManagementRuntime",
		},
		filepath.Join("..", "internal", "pricing", "cost_calculator_test.go"): {
			"managementClient",
		},
	}

	for path, phrases := range checks {
		content, err := os.ReadFile(path)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				continue
			}
			t.Fatalf("read %s: %v", path, err)
		}
		for _, phrase := range phrases {
			if strings.Contains(string(content), phrase) {
				t.Fatalf("%s mentions %q; use runtime/repository/listingadmin capability names", path, phrase)
			}
		}
	}
}

func TestListingKitRootOpenAIImportsStayAllowlisted(t *testing.T) {
	root := filepath.Join("..", "internal", "listingkit")
	allowedFiles := map[string]struct{}{}

	index, err := loadGoFileIndex(root, "")
	if err != nil {
		t.Fatal(err)
	}

	for path, facts := range index.files {
		if filepath.Dir(path) != filepath.Clean(root) || strings.HasSuffix(filepath.Base(path), "_test.go") {
			continue
		}
		if pathAllowed(path, allowedFiles) {
			continue
		}
		for quotedImport := range facts.imports {
			importPath := strings.Trim(quotedImport, `"`)
			if importMatchesPrefix(importPath, "task-processor/internal/integration/openai") {
				t.Errorf("%s imports %s; keep ListingKit root contracts provider-neutral", path, importPath)
			}
		}
	}
}

func TestCanonicalTypesDoNotUseProductEnrichCompatibilityAliases(t *testing.T) {
	assertNoBannedSelectorsOutside(t, filepath.Join("..", "internal"), filepath.Join("..", "internal", "productenrich"), map[string]struct{}{
		"CanonicalProduct":           {},
		"CanonicalVariant":           {},
		"CanonicalImage":             {},
		"CanonicalAttribute":         {},
		"CanonicalSource":            {},
		"CanonicalSourceType":        {},
		"FieldTrace":                 {},
		"ProductSpecs":               {},
		"Dimensions":                 {},
		"Weight":                     {},
		"PackageInfo":                {},
		"PriceInfo":                  {},
		"ScrapedVariantDimension":    {},
		"CanonicalSourceUserText":    {},
		"CanonicalSourceUserImage":   {},
		"CanonicalSourceProductURL":  {},
		"CanonicalSourceScrapedData": {},
		"CanonicalSourceLLM":         {},
		"CanonicalSourceDerived":     {},
	})
}

func TestAppProcessorCompatibilityLayerIsRetired(t *testing.T) {
	path := filepath.Join("..", "internal", "app", "processor")
	if _, err := os.Stat(path); err == nil {
		t.Fatalf("%s still exists; use internal/processor directly instead of the app compatibility layer", path)
	} else if !os.IsNotExist(err) {
		t.Fatalf("stat %s: %v", path, err)
	}
}

func TestAppStateCompatibilityLayerIsRetired(t *testing.T) {
	path := filepath.Join("..", "internal", "app", "state")
	entries, err := os.ReadDir(path)
	if os.IsNotExist(err) {
		return
	}
	if err != nil {
		t.Fatalf("stat %s: %v", path, err)
	}
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".go") {
			t.Fatalf("%s still contains Go compatibility file %s; use internal/state directly instead", path, entry.Name())
		}
	}
}

func TestInternalPackagesDoNotImportAppCompatibilityLayersAcrossBuildTargets(t *testing.T) {
	index, err := loadGoFileIndex(filepath.Join("..", "internal"), "")
	if err != nil {
		t.Fatal(err)
	}
	for path, facts := range index.files {
		for quotedImport := range facts.imports {
			importPath := strings.Trim(quotedImport, `"`)
			if importMatchesPrefix(importPath, "task-processor/internal/app/processor") || importMatchesPrefix(importPath, "task-processor/internal/app/state") {
				t.Errorf("%s imports %s; keep retired app compatibility packages out of internal production code", path, importPath)
			}
		}
	}
}

func TestInfraProductCrawlerAdapterIsRetired(t *testing.T) {
	path := filepath.Join("..", "internal", "infra", "productcrawler")
	if _, err := os.Stat(path); err == nil {
		t.Fatalf("%s still exists; use product/sourcing plus app crawler fetchers instead of the unwired infra productcrawler adapter", path)
	} else if !os.IsNotExist(err) {
		t.Fatalf("stat %s: %v", path, err)
	}
}

func TestAppCrawlerFetcherCompatibilityLayerIsRetired(t *testing.T) {
	path := filepath.Join("..", "internal", "app", "crawler", "fetcher")
	if _, err := os.Stat(path); err == nil {
		t.Fatalf("%s still exists; use current Product sourcing ports and integration/acquisition adapters", path)
	} else if !os.IsNotExist(err) {
		t.Fatalf("stat %s: %v", path, err)
	}
}

func TestCmdPackagesDoNotImportAppCompatibilityLayers(t *testing.T) {
	root := filepath.Join("..", "cmd")
	index, err := loadGoFileIndex(root, "")
	if err != nil {
		t.Fatal(err)
	}
	for path, facts := range index.files {
		for quotedImport := range facts.imports {
			importPath := strings.Trim(quotedImport, `"`)
			if importMatchesPrefix(importPath, "task-processor/internal/app/processor") || importMatchesPrefix(importPath, "task-processor/internal/app/state") {
				t.Errorf("%s imports banned boundary package %q", path, importPath)
			}
		}
	}
}

func TestDomainHTTPPackagesDoNotImportAppHTTPAPI(t *testing.T) {
	for _, domainRoot := range []string{
		filepath.Join("..", "internal", "amazonlisting", "httpapi"),
		filepath.Join("..", "internal", "authz", "httpapi"),
		filepath.Join("..", "internal", "sourceaccountregistry", "httpapi"),
	} {
		t.Run(filepath.Base(domainRoot), func(t *testing.T) {
			assertNoBannedImportsBelow(t, filepath.Join("..", "internal"), domainRoot, []string{
				`"task-processor/internal/app/httpapi"`,
			}, nil)
		})
	}
}

func TestBusinessDomainsDoNotImportAppHTTPAPI(t *testing.T) {
	for _, domainRoot := range []string{
		filepath.Join("..", "internal", "amazon"),
		filepath.Join("..", "internal", "amazonlisting"),
		filepath.Join("..", "internal", "product", "asset"),
		filepath.Join("..", "internal", "product", "catalog"),
		filepath.Join("..", "internal", "listing"),
		filepath.Join("..", "internal", "listingkit"),
		filepath.Join("..", "internal", "marketplace"),
		filepath.Join("..", "internal", "pricing"),
		filepath.Join("..", "internal", "publishing"),
		filepath.Join("..", "internal", "sds"),
		filepath.Join("..", "internal", "shein"),
		filepath.Join("..", "internal", "sourceaccountregistry"),
		filepath.Join("..", "internal", "temu"),
	} {
		t.Run(filepath.Base(domainRoot), func(t *testing.T) {
			assertNoBannedImportsBelow(t, filepath.Join("..", "internal"), domainRoot, []string{
				`"task-processor/internal/app/httpapi"`,
			}, nil)
		})
	}

	// SA1 is a clean-room current owner. Keep the no-legacy constraint in this
	// already-governed business-boundary test so it is reviewed with the domain.
	targets := []string{
		filepath.Join("..", "internal", "sourceaccountregistry"),
		filepath.Join("..", "internal", "integration", "persistence", "sourceaccountregistry"),
		filepath.Join("..", "internal", "app", "schema", "sourceaccountregistry"),
		filepath.Join("..", "internal", "app", "runtime", "sourceaccountregistryschemainit"),
		filepath.Join("..", "internal", "app", "httpapi", "source_account_application.go"),
		filepath.Join("..", "cmd", "source-account-registry-schema-init"),
	}
	banned := []string{
		"task-processor/internal/sourceaccount",
		"task-processor/internal/integration/persistence/sourceaccount",
		"task-processor/internal/listingkit",
		"task-processor/internal/compatibility",
		"task-processor/internal/tenantbridge",
	}
	for _, target := range targets {
		t.Run(filepath.ToSlash(target), func(t *testing.T) {
			assertNoBannedImportPrefixes(t, target, banned, nil)
		})
	}
}

func TestProjectBoundaryDomainsDoNotImportListingKitFacade(t *testing.T) {
	for _, domainRoot := range []string{
		filepath.Join("..", "internal", "amazon"),
		filepath.Join("..", "internal", "product", "asset"),
		filepath.Join("..", "internal", "product", "catalog"),
		filepath.Join("..", "internal", "infra"),
		filepath.Join("..", "internal", "integration"),
		filepath.Join("..", "internal", "marketplace"),
		filepath.Join("..", "internal", "platform"),
		filepath.Join("..", "internal", "product", "sourcing"),
		filepath.Join("..", "internal", "publishing"),
		filepath.Join("..", "internal", "shein"),
		filepath.Join("..", "internal", "temu"),
		filepath.Join("..", "internal", "workspace"),
	} {
		t.Run(filepath.ToSlash(domainRoot), func(t *testing.T) {
			if domainRoot == filepath.Join("..", "internal", "workspace") || domainRoot == filepath.Join("..", "internal", "infra") ||
				domainRoot == filepath.Join("..", "internal", "amazon") || domainRoot == filepath.Join("..", "internal", "temu") || domainRoot == filepath.Join("..", "internal", "publishing") {
				assertNoBannedImportPrefixesBelow(t, filepath.Join("..", "internal"), domainRoot, []string{
					"task-processor/internal/listingkit",
				}, nil)
				return
			}
			assertNoBannedImportPrefixes(t, domainRoot, []string{
				"task-processor/internal/listingkit",
			}, nil)
		})
	}
}

func TestProductDomainDoesNotDependOnOuterAdapters(t *testing.T) {
	t.Parallel()

	assertNoBannedImportPrefixes(t, filepath.Join("..", "internal", "product"), []string{
		"task-processor/internal/listingkit",
		"task-processor/internal/compatibility",
		"task-processor/internal/crawler",
		"task-processor/internal/integration",
	}, nil)
}

func TestInfrastructurePackagesDoNotImportBusinessDomains(t *testing.T) {
	catalogAdapterDirectory := filepath.Clean(filepath.Join("..", "internal", "integration", "persistence", "product", "catalog")) + string(os.PathSeparator)
	catalogAdapterOnly := map[string]struct{}{catalogAdapterDirectory: {}, filepath.Clean(filepath.Join("..", "internal", "integration", "persistence", "product", "review", "repository.go")): {}}
	submissionAdapterDirectory := filepath.Clean(filepath.Join("..", "internal", "integration", "persistence", "listing", "submission")) + string(os.PathSeparator)
	submissionAdapterOnly := map[string]struct{}{submissionAdapterDirectory: {}}
	for path := range supplyAdapterBusinessPorts() {
		submissionAdapterOnly[path] = struct{}{}
	}
	for _, infraRoot := range []string{
		filepath.Join("..", "internal", "infra"),
		filepath.Join("..", "internal", "integration"),
		filepath.Join("..", "internal", "platform"),
	} {
		t.Run(filepath.ToSlash(infraRoot), func(t *testing.T) {
			assertPrefixes := assertNoBannedImportPrefixes
			if infraRoot == filepath.Join("..", "internal", "infra") {
				assertPrefixes = func(t *testing.T, root string, banned []string, allowed map[string]struct{}) {
					assertNoBannedImportPrefixesBelow(t, filepath.Join("..", "internal"), root, banned, allowed)
				}
			}
			assertPrefixes(t, infraRoot, []string{
				"task-processor/internal/amazon",
				"task-processor/internal/amazonlisting",
				"task-processor/internal/asset",
				"task-processor/internal/listing",
				"task-processor/internal/listingkit",
				"task-processor/internal/marketplace",
				"task-processor/internal/pricing",
				"task-processor/internal/productenrich",
				"task-processor/internal/productimage",
				"task-processor/internal/publishing",
				"task-processor/internal/sds",
				"task-processor/internal/shein",
				"task-processor/internal/temu",
				"task-processor/internal/workspace",
			}, submissionAdapterOnly)
			assertPrefixes(t, infraRoot, []string{
				"task-processor/internal/product/catalog",
			}, catalogAdapterOnly)
		})
	}
}

func TestListingSubmissionPersistenceAdapterImplementsOnlySubmissionPort(t *testing.T) {
	adapterRoot := filepath.Join("..", "internal", "integration", "persistence", "listing", "submission")
	// #31 G4, frozen #669 and #733: exact atomic same-ProductDB effect policies consume the current
	// immutable target and digest ports. It remains in the original claim UoW.
	policy := filepath.Join(adapterRoot, "temu_effects_policy.go")
	tiktokPolicy := filepath.Join(adapterRoot, "tiktok_effects_policy.go")
	walmartPolicy := filepath.Join(adapterRoot, "walmart_effects_policy.go")
	sheinPolicy := filepath.Join(adapterRoot, "shein_shelf_policy.go")
	assertNoBannedImportPrefixes(t, adapterRoot, []string{
		"task-processor/internal/amazon",
		"task-processor/internal/amazonlisting",
		"task-processor/internal/asset",
		"task-processor/internal/listingkit",
		"task-processor/internal/marketplace",
		"task-processor/internal/pricing",
		"task-processor/internal/product",
		"task-processor/internal/productenrich",
		"task-processor/internal/productimage",
		"task-processor/internal/publishing",
		"task-processor/internal/sds",
		"task-processor/internal/shein",
		"task-processor/internal/temu",
		"task-processor/internal/workspace",
	}, map[string]struct{}{policy: {}, tiktokPolicy: {}, walmartPolicy: {}, sheinPolicy: {}})

	index, err := loadGoFileIndex(adapterRoot, "")
	if err != nil {
		t.Fatal(err)
	}
	foundSubmissionPort := false
	for path, facts := range index.files {
		if strings.HasSuffix(filepath.Base(path), "_test.go") {
			continue
		}
		for quotedImport := range facts.imports {
			importPath := strings.Trim(quotedImport, `"`)
			if !importMatchesPrefix(importPath, "task-processor/internal/listing") {
				continue
			}
			if importPath != "task-processor/internal/listing/submission" && !((path == policy || path == tiktokPolicy || path == walmartPolicy || path == sheinPolicy) && importPath == "task-processor/internal/listing/record/target") {
				t.Errorf("%s imports Listing sibling %s; the adapter may implement only Listing Submission-owned ports", path, importPath)
			}
			foundSubmissionPort = true
		}
	}
	if !foundSubmissionPort {
		t.Fatal("Listing Submission persistence adapter must implement the Submission-owned repository port")
	}
}

func TestInfrastructureCatalogGuardRejectsNonAdapterFixture(t *testing.T) {
	fixtureRoot := filepath.Join("testdata", "infrastructure_catalog_violation")
	violations, err := findBannedImportViolations(fixtureRoot, []string{
		`"task-processor/internal/product/catalog"`,
	}, nil, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(violations) != 1 || violations[0].importPath != "task-processor/internal/product/catalog" {
		t.Fatalf("Catalog guard violations = %+v, want the non-adapter fixture rejected", violations)
	}
}

func TestProductCatalogPersistenceAdapterImplementsOnlyApprovedBusinessPort(t *testing.T) {
	adapterRoot := filepath.Join("..", "internal", "integration", "persistence", "product", "catalog")
	assertNoBannedImportPrefixes(t, adapterRoot, []string{
		"task-processor/internal/amazon",
		"task-processor/internal/amazonlisting",
		"task-processor/internal/asset",
		"task-processor/internal/listing",
		"task-processor/internal/listingkit",
		"task-processor/internal/marketplace",
		"task-processor/internal/pricing",
		"task-processor/internal/productenrich",
		"task-processor/internal/productimage",
		"task-processor/internal/publishing",
		"task-processor/internal/sds",
		"task-processor/internal/shein",
		"task-processor/internal/temu",
		"task-processor/internal/workspace",
	}, nil)
	assertCatalogIsOnlyProductDomainImport(t, adapterRoot)

	repositorySource, err := os.ReadFile(filepath.Join(adapterRoot, "repository.go"))
	if err != nil {
		t.Fatalf("read Catalog persistence adapter: %v", err)
	}
	if !strings.Contains(string(repositorySource), `"task-processor/internal/product/catalog"`) {
		t.Fatal("Catalog persistence adapter must implement the Catalog-owned repository port")
	}
}

func TestProductCatalogPersistenceGuardRejectsEverySiblingProductDomainFixture(t *testing.T) {
	fixtureRoot := filepath.Join("testdata", "catalog_adapter_product_domain_violations")
	violations := findNonCatalogProductDomainImports(t, fixtureRoot)
	want := []string{
		"task-processor/internal/product/asset",
		"task-processor/internal/product/asset/recipe",
		"task-processor/internal/product/image",
		"task-processor/internal/product/image/presets",
		"task-processor/internal/product/review/proposal",
		"task-processor/internal/product/review/proposal/future",
		"task-processor/internal/product/sourcing",
		"task-processor/internal/product/sourcing/future",
	}
	if len(violations) != len(want) {
		t.Fatalf("sibling Product domain violations = %+v, want %v", violations, want)
	}
	for index, violation := range violations {
		if violation.importPath != want[index] {
			t.Fatalf("sibling Product domain violation[%d] = %q, want %q", index, violation.importPath, want[index])
		}
	}
}

func assertCatalogIsOnlyProductDomainImport(t *testing.T, root string) {
	t.Helper()
	for _, violation := range findNonCatalogProductDomainImports(t, root) {
		t.Errorf("%s imports sibling Product business domain %s; the Catalog persistence adapter may import only Catalog-owned ports", violation.path, violation.importPath)
	}
}

func findNonCatalogProductDomainImports(t *testing.T, root string) []bannedImportViolation {
	t.Helper()
	index, err := loadGoFileIndex(root, "")
	if err != nil {
		t.Fatal(err)
	}
	var violations []bannedImportViolation
	for path, facts := range index.files {
		if strings.HasSuffix(filepath.Base(path), "_test.go") {
			continue
		}
		for quotedImport := range facts.imports {
			importPath := strings.Trim(quotedImport, `"`)
			if importMatchesPrefix(importPath, "task-processor/internal/product") &&
				!importMatchesPrefix(importPath, "task-processor/internal/product/catalog") {
				violations = append(violations, bannedImportViolation{path: path, importPath: importPath})
			}
		}
	}
	sort.Slice(violations, func(i, j int) bool {
		if violations[i].path == violations[j].path {
			return violations[i].importPath < violations[j].importPath
		}
		return violations[i].path < violations[j].path
	})
	return violations
}

func TestBusinessDomainsDoNotImportAppRuntimeAssembly(t *testing.T) {
	allowedFiles := map[string]struct{}{}

	for _, domainRoot := range []string{
		filepath.Join("..", "internal", "amazon"),
		filepath.Join("..", "internal", "amazonlisting"),
		filepath.Join("..", "internal", "product", "asset"),
		filepath.Join("..", "internal", "product", "catalog"),
		filepath.Join("..", "internal", "listing"),
		filepath.Join("..", "internal", "listingkit"),
		filepath.Join("..", "internal", "marketplace"),
		filepath.Join("..", "internal", "pricing"),
		filepath.Join("..", "internal", "publishing"),
		filepath.Join("..", "internal", "sds"),
		filepath.Join("..", "internal", "shein"),
		filepath.Join("..", "internal", "temu"),
	} {
		t.Run(filepath.Base(domainRoot), func(t *testing.T) {
			assertNoBannedImportPrefixesBelow(t, filepath.Join("..", "internal"), domainRoot, []string{
				"task-processor/internal/app/bootstrap",
				"task-processor/internal/app/consumer",
				"task-processor/internal/app/runner",
				"task-processor/internal/app/runtime",
			}, allowedFiles)
		})
	}
}

func businessHTTPPackages(root string) map[string]struct{} {
	allowedHTTPPackages := map[string]struct{}{
		filepath.Clean(filepath.Join(root, "authz", "httpapi")) + string(os.PathSeparator):                 {},
		filepath.Clean(filepath.Join(root, "authruntime", "zitadel")) + string(os.PathSeparator):           {},
		filepath.Clean(filepath.Join(root, "app", "httpapi")) + string(os.PathSeparator):                   {},
		filepath.Clean(filepath.Join(root, "httproute")) + string(os.PathSeparator):                        {},
		filepath.Clean(filepath.Join(root, "imageagent", "httpapi")) + string(os.PathSeparator):            {},
		filepath.Clean(filepath.Join(root, "kernel", "module")) + string(os.PathSeparator):                 {},
		filepath.Clean(filepath.Join(root, "sourceaccountregistry", "httpapi")) + string(os.PathSeparator): {},
		filepath.Clean(filepath.Join(root, "sheinlogin")) + string(os.PathSeparator):                       {},
		filepath.Clean(filepath.Join(root, "taskrpcapi")) + string(os.PathSeparator):                       {},
		filepath.Clean(filepath.Join(root, "listingkit", "studio_session_handler.go")):                     {},
		filepath.Clean(filepath.Join(root, "listingsubscription", "handler.go")):                           {},
	}
	allowedHTTPPackages[filepath.Clean(filepath.Join(root, "storecenter", "httpapi"))+string(os.PathSeparator)] = struct{}{}
	// #556 freezes Knowledge HTTP as its own domain adapter; core stays framework-free.
	allowedHTTPPackages[filepath.Clean(filepath.Join(root, "knowledge", "httpapi"))+string(os.PathSeparator)] = struct{}{}
	// #608 admits only Notification Center's feature-local HTTP adapter.
	allowedHTTPPackages[filepath.Clean(filepath.Join(root, "notificationcenter", "httpapi"))+string(os.PathSeparator)] = struct{}{}
	// #628 admits only the reviewed Report Center HTTP adapter; core stays framework-free.
	allowedHTTPPackages[filepath.Clean(filepath.Join(root, "reportcenter", "httpapi"))+string(os.PathSeparator)] = struct{}{}
	// #622 frozen design admits only these feature-local adapters; core remains Gin-free.
	allowedHTTPPackages[filepath.Clean(filepath.Join(root, "product", "supplymarket", "httpapi"))+string(os.PathSeparator)] = struct{}{}
	allowedHTTPPackages[filepath.Clean(filepath.Join(root, "app", "pod", "httpapi"))+string(os.PathSeparator)] = struct{}{}
	// #570 / PR #571 admits only Agent Configuration's feature-local HTTP adapter.
	allowedHTTPPackages[filepath.Clean(filepath.Join(root, "agentconfig", "httpapi"))+string(os.PathSeparator)] = struct{}{}
	// #688 admits only the frozen Skill transport adapter; contracts stay framework-free.
	allowedHTTPPackages[filepath.Clean(filepath.Join(root, "skill", "httpapi"))+string(os.PathSeparator)] = struct{}{}
	// #611 admits only the approved customization HTTP adapter, not its domain.
	allowedHTTPPackages[filepath.Clean(filepath.Join(root, "agentcustomization", "httpapi"))+string(os.PathSeparator)] = struct{}{}
	allowedHTTPPackages[filepath.Clean(filepath.Join(root, "listingsubscription", "httpapi"))+string(os.PathSeparator)] = struct{}{}
	allowedHTTPPackages[filepath.Clean(filepath.Join(root, "workbenchcontext", "httpapi"))+string(os.PathSeparator)] = struct{}{}

	allowedHTTPPackages[filepath.Clean(filepath.Join(root, "organization", "membership", "httpapi"))+string(os.PathSeparator)] = struct{}{}
	allowedHTTPPackages[filepath.Clean(filepath.Join(root, "accountallocation", "httpapi"))+string(os.PathSeparator)] = struct{}{}
	allowedHTTPPackages[filepath.Clean(filepath.Join(root, "commercial", "billing", "httpapi"))+string(os.PathSeparator)] = struct{}{}
	allowedHTTPPackages[filepath.Clean(filepath.Join(root, "subjectverification", "httpapi"))+string(os.PathSeparator)] = struct{}{}
	// #603 admits the feature-local HTTP adapter, never the framework-free domain.
	allowedHTTPPackages[filepath.Clean(filepath.Join(root, "ecoservices", "httpapi"))+string(os.PathSeparator)] = struct{}{}
	// #613 admits only the reviewed Tool Market HTTP adapter, not its contracts.
	allowedHTTPPackages[filepath.Clean(filepath.Join(root, "toolmarket", "httpapi"))+string(os.PathSeparator)] = struct{}{}
	// #621 admits only Data Service's feature-local adapter, not its domain or DI.
	allowedHTTPPackages[filepath.Clean(filepath.Join(root, "dataservice", "httpapi"))+string(os.PathSeparator)] = struct{}{}
	allowedHTTPPackages[filepath.Clean(filepath.Join(root, "app", "supplychain", "httpapi"))+string(os.PathSeparator)] = struct{}{}
	// #614 admits only the observations feature-local HTTP adapter.
	allowedHTTPPackages[filepath.Clean(filepath.Join(root, "app", "storeobservations", "httpapi"))+string(os.PathSeparator)] = struct{}{}
	allowedHTTPPackages[filepath.Clean(filepath.Join(root, "product", "collection", "httpapi"))+string(os.PathSeparator)] = struct{}{}
	// #624 admits only Project Center's HTTP adapter; its domain stays framework-free.
	allowedHTTPPackages[filepath.Clean(filepath.Join(root, "aiworkbench", "projectcenter", "httpapi"))+string(os.PathSeparator)] = struct{}{}
	// #627 admits only the frozen Cockpit HTTP adapter; calculations stay framework-free.
	allowedHTTPPackages[filepath.Clean(filepath.Join(root, "operationscockpit", "httpapi"))+string(os.PathSeparator)] = struct{}{}
	return allowedHTTPPackages
}

func TestDataServiceHTTPBoundaryRegistration(t *testing.T) {
	root := filepath.Join("..", "internal")
	allowed := businessHTTPPackages(root)
	for path, want := range map[string]bool{"dataservice/httpapi/handler.go": true, "dataservice/httpapi/nested/handler.go": true, "dataservice/service.go": false, "dataservice/httpapi_extra/handler.go": false, "app/dataservices/module.go": false} {
		if got := pathAllowed(filepath.Join(root, filepath.FromSlash(path)), allowed); got != want {
			t.Errorf("HTTP boundary allows %s = %v, want %v", path, got, want)
		}
	}
}

func TestProjectCenterHTTPBoundaryRegistration(t *testing.T) {
	root := filepath.Join("..", "internal")
	allowed := businessHTTPPackages(root)
	for path, want := range map[string]bool{"aiworkbench/projectcenter/httpapi/handler.go": true, "aiworkbench/projectcenter/service.go": false, "aiworkbench/projectcenter/httpapi_extra/handler.go": false} {
		if got := pathAllowed(filepath.Join(root, filepath.FromSlash(path)), allowed); got != want {
			t.Errorf("HTTP boundary allows %s = %v, want %v", path, got, want)
		}
	}
}

func TestCockpitHTTPBoundaryRegistration(t *testing.T) {
	root := filepath.Join("..", "internal")
	allowed := businessHTTPPackages(root)
	for path, want := range map[string]bool{"operationscockpit/httpapi/handler.go": true, "operationscockpit/calculations.go": false, "operationscockpit/httpapi_extra/handler.go": false} {
		if got := pathAllowed(filepath.Join(root, filepath.FromSlash(path)), allowed); got != want {
			t.Errorf("HTTP boundary allows %s = %v, want %v", path, got, want)
		}
	}
}

func TestEcoservicesHTTPBoundaryRegistration(t *testing.T) {
	root := filepath.Join("..", "internal")
	allowed := businessHTTPPackages(root)
	for _, tc := range []struct {
		path string
		want bool
	}{
		{"ecoservices/httpapi/handler.go", true},
		{"ecoservices/service.go", false},
		{"ecoservices/httpapi_extra/handler.go", false},
		{"ecoservices/httpapi2/handler.go", false},
		{"ecoservices-extra/httpapi/handler.go", false},
	} {
		if got := pathAllowed(filepath.Join(root, filepath.FromSlash(tc.path)), allowed); got != tc.want {
			t.Errorf("HTTP boundary allows %s = %v, want %v", tc.path, got, tc.want)
		}
	}
}

func TestToolMarketHTTPBoundaryRegistration(t *testing.T) {
	root := filepath.Join("..", "internal")
	allowed := businessHTTPPackages(root)
	for path, want := range map[string]bool{
		"toolmarket/httpapi/handler.go":        true,
		"toolmarket/httpapi/nested/handler.go": true,
		"toolmarket/contracts.go":              false,
		"toolmarket/httpapi_extra/handler.go":  false,
		"toolmarket/httpapi2/handler.go":       false,
		"toolmarket-extra/httpapi/handler.go":  false,
	} {
		if got := pathAllowed(filepath.Join(root, filepath.FromSlash(path)), allowed); got != want {
			t.Errorf("HTTP boundary allows %s = %v, want %v", path, got, want)
		}
	}
}

func TestMembershipHTTPBoundaryRegistration(t *testing.T) {
	root := filepath.Join("..", "internal")
	allowed := businessHTTPPackages(root)
	for _, tc := range []struct {
		path string
		want bool
	}{
		{"organization/membership/httpapi/handler.go", true},
		{"organization/membership/httpapi/nested/handler.go", true},
		{"organization/membership/service.go", false},
		{"organization/membership/worker/worker.go", false},
		{"organization/membership/api/handler.go", false},
		{"organization/other/httpapi/handler.go", false},
		{"organization/membership/httpapi_extra/handler.go", false},
		{"organization/membership/httpapi2/handler.go", false},
		{"organization/membership-extra/httpapi/handler.go", false},
	} {
		t.Run(tc.path, func(t *testing.T) {
			if got := pathAllowed(filepath.Join(root, filepath.FromSlash(tc.path)), allowed); got != tc.want {
				t.Errorf("actual HTTP boundary allows %s = %v, want %v", tc.path, got, tc.want)
			}
		})
	}
}

func TestSubjectVerificationHTTPBoundaryRegistration(t *testing.T) {
	root := filepath.Join("..", "internal")
	allowed := businessHTTPPackages(root)
	for _, tc := range []struct {
		path string
		want bool
	}{
		{"subjectverification/httpapi/handler.go", true},
		{"subjectverification/httpapi/nested/handler.go", true},
		{"subjectverification/service.go", false},
		{"subjectverification/httpapi_extra/handler.go", false},
		{"subjectverification/httpapi2/handler.go", false},
		{"subjectverification-extra/httpapi/handler.go", false},
	} {
		if got := pathAllowed(filepath.Join(root, filepath.FromSlash(tc.path)), allowed); got != tc.want {
			t.Errorf("HTTP boundary allows %s = %v, want %v", tc.path, got, tc.want)
		}
	}
}

func TestNotificationCenterHTTPBoundaryRegistration(t *testing.T) {
	root := filepath.Join("..", "internal")
	allowed := businessHTTPPackages(root)
	for path, want := range map[string]bool{"notificationcenter/httpapi/handler.go": true, "notificationcenter/service.go": false, "notificationcenter/httpapi_extra/handler.go": false} {
		if got := pathAllowed(filepath.Join(root, filepath.FromSlash(path)), allowed); got != want {
			t.Errorf("HTTP boundary allows %s = %v, want %v", path, got, want)
		}
	}
}

func TestReportCenterHTTPBoundaryRegistration(t *testing.T) {
	root := filepath.Join("..", "internal")
	allowed := businessHTTPPackages(root)
	for path, want := range map[string]bool{
		"reportcenter/httpapi/handler.go":        true,
		"reportcenter/httpapi/nested/handler.go": true,
		"reportcenter/service.go":                false,
		"reportcenter/httpapi_extra/handler.go":  false,
		"reportcenter-extra/httpapi/handler.go":  false,
	} {
		if got := pathAllowed(filepath.Join(root, filepath.FromSlash(path)), allowed); got != want {
			t.Errorf("HTTP boundary allows %s = %v, want %v", path, got, want)
		}
	}
}

func TestSupplyMarketHTTPBoundaryRegistration(t *testing.T) {
	root := filepath.Join("..", "internal")
	allowed := businessHTTPPackages(root)
	for path, want := range map[string]bool{
		"product/supplymarket/httpapi/handler.go":       true,
		"app/pod/httpapi/handler.go":                    true,
		"product/supplymarket/service.go":               false,
		"app/pod/service.go":                            false,
		"app/pod/httpapi_extra/handler.go":              false,
		"product/supplymarket-extra/httpapi/handler.go": false,
	} {
		if got := pathAllowed(filepath.Join(root, filepath.FromSlash(path)), allowed); got != want {
			t.Errorf("HTTP boundary allows %s = %v, want %v", path, got, want)
		}
	}
}

func TestAgentCustomizationHTTPBoundaryRegistration(t *testing.T) {
	root := filepath.Join("..", "internal")
	allowed := businessHTTPPackages(root)
	for path, want := range map[string]bool{
		"agentcustomization/httpapi/handler.go":       true,
		"agentcustomization/service.go":               false,
		"agentcustomization/httpapi_extra/handler.go": false,
		"agentcustomization/httpapi2/handler.go":      false,
		"agentcustomization-extra/httpapi/handler.go": false,
	} {
		if got := pathAllowed(filepath.Join(root, filepath.FromSlash(path)), allowed); got != want {
			t.Errorf("HTTP boundary allows %s = %v, want %v", path, got, want)
		}
	}
}

func TestBusinessImplementationPackagesDoNotImportGinDirectly(t *testing.T) {
	root := filepath.Join("..", "internal")
	allowedHTTPPackages := businessHTTPPackages(root)

	index, err := loadGoFileIndex(root, "")
	if err != nil {
		t.Fatal(err)
	}
	for path, facts := range index.files {
		if strings.HasSuffix(filepath.Base(path), "_test.go") || pathAllowed(path, allowedHTTPPackages) {
			continue
		}
		if _, ok := facts.imports[`"github.com/gin-gonic/gin"`]; ok {
			t.Errorf("%s imports gin directly; keep HTTP framework dependencies in api/httpapi or explicitly registered legacy HTTP adapter packages", path)
		}
	}
}

func TestSkillHTTPBoundaryRegistration(t *testing.T) {
	root := filepath.Join("..", "internal")
	allowed := businessHTTPPackages(root)
	for path, want := range map[string]bool{"skill/httpapi/handler.go": true, "skill/httpapi/nested/handler.go": true, "skill/service.go": false, "skill/httpapi_extra/handler.go": false, "skill-extra/httpapi/handler.go": false} {
		if got := pathAllowed(filepath.Join(root, filepath.FromSlash(path)), allowed); got != want {
			t.Errorf("Skill HTTP boundary allows %s = %v, want %v", path, got, want)
		}
	}
}

func TestAmazonExternalClientImportsStayAllowlisted(t *testing.T) {
	root := filepath.Join("..", "internal", "amazon")
	allowedFiles := map[string]struct{}{
		filepath.Clean(filepath.Join(root, "processor.go")): {},
	}

	index, err := loadGoFileSubtreeIndex(filepath.Join("..", "internal"), root)
	if err != nil {
		t.Fatal(err)
	}

	for path, facts := range index.files {
		if pathAllowed(path, allowedFiles) {
			continue
		}
		for quotedImport := range facts.imports {
			importPath := strings.Trim(quotedImport, `"`)
			if importMatchesPrefix(importPath, "task-processor/internal/infra/clients/management") ||
				importMatchesPrefix(importPath, "task-processor/internal/integration/openai") {
				t.Errorf("%s imports %s; keep Amazon concrete external client dependencies limited to current management DTO and OpenAI LLM seams", path, importPath)
			}
		}
	}
}

func TestSheinBridgeExternalClientImportsStayAllowlisted(t *testing.T) {
	parentRoot := filepath.Join("..", "internal")
	root := filepath.Join(parentRoot, "sheinbridge")
	allowedFiles := map[string]struct{}{}

	// The retired bridge has no files in a clean checkout. Index its retained
	// parent so missing roots still fail, then keep the original bridge scope.
	index, err := loadGoFileIndex(parentRoot, "")
	if err != nil {
		t.Fatal(err)
	}

	for path, facts := range index.files {
		if !strings.HasPrefix(path, filepath.Clean(root)+string(filepath.Separator)) {
			continue
		}
		if pathAllowed(path, allowedFiles) {
			continue
		}
		for quotedImport := range facts.imports {
			importPath := strings.Trim(quotedImport, `"`)
			if importMatchesPrefix(importPath, "task-processor/internal/infra/clients/management") ||
				importMatchesPrefix(importPath, "task-processor/internal/integration/openai") {
				t.Errorf("%s imports %s; keep sheinbridge concrete external client dependencies limited to current sale-attribute runtime bridge seams", path, importPath)
			}
		}
	}
}

func TestSheinRetiredManagementImportsStayBlocked(t *testing.T) {
	root := filepath.Join("..", "internal", "shein")
	allowedFiles := map[string]struct{}{
		filepath.Clean(filepath.Join(root, "api", "image", "client.go")):                   {},
		filepath.Clean(filepath.Join(root, "authorizedbrand", "context_test.go")):          {},
		filepath.Clean(filepath.Join(root, "authorizedbrand", "types.go")):                 {},
		filepath.Clean(filepath.Join(root, "pipeline", "processor.go")):                    {},
		filepath.Clean(filepath.Join(root, "pipeline", "task.go")):                         {},
		filepath.Clean(filepath.Join(root, "pipeline", "task_authorized_brand_test.go")):   {},
		filepath.Clean(filepath.Join(root, "scheduler", "activity_task.go")):               {},
		filepath.Clean(filepath.Join(root, "scheduler", "factory.go")):                     {},
		filepath.Clean(filepath.Join(root, "scheduler", "inventory_sync_adapter.go")):      {},
		filepath.Clean(filepath.Join(root, "scheduler", "inventory_sync_adapter_test.go")): {},
		filepath.Clean(filepath.Join(root, "scheduler", "inventory_task.go")):              {},
		filepath.Clean(filepath.Join(root, "scheduler", "pricing_task.go")):                {},
		filepath.Clean(filepath.Join(root, "scheduler", "product_sync_adapter.go")):        {},
		filepath.Clean(filepath.Join(root, "scheduler", "product_sync_adapter_test.go")):   {},
		filepath.Clean(filepath.Join(root, "scheduler", "product_task.go")):                {},
	}

	index, err := loadGoFileIndex(root, "")
	if err != nil {
		t.Fatal(err)
	}

	for path, facts := range index.files {
		if pathAllowed(path, allowedFiles) {
			continue
		}
		for quotedImport := range facts.imports {
			importPath := strings.Trim(quotedImport, `"`)
			if importMatchesPrefix(importPath, "task-processor/internal/infra/clients/management") {
				t.Errorf("%s imports %s; keep SHEIN retired management service dependencies limited to current inventory, scheduler, publish, validation, activity, mapping, and product seams", path, importPath)
			}
		}
	}
}

func TestSheinOpenAIImportsStayAllowlisted(t *testing.T) {
	root := filepath.Join("..", "internal", "shein")
	allowedFiles := map[string]struct{}{
		filepath.Clean(filepath.Join(root, "pipeline", "pipeline.go")): {},
	}

	index, err := loadGoFileIndex(root, "")
	if err != nil {
		t.Fatal(err)
	}

	for path, facts := range index.files {
		if pathAllowed(path, allowedFiles) {
			continue
		}
		for quotedImport := range facts.imports {
			importPath := strings.Trim(quotedImport, `"`)
			if importMatchesPrefix(importPath, "task-processor/internal/integration/openai") {
				t.Errorf("%s imports %s; keep SHEIN concrete OpenAI client dependencies limited to current category, content, pipeline, product, submit-prep, and translate seams", path, importPath)
			}
		}
	}
}

func TestAppHTTPAPIProductImageExternalClientImportsStayAllowlisted(t *testing.T) {
	root := filepath.Join("..", "internal", "app", "httpapi")
	allowedFiles := map[string]struct{}{}

	index, err := loadGoFileIndex(root, "")
	if err != nil {
		t.Fatal(err)
	}

	for path, facts := range index.files {
		if !strings.Contains(strings.ToLower(filepath.Base(path)), "productimage") {
			continue
		}
		if pathAllowed(path, allowedFiles) {
			continue
		}
		for _, bannedImport := range []string{
			`"task-processor/internal/integration/grsai"`,
			`"task-processor/internal/integration/openai"`,
		} {
			if _, ok := facts.imports[bannedImport]; ok {
				t.Errorf("%s imports %s; keep app/httpapi free of ProductImage concrete external clients; ProductImage provider assembly belongs in internal/productimage/httpapi", path, bannedImport)
			}
		}
	}
}

func TestAppHTTPAPIRetiredManagementImportsStayBlocked(t *testing.T) {
	root := filepath.Join("..", "internal", "app", "httpapi")
	allowedFiles := map[string]struct{}{
		filepath.Clean(filepath.Join(root, "modules_shein_test.go")):   {},
		filepath.Clean(filepath.Join(root, "runtime_deps_methods.go")): {},
		filepath.Clean(filepath.Join(root, "runtime_deps_test.go")):    {},
	}

	index, err := loadGoFileIndex(root, "")
	if err != nil {
		t.Fatal(err)
	}

	for path, facts := range index.files {
		if pathAllowed(path, allowedFiles) {
			continue
		}
		for quotedImport := range facts.imports {
			importPath := strings.Trim(quotedImport, `"`)
			if importMatchesPrefix(importPath, "task-processor/internal/infra/clients/management") {
				t.Errorf("%s imports %s; keep app/httpapi management dependencies limited to current HTTP runtime dependency retirement seams and prefer in-repository database/repository access for new HTTP assembly data", path, importPath)
			}
		}
	}
}

func TestStateRetiredManagementImportsStayBlocked(t *testing.T) {
	root := filepath.Join("..", "internal", "state")
	allowedFiles := map[string]struct{}{}

	index, err := loadGoFileSubtreeIndex(filepath.Join("..", "internal"), root)
	if err != nil {
		t.Fatal(err)
	}

	for path, facts := range index.files {
		if pathAllowed(path, allowedFiles) {
			continue
		}
		for quotedImport := range facts.imports {
			importPath := strings.Trim(quotedImport, `"`)
			if importMatchesPrefix(importPath, "task-processor/internal/infra/clients/management") {
				t.Errorf("%s imports %s; keep state management dependencies limited to current state-runtime retirement seams and prefer in-repository database/repository access for new state data", path, importPath)
			}
		}
	}
}

func TestPlatformBaseRetiredManagementImportsStayBlocked(t *testing.T) {
	root := filepath.Join("..", "internal", "platformbase")
	if _, err := os.Stat(root); !os.IsNotExist(err) {
		t.Fatalf("retired platformbase root must stay absent: %v", err)
	}
}

func TestTaskRPCAPIRetiredManagementImportsStayBlocked(t *testing.T) {
	root := filepath.Join("..", "internal")
	assertNoBannedImportPrefixesBelow(t, root, filepath.Join(root, "taskrpcapi"), []string{"task-processor/internal/infra/clients/management"}, nil)
	assertNoBannedImportPrefixes(t, root, []string{"task-processor/internal/taskrpcapi"}, nil)
}

func TestTaskRPCAPIRetiredActionRoutesStayRetired(t *testing.T) {
	root := filepath.Join("..", "internal")
	index, err := loadGoFileIndex(root, "")
	if err != nil {
		t.Fatal(err)
	}
	retiredPrefix := filepath.Join(root, "taskrpcapi") + string(os.PathSeparator)
	routesCore := filepath.Join(root, "app", "httpapi", "routes_core.go")
	for path := range index.files {
		if strings.HasPrefix(path, retiredPrefix) || path == routesCore {
			assertNoRetiredTaskRPCActionSurface(t, path)
		}
	}
}

func assertNoRetiredTaskRPCActionSurface(t *testing.T, path string) {
	t.Helper()
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	for _, phrase := range []string{
		"TaskActionRespDTO",
		"RetryTask",
		"CancelTask",
		"/api/v1/management/tasks/:task_id/retry",
		"/api/v1/management/tasks/:task_id/cancel",
		"/api/v1/management/tasks/health",
		"/api/v1/management/tasks/:task_id/status",
		"/api/v1/management/tasks/queue-stats",
		"retiredTaskAction",
	} {
		if strings.Contains(string(content), phrase) {
			t.Fatalf("%s mentions %q; remove retired management task action RPC surfaces instead of returning retired action responses", path, phrase)
		}
	}
}

func TestSDSClientRetiredManagementImportsStayBlocked(t *testing.T) {
	root := filepath.Join("..", "internal")
	allowedFiles := map[string]struct{}{}
	for _, relative := range trackedFiles(t, "internal/sds/client") {
		t.Errorf("retired SDS client source remains: %s", relative)
	}

	index, err := loadGoFileIndex(root, "")
	if err != nil {
		t.Fatal(err)
	}

	for path, facts := range index.files {
		if pathAllowed(path, allowedFiles) {
			continue
		}
		for quotedImport := range facts.imports {
			importPath := strings.Trim(quotedImport, `"`)
			if importMatchesPrefix(importPath, "task-processor/internal/sds/client") {
				t.Errorf("%s imports %s; the complete old SDS client is retired, including its former management seam", path, importPath)
			}
		}
	}
}

func TestSheinLoginBootstrapRetiredManagementImportsStayBlocked(t *testing.T) {
	root := filepath.Join("..", "internal", "sheinlogin", "bootstrap")
	allowedFiles := map[string]struct{}{}

	index, err := loadGoFileIndex(root, "")
	if err != nil {
		t.Fatal(err)
	}

	for path, facts := range index.files {
		if pathAllowed(path, allowedFiles) {
			continue
		}
		for quotedImport := range facts.imports {
			importPath := strings.Trim(quotedImport, `"`)
			if importMatchesPrefix(importPath, "task-processor/internal/infra/clients/management") {
				t.Errorf("%s imports %s; keep sheinlogin/bootstrap management dependencies limited to current login bootstrap retirement seams and prefer in-repository database/repository access for new login bootstrap data", path, importPath)
			}
		}
	}
}

func TestSheinLoginManagedRetiredManagementImportsStayBlocked(t *testing.T) {
	index, err := loadGoFileIndex(filepath.Join("..", "internal"), "")
	if err != nil {
		t.Fatal(err)
	}
	for path := range index.files {
		if strings.Contains(filepath.ToSlash(path), "/sheinloginmanaged/") {
			t.Fatalf("retired managed bridge source remains: %s", path)
		}
	}
}

func TestSheinLoginUsesManagementAPIPortAliases(t *testing.T) {
	roots := []string{
		filepath.Join("..", "internal", "sheinlogin"),
	}
	for _, root := range roots {
		index, err := loadGoFileIndex(root, "")
		if err != nil {
			t.Fatal(err)
		}
		for path, facts := range index.files {
			if _, ok := facts.imports[`"task-processor/internal/infra/clients/management/api"`]; ok {
				t.Fatalf("%s imports concrete management API DTOs; use listingadmin/local runtime types", path)
			}
		}
	}
}

func TestSheinLoginServiceRetiredManagementImportsStayBlocked(t *testing.T) {
	root := filepath.Join("..", "internal", "sheinlogin")
	allowedFiles := map[string]struct{}{}

	index, err := loadGoFileIndex(root, "")
	if err != nil {
		t.Fatal(err)
	}

	for path, facts := range index.files {
		if pathAllowed(path, allowedFiles) {
			continue
		}
		for quotedImport := range facts.imports {
			importPath := strings.Trim(quotedImport, `"`)
			if importMatchesPrefix(importPath, "task-processor/internal/infra/clients/management") {
				t.Errorf("%s imports %s; keep sheinlogin management dependencies limited to current login-service retirement seams and prefer in-repository database/repository access for new login service data", path, importPath)
			}
		}
	}
}

func TestSharedPricingRetiredManagementImportsStayBlocked(t *testing.T) {
	root := filepath.Join("..", "internal", "pricing")
	if _, err := os.Stat(root); !os.IsNotExist(err) {
		t.Fatalf("retired numeric cost wrapper root must stay absent: %v", err)
	}
}

func TestTEMUSyncAndPricingRetiredManagementImportsStayBlocked(t *testing.T) {
	allowedFiles := map[string]struct{}{}

	for _, root := range []string{
		filepath.Join("..", "internal", "temu", "pricing"),
	} {
		t.Run(filepath.ToSlash(root), func(t *testing.T) {
			index, err := loadGoFileSubtreeIndex(filepath.Join("..", "internal"), root)
			if err != nil {
				t.Fatal(err)
			}

			for path, facts := range index.files {
				if !strings.HasPrefix(path, root+string(os.PathSeparator)) {
					continue
				}
				if pathAllowed(path, allowedFiles) {
					continue
				}
				for quotedImport := range facts.imports {
					importPath := strings.Trim(quotedImport, `"`)
					if importMatchesPrefix(importPath, "task-processor/internal/infra/clients/management") {
						t.Errorf("%s imports %s; keep TEMU retired management service dependencies limited to remaining pricing seams; sync is retired", path, importPath)
					}
				}
			}
		})
	}
}

func TestTEMUProductStoreAndSchedulerRetiredManagementImportsStayBlocked(t *testing.T) {
	allowedFiles := map[string]struct{}{}

	// Index the stable parent so the retired store directory can stay absent,
	// while both original subtrees remain covered if sources are reintroduced.
	index, err := loadGoFileIndex(filepath.Join("..", "internal"), "")
	if err != nil {
		t.Fatal(err)
	}
	for _, root := range []string{
		filepath.Join("..", "internal", "temu", "product"),
		filepath.Join("..", "internal", "temu", "store"),
	} {
		t.Run(filepath.ToSlash(root), func(t *testing.T) {
			for path, facts := range index.files {
				if !strings.HasPrefix(path, root+string(os.PathSeparator)) {
					continue
				}
				if pathAllowed(path, allowedFiles) {
					continue
				}
				for quotedImport := range facts.imports {
					importPath := strings.Trim(quotedImport, `"`)
					if importMatchesPrefix(importPath, "task-processor/internal/infra/clients/management") {
						t.Errorf("%s imports %s; keep TEMU retired management service dependencies limited to current product, store, and scheduler seams", path, importPath)
					}
				}
			}
		})
	}
}

func TestTEMURuntimeAndBridgeRetiredManagementImportsStayBlocked(t *testing.T) {
	allowedFiles := map[string]struct{}{
		filepath.Clean(filepath.Join("..", "internal", "temu", "api", "client", "client.go")): {},
		filepath.Clean(filepath.Join("..", "internal", "temu", "processor.go")):               {},
	}

	for _, root := range []string{
		filepath.Join("..", "internal", "temu", "api", "client"),
		filepath.Join("..", "internal", "temu", "context"),
		filepath.Join("..", "internal", "temu", "filter"),
		filepath.Join("..", "internal", "temu", "handlerbase"),
		filepath.Join("..", "internal", "temu", "rules"),
	} {
		t.Run(filepath.ToSlash(root), func(t *testing.T) {
			index, err := loadGoFileSubtreeIndex(filepath.Join("..", "internal"), root)
			if err != nil {
				t.Fatal(err)
			}

			for path, facts := range index.files {
				if !strings.HasPrefix(path, root+string(os.PathSeparator)) {
					continue
				}
				if pathAllowed(path, allowedFiles) {
					continue
				}
				for quotedImport := range facts.imports {
					importPath := strings.Trim(quotedImport, `"`)
					if importMatchesPrefix(importPath, "task-processor/internal/infra/clients/management") {
						t.Errorf("%s imports %s; keep TEMU runtime and bridge management dependencies limited to current runtime-assembly retirement seams and prefer in-repository database/repository access for new TEMU runtime data", path, importPath)
					}
				}
			}
		})
	}

	root := filepath.Join("..", "internal", "temu")
	index, err := loadGoFileSubtreeIndex(filepath.Join("..", "internal"), root)
	if err != nil {
		t.Fatal(err)
	}

	target := filepath.Clean(filepath.Join(root, "processor.go"))
	for path, facts := range index.files {
		if path != target || pathAllowed(path, allowedFiles) {
			continue
		}
		for quotedImport := range facts.imports {
			importPath := strings.Trim(quotedImport, `"`)
			if importMatchesPrefix(importPath, "task-processor/internal/infra/clients/management") {
				t.Errorf("%s imports %s; keep TEMU runtime and bridge management dependencies limited to current runtime-assembly retirement seams and prefer in-repository database/repository access for new TEMU runtime data", path, importPath)
			}
		}
	}
}

func TestTEMUOpenAIImportsStayAllowlisted(t *testing.T) {
	root := filepath.Join("..", "internal", "temu")
	allowedFiles := map[string]struct{}{
		filepath.Clean(filepath.Join(root, "pipeline_registry.go")): {},
	}

	index, err := loadGoFileSubtreeIndex(filepath.Join("..", "internal"), root)
	if err != nil {
		t.Fatal(err)
	}

	for path, facts := range index.files {
		if pathAllowed(path, allowedFiles) {
			continue
		}
		for quotedImport := range facts.imports {
			importPath := strings.Trim(quotedImport, `"`)
			if importMatchesPrefix(importPath, "task-processor/internal/integration/openai") {
				t.Errorf("%s imports %s; keep TEMU concrete OpenAI client dependencies limited to current AI, SKU, product, and pipeline seams", path, importPath)
			}
		}
	}
}

func TestAICapabilityModuleUsesOnlyApprovedDependencies(t *testing.T) {
	root := filepath.Join("..", "internal", "aicapability")
	approvedExternal := []string{
		"gorm.io/gorm",
	}

	index, err := loadGoFileIndex(root, "")
	if err != nil {
		t.Fatal(err)
	}
	for path, facts := range index.files {
		if strings.HasSuffix(filepath.Base(path), "_test.go") {
			continue
		}
		for quotedImport := range facts.imports {
			importPath := strings.Trim(quotedImport, `"`)
			if importMatchesPrefix(importPath, "task-processor/internal/aicapability") {
				continue
			}
			if strings.HasPrefix(importPath, "task-processor/") {
				t.Errorf("%s imports %s; AI capability contracts may depend only on their own internal package tree", path, importPath)
				continue
			}
			if !strings.Contains(strings.Split(importPath, "/")[0], ".") {
				continue
			}

			approved := false
			for _, allowed := range approvedExternal {
				if importMatchesPrefix(importPath, allowed) {
					approved = true
					break
				}
			}
			if !approved {
				t.Errorf("%s imports %s; AI capability contracts may depend only on the standard library, approved storage adapters, and their own package", path, importPath)
			}
		}
	}
}

func TestCmdProductionEntrypointsDoNotImportDomainOrInfraPackages(t *testing.T) {
	assertNoBannedImportPrefixes(t, filepath.Join("..", "cmd"), commandEntrypointBannedPrefixes(), nil)
}

func TestInternalCmdEntrypointsDoNotImportDomainOrInfraPackages(t *testing.T) {
	root := filepath.Join("..", "internal")
	index, err := loadGoFileIndex(root, "")
	if err != nil {
		t.Fatal(err)
	}
	for path, facts := range index.files {
		if strings.HasSuffix(filepath.Base(path), "_test.go") || !strings.Contains(filepath.ToSlash(path), "/cmd/") {
			continue
		}
		for quotedImport := range facts.imports {
			importPath := strings.Trim(quotedImport, `"`)
			for _, bannedPrefix := range commandEntrypointBannedPrefixes() {
				if importMatchesPrefix(importPath, bannedPrefix) {
					t.Errorf("%s imports banned command-entrypoint package prefix %s via %s", path, bannedPrefix, importPath)
				}
			}
		}
	}
}

func commandEntrypointBannedPrefixes() []string {
	return []string{
		"task-processor/internal/amazon",
		"task-processor/internal/amazonlisting",
		"task-processor/internal/asset",
		"task-processor/internal/product/catalog",
		"task-processor/internal/infra",
		"task-processor/internal/listingkit",
		"task-processor/internal/marketplace",
		"task-processor/internal/productenrich",
		"task-processor/internal/productimage",
		"task-processor/internal/publishing",
		"task-processor/internal/shein",
		"task-processor/internal/temu",
		"task-processor/internal/workspace",
	}
}

func TestTemporalSDKImportsStayInRuntimeAndOrchestrationAdapters(t *testing.T) {
	orchestrationAllowedFiles := map[string]struct{}{
		filepath.Clean(filepath.Join("..", "internal", "app", "runtime")) + string(os.PathSeparator):         {},
		filepath.Clean(filepath.Join("..", "internal", "imageagent", "temporal")) + string(os.PathSeparator): {},
	}
	assertNoBannedImports(t, filepath.Join("..", "internal"), []string{
		`"go.temporal.io/api/enums/v1"`,
		`"go.temporal.io/api/serviceerror"`,
		`"go.temporal.io/sdk/activity"`,
		`"go.temporal.io/sdk/converter"`,
		`"go.temporal.io/sdk/temporal"`,
		`"go.temporal.io/sdk/testsuite"`,
		`"go.temporal.io/sdk/worker"`,
		`"go.temporal.io/sdk/workflow"`,
	}, orchestrationAllowedFiles)

	platformClientFiles := temporalPlatformSDKClientAllowedFiles()
	clientAllowedFiles := make(map[string]struct{}, len(orchestrationAllowedFiles)+len(platformClientFiles))
	for path := range orchestrationAllowedFiles {
		clientAllowedFiles[path] = struct{}{}
	}
	for path := range platformClientFiles {
		clientAllowedFiles[path] = struct{}{}
	}
	assertNoBannedImports(t, filepath.Join("..", "internal"), []string{
		`"go.temporal.io/sdk/client"`,
	}, clientAllowedFiles)
}

func temporalPlatformSDKClientAllowedFiles() map[string]struct{} {
	return map[string]struct{}{
		filepath.Clean(filepath.Join("..", "internal", "platform", "temporal", "client.go")):      {},
		filepath.Clean(filepath.Join("..", "internal", "platform", "temporal", "client_test.go")): {},
	}
}

func TestTemporalRuntimePackagesDoNotImportHTTPAPI(t *testing.T) {
	for _, root := range []string{
		filepath.Join("..", "internal", "app", "runtime"),
	} {
		t.Run(filepath.ToSlash(root), func(t *testing.T) {
			assertNoBannedImportPrefixes(t, root, []string{
				"task-processor/internal/app/httpapi",
				"task-processor/internal/listingkit/httpapi",
			}, nil)
		})
	}
}

func TestAppHTTPAPIRootListingKitHelpersStayAllowlisted(t *testing.T) {
	root := filepath.Join("..", "internal", "app", "httpapi")
	allowed := map[string]struct{}{
		"listingkit_openai_runtime.go": {},
		"listingkit_shein_support.go":  {},
	}

	entries, err := os.ReadDir(root)
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if !strings.HasPrefix(name, "listingkit_") || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		if _, ok := allowed[name]; !ok {
			t.Errorf("%s is a new app/httpapi ListingKit helper; move ListingKit-specific logic into internal/listingkit/httpapi instead", filepath.Join(root, name))
		}
	}
}

func TestListingKitSupportFileStaysRetired(t *testing.T) {
	path := filepath.Join("..", "internal", "app", "httpapi", "listingkit_support.go")
	if _, err := os.Stat(path); err == nil {
		t.Fatalf("%s still exists; keep ListingKit runtime input shaping in feature_builder_listingkit.go instead of reviving the transition bucket", path)
	} else if !os.IsNotExist(err) {
		t.Fatalf("stat %s: %v", path, err)
	}
}

func TestAppHTTPAPIModuleBuildersStayAllowlisted(t *testing.T) {
	filePath := filepath.Join("..", "internal", "app", "httpapi", "modules.go")
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, filePath, nil, 0)
	if err != nil {
		if os.IsNotExist(err) {
			return
		}
		t.Fatal(err)
	}

	allowed := map[string]struct{}{
		"buildProductModule":       {},
		"buildImageModule":         {},
		"buildAmazonListingModule": {},
		"buildSheinLoginModule":    {},
		"buildSDSLoginModule":      {},
		"buildListingKitModule":    {},
	}

	for _, decl := range file.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Recv != nil {
			continue
		}
		if !strings.HasPrefix(fn.Name.Name, "build") || !strings.HasSuffix(fn.Name.Name, "Module") {
			continue
		}
		if _, ok := allowed[fn.Name.Name]; !ok {
			t.Errorf("%s declares new centralized module builder %s; prefer adding module/bootstrap in the owning domain package and keep app/httpapi as thin assembly", filePath, fn.Name.Name)
		}
	}
}

func TestAppHTTPAPIRouteDescriptorHelpersStayAllowlisted(t *testing.T) {
	root := filepath.Join("..", "internal", "app", "httpapi")
	allowed := map[string]struct{}{}

	index, err := loadGoFileIndex(root, "")
	if err != nil {
		t.Fatal(err)
	}
	fset := token.NewFileSet()
	for path, facts := range index.files {
		if strings.HasSuffix(filepath.Base(path), "_test.go") {
			continue
		}
		file, err := parser.ParseFile(fset, path, facts.source, 0)
		if err != nil {
			t.Fatal(err)
		}
		for _, decl := range file.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Recv != nil {
				continue
			}
			name := fn.Name.Name
			if !strings.HasPrefix(name, "append") || !strings.HasSuffix(name, "RouteDescriptors") {
				continue
			}
			if _, ok := allowed[name]; !ok {
				t.Errorf("%s declares app/httpapi route descriptor helper %s; add new feature routes in the owning domain httpapi package instead", path, name)
			}
		}
	}
}

func TestAppHTTPAPITaskRPCRouteDescriptorHelperStaysRetired(t *testing.T) {
	path := filepath.Join("..", "internal", "app", "httpapi", "routes_core.go")
	content, ok := readOptionalRetiredHTTPAPIRoutesCore(t, path)
	if !ok {
		return
	}
	for _, phrase := range []string{
		"appendTaskRPCRouteDescriptors",
		"/api/v1/management/tasks/",
	} {
		if strings.Contains(string(content), phrase) {
			t.Fatalf("%s mentions %q; delegate task RPC route descriptors to internal/taskrpcapi instead of duplicating module-owned routes in app/httpapi", path, phrase)
		}
	}
}

func TestAppHTTPAPILoginRouteDescriptorHelpersStayRetired(t *testing.T) {
	path := filepath.Join("..", "internal", "app", "httpapi", "routes_core.go")
	content, ok := readOptionalRetiredHTTPAPIRoutesCore(t, path)
	if !ok {
		return
	}
	for _, phrase := range []string{
		"appendSheinLoginRouteDescriptors",
		"appendSDSLoginRouteDescriptors",
		"/api/v1/shein-login/",
		"/api/v1/sds-login/",
	} {
		if strings.Contains(string(content), phrase) {
			t.Fatalf("%s mentions %q; keep retired login routes absent and preserve the current standalone SHEIN service owner", path, phrase)
		}
	}
}

func TestAppHTTPAPISDSCatalogRouteDescriptorHelperStaysRetired(t *testing.T) {
	path := filepath.Join("..", "internal", "app", "httpapi", "routes_core.go")
	content, ok := readOptionalRetiredHTTPAPIRoutesCore(t, path)
	if !ok {
		return
	}
	for _, phrase := range []string{
		"appendSDSCatalogRouteDescriptors",
		"/api/v1/sds/",
	} {
		if strings.Contains(string(content), phrase) {
			t.Fatalf("%s mentions %q; old SDS catalog routes are retired, and current POD routes retain their current owner", path, phrase)
		}
	}
}

func TestAppHTTPAPICoreRouteDescriptorHelperStaysRetired(t *testing.T) {
	path := filepath.Join("..", "internal", "app", "httpapi", "routes_core.go")
	if _, err := os.Stat(path); err == nil {
		t.Fatalf("%s exists; keep core health route registration inside newCoreHTTPModule instead of reviving a one-route app/httpapi descriptor helper", path)
	} else if !os.IsNotExist(err) {
		t.Fatalf("stat %s: %v", path, err)
	}
}

func readOptionalRetiredHTTPAPIRoutesCore(t *testing.T, path string) (string, bool) {
	t.Helper()

	content, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return "", false
		}
		t.Fatalf("read %s: %v", path, err)
	}
	return string(content), true
}

func TestAppHTTPAPISupportHTTPModuleWrappersStayRetired(t *testing.T) {
	httpModulesPath := filepath.Join("..", "internal", "app", "httpapi", "http_modules.go")
	httpModulesContent, err := os.ReadFile(httpModulesPath)
	if err != nil {
		t.Fatalf("read %s: %v", httpModulesPath, err)
	}
	for _, phrase := range []string{
		"newPromptTemplateHTTPModule",
		"newSDSCatalogHTTPModule",
		"newTaskRPCHTTPModule",
		"newSheinLoginHTTPModule",
		"newSDSLoginHTTPModule",
	} {
		if strings.Contains(string(httpModulesContent), phrase) {
			t.Fatalf("%s mentions %q; use owning support/login HTTP modules directly instead of keeping app/httpapi compatibility wrappers", httpModulesPath, phrase)
		}
	}

	handlersPath := filepath.Join("..", "internal", "app", "httpapi", "route_handler_types.go")
	handlersContent, err := os.ReadFile(handlersPath)
	if err != nil {
		if os.IsNotExist(err) {
			return
		}
		t.Fatalf("read %s: %v", handlersPath, err)
	}
	for _, phrase := range []string{
		"promptModule ",
		"sdsModule ",
		"taskRPCModule ",
		"sheinLoginModule ",
		"sdsLoginModule ",
	} {
		if strings.Contains(string(handlersContent), phrase) {
			t.Fatalf("%s mentions %q; keep app/httpapi production route handler scaffolding retired and carry built modules through httpFeatureComposition", handlersPath, phrase)
		}
	}
}

func TestAppHTTPAPIFeatureHTTPModuleWrappersStayRetired(t *testing.T) {
	path := filepath.Join("..", "internal", "app", "httpapi", "http_modules.go")
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	for _, phrase := range []string{
		"newProductHTTPModule",
		"newAmazonListingHTTPModule",
		"newListingKitHTTPModule",
		"newListingKitStudioHTTPModule",
	} {
		if strings.Contains(string(content), phrase) {
			t.Fatalf("%s mentions %q; keep runtime module selection on httpFeatureComposition and use owning feature HTTP modules directly in tests", path, phrase)
		}
	}
}

func TestHTTPAPIRouteDescriptorAliasStaysRetired(t *testing.T) {
	path := filepath.Join("..", "internal", "app", "httpapi", "route_handler_types.go")
	content, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return
		}
		t.Fatalf("read %s: %v", path, err)
	}
	if strings.Contains(string(content), "type routeDescriptor =") {
		t.Fatalf("%s exposes routeDescriptor; use httproute.Descriptor directly in HTTP API assembly", path)
	}
}

func TestHTTPAPIRouteHandlerAliasesStayRetired(t *testing.T) {
	path := filepath.Join("..", "internal", "app", "httpapi", "route_handler_types.go")
	content, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return
		}
		t.Fatalf("read %s: %v", path, err)
	}
	for _, phrase := range []string{
		"type productRouteHandler =",
		"type imageRouteHandler =",
		"type amazonListingRouteHandler =",
		"type listingKitRouteHandler =",
		"type studioSessionRouteHandler =",
		"type taskRPCRouteHandler =",
		"type promptTemplateRouteHandler =",
		"type sdsCatalogRouteHandler =",
		"type sdsLoginRouteHandler =",
	} {
		if strings.Contains(string(content), phrase) {
			t.Fatalf("%s exposes %q; use owning route handler types directly in app HTTP assembly", path, phrase)
		}
	}
}

func TestHTTPAPIProductionRouteHandlerScaffoldingStaysRetired(t *testing.T) {
	path := filepath.Join("..", "internal", "app", "httpapi", "route_handler_types.go")
	content, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return
		}
		t.Fatalf("read %s: %v", path, err)
	}
	for _, phrase := range []string{
		"type httpModuleHandlers struct",
		"type sheinLoginRouteHandler interface",
	} {
		if strings.Contains(string(content), phrase) {
			t.Fatalf("%s exposes %q; keep production route handler files limited to composition accessors and put test assembly scaffolding in tests", path, phrase)
		}
	}
}

func TestHTTPAPIRouteHandlerTypesFileStaysRetired(t *testing.T) {
	path := filepath.Join("..", "internal", "app", "httpapi", "route_handler_types.go")
	if _, err := os.Stat(path); err == nil {
		t.Fatalf("%s still exists; keep production handler accessors beside their bootstrap/module consumers instead of reviving a route handler types bucket", path)
	} else if !os.IsNotExist(err) {
		t.Fatalf("stat %s: %v", path, err)
	}
}

func TestAppHTTPAPIListingKitSupportImportsStayAllowlisted(t *testing.T) {
	filePath := filepath.Join("..", "internal", "app", "httpapi", "listingkit_support.go")
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, filePath, nil, parser.ImportsOnly)
	if err != nil {
		if os.IsNotExist(err) {
			return
		}
		t.Fatal(err)
	}

	allowed := map[string]struct{}{
		`"fmt"`:                                 {},
		`"path/filepath"`:                       {},
		`"strings"`:                             {},
		`"github.com/sirupsen/logrus"`:          {},
		`"gorm.io/gorm"`:                        {},
		`"task-processor/internal/app/runtime"`: {},
		`"task-processor/internal/asset/repository"`:       {},
		`"task-processor/internal/core/config"`:            {},
		`"task-processor/internal/integration/openai"`:     {},
		`"task-processor/internal/listingadmin"`:           {},
		`"task-processor/internal/listingkit"`:             {},
		`"task-processor/internal/listingkit/httpapi"`:     {},
		`"task-processor/internal/listingkit/reviewstore"`: {},
		`"task-processor/internal/listingkit/store"`:       {},
		`"task-processor/internal/listingsubscription"`:    {},
		`"task-processor/internal/publishing/shein"`:       {},
		`"task-processor/internal/sheinlogin"`:             {},
		`"task-processor/internal/tenantbridge"`:           {},
	}

	for _, imp := range file.Imports {
		path := imp.Path.Value
		if _, ok := allowed[path]; !ok {
			t.Errorf("%s imports %s; keep app/httpapi/listingkit_support.go limited to assembly and repository wiring, and move new ListingKit-specific logic into internal/listingkit/httpapi or domain packages", filePath, path)
		}
	}
}

func TestAppHTTPAPIListingKitRootImportsStayAllowlisted(t *testing.T) {
	root := filepath.Join("..", "internal", "app", "httpapi")
	allowedFiles := map[string]struct{}{
		filepath.Clean(filepath.Join(root, "image_agent_asset_catalog.go")):  {},
		filepath.Clean(filepath.Join(root, "listingkit_openai_runtime.go")):  {},
		filepath.Clean(filepath.Join(root, "runtime_support_listingkit.go")): {},
		filepath.Clean(filepath.Join(root, "types.go")):                      {},
	}

	index, err := loadGoFileIndex(root, "")
	if err != nil {
		t.Fatal(err)
	}
	for path, facts := range index.files {
		if strings.HasSuffix(filepath.Base(path), "_test.go") {
			continue
		}
		if _, ok := facts.imports[`"task-processor/internal/listingkit"`]; !ok {
			continue
		}
		if _, ok := allowedFiles[path]; !ok {
			t.Errorf("%s imports task-processor/internal/listingkit; keep app/httpapi ListingKit root dependencies limited to runtime support types and move new feature logic into internal/listingkit/httpapi", path)
		}
	}
}

func TestAppHTTPAPIListingKitHTTPAPIImportsStayAllowlisted(t *testing.T) {
	root := filepath.Join("..", "internal", "app", "httpapi")
	allowedFiles := map[string]struct{}{
		filepath.Clean(filepath.Join(root, "composition_builder.go")):        {},
		filepath.Clean(filepath.Join(root, "feature_module_builders.go")):    {},
		filepath.Clean(filepath.Join(root, "feature_builder_listingkit.go")): {},
		filepath.Clean(filepath.Join(root, "http_modules.go")):               {},
		filepath.Clean(filepath.Join(root, "runtime_login_modules.go")):      {},
		filepath.Clean(filepath.Join(root, "runtime.go")):                    {},
		filepath.Clean(filepath.Join(root, "runtime_deps_methods.go")):       {},
		filepath.Clean(filepath.Join(root, "server.go")):                     {},
		filepath.Clean(filepath.Join(root, "server_auth.go")):                {},
		filepath.Clean(filepath.Join(root, "types.go")):                      {},
	}

	index, err := loadGoFileIndex(root, "")
	if err != nil {
		t.Fatal(err)
	}
	for path, facts := range index.files {
		if strings.HasSuffix(filepath.Base(path), "_test.go") {
			continue
		}
		if _, ok := facts.imports[`"task-processor/internal/listingkit/httpapi"`]; !ok {
			continue
		}
		if _, ok := allowedFiles[path]; !ok {
			t.Errorf("%s imports task-processor/internal/listingkit/httpapi; keep app/httpapi feature-owned ListingKit HTTPAPI dependencies limited to current module, runtime, route, and server assembly files", path)
		}
	}
}

func TestImageAgentEffectPolicyInboundImportsStayOwnedByStoreAndPolicy(t *testing.T) {
	root := filepath.Join("..", "internal", "imageagent")
	violations, err := imageAgentEffectPolicyInboundImportViolations(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(violations) != 0 {
		t.Errorf("imageagent effect-policy inbound imports outside store and effectpolicy: %v", violations)
	}
}

func TestImageAgentEffectPolicyInboundImportGuardDistinguishesAllowedAndExactImports(t *testing.T) {
	root := t.TempDir()
	writeGoSource := func(relativePath, source string) {
		t.Helper()
		path := filepath.Join(root, relativePath)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(source), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	writeGoSource("blocked_single.go", "package imageagent\n\nimport _ \"task-processor/internal/imageagent/effectpolicy\"\n")
	writeGoSource("blocked_grouped.go", "package imageagent\n\nimport (\n\t\"fmt\"\n\t_ \"task-processor/internal/imageagent/effectpolicy\"\n)\n\nvar _ = fmt.Sprintf\n")
	writeGoSource("ignored_test.go", "package imageagent\n\nimport _ \"task-processor/internal/imageagent/effectpolicy\"\n")
	writeGoSource("unrelated.go", "package imageagent\n\nimport _ \"task-processor/internal/imageagent/effectpolicymirror\"\n")
	writeGoSource(filepath.Join("store", "allowed.go"), "package store\n\nimport _ \"task-processor/internal/imageagent/effectpolicy\"\n")
	writeGoSource(filepath.Join("effectpolicy", "allowed.go"), "package effectpolicy\n\nimport _ \"task-processor/internal/imageagent/effectpolicy\"\n")

	violations, err := imageAgentEffectPolicyInboundImportViolations(root)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{
		filepath.Join(root, "blocked_grouped.go"),
		filepath.Join(root, "blocked_single.go"),
	}
	if !reflect.DeepEqual(want, violations) {
		t.Fatalf("inbound import violations = %v, want %v", violations, want)
	}
}

func imageAgentEffectPolicyInboundImportViolations(root string) ([]string, error) {
	index, err := loadGoFileIndex(root, "")
	if err != nil {
		return nil, err
	}

	allowedOwners := map[string]struct{}{
		filepath.Join(root, "store") + string(os.PathSeparator):        {},
		filepath.Join(root, "effectpolicy") + string(os.PathSeparator): {},
	}
	const effectPolicyImport = `"task-processor/internal/imageagent/effectpolicy"`

	var violations []string
	for path, facts := range index.files {
		if strings.HasSuffix(filepath.Base(path), "_test.go") || pathAllowed(path, allowedOwners) {
			continue
		}
		if _, importsEffectPolicy := facts.imports[effectPolicyImport]; importsEffectPolicy {
			violations = append(violations, path)
		}
	}
	sort.Strings(violations)
	return violations, nil
}

func assertNoBannedImportsBelow(t *testing.T, parentRoot, root string, bannedImports []string, allowedFiles map[string]struct{}) {
	t.Helper()
	index, err := loadGoFileSubtreeIndex(parentRoot, root)
	if err != nil {
		t.Fatal(err)
	}
	for path, facts := range index.files {
		if pathAllowed(path, allowedFiles) {
			continue
		}
		for quoted := range facts.imports {
			imported, err := decodeGoImportPath(quoted)
			if err != nil {
				t.Fatal(err)
			}
			for _, literal := range bannedImports {
				banned, err := decodeGoImportPath(literal)
				if err != nil {
					t.Fatal(err)
				}
				if imported == banned {
					t.Errorf("%s imports banned boundary package %q", path, imported)
				}
			}
		}
	}
}

func assertNoBannedImportPrefixesBelow(t *testing.T, parentRoot, root string, bannedPrefixes []string, allowedFiles map[string]struct{}) {
	t.Helper()
	index, err := loadGoFileSubtreeIndex(parentRoot, root)
	if err != nil {
		t.Fatal(err)
	}
	for path, facts := range index.files {
		if strings.HasSuffix(filepath.Base(path), "_test.go") || pathAllowed(path, allowedFiles) {
			continue
		}
		for quoted := range facts.imports {
			imported, err := decodeGoImportPath(quoted)
			if err != nil {
				t.Fatal(err)
			}
			for _, banned := range bannedPrefixes {
				if importMatchesPrefix(imported, banned) {
					t.Errorf("%s imports banned boundary package prefix %s via %s", path, banned, imported)
				}
			}
		}
	}
}

func assertNoBannedImports(t *testing.T, root string, bannedImports []string, allowedFiles map[string]struct{}) {
	t.Helper()

	violations, err := findBannedImportViolations(root, bannedImports, allowedFiles, false)
	if err != nil {
		t.Fatal(err)
	}
	for _, violation := range violations {
		t.Errorf("%s imports banned boundary package %q", violation.path, violation.importPath)
	}
}

func assertNoProductionBannedImports(t *testing.T, root string, bannedImports []string, allowedFiles map[string]struct{}) {
	t.Helper()

	violations, err := findBannedImportViolations(root, bannedImports, allowedFiles, true)
	if err != nil {
		t.Fatal(err)
	}
	for _, violation := range violations {
		t.Errorf("%s imports banned production boundary package %q", violation.path, violation.importPath)
	}
}

func assertNoBannedImportPrefixes(t *testing.T, root string, bannedPrefixes []string, allowedFiles map[string]struct{}) {
	t.Helper()

	index, err := loadGoFileIndex(root, "")
	if err != nil {
		t.Fatal(err)
	}
	for path, facts := range index.files {
		if strings.HasSuffix(filepath.Base(path), "_test.go") || pathAllowed(path, allowedFiles) {
			continue
		}
		for quotedImport := range facts.imports {
			importPath := strings.Trim(quotedImport, `"`)
			for _, bannedPrefix := range bannedPrefixes {
				if importMatchesPrefix(importPath, bannedPrefix) {
					t.Errorf("%s imports banned boundary package prefix %s via %s", path, bannedPrefix, importPath)
				}
			}
		}
	}
}

func assertNoBannedSelectorsOutside(t *testing.T, root, allowedRoot string, bannedSelectors map[string]struct{}) {
	t.Helper()

	index, err := loadGoFileIndex(root, allowedRoot)
	if err != nil {
		t.Fatal(err)
	}
	fset := token.NewFileSet()
	for path, facts := range index.files {
		if !strings.Contains(string(facts.source), "productenrich.") {
			continue
		}
		file, err := parser.ParseFile(fset, path, facts.source, 0)
		if err != nil {
			t.Fatal(err)
		}
		ast.Inspect(file, func(node ast.Node) bool {
			selector, ok := node.(*ast.SelectorExpr)
			if !ok {
				return true
			}
			ident, ok := selector.X.(*ast.Ident)
			if !ok || ident.Name != "productenrich" {
				return true
			}
			if _, banned := bannedSelectors[selector.Sel.Name]; banned {
				t.Errorf("%s uses productenrich.%s compatibility alias; import internal/catalog/canonical directly", path, selector.Sel.Name)
			}
			return true
		})
	}
}

// Parse tracked Go declarations across build tags and nested modules, including tests.
func TestLegacyConsumersStayWithinDrainBaseline(t *testing.T) {
	sources := trackedProductionTextSources(t, []string{"."}, func(path string) bool {
		return strings.HasSuffix(path, ".go") && !strings.Contains("/"+path, "/testdata/")
	})
	violations, err := legacyConsumerViolations(sources, legacyConsumerBaseline)
	if err != nil {
		t.Fatal(err)
	}
	for _, edge := range violations {
		t.Errorf("new legacy consumer: %s; EXTRACT behavior to its current owner or RETIRE the dependency", edge)
	}
	edges, err := phase3BannedImportDeclarationViolations(sources, []string{"task-processor/internal/compatibility", "task-processor/internal/tenantbridge"})
	if err != nil {
		t.Fatal(err)
	}
	actual := make(map[string]bool, len(edges))
	for _, edge := range edges {
		actual[edge] = true
	}
	for edge := range legacyConsumerBaseline {
		if !actual[edge] {
			t.Errorf("drained legacy consumer: remove %s from baseline and depguard exclusions", edge)
		}
	}
}

var currentOwnerLegacyListingKitRoots = []string{
	"product", "listing", "marketplace", "agent", "agentconfig", "commercetool", "console", "businesstask", "storecenter",
}

func TestCurrentOwnersDoNotImportLegacyListingKitRoot(t *testing.T) {
	sources := trackedProductionTextSources(t, []string{"internal"}, func(path string) bool {
		for _, owner := range currentOwnerLegacyListingKitRoots {
			if strings.HasPrefix(path, "internal/"+owner+"/") {
				return strings.HasSuffix(path, ".go") && !strings.Contains(path, "/testdata/")
			}
		}
		return false
	})
	violations, err := currentOwnerLegacyListingKitRootViolations(sources)
	if err != nil {
		t.Fatal(err)
	}
	for _, violation := range violations {
		t.Errorf("%s; EXTRACT behavior into the current owner instead of legacy fallback", violation)
	}
}

func currentOwnerLegacyListingKitRootViolations(sources []listingKitImageBoundarySource) ([]string, error) {
	var violations []string
	for _, source := range sources {
		file, err := parser.ParseFile(token.NewFileSet(), source.path, source.text, parser.ImportsOnly)
		if err != nil {
			return nil, err
		}
		for _, spec := range file.Imports {
			imported, err := decodeGoImportPath(spec.Path.Value)
			if err != nil {
				return nil, err
			}
			if imported == "task-processor/internal/listingkit" {
				violations = append(violations, filepath.ToSlash(source.path)+" -> "+imported)
			}
		}
	}
	sort.Strings(violations)
	return violations, nil
}

// Frozen #605/#31 designs admit current persistence ports and exact official
// DTO transport files. Every admitted edge is separately checked below.
func supplyAdapterBusinessPorts() map[string][]string {
	base := filepath.Join("..", "internal", "integration")
	return map[string][]string{
		filepath.Join(base, "shein", "official_publication.go"):                                     {"task-processor/internal/marketplace/shein/model"},
		filepath.Join(base, "persistence", "listing", "submission", "shein_shelf_policy.go"):        {"task-processor/internal/listing/submission", "task-processor/internal/product/collection"},
		filepath.Join(base, "walmart", "client.go"):                                                 {"task-processor/internal/marketplace/walmart"},
		filepath.Join(base, "walmart", "products.go"):                                               {"task-processor/internal/marketplace/walmart"},
		filepath.Join(base, "persistence", "listing", "official", "walmart_results.go"):             {"task-processor/internal/marketplace/walmart"},
		filepath.Join(base, "persistence", "listing", "submission", "walmart_effects_policy.go"):    {"task-processor/internal/listing/submission", "task-processor/internal/listing/record/target", "task-processor/internal/product/collection", "task-processor/internal/marketplace/walmart"},
		filepath.Join(base, "persistence", "listing", "preparation") + string(os.PathSeparator):     {"task-processor/internal/listing/preparation", "task-processor/internal/product/collection"},
		filepath.Join(base, "persistence", "listing", "completiondraft") + string(os.PathSeparator): {"task-processor/internal/listing/completiondraft", "task-processor/internal/listing/preparation", "task-processor/internal/product/collection"},
		filepath.Join(base, "persistence", "listing", "record") + string(os.PathSeparator):          {"task-processor/internal/listing/record/target", "task-processor/internal/product/collection"},
		filepath.Join(base, "persistence", "listing", "official") + string(os.PathSeparator):        {"task-processor/internal/listing/submission", "task-processor/internal/product/collection"},
		filepath.Join(base, "persistence", "listing", "official", "official_repository.go"):         {"task-processor/internal/listing/record/target"},
		filepath.Join(base, "persistence", "listing", "submission", "temu_effects_policy.go"):       {"task-processor/internal/listing/submission", "task-processor/internal/listing/record/target", "task-processor/internal/product/collection"},
		filepath.Join(base, "persistence", "listing", "submission", "tiktok_effects_policy.go"):     {"task-processor/internal/listing/submission", "task-processor/internal/listing/record/target", "task-processor/internal/product/collection"},
		filepath.Join(base, "shein", "official_goods.go"):                                           {"task-processor/internal/marketplace/shein/goods", "task-processor/internal/marketplace/shein/model"},
		filepath.Join(base, "shein", "official_goods_rules.go"):                                     {"task-processor/internal/marketplace/shein/model"},
		filepath.Join(base, "temu", "official_categories.go"):                                       {"task-processor/internal/marketplace/temu/model"},
		filepath.Join(base, "temu", "official_category_rules.go"):                                   {"task-processor/internal/marketplace/temu/model"},
		filepath.Join(base, "temu", "official_supplier_size.go"):                                    {"task-processor/internal/marketplace/temu/model"},
		filepath.Join(base, "temu", "official_goods.go"):                                            {"task-processor/internal/marketplace/temu/model", "task-processor/internal/marketplace/temu/goods"},
		filepath.Join(base, "tiktokshop", "goods_read.go"):                                          {"task-processor/internal/marketplace/tiktokshop"},
		filepath.Join(base, "tiktokshop", "goods_write.go"):                                         {"task-processor/internal/marketplace/tiktokshop"},
		filepath.Join(base, "temu", "official_observations.go"):                                     {"task-processor/internal/marketplace/temu/model", "task-processor/internal/marketplace/temu/observations", "task-processor/internal/storecenter/observations"},
		filepath.Join(base, "shein", "official_readback.go"):                                        {"task-processor/internal/marketplace/shein/goods", "task-processor/internal/marketplace/shein/model"},
		filepath.Join(base, "shein", "official_observations.go"):                                    {"task-processor/internal/storecenter/observations"},
		filepath.Join(base, "persistence", "storeobservations", "repository.go"):                    {"task-processor/internal/storecenter/observations", "task-processor/internal/marketplace/temu/observations", "task-processor/internal/marketplace/walmart"},
		filepath.Join(base, "persistence", "storeobservations", "schema.go"):                        {"task-processor/internal/storecenter/observations"},
	}
}
func supplyAdapterPortAllowed(path, imported string) bool {
	for admitted, ports := range supplyAdapterBusinessPorts() {
		if path != admitted && !(strings.HasSuffix(admitted, string(os.PathSeparator)) && strings.HasPrefix(path, admitted)) {
			continue
		}
		for _, port := range ports {
			if imported == port {
				return true
			}
		}
	}
	return false
}
func TestSupplyPersistenceAndOfficialTransportOnlyImportAdmittedPorts(t *testing.T) {
	root := filepath.Join("..", "internal", "integration")
	index, err := loadGoFileIndex(root, "")
	if err != nil {
		t.Fatal(err)
	}
	for path, facts := range index.files {
		admitted := false
		for entry := range supplyAdapterBusinessPorts() {
			if path == entry || strings.HasSuffix(entry, string(os.PathSeparator)) && strings.HasPrefix(path, entry) {
				admitted = true
				break
			}
		}
		if !admitted || strings.HasSuffix(path, "_test.go") {
			continue
		}
		for quoted := range facts.imports {
			imp := strings.Trim(quoted, `"`)
			if (importMatchesPrefix(imp, "task-processor/internal/listing") || importMatchesPrefix(imp, "task-processor/internal/marketplace") || importMatchesPrefix(imp, "task-processor/internal/product")) && !supplyAdapterPortAllowed(path, imp) {
				t.Errorf("%s has unadmitted business port %s", path, imp)
			}
		}
	}
	good := filepath.Join(root, "shein", "official_goods.go")
	bad := filepath.Join(root, "shein", "other.go")
	if !supplyAdapterPortAllowed(good, "task-processor/internal/marketplace/shein/goods") || supplyAdapterPortAllowed(bad, "task-processor/internal/marketplace/shein/goods") || supplyAdapterPortAllowed(good, "task-processor/internal/marketplace/shein/goods/child") {
		t.Fatal("official DTO admission must be exact")
	}
	for _, file := range []string{"official_categories.go", "official_category_rules.go", "official_supplier_size.go"} {
		admitted := filepath.Join(root, "temu", file)
		if !supplyAdapterPortAllowed(admitted, "task-processor/internal/marketplace/temu/model") || supplyAdapterPortAllowed(admitted, "task-processor/internal/marketplace/temu/rules") || supplyAdapterPortAllowed(admitted, "task-processor/internal/marketplace/temu/model/child") || supplyAdapterPortAllowed(filepath.Join(root, "temu", "other.go"), "task-processor/internal/marketplace/temu/model") {
			t.Fatal("TEMU official metadata admission must be the exact DTO leaf and files")
		}
	}
}
func TestSupplyAndCollectionHTTPRegistrationKeepsBusinessSiblingsFrameworkFree(t *testing.T) {
	root := filepath.Join("..", "internal")
	allowed := businessHTTPPackages(root)
	for _, tc := range []struct {
		path string
		want bool
	}{{"authz/httpapi/route_authorization.go", true}, {"authz/authorization.go", false}, {"authz/httpapi_extra/other.go", false}, {"app/supplychain/httpapi/http_routes.go", true}, {"app/supplychain/operations.go", false}, {"app/supplychain/httpapi_extra/other.go", false}, {"product/collection/httpapi/handler.go", true}, {"product/collection/service.go", false}, {"product/collection/httpapi_extra/other.go", false}} {
		if pathAllowed(filepath.Join(root, filepath.FromSlash(tc.path)), allowed) != tc.want {
			t.Errorf("registration %s", tc.path)
		}
	}
}

func TestWalmartAdapterBusinessPortsStayExact(t *testing.T) {
	base := filepath.Join("..", "internal", "integration")
	port := "task-processor/internal/marketplace/walmart"
	for _, path := range []string{
		filepath.Join(base, "walmart", "client.go"),
		filepath.Join(base, "walmart", "products.go"),
		filepath.Join(base, "persistence", "listing", "official", "walmart_results.go"),
		filepath.Join(base, "persistence", "listing", "submission", "walmart_effects_policy.go"),
		filepath.Join(base, "persistence", "storeobservations", "repository.go"),
	} {
		if !supplyAdapterPortAllowed(path, port) {
			t.Errorf("approved Walmart adapter %s must consume its current leaf", path)
		}
		for _, bad := range []string{port + "/child", "task-processor/internal/marketplace/temu/model", "task-processor/internal/product/catalog"} {
			if supplyAdapterPortAllowed(path, bad) {
				t.Errorf("%s must not admit %s", path, bad)
			}
		}
		if supplyAdapterPortAllowed(filepath.Join(filepath.Dir(path), "unapproved.go"), port) {
			t.Errorf("%s admission must not extend to sibling files", path)
		}
	}
}
