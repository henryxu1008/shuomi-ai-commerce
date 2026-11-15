package einoruntime

import (
	"context"
	"encoding/json"
	"time"

	strict "sigs.k8s.io/json"
	"task-processor/internal/agent"
	"task-processor/internal/commercetool"
)

func (e *execution) script(ctx context.Context, s *flowState) {
	if s.State.ScriptReceipt != nil || !s.State.Request.ValidScript() || !s.State.Request.ScriptSelected() || nilPort(e.runtime.config.Script) {
		s.stop(agent.StopTool)
		return
	}
	if !e.guard(ctx, s) {
		return
	}
	if stop := s.State.Usage.Step(s.State.Request.Limits); stop != "" {
		s.stop(stop)
		return
	}
	if len(s.State.Candidate.Changes) != 1 || s.State.Candidate.Changes[0].Field != "title" || s.State.Candidate.Changes[0].Value == "" {
		s.stop(agent.StopInvalidOutput)
		return
	}
	meta := commercetool.CallMetadata{CallID: callID(s), AgentID: e.runtime.config.Definition.ID, AgentVersion: e.runtime.config.Definition.Version, AgentRunID: s.State.RunID, BusinessTaskID: s.State.Request.Binding.ContextID, TraceID: s.State.TraceID}
	ref := commercetool.ToolRef{ID: "skill.script.compute", Version: s.State.Request.ScriptDescriptor.ToolVersion}
	s.State.ScriptOriginalTitle = s.State.Candidate.Changes[0].Value
	s.State.ScriptReceipt = &agent.ScriptReceipt{Outcome: "outcome_unknown", Cleanup: "pending", Reason: "dispatch_unconfirmed", DescriptorDigest: s.State.Request.ScriptDescriptor.Digest()}
	s.State.History = append(s.State.History, agent.Observation{Step: s.State.Usage.Steps, Tool: ref, CallID: meta.CallID})
	stepCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	result, err := e.runtime.config.Script.InvokeScript(stepCtx, meta, s.State.Request, clone(s.State.Candidate))
	observation := &s.State.History[len(s.State.History)-1]
	observation.AuditStatus = result.AuditStatus
	var execution agent.ScriptResult
	if len(result.Output) > 16<<10 {
		s.stop(agent.StopTooLarge)
		return
	}
	input, inputErr := agent.ScriptInputBytes(s.State.Request, s.State.Candidate)
	violations, decodeErr := strict.UnmarshalStrict(result.Output, &execution)
	if inputErr == nil && decodeErr == nil && len(violations) == 0 && execution.ValidFor(s.State.Request.ScriptDescriptor, input) {
		s.State.ScriptReceipt = &execution.Receipt
		observation.Output = append(json.RawMessage(nil), result.Output...)
	}
	if result.AuditStatus != commercetool.AuditStatusRecorded {
		s.stop(agent.StopAudit)
		return
	}
	if s.State.ScriptReceipt.Outcome == "failed" || s.State.ScriptReceipt.Outcome == "not_dispatched" {
		s.stop(agent.StopTool)
		return
	}
	if err != nil || s.State.ScriptReceipt.Outcome != "succeeded" || execution.Title == "" {
		s.stop(agent.StopExecutionOutcomeUnknown)
		return
	}
	if !e.guard(ctx, s) {
		return
	}
	s.State.Candidate.Changes[0].Value = execution.Title
	s.Next = "validate"
}
