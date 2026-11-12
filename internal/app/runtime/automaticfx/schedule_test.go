package automaticfxruntime

import (
	"context"
	"errors"
	"reflect"
	"testing"

	commonpb "go.temporal.io/api/common/v1"
	enums "go.temporal.io/api/enums/v1"
	querypb "go.temporal.io/api/query/v1"
	workflowpb "go.temporal.io/api/workflow/v1"
	servicepb "go.temporal.io/api/workflowservice/v1"
	"go.temporal.io/sdk/client"
)

type fakeBootstrap struct {
	fail  string
	calls []string
}

type stoppedQuery struct{ condition enums.QueryRejectCondition }

func (*stoppedQuery) DescribeWorkflowExecution(context.Context, string, string) (*servicepb.DescribeWorkflowExecutionResponse, error) {
	return &servicepb.DescribeWorkflowExecutionResponse{WorkflowExecutionInfo: &workflowpb.WorkflowExecutionInfo{Execution: &commonpb.WorkflowExecution{WorkflowId: CoordinatorID, RunId: "fixture"}, Type: &commonpb.WorkflowType{Name: CoordinatorWorkflowName}, Status: enums.WORKFLOW_EXECUTION_STATUS_RUNNING}}, nil
}
func (s *stoppedQuery) QueryWorkflowWithOptions(_ context.Context, req *client.QueryWorkflowWithOptionsRequest) (*client.QueryWorkflowWithOptionsResponse, error) {
	s.condition = req.QueryRejectCondition
	return &client.QueryWorkflowWithOptionsResponse{QueryRejected: &querypb.QueryRejected{Status: enums.WORKFLOW_EXECUTION_STATUS_TERMINATED}}, nil
}
func TestStopRejectsCoordinatorClosedBetweenDescribeAndQuery(t *testing.T) {
	s := &stoppedQuery{}
	stopped, err := checkStop(context.Background(), s)
	if stopped || !errors.Is(err, ErrControlUnknown) || s.condition != enums.QUERY_REJECT_CONDITION_NOT_OPEN {
		t.Fatalf("stopped=%v err=%v condition=%v", stopped, err, s.condition)
	}
}

func (f *fakeBootstrap) step(name string) error {
	f.calls = append(f.calls, name)
	if f.fail == name {
		return errors.New("fixture")
	}
	return nil
}
func (f *fakeBootstrap) createPaused(context.Context) error      { return f.step("create") }
func (f *fakeBootstrap) startCoordinator(context.Context) error  { return f.step("start") }
func (f *fakeBootstrap) verifyCoordinator(context.Context) error { return f.step("verify") }
func (f *fakeBootstrap) unpause(context.Context) error           { return f.step("unpause") }

func TestBootstrapFailsClosedAtEveryAmbiguousStep(t *testing.T) {
	for _, tc := range []struct {
		fail string
		want []string
	}{
		{"create", []string{"create"}},
		{"start", []string{"create", "start"}},
		{"verify", []string{"create", "start", "verify"}},
		{"unpause", []string{"create", "start", "verify", "unpause"}},
		{"", []string{"create", "start", "verify", "unpause"}},
	} {
		f := &fakeBootstrap{fail: tc.fail}
		err := bootstrap(context.Background(), f)
		if (err != nil) != (tc.fail != "") || !reflect.DeepEqual(f.calls, tc.want) {
			t.Fatalf("fail=%s calls=%v err=%v", tc.fail, f.calls, err)
		}
	}
}
