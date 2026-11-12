package automaticfxruntime

import (
	"context"
	"errors"
	"time"

	enums "go.temporal.io/api/enums/v1"
	servicepb "go.temporal.io/api/workflowservice/v1"
	"go.temporal.io/sdk/client"
)

const ScheduleID = "platform-fx-usd-schedule-v1"

var ErrControlUnknown = errors.New("automatic FX control outcome unknown")

type bootstrapOperations interface {
	createPaused(context.Context) error
	startCoordinator(context.Context) error
	verifyCoordinator(context.Context) error
	unpause(context.Context) error
	initialTick(context.Context) error
}

func bootstrap(ctx context.Context, ops bootstrapOperations) error {
	if ctx == nil || ops == nil {
		return ErrControlUnknown
	}
	for _, step := range []func(context.Context) error{ops.createPaused, ops.startCoordinator, ops.verifyCoordinator, ops.unpause, ops.initialTick} {
		if step(ctx) != nil {
			return ErrControlUnknown
		}
	}
	return nil
}

type temporalBootstrap struct {
	client client.Client
	handle client.ScheduleHandle
	runID  string
}

func (b *temporalBootstrap) createPaused(ctx context.Context) error {
	h, err := b.client.ScheduleClient().Create(ctx, client.ScheduleOptions{ID: ScheduleID, Paused: true, Overlap: enums.SCHEDULE_OVERLAP_POLICY_SKIP, Spec: client.ScheduleSpec{Intervals: []client.ScheduleIntervalSpec{{Every: 7 * 24 * time.Hour}}}, Action: &client.ScheduleWorkflowAction{ID: "platform-fx-tick-v1", Workflow: TickWorkflowName, TaskQueue: TaskQueue}})
	if err != nil {
		return err
	}
	b.handle = h
	return nil
}
func (b *temporalBootstrap) startCoordinator(ctx context.Context) error {
	r, err := b.client.ExecuteWorkflow(ctx, client.StartWorkflowOptions{ID: CoordinatorID, TaskQueue: TaskQueue, WorkflowIDReusePolicy: enums.WORKFLOW_ID_REUSE_POLICY_REJECT_DUPLICATE, WorkflowExecutionErrorWhenAlreadyStarted: true}, CoordinatorWorkflowName, CoordinatorState{})
	if err != nil {
		return err
	}
	b.runID = r.GetRunID()
	if b.runID == "" {
		return ErrControlUnknown
	}
	return nil
}
func (b *temporalBootstrap) verifyCoordinator(ctx context.Context) error {
	desc, err := b.client.DescribeWorkflowExecution(ctx, CoordinatorID, b.runID)
	if err != nil || desc == nil || desc.WorkflowExecutionInfo == nil || desc.WorkflowExecutionInfo.Type == nil || desc.WorkflowExecutionInfo.Type.Name != CoordinatorWorkflowName || desc.WorkflowExecutionInfo.Status != enums.WORKFLOW_EXECUTION_STATUS_RUNNING {
		return ErrControlUnknown
	}
	return nil
}
func (b *temporalBootstrap) unpause(ctx context.Context) error {
	if b.handle == nil {
		return ErrControlUnknown
	}
	return b.handle.Unpause(ctx, client.ScheduleUnpauseOptions{Note: "explicit verified initial activation"})
}
func (b *temporalBootstrap) initialTick(ctx context.Context) error {
	return b.client.SignalWorkflow(ctx, CoordinatorID, b.runID, TickSignal, struct{}{})
}

// BootstrapFresh is a deliberate state-changing operation only for a fresh,
// separately authorized environment. Any ambiguous step remains UNKNOWN and
// must never be blindly retried or interpreted as a fresh installation.
func BootstrapFresh(ctx context.Context, c client.Client) error {
	if c == nil {
		return ErrControlUnknown
	}
	return bootstrap(ctx, &temporalBootstrap{client: c})
}

