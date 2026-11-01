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

func TestLegacyResourceTransactionClosureIsRetired(t *testing.T) {
	assertRestoredDebugPathsAbsent(t,
		"internal/ledger/orgresource/reservation.go",
		"internal/ledger/orgresource/settlement.go",
		"internal/ledger/orgresource/reservation_test.go",
		"internal/ledger/orgresource/settlement_test.go",
		"internal/integration/orgresource/reservation_repository.go",
		"internal/integration/orgresource/settlement_repository.go",
		"internal/integration/orgresource/store_service_executor.go",
		"internal/integration/orgresource/reservation_repository_test.go",
		"internal/integration/orgresource/settlement_repository_test.go",
		"internal/integration/orgresource/store_service_executor_test.go")

	banned := map[string]bool{}
	for _, name := range []string{
		"ReservationService", "NewReservationService", "SettlementService", "NewSettlementService",
		"GormReservationRepository", "NewGormReservationRepository", "TransactionalReservationOwnerStore",
		"OwnerAttemptIdentity", "OwnerAttemptSnapshot", "OwnerReservationBinding", "OwnerTerminalProof",
		"ReservationAuthorizer", "ReservationExecutor", "ReservationAuthorization", "ReserveInput",
		"ReservationReplay", "ReservationExecution", "ReservationSnapshot", "ReservationResult",
		"OwnerAttemptState", "SettlementAuthorizer", "SettlementExecutor", "SettlementInput",
		"SettlementReplay", "SettlementExecution", "SettlementSnapshot", "SettlementResult",
		"SettlementDecision", "SettlementCommit", "SettlementRelease", "OperationSettleReservation",
		"TransactionalStoreServiceStore", "storeServiceTransactionRunner", "storeServiceFailureSnapshot",
		"testOwnerAttemptRow", "testOwnerStore", "newReservationTestService", "newSettlementTestServices",
	} {
		banned[name] = true
	}
	for _, root := range []string{"internal", "cmd", "tests", "tools", "scripts"} {
		err := filepath.WalkDir(filepath.Join("..", root), func(path string, entry fs.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if entry.IsDir() || !strings.HasSuffix(path, ".go") {
				return nil
			}
			file, err := parser.ParseFile(token.NewFileSet(), path, nil, parser.SkipObjectResolution)
			if err != nil {
				return err
			}
			ast.Inspect(file, func(node ast.Node) bool {
				var name string
				switch n := node.(type) {
				case *ast.FuncDecl:
					name = n.Name.Name
				case *ast.TypeSpec:
					name = n.Name.Name
				case *ast.ValueSpec:
					for _, value := range n.Names {
						if banned[value.Name] {
							t.Errorf("retired resource declaration %s in %s", value.Name, path)
						}
					}
				case *ast.CallExpr:
					switch call := n.Fun.(type) {
					case *ast.Ident:
						name = call.Name
					case *ast.SelectorExpr:
						name = call.Sel.Name
					}
				}
				if banned[name] || (strings.Contains(filepath.ToSlash(path), "/integration/orgresource/") &&
					(name == "StoreServiceExecutor" || name == "NewStoreServiceExecutor")) {
					t.Errorf("retired resource protocol %s in %s", name, path)
				}
				return true
			})
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
	}
}

func TestResourceRetirementKeepsCurrentOwners(t *testing.T) {
	for path, required := range map[string][]string{
		"internal/ledger/orgresource/reservation_state.go":                {"ReservationState", "ReservationReserved", "ReservationCommitted", "ReservationReleased", "ReservationReconciliationRequired", "OperationReserve", "ErrReservationOwnerNotRegistered", "ErrOwnerScopeMismatch", "ErrOwnerNotReservable", "ErrInsufficientBalance", "ErrOwnerNotTerminal", "ErrInvalidOwnerProof", "ErrReservationNotFound"},
		"internal/integration/orgresource/image_generation_settlement.go": {"imageSettlementDecision", "calculateImageGenerationSettlement", "applyPositiveCredit(ctx, tx,"},
		"internal/integration/orgresource/image_generation_repository.go": {"calculateImageGenerationSettlement(tx.Statement.Context, tx,", "Reason: string(decision)"},
		"internal/app/httpapi/current_store_center.go":                    {"storeapp.NewServiceLifecycleExecutor(repo, charges, connections)"},
		"internal/storecenter/gorm_repository.go":                         {"LockServiceState", "ApplyServiceState"},
	} {
		raw, err := os.ReadFile(filepath.Join("..", filepath.FromSlash(path)))
		if err != nil {
			t.Error(err)
			continue
		}
		for _, text := range required {
			if !bytes.Contains(raw, []byte(text)) {
				t.Errorf("current resource owner missing %q: %s", text, path)
			}
		}
	}
}

// Frozen approved bodies remain suffixes; current progress may be prepended.
func TestResourceRetirementFrozenDocumentSuffixesStayIntact(t *testing.T) {
	for _, doc := range []struct {
		path   string
		length int
		hash   string
	}{
		{"docs/architecture/legacy-resource-transaction-closure-retirement-boundary.md", 22289, "901abd915c67645234343129fbe33435dc1f33cdd4839a4f3b9a6fa4c9b059fd"},
		{"docs/refactoring/legacy-register.md", 156056, "d225481094d98b26947a9c991346b0b7e5d3e71e45923e927225f354ffa24509"},
		{"docs/architecture/project-boundaries.md", 54450, "0d70694e1bf8a0d22053ba199e1b054f694b0d0064ec99ee35ff03324abfc38a"},
		{"docs/architecture/member-resource-allocation-v1.md", 28776, "65b3bf2b6aa80e560f9244dd62c967e7b91510b8efa72f6eb281e52049ff1ca2"},
		{"docs/architecture/store-center-current-application-v1.md", 29644, "af903e3e26a9af52ddbad95c6ff83a8a40be9ce586aa63d632f9f4256178255f"},
		{"docs/engineering/2026-09-25-issue487-main-image-execution-design.md", 40572, "6db0747b6cf12c3df2c5ed27e298fb3b7df9d20fbd0fefd040c1c763d899173f"},
		{"docs/architecture/legacy-subscription-metering-closure-retirement-boundary.md", 43013, "1d80b25e70cad2753d02f52ac627a9e17b720e1ee900245f786c2ad1d68d9a6b"},
		{"docs/archive/designs/2026-09-03-shuomi-resource-ledger-store-design.md", 11132, "91a1c4628d15e8fa043a7cd1631798749eb3e1ab1645e06735888dfa90fafaf7"},
	} {
		raw, err := os.ReadFile(filepath.Join("..", filepath.FromSlash(doc.path)))
		if err != nil {
			t.Fatal(err)
		}
		raw = bytes.ReplaceAll(raw, []byte("\r\n"), []byte("\n"))
		raw = restoreRelocatedFrozenDocumentNavigation(doc.path, raw)
		if len(raw) < doc.length {
			t.Errorf("frozen document truncated: %s", doc.path)
			continue
		}
		if fmt.Sprintf("%x", sha256.Sum256(raw[len(raw)-doc.length:])) != doc.hash {
			t.Errorf("frozen resource history changed: %s", doc.path)
		}
	}
}
