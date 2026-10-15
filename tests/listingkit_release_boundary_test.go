package tests

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"gopkg.in/yaml.v3"
)

type renderedObjectIdentity struct {
	Kind     string `yaml:"kind"`
	Metadata struct {
		Name string `yaml:"name"`
	} `yaml:"metadata"`
}

func TestListingKitImageAgentSteadyStateRendersNoFiniteCanary(t *testing.T) {
	root := filepath.Join("..", "deployments", "kubernetes", "listingkit-workbench")
	for _, test := range []struct {
		name        string
		path        string
		wantWorkers []string
	}{
		{name: "base", path: filepath.Join(root, "base"), wantWorkers: []string{"image-agent-temporal-worker", "image-agent-temporal-worker-v3"}},
		{name: "prod", path: filepath.Join(root, "overlays", "prod"), wantWorkers: []string{"image-agent-temporal-worker", "image-agent-temporal-worker-v3"}},
	} {
		t.Run(test.name, func(t *testing.T) {
			objects := renderKustomizeObjects(t, test.path)
			workers := map[string]bool{}
			canaries := 0
			for _, object := range objects {
				if object.Kind == "Deployment" {
					workers[object.Metadata.Name] = true
				}
				if object.Kind == "Job" && object.Metadata.Name == "image-agent-temporal-v3-canary" {
					canaries++
				}
			}
			for _, worker := range test.wantWorkers {
				if !workers[worker] {
					t.Errorf("%s render is missing long-lived worker %s", test.name, worker)
				}
			}
			if canaries != 0 {
				t.Fatalf("%s steady-state render has %d finite image-agent canaries, want zero", test.name, canaries)
			}
		})
	}
}

func renderKustomizeObjects(t *testing.T, path string) []renderedObjectIdentity {
	t.Helper()
	command := exec.Command("kubectl", "kustomize", path)
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("kubectl kustomize %s: %v\n%s", path, err, output)
	}
	decoder := yaml.NewDecoder(bytes.NewReader(output))
	var objects []renderedObjectIdentity
	for {
		var object renderedObjectIdentity
		err := decoder.Decode(&object)
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("decode kubectl kustomize %s: %v", path, err)
		}
		if object.Kind != "" {
			objects = append(objects, object)
		}
	}
	return objects
}

func TestListingKitReleaseAttestationRequiresExactWorkflowHeadSource(t *testing.T) {
	attestedSource := strings.Repeat("a", 40)
	runHead := strings.Repeat("b", 40)
	output, err := runListingKitReleaseAttestationVerifier(t, releaseAttestationScenario{
		selectedAttempt: "2",
		runAttempt:      "2",
		attestedAttempt: "2",
		attestedSource:  attestedSource,
		runHead:         runHead,
		includeAttempt:  true,
	})
	if err == nil {
		t.Fatalf("attestation source %q unexpectedly passed with workflow head %q: %s", attestedSource, runHead, output)
	}
	if !strings.Contains(string(output), "workflow head SHA does not match attested source") {
		t.Fatalf("source mismatch must fail for the exact workflow-head contract, got: %s", output)
	}
}

func TestListingKitReleaseAttestationRequiresExactRunAttempt(t *testing.T) {
	for _, test := range []struct {
		name     string
		scenario releaseAttestationScenario
		wantPass bool
	}{
		{name: "attempt_1", scenario: releaseAttestationScenario{selectedAttempt: "1", runAttempt: "1", attestedAttempt: "1", includeAttempt: true}, wantPass: true},
		{name: "attempt_2", scenario: releaseAttestationScenario{selectedAttempt: "2", runAttempt: "2", attestedAttempt: "2", includeAttempt: true}, wantPass: true},
		{name: "REST_attempt_does_not_match_selection", scenario: releaseAttestationScenario{selectedAttempt: "2", runAttempt: "1", attestedAttempt: "2", includeAttempt: true}},
		{name: "same_run_artifact_from_another_attempt", scenario: releaseAttestationScenario{selectedAttempt: "2", runAttempt: "2", attestedAttempt: "1", includeAttempt: true}},
		{name: "missing_attempt_selection", scenario: releaseAttestationScenario{runAttempt: "1", attestedAttempt: "1", includeAttempt: false}},
	} {
		t.Run(test.name, func(t *testing.T) {
			output, err := runListingKitReleaseAttestationVerifier(t, test.scenario)
			if test.wantPass && err != nil {
				t.Fatalf("exact attempt verification failed: %v\n%s", err, output)
			}
			if !test.wantPass && err == nil {
				t.Fatalf("attempt mismatch unexpectedly passed: %s", output)
			}
		})
	}
}

