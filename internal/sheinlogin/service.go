package sheinlogin

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"reflect"
	"time"

	"task-processor/internal/core/logger"
	"task-processor/internal/storecenter"
)

var sheinLoginServiceLogger = logger.GetGlobalLogger("sheinlogin_service")

type StoreOwner interface {
	Activate(context.Context, storecenter.SheinBindingCommand) (storecenter.SheinBinding, error)
	Quiesce(context.Context, storecenter.SheinBindingCommand) (storecenter.SheinBinding, error)
	ReadBinding(context.Context, storecenter.SheinLoginScope, string, string) (storecenter.SheinBinding, error)
	ReadStore(context.Context, storecenter.SheinLoginScope, string, string) (storecenter.StoreSnapshot, error)
	Claim(context.Context, storecenter.SheinLoginScope, string, string, string, string) error
	CheckOwner(context.Context, storecenter.SheinLoginScope, string, string, string, string) error
	ConfirmClosed(context.Context, storecenter.SheinLoginScope, string, string) error
	RecordSheinLoginIdentity(context.Context, storecenter.SheinIdentityCommand) (storecenter.SheinIdentityReceipt, error)
}
type SessionIdentityReader interface {
	Merchant(context.Context, Account, map[string]any) (string, error)
	Warehouses(context.Context, Account, map[string]any) ([]WarehouseOption, error)
}
type Service struct {
	config         Config
	store          *RedisStore
	owner          StoreOwner
	automation     Automation
	identityReader SessionIdentityReader
}

func NewService(c Config, redis *RedisStore, owner StoreOwner) (*Service, error) {
	if c.Validate() != nil || redis == nil || owner == nil {
		return nil, ErrInvalidInput
	}
	return &Service{config: c, store: redis, owner: owner, automation: NewPlaywrightAutomation(), identityReader: NewSessionIdentityReader()}, nil
}
func (s *Service) Close() error { return s.store.Close() }
func (s *Service) RedisReady(ctx context.Context) bool {
	return s != nil && s.store != nil && s.store.Ready(ctx)
}

type Command struct {
	Action                                                            string
	OrganizationID, StoreID, ActorID, MemberID, RequestKey, AttemptID string
	ForceLogin                                                        bool
	Code                                                              string
	ExpectedGeneration                                                int64
}

