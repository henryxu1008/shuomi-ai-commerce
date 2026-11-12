package automaticfxruntime

import (
	"context"
	"encoding/binary"
	"errors"
	"time"

	"github.com/google/uuid"
	commonpb "go.temporal.io/api/common/v1"
	enums "go.temporal.io/api/enums/v1"
	schedulepb "go.temporal.io/api/schedule/v1"
	"go.temporal.io/api/serviceerror"
	workflowpb "go.temporal.io/api/workflow/v1"
	servicepb "go.temporal.io/api/workflowservice/v1"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/proto"
)

const preparedNote = "platform-fx-v1:prepared"
const claimedNote = "platform-fx-v1:claimed"

type preparedService interface {
	DescribeSchedule(context.Context, *servicepb.DescribeScheduleRequest, ...grpc.CallOption) (*servicepb.DescribeScheduleResponse, error)
	UpdateSchedule(context.Context, *servicepb.UpdateScheduleRequest, ...grpc.CallOption) (*servicepb.UpdateScheduleResponse, error)
	DescribeWorkflowExecution(context.Context, *servicepb.DescribeWorkflowExecutionRequest, ...grpc.CallOption) (*servicepb.DescribeWorkflowExecutionResponse, error)
}

func validatePrepared(s *schedulepb.Schedule) bool {
	if s == nil || s.State == nil || !s.State.Paused || s.State.Notes != preparedNote || s.State.LimitedActions || s.State.RemainingActions != 0 || s.Spec == nil || s.Policies == nil || s.Policies.OverlapPolicy != enums.SCHEDULE_OVERLAP_POLICY_SKIP {
		return false
	}
	spec := proto.Clone(s.Spec).(*schedulepb.ScheduleSpec)
	if len(spec.Interval) != 1 || spec.Interval[0].GetInterval().AsDuration() != 7*24*time.Hour || spec.Interval[0].GetPhase().AsDuration() != 0 {
		return false
	}
	spec.Interval = nil
	if spec.GetJitter().AsDuration() == 0 {
		spec.Jitter = nil
	}
	if spec.TimezoneName == "UTC" {
		spec.TimezoneName = ""
	}
	if !proto.Equal(spec, &schedulepb.ScheduleSpec{}) {
		return false
	}
	if s.Policies.PauseOnFailure || s.Policies.KeepOriginalWorkflowId || (s.Policies.CatchupWindow != nil && s.Policies.CatchupWindow.AsDuration() != 365*24*time.Hour) {
		return false
	}
	action := s.GetAction().GetStartWorkflow()
	if action == nil || action.WorkflowId != "platform-fx-tick-v1" || action.GetWorkflowType().GetName() != TickWorkflowName || action.GetTaskQueue().GetName() != TaskQueue || len(action.GetInput().GetPayloads()) != 0 || len(action.GetHeader().GetFields()) != 0 || action.GetWorkflowExecutionTimeout().AsDuration() != 0 || action.GetWorkflowRunTimeout().AsDuration() != 0 || action.GetWorkflowTaskTimeout().AsDuration() != 0 {
		return false
	}
	queue := action.GetTaskQueue()
	if (queue.Kind != enums.TASK_QUEUE_KIND_UNSPECIFIED && queue.Kind != enums.TASK_QUEUE_KIND_NORMAL) || queue.NormalName != "" {
		return false
	}
	remaining := proto.Clone(action).(*workflowpb.NewWorkflowExecutionInfo)
	remaining.WorkflowId = ""
	remaining.WorkflowType = nil
	remaining.TaskQueue = nil
	remaining.Input = nil
	remaining.Header = nil
	remaining.WorkflowExecutionTimeout = nil
	remaining.WorkflowRunTimeout = nil
	remaining.WorkflowTaskTimeout = nil
	return proto.Equal(remaining, &workflowpb.NewWorkflowExecutionInfo{})
}

func validStatusSchedule(s *schedulepb.Schedule, state StatusSnapshot) bool {
	if s == nil || s.State == nil {
		return false
	}
	if state.StopRequested {
		if !s.State.Paused || s.State.Notes != "stop requested; draining original publication" {
			return false
		}
	} else if s.State.Paused || s.State.Notes != "explicit verified initial activation" {
		return false
	}
	copy := proto.Clone(s).(*schedulepb.Schedule)
	copy.State.Paused = true
	copy.State.Notes = preparedNote
	return validatePrepared(copy)
}

func preparedDescription(ctx context.Context, svc preparedService, namespace string) (*servicepb.DescribeScheduleResponse, error) {
	if ctx == nil || svc == nil || namespace == "" {
		return nil, ErrControlUnknown
	}
	_, err := svc.DescribeWorkflowExecution(ctx, &servicepb.DescribeWorkflowExecutionRequest{Namespace: namespace, Execution: &commonpb.WorkflowExecution{WorkflowId: CoordinatorID}})
	var missing *serviceerror.NotFound
	if !errors.As(err, &missing) {
		return nil, ErrControlUnknown
	}
	desc, err := svc.DescribeSchedule(ctx, &servicepb.DescribeScheduleRequest{Namespace: namespace, ScheduleId: ScheduleID})
	if err != nil || desc == nil || len(desc.ConflictToken) != 8 || binary.BigEndian.Uint64(desc.ConflictToken) == 0 || !validatePrepared(desc.Schedule) {
		return nil, ErrControlUnknown
	}
	return desc, nil
}

func claimPrepared(ctx context.Context, svc preparedService, namespace string) error {
	desc, err := preparedDescription(ctx, svc, namespace)
	if err != nil {
		return ErrControlUnknown
	}
	proposed := proto.Clone(desc.Schedule).(*schedulepb.Schedule)
	claim := claimedNote + ":" + uuid.NewString()
	proposed.State.Notes = claim
	_, err = svc.UpdateSchedule(ctx, &servicepb.UpdateScheduleRequest{Namespace: namespace, ScheduleId: ScheduleID, Schedule: proposed, ConflictToken: desc.ConflictToken, RequestId: uuid.NewString()})
	if err != nil {
		return ErrControlUnknown
	}
	// UpdateSchedule acknowledges acceptance; the scheduler may discard a stale
	// update later. Only the exact claim read back proves this caller won.
	for deadline := time.Now().Add(3 * time.Second); time.Now().Before(deadline); {
		current, e := svc.DescribeSchedule(ctx, &servicepb.DescribeScheduleRequest{Namespace: namespace, ScheduleId: ScheduleID})
		if e != nil || current == nil || current.Schedule == nil || current.Schedule.State == nil {
			return ErrControlUnknown
		}
		if current.Schedule.State.Notes == claim {
			copy := proto.Clone(current.Schedule).(*schedulepb.Schedule)
			copy.State.Notes = preparedNote
			if len(current.ConflictToken) != 8 || binary.BigEndian.Uint64(current.ConflictToken) == 0 || binary.BigEndian.Uint64(current.ConflictToken) == binary.BigEndian.Uint64(desc.ConflictToken) || !validatePrepared(copy) {
				return ErrControlUnknown
			}
			return nil
		}
		if current.Schedule.State.Notes != preparedNote {
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
