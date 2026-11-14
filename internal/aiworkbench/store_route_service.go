package aiworkbench

import (
	"context"
	"time"
	"unicode/utf8"
)

type StoreRouteRepository interface {
	ReplayStoreRoute(context.Context, Scope, string, string, string, string) (StoreRouteCommand, bool, error)
	BeginStoreRoute(context.Context, Scope, string, string, StoreRouteRequest) (StoreRouteCommand, bool, error)
	CompleteStoreRoute(context.Context, Scope, string, StoreRoute) (StoreRouteCommand, bool, error)
	MarkStoreRouteLost(context.Context, Scope, string, func(context.Context, StoreRouteCommand) error) (StoreRouteCommand, bool, error)
	FinalizeStoreRouteFailure(context.Context, Scope, string, func(context.Context, StoreRouteCommand) (PlanningState, error)) (StoreRouteCommand, bool, error)
	FinalizeStoreRouteAbsent(context.Context, Scope, string, func(context.Context, string) (bool, error)) (StoreRouteCommand, bool, error)
}

// The gateway must use the existing governed invocation owner to claim the
// frozen invocation before sending; replay may call Decide concurrently.
type StoreRouteGateway interface {
	Authorize(context.Context, Scope, string, string) error
	Admit(context.Context, Scope, string, string, string) (StoreRouteRequest, error)
	Decide(context.Context, StoreRouteCommand) (StoreRoute, error)
	VerifySettled(context.Context, StoreRouteCommand) error
	VerifyDecision(context.Context, StoreRouteCommand, StoreRoute) error
	FailureState(context.Context, StoreRouteCommand) (PlanningState, error)
	InvocationExists(context.Context, string) (bool, error)
}

type StoreRouteService struct {
	Store   StoreRouteRepository
	Gateway StoreRouteGateway
}

func (s *StoreRouteService) Route(ctx context.Context, scope Scope, conversationID, key, member, content string) (StoreRouteCommand, error) {
	if s == nil || s.Store == nil || s.Gateway == nil || ctx == nil {
		return StoreRouteCommand{}, ErrUnavailable
	}
	if content == "" || len(content) > 8192 || !utf8.ValidString(content) {
		return StoreRouteCommand{}, ErrInvalid
	}
	if err := s.Gateway.Authorize(ctx, scope, conversationID, member); err != nil {
		return StoreRouteCommand{}, err
	}
	command, found, err := s.Store.ReplayStoreRoute(ctx, scope, conversationID, key, member, content)
	if err != nil {
		return StoreRouteCommand{}, err
	}
	if !found {
		request, admissionErr := s.Gateway.Admit(ctx, scope, conversationID, member, content)
		if admissionErr != nil {
			// Authentication/authorization failures never become a receipt.
			if err = s.Gateway.Authorize(ctx, scope, conversationID, member); err != nil {
				return StoreRouteCommand{}, err
			}
			request = StoreRouteRequest{Content: content, MemberID: member, Unavailable: true}
		}
		if request.MemberID != member || request.Content != content {
			return StoreRouteCommand{}, ErrInvalid
		}
		if _, _, err = s.Store.BeginStoreRoute(ctx, scope, conversationID, key, request); err != nil {
			return StoreRouteCommand{}, err
		}
		command, found, err = s.Store.ReplayStoreRoute(ctx, scope, conversationID, key, member, content)
		if err != nil {
			return StoreRouteCommand{}, err
		}
		if !found {
			return StoreRouteCommand{}, ErrUnavailable
		}
	}
	if command.State == string(PlanningComplete) || command.State == "ROUTING_RESULT_LOST" || command.State == string(PlanningFailedBeforeDispatch) || command.State == string(PlanningInvalidOutput) {
		if err = s.Gateway.Authorize(ctx, scope, conversationID, member); err != nil {
			return StoreRouteCommand{}, err
		}
		return command, nil
	}
	if command.State != string(PlanningReadyToDispatch) || command.ModelDeadline.IsZero() {
		return StoreRouteCommand{}, ErrUnavailable
	}
	callCtx, cancel := context.WithDeadline(ctx, command.ModelDeadline)
	decision, callErr := s.Gateway.Decide(callCtx, command)
	cancel()
	// Completion uses a bounded context even if the requesting browser went
	// away. The store enforces the database-clock fence under its row lock.
	finishCtx, finish := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
	defer finish()
	if callErr != nil {
		failed, _, failureErr := s.Store.FinalizeStoreRouteFailure(finishCtx, scope, key, s.Gateway.FailureState)
		if failureErr == nil {
			if err = s.Gateway.Authorize(finishCtx, scope, conversationID, member); err != nil {
				return StoreRouteCommand{}, err
			}
			return failed, nil
		}
		lost, _, recoveryErr := s.Store.MarkStoreRouteLost(finishCtx, scope, key, s.Gateway.VerifySettled)
		if recoveryErr == nil {
			if err = s.Gateway.Authorize(finishCtx, scope, conversationID, member); err != nil {
				return StoreRouteCommand{}, err
			}
			return lost, nil
		}
		absent, _, absentErr := s.Store.FinalizeStoreRouteAbsent(finishCtx, scope, key, s.Gateway.InvocationExists)
		if absentErr == nil {
			if err = s.Gateway.Authorize(finishCtx, scope, conversationID, member); err != nil {
				return StoreRouteCommand{}, err
			}
			return absent, nil
		}
		return command, callErr
	}
	if err = s.Gateway.VerifyDecision(finishCtx, command, decision); err != nil {
		return command, err
	}
	completed, _, err := s.Store.CompleteStoreRoute(finishCtx, scope, key, decision)
	if err != nil {
		return StoreRouteCommand{}, err
	}
	if err = s.Gateway.Authorize(finishCtx, scope, conversationID, member); err != nil {
		return StoreRouteCommand{}, err
	}
	return completed, nil
}
