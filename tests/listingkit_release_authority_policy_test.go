package tests

import (
	"encoding/base64"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"gopkg.in/yaml.v3"
)

const (
	listingKitRepository         = "qq550723504/task-processor"
	listingKitAPIOIDCEnvironment = "listingkit-api-production"
	listingKitUIOIDCEnvironment  = "listingkit-ui-production"
	listingKitAPIOIDCAudience    = "api://listingkit-api-production"
	listingKitUIOIDCAudience     = "api://listingkit-ui-production"
	listingKitAPIWorkflowRef     = listingKitRepository + "/.github/workflows/listingkit-deploy.yml@refs/heads/main"
	listingKitUIWorkflowRef      = listingKitRepository + "/.github/workflows/listingkit-ui-deploy.yml@refs/heads/main"
)

func TestListingKitReleaseAuthorityMachinePolicyOwnsAllSecurityInputs(t *testing.T) {
	t.Parallel()

	path := filepath.Join("..", "deployments", "kubernetes", "listingkit-workbench", "release-authority", "release-policy.yaml")
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read machine release policy: %v", err)
	}

	var policy struct {
		ReleaseAuthority struct {
			Repository string `yaml:"repository"`
			Namespace  string `yaml:"namespace"`
			OIDC       struct {
				Issuer         string `yaml:"issuer"`
				UsernamePrefix string `yaml:"usernamePrefix"`
				RefreshHelper  string `yaml:"refreshHelper"`
				Identities     map[string]struct {
					Environment     string `yaml:"environment"`
					Audience        string `yaml:"audience"`
					Subject         string `yaml:"subject"`
					WorkflowRef     string `yaml:"workflowRef"`
					Job             string `yaml:"job"`
					ProtectedGroups []struct {
						RefreshStep    string   `yaml:"refreshStep"`
						OperationSteps []string `yaml:"operationSteps"`
					} `yaml:"protectedGroups"`
				} `yaml:"identities"`
			} `yaml:"oidc"`
			Tools struct {
				ConftestVersion string `yaml:"conftestVersion"`
				OPAVersion      string `yaml:"opaVersion"`
			} `yaml:"tools"`
			ProducerRouting struct {
				ImageLabel string `yaml:"imageLabel"`
				Annotation string `yaml:"annotation"`
				Contract   string `yaml:"contract"`
			} `yaml:"producerRouting"`
			ReleaseAttestation struct {
				Version              string `yaml:"version"`
				WorkerWireContract   string `yaml:"workerWireContract"`
				WorkerReplayContract string `yaml:"workerReplayContract"`
				SchemaContract       string `yaml:"schemaContract"`
			} `yaml:"releaseAttestation"`
			ReleaseIdentity struct {
				Owner       string `yaml:"owner"`
				Deployment  string `yaml:"deployment"`
				Step        string `yaml:"step"`
				Annotations struct {
					RunID      string `yaml:"runId"`
					RunAttempt string `yaml:"runAttempt"`
					Image      string `yaml:"image"`
					Routing    string `yaml:"routingContract"`
				} `yaml:"annotations"`
			} `yaml:"releaseIdentity"`
			UIAuthorization struct {
				Owner           string `yaml:"owner"`
				SharedConfigMap string `yaml:"sharedConfigMap"`
				ConfigMap       string `yaml:"configMap"`
				Deployment      string `yaml:"deployment"`
				Container       string `yaml:"container"`
				ScopesKey       string `yaml:"scopesKey"`
			} `yaml:"uiAuthorization"`
			RBAC struct {
				Kustomization string   `yaml:"kustomization"`
				Paths         []string `yaml:"paths"`
			} `yaml:"rbac"`
			ReleaseGateRunners struct {
				Manifest  string   `yaml:"manifest"`
				HoldImage string   `yaml:"holdImage"`
				Names     []string `yaml:"names"`
			} `yaml:"releaseGateRunners"`
			ProtectedResources []struct {
				Owner         string   `yaml:"owner"`
				Resource      string   `yaml:"resource"`
				ResourceNames []string `yaml:"resourceNames"`
				Verbs         []string `yaml:"verbs"`
			} `yaml:"protectedResources"`
			Documentation struct {
				Paths              []string `yaml:"paths"`
				CanonicalWorkflows []string `yaml:"canonicalWorkflows"`
			} `yaml:"documentation"`
		} `yaml:"releaseAuthority"`
	}
	if err := yaml.Unmarshal(content, &policy); err != nil {
		t.Fatalf("decode machine release policy: %v", err)
	}

	authority := policy.ReleaseAuthority
	if authority.Repository != "qq550723504/task-processor" || authority.Namespace != "task-processor" {
		t.Fatalf("release policy repository/namespace drifted: %#v", authority)
	}
	if authority.OIDC.Issuer != "https://token.actions.githubusercontent.com" || authority.OIDC.UsernamePrefix != "github:" {
		t.Fatalf("release policy OIDC trust drifted: %#v", authority.OIDC)
	}
	if authority.OIDC.RefreshHelper != "scripts/listingkit-refresh-github-oidc-kubeconfig.sh" {
		t.Fatalf("release policy must own the checked-in fresh OIDC helper, got %q", authority.OIDC.RefreshHelper)
	}
	api := authority.OIDC.Identities["api"]
	ui := authority.OIDC.Identities["ui"]
	if api.Environment != listingKitAPIOIDCEnvironment || ui.Environment != listingKitUIOIDCEnvironment ||
		api.Audience != listingKitAPIOIDCAudience || ui.Audience != listingKitUIOIDCAudience ||
		api.Subject == "" || ui.Subject == "" || api.Subject == ui.Subject ||
		api.WorkflowRef != listingKitAPIWorkflowRef || ui.WorkflowRef != listingKitUIWorkflowRef || api.Job == ui.Job {
		t.Fatalf("release policy must own distinct API/UI identities: api=%#v ui=%#v", api, ui)
	}
	if len(api.ProtectedGroups) != 8 || len(ui.ProtectedGroups) != 1 {
		t.Fatalf("release policy must own every fresh-OIDC protected group: api=%#v ui=%#v", api.ProtectedGroups, ui.ProtectedGroups)
	}
	if authority.Tools.ConftestVersion == "" || authority.Tools.OPAVersion == "" || strings.Contains(authority.Tools.ConftestVersion, "latest") || strings.Contains(authority.Tools.OPAVersion, "latest") {
		t.Fatalf("release policy must pin Conftest and embedded OPA versions: %#v", authority.Tools)
	}
	if authority.ProducerRouting.ImageLabel != "org.opencontainers.image.listingkit.image-agent-routing" ||
		authority.ProducerRouting.Annotation != "listingkit.sh/image-agent-routing-contract" ||
		authority.ProducerRouting.Contract != "image-agent-v3-new-starts-v1" {
		t.Fatalf("release policy must own the immutable v3 producer-routing contract: %#v", authority.ProducerRouting)
	}
	if authority.ReleaseAttestation.Version != "listingkit-api-release-gate/v2" ||
		authority.ReleaseAttestation.WorkerWireContract != "image-agent-workers-v2-v3" ||
		authority.ReleaseAttestation.WorkerReplayContract != "image-agent-replay-v2-v3" ||
		authority.ReleaseAttestation.SchemaContract != "listingkit-schema-additive-v1" {
		t.Fatalf("release policy must own the exact rollback attestation contracts: %#v", authority.ReleaseAttestation)
	}
	if authority.ReleaseIdentity.Owner != "api" || authority.ReleaseIdentity.Deployment != "product-listing-api" ||
		authority.ReleaseIdentity.Step != "Stamp API release identity and restart Pods" ||
		authority.ReleaseIdentity.Annotations.RunID != "listingkit.sh/api-release-run-id" ||
		authority.ReleaseIdentity.Annotations.RunAttempt != "listingkit.sh/api-release-run-attempt" ||
		authority.ReleaseIdentity.Annotations.Image != "listingkit.sh/api-release-image" ||
		authority.ReleaseIdentity.Annotations.Routing != "listingkit.sh/image-agent-routing-contract" {
		t.Fatalf("release policy must own the exact API release identity stamp: %#v", authority.ReleaseIdentity)
	}
	if authority.UIAuthorization.Owner != "ui" || authority.UIAuthorization.SharedConfigMap != "listingkit-workbench-config" ||
		authority.UIAuthorization.ConfigMap != "listingkit-ui-auth-config" || authority.UIAuthorization.Deployment != "listingkit-ui" ||
		authority.UIAuthorization.Container != "listingkit-ui" || authority.UIAuthorization.ScopesKey != "ZITADEL_SCOPES" {
		t.Fatalf("release policy must own the exclusive UI authorization config: %#v", authority.UIAuthorization)
	}
	if authority.RBAC.Kustomization == "" || len(authority.RBAC.Paths) < 6 || len(authority.ProtectedResources) == 0 {
		t.Fatalf("release policy must own RBAC paths and protected resources: %#v", authority.RBAC)
	}
	if authority.ReleaseGateRunners.Manifest != "deployments/kubernetes/listingkit-workbench/release-authority/listingkit-release-gate-runners.yaml" ||
		authority.ReleaseGateRunners.HoldImage != releaseGateHoldImage || len(authority.ReleaseGateRunners.Names) != 3 {
		t.Fatalf("release policy must own the aggregate runner manifest and exact inventory: %#v", authority.ReleaseGateRunners)
	}
	if len(authority.Documentation.Paths) == 0 || len(authority.Documentation.CanonicalWorkflows) != 2 {
		t.Fatalf("release policy must own supported release docs and canonical workflow links: %#v", authority.Documentation)
	}
}