func (s *Service) Execute(ctx context.Context, c Command) (Result, error) {
	if !validOpaque(c.OrganizationID) || !validUUID(c.StoreID) || !validOpaque(c.ActorID) || !validOpaque(c.MemberID) {
		return Result{}, ErrInvalidInput
	}
	b, e := s.config.Binding(c.OrganizationID, c.StoreID)
	if e != nil {
		return Result{}, e
	}
	scope := s.config.Scope(b)
	if c.Action == "activate" {
		if !validUUID(c.RequestKey) || c.AttemptID != "" || c.ForceLogin || c.Code != "" {
			return Result{}, ErrInvalidInput
		}
		binding, e := s.owner.Activate(ctx, storecenter.SheinBindingCommand{Scope: scope, ActorID: c.ActorID, MemberID: c.MemberID, RequestKey: c.RequestKey, ExpectedGeneration: c.ExpectedGeneration})
		if e != nil {
			return Result{}, e
		}
		return Result{OrganizationID: c.OrganizationID, StoreID: c.StoreID, Status: binding.State, Generation: binding.Generation}, nil
	}
	if c.Action == "status" || c.Action == "verify" || c.Action == "cancel" {
		if c.RequestKey != "" || !validUUID(c.AttemptID) || c.ForceLogin {
			return Result{}, ErrInvalidInput
		}
		// Read authorization is live but status does not change an attempt, launch a
		// browser, refresh cookies or trigger Store synchronization.
		if _, e = s.owner.ReadStore(ctx, scope, c.ActorID, c.MemberID); e != nil {
			return Result{}, e
		}
		a, e := s.store.Load(ctx, c.AttemptID)
		if e != nil {
			return Result{}, e
		}
		expected := scope
		expected.Generation = a.Scope.Generation
		if !reflect.DeepEqual(expected, a.Scope) || a.ActorID != c.ActorID || a.MemberID != c.MemberID {
			return Result{}, ErrInvalidInput
		}
		if c.Action != "status" {
			current, e := s.owner.ReadBinding(ctx, scope, c.ActorID, c.MemberID)
			if e != nil || current.State != "active" || current.Generation != a.Scope.Generation {
				return Result{}, storecenter.ErrSheinLoginFenced
			}
			if c.Action == "verify" {
				e = s.store.SubmitCode(ctx, *a, c.Code)
			} else {
				e = s.store.Cancel(ctx, *a)
			}
			if e != nil {
				return Result{}, e
			}
			a, e = s.store.Load(ctx, a.ID)
			if e != nil {
				return Result{}, e
			}
		}
		return a.Result(), nil
	}
	current, e := s.owner.ReadBinding(ctx, scope, c.ActorID, c.MemberID)
	if e != nil {
		return Result{}, e
	}
	scope.Generation = current.Generation
	if c.Action == "quiesce" {
		if !validUUID(c.RequestKey) || c.AttemptID != "" || c.ForceLogin || c.Code != "" {
			return Result{}, ErrInvalidInput
		}
		result, e := s.owner.Quiesce(ctx, storecenter.SheinBindingCommand{Scope: scope, ActorID: c.ActorID, MemberID: c.MemberID, RequestKey: c.RequestKey, ExpectedGeneration: c.ExpectedGeneration})
		if e != nil {
			return Result{}, e
		}
		return Result{OrganizationID: c.OrganizationID, StoreID: c.StoreID, Status: result.State, Generation: result.Generation}, nil
	}
	if c.Action != "login" || !validUUID(c.RequestKey) || c.AttemptID != "" || c.Code != "" || current.State != "active" {
		return Result{}, ErrInvalidInput
	}
	a, e := s.store.Enqueue(ctx, LoginAttempt{ID: c.RequestKey, Scope: scope, ActorID: c.ActorID, MemberID: c.MemberID, ForceLogin: c.ForceLogin, CreatedAt: time.Now().UTC()})
	if e != nil {
		return Result{}, e
	}
	return a.Result(), nil
}
func (s *Service) complete(ctx context.Context, a LoginAttempt, status LoginAttemptStatus, code string) error {
	next := a
	next.Status = status
	next.ErrorCode = code
	now := time.Now().UTC()
	next.CompletedAt = &now
	return s.store.CompareAndSwap(ctx, a, next, nil)
}
func safeCode(e error) string {
	if errors.Is(e, storecenter.ErrSheinLoginFenced) {
		return "EXECUTION_FENCED"
	}
	if errors.Is(e, storecenter.ErrIdempotencyConflict) {
		return "PAYLOAD_CONFLICT"
	}
	return "DEPENDENCY_UNAVAILABLE"
}
func scopeFingerprint(scope storecenter.SheinLoginScope) string {
	h := sha256.Sum256([]byte(scope.OrganizationID + "\x00" + scope.StoreID + "\x00" + scope.BindingRevision))
	return hex.EncodeToString(h[:])
}

func PublicErrorCode(err error) string {
	if errors.Is(err, ErrInvalidInput) || errors.Is(err, storecenter.ErrNotFound) {
		return "NOT_ALLOWED_OR_INVALID"
	}
	if errors.Is(err, storecenter.ErrIdempotencyConflict) {
		return "PAYLOAD_CONFLICT"
	}
	if errors.Is(err, storecenter.ErrSheinLoginFenced) {
		return "EXECUTION_FENCED"
	}
	if errors.Is(err, storecenter.ErrVersionConflict) {
		return "STATE_CHANGED"
	}
	return "DEPENDENCY_UNAVAILABLE"
}
