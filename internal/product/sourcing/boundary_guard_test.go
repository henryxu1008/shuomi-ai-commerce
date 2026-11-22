package sourcing

import (
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
)

func TestProductSourcingDoesNotDependOnListingKitMarketplaceOrRuntime(t *testing.T) {
	t.Parallel()

	forbiddenPrefixes := []string{
		"github.com/henryxu1008/shuomi-ai-commerce/internal/listingkit",
		"github.com/henryxu1008/shuomi-ai-commerce/internal/marketplace",
		"github.com/henryxu1008/shuomi-ai-commerce/internal/publishing/shein",
		"github.com/henryxu1008/shuomi-ai-commerce/internal/workspace/shein",
		"github.com/henryxu1008/shuomi-ai-commerce/internal/app",
		"github.com/henryxu1008/shuomi-ai-commerce/internal/httpbootstrap",
		"github.com/henryxu1008/shuomi-ai-commerce/internal/httproute",
		"github.com/henryxu1008/shuomi-ai-commerce/internal/infra",
		"github.com/henryxu1008/shuomi-ai-commerce/internal/platform",
	}
	assertProductSourcingDoesNotImportPrefixes(t, productSourcingPackageDir(t), forbiddenPrefixes)
}

func TestProductSourcingDependsOnlyOnNeutralProductFacts(t *testing.T) {
	t.Parallel()

	forbiddenPrefixes := []string{
		"github.com/henryxu1008/shuomi-ai-commerce/internal/asset",
		"github.com/henryxu1008/shuomi-ai-commerce/internal/crawler",
		"github.com/henryxu1008/shuomi-ai-commerce/internal/integration",
		"github.com/henryxu1008/shuomi-ai-commerce/internal/listingkit",
		"github.com/henryxu1008/shuomi-ai-commerce/internal/model",
		"github.com/henryxu1008/shuomi-ai-commerce/internal/product/asset",
		"github.com/henryxu1008/shuomi-ai-commerce/internal/product/enrichment",
		"github.com/henryxu1008/shuomi-ai-commerce/internal/product/review/proposal",
		"github.com/henryxu1008/shuomi-ai-commerce/internal/product/image",
		"github.com/henryxu1008/shuomi-ai-commerce/internal/productenrich",
	}
	assertProductSourcingDoesNotImportPrefixes(t, productSourcingPackageDir(t), forbiddenPrefixes)
}

func productSourcingPackageDir(t *testing.T) string {
	t.Helper()

	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	return filepath.Dir(filename)
}

func assertProductSourcingDoesNotImportPrefixes(t *testing.T, dir string, forbiddenPrefixes []string) {
	t.Helper()
	assertProductSourcingDoesNotImportPrefixesExcept(t, dir, forbiddenPrefixes, nil)
}

func assertProductSourcingDoesNotImportPrefixesExcept(t *testing.T, dir string, forbiddenPrefixes []string, allowedImports map[string]struct{}) {
	t.Helper()

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("read package dir: %v", err)
	}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".go") || strings.HasSuffix(entry.Name(), "_test.go") {
			continue
		}
		path := filepath.Join(dir, entry.Name())
		fset := token.NewFileSet()
		file, err := parser.ParseFile(fset, path, nil, parser.ImportsOnly)
		if err != nil {
			t.Fatalf("parse imports for %s: %v", path, err)
		}
		for _, imported := range file.Imports {
			importPath, err := strconv.Unquote(imported.Path.Value)
			if err != nil {
				t.Fatalf("unquote import %s: %v", imported.Path.Value, err)
			}
			if _, allowed := allowedImports[importPath]; allowed {
				continue
			}
			for _, forbidden := range forbiddenPrefixes {
				if strings.HasPrefix(importPath, forbidden) {
					t.Fatalf("%s imports forbidden boundary dependency %q", path, importPath)
				}
			}
		}
	}
}