type releaseAttestationScenario struct {
	selectedAttempt string
	runAttempt      string
	attestedAttempt string
	attestedSource  string
	runHead         string
	includeAttempt  bool
}

func runListingKitReleaseAttestationVerifier(t *testing.T, scenario releaseAttestationScenario) ([]byte, error) {
	t.Helper()
	if scenario.attestedSource == "" {
		scenario.attestedSource = strings.Repeat("a", 40)
	}
	if scenario.runHead == "" {
		scenario.runHead = scenario.attestedSource
	}
	binDir := t.TempDir()
	apiDigest := strings.Repeat("c", 64)
	now := time.Now().UTC()
	runJSON := fmt.Sprintf(`{"id":424242,"run_attempt":%s,"repository":{"full_name":"octo/task-processor"},"name":"ListingKit API Deploy","path":".github/workflows/listingkit-deploy.yml@refs/heads/main","conclusion":"success","head_sha":%q}`, scenario.runAttempt, scenario.runHead)
	attestationJSON := fmt.Sprintf(`{"gate_version":"listingkit-api-release-gate/v2","repository":"octo/task-processor","workflow_name":"ListingKit API Deploy","workflow_ref":"octo/task-processor/.github/workflows/listingkit-deploy.yml@refs/heads/main","source_sha":%q,"api_candidate_image":"docker.io/xuwei190/task-processor-product-listing-api@sha256:%s","api_workflow_run_id":424242,"api_workflow_run_attempt":%s,"issued_at":%q,"routing_contract":"image-agent-v3-new-starts-v1","worker_wire_contract":"image-agent-workers-v2-v3","worker_replay_contract":"image-agent-replay-v2-v3","schema_contract":"listingkit-schema-additive-v1"}`,
		scenario.attestedSource, apiDigest, scenario.attestedAttempt, now.Add(-time.Minute).Format(time.RFC3339))
	runPath := filepath.Join(t.TempDir(), "run.json")
	attestationPath := filepath.Join(t.TempDir(), "attestation.json")
	if err := os.WriteFile(runPath, []byte(runJSON), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(attestationPath, []byte(attestationJSON), 0o600); err != nil {
		t.Fatal(err)
	}
	writeBashPreflightFake(t, filepath.Join(binDir, "gh"), fmt.Sprintf(`#!/usr/bin/env bash
set -euo pipefail
if [[ "$*" == *"repos/octo/task-processor/commits/%s"* ]]; then
  printf '%s\n'
  exit 0
fi
exit 1
`, scenario.attestedSource, scenario.attestedSource))
	writeBashPreflightFake(t, filepath.Join(binDir, "jq"), fmt.Sprintf(`#!/usr/bin/env bash
set -euo pipefail
filter="${2:-}"
case "$filter" in
  *'keys == '*) exit 0 ;;
  *'.repository.full_name'*) printf 'octo/task-processor\n' ;;
  *'.id | select'*) printf '424242\n' ;;
  *'.run_attempt | select'*) printf '%s\n' ;;
  *'.name | select'*) printf 'ListingKit API Deploy\n' ;;
  *'.path | select'*) printf '.github/workflows/listingkit-deploy.yml@refs/heads/main\n' ;;
  *'.conclusion | select'*) printf 'success\n' ;;
  *'.head_sha | select'*) printf '%s\n' ;;
  *'.gate_version | select'*) printf 'listingkit-api-release-gate/v2\n' ;;
  *'.repository | select'*) printf 'octo/task-processor\n' ;;
  *'.workflow_name | select'*) printf 'ListingKit API Deploy\n' ;;
  *'.workflow_ref | select'*) printf 'octo/task-processor/.github/workflows/listingkit-deploy.yml@refs/heads/main\n' ;;
  *'.source_sha | select'*) printf '%s\n' ;;
  *'.api_candidate_image | select'*) printf 'docker.io/xuwei190/task-processor-product-listing-api@sha256:%s\n' ;;
  *'.api_workflow_run_id | select'*) printf '424242\n' ;;
  *'.api_workflow_run_attempt | select'*) printf '%s\n' ;;
  *'.issued_at | select'*) printf '%s\n' ;;
  *'.routing_contract | select'*) printf 'image-agent-v3-new-starts-v1\n' ;;
  *'.worker_wire_contract | select'*) printf 'image-agent-workers-v2-v3\n' ;;
  *'.worker_replay_contract | select'*) printf 'image-agent-replay-v2-v3\n' ;;
  *'.schema_contract | select'*) printf 'listingkit-schema-additive-v1\n' ;;
  *) exit 1 ;;
esac
`, scenario.runAttempt, scenario.runHead, scenario.attestedSource, apiDigest, scenario.attestedAttempt, now.Add(-time.Minute).Format(time.RFC3339)))

	verifierPath, err := filepath.Abs(filepath.Join("..", "scripts", "verify-listingkit-api-release-attestation.sh"))
	if err != nil {
		t.Fatal(err)
	}
	args := []string{filepath.ToSlash(verifierPath),
		"--attestation", attestationPath,
		"--run-json", runPath,
		"--run-id", "424242"}
	if scenario.includeAttempt {
		args = append(args, "--run-attempt", scenario.selectedAttempt)
	}
	args = append(args,
		"--repository", "octo/task-processor",
		"--api-repository", "docker.io/xuwei190/task-processor-product-listing-api")
	command := exec.Command(preflightBash(t), args...)
	command.Env = append(os.Environ(), "PATH="+binDir+string(os.PathListSeparator)+os.Getenv("PATH"))
	return command.CombinedOutput()
}

func TestListingKitImageAgentDrainRunbookDefinesCompleteSafeInventoryAndRecoveryHorizon(t *testing.T) {
	runbook := readRepositoryText(t, "..", "deployments", "kubernetes", "listingkit-workbench", "README.md")
	for _, required := range []string{
		"ImageAgentWorkflow",
		"ImageSlotWorkflow",
		"listingkit-image-agent-v2-drain-check.sh",
		"Temporal CLI 1.8.1",
		"--expected-run-id",
		"--expected-run-attempt",
		"--expected-api-image",
		"--namespace",
		"PGHOST",
		"PGPORT",
		"PGDATABASE",
		"PGUSER",
		"exactly three complete samples",
		"300 seconds",
		"10-minute first-to-final window",
		"ListingKit operational policy, not a Temporal SLA",
		"listingkit.sh/api-release-run-id",
		"listingkit.sh/api-release-run-attempt",
		"listingkit.sh/api-release-image",
		"listingkit.sh/image-agent-routing-contract",
		"image-agent-v3-new-starts-v1",
		"status NOT IN ('completed', 'failed', 'cancelled')",
		"Local test evidence is not live acceptance",
		"pendingChildren",
		"pendingActivities",
		"imageagent.execute_slot",
		"imageagent.execute_slot.v2",
		"open_v2_parent_count",
		"open_v2_child_count",
		"pending_v2_child_count",
		"pending_v2_activity_count",
		"pending_v2_activity_attempt_sum",
		"tenant_id, owner_user_id, id, created_at",
		"created_at >=",
		"created_at <",
		"image-agent:${tenant_id}:${owner_user_id}:${run_id}",
		"30 days",
		"7 days",
		"37 days",
		"45 days",
		"WorkflowExecutionTimeout",
	} {
		if !strings.Contains(runbook, required) {
			t.Errorf("image-agent recovery runbook is missing %q", required)
		}
	}
	sectionStart := strings.Index(runbook, "## Image-agent Temporal v2/v3 recovery rollout")
	if sectionStart < 0 {
		t.Fatal("image-agent recovery rollout section is missing")
	}
	section := runbook[sectionStart:]
	blocks := strings.Split(section, "```bash\n")
	if len(blocks) < 3 {
		t.Fatalf("image-agent recovery runbook must contain executable inventory and lifecycle bash blocks, got %d", len(blocks)-1)
	}
	for index, remainder := range blocks[1:] {
		end := strings.Index(remainder, "\n```")
		if end < 0 {
			t.Fatalf("bash block %d is not closed", index+1)
		}
		command := exec.Command(preflightBash(t), "-n")
		command.Stdin = strings.NewReader(remainder[:end])
		if output, syntaxErr := command.CombinedOutput(); syntaxErr != nil {
			t.Fatalf("bash block %d is not syntactically executable: %v\n%s", index+1, syntaxErr, output)
		}
	}
}

func TestListingKitRunbookRejectsV2ProducingRollback(t *testing.T) {
	t.Parallel()

	content, err := os.ReadFile(filepath.Join("..", "deployments", "kubernetes", "listingkit-workbench", "README.md"))
	if err != nil {
		t.Fatal(err)
	}
	runbook := string(content)
	for _, required := range []string{
		"Only a prior successful immutable API release attestation carrying",
		"image-agent-v3-new-starts-v1",
		"A v2-producing API rollback is unsupported",
		"invalidates the v2 drain evidence",
		"separately designed recovery procedure",
	} {
		if !strings.Contains(runbook, required) {
			t.Errorf("rollback contract is missing %q", required)
		}
	}
	if strings.Contains(runbook, "A rollback stops new v3 starts") {
		t.Error("supported rollback prose must not reopen v2 starts")
	}
}
