package tests

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestLegacySubscriptionMeteringClosureIsRetired(t *testing.T) {
	roots := []string{"internal/listingsubscription", "internal/accountallocation", "internal/integration/persistence/accountallocation", "internal/app/schema/accountallocation", "internal/integration/openmeter"}
	for _, path := range append(append([]string{}, roots...), []string{"deployments/docker/referrals-compose/terraform/commercial-schema.sql", "internal/app/httpapi/account_resource_allocation_module.go", "internal/app/httpapi/commercial_read_module.go", "internal/app/httpapi/commercial_subscription_adapter.go", "internal/app/httpapi/subscription_purchase_recovery.go", "internal/app/httpapi/subscription_purchase_recovery_test.go", "internal/app/worker/imageagent/review_invocation.go", "internal/commercial/billing/subscription_service.go", "internal/integration/persistence/commercialbilling/subscription_purchase_postgres_test.go", "internal/integration/persistence/commercialbilling/subscription_repository_test.go", "internal/integration/persistence/commercialbilling/subscription_service_test.go", "scripts/lib/openmeter-poc.ps1", "scripts/openmeter-poc.Tests.ps1", "scripts/run-openmeter-poc.ps1", "tests/imageagent_review_invocation_integration_test.go"}...) {
		if _, err := os.Stat(filepath.Join("..", filepath.FromSlash(path))); !os.IsNotExist(err) {
			t.Errorf("retired subscription/metering artifact remains: %s: %v", path, err)
		}
	}
	for _, path := range trackedFiles(t, ".") {
		if !strings.HasSuffix(path, ".go") {
			continue
		}
		file, err := parser.ParseFile(token.NewFileSet(), filepath.Join("..", filepath.FromSlash(path)), nil, 0)
		if err != nil {
			t.Fatal(err)
		}
		for _, imp := range file.Imports {
			imported, err := decodeGoImportPath(imp.Path.Value)
			if err != nil {
				t.Fatal(err)
			}
			for _, root := range roots {
				if importMatchesPrefix(imported, "task-processor/"+root) {
					t.Errorf("%s imports retired owner %s", path, imported)
				}
			}
		}
		if strings.HasSuffix(path, "_test.go") || strings.HasPrefix(path, "tests/testdata/") {
			continue
		}
		ast.Inspect(file, func(n ast.Node) bool {
			var name string
			switch d := n.(type) {
			case *ast.TypeSpec:
				name = d.Name.Name
			case *ast.FuncDecl:
				name = d.Name.Name
			case *ast.Field:
				for _, f := range d.Names {
					if f.Name == "CommercialDatabase" {
						t.Errorf("%s retains retired Token database configuration", path)
					}
				}
			}
			switch name {
			case "OrganizationReviewOptions", "OrganizationReviewPricing", "recordedReview", "AIInvocationUsageAdapter", "SubscriptionPurchasePort", "SubscriptionOrderStore", "SubscriptionQuoteStore", "SubscriptionOfferCatalog", "CreateSubscriptionOrder", "EnableSubscriptionPurchases", "RecoverSubscriptionPurchases", "buildCommercialReadModuleFromDatabase", "buildAccountResourceAllocationModule":
				t.Errorf("%s recreates retired declaration %s", path, name)
			}
			return true
		})
	}
	for _, path := range []string{"Makefile", "scripts/test-fast.ps1", "internal/core/config/loader_builder.go", "deployments/docker/account-compose/bootstrap.sh", "deployments/docker/account-compose/init.sh", "deployments/docker/account-compose/business-db-init.sh", "deployments/docker/account-compose/image-trial-init.sh", "deployments/docker/acquisition-compose/bootstrap.sh", "deployments/docker/acquisition-compose/init.sh", "deployments/docker/referrals-compose/bootstrap.sh", "deployments/docker/referrals-compose/init.sh", "deployments/docker/acquisition-compose/docker-compose.yml", "deployments/kubernetes/listingkit-workbench/base/secret.example.yaml", "go.mod"} {
		data, err := os.ReadFile(filepath.Join("..", filepath.FromSlash(path)))
		if err != nil {
			t.Fatal(err)
		}
		for _, old := range []string{"./internal/listingsubscription", "TASK_PROCESSOR_COMMERCIAL_DATABASE_", "commercial_token_reader_password", "github.com/openmeterio/openmeter", "terraform/commercial-schema.sql", "commercial-reader-password", `"commercialDatabase":`, "\ncommercialDatabase:"} {
			if strings.Contains(string(data), old) {
				t.Errorf("%s retains retired entry %s", path, old)
			}
		}
	}
	for _, path := range []string{"internal/integration/orgresource/balance_read.go", "internal/storecenter/service_summary.go", "internal/app/httpapi/unified_commercial_read.go", "internal/app/worker/imageagent/organization_main_admission.go", "internal/aicapability/store/gorm_invocation_recorder.go", "internal/app/runtime/currentapplication/store_observations.go", "internal/integration/shein/official_observations.go"} {
		if _, err := os.Stat(filepath.Join("..", filepath.FromSlash(path))); err != nil {
			t.Errorf("current owner missing %s: %v", path, err)
		}
	}
	// Removing the old SQL mount must preserve the current initializer's
	// credential, generated configuration, and completion-state mounts.
	data, err := os.ReadFile(filepath.Join("..", "deployments", "docker", "acquisition-compose", "docker-compose.yml"))
	if err != nil {
		t.Fatal(err)
	}
	var compose struct {
		Services map[string]struct {
			Volumes []string `yaml:"volumes"`
		} `yaml:"services"`
	}
	if err := yaml.Unmarshal(data, &compose); err != nil {
		t.Fatal(err)
	}
	volumes := compose.Services["schema-init"].Volumes
	expected := []string{"schema-state:/schema-state", "source-db-owner-secret:/secrets/source-owner:ro", "source-runtime-secret:/secrets/source-runtime:ro", "commercial-db-owner-secret:/secrets/commercial-owner:ro", "commercial-runtime-secret:/secrets/commercial-runtime:ro", "product-db-owner-secret:/secrets/product-owner:ro", "product-runtime-secret:/secrets/product-runtime:ro", "runtime-secrets:/runtime", "frontend-secrets:/frontend"}
	if len(volumes) != len(expected) {
		t.Errorf("acquisition schema-init current mounts changed: got %v", volumes)
	}
	for _, want := range expected {
		count := 0
		for _, volume := range volumes {
			if volume == want {
				count++
			}
		}
		if count != 1 {
			t.Errorf("acquisition schema-init must retain exactly one %q mount, got %d", want, count)
		}
	}
}
