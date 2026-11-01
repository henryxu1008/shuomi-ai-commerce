package record

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

func TestOnlyCurrentAdapterOwnsListingRecordTable(t *testing.T) {
	root := filepath.Join("..", "..", "..")
	err := filepath.WalkDir(filepath.Join(root, "internal"), func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		raw, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		normalized := filepath.ToSlash(path)
		if (strings.Contains(string(raw), "listing_shein_records") || strings.Contains(string(raw), "listing_shein_record_operations")) && !strings.Contains(normalized, "/app/listingrecordstore/") {
			if !strings.HasSuffix(normalized, "/integration/persistence/product/acquisition/runtime_permissions.go") {
				t.Errorf("second table owner: %s", path)
			} else {
				// The existing Product role catalog admits this owner's immutable tables;
				// only these exact privilege literals are allowed, never business SQL.
				allowed := map[string]string{
					"runtimePermissionsFor":   `,('listing_shein_records','SELECT'),('listing_shein_records','INSERT'),('listing_shein_record_operations','SELECT'),('listing_shein_record_operations','INSERT')`,
					"GrantRuntimePermissions": "GRANT SELECT,INSERT ON public.listing_shein_records,public.listing_shein_record_operations TO source_acquisition_runtime",
				}
				syntax, e := parser.ParseFile(token.NewFileSet(), path, raw, 0)
				if e != nil {
					return e
				}
				for _, decl := range syntax.Decls {
					name := ""
					if f, ok := decl.(*ast.FuncDecl); ok {
						name = f.Name.Name
					}
					ast.Inspect(decl, func(node ast.Node) bool {
						literal, ok := node.(*ast.BasicLit)
						if !ok || literal.Kind != token.STRING {
							return true
						}
						value, e := strconv.Unquote(literal.Value)
						if e == nil && (strings.Contains(value, "listing_shein_records") || strings.Contains(value, "listing_shein_record_operations")) && allowed[name] != value {
							t.Errorf("business SQL or widened offline privilege outside owner: %s (%s)", path, name)
						}
						return true
					})
				}
			}
		}
		isDraftS1Owner := strings.Contains(normalized, "/listing/record/") ||
			strings.Contains(normalized, "/marketplace/shein/draft/") ||
			strings.Contains(normalized, "/app/listingrecordstore/") ||
			strings.HasSuffix(normalized, "/app/httpapi/shein_records_application.go") ||
			strings.HasSuffix(normalized, "/app/httpapi/shein_records_handler.go") || strings.HasSuffix(normalized, "/app/httpapi/current_offline_records.go")
		if !isDraftS1Owner {
			return nil
		}
		file, err := parser.ParseFile(token.NewFileSet(), path, raw, parser.ImportsOnly)
		if err != nil {
			return err
		}
		for _, item := range file.Imports {
			pkg, _ := strconv.Unquote(item.Path.Value)
			if pkg == "task-processor/internal/listingkit" || strings.HasPrefix(pkg, "task-processor/internal/compatibility/") || strings.HasPrefix(pkg, "task-processor/internal/tenantbridge") {
				t.Errorf("legacy dependency in %s: %s", path, pkg)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}
