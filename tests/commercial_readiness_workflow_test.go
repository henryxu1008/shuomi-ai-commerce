package tests

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestListingKitCommercialImageAgentWorkersUseExactSecretAndConfigScope(t *testing.T) {
	base := filepath.Join("..", "deployments", "kubernetes", "listingkit-workbench")
	sharedSecrets := map[string]string{
		"TASK_PROCESSOR_DATABASE_HOST":                  "TASK_PROCESSOR_DATABASE_HOST",
		"TASK_PROCESSOR_DATABASE_PORT":                  "TASK_PROCESSOR_DATABASE_PORT",
		"TASK_PROCESSOR_DATABASE_USER":                  "TASK_PROCESSOR_DATABASE_USER",
		"TASK_PROCESSOR_DATABASE_PASSWORD":              "TASK_PROCESSOR_DATABASE_PASSWORD",
		"TASK_PROCESSOR_DATABASE_NAME":                  "TASK_PROCESSOR_DATABASE_NAME",
		"TASK_PROCESSOR_COMMERCIAL_DATABASE_HOST":       "TASK_PROCESSOR_COMMERCIAL_DATABASE_HOST",
		"TASK_PROCESSOR_COMMERCIAL_DATABASE_PORT":       "TASK_PROCESSOR_COMMERCIAL_DATABASE_PORT",
		"TASK_PROCESSOR_COMMERCIAL_DATABASE_USER":       "TASK_PROCESSOR_COMMERCIAL_DATABASE_USER",
		"TASK_PROCESSOR_COMMERCIAL_DATABASE_PASSWORD":   "TASK_PROCESSOR_COMMERCIAL_DATABASE_PASSWORD",
		"TASK_PROCESSOR_COMMERCIAL_DATABASE_NAME":       "TASK_PROCESSOR_COMMERCIAL_DATABASE_NAME",
		"TASK_PROCESSOR_OPENAI_API_KEY":                 "TASK_PROCESSOR_OPENAI_API_KEY",
		"TASK_PROCESSOR_OPENAI_CLIENTS_IMAGE_API_KEY":   "TASK_PROCESSOR_OPENAI_CLIENTS_IMAGE_API_KEY",
		"TASK_PROCESSOR_OPENAI_CLIENTS_IMAGE_API_STYLE": "TASK_PROCESSOR_OPENAI_CLIENTS_IMAGE_API_STYLE",
		"TASK_PROCESSOR_OPENAI_CLIENTS_IMAGE_BASE_URL":  "TASK_PROCESSOR_OPENAI_CLIENTS_IMAGE_BASE_URL",
		"TASK_PROCESSOR_OPENAI_CLIENTS_IMAGE_MODEL":     "TASK_PROCESSOR_OPENAI_CLIENTS_IMAGE_MODEL",
	}
	sharedConfig := map[string]string{
		"IMAGE_AGENT_TEMPORAL_ENABLED":   "IMAGE_AGENT_TEMPORAL_ENABLED",
		"IMAGE_AGENT_TEMPORAL_ADDRESS":   "IMAGE_AGENT_TEMPORAL_ADDRESS",
		"IMAGE_AGENT_TEMPORAL_NAMESPACE": "IMAGE_AGENT_TEMPORAL_NAMESPACE",
	}
	withArtifactStoreSecrets := func() map[string]string {
		result := map[string]string{}
		for key, value := range sharedSecrets {
			result[key] = value
		}
		result["TASK_PROCESSOR_IMAGEAGENT_ARTIFACT_STORE_S3_ACCESS_KEY_ID"] = "TASK_PROCESSOR_IMAGEAGENT_ARTIFACT_STORE_S3_ACCESS_KEY_ID"
		result["TASK_PROCESSOR_IMAGEAGENT_ARTIFACT_STORE_S3_SECRET_ACCESS_KEY"] = "TASK_PROCESSOR_IMAGEAGENT_ARTIFACT_STORE_S3_SECRET_ACCESS_KEY"
		return result
	}
	withArtifactStoreConfig := func() map[string]string {
		result := map[string]string{}
		for key, value := range sharedConfig {
			result[key] = value
		}
		for _, key := range []string{
			"TASK_PROCESSOR_IMAGEAGENT_ARTIFACT_STORE_ENABLED",
			"TASK_PROCESSOR_IMAGEAGENT_ARTIFACT_STORE_PROVIDER",
			"TASK_PROCESSOR_IMAGEAGENT_ARTIFACT_STORE_PUBLIC_BASE",
			"TASK_PROCESSOR_IMAGEAGENT_ARTIFACT_STORE_S3_BUCKET",
			"TASK_PROCESSOR_IMAGEAGENT_ARTIFACT_STORE_S3_REGION",
			"TASK_PROCESSOR_IMAGEAGENT_ARTIFACT_STORE_S3_ENDPOINT",
			"TASK_PROCESSOR_IMAGEAGENT_ARTIFACT_STORE_S3_USE_PATH_STYLE",
			"TASK_PROCESSOR_IMAGEAGENT_ARTIFACT_STORE_S3_ARTIFACT_MODE",
			"TASK_PROCESSOR_IMAGEAGENT_ARTIFACT_STORE_S3_COS_IMMUTABLE_NON_VERSIONED_BUCKET_POLICY",
		} {
			result[key] = key
		}
		return result
	}

	for _, test := range []struct {
		relativePath string
		wantSecrets  map[string]string
		wantConfig   map[string]string
	}{
		{relativePath: filepath.Join("base", "image-agent-temporal-worker-deployment.yaml"), wantSecrets: withArtifactStoreSecrets(), wantConfig: withArtifactStoreConfig()},
		{relativePath: filepath.Join("base", "image-agent-temporal-worker-v3-deployment.yaml"), wantSecrets: withArtifactStoreSecrets(), wantConfig: withArtifactStoreConfig()},
	} {
		t.Run(test.relativePath, func(t *testing.T) {
			manifest := loadImageAgentWorkloadManifest(t, filepath.Join(base, test.relativePath))
			container := onlyImageAgentContainer(t, manifest)
			if len(container.EnvFrom) != 0 {
				t.Fatalf("%s must use per-key configuration and Secret references, got envFrom=%#v", test.relativePath, container.EnvFrom)
			}
			actualSecrets := map[string]string{}
			actualConfig := map[string]string{}
			for _, variable := range container.Env {
				for _, forbidden := range []string{"ZITADEL", "INVITATION", "TENCENT_SMS", "SMS_WEBHOOK"} {
					if strings.Contains(variable.Name, forbidden) {
						t.Fatalf("%s must not receive forbidden credential/config %q", test.relativePath, variable.Name)
					}
				}
				if variable.ValueFrom == nil {
					t.Fatalf("%s must not contain literal environment value %q", test.relativePath, variable.Name)
				}
				if (variable.ValueFrom.SecretKeyRef == nil) == (variable.ValueFrom.ConfigMapKeyRef == nil) {
					t.Fatalf("%s environment %q must reference exactly one approved source", test.relativePath, variable.Name)
				}
				if ref := variable.ValueFrom.SecretKeyRef; ref != nil {
					if ref.Name != listingKitSharedSecret {
						t.Fatalf("%s references unexpected Secret %q", test.relativePath, ref.Name)
					}
					actualSecrets[variable.Name] = ref.Key
				}
				if ref := variable.ValueFrom.ConfigMapKeyRef; ref != nil {
					if ref.Name != "listingkit-workbench-config" {
						t.Fatalf("%s references unexpected ConfigMap %q", test.relativePath, ref.Name)
					}
					actualConfig[variable.Name] = ref.Key
				}
			}
			if !reflect.DeepEqual(actualSecrets, test.wantSecrets) {
				t.Fatalf("%s Secret allowlist=%#v want=%#v", test.relativePath, actualSecrets, test.wantSecrets)
			}
			if !reflect.DeepEqual(actualConfig, test.wantConfig) {
				t.Fatalf("%s ConfigMap allowlist=%#v want=%#v", test.relativePath, actualConfig, test.wantConfig)
			}
		})
	}

	canaryManifest := loadImageAgentWorkloadFromMultiDoc(t, filepath.Join(base, "release-authority", "listingkit-release-gate-runners.yaml"), "image-agent-temporal-v3-canary-runner")
	if len(canaryManifest.Spec.Template.Spec.InitContainers) != 1 {
		t.Fatal("canary runner must have one release-gate init container")
	}
	canary := canaryManifest.Spec.Template.Spec.InitContainers[0]
	if len(canary.EnvFrom) != 0 {
		t.Fatalf("canary must not use envFrom, got %#v", canary.EnvFrom)
	}
	actualCanaryConfig := map[string]string{}
	for _, variable := range canary.Env {
		if variable.ValueFrom == nil || variable.ValueFrom.ConfigMapKeyRef == nil || variable.ValueFrom.SecretKeyRef != nil {
			t.Fatalf("canary environment %q must be a Temporal ConfigMap key only", variable.Name)
		}
		if variable.ValueFrom != nil && variable.ValueFrom.SecretKeyRef != nil {
			t.Fatalf("canary must not receive Secret key %q", variable.ValueFrom.SecretKeyRef.Key)
		}
		if variable.ValueFrom != nil && variable.ValueFrom.ConfigMapKeyRef != nil {
			actualCanaryConfig[variable.Name] = variable.ValueFrom.ConfigMapKeyRef.Key
		}
	}
	wantCanaryConfig := map[string]string{
		"IMAGE_AGENT_TEMPORAL_ADDRESS":   "IMAGE_AGENT_TEMPORAL_ADDRESS",
		"IMAGE_AGENT_TEMPORAL_NAMESPACE": "IMAGE_AGENT_TEMPORAL_NAMESPACE",
	}
	if !reflect.DeepEqual(actualCanaryConfig, wantCanaryConfig) {
		t.Fatalf("canary ConfigMap allowlist=%#v want=%#v", actualCanaryConfig, wantCanaryConfig)
	}
}

