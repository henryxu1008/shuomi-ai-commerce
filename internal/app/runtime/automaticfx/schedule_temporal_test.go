package automaticfxruntime

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/worker"
	"go.temporal.io/sdk/workflow"
)

type terminateBeforeQuery struct{ client.Client }

func (c terminateBeforeQuery) QueryWorkflowWithOptions(ctx context.Context, req *client.QueryWorkflowWithOptionsRequest) (*client.QueryWorkflowWithOptionsResponse, error) {
	if err := c.TerminateWorkflow(ctx, req.WorkflowID, req.RunID, "synthetic race"); err != nil {
		return nil, err
	}
	return c.Client.QueryWorkflowWithOptions(ctx, req)
}

func TestAutomaticFXIsolatedTemporalRejectsClosedQuery(t *testing.T) {
	addr := os.Getenv("FX_ISOLATED_TEMPORAL")
	if addr == "" {
		t.Skip("requires explicitly isolated Temporal")
	}
	require.Equal(t, "127.0.0.1:7233", addr)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	c, err := client.Dial(client.Options{HostPort: addr, Namespace: "default"})
	require.NoError(t, err)
	defer c.Close()
	w := worker.New(c, TaskQueue, worker.Options{})
	w.RegisterWorkflowWithOptions(CoordinatorWorkflow, workflow.RegisterOptions{Name: CoordinatorWorkflowName})
	require.NoError(t, w.Start())
	defer w.Stop()
	_, err = c.ExecuteWorkflow(ctx, client.StartWorkflowOptions{ID: CoordinatorID, TaskQueue: TaskQueue}, CoordinatorWorkflowName, CoordinatorState{StopRequested: true})
	require.NoError(t, err)
	require.Eventually(t, func() bool { ok, _ := CheckStop(ctx, c); return ok }, 10*time.Second, 100*time.Millisecond)
	stopped, err := checkStop(ctx, terminateBeforeQuery{Client: c})
	require.False(t, stopped)
	require.ErrorIs(t, err, ErrControlUnknown)
}

func TestAutomaticFXIsolatedTemporalStopSurvivesContinueAsNew(t *testing.T) {
	addr := os.Getenv("FX_ISOLATED_TEMPORAL")
	if addr == "" {
		t.Skip("requires explicitly isolated Temporal")
	}
	require.Equal(t, "127.0.0.1:7233", addr)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	c, err := client.Dial(client.Options{HostPort: addr, Namespace: "default"})
	require.NoError(t, err)
	defer c.Close()
	w := worker.New(c, TaskQueue, worker.Options{})
	w.RegisterWorkflowWithOptions(CoordinatorWorkflow, workflow.RegisterOptions{Name: CoordinatorWorkflowName})
	require.NoError(t, w.Start())
	defer w.Stop()
	run, err := c.ExecuteWorkflow(ctx, client.StartWorkflowOptions{ID: CoordinatorID, TaskQueue: TaskQueue}, CoordinatorWorkflowName, CoordinatorState{StopRequested: true})
	require.NoError(t, err)
	for i := 0; i < 110; i++ {
		require.NoError(t, c.SignalWorkflow(ctx, CoordinatorID, "", TickSignal, struct{}{}))
		if i == 99 {
			require.NoError(t, c.SignalWorkflow(ctx, CoordinatorID, "", StopSignal, struct{}{}))
		}
	}
	require.Eventually(t, func() bool {
		desc, e := c.DescribeWorkflowExecution(ctx, CoordinatorID, "")
		if e != nil || desc.WorkflowExecutionInfo.Execution.RunId == run.GetRunID() {
			return false
		}
		stopped, _ := CheckStop(ctx, c)
		return stopped
	}, 20*time.Second, 100*time.Millisecond)
	require.NoError(t, c.TerminateWorkflow(ctx, CoordinatorID, "", "synthetic test cleanup"))
}
