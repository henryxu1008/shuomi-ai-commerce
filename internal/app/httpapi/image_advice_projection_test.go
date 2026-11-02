package httpapi

import (
	"encoding/json"
	"github.com/stretchr/testify/require"
	"task-processor/internal/agent"
	"task-processor/internal/agentconfig"
	"task-processor/internal/commercetool"
	"task-processor/internal/product/asset"
	"task-processor/internal/product/asset/tools/assetinspect"
	"task-processor/internal/product/collection"
	"testing"
)

func TestImageAdviceProjectionUsesOnlyExactRecordedOutput(t *testing.T) {
	for _, scenario := range []string{"exact", "wrong_call", "wrong_binding", "failed_audit", "model_output", "unknown_field", "historical"} {
		t.Run(scenario, func(t *testing.T) {
			inv := asset.ApprovedAssetInventory{Scope: asset.InventoryScope{TenantID: "org", ProductKey: "product", SourceSnapshotVersion: 1, TargetPlatform: "shein"}, Assets: []asset.ApprovedAsset{}}
			a, err := asset.AnalyzeImageFacts(asset.ImageAdviceBinding{ProductKey: "product", CatalogVersion: "1", PublicationID: "publication", TargetPlatform: "shein", InventoryDigest: collection.Digest(inv)}, inv, false)
			require.NoError(t, err)
			ref := assetinspect.ImageAdviceDefinition().Ref
			require.NoError(t, a.Seal(ref.ID, ref.Version, "call"))
			raw, err := json.Marshal(a)
			require.NoError(t, err)
			s := agent.State{Request: agent.Request{Binding: agent.Binding{ProductKey: "product", CatalogVersion: "1", PublicationID: "publication", TargetPlatform: "shein"}, PolicyVersion: agentconfig.TitlePolicyVersion, PromptVersion: agentconfig.TitlePromptVersion}, History: []agent.Observation{{Tool: ref, CallID: "call", AuditStatus: commercetool.AuditStatusRecorded, Output: raw}}}
			switch scenario {
			case "wrong_call":
				s.History[0].CallID = "other"
			case "wrong_binding":
				s.Request.Binding.PublicationID = "other"
			case "failed_audit":
				s.History[0].AuditStatus = commercetool.AuditStatusRecordFailed
			case "model_output":
				s.History[0].InvocationID = "model"
			case "unknown_field":
				s.History[0].Output = append(append([]byte{}, raw[:len(raw)-1]...), []byte(`,"execute":true}`)...)
			case "historical":
				s.Request.PromptVersion = "product-title-agent-v1"
				s.History[0].Output = []byte("not reparsed on historical read")
			}
			value, err := recordedImageAdvice(s, false)
			if scenario == "historical" || scenario == "failed_audit" {
				require.NoError(t, err)
				require.Nil(t, value)
				return
			}
			if scenario != "exact" {
				require.Error(t, err)
				require.Nil(t, value)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, value)
			require.Equal(t, "unknown", value.Checks[0].Status)
		})
	}
}