func TestCommercialReadinessWorkflowCollectsPinnedReleaseEvidence(t *testing.T) {
	path := filepath.Join("..", ".github", "workflows", "commercial-readiness.yml")
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read commercial-readiness workflow: %v", err)
	}

	workflow := string(content)
	packageContent, err := os.ReadFile(filepath.Join("..", "web", "listingkit-ui", "package.json"))
	if err != nil {
		t.Fatalf("read frontend package manifest: %v", err)
	}
	var packageManifest struct {
		PackageManager string `json:"packageManager"`
	}
	if err := json.Unmarshal(packageContent, &packageManifest); err != nil {
		t.Fatalf("parse frontend package manifest: %v", err)
	}
	packageManager, _, found := strings.Cut(packageManifest.PackageManager, "@")
	if !found || packageManager == "" {
		t.Fatalf("frontend packageManager must pin a package manager and version, got %q", packageManifest.PackageManager)
	}

	var installCommand, lockfile string
	scriptCommand := func(script string) string { return packageManager + " " + script }
	switch packageManager {
	case "pnpm":
		installCommand = "pnpm install --frozen-lockfile"
		lockfile = "pnpm-lock.yaml"
	case "npm":
		installCommand = "npm ci"
		lockfile = "package-lock.json"
		scriptCommand = func(script string) string {
			if script == "test" {
				return "npm test"
			}
			return "npm run " + script
		}
	default:
		t.Fatalf("commercial-readiness workflow policy does not support package manager %q", packageManager)
	}

	for _, required := range []string{
		"workflow_dispatch:",
		"commit_sha:",
		"^[0-9a-fA-F]{40}$",
		"ref: ${{ inputs.commit_sha }}",
		"go test ./... -count=1",
		"id: race_tests",
		"set -o pipefail",
		"tee evidence/listingadmin-race.log",
		"go test -race ./internal/listingadmin -count=1",
		"make build-all",
		"cache: " + packageManager,
		"cache-dependency-path: web/listingkit-ui/" + lockfile,
		installCommand,
		scriptCommand("lint"),
		scriptCommand("typecheck"),
		scriptCommand("test"),
		scriptCommand("build"),
		"deployments/docker/Dockerfile.product-listing-api",
		"deployments/docker/Dockerfile.listingkit-ui",
		"kustomize build deployments/kubernetes/listingkit-workbench/overlays/prod",
		"actions/upload-artifact@v4",
		"if: ${{ always() }}",
	} {
		if !strings.Contains(workflow, required) {
			t.Errorf("commercial-readiness workflow must contain %q", required)
		}
	}

	if strings.Contains(workflow, "continue-on-error: true") {
		t.Fatal("commercial-readiness workflow must not hide failed validation steps")
	}
	for _, retired := range []string{
		"./internal/app/runtime/listingcontrol", "evidence/listingcontrol-race.log",
		"TestConcurrentClaimForDispatchOnlyOneWorkerWins",
		"TestConcurrentRollbackDispatchOnlyOriginalQueuedClaimIsRestoredOnce",
		"TestConcurrentRecoveryOnlyUpdatesStillEligibleRowsOnce",
	} {
		if strings.Contains(workflow, retired) {
			t.Errorf("commercial-readiness workflow must not reference retired runtime evidence %q", retired)
		}
	}
}

