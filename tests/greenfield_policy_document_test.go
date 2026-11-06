package tests

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGreenfieldNoLegacyMigrationPolicyIsConsistent(t *testing.T) {
	decision := readGreenfieldPolicyDocument(t, filepath.Join("..", "docs", "product", "greenfield-no-legacy-migration.md"))
	for _, required := range []string{
		"PD-GREENFIELD-NO-LEGACY-MIGRATION-2026-09-08",
		"全新安装、空业务数据",
		"旧数据迁移",
		"旧 ID 映射",
		"旧 Service 包装",
		"tenantbridge consumer",
		"双读、双写、同步或第二事实源",
		"具体、当前的外部可观察契约证据",
		"正常开发不默认加载历史归档",
		"当前没有旧系统兼容、迁移、历史数据承接或旧执行续跑义务",
		"对现行产品、安全、权限、事务、幂等、资源和恢复行为的检查继续保留",
	} {
		if !strings.Contains(decision, required) {
			t.Errorf("greenfield decision must state %q", required)
		}
	}

	for _, path := range []string{
		filepath.Join("..", "AGENTS.md"),
		filepath.Join("..", "docs", "engineering", "issue-driven-development.md"),
		filepath.Join("..", "docs", "architecture", "project-boundaries.md"),
		filepath.Join("..", "docs", "product", "product-sourcing-handoff.md"),
	} {
		if !strings.Contains(readGreenfieldPolicyDocument(t, path), "PD-GREENFIELD-NO-LEGACY-MIGRATION-2026-09-08") {
			t.Errorf("%s must reference the current greenfield decision", path)
		}
	}

	agents := readGreenfieldPolicyDocument(t, filepath.Join("..", "AGENTS.md"))
	if strings.Contains(agents, "必须先停下来建立显式、可评审的 Exception") {
		t.Error("AGENTS must not retain an executable legacy compatibility Exception path")
	}
	if !strings.Contains(agents, "只有用户新的明确产品决定可以改变本禁令") {
		t.Error("AGENTS must reserve any change to the greenfield prohibition to a new explicit user product decision")
	}
}

func TestHistoricalIdentityPreparationToolsStayRetired(t *testing.T) {
	for _, relative := range []string{
		"cmd/source-account-ownership-preflight",
		"internal/integration/persistence/sourceaccount/ownershipmigration",
		"internal/tenantbridge/bootstrap",
	} {
		t.Run(relative, func(t *testing.T) {
			err := filepath.WalkDir(filepath.Join("..", filepath.FromSlash(relative)), func(path string, entry os.DirEntry, err error) error {
				if os.IsNotExist(err) {
					return nil
				}
				if err != nil {
					return err
				}
				if !entry.IsDir() {
					t.Errorf("retired identity preparation artifact remains: %s", path)
				}
				return nil
			})
			if err != nil {
				t.Fatal(err)
			}
		})
	}
	for _, name := range []string{"source-account-ownership-preflight.ps1", "yudao-zitadel-migration-dry-run.ps1"} {
		if _, err := os.Stat(filepath.Join("..", "scripts", name)); !os.IsNotExist(err) {
			t.Errorf("retired identity preparation script must be absent: %s", name)
		}
	}
}

func readGreenfieldPolicyDocument(t *testing.T, path string) string {
	t.Helper()
	contents, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(contents)
}

func TestHistoricalOwnerScopeToolsStayRetired(t *testing.T) {
	for _, relative := range []string{
		"cmd/listingkit-owner-scope-dry-run",
		"cmd/listingkit-owner-scope-exceptions",
		"internal/app/runtime/listingkitownerreconcile",
		"internal/app/runtime/listingkitownerexceptions",
	} {
		files := trackedFiles(t, relative)
		if len(files) != 0 {
			t.Errorf("retired owner-scope tool has tracked artifacts: %v", files)
		}
	}
	for _, name := range []string{"listingkit-owner-scope-dry-run.ps1", "listingkit-owner-scope-dry-run.Tests.ps1", "listingkit-owner-scope-exceptions.ps1"} {
		if _, err := os.Stat(filepath.Join("..", "scripts", name)); !os.IsNotExist(err) {
			t.Errorf("retired owner-scope wrapper remains: %s", name)
		}
	}
	files, err := filepath.Glob(filepath.Join("..", "internal", "listingkit", "ownerreconcile", "*.go"))
	if err != nil {
		t.Fatal(err)
	}
	for _, path := range files {
		source, err := parser.ParseFile(token.NewFileSet(), path, nil, 0)
		if err != nil {
			t.Fatal(err)
		}
		for _, declaration := range source.Decls {
			function, ok := declaration.(*ast.FuncDecl)
			if !ok {
				continue
			}
			switch function.Name.Name {
			case "ApplyUnique", "ValidateApprovedExceptionReport", "InsertSystemOwnedExceptions":
				t.Errorf("%s exposes retired historical owner mutation %s", path, function.Name.Name)
			}
		}
	}
}