func TestListingKitReleaseAuthorityCrossFilePolicyRejectsNativeSidecarRunner(t *testing.T) {
	t.Parallel()

	workflow := readRepositoryText(t, "..", ".github", "workflows", "listingkit-deploy.yml")
	runners := readRepositoryText(t, "..", "deployments", "kubernetes", "listingkit-workbench", "release-authority", "listingkit-release-gate-runners.yaml")
	mutatedRunners := strings.Replace(runners, "        - name: release-gate\n          image:", "        - name: release-gate\n          restartPolicy: Always\n          image:", 1)
	if mutatedRunners == runners {
		t.Fatal("test fixture did not add restartPolicy to release-gate init container")
	}

	output, runErr := runReleaseAuthorityCrossFilePolicy(t, workflow, "", mutatedRunners)
	if runErr == nil {
		t.Fatalf("cross-file release-authority policy accepted a native-sidecar release-gate runner:\n%s", output)
	}
}

func TestListingKitReleaseAuthorityAcceptsActualRetiredEntrypoints(t *testing.T) {
	api := readRepositoryText(t, "..", ".github", "workflows", "listingkit-deploy.yml")
	ui := readRepositoryText(t, "..", ".github", "workflows", "listingkit-ui-deploy.yml")
	output, err := runReleaseAuthorityCrossFilePolicy(t, api, ui, "")
	if err != nil {
		t.Fatalf("actual retired source bundle rejected: %v\n%s", err, output)
	}
}

