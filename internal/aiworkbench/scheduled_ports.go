package aiworkbench

import (
	"context"
	"time"
)

type ScheduledDispatcher interface {
	DispatchScheduled(context.Context, string, time.Time) error
	ReconcileScheduled(context.Context, string, time.Time) (bool, error)
}

// ScheduleBackend is a projection of Workbench's durable intent. It does not
// authorize business execution or own Agent state and actual usage.
type ScheduleBackend interface {
	Synchronize(context.Context, ScheduledPlan) error
	Start(ScheduledDispatcher) (func(), error)
	Ready() bool
}
