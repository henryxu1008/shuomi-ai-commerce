package assetinspect

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
	"task-processor/internal/commercetool"
	"task-processor/internal/product/asset"
)

type imageAdvicePrincipal struct{ calls int }

func (p *imageAdvicePrincipal) ResolveFreshPrincipal(ctx context.Context) (commercetool.Principal, error) {
	p.calls++
	return (principalFixture{}).ResolvePrincipal(ctx)
}

func TestImageAdviceUsesAuditedExactReadersWithoutStoreOrMutation(t *testing.T) {
	for _, scenario := range []string{"unknown_dimensions", "known_dimensions", "not_ready", "explicit_empty", "corrupt_scope", "dependency_error"} {
		t.Run(scenario, func(t *testing.T) {
			published, inventory := projectionFixture()
			if scenario == "known_dimensions" {
				inventory.Assets[0].Width, inventory.Assets[0].Height = 900, 900
			}
			snapshots := &snapshotFixture{value: published}
			assets := &inventoryFixture{value: inventory}
			switch scenario {
			case "not_ready":
				assets.err = asset.ErrApprovedAssetsNotReady
			case "explicit_empty":
				assets.value.Assets = []asset.ApprovedAsset{}
			case "corrupt_scope":
				assets.value.Scope.ProductKey = "other-product"
			case "dependency_error":
				assets.err = asset.ErrRepositoryUnavailable
			}
			definition := ImageAdviceDefinition()
			agent := commercetool.AgentDefinition{ID: "test.agent", Version: "v1.0.0", AllowedTools: []commercetool.ToolRef{definition.Ref}}
			principal := &imageAdvicePrincipal{}
			invoker, err := NewImageAdviceInvoker(snapshots, assets, principal, agent, testDependencies())
			require.NoError(t, err)
			var input Input
			require.NoError(t, json.Unmarshal([]byte(validArguments), &input))
			result, err := invoker.Invoke(context.Background(), testMetadata(), input)
			if scenario == "corrupt_scope" || scenario == "dependency_error" {
				require.Error(t, err)
				require.Empty(t, result.Output)
				return
			}
			require.NoError(t, err)
			require.Equal(t, commercetool.AuditStatusRecorded, result.AuditStatus)
			advice, err := asset.DecodeImageAdvice(result.Output)
			require.NoError(t, err)
			require.Equal(t, testMetadata().CallID, advice.Evidence.CallID)
			require.Equal(t, definition.Ref.ID, advice.Evidence.ToolID)
			require.Equal(t, published.PublicationID, advice.Binding.PublicationID)
			require.Nil(t, advice.Binding.Target, "generic image facts cannot invent a Store/Target")
			require.Equal(t, 1, principal.calls)
			require.Equal(t, 1, snapshots.calls)
			require.Equal(t, 1, assets.calls)
			wantCode, wantStatus := "dimensions_unknown", "unknown"
			switch scenario {
			case "known_dimensions":
				wantCode, wantStatus = "dimensions_known", "known_satisfied"
			case "not_ready":
				wantCode, wantStatus = "approved_assets_unavailable", "unknown"
			case "explicit_empty":
				wantCode, wantStatus = "approved_assets_unavailable", "known_conflict"
			}
			found := false
			for _, check := range advice.Checks {
				if check.Code == wantCode {
					require.Equal(t, wantStatus, check.Status)
					found = true
				}
			}
			require.True(t, found)
			tampered := advice
			tampered.Binding.PublicationID = "changed"
			require.Error(t, tampered.Verify(), "recorded facts cannot be rebound by a consumer")
			require.Equal(t, commercetool.RiskCompute, definition.Risk)
			require.Equal(t, commercetool.SideEffectNone, definition.SideEffects.Mode)
			require.Equal(t, commercetool.UsageOwnerUnmetered, definition.Usage.Owner)
		})
	}
}