func TestRetiredListingGenerationUsageCanaryIsAbsent(t *testing.T) {
	path := filepath.Join("..", "deployments", "kubernetes", "listingkit-workbench", "base", "configmap.yaml")
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read ListingKit ConfigMap: %v", err)
	}
	var configMap struct {
		Data map[string]string `yaml:"data"`
	}
	if err := yaml.Unmarshal(content, &configMap); err != nil {
		t.Fatalf("parse ListingKit ConfigMap: %v", err)
	}
	for _, key := range []string{"TASK_PROCESSOR_LISTINGKIT_GENERATION_USAGE_LEDGER_ENABLED", "TASK_PROCESSOR_LISTINGKIT_GENERATION_USAGE_LEDGER_TENANT_IDS"} {
		if _, exists := configMap.Data[key]; exists {
			t.Errorf("retired generation usage input remains: %s", key)
		}
	}
	if strings.Contains(string(content), "PAY-042") {
		t.Error("retired generation usage canary instruction remains")
	}
}

func TestListingKitRetainedWorkloadsExcludeInvitationCredentials(t *testing.T) {
	base := filepath.Join("..", "deployments", "kubernetes", "listingkit-workbench", "base")
	sharedSecret, err := os.ReadFile(filepath.Join(base, "secret.example.yaml"))
	if err != nil {
		t.Fatalf("read shared ListingKit Secret: %v", err)
	}
	for _, key := range []string{
		"TASK_PROCESSOR_LISTINGKIT_ZITADEL_MEMBER_INVITATION_TOKEN",
		"TASK_PROCESSOR_LISTINGKIT_ZITADEL_PROJECT_ID",
	} {
		if strings.Contains(string(sharedSecret), key) {
			t.Fatalf("shared ListingKit Secret must not contain %s", key)
		}
	}

	for _, name := range []string{
		"listingkit-ui-deployment.yaml",
		"imgproxy-deployment.yaml",
		"image-agent-temporal-worker-deployment.yaml",
		"image-agent-temporal-worker-v3-deployment.yaml",
	} {
		manifest, err := os.ReadFile(filepath.Join(base, name))
		if err != nil {
			t.Fatalf("read %s: %v", name, err)
		}
		if strings.Contains(string(manifest), "listingkit-member-invitation-secret") {
			t.Errorf("%s must not consume the member invitation Secret", name)
		}
	}

	configMap, err := os.ReadFile(filepath.Join(base, "configmap.yaml"))
	if err != nil {
		t.Fatalf("read ListingKit ConfigMap: %v", err)
	}
	if strings.Contains(string(configMap), "TASK_PROCESSOR_LISTINGKIT_ZITADEL_PROJECT_ID") {
		t.Fatal("ListingKit base ConfigMap must not overwrite the manually provisioned invitation project id during deploy")
	}

	deployWorkflow, err := os.ReadFile(filepath.Join("..", ".github", "workflows", "listingkit-deploy.yml"))
	if err != nil {
		t.Fatalf("read ListingKit deploy workflow: %v", err)
	}
	var workflow map[string]any
	if err := yaml.Unmarshal(deployWorkflow, &workflow); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(workflow, expectedRetiredListingKitWorkflow("API")) {
		t.Fatal("retired release source must reject before reading or mutating invitation credentials")
	}
}

