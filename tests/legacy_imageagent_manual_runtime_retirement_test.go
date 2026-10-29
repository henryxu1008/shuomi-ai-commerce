package tests

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"slices"
	"testing"
)

func TestLegacyImageAgentManualRuntimeClosureIsRetired(t *testing.T) {
	for _, path := range []string{
		"internal/imageagent/slot_effect.go", "internal/imageagent/store/slot_effect.go",
		"internal/imageagent/store/slot_effect_repository_test.go",
		"internal/imageagent/temporal/testdata/v2-slot-inflight.json",
		"internal/imageagent/temporal/testdata/v2-awaiting-approval.json",
		"internal/imageagent/temporal/testdata/v2-pre-atomic-awaiting-approval.json",
		"deployments/kubernetes/listingkit-workbench/base/image-agent-temporal-worker-deployment.yaml",
		"deployments/kubernetes/listingkit-workbench/base/image-agent-temporal-worker-v3-deployment.yaml",
		"scripts/listingkit-apply-image-agent-worker-deployment.sh",
	} {
		if _, err := os.Lstat(filepath.Join("..", filepath.FromSlash(path))); !os.IsNotExist(err) {
			t.Errorf("retired manual image artifact must be absent: %s: %v", path, err)
		}
	}
	for path, banned := range map[string][]string{
		"internal/imageagent/ports.go":                            {"SlotExecutor", "ApprovedAssetPublisher", "PublishApprovedInput"},
		"internal/imageagent/store/gorm.go":                       {"NewGormRepository"},
		"internal/imageagent/store/records.go":                    {"slotExternalEffectRecord"},
		"internal/imageagent/temporal/worker.go":                  {"NewClient"},
		"internal/imageagent/temporal/slot_workflow.go":           {"ImageSlotWorkflow", "activityWireForFrozenSlotWorkflow"},
		"internal/imageagent/temporal/activity_registration.go":   {"PublishApproved", "LegacyExecuteSlot", "LegacyPersistSlotResult", "LegacyPersistRunState", "LegacyPersistPlanRevision", "LegacyPersistPendingCommand", "LegacyPublishApproved"},
		"internal/imageagent/temporal/activities_execution_v3.go": {"ExecuteSlot"},
		"internal/imageagent/temporal/activities_persistence.go":  {"PersistSlotResult"},
		"internal/imageagent/tools/product_image_executor.go":     {"NewFrozenV2ProductImageSlotExecutor", "LegacyAssetMaterializer", "ExecuteSlot", "PublishSlot"},
		"internal/imageagent/assetpublication/publisher.go":       {"NewV2Publisher", "PublishApproved", "approvalCommitV2"},
		"internal/app/worker/imageagent/dependencies.go":          {"legacyV2AssetMaterializer", "resolveImageAgentTemporalDependencies"},
	} {
		file, err := parser.ParseFile(token.NewFileSet(), filepath.Join("..", filepath.FromSlash(path)), nil, 0)
		if err != nil {
			t.Fatal(err)
		}
		ast.Inspect(file, func(n ast.Node) bool {
			name := ""
			switch d := n.(type) {
			case *ast.FuncDecl:
				name = d.Name.Name
			case *ast.TypeSpec:
				name = d.Name.Name
			}
			if slices.Contains(banned, name) {
				t.Errorf("%s retains retired declaration %s", path, name)
			}
			return true
		})
	}
}
