package tests

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

func expectedRetiredListingKitWorkflow(owner string) map[string]any {
	return map[string]any{
		"name":        "ListingKit " + owner + " Deploy",
		"on":          map[string]any{"workflow_dispatch": map[string]any{}},
		"permissions": map[string]any{},
		"jobs": map[string]any{"retired": map[string]any{
			"runs-on": "ubuntu-latest",
			"steps":   []any{map[string]any{"run": "echo \"::error::Legacy ListingKit " + owner + " release and rollback are retired; current-application production release is NOT_READY.\"\nexit 1\n"}},
		}},
	}
}

func TestListingKitRetiredReleaseEntrypointsHaveOnlyFixedFailure(t *testing.T) {
	for file, owner := range map[string]string{"listingkit-deploy.yml": "API", "listingkit-ui-deploy.yml": "UI"} {
		content, err := os.ReadFile(filepath.Join("..", ".github", "workflows", file))
		if err != nil {
			t.Fatal(err)
		}
		var actual map[string]any
		if err := yaml.Unmarshal(content, &actual); err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(actual, expectedRetiredListingKitWorkflow(owner)) {
			t.Errorf("%s must have only the exact permission-free manual failure entry", file)
		}
	}
	content, err := os.ReadFile(filepath.Join("..", "scripts", "build-push-deploy-listingkit-workbench.ps1"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(string(content)) != `throw "Legacy ListingKit API/UI release and rollback are retired; current-application production release is NOT_READY."` {
		t.Error("retired PowerShell wrapper must contain only its unconditional literal throw")
	}
}

// TestListingKitProductionReleaseAuthorityRejectsCallerChosenProductionInputs
// catches a release workflow that can be redirected to a tag, arbitrary commit,
// or caller-provided image.  Those values are not provenance; only the exact
// main workflow commit or a previously verified release attestation is.

// TestListingKitProductionReleaseAuthorityUsesFreshOIDCAndImmutableActions
// catches the expired-token pattern: a single OIDC credential is reused across
// multiple 900-second gates or a mutable third-party action can change code.

// TestListingKitUIReleaseHasAnExclusiveAuthorizationConfigMap catches a UI
// principal that can rewrite API/worker runtime configuration or shares a
// ConfigMap with a different release owner.
func TestListingKitUIReleaseHasAnExclusiveAuthorizationConfigMap(t *testing.T) {
	uiManifest, err := os.ReadFile(filepath.Join("..", "deployments", "kubernetes", "listingkit-workbench", "base", "listingkit-ui-deployment.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(uiManifest), "listingkit-ui-auth-config") {
		t.Fatal("UI must consume its dedicated authorization ConfigMap")
	}
	sharedConfig, err := os.ReadFile(filepath.Join("..", "deployments", "kubernetes", "listingkit-workbench", "base", "configmap.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	uiConfig, err := os.ReadFile(filepath.Join("..", "deployments", "kubernetes", "listingkit-workbench", "base", "listingkit-ui-auth-config.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(sharedConfig), "ZITADEL_SCOPES") || !strings.Contains(string(uiConfig), "ZITADEL_SCOPES") {
		t.Fatal("ZITADEL_SCOPES must exist only in the UI-owned ConfigMap")
	}
	role, err := os.ReadFile(filepath.Join("..", "deployments", "kubernetes", "listingkit-workbench", "release-authority", "listingkit-ui-release-role.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(role), "listingkit-workbench-config") || !strings.Contains(string(role), "listingkit-ui-auth-config") {
		t.Fatal("UI release Role must mutate only listingkit-ui-auth-config")
	}
}

// TestListingKitUIReleaseRequestsExactProjectRoleClaim prevents the production
// release workflow from replacing provisioned project scopes with a generic
// role-only scope set that cannot identify the trusted ListingKit project.
func TestListingKitUIReleaseRequestsExactProjectRoleClaim(t *testing.T) {
	uiConfig, err := os.ReadFile(filepath.Join("..", "deployments", "kubernetes", "listingkit-workbench", "base", "listingkit-ui-auth-config.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(uiConfig), "ZITADEL_PROJECT_ID:") {
		t.Fatal("UI authorization ConfigMap must carry the public trusted ZITADEL project ID")
	}

}
