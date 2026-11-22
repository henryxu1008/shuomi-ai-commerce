package httpapi

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"testing"
	"time"

	"github.com/henryxu1008/shuomi-ai-commerce/internal/aicapability"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/aiworkbench"
	orgresourceadapter "github.com/henryxu1008/shuomi-ai-commerce/internal/integration/orgresource"
	"github.com/stretchr/testify/require"
)

func TestStoreRouteRecoveryRequiresMatchingSettledOwners(t *testing.T) {
	ctx := context.Background()
	command := aiworkbench.StoreRouteCommand{Scope: aiworkbench.Scope{OrganizationID: "org", ActorID: "actor"}, InvocationID: "invocation", MemberID: "member", InputHash: "input"}
	fact := aicapability.InvocationRecord{TenantID: "org", UserID: "actor", MemberID: "member", InvocationID: "invocation", InputHash: "input", Operation: aicapability.Operation("aiworkbench_store_route"), Outcome: aicapability.InvocationSucceeded, UsageKnown: true, EstimatedCostKnown: true}
	raw := `{"kind":"STORE_OVERVIEW","status":"ACTIVE"}`
	sum := sha256.Sum256([]byte(raw))
	fact.OutputHash = hex.EncodeToString(sum[:])
	now := time.Now()
	reservation := orgresourceadapter.ModelInvocationReservation{OrganizationID: "org", InvocationID: "invocation", MemberID: "member", State: "committed", SettledAt: &now}
	verify := func() error {
		return verifyStoreRouteSettlement(ctx, command, func(context.Context, string, string) (aicapability.InvocationRecord, error) { return fact, nil }, func(context.Context, string, string) (orgresourceadapter.ModelInvocationReservation, error) {
			return reservation, nil
		})
	}
	require.NoError(t, verify())
	readFact := func(context.Context, string, string) (aicapability.InvocationRecord, error) { return fact, nil }
	readReservation := func(context.Context, string, string) (orgresourceadapter.ModelInvocationReservation, error) {
		return reservation, nil
	}
	require.NoError(t, verifyStoreRouteDecision(ctx, command, aiworkbench.StoreRoute{Kind: "STORE_OVERVIEW", Status: "ACTIVE", RawOutput: raw}, readFact, readReservation))
	require.Error(t, verifyStoreRouteDecision(ctx, command, aiworkbench.StoreRoute{Kind: "STORE_OVERVIEW", Status: "ALL_NON_DELETED", RawOutput: raw}, readFact, readReservation))
	require.Error(t, verifyStoreRouteDecision(ctx, command, aiworkbench.StoreRoute{Kind: "STORE_OVERVIEW", Status: "ACTIVE", RawOutput: raw + " "}, readFact, readReservation))
	reservation.State = "reserved"
	require.Error(t, verify())
	reservation.State = "committed"
	reservation.SettledAt = nil
	require.Error(t, verify())
	reservation.SettledAt = &now
	reservation.MemberID = "other"
	require.Error(t, verify())
	reservation.MemberID = "member"
	fact.InputHash = "other"
	require.Error(t, verify())
	fact.InputHash = "input"
	fact.Outcome = aicapability.InvocationDispatched
	require.Error(t, verify())
}

func TestChatIntentDecisionRequiresItsOwnPurposeAndFrozenAvailableCreator(t *testing.T) {
	ctx := context.Background()
	envelope := aiworkbench.ChatTurnEnvelope{Version: 1, Input: aiworkbench.ChatTurnInput{Content: "创建标题技能"}, Creator: &aiworkbench.ChatSkillReference{Version: 2, Digest: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"}, GeneralScope: true}
	meta, _ := json.Marshal(envelope)
	command := aiworkbench.StoreRouteCommand{Scope: aiworkbench.Scope{OrganizationID: "org", ActorID: "actor"}, InvocationID: "inv", MemberID: "member", InputHash: "input", TurnInput: meta}
	raw := `{"kind":"CREATE_TITLE_SKILL","status":""}`
	hash := sha256.Sum256([]byte(raw))
	fact := aicapability.InvocationRecord{TenantID: "org", UserID: "actor", MemberID: "member", InvocationID: "inv", InputHash: "input", Operation: aicapability.OperationAIWorkbenchChatIntent, Outcome: aicapability.InvocationSucceeded, UsageKnown: true, EstimatedCostKnown: true, OutputHash: hex.EncodeToString(hash[:])}
	now := time.Now()
	reservation := orgresourceadapter.ModelInvocationReservation{OrganizationID: "org", InvocationID: "inv", MemberID: "member", State: "committed", SettledAt: &now}
	kind := "CREATE_TITLE_SKILL"
	verify := func() error {
		return verifyStoreRouteDecision(ctx, command, aiworkbench.StoreRoute{Kind: kind, RawOutput: raw}, func(context.Context, string, string) (aicapability.InvocationRecord, error) { return fact, nil }, func(context.Context, string, string) (orgresourceadapter.ModelInvocationReservation, error) {
			return reservation, nil
		})
	}
	require.NoError(t, verify())
	fact.BillingMode, fact.EstimatedCostKnown = aicapability.ModelUsageOnly, false
	require.NoError(t, verifyStoreRouteDecision(ctx, command, aiworkbench.StoreRoute{Kind: kind, RawOutput: raw},
		func(context.Context, string, string) (aicapability.InvocationRecord, error) { return fact, nil },
		func(context.Context, string, string) (orgresourceadapter.ModelInvocationReservation, error) {
			t.Fatal("owned Chat intent must use only its original usage proof")
			return orgresourceadapter.ModelInvocationReservation{}, nil
		}))
	fact.BillingMode, fact.EstimatedCostKnown = "", true
	fact.Operation = aicapability.OperationAIWorkbenchStoreRoute
	require.Error(t, verify(), "a store-purpose settlement cannot prove a Chat intent call")
	fact.Operation = aicapability.OperationAIWorkbenchChatIntent
	envelope.Input.CreatorOnly = true
	command.TurnInput, _ = json.Marshal(envelope)
	require.NoError(t, verify())
	kind, raw = "PRODUCT_TASK", `{"kind":"PRODUCT_TASK","status":""}`
	hash = sha256.Sum256([]byte(raw))
	fact.OutputHash = hex.EncodeToString(hash[:])
	require.Error(t, verify(), "the original creation-only request must reject a product receipt")
	kind, raw = "CLARIFY", `{"kind":"CLARIFY","status":""}`
	hash = sha256.Sum256([]byte(raw))
	fact.OutputHash = hex.EncodeToString(hash[:])
	require.NoError(t, verify(), "clarification remains recoverable through the same receipt")
	kind, raw = "CREATE_TITLE_SKILL", `{"kind":"CREATE_TITLE_SKILL","status":""}`
	hash = sha256.Sum256([]byte(raw))
	fact.OutputHash = hex.EncodeToString(hash[:])
	envelope.Creator = nil
	command.TurnInput, _ = json.Marshal(envelope)
	require.Error(t, verify(), "the decision cannot add an unavailable tool")
}
