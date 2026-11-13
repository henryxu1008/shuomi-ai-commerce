package aiworkbench

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type routeStoreFixture struct {
	command    StoreRouteCommand
	found      bool
	completes  int
	lostChecks int
}

func (f *routeStoreFixture) FinalizeStoreRouteFailure(ctx context.Context, _ Scope, _ string, verify func(context.Context, StoreRouteCommand) (PlanningState, error)) (StoreRouteCommand, bool, error) {
	state, err := verify(ctx, f.command)
	if err != nil {
		return StoreRouteCommand{}, false, err
	}
	f.command.State = string(state)
	return f.command, false, nil
}
func (f *routeStoreFixture) FinalizeStoreRouteAbsent(context.Context, Scope, string, func(context.Context, string) (bool, error)) (StoreRouteCommand, bool, error) {
	return StoreRouteCommand{}, false, ErrUnavailable
}

func (f *routeStoreFixture) ReplayStoreRoute(context.Context, Scope, string, string, string, string) (StoreRouteCommand, bool, error) {
	return f.command, f.found, nil
}
func (f *routeStoreFixture) BeginStoreRoute(_ context.Context, scope Scope, conversation, key string, in StoreRouteRequest) (StoreRouteCommand, bool, error) {
	f.found = true
	f.command = StoreRouteCommand{Scope: scope, ConversationID: conversation, IdempotencyKey: key, MemberID: in.MemberID, Content: in.Content, State: "READY_TO_DISPATCH", ModelDeadline: time.Now().Add(time.Minute)}
	if in.Unavailable {
		f.command.State = string(PlanningFailedBeforeDispatch)
	}
	return f.command, false, nil
}
func (f *routeStoreFixture) CompleteStoreRoute(context.Context, Scope, string, StoreRoute) (StoreRouteCommand, bool, error) {
	f.completes++
	f.command.State = "COMPLETE"
	return f.command, false, nil
}
func (f *routeStoreFixture) MarkStoreRouteLost(ctx context.Context, _ Scope, _ string, verify func(context.Context, StoreRouteCommand) error) (StoreRouteCommand, bool, error) {
	f.lostChecks++
	if err := verify(ctx, f.command); err != nil {
		return StoreRouteCommand{}, false, err
	}
	return StoreRouteCommand{}, false, ErrUnavailable
}

type routeGatewayFixture struct {
	admissionCalls, decisionCalls                        int
	fail, proofMissing, failProven, admissionUnavailable bool
}

func (f *routeGatewayFixture) Authorize(context.Context, Scope, string, string) error { return nil }
func (f *routeGatewayFixture) Admit(_ context.Context, _ Scope, _ string, member, content string) (StoreRouteRequest, error) {
	f.admissionCalls++
	if f.admissionUnavailable {
		return StoreRouteRequest{}, ErrUnavailable
	}
	return StoreRouteRequest{MemberID: member, Content: content}, nil
}

func TestUnavailableRouteAdmissionReturnsTerminalWithoutDispatch(t *testing.T) {
	store := &routeStoreFixture{}
	gateway := &routeGatewayFixture{admissionUnavailable: true}
	service := StoreRouteService{Store: store, Gateway: gateway}
	scope := Scope{OrganizationID: "org", ActorID: "actor"}
	result, err := service.Route(context.Background(), scope, "conversation", "key", "member", "question")
	require.NoError(t, err)
	require.Equal(t, string(PlanningFailedBeforeDispatch), result.State)
	require.Zero(t, gateway.decisionCalls)
	result, err = service.Route(context.Background(), scope, "conversation", "key", "member", "question")
	require.NoError(t, err)
	require.Equal(t, string(PlanningFailedBeforeDispatch), result.State)
	require.Equal(t, 1, gateway.admissionCalls)
}
func (f *routeGatewayFixture) Decide(context.Context, StoreRouteCommand) (StoreRoute, error) {
	f.decisionCalls++
	if f.fail {
		return StoreRoute{}, errors.New("unknown")
	}
	return StoreRoute{Kind: "STORE_OVERVIEW", Status: "ACTIVE"}, nil
}
func (f *routeGatewayFixture) VerifySettled(context.Context, StoreRouteCommand) error {
	return errors.New("not settled")
}
func (f *routeGatewayFixture) VerifyDecision(context.Context, StoreRouteCommand, StoreRoute) error {
	if f.proofMissing {
		return errors.New("unsettled or wrong output")
	}
	return nil
}
func (f *routeGatewayFixture) FailureState(context.Context, StoreRouteCommand) (PlanningState, error) {
	if f.failProven {
		return PlanningFailedBeforeDispatch, nil
	}
	return "", errors.New("unknown")
}
func (f *routeGatewayFixture) InvocationExists(context.Context, string) (bool, error) {
	return true, nil
}

func TestStoreRouteServiceReplaysOriginalAndDoesNotGuessUnknown(t *testing.T) {
	store := &routeStoreFixture{}
	gateway := &routeGatewayFixture{fail: true}
	service := StoreRouteService{Store: store, Gateway: gateway}
	scope := Scope{OrganizationID: "org", ActorID: "actor"}
	_, err := service.Route(context.Background(), scope, "conversation", "key", "member", "question")
	require.Error(t, err)
	require.Equal(t, 1, store.lostChecks)
	require.Zero(t, store.completes)
	_, err = service.Route(context.Background(), scope, "conversation", "key", "member", "question")
	require.Error(t, err)
	require.Equal(t, 1, gateway.admissionCalls, "retry must use the frozen original command")
	require.Equal(t, 2, gateway.decisionCalls)
	require.Zero(t, store.completes)
	gateway.fail = false
	gateway.proofMissing = true
	_, err = service.Route(context.Background(), scope, "conversation", "key", "member", "question")
	require.Error(t, err)
	require.Zero(t, store.completes)
	gateway.proofMissing = false
	command, err := service.Route(context.Background(), scope, "conversation", "key", "member", "question")
	require.NoError(t, err)
	require.Equal(t, "COMPLETE", command.State)
	_, err = service.Route(context.Background(), scope, "conversation", "key", "member", "question")
	require.NoError(t, err)
	require.Equal(t, 1, store.completes)
	require.Equal(t, 4, gateway.decisionCalls)
}

func TestStoreRouteServiceFinalizesProvenFailureOnly(t *testing.T) {
	store := &routeStoreFixture{}
	gateway := &routeGatewayFixture{fail: true, failProven: true}
	service := StoreRouteService{Store: store, Gateway: gateway}
	command, err := service.Route(context.Background(), Scope{OrganizationID: "org", ActorID: "actor"}, "conversation", "key", "member", "question")
	require.NoError(t, err)
	require.Equal(t, string(PlanningFailedBeforeDispatch), command.State)
	require.Zero(t, store.lostChecks)
	require.Equal(t, 1, gateway.decisionCalls)
	command, err = service.Route(context.Background(), Scope{OrganizationID: "org", ActorID: "actor"}, "conversation", "key", "member", "question")
	require.NoError(t, err)
	require.Equal(t, string(PlanningFailedBeforeDispatch), command.State)
	require.Equal(t, 1, gateway.decisionCalls)
}
