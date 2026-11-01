package tests

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLegacyStoreSourceAdmissionAndAuditTailIsRetired(t *testing.T) {
	assertRestoredDebugPathsAbsent(t,
		"internal/storecenter/source_access.go",
		"internal/storecenter/source_access_test.go")
	global := map[string]bool{"SourcingStoreReader": true, "ValidateSourcingStoreAccess": true}
	storeOnly := map[string]bool{
		"auditFailureFor": true, "stableFailureFromAudit": true, "sourcingStoreReader": true,
		"TestSourcingStoreAccessFailsClosed": true, "TestSourcingStoreAccessRechecksLifecycleOnReplay": true,
		"TestSourcingStoreAccessRejectsTypedNilDependency": true,
	}
	// Scan repository Go sources, including tests/build-tagged files. Exclude
	// local state and third-party/cache trees, never repository source owners.
	err := filepath.WalkDir("..", func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			switch entry.Name() {
			case ".git", ".local", "node_modules", "vendor", ".next", "__pycache__":
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") {
			return nil
		}
		file, err := parser.ParseFile(token.NewFileSet(), path, nil, parser.SkipObjectResolution)
		if err != nil {
			return err
		}
		inStore := strings.HasPrefix(filepath.ToSlash(path), "../internal/storecenter/")
		ast.Inspect(file, func(node ast.Node) bool {
			if id, ok := node.(*ast.Ident); ok && (global[id.Name] || inStore && storeOnly[id.Name]) {
				t.Errorf("retired Store source/audit identifier %s in %s", id.Name, path)
			}
			return true
		})
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestStoreSourceAuditRetirementKeepsCurrentOwners(t *testing.T) {
	for path, required := range map[string][]string{
		"internal/storecenter/audit.go": {
			"AuditEvent", "AuditRepository", "GormAuditRepository", "ErrAuditIdentityMismatch",
			"AuditActionStoreCreated", "AuditOutcomeFailed", "AuditFailureAlreadyExists", "AuditFailureDependencyUnavailable",
			"normalizeAuditEvent", "auditRecordFromEvent", "auditEventFromRecord", "sameAuditSemanticPayload", "newAuditEvent",
		},
		"internal/storecenter/service.go": {"Service", "NewService"},
		"internal/storecenter/product_execution.go": {
			"ProductExecutionSubject", "ProductExecutionReader", "ProductPurposePublish",
			"ReadProductExecution", "readOfficialMaterial", "productExecutionReadError",
		},
		"internal/storecenter/query_service.go":          {"Get", "List"},
		"internal/app/httpapi/current_store_center.go":   {"buildCurrentStoreCenterModule", "currentStoreMemberAuthorizer"},
		"internal/app/storecenter/product_access.go":     {"OfficialProductAccess", "MerchantHandle", "Authorize", "credential", "callMerchant"},
		"internal/storecenter/product_execution_test.go": {"TestProductExecutionPreservesRetiredSourceAccessSafety", "TestProductExecutionRechecksStoreStateOnNextRead"},
	} {
		file, err := parser.ParseFile(token.NewFileSet(), filepath.Join("..", filepath.FromSlash(path)), nil, parser.SkipObjectResolution)
		if err != nil {
			t.Fatal(err)
		}
		declared := map[string]bool{}
		ast.Inspect(file, func(node ast.Node) bool {
			switch declaration := node.(type) {
			case *ast.FuncDecl:
				declared[declaration.Name.Name] = true
			case *ast.TypeSpec:
				declared[declaration.Name.Name] = true
			case *ast.ValueSpec:
				for _, name := range declaration.Names {
					declared[name.Name] = true
				}
			}
			return true
		})
		for _, name := range required {
			if !declared[name] {
				t.Errorf("current Store owner or mapped regression %s missing in %s", name, path)
			}
		}
	}
}

// Approved design e78c30a37d589e3b93341b9a14b431e26e461a2a:
// documentation bodies remain suffixes; native tests remain an intact prefix.
func TestStoreSourceAuditRetirementFrozenHistoryStaysIntact(t *testing.T) {
	for _, frozen := range []struct {
		path   string
		length int
		hash   string
		prefix bool
	}{
		{"docs/architecture/legacy-store-source-admission-audit-tail-retirement-boundary.md", 21017, "1f31760a09d563b348281f9cfd5767916bf3fd368ca9badeaadf30f8b502f90e", false},
		{"docs/refactoring/legacy-register.md", 159507, "c2d62f98b6a2b63c71e03a82f9fd2c544a790b2119f077041fad80d8c3a923a4", false},
		{"docs/architecture/project-boundaries.md", 57374, "d99a39fd3a91f2bb1ce0a4bc4372a17961733660f42e94685221e559a7b3d8e5", false},
		{"docs/architecture/README.md", 159062, "7ad78e9611219672c3e94dc9a0adbd2901c57ea58cf7d3ce12095c7b0b67e1c9", false},
		{"docs/architecture/store-center-current-application-v1.md", 30310, "040dce7be9608abde62e4e2416ab8411996d301579e061a71aaa1c2dafa5cc6b", false},
		{"internal/storecenter/product_execution_test.go", 8132, "9534b66c0be23f580e40dcac3b4747031b8b6a1d32c9732a449da997f612fd54", true},
	} {
		raw, err := os.ReadFile(filepath.Join("..", filepath.FromSlash(frozen.path)))
		if err != nil {
			t.Fatal(err)
		}
		raw = bytes.ReplaceAll(raw, []byte("\r\n"), []byte("\n"))
		raw = restoreRelocatedFrozenDocumentNavigation(frozen.path, raw)
		if len(raw) < frozen.length {
			t.Errorf("frozen Store source/audit history truncated: %s", frozen.path)
			continue
		}
		part := raw[len(raw)-frozen.length:]
		if frozen.prefix {
			part = raw[:frozen.length]
		}
		if fmt.Sprintf("%x", sha256.Sum256(part)) != frozen.hash {
			t.Errorf("frozen Store source/audit history changed: %s", frozen.path)
		}
	}
}
