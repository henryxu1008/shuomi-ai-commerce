package tests

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

func TestLegacyImageAgentHistoryProtocolClosureIsRetired(t *testing.T) {
	for path, banned := range map[string][]string{
		"internal/imageagent/temporal/workflow.go":               {"workflowActivityWire", "activityWireForWorkflow", "resultDigestForWire", "approvalPublicationKeyForWire", "legacyApprovalActionPublicationKey", "errRetiredImageAgentWire", "enforceIngressPlanPolicy", "externalEffectFinalization"},
		"internal/imageagent/temporal/types.go":                  {"PersistWorkflowFailureActivityInput", "activityPersistWorkflowFailure", "enforceIngressPlanPolicy", "externalEffectFinalization"},
		"internal/imageagent/temporal/activity_registration.go":  {"PersistWorkflowFailure", "activityPersistWorkflowFailure"},
		"internal/imageagent/temporal/activities_persistence.go": {"PersistWorkflowFailure"},
		"internal/imageagent/temporal/slot_workflow.go":          {"effectPhaseForFinalizationWire"},
		"internal/imageagent/artifact.go":                        {"ValidateStagingManifest", "ValidateFinalManifest"},
		"internal/imageagent/budget.go":                          {"HasProviderLimits", "EnabledLimitNames"},
	} {
		file, err := parser.ParseFile(token.NewFileSet(), filepath.Join("..", filepath.FromSlash(path)), nil, 0)
		if err != nil {
			t.Fatal(err)
		}
		ast.Inspect(file, func(n ast.Node) bool {
			if id, ok := n.(*ast.Ident); ok && slices.Contains(banned, id.Name) {
				t.Errorf("%s retains retired identifier %s", path, id.Name)
			}
			return true
		})
	}
	for path, expected := range map[string]map[string][]string{
		"internal/imageagent/temporal/workflow.go": {
			"ImageAgentWorkflow":            {"commandIngressPlanPolicyPatch", "budgetAuthorizationPatch", "externalEffectFinalizationPatch", "projectionExecutionCommitPatch", "workflowFailureProjectionPatch"},
			"assertCurrentWorkflowProtocol": {"slotExecutionWireV3Patch", "approvalActionIDV3Patch", "approvalPublicationWireV3Patch", "resultDigestV3Patch", "approvalPublicationScopePatch", "approvalPublicationKeyLengthPatch", "activityWireV2Patch", "effectRecoveryStartWireV1Patch"},
			"newWorkflowEffectOwner":        {"externalEffectFinalizationPatch"},
		},
		"internal/imageagent/temporal/effect_recovery_workflow.go": {"reconcileEffectRecovery": {"effectRecoveryParentSignalPatch"}},
	} {
		file, err := parser.ParseFile(token.NewFileSet(), filepath.Join("..", filepath.FromSlash(path)), nil, 0)
		if err != nil {
			t.Fatal(err)
		}
		seen := map[string][]string{}
		for _, d := range file.Decls {
			f, ok := d.(*ast.FuncDecl)
			if !ok {
				continue
			}
			ast.Inspect(f, func(n ast.Node) bool {
				call, ok := n.(*ast.CallExpr)
				if !ok {
					return true
				}
				sel, ok := call.Fun.(*ast.SelectorExpr)
				if !ok || sel.Sel.Name != "GetVersion" {
					return true
				}
				if len(call.Args) != 4 {
					t.Errorf("%s malformed GetVersion", f.Name.Name)
					return true
				}
				id, ok := call.Args[1].(*ast.Ident)
				if !ok {
					t.Errorf("version ID must retain its original constant")
					return true
				}
				seen[f.Name.Name] = append(seen[f.Name.Name], id.Name)
				for _, arg := range call.Args[2:] {
					if v, ok := arg.(*ast.BasicLit); !ok || v.Kind != token.INT || v.Value != "1" {
						t.Errorf("%s must support only current version 1", id.Name)
					}
				}
				return true
			})
			positions := map[string]token.Pos{}
			ast.Inspect(f, func(n ast.Node) bool {
				call, ok := n.(*ast.CallExpr)
				if !ok {
					return true
				}
				name := ""
				switch target := call.Fun.(type) {
				case *ast.Ident:
					name = target.Name
				case *ast.SelectorExpr:
					name = target.Sel.Name
				}
				if name == "GetVersion" && len(call.Args) == 4 {
					if id, ok := call.Args[1].(*ast.Ident); ok {
						name = id.Name
					}
				}
				if _, exists := positions[name]; !exists {
					positions[name] = call.Pos()
				}
				return true
			})
			var order []string
			switch f.Name.Name {
			case "ImageAgentWorkflow":
				order = []string{"runImageAgentWorkflow", "workflowFailureProjectionPatch", "persistWorkflowFailure"}
			case "newWorkflowEffectOwner":
				order = []string{"NewChannel", "assertCurrentWorkflowProtocol", "externalEffectFinalizationPatch", "Go"}
			case "reconcileEffectRecovery":
				order = []string{"ExecuteActivity", "effectRecoveryParentSignalPatch", "signalEffectRecoveryCompletion"}
			}
			for i, name := range order {
				if positions[name] == token.NoPos || i > 0 && positions[order[i-1]] >= positions[name] {
					t.Errorf("%s moved protocol boundary: expected %v", f.Name.Name, order)
				}
			}
		}
		if len(seen) != len(expected) {
			t.Errorf("%s version read owners drifted: %v", path, seen)
		}
		for function, want := range expected {
			if !slices.Equal(seen[function], want) {
				t.Errorf("%s.%s version reads/order = %v, want %v", path, function, seen[function], want)
			}
		}
	}
}

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
		"internal/imageagent/ports.go":                                       {"SlotExecutor", "ApprovedAssetPublisher", "PublishApprovedInput"},
		"internal/imageagent/store/gorm.go":                                  {"NewGormRepository"},
		"internal/imageagent/store/records.go":                               {"slotExternalEffectRecord"},
		"internal/imageagent/temporal/worker.go":                             {"NewClient"},
		"internal/imageagent/temporal/slot_workflow.go":                      {"ImageSlotWorkflow", "activityWireForFrozenSlotWorkflow"},
		"internal/imageagent/temporal/activity_registration.go":              {"PublishApproved", "LegacyExecuteSlot", "LegacyPersistSlotResult", "LegacyPersistRunState", "LegacyPersistPlanRevision", "LegacyPersistPendingCommand", "LegacyPublishApproved"},
		"internal/imageagent/temporal/activities_execution_v3.go":            {"ExecuteSlot"},
		"internal/imageagent/temporal/activities_persistence.go":             {"PersistSlotResult"},
		"internal/imageagent/tools/source_generation_executor.go":            {"NewFrozenV2ProductImageSlotExecutor", "LegacyAssetMaterializer", "ExecuteSlot", "PublishSlot"},
		"internal/imageagent/assetpublication/acquisition_main_publisher.go": {"NewV2Publisher", "PublishApproved", "approvalCommitV2"},
		"internal/app/worker/imageagent/dependencies.go":                     {"legacyV2AssetMaterializer", "resolveImageAgentTemporalDependencies"},
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
		"internal/imageagent/result_digest.go":                   {"ResultDigestV2"},
		"internal/imageagent/temporal/workflow.go":               {"resultDigest"},
		"internal/imageagent/temporal/types.go":                  {"publicationKey"},
		"internal/imageagent/temporal/slot_workflow.go":          {"hasCandidateAsset"},
		"internal/app/worker/imageagent/source_main_executor.go": {"buildProductionImageCapabilities"},
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

