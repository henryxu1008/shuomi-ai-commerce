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
		{"docs/superpowers/plans/2026-06-20-listingkit-async-studio-image-generation.md", 25685, "d65a43c2f17e147959a994aff4fa4d5746990734d2dcfa4b718db2741a64a111"},
		{"docs/superpowers/plans/2026-08-06-ai-async-job-binding.md", 16069, "6c67ea9d51fb7c17ebc8b848245993a99e4716173dfa04157f34069f72f6150a"},
		{"docs/superpowers/plans/2026-08-06-ai-capability-phase-1.md", 60142, "27fce34434d70d03cf80366473e602664f295ab8351a0b2de96f8cb69dbf8115"},
		{"docs/superpowers/plans/2026-08-21-provider-neutral-ai-contracts.md", 11681, "bc2cfb4775d655c630e89b187e5eed5d077c8e932e5dadbc1655a85f1c7a7d86"},
	} {
		raw, err := os.ReadFile(filepath.Join("..", filepath.FromSlash(doc.path)))
		if err != nil {
			t.Fatal(err)
		}
		raw = bytes.ReplaceAll(raw, []byte("\r\n"), []byte("\n"))
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
