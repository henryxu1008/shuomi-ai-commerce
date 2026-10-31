package fulfillmentruntime

import (
	"context"
	"errors"
	"github.com/google/uuid"
	enums "go.temporal.io/api/enums/v1"
	"go.temporal.io/api/serviceerror"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/worker"
	"go.temporal.io/sdk/workflow"
	"sync/atomic"
	app "task-processor/internal/app/fulfillment"
	"task-processor/internal/authidentity"
	f "task-processor/internal/fulfillment"
	"time"
)

const Queue = "fulfillment-current"
const WorkflowName = "FulfillmentPlatformV1"
const sendActivity = "FulfillmentSendOnceV1"

type Execution struct{ Alias, Organization, Operation string }

func PlatformWorkflow(ctx workflow.Context, in Execution) error {
	if !authidentity.IsBoundedIdentifier(in.Alias) || !authidentity.IsBoundedIdentifier(in.Organization) || uuid.Validate(in.Operation) != nil {
		return temporal.NewNonRetryableApplicationError("invalid fulfillment operation", "invalid", nil)
	}
	// Persisted ERP fences, not Temporal retries, own the outcome of provider mutations.
	ctx = workflow.WithActivityOptions(ctx, workflow.ActivityOptions{StartToCloseTimeout: 3 * time.Minute, ScheduleToCloseTimeout: 5 * time.Minute, RetryPolicy: &temporal.RetryPolicy{MaximumAttempts: 1}})
	return workflow.ExecuteActivity(ctx, sendActivity, in).Get(ctx, nil)
}

type Starter struct{ Client client.Client }

func (s Starter) Ensure(ctx context.Context, alias, org, id string) error {
	if s.Client == nil || !authidentity.IsBoundedIdentifier(alias) || !authidentity.IsBoundedIdentifier(org) || uuid.Validate(id) != nil {
		return f.ErrInvalid
	}
	// Explicit ensure may restart a failed workflow only while its native
	// operation is still CONFIRMED. The native claim/dispatch fences still win.
	_, err := s.Client.ExecuteWorkflow(ctx, client.StartWorkflowOptions{ID: "fulfillment/" + alias + "/" + org + "/" + id, TaskQueue: Queue, WorkflowExecutionTimeout: 10 * time.Minute, WorkflowIDConflictPolicy: enums.WORKFLOW_ID_CONFLICT_POLICY_USE_EXISTING, WorkflowIDReusePolicy: enums.WORKFLOW_ID_REUSE_POLICY_ALLOW_DUPLICATE_FAILED_ONLY}, WorkflowName, Execution{alias, org, id})
	var existing *serviceerror.WorkflowExecutionAlreadyStarted
	if err != nil && !errors.As(err, &existing) {
		return f.ErrUnknown
	}
	return nil
}

type Worker interface {
	Start() error
	Stop()
}
type Lifecycle struct {
	Worker Worker
	ready  atomic.Bool
	failed atomic.Bool
}

func (l *Lifecycle) Ready() bool { return l != nil && !l.failed.Load() && l.ready.Load() }
func (l *Lifecycle) Start() error {
	if l == nil || l.Worker == nil {
		return f.ErrUnavailable
	}
	if err := l.Worker.Start(); err != nil {
		l.failed.Store(true)
		return err
	}
	if l.failed.Load() {
		return f.ErrUnavailable
	}
	l.ready.Store(true)
	return nil
}
func (l *Lifecycle) Stop() {
	if l != nil {
		l.failed.Store(true)
		l.ready.Store(false)
		if l.Worker != nil {
			l.Worker.Stop()
		}
	}
}
func NewWorker(c client.Client, s *app.Service, l *Lifecycle) (Worker, error) {
	if c == nil || s == nil || l == nil {
		return nil, f.ErrUnavailable
	}
	w := worker.New(c, Queue, worker.Options{MaxConcurrentActivityExecutionSize: 1, OnFatalError: func(error) { l.failed.Store(true); l.ready.Store(false) }})
	w.RegisterWorkflowWithOptions(PlatformWorkflow, workflow.RegisterOptions{Name: WorkflowName})
	w.RegisterActivityWithOptions(func(ctx context.Context, in Execution) error {
		return s.Run(ctx, in.Alias, in.Organization, in.Operation)
	}, activity.RegisterOptions{Name: sendActivity})
	return w, nil
}
