package tests

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestListingKitTaskBackendSourcesAreRetired(t *testing.T) {
	rootFiles, err := filepath.Glob(filepath.Join("..", "internal", "listingkit", "*.go"))
	if err != nil {
		t.Fatal(err)
	}
	if len(rootFiles) != 0 {
		t.Errorf("mixed ListingKit root must be absent, found %d Go files", len(rootFiles))
	}
	for _, relative := range []string{
		"internal/listingkit/api", "internal/listingkit/httpapi", "internal/listingkit/core",
		"internal/listingkit/temporal", "internal/listingkit/workflow", "internal/listingkit/workspace",
		"internal/listingkit/sheinadapter",
		"internal/app/httpapi/bootstrap.go", "internal/app/httpapi/image_agent_asset_catalog.go",
		"internal/pkg/httpapicmd", "cmd/product-listing-api", "scripts/listingkit-shein-pod-image-index-backfill",
	} {
		t.Run(relative, func(t *testing.T) {
			path := filepath.Join("..", filepath.FromSlash(relative))
			err := filepath.WalkDir(path, func(name string, entry os.DirEntry, err error) error {
				if os.IsNotExist(err) {
					return nil
				}
				if err != nil {
					return err
				}
				if !entry.IsDir() && strings.HasSuffix(name, ".go") {
					t.Errorf("retired Task backend source remains: %s", name)
				}
				return nil
			})
			if err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestCurrentCommandsDoNotCompileLegacyBusinessBackends(t *testing.T) {
	for _, command := range []string{"current-application", "image-agent-temporal-worker", "source-account-registry-schema-init"} {
		t.Run(command, func(t *testing.T) {
			cmd := exec.Command("go", "list", "-deps", "./cmd/"+command)
			cmd.Dir = ".."
			output, err := cmd.CombinedOutput()
			if err != nil {
				t.Fatalf("go list current command: %v\n%s", err, output)
			}
			for _, dependency := range strings.Fields(string(output)) {
				for _, retired := range []string{"task-processor/internal/listingkit", "task-processor/internal/tenantbridge", "task-processor/internal/compatibility", "task-processor/internal/integration/persistence/sourceaccount/ownershipmigration"} {
					if dependency == retired || strings.HasPrefix(dependency, retired+"/") {
						t.Errorf("%s still compiles retired business dependency %s", command, dependency)
					}
				}
			}
		})
	}
}

func TestStandaloneSheinWorkerDoesNotCompileCurrentHTTPApplication(t *testing.T) {
	cmd := exec.Command("go", "list", "-deps", "./cmd/shein-login-worker")
	cmd.Dir = ".."
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("go list standalone SHEIN worker: %v\n%s", err, output)
	}
	for _, dependency := range strings.Fields(string(output)) {
		if dependency == "task-processor/internal/app/httpapi" {
			t.Fatal("standalone SHEIN worker must use its own command runtime, not the current HTTP application assembly")
		}
		if dependency == "task-processor/internal/tenantbridge" || strings.HasPrefix(dependency, "task-processor/internal/tenantbridge/") {
			t.Errorf("standalone SHEIN worker still compiles legacy numeric tenant transport %s", dependency)
		}
	}
	for _, name := range []string{"shein_login_worker.go", "shein_login_worker_test.go"} {
		if _, err := os.Stat(filepath.Join("..", "internal", "app", "httpapi", name)); !os.IsNotExist(err) {
			t.Fatalf("shared HTTP worker source %s must be retired, without a forwarding wrapper", name)
		}
	}
	for _, name := range []string{"api.go", "api_test.go", "http_module.go", "http_module_test.go", "tenant_context.go", "tenant_context_test.go"} {
		if _, err := os.Stat(filepath.Join("..", "internal", "sheinlogin", name)); !os.IsNotExist(err) {
			t.Errorf("unused SHEIN login HTTP source %s must be retired; the standalone worker retains its service and health endpoint", name)
		}
	}
}

func TestLegacyReleaseIdentityPreflightIsRetired(t *testing.T) {
	for _, relative := range []string{
		"cmd/listingkit-identity-preflight", "internal/app/runtime/listingkitidentitypreflight",
		"internal/listingkit/identitypreflight", "internal/listingkit/ownerreconcile",
		"internal/listingkit/userdirectory", "internal/tenantbridge",
		"deployments/docker/Dockerfile.listingkit-identity-preflight",
		"deployments/kubernetes/listingkit-workbench/jobs/listingkit-identity-preflight-job.yaml",
		"scripts/listingkit-identity-preflight-job.sh", "scripts/tests/listingkit-identity-preflight-job-test.sh",
		"scripts/listingkit-clean-legacy-identity-secret.sh", "scripts/tests/listingkit-clean-legacy-identity-secret-test.sh",
	} {
		if _, err := os.Stat(filepath.Join("..", filepath.FromSlash(relative))); !os.IsNotExist(err) {
			t.Errorf("retired identity-preflight artifact remains: %s (stat: %v)", relative, err)
		}
	}
	for _, relative := range []string{
		"deployments/docker/Dockerfile.product-listing-api",
		"deployments/kubernetes/listingkit-workbench/release-authority/release-policy.yaml",
		"deployments/kubernetes/listingkit-workbench/release-authority/listingkit-api-release-role.yaml",
		"deployments/kubernetes/listingkit-workbench/release-authority/listingkit-release-gate-runners.yaml",
	} {
		content, err := os.ReadFile(filepath.Join("..", filepath.FromSlash(relative)))
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(string(content), "listingkit-identity-preflight") {
			t.Errorf("%s still packages or authorizes retired identity preflight", relative)
		}
	}
}

func TestListingAdminLegacyHTTPProjectionIsRetired(t *testing.T) {
	files, err := filepath.Glob(filepath.Join("..", "internal", "listingadmin", "*handler*.go"))
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 0 {
		t.Errorf("retired ListingAdmin HTTP projection remains: %v", files)
	}
	for _, name := range []string{"request_context.go", "list_query_scope.go"} {
		if _, err := os.Stat(filepath.Join("..", "internal", "listingadmin", name)); !os.IsNotExist(err) {
			t.Fatalf("retired ListingAdmin HTTP helper %s must be absent", name)
		}
	}
}

func TestCurrentBuildDoesNotPackageRetiredTaskEntrypoints(t *testing.T) {
	for _, relative := range []string{"Makefile", "deployments/docker/Dockerfile.product-listing-api", "scripts/test-fast.ps1"} {
		contents, err := os.ReadFile(filepath.Join("..", filepath.FromSlash(relative)))
		if err != nil {
			t.Fatal(err)
		}
		for _, retired := range []string{"./cmd/product-listing-api ", "./cmd/product-listing-api\"", "cmd/product-listing-api/", "./internal/listingkit ", "./internal/listingkit\"", "listingkit-shein-pod-image-index-backfill"} {
			if strings.Contains(string(contents), retired) {
				t.Errorf("%s still builds or packages retired entrypoint %q", relative, retired)
			}
		}
	}
}

func TestRetiredLocalLaunchersRejectBeforeOperations(t *testing.T) {
	for _, name := range []string{"start-listingkit-local-api.ps1", "start-listingkit-local-dev.ps1"} {
		content, err := os.ReadFile(filepath.Join("..", "scripts", name))
		if err != nil {
			t.Fatal(err)
		}
		source := string(content)
		retired := strings.Index(source, "throw \"Legacy ListingKit")
		firstFunction := strings.Index(source, "function ")
		if retired < 0 || firstFunction < 0 || retired > firstFunction {
			t.Errorf("%s must reject the retired entry before process, config or connection operations", name)
		}
	}
	if _, err := os.Stat(filepath.Join("..", "scripts", "start-listingkit-api-local-replay.ps1")); !os.IsNotExist(err) {
		t.Fatal("dedicated retired API replay entry must be absent")
	}
}
