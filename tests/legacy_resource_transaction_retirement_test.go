package tests

import (
	"bytes"
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
