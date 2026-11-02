package tests

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"testing"
)

func TestLegacyImageAsyncReviewClosureIsRetired(t *testing.T) {
	assertRestoredDebugPathsAbsent(t,
		"internal/integration/grsai/error.go", "internal/integration/grsai/logger.go",
		"internal/integration/grsai/logger_test.go", "internal/integration/grsai/client_integration_test.go",
		"internal/aicapability/productimage_scene_test.go",
		"tests/provider_logger_ssa_test.go", "tests/provider_logger_ssa_review_test.go",
		"tests/provider_logger_ssa_nil_analysis_test.go", "tests/provider_logger_ssa_managed_use_analysis_test.go",
		"tests/provider_logger_ssa_loading_test.go", "tests/provider_logger_ssa_analysis_test.go")
	for path, names := range map[string][]string{
		"internal/ai/contracts.go":                           {"ImageGenerator", "RouteBoundImageGenerator", "ImageAsyncSubmitResponse", "ImageAsyncQueryResponse", "ErrAsyncImageGenerationNotSupported"},
		"internal/integration/grsai/client.go":               {"GenerateImage", "EditImage", "GetDefaultModel", "SupportsAsyncImageGeneration", "SubmitImageGeneration", "SubmitImageEdit", "QueryImageGeneration", "pollGenerationResult", "fetchGenerationResult", "downloadGeneratedImages", "materializeRemoteImageReference", "logSubmitDiagnostic"},
		"internal/integration/bailian/image.go":              {"GenerateImage", "EditImage", "GetDefaultModel", "SupportsAsyncImageGeneration", "SubmitImageGeneration", "SubmitImageEdit", "QueryImageGeneration"},
		"internal/integration/grsai/source_image_adapter.go": {"NewSourceImageAdapter", "GetDefaultModel", "GenerateImage", "EditImage", "SupportsAsyncImageGeneration", "SubmitImageGeneration", "SubmitImageEdit", "QueryImageGeneration"},
		"internal/integration/grsai/synchronous_edit.go":     {"synchronousProductImageClient"},
		"internal/aicapability/model.go":                     {"CapabilityProductImageScene", "OperationProductImageReview"},
	} {
		file, err := parser.ParseFile(token.NewFileSet(), filepath.Join("..", filepath.FromSlash(path)), nil, 0)
		if err != nil {
			t.Fatal(err)
		}
		banned := make(map[string]bool)
		for _, name := range names {
			banned[name] = true
		}
		for _, declaration := range file.Decls {
			switch d := declaration.(type) {
			case *ast.FuncDecl:
				if banned[d.Name.Name] {
					t.Errorf("retired declaration %s in %s", d.Name.Name, path)
				}
			case *ast.GenDecl:
				for _, spec := range d.Specs {
					switch s := spec.(type) {
					case *ast.TypeSpec:
						if banned[s.Name.Name] {
							t.Errorf("retired type %s in %s", s.Name.Name, path)
						}
					case *ast.ValueSpec:
						for _, name := range s.Names {
							if banned[name.Name] {
								t.Errorf("retired value %s in %s", name.Name, path)
							}
						}
					}
				}
			}
		}
	}
	for _, path := range []string{
		"internal/integration/grsai/synchronous_edit.go", "internal/integration/bailian/image.go",
		"internal/product/image/source_white_background.go", "internal/integration/orgresource/model_invocation_fingerprint.go",
		"docs/architecture/legacy-image-async-review-closure-retirement-boundary.md",
	} {
		if _, err := os.Stat(filepath.Join("..", filepath.FromSlash(path))); err != nil {
			t.Errorf("current owner missing: %s: %v", path, err)
		}
	}
}

