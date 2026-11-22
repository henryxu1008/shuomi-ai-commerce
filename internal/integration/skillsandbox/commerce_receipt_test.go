package skillsandbox

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/henryxu1008/shuomi-ai-commerce/internal/agent"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/commercetool"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/product/contentagent"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/product/contentagent/scriptcompute"
	"go.opentelemetry.io/otel/trace/noop"
)

type commerceReceiptAuthority struct{ records []commercetool.AuditRecord }

func (*commerceReceiptAuthority) ResolvePrincipal(context.Context) (commercetool.Principal, error) {
	return commercetool.Principal{TenantID: "org", UserID: "actor", Roles: []string{"operator"}}, nil
}
func (*commerceReceiptAuthority) Authorize(context.Context, commercetool.Principal, commercetool.PermissionRequirement) error {
	return nil
}
func (a *commerceReceiptAuthority) RecordToolCall(_ context.Context, record commercetool.AuditRecord) error {
	a.records = append(a.records, record)
	return nil
}

func invokeCommerceReceipt(t *testing.T, ctx context.Context, executor agent.ScriptExecutor, task agent.ScriptTask) agent.ScriptResult {
	t.Helper()
	definition, _ := contentagent.Registration()
	authority := &commerceReceiptAuthority{}
	result, err := scriptcompute.Invoke(ctx, definition, commercetool.InvocationDependencies{PrincipalResolver: authority, Authorizer: authority, Recorder: authority, Tracer: noop.NewTracerProvider().Tracer("receipt-test"), Now: time.Now, AuditTimeout: time.Second}, commercetool.CallMetadata{CallID: task.CallID, AgentID: definition.ID, AgentVersion: definition.Version, AgentRunID: task.RunID, BusinessTaskID: "product"}, task, executor)
	if err != nil || result.AuditStatus != commercetool.AuditStatusRecorded || len(authority.records) != 1 {
		t.Fatalf("Commerce dropped receipt: output=%s audit=%s records=%d err=%v", result.Output, result.AuditStatus, len(authority.records), err)
	}
	var receipt agent.ScriptResult
	if json.Unmarshal(result.Output, &receipt) != nil || !receipt.ValidFor(task.Descriptor, task.Input) {
		t.Fatalf("Commerce returned invalid receipt: %s", result.Output)
	}
	if ctx.Err() != nil {
		t.Fatalf("receipt reached caller after deadline: %v", ctx.Err())
	}
	return receipt
}

func TestCommerceReceiptPreservesCapacityDeadlineWithoutDispatch(t *testing.T) {
	t.Parallel()
	executor, task, counts := lifecycleFixture(t, "", false, lifecycleOptions{})
	executor.gate <- struct{}{}
	defer func() { <-executor.gate }()
	ctx, cancel := context.WithTimeout(context.Background(), 9*time.Second)
	defer cancel()
	result := invokeCommerceReceipt(t, ctx, executor, task)
	created, commands, _ := counts()
	if result.Receipt.Outcome != "not_dispatched" || result.Receipt.Cleanup != "not_needed" || result.Receipt.Reason != "capacity_deadline" || created != 0 || commands != 0 {
		t.Fatalf("unstarted work became uncertain: %+v created=%d commands=%d", result, created, commands)
	}
}

func TestCommerceReceiptPreservesSuccessAcrossIndependentCleanup(t *testing.T) {
	t.Parallel()
	executor, task, counts := lifecycleFixture(t, `{"schemaVersion":"title-result-v1","title":"after","diagnostics":[]}`, false, lifecycleOptions{nearDeadlineCollector: true, cleanupDelay: 4 * time.Second})
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	result := invokeCommerceReceipt(t, ctx, executor, task)
	created, commands, _ := counts()
	if result.Receipt.Outcome != "succeeded" || result.Receipt.Cleanup != "completed" || result.Title != "after" || created != 1 || commands != 2 {
		t.Fatalf("known result or cleanup lost: %+v created=%d commands=%d", result, created, commands)
	}
}

func TestCommerceReceiptPreservesSuccessWhenCleanupRemainsPending(t *testing.T) {
	t.Parallel()
	executor, task, _ := lifecycleFixture(t, `{"schemaVersion":"title-result-v1","title":"after","diagnostics":[]}`, false, lifecycleOptions{cleanupDelay: 6 * time.Second})
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	result := invokeCommerceReceipt(t, ctx, executor, task)
	if result.Receipt.Outcome != "succeeded" || result.Receipt.Cleanup != "pending" || result.Title != "after" {
		t.Fatalf("cleanup changed the known calculation or falsely completed: %+v", result)
	}
}

func TestCommerceReceiptDistinguishesCollectorFailureFromUnknown(t *testing.T) {
	const init = "{\"type\":\"init\",\"text\":\"collector\"}\n\n"
	for _, tc := range []struct{ name, events, outcome string }{
		{"confirmed invalid file", init + "{\"type\":\"error\",\"error\":{\"exit_code\":1}}\n\n", "failed"},
		{"missing terminal", init, "outcome_unknown"},
		{"conflicting terminal", init + "{\"type\":\"execution_complete\",\"exit_code\":0}\n\n{\"type\":\"error\",\"error\":{\"exit_code\":1}}\n\n", "outcome_unknown"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			executor, task, counts := lifecycleFixture(t, "", false, lifecycleOptions{collectorEvents: tc.events})
			result := invokeCommerceReceipt(t, context.Background(), executor, task)
			created, commands, _ := counts()
			if result.Receipt.Outcome != tc.outcome || result.Receipt.Cleanup != "completed" || result.Title != "" || created != 1 || commands != 2 {
				t.Fatalf("collector classification lost: %+v created=%d commands=%d", result, created, commands)
			}
			if tc.outcome == "failed" && result.Receipt.Reason != "invalid_result" {
				t.Fatalf("confirmed invalid result lacks actionable reason: %+v", result.Receipt)
			}
		})
	}
}
