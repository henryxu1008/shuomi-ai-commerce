package orgresourceadapter

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"task-processor/internal/aicapability"
	resource "task-processor/internal/ledger/orgresource"
)

func TestModelPointFingerprintKeepsOriginalClaimWhenUsagePartitionsArrive(t *testing.T) {
	fact := aicapability.InvocationRecord{InvocationID: "invocation", TenantID: "org", MemberID: "member", UserID: "actor", AgentRunID: "run", Operation: aicapability.OperationProductAgentDecision, InputHash: "input", StartedAt: time.Now().UTC(), Outcome: aicapability.InvocationDispatched}
	original := modelPointFingerprint(fact)
	fact.BillableUsage = &aicapability.BillableModelUsage{MappingVersion: "openai-text-v1", Complete: true, InputText: 5, CachedInputText: 2, OutputText: 2}
	fact.PromptTokens, fact.CompletionTokens, fact.TotalTokens = 5, 2, 7
	fact.Outcome, fact.UsageKnown = aicapability.InvocationSucceeded, true
	require.Equal(t, original, modelPointFingerprint(fact), "observed counters cannot change the original reservation identity")
}

func TestOfficialPriceQuantityAndResourceDisplayUseTheSameMicrounit(t *testing.T) {
	require.Equal(t, aicapability.AIPointMicrounitV1, resource.AIPointQuantityUnitV1)
	quantity := resource.AIPointQuantity{Micropoints: 745500, UnitVersion: aicapability.AIPointMicrounitV1}
	text, err := quantity.PointsString()
	require.NoError(t, err)
	require.Equal(t, "0.7455", text)
	parsed, err := resource.ParseAIPointQuantity(text)
	require.NoError(t, err)
	require.Equal(t, quantity, parsed)
}