func TestCurrentImageRetirementKeepsLedgerAndRouteBoundaries(t *testing.T) {
	for path, types := range map[string]map[string][]string{
		"internal/integration/grsai/client.go": {"Config": {"Logger", "PollInterval", "MaxAttempts", "MaxReferenceMaterializationConcurrency"}},
		"internal/aicapability/invocation.go":  {"InvocationRecord": {"ReviewScore", "ReviewNeedsHumanReview", "ReviewReasons"}},
	} {
		file, err := parser.ParseFile(token.NewFileSet(), filepath.Join("..", filepath.FromSlash(path)), nil, 0)
		if err != nil {
			t.Fatal(err)
		}
		for _, decl := range file.Decls {
			g, ok := decl.(*ast.GenDecl)
			if !ok {
				continue
			}
			for _, spec := range g.Specs {
				typ, ok := spec.(*ast.TypeSpec)
				if !ok {
					continue
				}
				names, ok := types[typ.Name.Name]
				if !ok {
					continue
				}
				st, ok := typ.Type.(*ast.StructType)
				if !ok {
					t.Fatalf("current type changed: %s", typ.Name.Name)
				}
				for _, field := range st.Fields.List {
					for _, name := range field.Names {
						for _, banned := range names {
							if name.Name == banned {
								t.Errorf("retired business/config field %s.%s", typ.Name.Name, banned)
							}
						}
					}
				}
			}
		}
	}
	file, err := parser.ParseFile(token.NewFileSet(), filepath.Join("..", "internal", "integration", "orgresource", "model_invocation_repository.go"), nil, 0)
	if err != nil {
		t.Fatal(err)
	}
	seen := map[string]bool{}
	for _, decl := range file.Decls {
		f, ok := decl.(*ast.FuncDecl)
		if !ok || (f.Name.Name != "modelPointFingerprint" && f.Name.Name != "finalize") {
			continue
		}
		calls := 0
		ast.Inspect(f.Body, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}
			name, ok := call.Fun.(*ast.Ident)
			if ok && name.Name == "modelPointFactHash" {
				calls++
			}
			return true
		})
		if calls != 1 {
			t.Errorf("%s must encode the original current fact exactly once", f.Name.Name)
		}
		seen[f.Name.Name] = true
	}
	if !seen["modelPointFingerprint"] || !seen["finalize"] {
		t.Fatal("ledger hash owner missing")
	}
}

// These are frozen approved document bodies, not current progress text.
func TestImageRetirementFrozenDocumentSuffixesStayIntact(t *testing.T) {
	for _, doc := range []struct {
		path   string
		length int
		hash   string
	}{
		{"docs/architecture/legacy-image-async-review-closure-retirement-boundary.md", 36179, "e1f5b10ba5e9c8b9274509dda4618fdaa8fd15f51d9dfe660d4907b40cedd004"},
		{"docs/architecture/README.md", 157059, "027eae0200641857d23e5b959a90fa14adc16523549a2b46599e22990e00dafe"},
		{"docs/refactoring/legacy-register.md", 154765, "2531c30fe8dc82a4cc0930a9228a8333136390e6f69a91fe6c8a27f898eca729"},
		{"docs/architecture/legacy-asset-source-openai-tail-closure-retirement-boundary.md", 28031, "43b74f4726dddb776d3079e1b6a9496423236e33a26905880f69bb3d925294b8"},
		{"docs/architecture/legacy-task-openai-convenience-closure-retirement-boundary.md", 30824, "fb010580206417c7e11ce7372180c5676cf82b5fb4706e5467f16376b0d830c4"},
		{"docs/architecture/legacy-ai-routing-image-policy-closure-retirement-boundary.md", 24540, "ec9fdafc504626afc147f32b82c34517cc8cea6b379f69cddc6c15896206f378"},
		{"docs/architecture/legacy-logger-facade-closure-retirement-boundary.md", 44269, "438b8cd217abd3ecadcae1fa610112ba7816e7efac2f5a4f1f046ae3c4f8f080"},
		{"docs/architecture/legacy-idle-adapter-helper-closure-retirement-boundary.md", 30480, "ecfbc81bafc38d5755b9d0e3d094fe958a897341c87d2d2b69db1813ff790109"},
		{"docs/architecture/legacy-auth-route-s3-operation-closure-retirement-boundary.md", 29020, "d1fbcb9c1a92c7fe938e4e3e91e0387187a00adab1508c31a08b7901d80f9264"},
		{"docs/architecture/legacy-idle-provider-bootstrap-closure-retirement-boundary.md", 27995, "8a0758edade3335dc6ed62829a2eb029123dc23599c8e76b81082a23a6f20031"},
		{"docs/architecture/legacy-imageagent-generic-execution-closure-retirement-boundary.md", 40241, "61feae87ec9c0c3c83f2980ac548749e064f59cf804a52651e7663ab347bf8f1"},
	} {
		raw, err := os.ReadFile(filepath.Join("..", filepath.FromSlash(doc.path)))
		if err != nil {
			t.Fatal(err)
		}
		raw = bytes.ReplaceAll(raw, []byte("\r\n"), []byte("\n"))
		raw = restoreRelocatedFrozenDocumentNavigation(doc.path, raw)
		if len(raw) < doc.length {
			t.Errorf("frozen document truncated: %s", doc.path)
			continue
		}
		got := fmt.Sprintf("%x", sha256.Sum256(raw[len(raw)-doc.length:]))
		if got != doc.hash {
			t.Errorf("frozen history changed: %s", doc.path)
		}
	}
}

