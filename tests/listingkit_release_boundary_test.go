package tests

import (
	"bytes"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

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
		name             string
		path             string
		forbiddenWorkers []string
	}{
		{name: "base", path: filepath.Join(root, "base"), forbiddenWorkers: []string{"image-agent-temporal-worker", "image-agent-temporal-worker-v3"}},
		{name: "prod", path: filepath.Join(root, "overlays", "prod"), forbiddenWorkers: []string{"image-agent-temporal-worker", "image-agent-temporal-worker-v3"}},
	} {
		t.Run(test.name, func(t *testing.T) {
			objects := renderKustomizeObjects(t, test.path)
			if len(objects) == 0 {
				t.Fatal("current steady-state render must not be empty")
			}
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
			for _, worker := range test.forbiddenWorkers {
				if workers[worker] {
					t.Errorf("%s render contains retired worker %s", test.name, worker)
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

func TestListingKitHistoricalDrainEvidenceRetainsDurableSafetyBoundaries(t *testing.T) {
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
	for _, required := range []string{"Historical source retirement", "not current execution, rollback, credential provisioning or worker-stop authority", "Current ImageAgent Go worker/store/Temporal recovery remains unchanged"} {
		if !strings.Contains(runbook, required) {
			t.Errorf("historical drain evidence must preserve current recovery and deny execution authority: %q", required)
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
