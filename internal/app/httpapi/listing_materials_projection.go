package httpapi

import (
	"context"
	"errors"
	"fmt"
	"strconv"

	"task-processor/internal/agent"
	"task-processor/internal/agentconfig"
	"task-processor/internal/listing/materials"
	record "task-processor/internal/listing/record/target"
)

type materialsRunDTO struct {
	RunID                    string                         `json:"runId"`
	RequestKey               string                         `json:"requestKey"`
	RecordID                 string                         `json:"recordId"`
	Phase                    agent.Phase                    `json:"phase"`
	Revision                 string                         `json:"revision"`
	StopReason               agent.StopReason               `json:"stopReason,omitempty"`
	ConfigurationSnapshotRef agent.ConfigurationSnapshotRef `json:"configurationSnapshotRef"`
	ModeSnapshotRef          agent.ModeSnapshotRef          `json:"modeSnapshotRef"`
	ExecutionMode            agent.ExecutionMode            `json:"executionMode"`
	ModeRevision             string                         `json:"modeRevision"`
	TemplateSelection        *agentconfig.TemplateRef       `json:"templateSelection,omitempty"`
	Proposal                 *materials.Proposal            `json:"proposal,omitempty"`
	CandidateHash            string                         `json:"candidateHash,omitempty"`
	Before                   record.TargetInput             `json:"before"`
	After                    *record.TargetInput            `json:"after,omitempty"`
	Unresolved               []string                       `json:"unresolved"`
	CanRequestAdoption       bool                           `json:"canRequestAdoption"`
	Adoption                 *record.TargetReceipt          `json:"adoption,omitempty"`
	Steps                    []productAgentStepDTO          `json:"steps"`
	Tokens                   int64                          `json:"tokens"`
	EstimatedCostMicros      int64                          `json:"estimatedCostMicros"`
	Currency                 string                         `json:"currency"`
	UsageStatus              string                         `json:"usageStatus"`
}

// Historical projection uses the original immutable base and snapshots. It
// neither reruns the model nor applies new-execution head/configuration gates.
func (a *listingMaterialsApplication) projectRun(ctx context.Context, recordID string, run agent.Record, snapshot agentconfig.Snapshot) (materialsRunDTO, error) {
	s := run.State
	id, base, binding, err := a.readBase(ctx, recordID)
	if err != nil {
		return materialsRunDTO{}, err
	}
	if binding != s.Request.Binding || s.Scope != (agent.Scope{OrganizationID: id.OrganizationID, ActorID: id.ActorID}) {
		return materialsRunDTO{}, agentconfig.ErrForbidden
	}
	if snapshot.ID == "" {
		snapshot, err = a.parent.configuration.LoadSnapshot(ctx, s.Scope, s.Request.ConfigurationSnapshotRef)
		if err != nil {
			return materialsRunDTO{}, err
		}
	}
	if snapshot.AgentID != materials.AgentID || snapshot.MemberID != id.MemberID {
		return materialsRunDTO{}, agentconfig.ErrConflict
	}
	mode, err := a.parent.configuration.LoadModeSnapshot(ctx, id, s.Request.ModeSnapshotRef)
	if err != nil {
		return materialsRunDTO{}, err
	}
	out := materialsRunDTO{RunID: s.RunID, RequestKey: s.Request.Key, RecordID: recordID, Phase: s.Phase, Revision: strconv.FormatUint(s.Revision, 10), StopReason: s.StopReason, ConfigurationSnapshotRef: s.Request.ConfigurationSnapshotRef, ModeSnapshotRef: mode.Ref(), ExecutionMode: mode.Mode, ModeRevision: mode.Revision, TemplateSelection: snapshot.Template, Before: base.Input, Unresolved: append([]string{}, s.Unresolved...), Steps: []productAgentStepDTO{}, Tokens: s.Usage.Tokens, EstimatedCostMicros: s.Usage.CostMicros, Currency: s.Request.Limits.Currency, UsageStatus: "observed"}
	out.Unresolved = materialsRunDiagnostics(s)
	if s.PendingInvocationID != "" || s.Phase == agent.Running || s.StopReason == agent.StopUsageUnknown || s.StopReason == agent.StopModelUnknown {
		out.UsageStatus = "unknown_reserved"
	}
	for _, step := range s.History {
		out.Steps = append(out.Steps, productAgentStepDTO{Step: step.Step, Tool: step.Tool.ID, CallID: step.CallID, InvocationID: step.InvocationID, AuditStatus: string(step.AuditStatus)})
	}
	if materialsRunReviewable(s) {
		proposal, e := materials.ParseProposal(s.DomainProposal.Payload)
		if e != nil {
			return out, e
		}
		after, e := materials.ApplyProposal(base, proposal)
		if e != nil {
			return out, e
		}
		out.Proposal, out.After, out.CandidateHash = &proposal, &after, s.Validation.CandidateHash
		out.CanRequestAdoption = mode.Mode != agent.ModeReadOnly
		receipt, e := a.supply.Targets.ReadCommand(ctx, materialsAdoptionKey(s))
		if e == nil {
			out.CanRequestAdoption = false
			if verifyMaterialsAdoptionReceipt(base, s, receipt) == nil {
				out.Adoption = &receipt
			} else {
				out.Unresolved = materialsRunDiagnostics(s, "原保存编号对应的草稿与本次建议不一致，请核对原记录。")
			}
		} else if !errors.Is(e, record.ErrNotFound) {
			return out, e
		}
	}
	return out, nil
}

func materialsRunDiagnostics(s agent.State, extra ...string) []string {
	const displayLimit = 4096
	owner := []string{}
	if s.Validation != nil {
		owner = append(owner, s.Validation.Unresolved...)
	}
	owner = append(owner, extra...)
	total := len(s.Unresolved) + len(owner)
	if total <= displayLimit {
		return append(append([]string{}, s.Unresolved...), owner...)
	}
	// Preserve owner/receipt diagnostics first. This bounds only the public
	// display; the paid run and its original model notes are never rewritten.
	if len(owner) >= displayLimit {
		owner = owner[len(owner)-(displayLimit-1):]
	}
	modelCount := min(len(s.Unresolved), displayLimit-1-len(owner))
	out := append(append([]string{}, s.Unresolved[:modelCount]...), owner...)
	out = append(out, fmt.Sprintf("诊断展示已省略 %d 条说明；完整历史记录仍然保留。", total-len(out)))
	return out
}
