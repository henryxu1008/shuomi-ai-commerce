package dataservicesapp

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	jobstore "task-processor/internal/integration/persistence/product/dataacquisition"
	"task-processor/internal/ledger/orgresource"
	"task-processor/internal/product/collection"
	"task-processor/internal/product/dataacquisition"
)

type trialAuthority struct {
	revoked  *dataacquisition.SHEINUSTrialAction
	approval dataacquisition.SHEINUSTrialApproval
	budget   dataacquisition.SHEINUSTrialBudget
	denied   dataacquisition.SHEINUSTrialAction
}

func (a trialAuthority) AuthorizeSHEINUSTrial(_ context.Context, _ dataacquisition.Principal, _ string, action dataacquisition.SHEINUSTrialAction) (dataacquisition.SHEINUSTrialApproval, error) {
	if a.denied == action || (a.revoked != nil && *a.revoked == action) {
		return dataacquisition.SHEINUSTrialApproval{}, dataacquisition.ErrForbidden
	}
	return a.approval, nil
}

func (a trialAuthority) CheckSHEINUSTrialBudget(context.Context, dataacquisition.SHEINUSTrialApproval) (dataacquisition.SHEINUSTrialBudget, error) {
	return a.budget, nil
}

func TestSHEINUSSinglePageTrial(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	db := publicationTestDB(t, ctx)
	pool, err := db.DB()
	require.NoError(t, err)
	scope := collection.Scope{OrganizationID: "fixture-org", ActorID: "fixture-actor", MemberID: "fixture-member"}
	p := dataacquisition.Principal{Scope: scope}

	approval := func() trialAuthority {
		a := dataacquisition.SHEINUSTrialApproval{
			ID: uuid.NewString(), ProjectID: uuid.NewString(), Scope: scope,
			AccountRef: uuid.NewString(), KeyRef: uuid.NewString(), ExpiresAt: time.Now().Add(90 * time.Second), Funding: orgresource.FundingMember,
		}
		return trialAuthority{approval: a, budget: dataacquisition.SHEINUSTrialBudget{
			Binding: a.Binding(), AvailableCredits: 25, MaximumRequestCredits: 25,
			ZeroIncrementalCash: true, ExclusiveKeyWindow: true,
		}}
	}
	receipt := func(a dataacquisition.SHEINUSTrialApproval, requestID string, credits *int) dataacquisition.SHEINUSTrialUsage {
		return dataacquisition.SHEINUSTrialUsage{ApprovalID: a.ID, Binding: a.Binding(), RequestID: requestID, Credits: credits}
	}
	noMatch := func(context.Context, dataacquisition.SHEINUSTrialApproval, dataacquisition.SHEINUSTrialRequest, dataacquisition.SHEINUSTrialAttempt) (dataacquisition.SHEINUSTrialUsage, error) {
		return dataacquisition.SHEINUSTrialUsage{}, dataacquisition.ErrNotFound
	}
	a := approval()
	var submissions atomic.Int32
	credits := 25
	sender := func(sendCtx context.Context, a dataacquisition.SHEINUSTrialApproval, request dataacquisition.SHEINUSTrialRequest) (dataacquisition.SHEINUSTrialSubmission, error) {
		submissions.Add(1)
		assertedDeadline, ok := sendCtx.Deadline()
		if !ok || assertedDeadline.After(a.ExpiresAt) || request.URL != "https://us.shein.com/Women-Dresses-c-1727.html" ||
			request.Mode != "auto" || request.Extract != "auto" || request.ProxyCountry != "us" || request.CardLimit != 20 {
			t.Error("send did not preserve fixed request and deadline")
		}
		return dataacquisition.SHEINUSTrialSubmission{SHEINUSTrialUsage: receipt(a, "fixture-request-25", &credits)}, nil
	}

	_, err = NewSHEINUSTrial(pool, a, nil, noMatch)
	require.ErrorIs(t, err, dataacquisition.ErrUnavailable)
	service, err := NewSHEINUSTrial(pool, a, sender, noMatch)
	require.NoError(t, err)
	_, err = service.Send(ctx, p, a.approval.ID)
	require.ErrorIs(t, err, dataacquisition.ErrUnavailable, "an uninstalled trial store must fail closed")
	require.Zero(t, submissions.Load())
	require.NoError(t, jobstore.InstallSHEINUSTrialSchema(ctx, pool))

	t.Run("budget and authorization", func(t *testing.T) {
		for name, change := range map[string]func(*trialAuthority){
			"24 available":         func(a *trialAuthority) { a.budget.AvailableCredits = 24 },
			"unproven unit price":  func(a *trialAuthority) { a.budget.MaximumRequestCredits = 0 },
			"26 possible":          func(a *trialAuthority) { a.budget.MaximumRequestCredits = 26 },
			"cash possible":        func(a *trialAuthority) { a.budget.ZeroIncrementalCash = false },
			"shared key":           func(a *trialAuthority) { a.budget.ExclusiveKeyWindow = false },
			"wrong budget binding": func(a *trialAuthority) { a.budget.Binding = "another-trial" },
			"no send approval":     func(a *trialAuthority) { a.denied = dataacquisition.SHEINUSTrialSend },
			"expired": func(a *trialAuthority) {
				a.approval.ExpiresAt = time.Now().Add(-time.Second)
				a.budget.Binding = a.approval.Binding()
			},
		} {
			t.Run(name, func(t *testing.T) {
				rejected := approval()
				change(&rejected)
				service, err := NewSHEINUSTrial(pool, rejected, sender, noMatch)
				require.NoError(t, err)
				_, err = service.Send(ctx, p, rejected.approval.ID)
				require.Error(t, err)
				require.Zero(t, submissions.Load())
			})
		}
		other := p
		other.Scope.ActorID = "fixture-other"
		_, err := service.Send(ctx, other, a.approval.ID)
		require.ErrorIs(t, err, dataacquisition.ErrForbidden)
		require.Zero(t, submissions.Load())
	})

	t.Run("one durable winner", func(t *testing.T) {
		const contenders = 12
		start := make(chan struct{})
		errs := make(chan error, contenders)
		var wg sync.WaitGroup
		for range contenders {
			wg.Go(func() {
				<-start
				_, err := service.Send(ctx, p, a.approval.ID)
				errs <- err
			})
		}
		close(start)
		wg.Wait()
		close(errs)
		for err := range errs {
			if err != nil {
				require.ErrorIs(t, err, dataacquisition.ErrUnknown)
			}
		}
		require.EqualValues(t, 1, submissions.Load())
		reopened, err := gorm.Open(db.Dialector, &gorm.Config{})
		require.NoError(t, err)
		newPool, err := reopened.DB()
		require.NoError(t, err)
		require.NotSame(t, pool, newPool)
		t.Cleanup(func() { require.NoError(t, newPool.Close()) })
		fresh, err := NewSHEINUSTrial(newPool, a, sender, noMatch)
		require.NoError(t, err)
		got, err := fresh.Send(ctx, p, a.approval.ID)
		require.NoError(t, err)
		require.Equal(t, "fixture-request-25", got.RequestID)
		require.Equal(t, 25, *got.Credits)
		require.Zero(t, got.ReservedCredits())
		require.EqualValues(t, 1, submissions.Load())

		changed := a
		changed.approval.ProjectID = uuid.NewString()
		changed.budget.Binding = changed.approval.Binding()
		conflicting, err := NewSHEINUSTrial(newPool, changed, sender, noMatch)
		require.NoError(t, err)
		_, err = conflicting.Send(ctx, p, a.approval.ID)
		require.ErrorIs(t, err, dataacquisition.ErrConflict)
		require.EqualValues(t, 1, submissions.Load())
	})

	t.Run("timeout pins original receipt and holds budget", func(t *testing.T) {
		a := approval()
		a.approval.ExpiresAt = time.Now().Add(time.Second)
		a.budget.Binding = a.approval.Binding()
		const requestID = "fixture-timeout-original"
		var sends int
		timeout := func(sendCtx context.Context, a dataacquisition.SHEINUSTrialApproval, _ dataacquisition.SHEINUSTrialRequest) (dataacquisition.SHEINUSTrialSubmission, error) {
			sends++
			deadline, ok := sendCtx.Deadline()
			require.True(t, ok)
			require.True(t, deadline.Equal(a.ExpiresAt))
			<-sendCtx.Done()
			return dataacquisition.SHEINUSTrialSubmission{SHEINUSTrialUsage: receipt(a, requestID, &credits)}, nil
		}
		first, err := NewSHEINUSTrial(pool, a, timeout, noMatch)
		require.NoError(t, err)
		pending, err := first.Send(ctx, p, a.approval.ID)
		require.ErrorIs(t, err, dataacquisition.ErrUnknown)
		require.Equal(t, requestID, pending.RequestID)
		require.Equal(t, 25, pending.ReservedCredits())

		unknown, err := first.Reconcile(ctx, p, a.approval.ID)
		require.ErrorIs(t, err, dataacquisition.ErrUnknown)
		require.Equal(t, 25, unknown.ReservedCredits())

		requestToConfirm := "fixture-unrelated-request"
		actual := 10
		var lookups int
		lookup := func(_ context.Context, a dataacquisition.SHEINUSTrialApproval, _ dataacquisition.SHEINUSTrialRequest, original dataacquisition.SHEINUSTrialAttempt) (dataacquisition.SHEINUSTrialUsage, error) {
			lookups++
			require.Equal(t, requestID, original.RequestID)
			require.Equal(t, a.ID, original.ApprovalID)
			return receipt(a, requestToConfirm, &actual), nil
		}
		reopened, err := gorm.Open(db.Dialector, &gorm.Config{})
		require.NoError(t, err)
		newPool, err := reopened.DB()
		require.NoError(t, err)
		require.NotSame(t, pool, newPool)
		t.Cleanup(func() { require.NoError(t, newPool.Close()) })
		fresh, err := NewSHEINUSTrial(newPool, a, timeout, lookup)
		require.NoError(t, err)
		_, err = fresh.Send(ctx, p, a.approval.ID)
		require.ErrorIs(t, err, dataacquisition.ErrUnknown)
		_, err = fresh.Reconcile(ctx, p, a.approval.ID)
		require.ErrorIs(t, err, dataacquisition.ErrConflict)
		requestToConfirm = requestID
		otherTenant := p
		otherTenant.Scope.OrganizationID = "other-fixture-org"
		_, err = fresh.Reconcile(ctx, otherTenant, a.approval.ID)
		require.ErrorIs(t, err, dataacquisition.ErrForbidden)
		accountOwner := p
		accountOwner.Scope.ActorID = "fixture-account-owner"
		accountOwner.Scope.MemberID = "fixture-account-member"
		accounted, err := fresh.Reconcile(ctx, accountOwner, a.approval.ID)
		require.NoError(t, err)
		require.Equal(t, 10, *accounted.Credits)
		require.Zero(t, accounted.ReservedCredits())
		replay, err := fresh.Reconcile(ctx, p, a.approval.ID)
		require.NoError(t, err)
		require.Equal(t, accounted, replay)
		require.Equal(t, 2, lookups)
		require.Equal(t, 1, sends)
	})

	t.Run("revocation during send and reconciliation", func(t *testing.T) {
		a := approval()
		var revoked dataacquisition.SHEINUSTrialAction
		a.revoked = &revoked
		const requestID = "fixture-revoked-original"
		var sends int
		submit := func(_ context.Context, a dataacquisition.SHEINUSTrialApproval, _ dataacquisition.SHEINUSTrialRequest) (dataacquisition.SHEINUSTrialSubmission, error) {
			sends++
			revoked = dataacquisition.SHEINUSTrialView
			return dataacquisition.SHEINUSTrialSubmission{SHEINUSTrialUsage: receipt(a, requestID, &credits)}, nil
		}
		withdrawReconcile := true
		actual := 5
		lookup := func(_ context.Context, a dataacquisition.SHEINUSTrialApproval, _ dataacquisition.SHEINUSTrialRequest, original dataacquisition.SHEINUSTrialAttempt) (dataacquisition.SHEINUSTrialUsage, error) {
			require.Equal(t, requestID, original.RequestID)
			require.Equal(t, 25, original.ReservedCredits())
			if withdrawReconcile {
				revoked = dataacquisition.SHEINUSTrialReconcile
			}
			return receipt(a, requestID, &actual), nil
		}
		service, err := NewSHEINUSTrial(pool, a, submit, lookup)
		require.NoError(t, err)
		invisible, err := service.Send(ctx, p, a.approval.ID)
		require.ErrorIs(t, err, dataacquisition.ErrForbidden)
		require.Zero(t, invisible)

		revoked = ""
		invisible, err = service.Reconcile(ctx, p, a.approval.ID)
		require.ErrorIs(t, err, dataacquisition.ErrForbidden)
		require.Zero(t, invisible)

		revoked = ""
		pending, err := service.Send(ctx, p, a.approval.ID)
		require.ErrorIs(t, err, dataacquisition.ErrUnknown)
		require.Equal(t, 25, pending.ReservedCredits())
		require.Equal(t, requestID, pending.RequestID)

		withdrawReconcile = false
		accounted, err := service.Reconcile(ctx, p, a.approval.ID)
		require.NoError(t, err)
		require.Equal(t, actual, *accounted.Credits)
		require.Equal(t, 1, sends)
	})

	t.Run("commit before crash and no response", func(t *testing.T) {
		a := approval()
		store, err := jobstore.NewSHEINUSTrialStore(pool)
		require.NoError(t, err)
		_, won, err := store.Claim(ctx, a.approval)
		require.NoError(t, err)
		require.True(t, won)
		fresh, err := NewSHEINUSTrial(pool, a, sender, noMatch)
		require.NoError(t, err)
		pending, err := fresh.Send(ctx, p, a.approval.ID)
		require.ErrorIs(t, err, dataacquisition.ErrUnknown)
		require.Empty(t, pending.RequestID)
		unconfirmed, err := fresh.Reconcile(ctx, p, a.approval.ID)
		require.ErrorIs(t, err, dataacquisition.ErrUnknown)
		require.Nil(t, unconfirmed.Credits)
		require.Equal(t, 25, unconfirmed.ReservedCredits())
		require.Equal(t, 25, pending.ReservedCredits())
		require.EqualValues(t, 1, submissions.Load())
	})

	t.Run("record actual over-limit and reject reused request ID", func(t *testing.T) {
		a := approval()
		actual := 26
		requestID := "fixture-over-limit"
		over := func(_ context.Context, a dataacquisition.SHEINUSTrialApproval, _ dataacquisition.SHEINUSTrialRequest) (dataacquisition.SHEINUSTrialSubmission, error) {
			return dataacquisition.SHEINUSTrialSubmission{SHEINUSTrialUsage: receipt(a, requestID, &actual)}, nil
		}
		first, err := NewSHEINUSTrial(pool, a, over, noMatch)
		require.NoError(t, err)
		got, err := first.Send(ctx, p, a.approval.ID)
		require.NoError(t, err)
		require.True(t, got.OverLimit())
		require.Equal(t, 26, *got.Credits)
		second := approval()
		another, err := NewSHEINUSTrial(pool, second, over, noMatch)
		require.NoError(t, err)
		_, err = another.Send(ctx, p, second.approval.ID)
		require.ErrorIs(t, err, dataacquisition.ErrConflict)
		pending, err := another.Send(ctx, p, second.approval.ID)
		require.ErrorIs(t, err, dataacquisition.ErrUnknown)
		require.Equal(t, 25, pending.ReservedCredits())
	})
}