func TestLegacyImageMethodSnapshotClosureIsRetired(t *testing.T) {
	for _, path := range []string{"internal/imageagent/image_detail_copy.go", "internal/imageagent/image_visual_prompt.go", "internal/imageagent/image_detail_copy_test.go", "hack/debug/image-reconcile-774/main.go"} {
		if _, err := os.Stat(filepath.Join("..", filepath.FromSlash(path))); !os.IsNotExist(err) {
			t.Errorf("retired path %s must be absent: %v", path, err)
		}
	}
	for path, banned := range map[string][]string{
		"internal/imageagent/image_method.go":                     {"HistoricalImageMethodPolicyVersion", "FrozenVisualBriefPolicyVersion", "FrozenLiteralCopyPolicyVersion", "Executable"},
		"internal/imageagent/image_set_service.go":                {"visualProductFacts", "visualImagePrompt", "usesLiteralDetailCopy", "detailVisibleText", "literalDetailPrompt"},
		"internal/imageagent/asset_catalog.go":                    {"ValidateSubmittedPlanAgainstCatalog", "validateLiteralDetailRecipe"},
		"internal/imageagent/validation.go":                       {"ValidateSubmittedPlan"},
		"internal/imageagent/generation_input.go":                 {"usesLiteralDetailCopy", "validateLiteralDetailRecipe"},
		"internal/integration/skillimage/context.go":              {"HistoricalImageMethodPolicyVersion"},
		"internal/imageagent/store/generation_output_recovery.go": {"FrozenLiteralCopyPolicyVersion"},
	} {
		file, err := parser.ParseFile(token.NewFileSet(), filepath.Join("..", filepath.FromSlash(path)), nil, 0)
		if err != nil {
			t.Fatal(err)
		}
		ast.Inspect(file, func(n ast.Node) bool {
			if id, ok := n.(*ast.Ident); ok && slices.Contains(banned, id.Name) {
				t.Errorf("%s retains %s", path, id.Name)
			}
			return true
		})
	}
	for path, required := range map[string][]string{
		"internal/imageagent/image_method.go":        {"ImageMethodPolicyVersion", "ValidateImageMethodDispatch", "AcquireImageMethod", "ComposeImageMethodPrompt"},
		"internal/imageagent/image_text.go":          {"validateReviewedImageText", "imageTextPrompt", "validReviewedImageText", "CloneImageTextBinding"},
		"internal/imageagent/validation.go":          {"ValidatePlan", "MaxPlanSlots", "MaxJSONSafePlanRevision", "ValidateInitialSubmittedPlan", "ValidateReplacementSubmittedPlan"},
		"internal/imageagent/asset_catalog.go":       {"ValidatePlanAgainstCatalog", "CatalogHash", "CatalogSnapshotHash", "NormalizeProductContextRef"},
		"internal/imageagent/projection.go":          {"ValidatePlanAgainstCatalog", "ValidateVisualAdmission", "ErrRevisionConflict"},
		"internal/imageagent/image_set_admission.go": {"ValidateImageSetAdmission", "SlotStatusPending", "ErrRevisionConflict"},
	} {
		file, err := parser.ParseFile(token.NewFileSet(), filepath.Join("..", filepath.FromSlash(path)), nil, 0)
		if err != nil {
			t.Fatal(err)
		}
		seen := map[string]bool{}
		ast.Inspect(file, func(n ast.Node) bool {
			if id, ok := n.(*ast.Ident); ok {
				seen[id.Name] = true
			}
			return true
		})
		for _, name := range required {
			if !seen[name] {
				t.Errorf("%s must retain %s", path, name)
			}
		}
	}
}

