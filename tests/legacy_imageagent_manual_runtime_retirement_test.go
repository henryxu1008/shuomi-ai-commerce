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

func TestLegacyImageAgentBusinessSignalClosureIsRetired(t *testing.T) {
	retired := []string{
		"startSignalHandlers", "dispatchReplaceSignal", "dispatchRetrySignal", "dispatchApprovalSignal", "dispatchCancelSignal",
		"signalCanEnter", "signalCanEnterWithFingerprint", "dispatchDeferredRetries", "deferredRetries", "legacyRetryPending", "signalIngressDeferred",
	}
	businessNames := []string{"signalApproveResults", "signalRetrySlot", "signalReplacePlan", "signalCancel"}
	requiredFunctions := map[string]bool{}
	requiredUpdates := map[string]bool{}
	recoveryChannel, recoveryProducer, sdkPort := false, false, false
	for _, path := range []string{
		"internal/imageagent/temporal/workflow.go", "internal/imageagent/temporal/worker.go",
		"internal/imageagent/temporal/effect_recovery_workflow.go",
	} {
		file, err := parser.ParseFile(token.NewFileSet(), filepath.Join("..", filepath.FromSlash(path)), nil, 0)
		if err != nil {
			t.Fatal(err)
		}
		ast.Inspect(file, func(n ast.Node) bool {
			switch node := n.(type) {
			case *ast.Ident:
				if slices.Contains(retired, node.Name) {
					t.Errorf("%s retains retired business Signal declaration/reference %s", path, node.Name)
				}
			case *ast.FuncDecl:
				requiredFunctions[node.Name.Name] = true
			case *ast.TypeSpec:
				if path != "internal/imageagent/temporal/worker.go" || node.Name.Name != "sdkWorkflowClient" {
					break
				}
				sdkPort = true
				shape, ok := node.Type.(*ast.InterfaceType)
				if !ok {
					t.Fatal("current Temporal SDK port must remain an interface")
				}
				var methods []string
				for _, method := range shape.Methods.List {
					if len(method.Names) != 1 {
						t.Fatal("current Temporal SDK port must not embed a broader client")
					}
					methods = append(methods, method.Names[0].Name)
				}
				if !slices.Equal(methods, []string{"ExecuteWorkflow", "QueryWorkflow", "UpdateWorkflow"}) {
					t.Errorf("current SDK port must retain Start/Query/Update only: %v", methods)
				}
			case *ast.CallExpr:
				selector, ok := node.Fun.(*ast.SelectorExpr)
				if !ok {
					break
				}
				if selector.Sel.Name == "GetSignalChannel" && len(node.Args) == 2 {
					if name, ok := node.Args[1].(*ast.Ident); ok {
						if slices.Contains(businessNames, name.Name) {
							t.Errorf("retired business Signal channel %s remains", name.Name)
						}
						if name.Name == "signalEffectRecoveryCompleted" {
							recoveryChannel = true
						}
					}
				}
				if selector.Sel.Name == "SetUpdateHandlerWithOptions" && len(node.Args) == 4 {
					if name, ok := node.Args[1].(*ast.Ident); ok {
						requiredUpdates[name.Name] = true
					}
				}
				if selector.Sel.Name == "SignalExternalWorkflow" && len(node.Args) == 5 {
					if name, ok := node.Args[3].(*ast.Ident); ok && name.Name == "signalEffectRecoveryCompleted" {
						recoveryProducer = true
					}
				}
			}
			return true
		})
	}
	for _, name := range append(businessNames, "updateResumeCommand") {
		if !requiredUpdates[name] {
			t.Errorf("current Update registration %s must remain", name)
		}
	}
	for _, name := range []string{"handleReplacePlan", "handleRetrySlot", "handleApproveResults", "handleCancel", "handleResume", "GetProjection", "dispatchEffectRecoveryCompleted", "applyEffectRecoveryCompleted", "signalEffectRecoveryCompletion"} {
		if !requiredFunctions[name] {
			t.Errorf("current command/recovery function %s must remain", name)
		}
	}
	if !sdkPort || !recoveryChannel || !recoveryProducer {
		t.Errorf("current SDK port and recovery Signal producer/receiver must remain: port=%t receiver=%t producer=%t", sdkPort, recoveryChannel, recoveryProducer)
	}
}
