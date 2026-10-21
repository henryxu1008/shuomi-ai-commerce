package sheinlogin

import (
	"context"
	"sync"
	"time"

	"github.com/google/uuid"
	"task-processor/internal/storecenter"
)

// RunWorker is the only browser execution entry. Producers never call it.
func (s *Service) RunWorker(ctx context.Context, workerID string) error {
	if workerID == "" {
		workerID = uuid.NewString()
	}
	if e := s.store.EnsureGroup(ctx); e != nil {
		return e
	}
	ctx, cancelWorker := context.WithCancel(ctx)
	sem := make(chan struct{}, s.config.MaxConcurrentLogins)
	var wg sync.WaitGroup
	var running sync.Map
	defer func() { cancelWorker(); wg.Wait() }()
	for ctx.Err() == nil {
		messages, e := s.store.ReadJobs(ctx, workerID)
		if e != nil {
			if ctx.Err() != nil {
				return nil
			}
			return e
		}
		for _, message := range messages {
			id, _ := message.Values["attempt_id"].(string)
			if !validUUID(id) {
				_ = s.store.Ack(ctx, message.ID)
				continue
			}
			select {
			case sem <- struct{}{}:
			case <-ctx.Done():
				return nil
			}
			running.Store(id, true)
			wg.Add(1)
			go func(attemptID, messageID string) {
				defer wg.Done()
				defer func() { <-sem; running.Delete(attemptID) }()
				_ = s.processAttempt(ctx, attemptID)
				if final, e := s.store.Load(context.WithoutCancel(ctx), attemptID); e == nil && !final.Status.IsActive() && final.SyncStatus != "pending" && final.SyncStatus != "unavailable" {
					_ = s.store.Ack(context.WithoutCancel(ctx), messageID)
				}
			}(id, message.ID)
		}
		stale, e := s.store.Stale(ctx, workerID)
		if e != nil && ctx.Err() == nil {
			return ErrUnavailable
		}
		for _, message := range stale {
			id, _ := message.Values["attempt_id"].(string)
			if _, ok := running.Load(id); ok {
				continue
			}
			if ack, _ := s.handleReclaimedAttempt(ctx, id); ack {
				_ = s.store.Ack(ctx, message.ID)
			}
		}
	}
	return nil
}

