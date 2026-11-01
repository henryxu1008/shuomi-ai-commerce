package tests

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLegacyImageAgentControlSummaryClosureIsRetired(t *testing.T) {
	retired := map[string]bool{
		"AssetRef": true, "clonePendingCommand": true, "initialActionPhase": true,
		"validateUpdateIdentity": true, "validateApproveResultsBusiness": true,
		"summarizeResultsV3": true, "slotProjectionsV3": true,
	}
	// Parse every Go file, including tests and build-tagged files. Exact identifiers
	// also catch aliases/function values without banning current V3 or asset types.
	err := filepath.WalkDir(filepath.Join("..", "internal", "imageagent"), func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() || !strings.HasSuffix(path, ".go") {
			return nil
		}
		file, err := parser.ParseFile(token.NewFileSet(), path, nil, parser.SkipObjectResolution)
		if err != nil {
			return err
		}
		ast.Inspect(file, func(node ast.Node) bool {
			if id, ok := node.(*ast.Ident); ok && retired[id.Name] {
				t.Errorf("retired ImageAgent control/summary identifier %s in %s", id.Name, path)
			}
			return true
		})
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestImageAgentControlSummaryRetirementKeepsCurrentOwners(t *testing.T) {
	for path, required := range map[string][]string{
		"internal/imageagent/model.go":    {"AuthorizedAsset", "AssetCandidate", "PendingCommandReceipt"},
		"internal/imageagent/ports.go":    {"SlotExecutionResult", "CommandAcknowledgement"},
		"internal/imageagent/artifact.go": {"StagedAssetRef"},
		"internal/imageagent/temporal/types.go": {
			"WorkflowInput", "SlotWorkflowResult", "SlotWorkflowV3Input", "SlotWorkflowV3Result",
			"ReplacePlanSignal", "RetrySlotSignal", "ApproveResultsSignal", "CancelSignal", "ResumeCommandInput",
		},
		"internal/imageagent/temporal/workflow.go": {
			"workflowUpdateState", "register", "handleReplacePlan", "handleRetrySlot", "handleApproveResults",
			"handleResultAdoption", "handleCancel", "handleResume", "prepareAction", "runActionAttempt",
			"validateCommandOwner", "validateCommandRevision", "validateResultAdoptionBusiness",
			"summarizeCurrentResults", "summarizeResults", "slotProjections",
			"updatePhaseReplacePersistPlan", "updatePhaseRetryExecuteChild", "updatePhaseApprovalPublish", "updatePhaseCancelPersist",
		},
	} {
		file, err := parser.ParseFile(token.NewFileSet(), filepath.Join("..", filepath.FromSlash(path)), nil, parser.SkipObjectResolution)
		if err != nil {
			t.Fatal(err)
		}
		declared := map[string]bool{}
		ast.Inspect(file, func(node ast.Node) bool {
			switch declaration := node.(type) {
			case *ast.FuncDecl:
				declared[declaration.Name.Name] = true
			case *ast.TypeSpec:
				declared[declaration.Name.Name] = true
			case *ast.ValueSpec:
				for _, name := range declaration.Names {
					declared[name.Name] = true
				}
			}
			return true
		})
		for _, name := range required {
			if !declared[name] {
				t.Errorf("current ImageAgent owner %s missing from %s", name, path)
			}
		}
	}
}

// The approved design HEAD 44fa04b6fb8cc151974fe28ce80c3233f7b9af4b is
// immutable history. Current implementation notes may only be prepended.
func TestImageAgentControlSummaryRetirementFrozenDocumentSuffixesStayIntact(t *testing.T) {
	for _, doc := range []struct {
		path   string
		length int
		hash   string
	}{
		{"docs/architecture/legacy-imageagent-control-summary-tail-retirement-boundary.md", 19658, "17bdeee63654fe88e9c9c8bb280cfdaf9050b419c3cda03d4a409000a71d5df7"},
		{"docs/refactoring/legacy-register.md", 157812, "358c4643367bfb9f367d443a86546efe04f61decb2691e8d866988520cca42c4"},
		{"docs/architecture/project-boundaries.md", 55912, "ac4815f08eb7b86da3ca5e55b7af6c9dc1fa090e42a4e632e529681444babac4"},
		{"docs/architecture/README.md", 158183, "52676d98ac61f295c63d6cd704a3f80c17c65f847a9e15f9c75114e2f827f0ed"},
		{"docs/architecture/legacy-imageagent-business-signal-closure-retirement-boundary.md", 25547, "191a9f9d5a98ee669d8d9e8eb13295b14f7b9b619bfeee9fd3504a6001b671f9"},
		{"docs/archive/implementation-plans/2026-08-26-image-agent-core-manual-workflow.md", 26828, "a8e80f22333a7e14f662b1f64a774c3b3a68b3f70b052bd7a07cd5561c6e394d"},
		{"docs/archive/implementation-plans/2026-08-26-image-agent-assisted-planning.md", 16514, "99cc4f666256bde3a7213b5f9f90013c4e06672d70295d35de7c3df297c72709"},
		{"docs/archive/implementation-plans/2026-08-26-image-agent-automatic-repair.md", 19377, "1c9148c64927d677776c06ca327cfea1516e96177a2e783e5b76dce3b6521efd"},
	} {
		raw, err := os.ReadFile(filepath.Join("..", filepath.FromSlash(doc.path)))
		if err != nil {
			t.Fatal(err)
		}
		raw = bytes.ReplaceAll(raw, []byte("\r\n"), []byte("\n"))
		raw = restoreRelocatedFrozenDocumentNavigation(doc.path, raw)
		if len(raw) < doc.length {
			t.Errorf("frozen ImageAgent document truncated: %s", doc.path)
			continue
		}
		if fmt.Sprintf("%x", sha256.Sum256(raw[len(raw)-doc.length:])) != doc.hash {
			t.Errorf("frozen ImageAgent control/summary history changed: %s", doc.path)
		}
	}
}
