package tests

import (
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
		"internal/ai/contracts.go":              {"ImageGenerator", "RouteBoundImageGenerator", "ImageAsyncSubmitResponse", "ImageAsyncQueryResponse", "ErrAsyncImageGenerationNotSupported"},
		"internal/integration/grsai/client.go":  {"GenerateImage", "EditImage", "GetDefaultModel", "SupportsAsyncImageGeneration", "SubmitImageGeneration", "SubmitImageEdit", "QueryImageGeneration", "pollGenerationResult", "fetchGenerationResult", "downloadGeneratedImages", "materializeRemoteImageReference", "logSubmitDiagnostic"},
		"internal/integration/bailian/image.go": {"GenerateImage", "EditImage", "GetDefaultModel", "SupportsAsyncImageGeneration", "SubmitImageGeneration", "SubmitImageEdit", "QueryImageGeneration"},
		"internal/aicapability/model.go":        {"CapabilityProductImageScene", "OperationProductImageReview"},
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
