package aiworkbench

import (
	"github.com/google/uuid"
	"strconv"
	"task-processor/internal/agent"
	"task-processor/internal/agentconfig"
	"task-processor/internal/product/enrichment"
)

const TitleTaskKind = "product.title.optimize"
const ContentTaskKind = "product.content.localize"

// Subject preserves the selected owner's identity and effective version. Source
// provenance belongs to that owner; it is not an alternative subject identity.
type Subject struct {
	Kind           string `json:"kind"`
	ID             string `json:"id"`
	CatalogVersion string `json:"catalogVersion"`
	PublicationID  string `json:"publicationId"`
}
type TaskSelection struct {
	Subject    Subject                     `json:"subject"`
	Capability string                      `json:"capability"`
	Parameters enrichment.ContentSelection `json:"parameters"`
}

func (s TaskSelection) Valid() bool {
	id, err := uuid.Parse(s.Subject.ID)
	version, ve := strconv.ParseUint(s.Subject.CatalogVersion, 10, 63)
	return err == nil && id.String() == s.Subject.ID && ve == nil && version > 0 && strconv.FormatUint(version, 10) == s.Subject.CatalogVersion &&
		(s.Subject.Kind == "own_product" || s.Subject.Kind == "acquisition") && agent.ValidID(s.Subject.PublicationID) &&
		s.Capability == ContentTaskKind && s.Parameters.Valid()
}
func (m MessageInput) ValidSelection() bool {
	if m.TaskSelection != nil {
		return m.OperationID == "" && m.TargetPlatform == "" && m.TaskSelection.Valid()
	}
	return agent.ValidID(m.OperationID) && (m.TargetPlatform == "shein" || m.TargetPlatform == "temu" || m.TargetPlatform == "amazon")
}
func (w WorkScope) Target() string {
	if w.TaskSelection != nil {
		return "product"
	}
	return w.TargetPlatform
}
func (p ExecutionProposal) Selection() *TaskSelection {
	if p.Kind != ContentTaskKind {
		return nil
	}
	return &TaskSelection{Subject: Subject{Kind: p.ContextKind, ID: p.OperationID, CatalogVersion: p.CatalogVersion, PublicationID: p.PublicationID}, Capability: p.Kind, Parameters: p.ContentSelection}
}

// ValidCapability prevents interpreting a stored title receipt as a content run.
func (p ExecutionProposal) ValidCapability() bool {
	switch p.Kind {
	case ContentTaskKind:
		return p.Selection().Valid() && p.TargetPlatform == "product" && p.AgentID == agentconfig.ContentAgentID && p.AgentVersion == agentconfig.ContentAgentVersion && p.ExecutionLimits.Valid()
	case TitleTaskKind:
		return p.AgentID == "product.title.agent" && p.AgentVersion == "v1.0.0" && p.ContextKind == "" && p.ContentSelection.Absent() && p.ExecutionLimits == (agent.Limits{}) && (p.TargetPlatform == "shein" || p.TargetPlatform == "temu" || p.TargetPlatform == "amazon")
	}
	return false
}
func (p ExecutionProposal) MatchesRequest(r agent.Request) bool {
	if !p.ValidCapability() {
		return false
	}
	if p.Kind == ContentTaskKind {
		return r.Binding.ContextKind == p.ContextKind && r.ContentSelection == p.ContentSelection && r.Limits == p.ExecutionLimits && r.PolicyVersion == enrichment.ContentPolicyVersion && r.PromptVersion == agentconfig.ContentPromptVersion
	}
	prompt := "product-title-agent-v1"
	if !r.ContextSnapshotRef.Absent() {
		prompt = "product-title-agent-knowledge-v1"
	}
	return r.Binding.ContextKind == "acquisition" && r.ContentSelection.Absent() && r.PolicyVersion == "title-review-v1" && r.PromptVersion == prompt
}

func reviewableFields(r agent.Request, c enrichment.Candidate) bool {
	if r.PolicyVersion != enrichment.ContentPolicyVersion {
		return r.ContentSelection.Absent() && len(c.Changes) == 1
	}
	if !r.ContentSelection.Valid() || len(c.Warnings) != 0 || len(c.Rejections) != 0 {
		return false
	}
	fields := r.ContentSelection.Fields()
	if len(fields) != len(c.Changes) {
		return false
	}
	seen := map[string]bool{}
	for _, change := range c.Changes {
		seen[change.Field] = true
	}
	for _, field := range fields {
		if !seen[field] {
			return false
		}
	}
	return len(seen) == len(fields)
}
