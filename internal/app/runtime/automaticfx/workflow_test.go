package automaticfxruntime

import (
	"context"
	"errors"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/testsuite"
	"go.temporal.io/sdk/workflow"
	d "task-processor/internal/aicapability/routeconfig"
	fx "task-processor/internal/integration/aicapability/automaticfx"
	"testing"
	"time"
)

func TestCoordinatorReconcilesUnknownWithoutRepublishing(t *testing.T) {
	var suite testsuite.WorkflowTestSuite
	env := suite.NewTestWorkflowEnvironment()
	env.RegisterActivityWithOptions(func(context.Context) (fx.Observation, error) { return fx.Observation{}, nil }, activity.RegisterOptions{Name: FetchActivity})
	env.RegisterActivityWithOptions(func(context.Context, fx.Observation) (Intent, error) { return Intent{}, nil }, activity.RegisterOptions{Name: PrepareActivity})
	env.RegisterActivityWithOptions(func(context.Context, Intent) (PublishResult, error) { return PublishResult{}, nil }, activity.RegisterOptions{Name: PublishActivity})
	env.RegisterActivityWithOptions(func(context.Context, Intent) (bool, error) { return false, nil }, activity.RegisterOptions{Name: InspectActivity})
	obs := fx.Observation{Date: "2026-11-12", CNYMicrosPerUSD: 6717400}
	intent := Intent{Day: obs.Date, Input: d.PlatformPricesInput{ExpectedRevision: "fixture"}, Publish: true}
	dispatched := intent
	dispatched.Attempt = 1
	env.OnActivity(FetchActivity, mock.Anything).Return(obs, nil).Once()
	env.OnActivity(PrepareActivity, mock.Anything, obs).Return(intent, nil).Once()
	env.OnActivity(PublishActivity, mock.Anything, dispatched).Return(PublishResult{}, errors.New("unknown acknowledgement")).Once()
	env.OnActivity(InspectActivity, mock.Anything, dispatched).Return(false, nil).Once()
	env.OnActivity(InspectActivity, mock.Anything, dispatched).Return(true, nil).Once()
	env.RegisterDelayedCallback(func() { env.SignalWorkflow(TickSignal, struct{}{}) }, time.Second)
	env.RegisterDelayedCallback(func() { env.CancelWorkflow() }, 7*time.Minute)
	env.ExecuteWorkflow(CoordinatorWorkflow, CoordinatorState{})
	require.True(t, env.IsWorkflowCompleted())
	require.Error(t, env.GetWorkflowError())
	env.AssertExpectations(t)
}

func TestCoordinatorRestoredPendingOnlyInspects(t *testing.T) {
	var suite testsuite.WorkflowTestSuite
	env := suite.NewTestWorkflowEnvironment()
	env.RegisterActivityWithOptions(func(context.Context, Intent) (bool, error) { return false, nil }, activity.RegisterOptions{Name: InspectActivity})
	intent := Intent{Day: "2026-11-12", Input: d.PlatformPricesInput{ExpectedRevision: "fixture"}, Publish: true}
	env.OnActivity(InspectActivity, mock.Anything, intent).Return(true, nil).Once()
	env.RegisterDelayedCallback(func() { env.CancelWorkflow() }, time.Second)
	env.ExecuteWorkflow(CoordinatorWorkflow, CoordinatorState{Pending: &intent})
	require.True(t, env.IsWorkflowCompleted())
	require.Error(t, env.GetWorkflowError())
	env.AssertExpectations(t)
}

func TestCoordinatorDoesNotConsumeNewTicksWhileUnknown(t *testing.T) {
	var suite testsuite.WorkflowTestSuite
	env := suite.NewTestWorkflowEnvironment()
	env.RegisterActivityWithOptions(func(context.Context, Intent) (bool, error) { return false, nil }, activity.RegisterOptions{Name: InspectActivity})
	intent := Intent{Day: "2026-11-12", Input: d.PlatformPricesInput{ExpectedRevision: "fixture"}, Publish: true}
	env.OnActivity(InspectActivity, mock.Anything, intent).Return(false, nil)
	env.RegisterDelayedCallback(func() { env.SignalWorkflow(TickSignal, struct{}{}) }, time.Second)
	env.RegisterDelayedCallback(func() { env.SignalWorkflow(TickSignal, struct{}{}) }, time.Minute)
	env.RegisterDelayedCallback(func() { env.CancelWorkflow() }, 6*time.Minute)
	env.ExecuteWorkflow(CoordinatorWorkflow, CoordinatorState{Pending: &intent})
	require.True(t, env.IsWorkflowCompleted())
	require.Error(t, env.GetWorkflowError())
	env.AssertExpectations(t)
}

