package tests

import (
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
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
			Tools      struct {
				ConftestVersion string `yaml:"conftestVersion"`
				OPAVersion      string `yaml:"opaVersion"`
			} `yaml:"tools"`
			UIAuthorization struct {
				Owner           string `yaml:"owner"`
				SharedConfigMap string `yaml:"sharedConfigMap"`
				ConfigMap       string `yaml:"configMap"`
				Deployment      string `yaml:"deployment"`
				Container       string `yaml:"container"`
				ScopesKey       string `yaml:"scopesKey"`
			} `yaml:"uiAuthorization"`
			Assembly struct {
				Kustomization string `yaml:"kustomization"`
			} `yaml:"assembly"`
			ReleaseGateRunners struct {
				Manifest  string   `yaml:"manifest"`
				HoldImage string   `yaml:"holdImage"`
				Names     []string `yaml:"names"`
			} `yaml:"releaseGateRunners"`
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
	if authority.Tools.ConftestVersion == "" || authority.Tools.OPAVersion == "" || strings.Contains(authority.Tools.ConftestVersion, "latest") || strings.Contains(authority.Tools.OPAVersion, "latest") {
		t.Fatalf("release policy must pin Conftest and embedded OPA versions: %#v", authority.Tools)
	}
	if authority.UIAuthorization.Owner != "ui" || authority.UIAuthorization.SharedConfigMap != "listingkit-workbench-config" ||
		authority.UIAuthorization.ConfigMap != "listingkit-ui-auth-config" || authority.UIAuthorization.Deployment != "listingkit-ui" ||
		authority.UIAuthorization.Container != "listingkit-ui" || authority.UIAuthorization.ScopesKey != "ZITADEL_SCOPES" {
		t.Fatalf("release policy must own the exclusive UI authorization config: %#v", authority.UIAuthorization)
	}
	if authority.Assembly.Kustomization != "deployments/kubernetes/listingkit-workbench/release-authority/kustomization.yaml" {
		t.Fatalf("release policy must own the governed source assembly: %#v", authority.Assembly)
	}
	var assembly struct {
		Namespace string   `yaml:"namespace"`
		Resources []string `yaml:"resources"`
	}
	assemblyContent, err := os.ReadFile(filepath.Join("..", filepath.FromSlash(authority.Assembly.Kustomization)))
	if err != nil {
		t.Fatal(err)
	}
	if err := yaml.Unmarshal(assemblyContent, &assembly); err != nil {
		t.Fatal(err)
	}
	if assembly.Namespace != "task-processor" || !reflect.DeepEqual(assembly.Resources, []string{"listingkit-release-gate-runners.yaml"}) {
		t.Fatalf("release assembly must contain only the current canary: %#v", assembly)
	}

	if authority.ReleaseGateRunners.Manifest != "deployments/kubernetes/listingkit-workbench/release-authority/listingkit-release-gate-runners.yaml" ||
		authority.ReleaseGateRunners.HoldImage != releaseGateHoldImage || len(authority.ReleaseGateRunners.Names) != 1 || authority.ReleaseGateRunners.Names[0] != "image-agent-temporal-v3-canary-runner" {
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

func TestListingKitReleaseAuthorityNegativeFixturesAreCheckedIn(t *testing.T) {
	t.Parallel()
	root := filepath.Join("..", "policy", "listingkit-release-authority", "fixtures", "negative")
	expected := map[string]map[string]any{
		"kube-config-fallback.yaml":  {"workflow": map[string]any{"credentials": []any{"KUBE_CONFIG"}}},
		"job-reusable-workflow.yaml": {"workflow": map[string]any{"uses": "./.github/workflows/unowned-release.yml"}},
		"step-composite-action.yaml": {"workflow": map[string]any{"stepUses": "./unowned-release-action"}},
		"policy-drift.yaml":          {"policy": map[string]any{"repository": "attacker/example"}},
		"mutable-action-ref.yaml":    {"workflow": map[string]any{"actionUses": "actions/checkout@v4"}},
		"pod-rbac-and-query.yaml":    {"releaseGateCommand": "kubectl -n task-processor get pods -l app=image-agent-temporal-v3-canary-runner -o json"},
	}
	paths, err := filepath.Glob(filepath.Join(root, "*.yaml"))
	if err != nil || len(paths) != len(expected) {
		t.Fatalf("current negative fixture inventory drifted: paths=%v err=%v", paths, err)
	}
	for name, want := range expected {
		content, err := os.ReadFile(filepath.Join(root, name))
		if err != nil {
			t.Fatal(err)
		}
		var actual map[string]any
		if err := yaml.Unmarshal(content, &actual); err != nil {
			t.Fatalf("decode %s: %v", name, err)
		}
		if !reflect.DeepEqual(actual, want) {
			t.Errorf("%s must cover its current safety rule without retired identity inputs: %#v", name, actual)
		}
	}
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
}

func TestListingKitReleaseAuthorityRejectsRetiredAuthorityAndMissingInputs(t *testing.T) {
	const authorityRoot = "deployments/kubernetes/listingkit-workbench/release-authority/"
	const policyPath = authorityRoot + "release-policy.yaml"
	const assemblyPath = authorityRoot + "kustomization.yaml"
	api := readRepositoryText(t, "..", ".github", "workflows", "listingkit-deploy.yml")
	check := func(t *testing.T, path, content, reason string) {
		t.Helper()
		output, err := runReleaseAuthorityCrossFilePolicy(t, api, "", "", map[string]string{path: content})
		if err == nil || !strings.Contains(output, reason) {
			t.Fatalf("policy must reject the actual bundle for %q: err=%v\n%s", reason, err, output)
		}
	}
	for _, kind := range []string{"Role", "RoleBinding", "AuthenticationConfiguration"} {
		t.Run(kind, func(t *testing.T) {
			check(t, authorityRoot+"forbidden.yaml", "apiVersion: v1\nkind: "+kind+"\nmetadata: {name: retired}\n", "must not contain retired authority object "+kind)
		})
	}
	t.Run("machine policy document inventory", func(t *testing.T) {
		policy := readRepositoryText(t, "..", filepath.FromSlash(policyPath))
		check(t, policyPath, policy+"\n---\nkind: ClusterRole\nmetadata: {name: unrelated}\n", "machine policy input must contain exactly its policy document")
		check(t, policyPath, "kind: ClusterRole\n"+policy, "machine policy input must contain exactly its policy document")
	})

	t.Run("runner document inventory", func(t *testing.T) {
		runner := readRepositoryText(t, "..", filepath.FromSlash(authorityRoot+"listingkit-release-gate-runners.yaml"))
		for _, kind := range []string{"ClusterRole", "ClusterRoleBinding", "Job", "Deployment"} {
			t.Run("extra/"+kind, func(t *testing.T) {
				mutated := runner + "\n---\napiVersion: v1\nkind: " + kind + "\nmetadata: {name: unrelated}\n"
				output, err := runReleaseAuthorityCrossFilePolicy(t, api, "", mutated)
				if err == nil || !strings.Contains(output, "release-gate manifest must contain exactly the current canary Deployment") {
					t.Fatalf("extra runner-manifest document must fail at its inventory boundary: err=%v\n%s", err, output)
				}
			})
		}
		for _, test := range []struct{ name, content string }{
			{"List", "apiVersion: v1\nkind: List\nitems: []\n"},
			{"alternate deployment", strings.Replace(runner, "name: image-agent-temporal-v3-canary-runner", "name: unrelated", 1)},
		} {
			t.Run(test.name, func(t *testing.T) {
				if test.content == runner {
					t.Fatal("runner fixture mutation did not change its input")
				}
				output, err := runReleaseAuthorityCrossFilePolicy(t, api, "", test.content)
				if err == nil || !strings.Contains(output, "release-gate manifest must contain exactly the current canary Deployment") {
					t.Fatalf("only the current Deployment document is allowed: err=%v\n%s", err, output)
				}
			})
		}
	})

	t.Run("extra assembly resource", func(t *testing.T) {
		assembly := readRepositoryText(t, "..", filepath.FromSlash(assemblyPath))
		check(t, assemblyPath, assembly+"  - unowned.yaml\n", "release assembly must contain only the current canary manifest")
	})
	for _, field := range []string{"oidc", "rbac", "protectedResources"} {
		t.Run("retired field/"+field, func(t *testing.T) {
			policy := readRepositoryText(t, "..", filepath.FromSlash(policyPath))
			check(t, policyPath, policy+"  "+field+": {}\n", "retired machine policy field "+field+" must be absent")
		})
	}
	for _, field := range []string{"tools", "assembly", "uiAuthorization", "releaseGateRunners", "documentation"} {
		t.Run("missing policy field/"+field, func(t *testing.T) {
			var policy map[string]map[string]any
			if err := yaml.Unmarshal([]byte(readRepositoryText(t, "..", filepath.FromSlash(policyPath))), &policy); err != nil {
				t.Fatal(err)
			}
			delete(policy["releaseAuthority"], field)
			content, err := yaml.Marshal(policy)
			if err != nil {
				t.Fatal(err)
			}
			check(t, policyPath, string(content), "machine policy is missing required field "+field)
		})
	}
	for path, reason := range map[string]string{
		policyPath:   "release bundle must contain exactly one machine policy",
		assemblyPath: "release bundle must contain exactly one governed assembly",
		authorityRoot + "listingkit-release-gate-runners.yaml":                            "release-gate runner inventory drifted",
		".github/workflows/listingkit-deploy.yml":                                         "release bundle must contain exactly one api workflow",
		".github/workflows/listingkit-ui-deploy.yml":                                      "release bundle must contain exactly one ui workflow",
		"deployments/kubernetes/listingkit-workbench/base/configmap.yaml":                 "UI authorization ConfigMap inventory drifted",
		"deployments/kubernetes/listingkit-workbench/base/listingkit-ui-auth-config.yaml": "UI authorization ConfigMap inventory drifted",
		"deployments/kubernetes/listingkit-workbench/base/listingkit-ui-deployment.yaml":  "UI deployment authorization ConfigMap ownership drifted",
	} {
		t.Run("missing input/"+filepath.Base(path), func(t *testing.T) { check(t, path, "", reason) })
	}
}

func runReleaseAuthorityCrossFilePolicy(t *testing.T, mutatedWorkflow, mutatedUIWorkflow, mutatedRunnerManifest string, mutatedInputs ...map[string]string) (string, error) {
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

	inputs := []string{
		"deployments/kubernetes/listingkit-workbench/release-authority/release-policy.yaml",
		"deployments/kubernetes/listingkit-workbench/release-authority/kustomization.yaml",
		runnerManifestPath,
		"deployments/kubernetes/listingkit-workbench/base/configmap.yaml",
		"deployments/kubernetes/listingkit-workbench/base/listingkit-ui-auth-config.yaml",
		"deployments/kubernetes/listingkit-workbench/base/listingkit-ui-deployment.yaml",
		"/fixture/.github/workflows/listingkit-deploy.yml",
		uiWorkflowPath,
	}
	for _, overrides := range mutatedInputs {
		for relative, content := range overrides {
			index := -1
			for i, input := range inputs {
				if input == relative || strings.HasSuffix(input, "/"+relative) {
					index = i
					break
				}
			}
			if content == "" {
				if index < 0 {
					t.Fatalf("cannot omit unknown policy input %s", relative)
				}
				inputs = append(inputs[:index], inputs[index+1:]...)
				continue
			}
			fixturePath := filepath.Join(fixtureRoot, filepath.FromSlash(relative))
			if err := os.MkdirAll(filepath.Dir(fixturePath), 0o700); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(fixturePath, []byte(content), 0o600); err != nil {
				t.Fatal(err)
			}
			input := "/fixture/" + relative
			if index >= 0 {
				inputs[index] = input
			} else {
				inputs = append(inputs, input)
			}
		}
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
	}
	arguments = append(arguments, inputs...)

	command := exec.Command("docker", arguments...)
	command.Dir = repoRoot
	output, runErr := command.CombinedOutput()
	return string(output), runErr
}

func preflightBash(t *testing.T) string {
	t.Helper()
	if runtime.GOOS == "windows" {
		path := `C:\Program Files\Git\bin\bash.exe`
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}
	if path, err := exec.LookPath("bash"); err == nil {
		return path
	}
	t.Fatal("bash is required to execute the ListingKit invitation Secret preflight")
	return ""
}

func writeBashPreflightFake(t *testing.T, path, content string) {
	t.Helper()
	writeExecutablePreflightFake(t, path, content)
	output, err := exec.Command(preflightBash(t), "-n", path).CombinedOutput()
	if err != nil {
		t.Fatalf("validate Bash fake %s: %v\n%s", filepath.Base(path), err, output)
	}
}

func writeExecutablePreflightFake(t *testing.T, path, content string) {
	t.Helper()
	tmp, err := os.CreateTemp(filepath.Dir(path), ".preflight-fake-*")
	if err != nil {
		t.Fatalf("create fake %s: %v", filepath.Base(path), err)
	}
	tmpPath := tmp.Name()
	defer os.Remove(tmpPath)
	if _, err := tmp.WriteString(content); err != nil {
		_ = tmp.Close()
		t.Fatalf("write fake %s: %v", filepath.Base(path), err)
	}
	if err := tmp.Chmod(0o700); err != nil {
		_ = tmp.Close()
		t.Fatalf("chmod fake %s: %v", filepath.Base(path), err)
	}
	if err := tmp.Close(); err != nil {
		t.Fatalf("close fake %s: %v", filepath.Base(path), err)
	}
	if err := os.Rename(tmpPath, path); err != nil {
		t.Fatalf("publish fake %s: %v", filepath.Base(path), err)
	}
}
