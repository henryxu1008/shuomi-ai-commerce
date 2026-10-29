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

func TestLegacyImageAgentV2ContractClosureIsRetired(t *testing.T) {
	for path, banned := range map[string][]string{
		"internal/imageagent/result_digest.go":           {"ResultDigestV2"},
		"internal/imageagent/temporal/workflow.go":       {"resultDigest"},
		"internal/imageagent/temporal/types.go":          {"publicationKey"},
		"internal/imageagent/temporal/slot_workflow.go":  {"hasCandidateAsset"},
		"internal/app/worker/imageagent/capabilities.go": {"buildProductionImageCapabilities"},
	} {
		file, err := parser.ParseFile(token.NewFileSet(), filepath.Join("..", filepath.FromSlash(path)), nil, 0)
		if err != nil {
			t.Fatal(err)
		}
		for _, declaration := range file.Decls {
			if function, ok := declaration.(*ast.FuncDecl); ok && slices.Contains(banned, function.Name.Name) {
				t.Errorf("%s retains retired function %s", path, function.Name.Name)
			}
		}
	}
	for path, contracts := range map[string]map[string][]string{
		"internal/imageagent/model.go":                   {"AssetCandidate": {"Artwork", "GenerationProof", "AssetID", "SourceAssetID", "Width", "Height", "Operations", "DurableAsset"}},
		"internal/imageagent/ports.go":                   {"SlotExecutionResult": {"SlotID", "Attempt", "Candidates"}},
		"internal/app/worker/imageagent/dependencies.go": {"imageAgentWorkerDependencyResolver": nil},
	} {
		file, err := parser.ParseFile(token.NewFileSet(), filepath.Join("..", filepath.FromSlash(path)), nil, 0)
		if err != nil {
			t.Fatal(err)
		}
		found := map[string]bool{}
		ast.Inspect(file, func(n ast.Node) bool {
			declaration, ok := n.(*ast.TypeSpec)
			if !ok {
				return true
			}
			want, required := contracts[declaration.Name.Name]
			if !required {
				return true
			}
			found[declaration.Name.Name] = true
			shape, ok := declaration.Type.(*ast.StructType)
			if !ok {
				t.Errorf("%s must remain a struct", declaration.Name.Name)
				return false
			}
			var names []string
			for _, field := range shape.Fields.List {
				if len(field.Names) == 0 {
					t.Errorf("%s must not embed another contract", declaration.Name.Name)
				}
				for _, name := range field.Names {
					names = append(names, name.Name)
				}
			}
			if want != nil && !slices.Equal(names, want) {
				t.Errorf("%s fields = %v; want %v", declaration.Name.Name, names, want)
			}
			if slices.Contains(names, "BuildCapabilities") {
				t.Error("idle default BuildCapabilities must be retired")
			}
			return false
		})
		for name := range contracts {
			if !found[name] {
				t.Errorf("current contract %s must exist in %s", name, path)
			}
		}
	}
}