// The current source-edit/approval path must survive without restoring the
// cancelled generic executor, SDK transport, or staged model-review contracts.
func TestLegacyImageAgentGenericExecutionClosureIsRetired(t *testing.T) {
	for _, path := range []string{
		"internal/app/worker/imageagent/capabilities.go",
		"internal/app/worker/imageagent/capabilities_test.go",
		"internal/app/worker/imageagent/organization_main_admission.go",
		"internal/app/worker/imageagent/organization_main_admission_test.go",
		"internal/core/config/type_openai.go",
		"internal/core/config/validator_openai.go",
		"internal/imageagent/assetpublication/publisher.go",
		"internal/imageagent/assetpublication/publisher_test.go",
		"internal/imageagent/effectpolicy/review.go",
		"internal/imageagent/effectpolicy/review_test.go",
		"internal/imageagent/review_outcome.go",
		"internal/imageagent/store/slot_effect_v3_review.go",
		"internal/imageagent/store/slot_effect_v3_review_memory.go",
		"internal/imageagent/temporal/activities_review_v3.go",
		"internal/imageagent/tools/image_set_executor.go",
		"internal/imageagent/tools/image_set_executor_test.go",
		"internal/imageagent/tools/product_image_executor.go",
		"internal/imageagent/temporal/organization_review_quote_test.go",
		"internal/imageagent/tools/product_image_executor_contract_test.go",
		"internal/integration/grsai/product_image_adapter.go",
		"internal/integration/grsai/product_image_adapter_test.go",
		"internal/integration/openai/client.go",
		"internal/integration/openai/client_manager.go",
		"internal/integration/openai/contextual_client.go",
		"internal/integration/openai/effective_configuration_test.go",
		"internal/integration/openai/identity.go",
		"internal/integration/openai/images.go",
		"internal/integration/openai/images_test.go",
		"internal/integration/openai/logger.go",
		"internal/integration/openai/logger_test.go",
		"internal/integration/openai/pool.go",
		"internal/integration/openai/pool_request_test.go",
		"internal/integration/openai/pool_retry_test.go",
		"internal/integration/openai/product_image_adapter.go",
		"internal/integration/openai/product_image_adapter_test.go",
		"internal/integration/openai/tenant_manager_test.go",
		"internal/integration/openai/types.go",
		"internal/integration/openai/types_contract_test.go",
		"internal/product/image/scene_options.go",
		"internal/product/image/scene_options_test.go",
	} {
		if _, err := os.Stat(filepath.Join("..", filepath.FromSlash(path))); !os.IsNotExist(err) {
			t.Errorf("retired source %s must be absent (stat: %v)", path, err)
		}
	}
	for path, required := range map[string][]string{
		"internal/imageagent/tools/source_generation_executor.go":            {"SourceGenerationExecutor", "QuoteSlot", "GenerateQuotedSlot"},
		"internal/imageagent/tools/result_builder.go":                        {"ImageResultBuilder", "NewAcquisitionMainResultBuilder", "NewImageSetResultBuilder"},
		"internal/imageagent/acquisition_main.go":                            {"ValidateAcquisitionMainPlan"},
		"internal/app/worker/imageagent/source_main_executor.go":             {"SourceMainExecutor"},
		"internal/integration/openai/source_image_adapter.go":                {"SourceImageAdapter"},
		"internal/integration/openai/credential_config.go":                   {"ClientConfig", "ResolvedClientConfig"},
		"internal/product/image/source_white_background.go":                  {"SourceWhiteBackgroundRequest"},
		"internal/product/image/policy_scene_defaults.go":                    {"PolicySceneDefaults"},
		"internal/imageagent/assetpublication/acquisition_main_publisher.go": {"AcquisitionMainPublisher", "PublishAcquisitionMain"},
	} {
		file, err := parser.ParseFile(token.NewFileSet(), filepath.Join("..", filepath.FromSlash(path)), nil, 0)
		if err != nil {
			t.Error(err)
			continue
		}
		found := map[string]bool{}
		for _, decl := range file.Decls {
			switch d := decl.(type) {
			case *ast.FuncDecl:
				found[d.Name.Name] = true
			case *ast.GenDecl:
				for _, spec := range d.Specs {
					if typ, ok := spec.(*ast.TypeSpec); ok {
						found[typ.Name.Name] = true
					}
				}
			}
		}
		for _, name := range required {
			if !found[name] {
				t.Errorf("%s requires current declaration %s", path, name)
			}
		}
	}
}

