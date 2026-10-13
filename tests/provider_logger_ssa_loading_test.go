package tests

import (
	"fmt"
	"path/filepath"
	"strings"
	"testing"
)

func TestRepositoryProviderSSALoadsOwnedCrossModuleHelpersWithoutDependencySyntax(t *testing.T) {
	root := t.TempDir()
	dependency := t.TempDir()
	writeModuleFixtureFile(t, dependency, "go.mod", "module example/dependency\n\ngo 1.26.0\n")
	writeModuleFixtureFile(t, dependency, "value.go", "package dependency\n\nvar Value = \"bound\"\n")
	writeModuleFixtureFile(t, root, "go.mod", fmt.Sprintf("module example/root\n\ngo 1.26.0\n\nrequire example/dependency v0.0.0\nreplace example/dependency => %q\n", filepath.ToSlash(dependency)))
	writeModuleFixtureFile(t, root, "openai/provider.go", providerLoggerFixturePackage("openai"))
	writeModuleFixtureFile(t, root, "helpers/config.go", `package helpers

import (
	"fmt"
	"example/dependency"
	"example/root/openai"
)

func Bound() *openai.ClientConfig {
	fmt.Sprint(dependency.Value)
	return &openai.ClientConfig{Logger: openai.AdaptLogrus(&openai.Entry{})}
}

func Unbound() *openai.ClientConfig { return &openai.ClientConfig{} }
`)
	writeModuleFixtureFile(t, root, "nested/go.mod", fmt.Sprintf("module example/root/nested\n\ngo 1.26.0\n\nrequire (\nexample/root v0.0.0\nexample/dependency v0.0.0\n)\nreplace example/root => ..\nreplace example/dependency => %q\n", filepath.ToSlash(dependency)))
	writeModuleFixtureFile(t, root, "nested/caller.go", `package nested

import (
	"example/root/helpers"
	"example/root/openai"
)

func good() { openai.NewClient(helpers.Bound()) }
func bad() { openai.NewClient(helpers.Unbound()) }
`)
	loaded := loadRepositoryModulePackages(t, root, filepath.Join(root, "nested", "go.mod"))
	owned := make(map[string]bool)
	for _, pkg := range loaded {
		owned[pkg.PkgPath] = len(pkg.Syntax) > 0
	}
	for _, path := range []string{"example/root/nested", "example/root/helpers", "example/root/openai"} {
		if !owned[path] {
			t.Errorf("owned source package %s missing from SSA roots", path)
		}
	}
	for _, pkg := range providerPackageGraph(loaded) {
		if (pkg.PkgPath == "fmt" || pkg.PkgPath == "example/dependency") && len(pkg.Syntax) != 0 {
			t.Errorf("unrelated dependency %s has %d syntax trees; dependency type information is sufficient", pkg.PkgPath, len(pkg.Syntax))
		}
		if pkg.PkgPath == "example/dependency" && (pkg.Types == nil || pkg.Types.Scope().Lookup("Value") == nil) {
			t.Error("dependency export type information missing")
		}
	}
	violations := loggerContractSSAViolations(loaded, fixtureProviderLoggerRules()[:1], nil)
	if len(violations) != 1 || !strings.Contains(violations[0], "caller.go") {
		t.Fatalf("violations = %v, want the unbound call rejected and the owned cross-module helper accepted", violations)
	}
}
