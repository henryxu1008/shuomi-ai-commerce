package einoruntime

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"go.opentelemetry.io/otel/trace/noop"
	"task-processor/internal/agent"
	"task-processor/internal/commercetool"
	"task-processor/internal/product/contentagent/scriptcompute"
	reviewproposal "task-processor/internal/product/review/proposal"
)

// Only the remote calculation is controlled here. Invocation, schema/principal
// checks, audit, Graph state and replay traverse their real implementations.
type commerceScriptGateway struct {
	outcome, cleanup, reason string
	calls, audits            int
}

func (*commerceScriptGateway) ResolvePrincipal(context.Context) (commercetool.Principal, error) {
	return commercetool.Principal{TenantID: "org-1", UserID: "actor-1", Roles: []string{"operator"}}, nil
}
func (*commerceScriptGateway) Authorize(context.Context, commercetool.Principal, commercetool.PermissionRequirement) error {
	return nil
}
func (g *commerceScriptGateway) RecordToolCall(context.Context, commercetool.AuditRecord) error {
	g.audits++
	return nil
}
func (g *commerceScriptGateway) Execute(_ context.Context, task agent.ScriptTask) (agent.ScriptResult, error) {
	g.calls++
	result := agent.ScriptResult{Receipt: agent.ScriptReceipt{Outcome: g.outcome, Cleanup: g.cleanup, Reason: g.reason, InputDigest: agent.ScriptDigest(task.Input), DescriptorDigest: task.Descriptor.Digest()}}
	if g.outcome == "succeeded" {
		result.Title = "supported title"
		result.Receipt.OutputDigest = strings.Repeat("b", 64)
	}
	return result, nil
}
func (g *commerceScriptGateway) InvokeScript(ctx context.Context, metadata commercetool.CallMetadata, request agent.Request, candidate reviewproposal.Candidate) (commercetool.Result, error) {
	input, err := agent.ScriptInputBytes(request, candidate)
	if err != nil {
		return commercetool.Result{}, err
	}
	definition := commercetool.AgentDefinition{ID: metadata.AgentID, Version: metadata.AgentVersion, AllowedTools: []commercetool.ToolRef{scriptcompute.Definition().Ref}}
	return scriptcompute.Invoke(ctx, definition, commercetool.InvocationDependencies{PrincipalResolver: g, Authorizer: g, Recorder: g, Tracer: noop.NewTracerProvider().Tracer("script-receipt"), Now: time.Now, AuditTimeout: 2 * time.Second}, metadata, agent.ScriptTask{Descriptor: request.ScriptDescriptor, PackageDigest: request.SkillVersionRef.Digest, Input: input, RunID: metadata.AgentRunID, CallID: metadata.CallID}, g)
}

func TestCommerceScriptReceiptsPersistWithoutRepairOrReplay(t *testing.T) {
	for _, tc := range []struct{ outcome, cleanup, reason string }{
		{"not_dispatched", "not_needed", "capacity_deadline"},
		{"failed", "completed", "invalid_result"},
		{"outcome_unknown", "completed", "result_unconfirmed"},
		{"succeeded", "pending", "completed"},
	} {
		t.Run(tc.outcome, func(t *testing.T) {
			r, request, model, _, validator, _, store := fixture(t, proposal("before"))
			gateway := &commerceScriptGateway{outcome: tc.outcome, cleanup: tc.cleanup, reason: tc.reason}
			r.config.Script = gateway
			request = skillRequest(t, request, strings.Repeat("a", 64))
			request.SkillVersionRef.PolicyVersion = "skill-script-method-v1"
			request.ContentSelection = reviewproposal.ContentSelection{TargetLanguage: "en", Title: true}
			request.PolicyVersion = reviewproposal.ContentPolicyVersion
			request.ScriptDescriptor = agent.ScriptDescriptor{Profile: "python-title-json-v1", Revision: "1", NetworkRevision: "1", ToolVersion: "v1.0.0", ImageDigest: "sha256:" + strings.Repeat("a", 64), ExecdDigest: "sha256:" + strings.Repeat("b", 64)}
			record, err := r.Start(context.Background(), request)
			if err != nil || record.State.ScriptReceipt == nil {
				t.Fatalf("receipt not saved: %v", err)
			}
			got := record.State.ScriptReceipt
			if got.Outcome != tc.outcome || got.Cleanup != tc.cleanup || got.Reason != tc.reason || store.record.State.ScriptReceipt == nil || *store.record.State.ScriptReceipt != *got {
				t.Fatalf("receipt changed at persistence: %+v", got)
			}
			history := record.State.History[len(record.State.History)-1]
			var result agent.ScriptResult
			if json.Unmarshal(history.Output, &result) != nil || result.Receipt != *got || history.AuditStatus != commercetool.AuditStatusRecorded {
				t.Fatal("durable History does not carry the audited receipt")
			}
			if tc.outcome == "failed" || tc.outcome == "not_dispatched" {
				if record.State.StopReason != agent.StopTool || validator.calls != 0 {
					t.Fatal("known failure became UNKNOWN or entered repair")
				}
			} else if tc.outcome == "outcome_unknown" && record.State.StopReason != agent.StopExecutionOutcomeUnknown {
				t.Fatal("unconfirmed output became a known result")
			}
			if _, err = r.Start(context.Background(), request); err != nil || gateway.calls != 1 || gateway.audits != 1 || model.calls != 1 {
				t.Fatalf("saved receipt was replayed: calls=%d audits=%d model=%d err=%v", gateway.calls, gateway.audits, model.calls, err)
			}
		})
	}
}