func (s *Service) handleReclaimedAttempt(ctx context.Context, id string) (bool, error) {
	if !validUUID(id) {
		return true, nil
	}
	a, err := s.store.Load(ctx, id)
	if err != nil {
		return false, err
	}
	if a.Status == LoginAttemptSucceeded && (a.SyncStatus == "pending" || a.SyncStatus == "unavailable") {
		err = s.resumePending(ctx, *a)
		return err == nil, err
	}
	if a.Status.IsActive() {
		b, err := s.owner.ReadBinding(executionContext(ctx, *a), a.Scope, a.ActorID, a.MemberID)
		if err != nil {
			return false, err
		}
		// Stream idle time cannot distinguish a live browser from a crashed owner.
		// Keep its state AND delivery pending until exact closure is established.
		if b.Generation == a.Scope.Generation && b.ActiveAttemptID == a.ID && !b.ClosedConfirmed {
			return false, nil
		}
		err = s.complete(ctx, *a, LoginAttemptInterrupted, "WORKER_INTERRUPTED")
		return err == nil, err
	}
	return true, nil
}
func (s *Service) processAttempt(parent context.Context, id string) error {
	a, e := s.store.Load(parent, id)
	if e != nil {
		return e
	}
	if !a.Status.IsActive() {
		return nil
	}
	parent = executionContext(parent, *a)
	binding, e := s.config.Binding(a.Scope.OrganizationID, a.Scope.StoreID)
	if e != nil {
		return e
	}
	scope := s.config.Scope(binding)
	scope.Generation = a.Scope.Generation
	if scope != a.Scope {
		return ErrInvalidInput
	}
	nonce := uuid.NewString()
	if e = s.owner.Claim(parent, a.Scope, a.ActorID, a.MemberID, a.ID, nonce); e != nil {
		_ = s.complete(parent, *a, LoginAttemptFailed, safeCode(e))
		return e
	}
	// Active ownership survives every error until this exact process has finished
	// all browser/verification/HTTP work and can synchronously confirm closure.
	ctx, cancel := context.WithTimeout(parent, 12*time.Minute)
	var session VerifySession
	closed := true
	defer func() {
		cancel()
		if session != nil && session.Close() != nil {
			closed = false
		}
		if closed {
			closeCtx, done := context.WithTimeout(context.Background(), 10*time.Second)
			defer done()
			_ = s.owner.ConfirmClosed(closeCtx, a.Scope, a.ID, nonce)
		}
	}()
	next := *a
	next.OwnerNonce = nonce
	next.Status = LoginAttemptLaunching
	if e = s.store.CompareAndSwap(ctx, *a, next, nil); e != nil {
		return e
	}
	*a = next
	identity := *a
	monitorCtx, stopMonitor := context.WithCancel(ctx)
	monitorDone := make(chan struct{})
	go func() {
		defer close(monitorDone)
		ticker := time.NewTicker(300 * time.Millisecond)
		defer ticker.Stop()
		authorization := time.NewTicker(15 * time.Second)
		defer authorization.Stop()
		for {
			select {
			case <-monitorCtx.Done():
				return
			case <-ticker.C:
				current, e := s.store.Load(monitorCtx, identity.ID)
				if e != nil || current.Status == LoginAttemptCancelled || current.Status == LoginAttemptInterrupted {
					cancel()
					return
				}
			case <-authorization.C:
				if s.owner.CheckOwner(monitorCtx, scope, identity.ActorID, identity.MemberID, identity.ID, nonce) != nil {
					cancel()
					return
				}
			}
		}
	}()
	defer func() { stopMonitor(); <-monitorDone }()
	check := func() error { return s.owner.CheckOwner(ctx, scope, a.ActorID, a.MemberID, a.ID, nonce) }
	var state map[string]any
	if !a.ForceLogin {
		var ok bool
		state, ok, e = s.store.Cookie(ctx, scope)
		if e != nil {
			return e
		}
		if !ok {
			state = nil
		}
	}
	var account Account
	if state == nil {
		account, e = binding.LoadAccount(ctx, scope)
	} else {
		account, e = binding.SessionAccount(ctx, scope)
	}
	if e != nil {
		_ = s.complete(context.WithoutCancel(ctx), *a, LoginAttemptFailed, "SESSION_INPUT_UNAVAILABLE")
		return e
	}
	if state == nil {
		if e = check(); e != nil {
			return e
		}
		freshProfile := ""
		if a.ForceLogin {
			freshProfile = a.ID
		}
		result, owned, e := s.automation.StartLogin(ctx, account, AutomationConfig{FreshProfileID: freshProfile, Headless: binding.Headless, ProfileRoot: s.config.ProfileRoot, ArtifactDir: s.config.ArtifactRoot, BrowserPath: s.config.BrowserPath, DriverDirectory: s.config.DriverDirectory, ViewportWidth: s.config.ViewportWidth, ViewportHeight: s.config.ViewportHeight})
		session = owned
		if e == nil && result != nil && result.WaitingForVerifyCode {
			next = *a
			next.Status = LoginAttemptWaitingVerifyCode
			if e = s.store.CompareAndSwap(ctx, *a, next, nil); e == nil {
				*a = next
				result, e = s.waitVerification(ctx, *a, session, check)
			}
		}
		if e != nil || result == nil || cookieCount(result.BrowserState) == 0 {
			if current, loadErr := s.store.Load(context.WithoutCancel(ctx), a.ID); loadErr == nil && current.Status.IsActive() {
				status, code := LoginAttemptFailed, "LOGIN_FAILED"
				if ctx.Err() != nil {
					status = LoginAttemptInterrupted
					code = "EXECUTION_INTERRUPTED"
				}
				_ = s.complete(context.WithoutCancel(ctx), *current, status, code)
			}
			return ErrUnavailable
		}
		state = result.BrowserState
	}
	if e = check(); e != nil {
		return e
	}
	next = *a
	next.Status = LoginAttemptSucceeded
	next.SyncStatus = "pending"
	now := time.Now().UTC()
	next.CompletedAt = &now
	if e = s.store.CompareAndSwap(ctx, *a, next, state); e != nil {
		return e
	}
	*a = next
	return s.syncPending(ctx, a, account, state, check)
}

