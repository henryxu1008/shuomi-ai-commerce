package automaticfxruntime

import (
	"context"
	"encoding/binary"
	"errors"
	"strings"
	"testing"
	"time"

	commonpb "go.temporal.io/api/common/v1"
	enums "go.temporal.io/api/enums/v1"
	schedulepb "go.temporal.io/api/schedule/v1"
	"go.temporal.io/api/serviceerror"
	taskqueuepb "go.temporal.io/api/taskqueue/v1"
	workflowpb "go.temporal.io/api/workflow/v1"
	servicepb "go.temporal.io/api/workflowservice/v1"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/durationpb"
)

type fakePreparedService struct {
	schedule         *schedulepb.Schedule
	token            []byte
	updates          int
	coordinatorErr   error
	updateErr        error
	applyBeforeError bool
}

func (f *fakePreparedService) DescribeSchedule(context.Context, *servicepb.DescribeScheduleRequest, ...grpc.CallOption) (*servicepb.DescribeScheduleResponse, error) {
	return &servicepb.DescribeScheduleResponse{Schedule: f.schedule, ConflictToken: f.token}, nil
}
func (f *fakePreparedService) DescribeWorkflowExecution(context.Context, *servicepb.DescribeWorkflowExecutionRequest, ...grpc.CallOption) (*servicepb.DescribeWorkflowExecutionResponse, error) {
	return nil, f.coordinatorErr
}
func (f *fakePreparedService) UpdateSchedule(_ context.Context, r *servicepb.UpdateScheduleRequest, _ ...grpc.CallOption) (*servicepb.UpdateScheduleResponse, error) {
	f.updates++
	if len(r.ConflictToken) != 8 || binary.BigEndian.Uint64(r.ConflictToken) != 1 || !strings.HasPrefix(r.Schedule.GetState().GetNotes(), claimedNote+":") {
		return nil, errors.New("missing CAS or claim")
	}
	if f.updateErr == nil || f.applyBeforeError {
		f.schedule = r.Schedule
		f.token = []byte{0, 0, 0, 0, 0, 0, 0, 2}
	}
	return &servicepb.UpdateScheduleResponse{}, f.updateErr
}
func TestClaimLostAcknowledgementDoesNotConfirmOrRetry(t *testing.T) {
	f := &fakePreparedService{schedule: preparedFixture(), token: []byte{0, 0, 0, 0, 0, 0, 0, 1}, coordinatorErr: serviceerror.NewNotFound("absent"), updateErr: errors.New("lost"), applyBeforeError: true}
	if err := claimPrepared(context.Background(), f, "default"); !errors.Is(err, ErrControlUnknown) {
		t.Fatalf("lost acknowledgement=%v", err)
	}
	if err := claimPrepared(context.Background(), f, "default"); !errors.Is(err, ErrControlUnknown) || f.updates != 1 {
		t.Fatalf("unexpected retry err=%v updates=%d", err, f.updates)
	}
}

func TestRunningStatusRequiresValidMatchingSchedule(t *testing.T) {
	s := preparedFixture()
	s.State.Paused = false
	s.State.Notes = "explicit verified initial activation"
	if !validStatusSchedule(s, StatusSnapshot{State: "WAITING"}) {
		t.Fatal("valid active schedule rejected")
	}
	s.State.Paused = true
	if validStatusSchedule(s, StatusSnapshot{State: "WAITING"}) {
		t.Fatal("paused schedule reported running")
	}
	s.State.Notes = "stop requested; draining original publication"
	if !validStatusSchedule(s, StatusSnapshot{State: "QUIESCED", StopRequested: true}) {
		t.Fatal("verified stop rejected")
	}
	s.Spec.Interval[0].Interval = durationpb.New(time.Hour)
	if validStatusSchedule(s, StatusSnapshot{State: "QUIESCED", StopRequested: true}) {
		t.Fatal("changed schedule reported as verified")
	}
}

func TestStartupRejectsStoppedCoordinator(t *testing.T) {
	if !eligibleStartup(StatusSnapshot{State: "WAITING"}) {
		t.Fatal("active coordinator rejected")
	}
	for _, s := range []StatusSnapshot{{State: "QUIESCED", StopRequested: true}, {State: "DRAINING", StopRequested: true}, {State: "QUIESCED"}, {State: ""}} {
		if eligibleStartup(s) {
			t.Fatalf("ineligible state confirmed: %+v", s)
		}
	}
}
func preparedFixture() *schedulepb.Schedule {
	return &schedulepb.Schedule{Spec: &schedulepb.ScheduleSpec{Interval: []*schedulepb.IntervalSpec{{Interval: durationpb.New(7 * 24 * time.Hour)}}}, Action: &schedulepb.ScheduleAction{Action: &schedulepb.ScheduleAction_StartWorkflow{StartWorkflow: &workflowpb.NewWorkflowExecutionInfo{WorkflowId: "platform-fx-tick-v1", WorkflowType: &commonpb.WorkflowType{Name: TickWorkflowName}, TaskQueue: &taskqueuepb.TaskQueue{Name: TaskQueue}}}}, Policies: &schedulepb.SchedulePolicies{OverlapPolicy: enums.SCHEDULE_OVERLAP_POLICY_SKIP}, State: &schedulepb.ScheduleState{Paused: true, Notes: preparedNote}}
}
func TestClaimPreparedUsesConflictTokenAndFailsClosed(t *testing.T) {
	for _, tc := range []struct {
		name   string
		change func(*fakePreparedService)
		ok     bool
	}{
		{"prepared", func(*fakePreparedService) {}, true},
		{"missing token", func(f *fakePreparedService) { f.token = nil }, false},
		{"zero token", func(f *fakePreparedService) { f.token = make([]byte, 8) }, false},
		{"coordinator unknown", func(f *fakePreparedService) { f.coordinatorErr = errors.New("unavailable") }, false},
		{"already claimed", func(f *fakePreparedService) { f.schedule.State.Notes = claimedNote }, false},
		{"unpaused", func(f *fakePreparedService) { f.schedule.State.Paused = false }, false},
		{"wrong interval", func(f *fakePreparedService) { f.schedule.Spec.Interval[0].Interval = durationpb.New(time.Hour) }, false},
		{"sticky queue", func(f *fakePreparedService) {
			f.schedule.GetAction().GetStartWorkflow().TaskQueue.Kind = enums.TASK_QUEUE_KIND_STICKY
		}, false},
		{"CAS response lost", func(f *fakePreparedService) { f.updateErr = errors.New("lost") }, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := &fakePreparedService{schedule: preparedFixture(), token: []byte{0, 0, 0, 0, 0, 0, 0, 1}, coordinatorErr: serviceerror.NewNotFound("absent")}
			tc.change(f)
			err := claimPrepared(context.Background(), f, "default")
			if (err == nil) != tc.ok {
				t.Fatalf("claim result %v", err)
			}
			if !tc.ok && tc.name != "CAS response lost" && f.updates != 0 {
				t.Fatal("invalid state reached update")
			}
		})
	}
}