func TestListingKitReleaseAuthorityRejectsRetiredEntrypointBypasses(t *testing.T) {
	for _, owner := range []string{"API", "UI"} {
		for _, test := range []struct {
			name   string
			mutate func(map[string]any, map[string]any)
		}{
			{"publish", func(w, j map[string]any) {
				j["steps"] = []any{map[string]any{"run": "docker push example/image:latest\nexit 1\n"}}
			}},
			{"mutation", func(w, j map[string]any) {
				j["steps"] = []any{map[string]any{"run": "kubectl apply -f production.yaml\nexit 1\n"}}
			}},
			{"success", func(w, j map[string]any) { j["steps"] = []any{map[string]any{"run": "exit 0\n"}} }},
			{"id-token", func(w, j map[string]any) { w["permissions"] = map[string]any{"id-token": "write"} }},
			{"job-permissions", func(w, j map[string]any) { j["permissions"] = map[string]any{"contents": "write"} }},
			{"environment", func(w, j map[string]any) { j["environment"] = "listingkit-api-production" }},
			{"secrets", func(w, j map[string]any) { j["env"] = map[string]any{"TOKEN": "${{ secrets.RELEASE_TOKEN }}"} }},
			{"reusable-job", func(w, j map[string]any) { j["uses"] = "./.github/workflows/another.yml" }},
			{"action-step", func(w, j map[string]any) {
				j["steps"] = []any{map[string]any{"uses": "actions/checkout@" + strings.Repeat("a", 40)}}
			}},
			{"second-step", func(w, j map[string]any) {
				j["steps"] = append(j["steps"].([]any), map[string]any{"run": "echo success"})
			}},
			{"second-job", func(w, j map[string]any) {
				w["jobs"].(map[string]any)["publish"] = map[string]any{"runs-on": "ubuntu-latest", "steps": []any{map[string]any{"run": "echo success"}}}
			}},
			{"condition", func(w, j map[string]any) { j["if"] = "false" }},
			{"continue-on-error", func(w, j map[string]any) { j["continue-on-error"] = true }},
			{"automatic-push", func(w, j map[string]any) {
				w["on"].(map[string]any)["push"] = map[string]any{"tags": []any{"listingkit-ui-v*"}}
			}},
			{"automatic-receipt", func(w, j map[string]any) {
				w["on"].(map[string]any)["workflow_run"] = map[string]any{"workflows": []any{"ListingKit API Deploy"}, "types": []any{"completed"}}
			}},
			{"rollback-input", func(w, j map[string]any) {
				w["on"].(map[string]any)["workflow_dispatch"] = map[string]any{"inputs": map[string]any{"rollback_run_id": map[string]any{"type": "string"}}}
			}},
			{"shell-expression", func(w, j map[string]any) {
				j["steps"] = []any{map[string]any{"run": "echo $(kubectl get secrets)\nexit 1\n"}}
			}},
		} {
			t.Run(owner+"/"+test.name, func(t *testing.T) {
				workflow := expectedRetiredListingKitWorkflow(owner)
				job := workflow["jobs"].(map[string]any)["retired"].(map[string]any)
				test.mutate(workflow, job)
				mutated, err := yaml.Marshal(workflow)
				if err != nil {
					t.Fatal(err)
				}
				api := readRepositoryText(t, "..", ".github", "workflows", "listingkit-deploy.yml")
				ui := readRepositoryText(t, "..", ".github", "workflows", "listingkit-ui-deploy.yml")
				if owner == "API" {
					api = string(mutated)
				} else {
					ui = string(mutated)
				}
				output, err := runReleaseAuthorityCrossFilePolicy(t, api, ui, "")
				if err == nil || !strings.Contains(output, strings.ToLower(owner)+" retired release entry must be exactly") {
					t.Fatalf("policy must reject this actual workflow bypass at the retired boundary: err=%v\n%s", err, output)
				}
			})
		}
	}
}

