package temporal

import (
	"github.com/stretchr/testify/require"
	historypb "go.temporal.io/api/history/v1"
	sdkconverter "go.temporal.io/sdk/converter"
	"go.temporal.io/sdk/worker"
	sdkworkflow "go.temporal.io/sdk/workflow"
	"google.golang.org/protobuf/encoding/protojson"
	"os"
	"path/filepath"
	"strings"
	"task-processor/internal/imageagent"
	"testing"
)

// These hand-authored prefixes are synthetic. They are not server histories or
// runtime acceptance. Both replayed with the frozen baseline production code.
func TestSyntheticParentHistoryPrefixesReplay(t *testing.T) {
	current := readHistoryFixture(t, "current-organization-parent-prefix.json")
	require.Equal(t, []string{commandIngressPlanPolicyPatch, budgetAuthorizationPatch, externalEffectFinalizationPatch, projectionExecutionCommitPatch,
		slotExecutionWireV3Patch, approvalActionIDV3Patch, approvalPublicationWireV3Patch, resultDigestV3Patch,
		approvalPublicationScopePatch, approvalPublicationKeyLengthPatch, activityWireV2Patch, effectRecoveryStartWireV1Patch}, historyVersionChangeIDs(t, current))
	require.Equal(t, []string{activityPersistRunState}, scheduledActivityNames(current))
	require.NoError(t, replayParentPrefix(current))
	retired := readHistoryFixture(t, "retired-default-version-parent-prefix.json")
	require.Empty(t, historyVersionChangeIDs(t, retired))
	require.Empty(t, scheduledActivityNames(retired))
	err := replayParentPrefix(retired)
	require.ErrorContains(t, err, "[TMPRL1100] Workflow code removed support of version -1")
	require.ErrorContains(t, err, commandIngressPlanPolicyPatch)
}

func replayParentPrefix(history *historypb.History) error {
	r := worker.NewWorkflowReplayer()
	r.RegisterWorkflowWithOptions(ImageAgentWorkflow, sdkworkflow.RegisterOptions{Name: organizationWorkflowName})
	return r.ReplayWorkflowHistory(nil, history)
}

func assertParentVersionRejected(t *testing.T, id string) {
	t.Helper()
	history := readHistoryFixture(t, "current-organization-parent-prefix.json")
	for _, event := range history.Events {
		if event.GetMarkerRecordedEventAttributes() == nil {
			continue
		}
		var change string
		require.NoError(t, sdkconverter.GetDefaultDataConverter().FromPayloads(event.GetMarkerRecordedEventAttributes().Details["change-id"], &change))
		if change == id {
			// Keep a structurally valid prefix: an unrelated Version marker is
			// allowed by the SDK, but cannot satisfy this reached change ID.
			payload, err := sdkconverter.GetDefaultDataConverter().ToPayloads("synthetic-unreached:" + id)
			require.NoError(t, err)
			event.GetMarkerRecordedEventAttributes().Details["change-id"] = payload
			break
		}
	}
	err := replayParentPrefix(history)
	require.ErrorContains(t, err, "[TMPRL1100] Workflow code removed support of version -1")
	require.ErrorContains(t, err, id)
}
func historyVersionChangeIDs(t *testing.T, history *historypb.History) []string {
	t.Helper()
	var changeIDs []string
	for _, event := range history.Events {
		attributes := event.GetMarkerRecordedEventAttributes()
		if attributes == nil || attributes.GetMarkerName() != "Version" {
			continue
		}
		payloads := attributes.GetDetails()["change-id"]
		if payloads == nil {
			continue
		}
		var changeID string
		require.NoError(t, sdkconverter.GetDefaultDataConverter().FromPayloads(payloads, &changeID))
		changeIDs = append(changeIDs, changeID)
	}
	return changeIDs
}

func scheduledActivityNames(history *historypb.History) []string {
	names := make([]string, 0, len(history.Events))
	for _, event := range history.Events {
		attributes := event.GetActivityTaskScheduledEventAttributes()
		if attributes == nil {
			continue
		}
		names = append(names, attributes.GetActivityType().GetName())
	}
	return names
}

func wireProbePlanAndResults() (imageagent.Plan, []SlotWorkflowResult) {
	const sha = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	plan := imageagent.Plan{Revision: 1, Slots: []imageagent.Slot{{ID: "slot-1", Role: imageagent.SlotRoleScene}}}
	results := []SlotWorkflowResult{{
		Execution: imageagent.SlotExecutionResult{SlotID: "slot-1", Attempt: 1, Candidates: []imageagent.AssetCandidate{{
			AssetID: "candidate-1", DurableAsset: imageagent.DurableAssetIdentity{ObjectKey: "image-agent/public/tenant-a/capture-run/1/slot-1/1/0-" + sha + ".png", SHA256: sha},
		}}},
		Status: imageagent.SlotStatusAccepted,
	}}
	return plan, results
}

func readHistoryFixture(t *testing.T, name string) *historypb.History {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join("testdata", name))
	require.NoError(t, err)
	history := &historypb.History{}
	require.NoError(t, protojson.Unmarshal(raw, history))
	return history
}

func TestRunScopedApprovalPublicationKeyFitsReceiptSchema(t *testing.T) {
	key := approvalActionPublicationKey(strings.Repeat("a", imageagent.MaxActionIDLength), strings.Repeat("r", 64), imageagent.MaxJSONSafePlanRevision)
	require.LessOrEqual(t, len(key), 128)
}
