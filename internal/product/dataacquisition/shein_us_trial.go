package dataacquisition

import (
	"context"
	"errors"
	"math"
	"regexp"
	"time"

	"task-processor/internal/product/collection"
)

const (
	sheinUSTrialCreditLimit = 25
	sheinUSTrialTimeout     = 30 * time.Second
)

var sheinUSTrialRequestID = regexp.MustCompile(`^[A-Za-z0-9_-]{1,128}$`)

type SHEINUSTrialRequest struct {
	URL, Mode, Extract, ProxyCountry string
	CardLimit                        int
}

func sheinUSFirstPage() SHEINUSTrialRequest {
	return SHEINUSTrialRequest{
		URL:  "https://us.shein.com/Women-Dresses-c-1727.html",
		Mode: "auto", Extract: "auto", ProxyCountry: "us", CardLimit: 20,
	}
}

type SHEINUSTrialAction string

const (
	SHEINUSTrialView      SHEINUSTrialAction = "view"
	SHEINUSTrialSend      SHEINUSTrialAction = "send"
	SHEINUSTrialReconcile SHEINUSTrialAction = "reconcile"
)

type SHEINUSTrialApproval struct {
	ID, ProjectID, AccountRef, KeyRef string
	Scope                             collection.Scope
	ExpiresAt                         time.Time
}

func (a SHEINUSTrialApproval) Binding() string {
	a.ExpiresAt = a.ExpiresAt.UTC()
	return collection.Digest(struct {
		Contract string
		Approval SHEINUSTrialApproval
		Request  SHEINUSTrialRequest
	}{"shein-us-category-page@v1", a, sheinUSFirstPage()})
}

type SHEINUSTrialBudget struct {
	Binding                                 string
	AvailableCredits, MaximumRequestCredits int
	ZeroIncrementalCash, ExclusiveKeyWindow bool
}

// The authority supplies the frozen approval, current project/source permissions
// and account evidence. The key must be dedicated to this approval's time window
// with no other requests in flight.
type SHEINUSTrialAuthority interface {
	AuthorizeSHEINUSTrial(context.Context, Principal, string, SHEINUSTrialAction) (SHEINUSTrialApproval, error)
	CheckSHEINUSTrialBudget(context.Context, SHEINUSTrialApproval) (SHEINUSTrialBudget, error)
}

type SHEINUSTrialUsage struct {
	ApprovalID, Binding, RequestID string
	Credits                        *int // nil until the original request's final usage is verified
}

type SHEINUSTrialAttempt struct {
	SHEINUSTrialUsage
	ClaimedAt time.Time
}

func (a SHEINUSTrialAttempt) ReservedCredits() int {
	if a.ClaimedAt.IsZero() || a.Credits != nil {
		return 0
	}
	return sheinUSTrialCreditLimit
}

func (a SHEINUSTrialAttempt) OverLimit() bool {
	return a.Credits != nil && *a.Credits > sheinUSTrialCreditLimit
}

// Submit sends at most one request. A partial response may identify the request
// even when the final usage is still unknown.
type SHEINUSTrialSender func(context.Context, SHEINUSTrialApproval, SHEINUSTrialRequest) (SHEINUSTrialUsage, error)

// Lookup must prove a single original request, using its ID if known or the
// dedicated key, fixed URL and approval window. Absence is never proof of zero.
type SHEINUSTrialLookup func(context.Context, SHEINUSTrialApproval, SHEINUSTrialRequest, SHEINUSTrialAttempt) (SHEINUSTrialUsage, error)

type SHEINUSTrialStore interface {
	Read(context.Context, SHEINUSTrialApproval) (SHEINUSTrialAttempt, error)
	Claim(context.Context, SHEINUSTrialApproval) (SHEINUSTrialAttempt, bool, error)
	Record(context.Context, SHEINUSTrialUsage) (SHEINUSTrialAttempt, error)
}

type SHEINUSTrial struct {
	store     SHEINUSTrialStore
	authority SHEINUSTrialAuthority
	submit    SHEINUSTrialSender
	lookup    SHEINUSTrialLookup
}

func NewSHEINUSTrial(store SHEINUSTrialStore, authority SHEINUSTrialAuthority, submit SHEINUSTrialSender, lookup SHEINUSTrialLookup) (*SHEINUSTrial, error) {
	if store == nil || authority == nil || submit == nil || lookup == nil {
		return nil, ErrUnavailable
	}
	return &SHEINUSTrial{store: store, authority: authority, submit: submit, lookup: lookup}, nil
}

func (s *SHEINUSTrial) authorize(ctx context.Context, p Principal, id string, action SHEINUSTrialAction) (SHEINUSTrialApproval, error) {
	if ctx.Err() != nil || p.Scope.Validate() != nil || p.CredentialID != "" || p.CredentialRevision != 0 || !collection.ValidID(id) {
		return SHEINUSTrialApproval{}, ErrForbidden
	}
	a, err := s.authority.AuthorizeSHEINUSTrial(ctx, p, id, action)
	if err != nil {
		return SHEINUSTrialApproval{}, ErrForbidden
	}
	if ctx.Err() != nil || a.ID != id || a.Scope.Validate() != nil || !collection.ValidID(a.ProjectID) ||
		!collection.ValidID(a.AccountRef) || !collection.ValidID(a.KeyRef) || a.ExpiresAt.IsZero() ||
		a.Scope.OrganizationID != p.Scope.OrganizationID || (action != SHEINUSTrialReconcile && a.Scope != p.Scope) {
		return SHEINUSTrialApproval{}, ErrForbidden
	}
	return a, nil
}

