package automaticfxruntime

import (
	"context"
	"errors"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	enums "go.temporal.io/api/enums/v1"
	schedulepb "go.temporal.io/api/schedule/v1"
	"go.temporal.io/api/serviceerror"
	servicepb "go.temporal.io/api/workflowservice/v1"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/worker"
	"go.temporal.io/sdk/workflow"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/proto"
)

type realUpdateLost struct{ preparedService }

type lostScheduleCreate struct{ client.ScheduleClient }

func (l lostScheduleCreate) Create(ctx context.Context, options client.ScheduleOptions) (client.ScheduleHandle, error) {
	h, err := l.ScheduleClient.Create(ctx, options)
	if err != nil {
		return h, err
	}
	return nil, errors.New("synthetic lost prepare acknowledgement")
}

type realPrepareLost struct{ client.Client }

func (l realPrepareLost) ScheduleClient() client.ScheduleClient {
	return lostScheduleCreate{l.Client.ScheduleClient()}
}

func TestIsolatedLostPrepareAcknowledgementNeedsReadOnlyVerification(t *testing.T) {
	if os.Getenv("FX_PREPARED_ISOLATED") != "1" {
		t.Skip("requires dedicated isolated Temporal")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	c, err := client.Dial(client.Options{HostPort: "127.0.0.1:7233", Namespace: "default"})
	require.NoError(t, err)
	defer c.Close()
	err = PrepareFresh(ctx, realPrepareLost{c}, "default")
	require.ErrorIs(t, err, ErrControlUnknown)
	_, err = preparedDescription(ctx, c.WorkflowService(), "default")
	require.NoError(t, err, "the original marker is established by read-only verification")
	require.ErrorIs(t, PrepareFresh(ctx, c, "default"), ErrControlUnknown, "never create another schedule")
	_, err = c.DescribeWorkflowExecution(ctx, CoordinatorID, "")
	var missing *serviceerror.NotFound
	require.ErrorAs(t, err, &missing)
}

func (l realUpdateLost) UpdateSchedule(ctx context.Context, req *servicepb.UpdateScheduleRequest, opts ...grpc.CallOption) (*servicepb.UpdateScheduleResponse, error) {
	response, err := l.preparedService.UpdateSchedule(ctx, req, opts...)
	if err != nil {
		return response, err
	}
	return nil, errors.New("synthetic lost update acknowledgement")
}

func TestIsolatedLostClaimAcknowledgementIsNotReplayed(t *testing.T) {
	if os.Getenv("FX_PREPARED_ISOLATED") != "1" {
		t.Skip("requires dedicated isolated Temporal")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	c, err := client.Dial(client.Options{HostPort: "127.0.0.1:7233", Namespace: "default"})
	require.NoError(t, err)
	defer c.Close()
	require.NoError(t, PrepareFresh(ctx, c, "default"))
	err = claimPrepared(ctx, realUpdateLost{c.WorkflowService()}, "default")
	require.ErrorIs(t, err, ErrControlUnknown)
	require.Eventually(t, func() bool {
		d, e := c.WorkflowService().DescribeSchedule(ctx, &servicepb.DescribeScheduleRequest{Namespace: "default", ScheduleId: ScheduleID})
		return e == nil && strings.HasPrefix(d.Schedule.State.Notes, claimedNote+":")
	}, 5*time.Second, 100*time.Millisecond)
	require.ErrorIs(t, claimPrepared(ctx, c.WorkflowService(), "default"), ErrControlUnknown)
	_, err = c.DescribeWorkflowExecution(ctx, CoordinatorID, "")
	var missing *serviceerror.NotFound
	require.ErrorAs(t, err, &missing)
	d, err := c.ScheduleClient().GetHandle(ctx, ScheduleID).Describe(ctx)
	require.NoError(t, err)
	require.True(t, d.Schedule.State.Paused)
}

type realStartLost struct{ client.Client }

func (l realStartLost) ExecuteWorkflow(ctx context.Context, options client.StartWorkflowOptions, w interface{}, args ...interface{}) (client.WorkflowRun, error) {
	run, err := l.Client.ExecuteWorkflow(ctx, options, w, args...)
	if err != nil {
		return run, err
	}
	return nil, errors.New("synthetic lost coordinator acknowledgement")
}

type lostLateStep struct {
	*temporalBootstrap
	stage        string
	initialCalls int
}

func (b *lostLateStep) unpause(ctx context.Context) error {
	err := b.temporalBootstrap.unpause(ctx)
	if err == nil && b.stage == "unpause" {
		return errors.New("synthetic lost unpause acknowledgement")
	}
	return err
}
func (b *lostLateStep) initialTick(ctx context.Context) error {
	b.initialCalls++
	err := b.temporalBootstrap.initialTick(ctx)
	if err == nil && b.stage == "signal" {
		return errors.New("synthetic lost signal acknowledgement")
	}
	return err
}
func exerciseIsolatedLostLateAcknowledgement(t *testing.T, stage string) {
	if os.Getenv("FX_PREPARED_ISOLATED") != "1" {
		t.Skip("requires dedicated isolated Temporal")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	c, err := client.Dial(client.Options{HostPort: "127.0.0.1:7233", Namespace: "default"})
	require.NoError(t, err)
	defer c.Close()
	require.NoError(t, PrepareFresh(ctx, c, "default"))
	b := &lostLateStep{temporalBootstrap: &temporalBootstrap{client: c, namespace: "default"}, stage: stage}
	require.ErrorIs(t, bootstrap(ctx, b), ErrControlUnknown)
	first, err := c.DescribeWorkflowExecution(ctx, CoordinatorID, "")
	require.NoError(t, err)
	runID := first.WorkflowExecutionInfo.Execution.RunId
	require.NotEmpty(t, runID)
	if stage == "unpause" {
		require.Zero(t, b.initialCalls)
	} else {
		require.Equal(t, 1, b.initialCalls)
	}
	require.Eventually(t, func() bool {
		d, e := c.ScheduleClient().GetHandle(ctx, ScheduleID).Describe(ctx)
		return e == nil && !d.Schedule.State.Paused
	}, 5*time.Second, 100*time.Millisecond)
	require.ErrorIs(t, BootstrapFresh(ctx, c, "default"), ErrControlUnknown)
	second, err := c.DescribeWorkflowExecution(ctx, CoordinatorID, "")
	require.NoError(t, err)
	require.Equal(t, runID, second.WorkflowExecutionInfo.Execution.RunId)
	wantSignals := 0
	if stage == "signal" {
		wantSignals = 1
	}
	require.Eventually(t, func() bool {
		it := c.GetWorkflowHistory(ctx, CoordinatorID, runID, false, enums.HISTORY_EVENT_FILTER_TYPE_ALL_EVENT)
		count := 0
		for it.HasNext() {
			event, e := it.Next()
			if e != nil {
				return false
			}
			if signal := event.GetWorkflowExecutionSignaledEventAttributes(); signal != nil && signal.SignalName == TickSignal {
				count++
			}
		}
		return count == wantSignals
	}, 5*time.Second, 100*time.Millisecond, "the original run must not receive a duplicate initial signal")
}
func TestIsolatedLostUnpauseAcknowledgementDoesNotRestart(t *testing.T) {
	exerciseIsolatedLostLateAcknowledgement(t, "unpause")
}
func TestIsolatedLostInitialSignalAcknowledgementDoesNotRestart(t *testing.T) {
	exerciseIsolatedLostLateAcknowledgement(t, "signal")
}

func TestIsolatedLostCoordinatorAcknowledgementDoesNotUnpauseOrRestart(t *testing.T) {
	if os.Getenv("FX_PREPARED_ISOLATED") != "1" {
		t.Skip("requires dedicated isolated Temporal")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	c, err := client.Dial(client.Options{HostPort: "127.0.0.1:7233", Namespace: "default"})
	require.NoError(t, err)
	defer c.Close()
	require.NoError(t, PrepareFresh(ctx, c, "default"))
	err = BootstrapFresh(ctx, realStartLost{c}, "default")
	require.ErrorIs(t, err, ErrControlUnknown)
	first, err := c.DescribeWorkflowExecution(ctx, CoordinatorID, "")
	require.NoError(t, err)
	runID := first.WorkflowExecutionInfo.Execution.RunId
	require.NotEmpty(t, runID)
	d, err := c.ScheduleClient().GetHandle(ctx, ScheduleID).Describe(ctx)
	require.NoError(t, err)
	require.True(t, d.Schedule.State.Paused)
	require.ErrorIs(t, BootstrapFresh(ctx, c, "default"), ErrControlUnknown)
	second, err := c.DescribeWorkflowExecution(ctx, CoordinatorID, "")
	require.NoError(t, err)
	require.Equal(t, runID, second.WorkflowExecutionInfo.Execution.RunId)
	d, err = c.ScheduleClient().GetHandle(ctx, ScheduleID).Describe(ctx)
	require.NoError(t, err)
	require.True(t, d.Schedule.State.Paused)
}

func TestIsolatedTemporalRejectsStaleScheduleToken(t *testing.T) {
	if os.Getenv("FX_PREPARED_ISOLATED") != "1" {
		t.Skip("requires dedicated isolated Temporal")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	c, err := client.Dial(client.Options{HostPort: "127.0.0.1:7233", Namespace: "default"})
	require.NoError(t, err)
	defer c.Close()
	id := "cas-probe-" + uuid.NewString()
	_, err = c.ScheduleClient().Create(ctx, client.ScheduleOptions{ID: id, Paused: true, Note: "READY", Overlap: enums.SCHEDULE_OVERLAP_POLICY_SKIP, Spec: client.ScheduleSpec{Intervals: []client.ScheduleIntervalSpec{{Every: 7 * 24 * time.Hour}}}, Action: &client.ScheduleWorkflowAction{ID: "cas-probe-tick", Workflow: TickWorkflowName, TaskQueue: TaskQueue}})
	require.NoError(t, err)
	desc, err := c.WorkflowService().DescribeSchedule(ctx, &servicepb.DescribeScheduleRequest{Namespace: "default", ScheduleId: id})
	require.NoError(t, err)
	require.NotEmpty(t, desc.ConflictToken)
	s := proto.Clone(desc.Schedule).(*schedulepb.Schedule)
	s.State.Notes = "CLAIM-A"
	_, err = c.WorkflowService().UpdateSchedule(ctx, &servicepb.UpdateScheduleRequest{Namespace: "default", ScheduleId: id, Schedule: s, ConflictToken: desc.ConflictToken, RequestId: uuid.NewString()})
	require.NoError(t, err)
	require.Eventually(t, func() bool {
		d, e := c.WorkflowService().DescribeSchedule(ctx, &servicepb.DescribeScheduleRequest{Namespace: "default", ScheduleId: id})
		return e == nil && d.Schedule.State.Notes == "CLAIM-A"
	}, 5*time.Second, 25*time.Millisecond)
	require.NoError(t, c.ScheduleClient().GetHandle(ctx, id).Unpause(ctx, client.ScheduleUnpauseOptions{Note: "ACTIVE"}))
	stale := proto.Clone(desc.Schedule).(*schedulepb.Schedule)
	stale.State.Notes = "CLAIM-B"
	_, err = c.WorkflowService().UpdateSchedule(ctx, &servicepb.UpdateScheduleRequest{Namespace: "default", ScheduleId: id, Schedule: stale, ConflictToken: desc.ConflictToken, RequestId: uuid.NewString()})
	require.NoError(t, err, "RPC acknowledgement alone does not establish ownership")
	for until := time.Now().Add(2 * time.Second); time.Now().Before(until); time.Sleep(50 * time.Millisecond) {
		d, e := c.WorkflowService().DescribeSchedule(ctx, &servicepb.DescribeScheduleRequest{Namespace: "default", ScheduleId: id})
		require.NoError(t, e)
		require.False(t, d.Schedule.State.Paused)
		require.Equal(t, "ACTIVE", d.Schedule.State.Notes)
	}
	require.NoError(t, c.ScheduleClient().GetHandle(ctx, id).Pause(ctx, client.SchedulePauseOptions{Note: "STOP"}))
	_, err = c.WorkflowService().UpdateSchedule(ctx, &servicepb.UpdateScheduleRequest{Namespace: "default", ScheduleId: id, Schedule: stale, ConflictToken: desc.ConflictToken, RequestId: uuid.NewString()})
	require.NoError(t, err)
	for until := time.Now().Add(2 * time.Second); time.Now().Before(until); time.Sleep(50 * time.Millisecond) {
		d, e := c.WorkflowService().DescribeSchedule(ctx, &servicepb.DescribeScheduleRequest{Namespace: "default", ScheduleId: id})
		require.NoError(t, e)
		require.True(t, d.Schedule.State.Paused)
		require.Equal(t, "STOP", d.Schedule.State.Notes)
	}
}

type stopBeforeUnpause struct{ *temporalBootstrap }

func (b *stopBeforeUnpause) unpause(ctx context.Context) error {
	if e := b.client.ScheduleClient().GetHandle(ctx, ScheduleID).Pause(ctx, client.SchedulePauseOptions{Note: "stop requested; draining original publication"}); e != nil {
		return e
	}
	if e := b.client.SignalWorkflow(ctx, CoordinatorID, b.runID, StopSignal, struct{}{}); e != nil {
		return e
	}
	for until := time.Now().Add(5 * time.Second); time.Now().Before(until); time.Sleep(100 * time.Millisecond) {
		s, e := readStatus(ctx, b.client)
		if e == nil && s.StopRequested {
			return b.temporalBootstrap.unpause(ctx)
		}
	}
	return ErrControlUnknown
}

type stopAfterUnpause struct{ *temporalBootstrap }

func (b *stopAfterUnpause) initialTick(ctx context.Context) error {
	_ = RequestStop(ctx, b.client)
	for until := time.Now().Add(5 * time.Second); time.Now().Before(until); time.Sleep(100 * time.Millisecond) {
		s, e := readStatus(ctx, b.client)
		if e == nil && s.StopRequested {
			return b.temporalBootstrap.initialTick(ctx)
		}
	}
	return ErrControlUnknown
}

func TestIsolatedStopAfterUnpauseCannotReportStarted(t *testing.T) {
	if os.Getenv("FX_PREPARED_ISOLATED") != "1" {
		t.Skip("requires dedicated isolated Temporal")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	c, err := client.Dial(client.Options{HostPort: "127.0.0.1:7233", Namespace: "default"})
	require.NoError(t, err)
	defer c.Close()
	w := worker.New(c, TaskQueue, worker.Options{})
	w.RegisterWorkflowWithOptions(CoordinatorWorkflow, workflow.RegisterOptions{Name: CoordinatorWorkflowName})
	require.NoError(t, w.Start())
	defer w.Stop()
	require.NoError(t, PrepareFresh(ctx, c, "default"))
	err = bootstrap(ctx, &stopAfterUnpause{&temporalBootstrap{client: c, namespace: "default"}})
	require.ErrorIs(t, err, ErrControlUnknown)
	require.Eventually(t, func() bool { s, e := readStatus(ctx, c); return e == nil && s.StopRequested && s.State == "QUIESCED" }, 5*time.Second, 100*time.Millisecond)
}

func TestIsolatedStopBeforeUnpauseCannotReportStarted(t *testing.T) {
	if os.Getenv("FX_PREPARED_ISOLATED") != "1" {
		t.Skip("requires dedicated isolated Temporal")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	c, err := client.Dial(client.Options{HostPort: "127.0.0.1:7233", Namespace: "default"})
	require.NoError(t, err)
	defer c.Close()
	w := worker.New(c, TaskQueue, worker.Options{})
	w.RegisterWorkflowWithOptions(CoordinatorWorkflow, workflow.RegisterOptions{Name: CoordinatorWorkflowName})
	require.NoError(t, w.Start())
	defer w.Stop()
	require.NoError(t, PrepareFresh(ctx, c, "default"))
	err = bootstrap(ctx, &stopBeforeUnpause{&temporalBootstrap{client: c, namespace: "default"}})
	require.ErrorIs(t, err, ErrControlUnknown)
	require.Eventually(t, func() bool { s, e := readStatus(ctx, c); return e == nil && s.StopRequested && s.State == "QUIESCED" }, 5*time.Second, 100*time.Millisecond)
}

func TestIsolatedUniqueClaimReadback(t *testing.T) {
	if os.Getenv("FX_PREPARED_ISOLATED") != "1" {
		t.Skip("requires dedicated isolated Temporal")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	c, err := client.Dial(client.Options{HostPort: "127.0.0.1:7233", Namespace: "default"})
	require.NoError(t, err)
	defer c.Close()
	id := "unique-cas-" + uuid.NewString()
	_, err = c.ScheduleClient().Create(ctx, client.ScheduleOptions{ID: id, Paused: true, Note: "READY", Overlap: enums.SCHEDULE_OVERLAP_POLICY_SKIP, Spec: client.ScheduleSpec{Intervals: []client.ScheduleIntervalSpec{{Every: 7 * 24 * time.Hour}}}, Action: &client.ScheduleWorkflowAction{ID: "cas-probe-tick", Workflow: TickWorkflowName, TaskQueue: TaskQueue}})
	require.NoError(t, err)
	d, err := c.WorkflowService().DescribeSchedule(ctx, &servicepb.DescribeScheduleRequest{Namespace: "default", ScheduleId: id})
	require.NoError(t, err)
	require.NotEmpty(t, d.ConflictToken)
	var wg sync.WaitGroup
	results := make([]bool, 2)
	observed := make([]string, 2)
	names := []string{"CLAIM-" + uuid.NewString(), "CLAIM-" + uuid.NewString()}
	start := make(chan struct{})
	for i := range results {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			s := proto.Clone(d.Schedule).(*schedulepb.Schedule)
			s.State.Notes = names[i]
			<-start
			_, e := c.WorkflowService().UpdateSchedule(ctx, &servicepb.UpdateScheduleRequest{Namespace: "default", ScheduleId: id, Schedule: s, ConflictToken: d.ConflictToken, RequestId: uuid.NewString()})
			if e != nil {
				observed[i] = "update error: " + e.Error()
				return
			}
			for until := time.Now().Add(2 * time.Second); time.Now().Before(until); time.Sleep(10 * time.Millisecond) {
				got, e := c.WorkflowService().DescribeSchedule(ctx, &servicepb.DescribeScheduleRequest{Namespace: "default", ScheduleId: id})
				if e != nil {
					observed[i] = "describe error: " + e.Error()
					return
				}
				observed[i] = got.Schedule.State.Notes
				if got.Schedule.State.Notes == names[i] {
					results[i] = true
					return
				}
				if got.Schedule.State.Notes != "READY" {
					return
				}
			}
		}(i)
	}
	close(start)
	wg.Wait()
	winners := 0
	for _, v := range results {
		if v {
			winners++
		}
	}
	t.Logf("claims=%v observed=%v", names, observed)
	require.LessOrEqual(t, winners, 1, "query failures must remain unknown, never grant two owners")
}

func TestIsolatedSingleClaimCanBeConfirmed(t *testing.T) {
	if os.Getenv("FX_PREPARED_ISOLATED") != "1" {
		t.Skip("requires dedicated isolated Temporal")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	c, err := client.Dial(client.Options{HostPort: "127.0.0.1:7233", Namespace: "default"})
	require.NoError(t, err)
	defer c.Close()
	id := "single-claim-" + uuid.NewString()
	want := "CLAIM-" + uuid.NewString()
	_, err = c.ScheduleClient().Create(ctx, client.ScheduleOptions{ID: id, Paused: true, Note: "READY", Overlap: enums.SCHEDULE_OVERLAP_POLICY_SKIP, Spec: client.ScheduleSpec{Intervals: []client.ScheduleIntervalSpec{{Every: 7 * 24 * time.Hour}}}, Action: &client.ScheduleWorkflowAction{ID: "cas-probe-tick", Workflow: TickWorkflowName, TaskQueue: TaskQueue}})
	require.NoError(t, err)
	d, err := c.WorkflowService().DescribeSchedule(ctx, &servicepb.DescribeScheduleRequest{Namespace: "default", ScheduleId: id})
	require.NoError(t, err)
	require.NotEmpty(t, d.ConflictToken)
	s := proto.Clone(d.Schedule).(*schedulepb.Schedule)
	s.State.Notes = want
	_, err = c.WorkflowService().UpdateSchedule(ctx, &servicepb.UpdateScheduleRequest{Namespace: "default", ScheduleId: id, Schedule: s, ConflictToken: d.ConflictToken, RequestId: uuid.NewString()})
	require.NoError(t, err)
	require.Eventually(t, func() bool {
		r, e := c.WorkflowService().DescribeSchedule(ctx, &servicepb.DescribeScheduleRequest{Namespace: "default", ScheduleId: id})
		return e == nil && r.Schedule.State.Notes == want
	}, 5*time.Second, 100*time.Millisecond)
}

func TestPreparedClaimIsAtomicInIsolatedTemporal(t *testing.T) {
	if os.Getenv("FX_PREPARED_ISOLATED") != "1" {
		t.Skip("requires dedicated isolated Temporal")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	c, err := client.Dial(client.Options{HostPort: "127.0.0.1:7233", Namespace: "default"})
	require.NoError(t, err)
	defer c.Close()
	require.NoError(t, PrepareFresh(ctx, c, "default"))
	_, err = preparedDescription(ctx, c.WorkflowService(), "default")
	require.NoError(t, err)
	var wg sync.WaitGroup
	errs := make([]error, 2)
	for i := range errs {
		wg.Add(1)
		go func(i int) { defer wg.Done(); errs[i] = claimPrepared(ctx, c.WorkflowService(), "default") }(i)
	}
	wg.Wait()
	successes := 0
	for _, err := range errs {
		if err == nil {
			successes++
		}
	}
	require.LessOrEqual(t, successes, 1, "concurrent contenders must never both be confirmed")
	require.Eventually(t, func() bool {
		_, err = preparedDescription(ctx, c.WorkflowService(), "default")
		return errors.Is(err, ErrControlUnknown)
	}, 5*time.Second, 100*time.Millisecond)
	require.ErrorIs(t, PrepareFresh(ctx, c, "default"), ErrControlUnknown)
}

func TestIsolatedConcurrentBootstrapNeverCreatesTwoCoordinators(t *testing.T) {
	if os.Getenv("FX_PREPARED_ISOLATED") != "1" {
		t.Skip("requires dedicated isolated Temporal")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	c, err := client.Dial(client.Options{HostPort: "127.0.0.1:7233", Namespace: "default"})
	require.NoError(t, err)
	defer c.Close()
	require.NoError(t, PrepareFresh(ctx, c, "default"))
	var wg sync.WaitGroup
	errs := make([]error, 2)
	start := make(chan struct{})
	for i := range errs {
		wg.Add(1)
		go func(i int) { defer wg.Done(); <-start; errs[i] = BootstrapFresh(ctx, c, "default") }(i)
	}
	close(start)
	wg.Wait()
	winners := 0
	for _, e := range errs {
		if e == nil {
			winners++
		}
	}
	require.LessOrEqual(t, winners, 1)
	t.Logf("confirmed starts=%d; other results remain unknown", winners)
	if winners == 1 {
		d, e := c.DescribeWorkflowExecution(ctx, CoordinatorID, "")
		require.NoError(t, e)
		require.NotEmpty(t, d.GetWorkflowExecutionInfo().GetExecution().GetRunId())
		s, e := c.ScheduleClient().GetHandle(ctx, ScheduleID).Describe(ctx)
		require.NoError(t, e)
		require.False(t, s.Schedule.State.Paused)
	}
	require.ErrorIs(t, BootstrapFresh(ctx, c, "default"), ErrControlUnknown)
}
