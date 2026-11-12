package automaticfxruntime

import (
	"context"
	"errors"
	"net/http"
	"time"

	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/worker"
	"go.temporal.io/sdk/workflow"
	d "task-processor/internal/aicapability/routeconfig"
	fx "task-processor/internal/integration/aicapability/automaticfx"
)

const TaskQueue = "platform-fx-usd-v1"

type activities struct {
	prices    *d.PlatformPricesService
	transport http.RoundTripper
}

func (a activities) Fetch(ctx context.Context) (fx.Observation, error) {
	return fx.FetchCFETS(ctx, a.transport, time.Now())
}
func (a activities) Prepare(ctx context.Context, obs fx.Observation) (Intent, error) {
	now := time.Now().UTC()
	current, err := a.prices.Current(ctx)
	if err != nil {
		return Intent{}, err
	}
	if current.Payload.PlatformPrices == nil {
		return Intent{}, d.ErrUnavailable
	}
	var usdBase []d.PublishedFXRate
	for _, rate := range current.Payload.PlatformPrices.FX {
		if rate.Currency == "USD" {
			usdBase = append(usdBase, rate)
		}
	}
	input, changed, err := fx.PrepareWeeklyPublication(current.ID, *current.Payload.PlatformPrices, obs, now)
	if err != nil {
		return Intent{}, err
	}
	if !changed {
		var due time.Time
		for _, rate := range current.Payload.PlatformPrices.FX {
			if rate.Currency == "USD" && rate.RateDate == obs.Date && rate.CNYMicrosPerCurrencyUnit == obs.CNYMicrosPerUSD && !now.Before(rate.EffectiveFrom) && now.Before(rate.EffectiveUntil) {
				due = rate.EffectiveUntil
			}
		}
		if due.IsZero() {
			return Intent{}, d.ErrNeedsConfiguration
		}
		return Intent{Day: obs.Date, USDBase: usdBase, NextDue: due}, nil
	}
	for _, rate := range usdBase {
		if !now.Before(rate.EffectiveFrom) && now.Before(rate.EffectiveUntil) && rate.EffectiveUntil.Sub(rate.EffectiveFrom) == 7*24*time.Hour {
			return Intent{Day: obs.Date, USDBase: usdBase, NextDue: rate.EffectiveUntil}, nil
		}
	}
	return Intent{Day: obs.Date, Input: input, Publish: changed, USDBase: usdBase}, nil
}
func (a activities) Publish(ctx context.Context, intent Intent) (PublishResult, error) {
	if !intent.Publish {
		return PublishResult{NotDispatched: true}, nil
	}
	now := time.Now().UTC()
	valid := false
	for _, rate := range intent.Input.FX {
		if rate.Currency == "USD" && rate.RateDate == intent.Day && !now.Before(rate.EffectiveFrom) && now.Before(rate.EffectiveUntil) && now.Sub(rate.EffectiveFrom) <= 30*time.Second {
			valid = true
		}
	}
	if !valid {
		return PublishResult{NotDispatched: true}, nil
	}
	_, err := fx.PublishFirstAttempt(ctx, fx.ServiceOwner{Service: a.prices}, intent.Day, intent.Attempt, intent.Input)
	if errors.Is(err, d.ErrStaleSnapshot) {
		return PublishResult{StaleSnapshot: true}, nil
	}
	if errors.Is(err, d.ErrPublicationExpired) {
		return PublishResult{NotDispatched: true}, nil
	}
	return PublishResult{}, err
}
func (a activities) Inspect(ctx context.Context, intent Intent) (bool, error) {
	_, err := fx.InspectPublication(ctx, fx.ServiceOwner{Service: a.prices}, intent.Day, intent.Attempt, intent.Input)
	if errors.Is(err, d.ErrOutcomeUnknown) {
		return false, nil
	}
	return err == nil, err
}

// RegisterWorker only registers functions; it does not start a worker, create
// a schedule, read a provider, or modify platform data.
func RegisterWorker(w worker.Worker, prices *d.PlatformPricesService, transport http.RoundTripper) error {
	if w == nil || prices == nil || prices.Repository == nil {
		return d.ErrUnavailable
	}
	a := activities{prices: prices, transport: transport}
	w.RegisterWorkflowWithOptions(CoordinatorWorkflow, workflow.RegisterOptions{Name: CoordinatorWorkflowName})
	w.RegisterWorkflowWithOptions(TickWorkflow, workflow.RegisterOptions{Name: TickWorkflowName})
	w.RegisterActivityWithOptions(a.Fetch, activity.RegisterOptions{Name: FetchActivity})
	w.RegisterActivityWithOptions(a.Prepare, activity.RegisterOptions{Name: PrepareActivity})
	w.RegisterActivityWithOptions(a.Publish, activity.RegisterOptions{Name: PublishActivity})
	w.RegisterActivityWithOptions(a.Inspect, activity.RegisterOptions{Name: InspectActivity})
	return nil
}