// PauseSchedule stops future ticks but cannot prove all workers/transactions
// have stopped. Even successful pause returns UNKNOWN, not global success.
func PauseSchedule(ctx context.Context, c client.Client) error {
	if ctx == nil || c == nil {
		return ErrControlUnknown
	}
	if err := c.ScheduleClient().GetHandle(ctx, ScheduleID).Pause(ctx, client.SchedulePauseOptions{Note: "stop requested; in-flight state requires reconciliation"}); err != nil {
		return ErrControlUnknown
	}
	return ErrControlUnknown
}

// RequestStop pauses future schedule ticks and asks the one durable
// coordinator to drain. Signal acknowledgement is not proof of quiescence.
func RequestStop(ctx context.Context, c client.Client) error {
	if ctx == nil || c == nil {
		return ErrControlUnknown
	}
	if err := c.ScheduleClient().GetHandle(ctx, ScheduleID).Pause(ctx, client.SchedulePauseOptions{Note: "stop requested; draining original publication"}); err != nil {
		return ErrControlUnknown
	}
	if err := c.SignalWorkflow(ctx, CoordinatorID, "", StopSignal, struct{}{}); err != nil {
		return ErrControlUnknown
	}
	return ErrControlUnknown
}

// CheckStop proves only that the existing running coordinator entered its
// permanent no-automatic-writes state. It does not assert workers exited.
func CheckStop(ctx context.Context, c client.Client) (bool, error) {
	if c == nil {
		return false, ErrControlUnknown
	}
	return checkStop(ctx, c)
}

type stopQueryClient interface {
	DescribeWorkflowExecution(context.Context, string, string) (*servicepb.DescribeWorkflowExecutionResponse, error)
	QueryWorkflowWithOptions(context.Context, *client.QueryWorkflowWithOptionsRequest) (*client.QueryWorkflowWithOptionsResponse, error)
}

func checkStop(ctx context.Context, c stopQueryClient) (bool, error) {
	state, err := readStatus(ctx, c)
	if err == nil && state.State == "QUIESCED" && state.StopRequested {
		return true, nil
	}
	return false, ErrControlUnknown
}

func readStatus(ctx context.Context, c stopQueryClient) (StatusSnapshot, error) {
	var state StatusSnapshot
	if ctx == nil || c == nil {
		return state, ErrControlUnknown
	}
	desc, err := c.DescribeWorkflowExecution(ctx, CoordinatorID, "")
	if err != nil || desc == nil || desc.WorkflowExecutionInfo == nil || desc.WorkflowExecutionInfo.Execution == nil || desc.WorkflowExecutionInfo.Type == nil || desc.WorkflowExecutionInfo.Type.Name != CoordinatorWorkflowName || desc.WorkflowExecutionInfo.Status != enums.WORKFLOW_EXECUTION_STATUS_RUNNING {
		return state, ErrControlUnknown
	}
	runID := desc.WorkflowExecutionInfo.Execution.RunId
	if runID == "" {
		return state, ErrControlUnknown
	}
	response, err := c.QueryWorkflowWithOptions(ctx, &client.QueryWorkflowWithOptionsRequest{WorkflowID: CoordinatorID, RunID: runID, QueryType: "status", QueryRejectCondition: enums.QUERY_REJECT_CONDITION_NOT_OPEN})
	if err != nil || response == nil || response.QueryRejected != nil || response.QueryResult == nil {
		return state, ErrControlUnknown
	}
	if err = response.QueryResult.Get(&state); err != nil {
		return state, ErrControlUnknown
	}
	return state, nil
}

// Control exposes only fixed operations; callers cannot choose actors,
// endpoints, workflow identifiers, or publication parameters.
type Control struct{ Client client.Client }

func (c Control) Status(ctx context.Context) (StatusSnapshot, error) {
	if c.Client == nil {
		return StatusSnapshot{}, ErrControlUnknown
	}
	return readStatus(ctx, c.Client)
}
func (c Control) Start(ctx context.Context) error { return BootstrapFresh(ctx, c.Client) }
func (c Control) Stop(ctx context.Context) error  { return RequestStop(ctx, c.Client) }
