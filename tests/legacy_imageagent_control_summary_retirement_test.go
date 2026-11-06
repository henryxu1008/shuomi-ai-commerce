package tests

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
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