func TestCoordinatorNeverContinuesAsNewWhileOutcomeUnknown(t *testing.T) {
	var suite testsuite.WorkflowTestSuite
	env := suite.NewTestWorkflowEnvironment()
	env.RegisterActivityWithOptions(func(context.Context, Intent) (bool, error) { return false, nil }, activity.RegisterOptions{Name: InspectActivity})
	intent := Intent{Day: "2026-11-12", Input: d.PlatformPricesInput{ExpectedRevision: "fixture"}, Publish: true}
	env.OnActivity(InspectActivity, mock.Anything, intent).Return(false, nil)
	env.RegisterDelayedCallback(func() { env.CancelWorkflow() }, 510*time.Minute)
	env.ExecuteWorkflow(CoordinatorWorkflow, CoordinatorState{Pending: &intent})
	require.Error(t, env.GetWorkflowError())
	var continued *workflow.ContinueAsNewError
	require.False(t, errors.As(env.GetWorkflowError(), &continued), "unknown original command must remain in its current history")
}

func TestCoordinatorStopsOnlyAfterUnknownReceiptResolved(t *testing.T) {
	var suite testsuite.WorkflowTestSuite
	env := suite.NewTestWorkflowEnvironment()
	env.RegisterActivityWithOptions(func(context.Context, Intent) (bool, error) { return false, nil }, activity.RegisterOptions{Name: InspectActivity})
	intent := Intent{Day: "2026-11-12", Input: d.PlatformPricesInput{ExpectedRevision: "fixture"}, Publish: true, Attempt: 1}
	env.OnActivity(InspectActivity, mock.Anything, intent).Return(false, nil).Once()
	env.OnActivity(InspectActivity, mock.Anything, intent).Return(true, nil).Once()
	env.RegisterDelayedCallback(func() { env.SignalWorkflow(StopSignal, struct{}{}) }, time.Minute)
	env.RegisterDelayedCallback(func() { env.SignalWorkflow(TickSignal, struct{}{}) }, 2*time.Minute)
	env.RegisterDelayedCallback(func() {
		v, e := env.QueryWorkflow("status")
		require.NoError(t, e)
		var state StatusSnapshot
		require.NoError(t, v.Get(&state))
		require.Equal(t, "QUIESCED", state.State)
	}, 3*time.Minute)
	env.RegisterDelayedCallback(func() { env.CancelWorkflow() }, 4*time.Minute)
	env.ExecuteWorkflow(CoordinatorWorkflow, CoordinatorState{Pending: &intent})
	require.True(t, env.IsWorkflowCompleted())
	require.Error(t, env.GetWorkflowError())
	env.AssertExpectations(t)
}

func TestStopDuringFetchPreventsPublication(t *testing.T) {
	var suite testsuite.WorkflowTestSuite
	env := suite.NewTestWorkflowEnvironment()
	env.RegisterActivityWithOptions(func(context.Context) (fx.Observation, error) { return fx.Observation{}, nil }, activity.RegisterOptions{Name: FetchActivity})
	obs := fx.Observation{Date: "2026-11-12", CNYMicrosPerUSD: 6717400}
	env.OnActivity(FetchActivity, mock.Anything).After(2*time.Second).Return(obs, nil).Once()
	env.RegisterDelayedCallback(func() { env.SignalWorkflow(TickSignal, struct{}{}) }, time.Second)
	env.RegisterDelayedCallback(func() { env.SignalWorkflow(StopSignal, struct{}{}) }, 1500*time.Millisecond)
	env.RegisterDelayedCallback(func() {
		v, e := env.QueryWorkflow("status")
		require.NoError(t, e)
		var state StatusSnapshot
		require.NoError(t, v.Get(&state))
		require.Equal(t, "QUIESCED", state.State)
	}, 5*time.Second)
	env.RegisterDelayedCallback(func() { env.CancelWorkflow() }, 6*time.Second)
	env.ExecuteWorkflow(CoordinatorWorkflow, CoordinatorState{})
	require.True(t, env.IsWorkflowCompleted())
	require.Error(t, env.GetWorkflowError())
	env.AssertExpectations(t)
}

