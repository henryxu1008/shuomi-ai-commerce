package fulfillmentruntime

import (
	"context"
	"errors"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/testsuite"
	"testing"
)

func TestMutationActivityHasExactlyOneAttempt(t *testing.T) {
	var suite testsuite.WorkflowTestSuite
	environment := suite.NewTestWorkflowEnvironment()
	calls := 0
	environment.RegisterActivityWithOptions(func(context.Context, Execution) error { calls++; return errors.New("lost provider response") }, activity.RegisterOptions{Name: sendActivity})
	environment.ExecuteWorkflow(PlatformWorkflow, Execution{"site", "org", "a182d2e0-6f91-4e71-8b7f-5df25b75fb11"})
	if !environment.IsWorkflowCompleted() || environment.GetWorkflowError() == nil || calls != 1 {
		t.Fatal("mutation replayed", calls, environment.GetWorkflowError())
	}
}

type fatalStartWorker struct{ lifecycle *Lifecycle }

func (w fatalStartWorker) Start() error { w.lifecycle.Stop(); return nil }
func (w fatalStartWorker) Stop()        {}
func TestFatalDuringStartNeverPublishesReady(t *testing.T) {
	l := &Lifecycle{}
	l.Worker = fatalStartWorker{l}
	if l.Start() == nil || l.Ready() {
		t.Fatal("failed worker became available")
	}
}
