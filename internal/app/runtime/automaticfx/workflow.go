// Package automaticfxruntime orchestrates the opt-in, platform-owned FX feed.
package automaticfxruntime

import (
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
	"reflect"
	d "task-processor/internal/aicapability/routeconfig"
	fx "task-processor/internal/integration/aicapability/automaticfx"
	"time"
)

const (
	CoordinatorWorkflowName = "PlatformFXCoordinatorV1"
	CoordinatorID           = "platform-fx-usd-coordinator-v1"
	TickWorkflowName        = "PlatformFXTickV1"
	TickSignal              = "platform-fx-tick-v1"
	StopSignal              = "platform-fx-stop-v1"
	FetchActivity           = "PlatformFXFetchV1"
	PrepareActivity         = "PlatformFXPrepareV1"
	PublishActivity         = "PlatformFXPublishAttemptV1"
	InspectActivity         = "PlatformFXInspectReceiptV1"
)

type Intent struct {
	Day     string
	Input   d.PlatformPricesInput
	Publish bool
	Attempt uint32
	USDBase []d.PublishedFXRate
	NextDue time.Time
}
type USDBase struct{ Rates []d.PublishedFXRate }
type PublishResult struct {
	NotDispatched bool
	StaleSnapshot bool
}
type CoordinatorState struct {
	Pending       *Intent
	Wake          bool
	Status        string
	AttemptDay    string
	NextAttempt   uint32
	StatusAt      time.Time
	StopRequested bool
	RetryUSD      *USDBase
	BlockedUSD    bool
	DueAt         time.Time
}

func dueForIntent(intent Intent) time.Time {
	for _, rate := range intent.Input.FX {
		if rate.Currency == "USD" && rate.RateDate == intent.Day {
			return rate.EffectiveUntil
		}
	}
	return time.Time{}
}

type StatusSnapshot struct {
	State         string
	CheckedAt     time.Time
	StopRequested bool
}

// TickWorkflow signals only the existing coordinator: it never starts or
// recreates it, including when coordinator history is missing or terminated.
func TickWorkflow(ctx workflow.Context) error {
	return workflow.SignalExternalWorkflow(ctx, CoordinatorID, "", TickSignal, struct{}{}).Get(ctx, nil)
}