func TestCoordinatorRetriesOnlyKnownUncommittedSnapshot(t *testing.T) {
	var suite testsuite.WorkflowTestSuite
	env := suite.NewTestWorkflowEnvironment()
	env.RegisterActivityWithOptions(func(context.Context) (fx.Observation, error) { return fx.Observation{}, nil }, activity.RegisterOptions{Name: FetchActivity})
	env.RegisterActivityWithOptions(func(context.Context, fx.Observation) (Intent, error) { return Intent{}, nil }, activity.RegisterOptions{Name: PrepareActivity})
	env.RegisterActivityWithOptions(func(context.Context, Intent) (PublishResult, error) { return PublishResult{}, nil }, activity.RegisterOptions{Name: PublishActivity})
	obs := fx.Observation{Date: "2026-11-12", CNYMicrosPerUSD: 6717400}
	first := Intent{Day: obs.Date, Input: d.PlatformPricesInput{ExpectedRevision: "before"}, Publish: true}
	second := Intent{Day: obs.Date, Input: d.PlatformPricesInput{ExpectedRevision: "after"}, Publish: true}
	one, two := first, second
	one.Attempt = 1
	two.Attempt = 2
	env.OnActivity(FetchActivity, mock.Anything).Return(obs, nil).Twice()
	env.OnActivity(PrepareActivity, mock.Anything, obs).Return(first, nil).Once()
	env.OnActivity(PrepareActivity, mock.Anything, obs).Return(second, nil).Once()
	env.OnActivity(PublishActivity, mock.Anything, one).Return(PublishResult{StaleSnapshot: true}, nil).Once()
	env.OnActivity(PublishActivity, mock.Anything, two).Return(PublishResult{}, nil).Once()
	env.RegisterDelayedCallback(func() { env.SignalWorkflow(TickSignal, struct{}{}) }, time.Second)
	env.RegisterDelayedCallback(func() { env.CancelWorkflow() }, time.Minute)
	env.ExecuteWorkflow(CoordinatorWorkflow, CoordinatorState{})
	require.True(t, env.IsWorkflowCompleted())
	require.Error(t, env.GetWorkflowError())
	env.AssertExpectations(t)
}

func TestRegisterRejectsMissingDependencies(t *testing.T) {
	require.Error(t, RegisterWorker(nil, nil, fx.Calendar{}, nil))
}

func TestCoordinatorStopsIfUSDChangedAfterStaleConflict(t *testing.T) {
	var suite testsuite.WorkflowTestSuite
	env := suite.NewTestWorkflowEnvironment()
	env.RegisterActivityWithOptions(func(context.Context) (fx.Observation, error) { return fx.Observation{}, nil }, activity.RegisterOptions{Name: FetchActivity})
	env.RegisterActivityWithOptions(func(context.Context, fx.Observation) (Intent, error) { return Intent{}, nil }, activity.RegisterOptions{Name: PrepareActivity})
	env.RegisterActivityWithOptions(func(context.Context, Intent) (PublishResult, error) { return PublishResult{}, nil }, activity.RegisterOptions{Name: PublishActivity})
	obs := fx.Observation{Date: "2026-11-12", CNYMicrosPerUSD: 6717400}
	first := Intent{Day: obs.Date, Publish: true, USDBase: []d.PublishedFXRate{{Currency: "USD", CNYMicrosPerCurrencyUnit: 7000000}}}
	changed := first
	changed.USDBase = []d.PublishedFXRate{{Currency: "USD", CNYMicrosPerCurrencyUnit: 7100000}}
	dispatched := first
	dispatched.Attempt = 1
	env.OnActivity(FetchActivity, mock.Anything).Return(obs, nil).Twice()
	env.OnActivity(PrepareActivity, mock.Anything, obs).Return(first, nil).Once()
	env.OnActivity(PrepareActivity, mock.Anything, obs).Return(changed, nil).Once()
	env.OnActivity(PublishActivity, mock.Anything, dispatched).Return(PublishResult{StaleSnapshot: true}, nil).Once()
	env.RegisterDelayedCallback(func() { env.SignalWorkflow(TickSignal, struct{}{}) }, time.Second)
	env.RegisterDelayedCallback(func() {
		v, e := env.QueryWorkflow("status")
		require.NoError(t, e)
		var s StatusSnapshot
		require.NoError(t, v.Get(&s))
		require.Equal(t, "USD_CONFLICT", s.State)
		env.SignalWorkflow(TickSignal, struct{}{})
	}, time.Minute)
	env.RegisterDelayedCallback(func() { env.CancelWorkflow() }, 2*time.Minute)
	env.ExecuteWorkflow(CoordinatorWorkflow, CoordinatorState{})
	env.AssertExpectations(t)
}

func TestPublishDoesNotDispatchStalePreparedIntent(t *testing.T) {
	now := time.Now()
	intent := Intent{Day: "2026-11-12", Publish: true, Input: d.PlatformPricesInput{FX: []d.PublishedFXRate{{Currency: "USD", RateDate: "2026-11-12", EffectiveFrom: now.Add(-2 * time.Minute), EffectiveUntil: now.Add(time.Hour)}}}}
	result, err := (activities{}).Publish(context.Background(), intent)
	require.NoError(t, err)
	require.True(t, result.NotDispatched)
}