func TestListingKitReleaseAuthorityOIDCConfiguratorFailsClosedOnClaimMismatch(t *testing.T) {
	t.Parallel()

	script, err := filepath.Abs(filepath.Join("..", "scripts", "listingkit-configure-github-oidc-kubeconfig.sh"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(script); err != nil {
		t.Fatalf("OIDC kubeconfig configurator is missing: %v", err)
	}

	now := time.Now().UTC()
	claims := map[string]interface{}{
		"iss":          "https://token.actions.githubusercontent.com",
		"aud":          listingKitAPIOIDCAudience,
		"sub":          "repo:qq550723504/task-processor:environment:" + listingKitAPIOIDCEnvironment,
		"repository":   "qq550723504/task-processor",
		"environment":  listingKitAPIOIDCEnvironment,
		"workflow_ref": listingKitAPIWorkflowRef,
		"nbf":          now.Add(-time.Minute).Unix(),
		"exp":          now.Add(10 * time.Minute).Unix(),
	}

	run := func(t *testing.T, tokenClaims map[string]interface{}, audience, workflowFlag, workflowValue string) ([]byte, string, error) {
		t.Helper()
		dir := t.TempDir()
		tokenPath := filepath.Join(dir, "oidc-token")
		kubeconfigPath := filepath.Join(dir, "kubeconfig")
		if err := os.WriteFile(tokenPath, []byte(unsignedJWT(t, tokenClaims)), 0o600); err != nil {
			t.Fatal(err)
		}
		args := []string{filepath.ToSlash(script),
			"--token-file", filepath.ToSlash(tokenPath),
			"--kubeconfig", filepath.ToSlash(kubeconfigPath),
			"--cluster-server", "https://kubernetes.example.invalid",
			"--cluster-ca-b64", base64.StdEncoding.EncodeToString([]byte("test-ca")),
			"--issuer", "https://token.actions.githubusercontent.com",
			"--audience", audience,
			"--subject", "repo:qq550723504/task-processor:environment:" + listingKitAPIOIDCEnvironment,
			"--repository", listingKitRepository,
			"--environment", listingKitAPIOIDCEnvironment,
			workflowFlag, workflowValue,
		}
		command := exec.Command(preflightBash(t), args...)
		output, runErr := command.CombinedOutput()
		return output, kubeconfigPath, runErr
	}

	t.Run("wrong audience writes no kubeconfig", func(t *testing.T) {
		output, kubeconfigPath, runErr := run(t, claims, listingKitUIOIDCAudience, "--workflow-ref", listingKitAPIWorkflowRef)
		if runErr == nil {
			t.Fatalf("wrong audience unexpectedly configured Kubernetes credentials: %s", output)
		}
		if _, statErr := os.Stat(kubeconfigPath); !os.IsNotExist(statErr) {
			t.Fatalf("claim mismatch must fail before writing kubeconfig, statErr=%v", statErr)
		}
	})

	for _, test := range []struct {
		name        string
		workflowRef interface{}
	}{
		{name: "feature branch", workflowRef: listingKitRepository + "/.github/workflows/listingkit-deploy.yml@refs/heads/feature/release"},
		{name: "tag", workflowRef: listingKitRepository + "/.github/workflows/listingkit-deploy.yml@refs/tags/listingkit-api-v-test"},
		{name: "trusted ref prefix lookalike", workflowRef: listingKitAPIWorkflowRef + "-attacker"},
		{name: "missing ref", workflowRef: nil},
	} {
		t.Run(test.name+" writes no kubeconfig", func(t *testing.T) {
			mismatched := cloneOIDCClaims(claims)
			if test.workflowRef == nil {
				delete(mismatched, "workflow_ref")
			} else {
				mismatched["workflow_ref"] = test.workflowRef
			}
			output, kubeconfigPath, runErr := run(t, mismatched, listingKitAPIOIDCAudience, "--workflow-ref", listingKitAPIWorkflowRef)
			if runErr == nil {
				t.Fatalf("untrusted workflow ref unexpectedly configured Kubernetes credentials: %s", output)
			}
			if _, statErr := os.Stat(kubeconfigPath); !os.IsNotExist(statErr) {
				t.Fatalf("workflow ref mismatch must fail before writing kubeconfig, statErr=%v", statErr)
			}
		})
	}

	t.Run("path only configuration is rejected", func(t *testing.T) {
		output, kubeconfigPath, runErr := run(t, claims, listingKitAPIOIDCAudience, "--workflow-path", ".github/workflows/listingkit-deploy.yml")
		if runErr == nil {
			t.Fatalf("path-only OIDC configuration unexpectedly created Kubernetes credentials: %s", output)
		}
		if _, statErr := os.Stat(kubeconfigPath); !os.IsNotExist(statErr) {
			t.Fatalf("path-only configuration must fail before writing kubeconfig, statErr=%v", statErr)
		}
	})

	t.Run("exact claims create ephemeral kubeconfig without logging token", func(t *testing.T) {
		output, kubeconfigPath, runErr := run(t, claims, listingKitAPIOIDCAudience, "--workflow-ref", listingKitAPIWorkflowRef)
		if runErr != nil {
			t.Fatalf("exact OIDC claims failed: %v\n%s", runErr, output)
		}
		kubeconfig, readErr := os.ReadFile(kubeconfigPath)
		if readErr != nil {
			t.Fatalf("read ephemeral kubeconfig: %v", readErr)
		}
		token := unsignedJWT(t, claims)
		if !strings.Contains(string(kubeconfig), token) {
			t.Fatal("ephemeral kubeconfig is missing the short-lived OIDC token")
		}
		if strings.Contains(string(output), token) {
			t.Fatal("OIDC token must never be printed")
		}
	})
}

func TestListingKitReleaseAuthorityBindsWorkflowRefToMain(t *testing.T) {
	t.Parallel()

	authenticationPath := filepath.Join("..", "deployments", "kubernetes", "listingkit-workbench", "release-authority", "kubernetes-authentication-config.example.yaml")
	authentication, err := os.ReadFile(authenticationPath)
	if err != nil {
		t.Fatal(err)
	}
	authenticationText := string(authentication)
	for _, workflowRef := range []string{listingKitAPIWorkflowRef, listingKitUIWorkflowRef} {
		if !strings.Contains(authenticationText, workflowRef) {
			t.Errorf("Kubernetes AuthenticationConfiguration must bind exact workflow ref %q", workflowRef)
		}
	}
	if strings.Contains(authenticationText, "workflow_ref.startsWith") {
		t.Error("Kubernetes AuthenticationConfiguration must not authorize a workflow-ref prefix")
	}

}

func TestListingKitReleaseAuthorityNegativeFixturesAreCheckedIn(t *testing.T) {
	t.Parallel()

	root := filepath.Join("..", "policy", "listingkit-release-authority", "fixtures", "negative")
	for _, name := range []string{
		"kube-config-fallback.yaml",
		"shared-environment.yaml",
		"shared-subject.yaml",
		"missing-id-token.yaml",
		"wrong-audience.yaml",
		"workflow-job-owner-drift.yaml",
		"workflow-step-owner-drift.yaml",
		"job-reusable-workflow.yaml",
		"step-composite-action.yaml",
		"with-target-bypass.yaml",
		"excess-rbac-verbs.yaml",
		"excess-rbac-resources.yaml",
		"excess-rbac-resource-names.yaml",
		"policy-drift.yaml",
		"release-identity-owner-drift.yaml",
		"workflow-ref-feature-branch.yaml",
		"workflow-ref-tag.yaml",
		"workflow-ref-path-prefix-lookalike.yaml",
		"workflow-ref-missing.yaml",
		"trusted-workflow-ref-policy-drift.yaml",
		"mutable-action-ref.yaml",
	} {
		if _, err := os.Stat(filepath.Join(root, name)); err != nil {
			t.Errorf("missing release-authority negative fixture %s: %v", name, err)
		}
	}
}

func cloneOIDCClaims(source map[string]interface{}) map[string]interface{} {
	clone := make(map[string]interface{}, len(source))
	for key, value := range source {
		clone[key] = value
	}
	return clone
}

func unsignedJWT(t *testing.T, claims map[string]interface{}) string {
	t.Helper()
	header, err := json.Marshal(map[string]string{"alg": "RS256", "typ": "JWT"})
	if err != nil {
		t.Fatal(err)
	}
	payload, err := json.Marshal(claims)
	if err != nil {
		t.Fatal(err)
	}
	return base64.RawURLEncoding.EncodeToString(header) + "." + base64.RawURLEncoding.EncodeToString(payload) + ".test-signature"
}

func TestListingKitReleaseAuthorityHasNoLongLivedKubeconfigFallback(t *testing.T) {
	t.Parallel()

	for _, name := range []string{"listingkit-deploy.yml", "listingkit-ui-deploy.yml"} {
		path := filepath.Join("..", ".github", "workflows", name)
		content, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read %s: %v", path, err)
		}
		if strings.Contains(string(content), "KUBE_CONFIG") {
			t.Errorf("%s retains a long-lived KUBE_CONFIG credential path", name)
		}

	}
	helper, err := os.ReadFile(filepath.Join("..", "scripts", "listingkit-refresh-github-oidc-kubeconfig.sh"))
	if err != nil {
		t.Fatal(err)
	}
	for _, required := range []string{"ACTIONS_ID_TOKEN_REQUEST_URL", "ACTIONS_ID_TOKEN_REQUEST_TOKEN", "listingkit-configure-github-oidc-kubeconfig.sh"} {
		if !strings.Contains(string(helper), required) {
			t.Errorf("fresh OIDC helper is missing GitHub-issued credential contract %q", required)
		}
	}
}

func TestListingKitReleaseAuthorityBootstrapIsStandaloneAndCreateFree(t *testing.T) {
	t.Parallel()

	root := filepath.Join("..", "deployments", "kubernetes", "listingkit-workbench", "release-authority")
	for _, name := range []string{
		"kustomization.yaml",
		"release-policy.yaml",
		"kubernetes-authentication-config.example.yaml",
		"listingkit-api-release-role.yaml",
		"listingkit-api-release-rolebinding.yaml",
		"listingkit-ui-release-role.yaml",
		"listingkit-ui-release-rolebinding.yaml",
	} {
		if _, err := os.Stat(filepath.Join(root, name)); err != nil {
			t.Errorf("release-authority bootstrap is missing %s: %v", name, err)
		}
	}

	for _, name := range []string{"listingkit-api-release-role.yaml", "listingkit-ui-release-role.yaml"} {
		path := filepath.Join(root, name)
		content, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		var role struct {
			Rules []struct {
				ResourceNames []string `yaml:"resourceNames"`
				Verbs         []string `yaml:"verbs"`
			} `yaml:"rules"`
		}
		if err := yaml.Unmarshal(content, &role); err != nil {
			t.Fatalf("decode %s: %v", path, err)
		}
		for i, rule := range role.Rules {
			if len(rule.ResourceNames) == 0 {
				t.Errorf("%s rule %d is not constrained by resourceNames", name, i)
			}
			for _, verb := range rule.Verbs {
				if verb == "create" || verb == "*" || verb == "deletecollection" {
					t.Errorf("%s rule %d grants forbidden top-level verb %q", name, i, verb)
				}
			}
		}
	}
}

func runReleaseAuthorityCrossFilePolicy(t *testing.T, mutatedWorkflow, mutatedUIWorkflow, mutatedRunnerManifest string) (string, error) {
	t.Helper()

	repoRoot, err := filepath.Abs(filepath.Join(".."))
	if err != nil {
		t.Fatal(err)
	}
	fixtureRoot := t.TempDir()
	workflowPath := filepath.Join(fixtureRoot, ".github", "workflows", "listingkit-deploy.yml")
	if err := os.MkdirAll(filepath.Dir(workflowPath), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(workflowPath, []byte(mutatedWorkflow), 0o600); err != nil {
		t.Fatal(err)
	}
	uiWorkflowPath := ".github/workflows/listingkit-ui-deploy.yml"
	if mutatedUIWorkflow != "" {
		uiFixturePath := filepath.Join(fixtureRoot, ".github", "workflows", "listingkit-ui-deploy.yml")
		if err := os.WriteFile(uiFixturePath, []byte(mutatedUIWorkflow), 0o600); err != nil {
			t.Fatal(err)
		}
		uiWorkflowPath = "/fixture/.github/workflows/listingkit-ui-deploy.yml"
	}
	runnerManifestPath := "deployments/kubernetes/listingkit-workbench/release-authority/listingkit-release-gate-runners.yaml"
	if mutatedRunnerManifest != "" {
		runnerFixturePath := filepath.Join(fixtureRoot, "deployments", "kubernetes", "listingkit-workbench", "release-authority", "listingkit-release-gate-runners.yaml")
		if err := os.MkdirAll(filepath.Dir(runnerFixturePath), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(runnerFixturePath, []byte(mutatedRunnerManifest), 0o600); err != nil {
			t.Fatal(err)
		}
		runnerManifestPath = "/fixture/deployments/kubernetes/listingkit-workbench/release-authority/listingkit-release-gate-runners.yaml"
	}

	arguments := []string{
		"run", "--rm",
		"--volume", repoRoot + ":/project",
		"--volume", fixtureRoot + ":/fixture",
		"--workdir", "/project",
		"openpolicyagent/conftest@sha256:5fd81e332d7e4bc01daf3ef35371800a9a9720a30c0c37a78de0c5fbe4b6d622",
		"test", "--combine",
		"--namespace", "listingkit_release_authority",
		"--policy", "policy/listingkit-release-authority",
		"deployments/kubernetes/listingkit-workbench/release-authority/release-policy.yaml",
		"deployments/kubernetes/listingkit-workbench/release-authority/kubernetes-authentication-config.example.yaml",
		"deployments/kubernetes/listingkit-workbench/release-authority/listingkit-api-release-role.yaml",
		"deployments/kubernetes/listingkit-workbench/release-authority/listingkit-api-release-rolebinding.yaml",
		"deployments/kubernetes/listingkit-workbench/release-authority/listingkit-ui-release-role.yaml",
		"deployments/kubernetes/listingkit-workbench/release-authority/listingkit-ui-release-rolebinding.yaml",
		"deployments/kubernetes/listingkit-workbench/release-authority/kustomization.yaml",
		runnerManifestPath,
		"deployments/kubernetes/listingkit-workbench/base/configmap.yaml",
		"deployments/kubernetes/listingkit-workbench/base/listingkit-ui-auth-config.yaml",
		"deployments/kubernetes/listingkit-workbench/base/listingkit-ui-deployment.yaml",
		"/fixture/.github/workflows/listingkit-deploy.yml",
		uiWorkflowPath,
	}
	command := exec.Command("docker", arguments...)
	command.Dir = repoRoot
	output, runErr := command.CombinedOutput()
	return string(output), runErr
}
