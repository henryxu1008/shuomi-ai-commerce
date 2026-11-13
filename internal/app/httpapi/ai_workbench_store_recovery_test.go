package httpapi

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"task-processor/internal/aicapability"
	"task-processor/internal/aiworkbench"
	orgresourceadapter "task-processor/internal/integration/orgresource"
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
