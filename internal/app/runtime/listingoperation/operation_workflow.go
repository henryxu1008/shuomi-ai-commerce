package listingoperation

import (
	"context"
	"errors"
	enums "go.temporal.io/api/enums/v1"
	"go.temporal.io/api/serviceerror"
	"go.temporal.io/api/workflowservice/v1"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/worker"
	"go.temporal.io/sdk/workflow"
	listingapp "task-processor/internal/app/listing"
	"task-processor/internal/authidentity"
	"task-processor/internal/listing/preparation"
	"task-processor/internal/product/collection"
	"time"
)

const ListingTaskQueue = "supply-chain-current"
const ListingWorkflowName = "SupplyPreparationOperationV1"
const listingListActivity = "SupplyPreparationListV1"
const listingItemActivity = "SupplyPreparationItemV1"

func ListingOperationWorkflow(ctx workflow.Context, in listingapp.OperationExecution) error {
	if !authidentity.IsBoundedIdentifier(in.OrganizationID) || !collection.ValidID(in.OperationID) || in.After != "" && !collection.ValidID(in.After) {
		return temporal.NewNonRetryableApplicationError("invalid supply operation identity", "invalid", nil)
	}
	ctx = workflow.WithActivityOptions(ctx, workflow.ActivityOptions{StartToCloseTimeout: 9 * time.Minute, RetryPolicy: &temporal.RetryPolicy{InitialInterval: time.Second, MaximumInterval: 5 * time.Minute}})
	for pages := 0; pages < 10; pages++ {
		var page listingapp.OperationPage
		if err := workflow.ExecuteActivity(ctx, listingListActivity, in).Get(ctx, &page); err != nil {
			return err
		}
		if page.Cancelled {
			return nil
		}
		for _, item := range page.Items {
			if preparation.ItemTerminal(item.Status) {
				continue
			}
			var result preparation.OperationItem
			if err := workflow.ExecuteActivity(ctx, listingItemActivity, in, item.SourceID).Get(ctx, &result); err != nil {
				return err
			}
		}
		if page.NextCursor == "" {
			return nil
		}
		in.After = page.NextCursor
	}
	return workflow.NewContinueAsNewError(ctx, ListingWorkflowName, in)
}

type ListingWorkflowClient interface {
	ExecuteWorkflow(context.Context, client.StartWorkflowOptions, interface{}, ...interface{}) (client.WorkflowRun, error)
	DescribeWorkflowExecution(context.Context, string, string) (*workflowservice.DescribeWorkflowExecutionResponse, error)
}
type TemporalOperationStarter struct{ Client ListingWorkflowClient }

func (s TemporalOperationStarter) Ensure(ctx context.Context, org, id string) error {
	if ctx == nil || s.Client == nil || !authidentity.IsBoundedIdentifier(org) || !collection.ValidID(id) {
		return preparation.ErrInvalid
	}
	workflowID := preparation.WorkflowID(org, id)
	_, err := s.Client.ExecuteWorkflow(ctx, client.StartWorkflowOptions{ID: workflowID, TaskQueue: ListingTaskQueue, WorkflowExecutionTimeout: 7 * 24 * time.Hour, WorkflowIDConflictPolicy: enums.WORKFLOW_ID_CONFLICT_POLICY_USE_EXISTING, WorkflowIDReusePolicy: enums.WORKFLOW_ID_REUSE_POLICY_REJECT_DUPLICATE}, ListingWorkflowName, listingapp.OperationExecution{OrganizationID: org, OperationID: id})
	var exists *serviceerror.WorkflowExecutionAlreadyStarted
	if err != nil && !errors.As(err, &exists) {
		return preparation.ErrUnknown
	}
	description, err := s.Client.DescribeWorkflowExecution(ctx, workflowID, "")
	if err != nil || description == nil || description.WorkflowExecutionInfo == nil {
		return preparation.ErrUnknown
	}
	info := description.WorkflowExecutionInfo
	if info.Execution == nil || info.Execution.WorkflowId != workflowID || info.Type == nil || info.Type.Name != ListingWorkflowName {
		return preparation.ErrUnknown
	}
	if info.Status != enums.WORKFLOW_EXECUTION_STATUS_RUNNING && info.Status != enums.WORKFLOW_EXECUTION_STATUS_COMPLETED {
		return preparation.ErrUnknown
	}
	return nil
}
func NewListingWorker(client client.Client, activities *listingapp.OperationActivities) (listingapp.OperationWorker, error) {
	if client == nil || activities == nil || activities.Operations == nil || activities.Repository == nil || activities.Sources == nil || activities.Targets == nil || activities.Products == nil || activities.Creator == nil || activities.Uploader == nil {
		return nil, preparation.ErrUnavailable
	}
	current := worker.New(client, ListingTaskQueue, worker.Options{MaxConcurrentActivityExecutionSize: 4})
	current.RegisterWorkflowWithOptions(ListingOperationWorkflow, workflow.RegisterOptions{Name: ListingWorkflowName})
	current.RegisterActivityWithOptions(activities.List, activity.RegisterOptions{Name: listingListActivity})
	current.RegisterActivityWithOptions(activities.Process, activity.RegisterOptions{Name: listingItemActivity})
	return current, nil
}

func WorkerFactory(c client.Client) listingapp.OperationWorkerFactory {
	return func(activities *listingapp.OperationActivities) (listingapp.OperationWorker, error) {
		return NewListingWorker(c, activities)
	}
}