// CoordinatorWorkflow serializes all dates. The complete original command is
// recorded before dispatch; an UNKNOWN result can only be inspected.
func CoordinatorWorkflow(ctx workflow.Context, state CoordinatorState) error {
	activityCtx := workflow.WithActivityOptions(ctx, workflow.ActivityOptions{StartToCloseTimeout: 30 * time.Second, RetryPolicy: &temporal.RetryPolicy{MaximumAttempts: 1}})
	ticks := workflow.GetSignalChannel(ctx, TickSignal)
	stops := workflow.GetSignalChannel(ctx, StopSignal)
	drainStops := func() {
		var event struct{}
		for stops.ReceiveAsync(&event) {
			state.StopRequested = true
		}
	}
	if err := workflow.SetQueryHandler(ctx, "status", func() (StatusSnapshot, error) {
		return StatusSnapshot{State: state.Status, CheckedAt: state.StatusAt, StopRequested: state.StopRequested}, nil
	}); err != nil {
		return err
	}
	cycles := 0
	for {
		drainStops()
		if state.StopRequested && state.Pending == nil {
			state.Wake = false
			state.Status = "QUIESCED"
			state.StatusAt = workflow.Now(ctx)
			// Keep the unique coordinator alive as a durable stop fence. Ticks are
			// consumed but never cause an automatic write after QUIESCED.
			selector := workflow.NewSelector(ctx)
			selector.AddReceive(stops, func(c workflow.ReceiveChannel, _ bool) { var e struct{}; c.Receive(ctx, &e) })
			selector.AddReceive(ticks, func(c workflow.ReceiveChannel, _ bool) { var e struct{}; c.Receive(ctx, &e) })
			selector.Select(ctx)
			if ctx.Err() != nil {
				return ctx.Err()
			}
		} else if state.BlockedUSD {
			state.Status = "USD_CONFLICT"
			state.StatusAt = workflow.Now(ctx)
			selector := workflow.NewSelector(ctx)
			selector.AddReceive(stops, func(c workflow.ReceiveChannel, _ bool) {
				var e struct{}
				c.Receive(ctx, &e)
				state.StopRequested = true
			})
			selector.AddReceive(ticks, func(c workflow.ReceiveChannel, _ bool) { var e struct{}; c.Receive(ctx, &e) })
			selector.Select(ctx)
			if ctx.Err() != nil {
				return ctx.Err()
			}
		} else if state.Pending != nil {
			if state.StopRequested {
				state.Status = "DRAINING"
			} else {
				state.Status = "UNKNOWN"
			}
			state.StatusAt = workflow.Now(ctx)
			var found bool
			if err := workflow.ExecuteActivity(activityCtx, InspectActivity, *state.Pending).Get(ctx, &found); err == nil && found {
				state.DueAt = dueForIntent(*state.Pending)
				state.Pending = nil
				state.Status = "PUBLISHED"
			} else {
				// STOP must wake an unknown wait; after STOP, keep inspecting the original
				// key without processing a queued tick or issuing a new publication.
				if state.StopRequested {
					if err := workflow.Sleep(ctx, 5*time.Minute); err != nil {
						return err
					}
				} else {
					timerCtx, cancel := workflow.WithCancel(ctx)
					timer := workflow.NewTimer(timerCtx, 5*time.Minute)
					selector := workflow.NewSelector(ctx)
					selector.AddReceive(stops, func(c workflow.ReceiveChannel, _ bool) {
						var e struct{}
						c.Receive(ctx, &e)
						state.StopRequested = true
					})
					selector.AddFuture(timer, func(workflow.Future) {})
					selector.Select(ctx)
					cancel()
					if ctx.Err() != nil {
						return ctx.Err()
					}
				}
			}
		} else {
			if !state.DueAt.IsZero() && workflow.Now(ctx).Before(state.DueAt) {
				state.Wake = false
			}
			if !state.Wake {
				if !state.DueAt.IsZero() && !workflow.Now(ctx).Before(state.DueAt) {
					state.Wake = true
				} else {
					selector := workflow.NewSelector(ctx)
					selector.AddReceive(stops, func(c workflow.ReceiveChannel, _ bool) {
						var e struct{}
						c.Receive(ctx, &e)
						state.StopRequested = true
					})
					selector.AddReceive(ticks, func(c workflow.ReceiveChannel, _ bool) {
						var e struct{}
						c.Receive(ctx, &e)
						if state.DueAt.IsZero() || !workflow.Now(ctx).Before(state.DueAt) {
							state.Wake = true
						}
					})
					var cancel workflow.CancelFunc
					if !state.DueAt.IsZero() {
						timerCtx, c := workflow.WithCancel(ctx)
						cancel = c
						selector.AddFuture(workflow.NewTimer(timerCtx, state.DueAt.Sub(workflow.Now(ctx))), func(workflow.Future) { state.Wake = true })
						state.Status = "WAITING"
					}
					selector.Select(ctx)
					if cancel != nil {
						cancel()
					}
					if ctx.Err() != nil {
						return ctx.Err()
					}
				}
			}
			drainStops()
			if state.StopRequested || !state.Wake {
				continue
			}
			state.Wake = false
			state.DueAt = time.Time{}
			state.Status = "CHECKING"
			state.StatusAt = workflow.Now(ctx)
			var obs fx.Observation
			if err := workflow.ExecuteActivity(activityCtx, FetchActivity).Get(ctx, &obs); err != nil {
				state.Status = "SOURCE_UNAVAILABLE"
			} else {
				drainStops()
				if state.StopRequested {
					continue
				}
				var intent Intent
				if err := workflow.ExecuteActivity(activityCtx, PrepareActivity, obs).Get(ctx, &intent); err != nil {
					state.Status = "NOT_READY"
				} else if state.RetryUSD != nil && !reflect.DeepEqual(state.RetryUSD.Rates, intent.USDBase) {
					state.BlockedUSD = true
					state.Status = "USD_CONFLICT"
				} else if !intent.Publish {
					state.RetryUSD = nil
					state.DueAt = intent.NextDue
					state.Status = "CHECKED"
				} else {
					state.RetryUSD = nil
					drainStops()
					if state.StopRequested {
						continue
					}
					if state.AttemptDay != intent.Day {
						state.AttemptDay = intent.Day
						state.NextAttempt = 1
					}
					if state.NextAttempt == 0 || state.NextAttempt > 100 {
						state.Status = "NOT_READY"
						continue
					}
					intent.Attempt = state.NextAttempt
					state.Pending = &intent
					state.Status = "PUBLISHING"
					state.StatusAt = workflow.Now(ctx)
					var result PublishResult
					if err := workflow.ExecuteActivity(activityCtx, PublishActivity, intent).Get(ctx, &result); err == nil {
						state.Pending = nil
						if result.StaleSnapshot {
							state.RetryUSD = &USDBase{Rates: intent.USDBase}
							state.Status = "RETRYING"
							state.NextAttempt++
							state.Wake = true
						} else if result.NotDispatched {
							state.Status = "NOT_READY"
							state.NextAttempt++
						} else {
							state.DueAt = dueForIntent(intent)
							state.Status = "PUBLISHED"
						}
					} else {
						state.Status = "UNKNOWN"
					}
				}
			}
		}
		state.StatusAt = workflow.Now(ctx)
		cycles++
		if cycles >= 100 && state.Pending == nil {
			var event struct{}
			for ticks.ReceiveAsync(&event) {
				if !state.StopRequested && (state.DueAt.IsZero() || !workflow.Now(ctx).Before(state.DueAt)) {
					state.Wake = true
				}
			}
			drainStops()
			return workflow.NewContinueAsNewError(ctx, CoordinatorWorkflowName, state)
		}
	}
}
