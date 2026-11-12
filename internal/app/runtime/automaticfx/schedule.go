package automaticfxruntime

import (
	"context"
	"errors"
	"time"

	commonpb "go.temporal.io/api/common/v1"
	enums "go.temporal.io/api/enums/v1"
	"go.temporal.io/api/serviceerror"
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
	confirmRunning(context.Context) error
}

func bootstrap(ctx context.Context, ops bootstrapOperations) error {
	if ctx == nil || ops == nil {
		return ErrControlUnknown
	}
	for _, step := range []func(context.Context) error{ops.createPaused, ops.startCoordinator, ops.verifyCoordinator, ops.unpause, ops.initialTick, ops.confirmRunning} {
		if step(ctx) != nil {
			return ErrControlUnknown
		}
	}
	return nil
}

type temporalBootstrap struct {
	client    client.Client
	namespace string
	handle    client.ScheduleHandle
	runID     string
}

func (b *temporalBootstrap) createPaused(ctx context.Context) error {
	if err := claimPrepared(ctx, b.client.WorkflowService(), b.namespace); err != nil {
		return ErrControlUnknown
	}
	b.handle = b.client.ScheduleClient().GetHandle(ctx, ScheduleID)
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

func eligibleStartup(s StatusSnapshot) bool {
	return !s.StopRequested && s.State != "" && s.State != "QUIESCED" && s.State != "DRAINING" && s.State != "UNKNOWN" && s.State != "USD_CONFLICT"
}

func (b *temporalBootstrap) confirmRunning(ctx context.Context) error {
	for deadline := time.Now().Add(3 * time.Second); time.Now().Before(deadline); {
		desc, err := b.client.WorkflowService().DescribeSchedule(ctx, &servicepb.DescribeScheduleRequest{Namespace: b.namespace, ScheduleId: ScheduleID})
		if err != nil || desc == nil || desc.Schedule == nil || desc.Schedule.State == nil {
			return ErrControlUnknown
		}
		if validStatusSchedule(desc.Schedule, StatusSnapshot{}) {
			if err = b.verifyCoordinator(ctx); err != nil {
				return ErrControlUnknown
			}
			state, queryErr := readStatus(ctx, b.client)
			if queryErr != nil {
				return ErrControlUnknown
			}
			if state.StopRequested || state.State == "QUIESCED" || state.State == "DRAINING" {
				return ErrControlUnknown
			}
			if eligibleStartup(state) {
				if err = b.verifyCoordinator(ctx); err != nil {
					return ErrControlUnknown
				}
				return nil
			}
			if state.State != "" {
				return ErrControlUnknown
			}
		} else if !desc.Schedule.State.Paused {
			return ErrControlUnknown
		}
		timer := time.NewTimer(100 * time.Millisecond)
		select {
		case <-ctx.Done():
			timer.Stop()
			return ErrControlUnknown
		case <-timer.C:
		}
	}
	return ErrControlUnknown
}

// BootstrapFresh is a deliberate state-changing operation only for a fresh,
// separately authorized environment. Any ambiguous step remains UNKNOWN and
// must never be blindly retried or interpreted as a fresh installation.
func BootstrapFresh(ctx context.Context, c client.Client, namespace string) error {
	if c == nil || namespace == "" {
		return ErrControlUnknown
	}
	return bootstrap(ctx, &temporalBootstrap{client: c, namespace: namespace})
}

// PrepareFresh creates the one durable paused marker only in a fresh target.
// Any ambiguous outcome must be inspected and never blindly retried.
func PrepareFresh(ctx context.Context, c client.Client, namespace string) error {
	if ctx == nil || c == nil || namespace == "" {
		return ErrControlUnknown
	}
	_, err := c.WorkflowService().DescribeWorkflowExecution(ctx, &servicepb.DescribeWorkflowExecutionRequest{Namespace: namespace, Execution: &commonpb.WorkflowExecution{WorkflowId: CoordinatorID}})
	var missing *serviceerror.NotFound
	if !errors.As(err, &missing) {
		return ErrControlUnknown
	}
	_, err = c.ScheduleClient().Create(ctx, client.ScheduleOptions{ID: ScheduleID, Paused: true, Note: preparedNote, Overlap: enums.SCHEDULE_OVERLAP_POLICY_SKIP, Spec: client.ScheduleSpec{Intervals: []client.ScheduleIntervalSpec{{Every: 7 * 24 * time.Hour}}}, Action: &client.ScheduleWorkflowAction{ID: "platform-fx-tick-v1", Workflow: TickWorkflowName, TaskQueue: TaskQueue}})
	if err != nil {
		return ErrControlUnknown
	}
	return nil
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
type Control struct {
	Client    client.Client
	Namespace string
}

func (c Control) Status(ctx context.Context) (StatusSnapshot, error) {
	if c.Client == nil || c.Namespace == "" {
		return StatusSnapshot{}, ErrControlUnknown
	}
	if _, err := preparedDescription(ctx, c.Client.WorkflowService(), c.Namespace); err == nil {
		return StatusSnapshot{State: "PREPARED"}, nil
	}
	desc, err := c.Client.WorkflowService().DescribeSchedule(ctx, &servicepb.DescribeScheduleRequest{Namespace: c.Namespace, ScheduleId: ScheduleID})
	if err != nil || desc == nil {
		return StatusSnapshot{}, ErrControlUnknown
	}
	state, err := readStatus(ctx, c.Client)
	if err != nil || !validStatusSchedule(desc.Schedule, state) {
		return StatusSnapshot{}, ErrControlUnknown
	}
	return state, nil
}
func (c Control) Prepare(ctx context.Context) error { return PrepareFresh(ctx, c.Client, c.Namespace) }
func (c Control) Start(ctx context.Context) error   { return BootstrapFresh(ctx, c.Client, c.Namespace) }
func (c Control) Stop(ctx context.Context) error    { return RequestStop(ctx, c.Client) }