// restoreRelocatedFrozenDocumentNavigation reverses only the exact navigation
// edits authorized by the documentation relocation. Frozen bodies still use
// their original lengths and hashes; content, state and evidence changes fail.
func restoreRelocatedFrozenDocumentNavigation(path string, raw []byte) []byte {
	// The 2026-11-02 greenfield decision removes archived material from checkout.
	// Exact navigation inverses keep current retained contracts on their original
	// hashes; no arbitrary content or evidence changes are normalized away.
	var edits [][2]string
	switch filepath.ToSlash(path) {
	case "docs/architecture/README.md":
		edits = [][2]string{
			{"- 旧阶段盘点、旧协议、TEMU设计经验及PoC报告已退出当前工作树，不作为当前开发依据。", "- 历史阶段盘点、旧协议、TEMU设计经验及PoC结果已移到[全新系统开发基线](../product/greenfield-no-legacy-migration.md)，不再与当前领域合同并列。"},
			{"旧阶段资料已退出当前工作树，后续开发遵守[全新系统开发基线](../product/greenfield-no-legacy-migration.md)", "历史设计、V7计划与验证记录见[全新系统开发基线](../product/greenfield-no-legacy-migration.md)"},
			{"[openmeter-shadow-metering-poc-report.md（历史）](https://github.com/henryxu1008/shuomi-ai-commerce/blob/f936702227f871e122d478276868e63a6efcc4fc/docs/archive/architecture/openmeter-shadow-metering-poc-report.md)", "`docs/archive/architecture/openmeter-shadow-metering-poc-report.md`"},
			{"[task-event-v2-migration.md（历史）](https://github.com/henryxu1008/shuomi-ai-commerce/blob/f936702227f871e122d478276868e63a6efcc4fc/docs/archive/architecture/task-event-v2-migration.md)", "`docs/archive/architecture/task-event-v2-migration.md`"},
			{"[listingkit-refactor-status.md（历史）](https://github.com/henryxu1008/shuomi-ai-commerce/blob/f936702227f871e122d478276868e63a6efcc4fc/docs/archive/architecture/listingkit-refactor-status.md)", "`docs/archive/architecture/listingkit-refactor-status.md`"},
			{"[temu-pipeline-stages.md（历史）](https://github.com/henryxu1008/shuomi-ai-commerce/blob/f936702227f871e122d478276868e63a6efcc4fc/docs/archive/architecture/temu-pipeline-stages.md)", "`docs/archive/architecture/temu-pipeline-stages.md`"},
			{"[temu-architecture-patterns.md（历史）](https://github.com/henryxu1008/shuomi-ai-commerce/blob/f936702227f871e122d478276868e63a6efcc4fc/docs/archive/architecture/temu-architecture-patterns.md)", "`docs/archive/architecture/temu-architecture-patterns.md`"},
			{"[task-status-lifecycle.md（历史）](https://github.com/henryxu1008/shuomi-ai-commerce/blob/f936702227f871e122d478276868e63a6efcc4fc/docs/archive/architecture/task-status-lifecycle.md)", "`docs/archive/architecture/task-status-lifecycle.md`"},
			{"[historical-platform-migration-inventory.md（历史）](https://github.com/henryxu1008/shuomi-ai-commerce/blob/f936702227f871e122d478276868e63a6efcc4fc/docs/archive/architecture/historical-platform-migration-inventory.md)", "`docs/archive/architecture/historical-platform-migration-inventory.md`"},
			{"[Publication Identity draft](https://github.com/henryxu1008/shuomi-ai-commerce/blob/f936702227f871e122d478276868e63a6efcc4fc/docs/archive/refactoring/2026-09-05-publication-identity-cutover-issue-draft.md)", "[Publication Identity draft](../archive/refactoring/2026-09-05-publication-identity-cutover-issue-draft.md)"},
			{"[next-phase-plan.md](https://github.com/henryxu1008/shuomi-ai-commerce/blob/f936702227f871e122d478276868e63a6efcc4fc/docs/archive/refactoring/next-phase-plan.md)", "[next-phase-plan.md](../archive/refactoring/next-phase-plan.md)"},
			{"[Current status](https://github.com/henryxu1008/shuomi-ai-commerce/blob/f936702227f871e122d478276868e63a6efcc4fc/docs/archive/refactoring/current-refactoring-status.md)", "[Current status](../archive/refactoring/current-refactoring-status.md)"},
			{"[Identity / Organization](https://github.com/henryxu1008/shuomi-ai-commerce/blob/f936702227f871e122d478276868e63a6efcc4fc/docs/archive/designs/2026-08-30-shuomi-workbench-store-center-zitadel-multi-org-design.md)", "[Identity / Organization](../archive/designs/2026-08-30-shuomi-workbench-store-center-zitadel-multi-org-design.md)"},
			{"[全新系统开发基线](../product/greenfield-no-legacy-migration.md)", "[历史架构材料](../archive/README.md#architecture-history)"},
			{"[全新系统开发基线](../product/greenfield-no-legacy-migration.md)", "[归档索引](../archive/README.md)"},
			{"[全新系统开发基线](../product/greenfield-no-legacy-migration.md)", "[历史归档](../archive/README.md)"},
			{"`docs/archive/architecture/temu-architecture-patterns.md`", "`temu-architecture-patterns.md`"},
			{"`docs/archive/architecture/temu-pipeline-stages.md`", "`temu-pipeline-stages.md`"},
			{"`docs/archive/architecture/task-status-lifecycle.md`", "`task-status-lifecycle.md`"},
			{"`docs/archive/architecture/task-event-v2-migration.md`", "`task-event-v2-migration.md`"},
			{"`docs/archive/architecture/openmeter-shadow-metering-poc-report.md`", "`openmeter-shadow-metering-poc-report.md`"},
			{"`docs/archive/architecture/listingkit-refactor-status.md`", "`listingkit-refactor-status.md`"},
			{"`docs/archive/architecture/historical-platform-migration-inventory.md`", "`historical-platform-migration-inventory.md`"},
			{"](../archive/refactoring/2026-09-05-publication-identity-cutover-issue-draft.md)", "](../refactoring/2026-09-05-publication-identity-cutover-issue-draft.md)"},
			{"](../archive/refactoring/next-phase-plan.md)", "](../refactoring/next-phase-plan.md)"},
			{"](../archive/refactoring/current-refactoring-status.md)", "](../refactoring/current-refactoring-status.md)"},
			{"](module-target-mapping.md)", "](../refactoring/module-target-mapping.md)"},
			{"](../archive/designs/2026-08-30-shuomi-workbench-store-center-zitadel-multi-org-design.md)", "](../superpowers/specs/2026-08-30-shuomi-workbench-store-center-zitadel-multi-org-design.md)"},
			{"](product-domain-contracts.md)", "](../superpowers/specs/2026-09-01-internal-target-architecture-phase3-product-design.md)"},
		}
	case "docs/architecture/legacy-ai-routing-image-policy-closure-retirement-boundary.md":
		edits = [][2]string{
			{"[2026-08-06-ai-capability-phase-1.md（历史）](https://github.com/henryxu1008/shuomi-ai-commerce/blob/f936702227f871e122d478276868e63a6efcc4fc/docs/archive/implementation-plans/2026-08-06-ai-capability-phase-1.md)", "`docs/archive/implementation-plans/2026-08-06-ai-capability-phase-1.md`"},
			{"docs/architecture/product-domain-contracts.md", "docs/superpowers/specs/2026-09-01-internal-target-architecture-phase3-product-design.md"},
			{"docs/architecture/ai-capability-platform.md", "docs/superpowers/specs/2026-08-06-ai-capability-agent-platform-design.md"},
			{"docs/archive/implementation-plans/2026-08-06-ai-capability-phase-1.md", "docs/superpowers/plans/2026-08-06-ai-capability-phase-1.md"},
		}
	case "docs/architecture/legacy-asset-source-openai-tail-closure-retirement-boundary.md":
		edits = [][2]string{
			{"[2026-08-21-provider-neutral-ai-contracts.md（历史）](https://github.com/henryxu1008/shuomi-ai-commerce/blob/f936702227f871e122d478276868e63a6efcc4fc/docs/archive/implementation-plans/2026-08-21-provider-neutral-ai-contracts.md)", "`docs/archive/implementation-plans/2026-08-21-provider-neutral-ai-contracts.md`"},
			{"docs/architecture/product-domain-contracts.md", "docs/superpowers/specs/2026-09-01-internal-target-architecture-phase3-product-design.md"},
			{"docs/architecture/ai-provider-contracts.md", "docs/superpowers/specs/2026-08-21-provider-neutral-ai-contracts-design.md"},
			{"docs/archive/implementation-plans/2026-08-21-provider-neutral-ai-contracts.md", "docs/superpowers/plans/2026-08-21-provider-neutral-ai-contracts.md"},
			{"](ai-provider-contracts.md)", "](../superpowers/specs/2026-08-21-provider-neutral-ai-contracts-design.md)"},
		}
	case "docs/architecture/legacy-auth-route-s3-operation-closure-retirement-boundary.md":
		edits = [][2]string{
			{"[2026-08-30-internal-target-architecture-phase2.md（历史）](https://github.com/henryxu1008/shuomi-ai-commerce/blob/f936702227f871e122d478276868e63a6efcc4fc/docs/archive/implementation-plans/2026-08-30-internal-target-architecture-phase2.md)", "`docs/archive/implementation-plans/2026-08-30-internal-target-architecture-phase2.md`"},
			{"[2026-08-25-zitadel-authruntime-extraction.md（历史）](https://github.com/henryxu1008/shuomi-ai-commerce/blob/f936702227f871e122d478276868e63a6efcc4fc/docs/archive/implementation-plans/2026-08-25-zitadel-authruntime-extraction.md)", "`docs/archive/implementation-plans/2026-08-25-zitadel-authruntime-extraction.md`"},
			{"[Phase 2 迁包计划](https://github.com/henryxu1008/shuomi-ai-commerce/blob/f936702227f871e122d478276868e63a6efcc4fc/docs/archive/implementation-plans/2026-08-30-internal-target-architecture-phase2.md)", "[Phase 2 迁包计划](../archive/implementation-plans/2026-08-30-internal-target-architecture-phase2.md)"},
			{"[旧抽取计划](https://github.com/henryxu1008/shuomi-ai-commerce/blob/f936702227f871e122d478276868e63a6efcc4fc/docs/archive/implementation-plans/2026-08-25-zitadel-authruntime-extraction.md)", "[旧抽取计划](../archive/implementation-plans/2026-08-25-zitadel-authruntime-extraction.md)"},
			{"docs/architecture/authentication-runtime.md", "docs/superpowers/specs/2026-08-25-zitadel-authruntime-design.md"},
			{"docs/archive/implementation-plans/2026-08-30-internal-target-architecture-phase2.md", "docs/superpowers/plans/2026-08-30-internal-target-architecture-phase2.md"},
			{"docs/archive/implementation-plans/2026-08-25-zitadel-authruntime-extraction.md", "docs/superpowers/plans/2026-08-25-zitadel-authruntime-extraction.md"},
			{"](../archive/implementation-plans/2026-08-30-internal-target-architecture-phase2.md)", "](../superpowers/plans/2026-08-30-internal-target-architecture-phase2.md)"},
			{"](../archive/implementation-plans/2026-08-25-zitadel-authruntime-extraction.md)", "](../superpowers/plans/2026-08-25-zitadel-authruntime-extraction.md)"},
			{"](authentication-runtime.md)", "](../superpowers/specs/2026-08-25-zitadel-authruntime-design.md)"},
		}
	case "docs/architecture/legacy-idle-adapter-helper-closure-retirement-boundary.md":
		edits = [][2]string{
			{"[Phase2 原计划](https://github.com/henryxu1008/shuomi-ai-commerce/blob/f936702227f871e122d478276868e63a6efcc4fc/docs/archive/implementation-plans/2026-08-30-internal-target-architecture-phase2.md)", "[Phase2 原计划](../archive/implementation-plans/2026-08-30-internal-target-architecture-phase2.md)"},
			{"docs/archive/implementation-plans/2026-08-30-internal-target-architecture-phase2.md", "docs/superpowers/plans/2026-08-30-internal-target-architecture-phase2.md"},
			{"](../archive/implementation-plans/2026-08-30-internal-target-architecture-phase2.md)", "](../superpowers/plans/2026-08-30-internal-target-architecture-phase2.md)"},
		}
	case "docs/architecture/legacy-image-async-review-closure-retirement-boundary.md":
		edits = [][2]string{
			{"[2026-08-21-provider-neutral-ai-contracts.md（历史）](https://github.com/henryxu1008/shuomi-ai-commerce/blob/f936702227f871e122d478276868e63a6efcc4fc/docs/archive/implementation-plans/2026-08-21-provider-neutral-ai-contracts.md)", "`docs/archive/implementation-plans/2026-08-21-provider-neutral-ai-contracts.md`"},
			{"[2026-08-06-ai-capability-phase-1.md（历史）](https://github.com/henryxu1008/shuomi-ai-commerce/blob/f936702227f871e122d478276868e63a6efcc4fc/docs/archive/implementation-plans/2026-08-06-ai-capability-phase-1.md)", "`docs/archive/implementation-plans/2026-08-06-ai-capability-phase-1.md`"},
			{"[2026-08-06-ai-async-job-binding.md（历史）](https://github.com/henryxu1008/shuomi-ai-commerce/blob/f936702227f871e122d478276868e63a6efcc4fc/docs/archive/implementation-plans/2026-08-06-ai-async-job-binding.md)", "`docs/archive/implementation-plans/2026-08-06-ai-async-job-binding.md`"},
			{"[2026-06-20-listingkit-async-studio-image-generation.md（历史）](https://github.com/henryxu1008/shuomi-ai-commerce/blob/f936702227f871e122d478276868e63a6efcc4fc/docs/archive/implementation-plans/2026-06-20-listingkit-async-studio-image-generation.md)", "`docs/archive/implementation-plans/2026-06-20-listingkit-async-studio-image-generation.md`"},
			{"docs/archive/implementation-plans/2026-08-21-provider-neutral-ai-contracts.md", "docs/superpowers/plans/2026-08-21-provider-neutral-ai-contracts.md"},
			{"docs/archive/implementation-plans/2026-08-06-ai-capability-phase-1.md", "docs/superpowers/plans/2026-08-06-ai-capability-phase-1.md"},
			{"docs/archive/implementation-plans/2026-08-06-ai-async-job-binding.md", "docs/superpowers/plans/2026-08-06-ai-async-job-binding.md"},
			{"docs/archive/implementation-plans/2026-06-20-listingkit-async-studio-image-generation.md", "docs/superpowers/plans/2026-06-20-listingkit-async-studio-image-generation.md"},
		}
	case "docs/architecture/legacy-imageagent-business-signal-closure-retirement-boundary.md":
		edits = [][2]string{
			{"[2026-08-29-image-agent-external-effect-finalization.md（历史）](https://github.com/henryxu1008/shuomi-ai-commerce/blob/f936702227f871e122d478276868e63a6efcc4fc/docs/archive/implementation-plans/2026-08-29-image-agent-external-effect-finalization.md)", "`docs/archive/implementation-plans/2026-08-29-image-agent-external-effect-finalization.md`"},
			{"[2026-08-26-image-agent-core-manual-workflow.md（历史）](https://github.com/henryxu1008/shuomi-ai-commerce/blob/f936702227f871e122d478276868e63a6efcc4fc/docs/archive/implementation-plans/2026-08-26-image-agent-core-manual-workflow.md)", "`docs/archive/implementation-plans/2026-08-26-image-agent-core-manual-workflow.md`"},
			{"[2026-08-29 effect finalization 计划](https://github.com/henryxu1008/shuomi-ai-commerce/blob/f936702227f871e122d478276868e63a6efcc4fc/docs/archive/implementation-plans/2026-08-29-image-agent-external-effect-finalization.md)", "[2026-08-29 effect finalization 计划](../archive/implementation-plans/2026-08-29-image-agent-external-effect-finalization.md)"},
			{"[原实施计划](https://github.com/henryxu1008/shuomi-ai-commerce/blob/f936702227f871e122d478276868e63a6efcc4fc/docs/archive/implementation-plans/2026-08-26-image-agent-core-manual-workflow.md)", "[原实施计划](../archive/implementation-plans/2026-08-26-image-agent-core-manual-workflow.md)"},
			{"docs/architecture/image-agent-workflow.md", "docs/superpowers/specs/2026-08-26-image-agent-workflow-design.md"},
			{"docs/archive/implementation-plans/2026-08-29-image-agent-external-effect-finalization.md", "docs/superpowers/plans/2026-08-29-image-agent-external-effect-finalization.md"},
			{"docs/archive/implementation-plans/2026-08-26-image-agent-core-manual-workflow.md", "docs/superpowers/plans/2026-08-26-image-agent-core-manual-workflow.md"},
			{"](../archive/implementation-plans/2026-08-29-image-agent-external-effect-finalization.md)", "](../superpowers/plans/2026-08-29-image-agent-external-effect-finalization.md)"},
			{"](../archive/implementation-plans/2026-08-26-image-agent-core-manual-workflow.md)", "](../superpowers/plans/2026-08-26-image-agent-core-manual-workflow.md)"},
			{"](image-agent-workflow.md)", "](../superpowers/specs/2026-08-26-image-agent-workflow-design.md)"},
		}
	case "docs/architecture/legacy-imageagent-control-summary-tail-retirement-boundary.md":
		edits = [][2]string{
			{"[2026-08-26-image-agent-automatic-repair.md（历史）](https://github.com/henryxu1008/shuomi-ai-commerce/blob/f936702227f871e122d478276868e63a6efcc4fc/docs/archive/implementation-plans/2026-08-26-image-agent-automatic-repair.md)", "`docs/archive/implementation-plans/2026-08-26-image-agent-automatic-repair.md`"},
			{"[2026-08-26-image-agent-assisted-planning.md（历史）](https://github.com/henryxu1008/shuomi-ai-commerce/blob/f936702227f871e122d478276868e63a6efcc4fc/docs/archive/implementation-plans/2026-08-26-image-agent-assisted-planning.md)", "`docs/archive/implementation-plans/2026-08-26-image-agent-assisted-planning.md`"},
			{"[2026-08-26-image-agent-core-manual-workflow.md（历史）](https://github.com/henryxu1008/shuomi-ai-commerce/blob/f936702227f871e122d478276868e63a6efcc4fc/docs/archive/implementation-plans/2026-08-26-image-agent-core-manual-workflow.md)", "`docs/archive/implementation-plans/2026-08-26-image-agent-core-manual-workflow.md`"},
			{"docs/archive/implementation-plans/2026-08-26-image-agent-core-manual-workflow.md", "docs/superpowers/plans/2026-08-26-image-agent-core-manual-workflow.md"},
			{"docs/archive/implementation-plans/2026-08-26-image-agent-automatic-repair.md", "docs/superpowers/plans/2026-08-26-image-agent-automatic-repair.md"},
			{"docs/archive/implementation-plans/2026-08-26-image-agent-assisted-planning.md", "docs/superpowers/plans/2026-08-26-image-agent-assisted-planning.md"},
		}
	case "docs/architecture/legacy-imageagent-generic-execution-closure-retirement-boundary.md":
		edits = [][2]string{
			{"[2026-08-30-internal-target-architecture-phase2.md（历史）](https://github.com/henryxu1008/shuomi-ai-commerce/blob/f936702227f871e122d478276868e63a6efcc4fc/docs/archive/implementation-plans/2026-08-30-internal-target-architecture-phase2.md)", "`docs/archive/implementation-plans/2026-08-30-internal-target-architecture-phase2.md`"},
			{"[2026-08-26-image-agent-core-manual-workflow.md（历史）](https://github.com/henryxu1008/shuomi-ai-commerce/blob/f936702227f871e122d478276868e63a6efcc4fc/docs/archive/implementation-plans/2026-08-26-image-agent-core-manual-workflow.md)", "`docs/archive/implementation-plans/2026-08-26-image-agent-core-manual-workflow.md`"},
			{"[2026-08-06-ai-capability-phase-1.md（历史）](https://github.com/henryxu1008/shuomi-ai-commerce/blob/f936702227f871e122d478276868e63a6efcc4fc/docs/archive/implementation-plans/2026-08-06-ai-capability-phase-1.md)", "`docs/archive/implementation-plans/2026-08-06-ai-capability-phase-1.md`"},
			{"docs/architecture/runtime-consumer-register.md", "docs/refactoring/phase2-runtime-inventory.md"},
			{"docs/architecture/module-target-mapping.md", "docs/refactoring/module-target-mapping.md"},
			{"docs/archive/implementation-plans/2026-08-30-internal-target-architecture-phase2.md", "docs/superpowers/plans/2026-08-30-internal-target-architecture-phase2.md"},
			{"docs/archive/implementation-plans/2026-08-26-image-agent-core-manual-workflow.md", "docs/superpowers/plans/2026-08-26-image-agent-core-manual-workflow.md"},
			{"docs/archive/implementation-plans/2026-08-06-ai-capability-phase-1.md", "docs/superpowers/plans/2026-08-06-ai-capability-phase-1.md"},
		}
	case "docs/architecture/legacy-logger-facade-closure-retirement-boundary.md":
		edits = [][2]string{
			{"](runtime-consumer-register.md)", "](../refactoring/phase2-runtime-inventory.md)"},
		}
	case "docs/architecture/legacy-resource-transaction-closure-retirement-boundary.md":
		edits = [][2]string{
			{"[2026-09-03-shuomi-resource-ledger-store-design.md（历史）](https://github.com/henryxu1008/shuomi-ai-commerce/blob/f936702227f871e122d478276868e63a6efcc4fc/docs/archive/designs/2026-09-03-shuomi-resource-ledger-store-design.md)", "`docs/archive/designs/2026-09-03-shuomi-resource-ledger-store-design.md`"},
			{"[原资源/Store 设计](https://github.com/henryxu1008/shuomi-ai-commerce/blob/f936702227f871e122d478276868e63a6efcc4fc/docs/archive/designs/2026-09-03-shuomi-resource-ledger-store-design.md)", "[原资源/Store 设计](../archive/designs/2026-09-03-shuomi-resource-ledger-store-design.md)"},
			{"docs/archive/designs/2026-09-03-shuomi-resource-ledger-store-design.md", "docs/superpowers/specs/2026-09-03-shuomi-resource-ledger-store-design.md"},
			{"](../archive/designs/2026-09-03-shuomi-resource-ledger-store-design.md)", "](../superpowers/specs/2026-09-03-shuomi-resource-ledger-store-design.md)"},
		}
	case "docs/architecture/legacy-task-openai-convenience-closure-retirement-boundary.md":
		edits = [][2]string{
			{"[2026-08-30-internal-target-architecture-phase2.md（历史）](https://github.com/henryxu1008/shuomi-ai-commerce/blob/f936702227f871e122d478276868e63a6efcc4fc/docs/archive/implementation-plans/2026-08-30-internal-target-architecture-phase2.md)", "`docs/archive/implementation-plans/2026-08-30-internal-target-architecture-phase2.md`"},
			{"[2026-08-21-provider-neutral-ai-contracts.md（历史）](https://github.com/henryxu1008/shuomi-ai-commerce/blob/f936702227f871e122d478276868e63a6efcc4fc/docs/archive/implementation-plans/2026-08-21-provider-neutral-ai-contracts.md)", "`docs/archive/implementation-plans/2026-08-21-provider-neutral-ai-contracts.md`"},
			{"docs/architecture/shein-offline-validation-input-binding.md", "docs/superpowers/specs/2026-09-05-shein-offline-validation-input-binding-design.md"},
			{"docs/architecture/commerce-tool-canonical-inspection.md", "docs/superpowers/specs/2026-09-05-commerce-tool-canonical-inspection-design.md"},
			{"docs/architecture/ai-provider-contracts.md", "docs/superpowers/specs/2026-08-21-provider-neutral-ai-contracts-design.md"},
			{"docs/archive/implementation-plans/2026-08-30-internal-target-architecture-phase2.md", "docs/superpowers/plans/2026-08-30-internal-target-architecture-phase2.md"},
			{"docs/archive/implementation-plans/2026-08-21-provider-neutral-ai-contracts.md", "docs/superpowers/plans/2026-08-21-provider-neutral-ai-contracts.md"},
		}
	case "docs/architecture/project-boundaries.md":
		edits = [][2]string{
			{"[project-wide-refactoring-plan.md（历史）](https://github.com/henryxu1008/shuomi-ai-commerce/blob/f936702227f871e122d478276868e63a6efcc4fc/docs/archive/refactoring/project-wide-refactoring-plan.md)", "`docs/archive/refactoring/project-wide-refactoring-plan.md`"},
			{"[`docs/archive/refactoring/project-wide-refactoring-plan.md`](https://github.com/henryxu1008/shuomi-ai-commerce/blob/f936702227f871e122d478276868e63a6efcc4fc/docs/archive/refactoring/project-wide-refactoring-plan.md)", "[`docs/archive/refactoring/project-wide-refactoring-plan.md`](../archive/refactoring/project-wide-refactoring-plan.md)"},
			{"docs/archive/refactoring/project-wide-refactoring-plan.md", "docs/refactoring/project-wide-refactoring-plan.md"},
			{"](runtime-consumer-register.md)", "](../refactoring/phase2-runtime-inventory.md)"},
			{"](module-target-mapping.md)", "](../refactoring/module-target-mapping.md)"},
			{"](../archive/refactoring/project-wide-refactoring-plan.md)", "](../refactoring/project-wide-refactoring-plan.md)"},
		}
	case "docs/architecture/store-center-current-application-v1.md":
		edits = [][2]string{
			{"[当前统一资源合同](unified-base-prepaid-resources-v1.md)", "[V7 Resource / Store Service 计划](../archive/implementation-plans/2026-09-03-shuomi-resource-ledger-store-plan-v7.md)"},
			{"[身份与租户合同](auth-and-tenancy.md)", "[现有身份/企业空间设计](../archive/designs/2026-08-30-shuomi-workbench-store-center-zitadel-multi-org-design.md)"},
			{"](../archive/implementation-plans/2026-09-03-shuomi-resource-ledger-store-plan-v7.md)", "](../superpowers/plans/2026-09-03-shuomi-resource-ledger-store-plan-v7.md)"},
			{"](../archive/designs/2026-08-30-shuomi-workbench-store-center-zitadel-multi-org-design.md)", "](../superpowers/specs/2026-08-30-shuomi-workbench-store-center-zitadel-multi-org-design.md)"},
		}
	case "docs/refactoring/legacy-register.md":
		edits = [][2]string{
			{"](../architecture/module-target-mapping.md#21-bounded-current-delivery-calibration-386)", "](module-target-mapping.md#21-bounded-current-delivery-calibration-386)"},
		}
	}
	for _, edit := range edits {
		raw = bytes.ReplaceAll(raw, []byte(edit[0]), []byte(edit[1]))
	}
	return raw
}
