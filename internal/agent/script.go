package agent

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"strings"
	"unicode/utf8"

	"task-processor/internal/commercetool"
	reviewproposal "task-processor/internal/product/review/proposal"
)

// ScriptDescriptor is deployment-owned, comparable and frozen before Claim.
// It contains no endpoint or credential and is part of the original fingerprint.
type ScriptDescriptor struct {
	Profile         string `json:"profile"`
	Revision        string `json:"revision"`
	ImageDigest     string `json:"imageDigest"`
	ExecdDigest     string `json:"execdDigest"`
	NetworkRevision string `json:"networkRevision"`
	ToolVersion     string `json:"toolVersion"`
}

func (d ScriptDescriptor) Absent() bool { return d == ScriptDescriptor{} }
func (d ScriptDescriptor) Valid() bool {
	return d.Profile == "python-title-json-v1" && ValidID(d.Revision) && ValidID(d.NetworkRevision) && d.ToolVersion == "v1.0.0" && imageHash(d.ImageDigest) && imageHash(d.ExecdDigest)
}
func imageHash(v string) bool {
	b, e := hex.DecodeString(strings.TrimPrefix(v, "sha256:"))
	return strings.HasPrefix(v, "sha256:") && e == nil && len(b) == 32 && hex.EncodeToString(b) == strings.TrimPrefix(v, "sha256:")
}
func ScriptDigest(raw []byte) string      { h := sha256.Sum256(raw); return hex.EncodeToString(h[:]) }
func (d ScriptDescriptor) Digest() string { b, _ := json.Marshal(d); return ScriptDigest(b) }
func (r Request) ScriptSelected() bool {
	return r.SkillVersionRef.PolicyVersion == "skill-script-method-v1"
}
func (r Request) ValidScript() bool {
	if !r.ScriptSelected() {
		return r.ScriptDescriptor.Absent()
	}
	return r.SkillVersionRef.Valid() && r.ScriptDescriptor.Valid() && r.ContentSelection.TitleOnly() && r.PolicyVersion == reviewproposal.ContentPolicyVersion && r.ScheduledSource.Absent()
}

type ScriptInput struct {
	SchemaVersion  string   `json:"schemaVersion"`
	Title          string   `json:"title"`
	TargetLanguage string   `json:"targetLanguage"`
	EvidenceRefs   []string `json:"evidenceRefs"`
}
type ScriptFile struct{ Path, Text string }
type ScriptTask struct {
	Descriptor    ScriptDescriptor
	Files         []ScriptFile
	Entrypoint    string
	PackageDigest string
	Input         []byte
	RunID, CallID string
}

// ScriptExecutor is the narrow remote calculation port, not user authorization.
type ScriptExecutor interface {
	Execute(context.Context, ScriptTask) (ScriptResult, error)
}
type ScriptReceipt struct {
	Outcome          string `json:"outcome"`
	Cleanup          string `json:"cleanup"`
	Reason           string `json:"reason"`
	InputDigest      string `json:"inputDigest"`
	OutputDigest     string `json:"outputDigest"`
	DescriptorDigest string `json:"descriptorDigest"`
}
type ScriptResult struct {
	Title   string        `json:"title"`
	Receipt ScriptReceipt `json:"receipt"`
}

func ScriptInputBytes(r Request, c reviewproposal.Candidate) ([]byte, error) {
	if !r.ValidScript() || len(c.Changes) != 1 || c.Changes[0].Field != "title" || c.Changes[0].Value == "" {
		return nil, ErrInvalid
	}
	raw, err := json.Marshal(ScriptInput{SchemaVersion: "title-candidate-v1", Title: c.Changes[0].Value, TargetLanguage: r.ContentSelection.TargetLanguage, EvidenceRefs: append([]string{}, c.Changes[0].EvidenceIDs...)})
	if err != nil || len(raw) > 16<<10 {
		return nil, ErrInvalid
	}
	return raw, nil
}
func (s ScriptResult) ValidFor(d ScriptDescriptor, input []byte) bool {
	if s.Receipt.DescriptorDigest != d.Digest() || s.Receipt.InputDigest != ScriptDigest(input) || len(s.Receipt.Reason) > 64 || !utf8.ValidString(s.Title) || len(s.Title) > 8<<10 {
		return false
	}
	switch s.Receipt.Outcome {
	case "succeeded", "failed", "outcome_unknown", "not_dispatched":
	default:
		return false
	}
	switch s.Receipt.Cleanup {
	case "completed", "pending", "not_needed":
	default:
		return false
	}
	if s.Receipt.Outcome == "succeeded" {
		return s.Title != "" && imageHash("sha256:"+s.Receipt.OutputDigest)
	}
	return s.Title == ""
}

// ScriptGateway is called only by the fixed graph node and traverses the
// current Commerce Registry/principal/audit boundary with a typed payload.
type ScriptGateway interface {
	InvokeScript(context.Context, commercetool.CallMetadata, Request, reviewproposal.Candidate) (commercetool.Result, error)
}