func TestRetiredTencentSMSCredentialsAndRouteAreAbsent(t *testing.T) {
	base := filepath.Join("..", "deployments", "kubernetes", "listingkit-workbench", "base")
	const secretName = "listingkit-tencent-sms-secret"
	requiredKeys := []string{
		"TASK_PROCESSOR_LISTINGKIT_ZITADEL_SMS_SIGNING_KEY",
		"TASK_PROCESSOR_LISTINGKIT_TENCENT_SMS_SECRET_ID",
		"TASK_PROCESSOR_LISTINGKIT_TENCENT_SMS_SECRET_KEY",
		"TASK_PROCESSOR_LISTINGKIT_TENCENT_SMS_APP_ID",
		"TASK_PROCESSOR_LISTINGKIT_TENCENT_SMS_SIGN_NAME",
		"TASK_PROCESSOR_LISTINGKIT_TENCENT_SMS_TEMPLATE_ID",
	}

	sharedSecret, err := os.ReadFile(filepath.Join(base, "secret.example.yaml"))
	if err != nil {
		t.Fatalf("read shared ListingKit Secret: %v", err)
	}
	for _, key := range requiredKeys {
		if strings.Contains(string(sharedSecret), key) {
			t.Errorf("shared ListingKit Secret must not contain %s", key)
		}
	}
	configMap, err := os.ReadFile(filepath.Join(base, "configmap.yaml"))
	if err != nil {
		t.Fatalf("read ListingKit ConfigMap: %v", err)
	}
	for _, key := range requiredKeys {
		if strings.Contains(string(configMap), key) {
			t.Errorf("ListingKit ConfigMap must not contain %s", key)
		}
	}

	for _, name := range []string{
		"listingkit-ui-deployment.yaml",
		"imgproxy-deployment.yaml",
		"image-agent-temporal-worker-deployment.yaml",
		"image-agent-temporal-worker-v3-deployment.yaml",
	} {
		manifest, err := os.ReadFile(filepath.Join(base, name))
		if err != nil {
			t.Fatalf("read %s: %v", name, err)
		}
		if strings.Contains(string(manifest), secretName) {
			t.Errorf("%s must not consume the Tencent SMS Secret", name)
		}
	}

	const webhookPath = "/api/v1/listing-kits/integrations/zitadel/sms"
	for _, ingressPath := range []string{
		filepath.Join(base, "ingress.yaml"),
		filepath.Join(base, "..", "overlays", "prod", "patch-ingress.yaml"),
	} {
		ingress, err := os.ReadFile(ingressPath)
		if err != nil {
			t.Fatalf("read ListingKit Ingress %s: %v", ingressPath, err)
		}
		manifest := strings.ReplaceAll(string(ingress), "\r\n", "\n")
		if strings.Contains(manifest, webhookPath) || strings.Contains(manifest, "product-listing-api") {
			t.Fatalf("Ingress %s must not route to the retired API/SMS endpoint", ingressPath)
		}
	}
}

func TestListingKitReleaseImageRetainsNormalSchemaCLI(t *testing.T) {
	dockerfilePath := filepath.Join("..", "deployments", "docker", "Dockerfile.product-listing-api")
	dockerfile, err := os.ReadFile(dockerfilePath)
	if err != nil {
		t.Fatalf("read product-listing API Dockerfile: %v", err)
	}
	for _, required := range []string{
		"cmd/listingkit-schema-migrate/",
		"/out/listingkit-schema-migrate",
		"/app/listingkit-schema-migrate",
	} {
		if !strings.Contains(string(dockerfile), required) {
			t.Errorf("product-listing API image must contain %q", required)
		}
	}

}