func (s *SHEINUSTrial) checkSend(ctx context.Context, p Principal, approved SHEINUSTrialApproval) error {
	a, err := s.authorize(ctx, p, approved.ID, SHEINUSTrialSend)
	if err != nil {
		return err
	}
	binding := approved.Binding()
	if a.Binding() != binding || !time.Now().Before(a.ExpiresAt) {
		return ErrForbidden
	}
	budget, err := s.authority.CheckSHEINUSTrialBudget(ctx, a)
	if err != nil || budget.Binding != binding || budget.AvailableCredits < sheinUSTrialCreditLimit ||
		budget.MaximumRequestCredits < 1 || budget.MaximumRequestCredits > sheinUSTrialCreditLimit ||
		!budget.ZeroIncrementalCash || !budget.ExclusiveKeyWindow {
		return ErrUnavailable
	}
	return ctx.Err()
}

func (s *SHEINUSTrial) stillAuthorized(ctx context.Context, p Principal, a SHEINUSTrialApproval, action SHEINUSTrialAction) bool {
	current, err := s.authorize(ctx, p, a.ID, action)
	return err == nil && current.Binding() == a.Binding()
}

func (s *SHEINUSTrial) visible(ctx context.Context, p Principal, a SHEINUSTrialApproval, action SHEINUSTrialAction, attempt SHEINUSTrialAttempt, err error) (SHEINUSTrialAttempt, error) {
	if !s.stillAuthorized(ctx, p, a, action) {
		return SHEINUSTrialAttempt{}, ErrForbidden
	}
	return attempt, err
}

func (s *SHEINUSTrial) Send(ctx context.Context, p Principal, id string) (SHEINUSTrialAttempt, error) {
	ctx, cancel := context.WithTimeout(ctx, sheinUSTrialTimeout)
	defer cancel()
	a, err := s.authorize(ctx, p, id, SHEINUSTrialView)
	if err != nil {
		return SHEINUSTrialAttempt{}, err
	}
	original, err := s.store.Read(ctx, a)
	if err == nil {
		attempt, outcomeErr := sheinUSTrialOutcome(original)
		return s.visible(ctx, p, a, SHEINUSTrialView, attempt, outcomeErr)
	}
	if !errors.Is(err, ErrNotFound) {
		return SHEINUSTrialAttempt{}, err
	}

	sendCtx, cancelSend := context.WithDeadline(ctx, a.ExpiresAt)
	defer cancelSend()
	if err := s.checkSend(sendCtx, p, a); err != nil {
		return SHEINUSTrialAttempt{}, err
	}
	original, won, err := s.store.Claim(sendCtx, a)
	if err != nil {
		return SHEINUSTrialAttempt{}, err
	}
	if !won {
		attempt, outcomeErr := sheinUSTrialOutcome(original)
		return s.visible(ctx, p, a, SHEINUSTrialView, attempt, outcomeErr)
	}
	if err := s.checkSend(sendCtx, p, a); err != nil {
		return s.visible(ctx, p, a, SHEINUSTrialView, original, ErrUnknown)
	}
	usage, err := s.submit(sendCtx, a, sheinUSFirstPage())
	late := sendCtx.Err() != nil

	// Keep the provider request ID after cancellation, without trusting a late
	// or unauthorized result as final usage.
	writeCtx, cancelWrite := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
	defer cancelWrite()
	canView := s.stillAuthorized(writeCtx, p, a, SHEINUSTrialView)
	if err != nil || late || !canView {
		usage.Credits = nil
	}
	attempt, err := s.record(writeCtx, original, usage)
	if !canView {
		return SHEINUSTrialAttempt{}, ErrForbidden
	}
	return s.visible(writeCtx, p, a, SHEINUSTrialView, attempt, err)
}

func (s *SHEINUSTrial) Reconcile(ctx context.Context, p Principal, id string) (SHEINUSTrialAttempt, error) {
	ctx, cancel := context.WithTimeout(ctx, sheinUSTrialTimeout)
	defer cancel()
	a, err := s.authorize(ctx, p, id, SHEINUSTrialReconcile)
	if err != nil {
		return SHEINUSTrialAttempt{}, err
	}
	original, err := s.store.Read(ctx, a)
	if err != nil {
		return SHEINUSTrialAttempt{}, err
	}
	if original.Credits != nil {
		return s.visible(ctx, p, a, SHEINUSTrialReconcile, original, nil)
	}
	usage, err := s.lookup(ctx, a, sheinUSFirstPage(), original)
	if !s.stillAuthorized(ctx, p, a, SHEINUSTrialReconcile) {
		return SHEINUSTrialAttempt{}, ErrForbidden
	}
	if err != nil {
		return original, ErrUnknown
	}
	attempt, err := s.record(ctx, original, usage)
	return s.visible(ctx, p, a, SHEINUSTrialReconcile, attempt, err)
}

func (s *SHEINUSTrial) record(ctx context.Context, original SHEINUSTrialAttempt, usage SHEINUSTrialUsage) (SHEINUSTrialAttempt, error) {
	if !sheinUSTrialRequestID.MatchString(usage.RequestID) {
		return original, ErrUnknown
	}
	if usage.ApprovalID != original.ApprovalID || usage.Binding != original.Binding ||
		(original.RequestID != "" && original.RequestID != usage.RequestID) {
		return original, ErrConflict
	}
	if usage.Credits != nil && (*usage.Credits < 0 || *usage.Credits > math.MaxInt32) {
		usage.Credits = nil
	}
	saved, err := s.store.Record(ctx, usage)
	if err != nil {
		return original, err
	}
	return sheinUSTrialOutcome(saved)
}

func sheinUSTrialOutcome(a SHEINUSTrialAttempt) (SHEINUSTrialAttempt, error) {
	if a.Credits == nil {
		return a, ErrUnknown
	}
	return a, nil
}
