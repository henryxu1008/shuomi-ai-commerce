package httpapi

import (
	"context"
	"crypto/sha256"
	"encoding/hex"

	"task-processor/internal/aicapability"
	"task-processor/internal/aiworkbench"
	orgresourceadapter "task-processor/internal/integration/orgresource"
)

// verifyStoreRouteSettlement never changes either owner. A successful model
// record alone is insufficient: resource settlement may have failed later.
func verifyStoreRouteSettlement(ctx context.Context, command aiworkbench.StoreRouteCommand,
	readInvocation func(context.Context, string, string) (aicapability.InvocationRecord, error),
	readReservation func(context.Context, string, string) (orgresourceadapter.ModelInvocationReservation, error),
) error {
	if ctx == nil || readInvocation == nil || readReservation == nil || command.Scope.OrganizationID == "" || command.Scope.ActorID == "" || command.InvocationID == "" || command.MemberID == "" || command.InputHash == "" {
		return aiworkbench.ErrUnavailable
	}
	fact, err := readInvocation(ctx, command.Scope.OrganizationID, command.InvocationID)
	if err != nil || fact.TenantID != command.Scope.OrganizationID || fact.UserID != command.Scope.ActorID || fact.MemberID != command.MemberID || fact.InvocationID != command.InvocationID || fact.InputHash != command.InputHash || fact.Operation != aicapability.OperationAIWorkbenchStoreRoute || fact.Outcome != aicapability.InvocationSucceeded || !fact.UsageKnown || !fact.EstimatedCostKnown {
		return aiworkbench.ErrUnavailable
	}
	reservation, err := readReservation(ctx, command.Scope.OrganizationID, command.InvocationID)
	if err != nil || reservation.OrganizationID != command.Scope.OrganizationID || reservation.InvocationID != command.InvocationID || reservation.MemberID != command.MemberID || reservation.State != "committed" || reservation.SettledAt == nil || reservation.SettledAt.IsZero() {
		return aiworkbench.ErrUnavailable
	}
	return nil
}

func verifyStoreRouteDecision(ctx context.Context, command aiworkbench.StoreRouteCommand, decision aiworkbench.StoreRoute,
	readInvocation func(context.Context, string, string) (aicapability.InvocationRecord, error),
	readReservation func(context.Context, string, string) (orgresourceadapter.ModelInvocationReservation, error),
) error {
	parsed, err := aiworkbench.ParseStoreRoute([]byte(decision.RawOutput))
	if err != nil || parsed.Kind != decision.Kind || parsed.Status != decision.Status {
		return aiworkbench.ErrUnavailable
	}
	if err = verifyStoreRouteSettlement(ctx, command, readInvocation, readReservation); err != nil {
		return err
	}
	fact, err := readInvocation(ctx, command.Scope.OrganizationID, command.InvocationID)
	sum := sha256.Sum256([]byte(decision.RawOutput))
	if err != nil || fact.OutputHash != hex.EncodeToString(sum[:]) {
		return aiworkbench.ErrUnavailable
	}
	return nil
}