// Only a prior confirmed-close permits another executor to resume the original
// bounded pending sync. A crashed/UNKNOWN browser owner remains fenced. This
// path cannot launch a browser or persist/refresh a cookie.
func (s *Service) resumePending(parent context.Context, a LoginAttempt) error {
	parent = executionContext(parent, a)
	binding, e := s.config.Binding(a.Scope.OrganizationID, a.Scope.StoreID)
	if e != nil {
		return e
	}
	scope := s.config.Scope(binding)
	scope.Generation = a.Scope.Generation
	if scope != a.Scope {
		return ErrInvalidInput
	}
	nonce := uuid.NewString()
	if e = s.owner.Claim(parent, scope, a.ActorID, a.MemberID, a.ID, nonce); e != nil {
		return e
	}
	defer func() {
		closeCtx, done := context.WithTimeout(context.Background(), 10*time.Second)
		defer done()
		_ = s.owner.ConfirmClosed(closeCtx, scope, a.ID, nonce)
	}()
	ctx, cancel := context.WithTimeout(parent, 45*time.Second)
	defer cancel()
	next := a
	next.OwnerNonce = nonce
	next.SyncStatus = "pending"
	if e = s.store.CompareAndSwap(ctx, a, next, nil); e != nil {
		return e
	}
	a = next
	var state map[string]any
	var account Account
	if a.MerchantID == "" {
		var ok bool
		state, ok, e = s.store.Cookie(ctx, scope)
		if e != nil || !ok {
			return ErrUnavailable
		}
		// An identity read needs endpoint/proxy metadata, never a password.
		account, e = binding.SessionAccount(ctx, scope)
		if e != nil {
			return e
		}
	}
	check := func() error { return s.owner.CheckOwner(ctx, scope, a.ActorID, a.MemberID, a.ID, nonce) }
	return s.syncPending(ctx, &a, account, state, check)
}
func (s *Service) syncPending(ctx context.Context, a *LoginAttempt, account Account, state map[string]any, check func() error) error {
	scope := a.Scope
	nonce := a.OwnerNonce
	next := *a
	var e error
	if e = check(); e != nil {
		return e
	}
	merchant := a.MerchantID
	if merchant == "" {
		merchant, e = s.identityReader.Merchant(ctx, account, state)
	}
	if e != nil {
		next = *a
		next.SyncStatus = "unavailable"
		next.ErrorCode = "IDENTITY_READ_UNAVAILABLE"
		if err := s.store.CompareAndSwap(context.WithoutCancel(ctx), *a, next, nil); err != nil {
			return err
		}
		if next.SyncStatus == "unavailable" {
			return ErrUnavailable
		}
		return nil
	}
	if a.MerchantID == "" {
		st, e := s.owner.ReadStore(ctx, scope, a.ActorID, a.MemberID)
		if e != nil {
			return e
		}
		next = *a
		next.MerchantID = merchant
		next.ExpectedStoreVersion = st.Version
		if e = s.store.CompareAndSwap(ctx, *a, next, nil); e != nil {
			return e
		}
		*a = next
	}
	if e = check(); e != nil {
		return e
	}
	receipt, e := s.owner.RecordSheinLoginIdentity(ctx, storecenter.SheinIdentityCommand{Scope: scope, ActorID: a.ActorID, MemberID: a.MemberID, AttemptID: a.ID, OwnerNonce: nonce, ExpectedStoreVersion: a.ExpectedStoreVersion, MerchantID: a.MerchantID})
	next = *a
	if e != nil {
		next.SyncStatus = "unavailable"
		next.ErrorCode = "STORE_SYNC_UNAVAILABLE"
	} else {
		next.SyncStatus = receipt.Status
		next.ErrorCode = ""
		if receipt.Status == "blocked" {
			next.ErrorCode = "SYNC_BLOCKED"
		}
	}
	// A response-loss replay keeps exactly the fixed payload; it never logs in or
	// derives a new merchant/version. The Store receipt is the durable authority.
	if e != nil && ctx.Err() == nil {
		receipt, replayErr := s.owner.RecordSheinLoginIdentity(ctx, storecenter.SheinIdentityCommand{Scope: scope, ActorID: a.ActorID, MemberID: a.MemberID, AttemptID: a.ID, OwnerNonce: nonce, ExpectedStoreVersion: a.ExpectedStoreVersion, MerchantID: a.MerchantID})
		if replayErr == nil {
			next.SyncStatus = receipt.Status
			next.ErrorCode = ""
			if receipt.Status == "blocked" {
				next.ErrorCode = "SYNC_BLOCKED"
			}
		}
	}
	if err := s.store.CompareAndSwap(context.WithoutCancel(ctx), *a, next, nil); err != nil {
		return err
	}
	if next.SyncStatus == "unavailable" {
		return ErrUnavailable
	}
	return nil
}

func (s *Service) waitVerification(ctx context.Context, a LoginAttempt, session VerifySession, check func() error) (*AutomationResult, error) {
	if session == nil {
		return nil, ErrUnavailable
	}
	waitCtx, cancel := context.WithTimeout(ctx, 10*time.Minute)
	defer cancel()
	type watched struct {
		result *AutomationResult
		err    error
	}
	results := make(chan watched, 1)
	done := make(chan struct{})
	if watcher, ok := session.(VerifySessionLoginWatcher); ok {
		go func() { defer close(done); r, e := watcher.WaitForLogin(waitCtx); results <- watched{r, e} }()
	} else {
		close(done)
	}
	defer func() { cancel(); <-done }()
	ticker := time.NewTicker(250 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-waitCtx.Done():
			return nil, waitCtx.Err()
		case v := <-results:
			if v.err == nil && v.result != nil && cookieCount(v.result.BrowserState) > 0 {
				return v.result, nil
			}
		case <-ticker.C:
			code, e := s.store.ConsumeCode(waitCtx, a.ID)
			if e != nil {
				return nil, e
			}
			if code == "" {
				continue
			}
			if e = check(); e != nil {
				return nil, e
			}
			result, e := session.SubmitCode(waitCtx, code)
			if e != nil {
				return nil, e
			}
			if result != nil && cookieCount(result.BrowserState) > 0 {
				return result, nil
			}
		}
	}
}