func TestRetiredGenericImageContractsCannotReturn(t *testing.T) {
	banned := map[string][]string{
		"internal/imageagent/ports.go":                     {"StagedSlotReviewer", "BudgetedStagedSlotReviewer", "ApprovedAssetPublisherV3", "PublishApprovedV3Input"},
		"internal/imageagent/slot_execution.go":            {"ReviewUsageReservation", "ReviewUsageAttempt", "ReviewUsageOutcome", "ReviewRetryInput"},
		"internal/product/image/ports.go":                  {"SubjectExtractor", "WhiteBackgroundRenderer", "SceneRenderer", "Reviewer", "CompleteCapabilities"},
		"internal/product/image/model.go":                  {"ExtractRequest", "RenderRequest", "SceneRequest", "ReviewRequest", "Review"},
		"internal/integration/openai/credential_config.go": {"Manager", "Client", "RequestPool", "Logger", "PoolConfig", "ManagerConfig"},
	}
	for path, names := range banned {
		file, err := parser.ParseFile(token.NewFileSet(), filepath.Join("..", filepath.FromSlash(path)), nil, 0)
		if err != nil {
			t.Fatal(err)
		}
		ast.Inspect(file, func(node ast.Node) bool {
			name := ""
			switch n := node.(type) {
			case *ast.TypeSpec:
				name = n.Name.Name
				if path == "internal/integration/openai/credential_config.go" && n.Assign.IsValid() {
					t.Errorf("minimal current credential contracts must not be aliases: %s", n.Name.Name)
				}
			case *ast.FuncDecl:
				name = n.Name.Name
			}
			if slices.Contains(names, name) {
				t.Errorf("retired generic image declaration in current owner: %s:%s", path, name)
			}
			return true
		})
	}
	for _, path := range trackedFiles(t, "internal") {
		if !strings.HasSuffix(path, ".go") {
			continue
		}
		file, err := parser.ParseFile(token.NewFileSet(), filepath.Join("..", filepath.FromSlash(path)), nil, 0)
		if err != nil {
			t.Fatal(err)
		}
		for _, spec := range file.Imports {
			imported, err := decodeGoImportPath(spec.Path.Value)
			if err != nil {
				t.Fatal(err)
			}
			if importMatchesPrefix(imported, "github.com/sashabaranov/go-openai") {
				t.Errorf("retired SDK import in %s", path)
			}
		}
	}
}
