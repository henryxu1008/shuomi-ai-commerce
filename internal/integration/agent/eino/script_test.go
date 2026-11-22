package einoruntime

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/agent"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/commercetool"
	reviewproposal "github.com/henryxu1008/shuomi-ai-commerce/internal/product/review/proposal"
	"strings"
	"testing"
)

func TestScriptSelectionCannotRunWithoutFrozenExecutor(t *testing.T) {
	r, request, model, _, _, _, store := fixture(t, proposal("before"))
	request = skillRequest(t, request, strings.Repeat("a", 64))
	request.SkillVersionRef.PolicyVersion = "skill-script-method-v1"
	_, err := r.Start(context.Background(), request)
	if !errors.Is(err, agent.ErrInvalid) || model.calls != 0 || store.exists {
		t.Fatal("script selection silently executed as text", err)
	}
}

type scriptGateway struct {
	calls   int
	outcome string
	corrupt string
}

func (g *scriptGateway) InvokeScript(_ context.Context, _ commercetool.CallMetadata, r agent.Request, c reviewproposal.Candidate) (commercetool.Result, error) {
	g.calls++
	raw, _ := json.Marshal(agent.ScriptInput{SchemaVersion: "title-candidate-v1", Title: c.Changes[0].Value, TargetLanguage: r.ContentSelection.TargetLanguage, EvidenceRefs: append([]string{}, c.Changes[0].EvidenceIDs...)})
	title := ""
	if g.outcome == "succeeded" {
		title = "supported title"
	}
	output, _ := json.Marshal(agent.ScriptResult{Title: title, Receipt: agent.ScriptReceipt{Outcome: g.outcome, Cleanup: "completed", Reason: "completed", InputDigest: agent.ScriptDigest(raw), OutputDigest: strings.Repeat("b", 64), DescriptorDigest: r.ScriptDescriptor.Digest()}})
	if g.corrupt == "input" {
		output = []byte(strings.ReplaceAll(string(output), agent.ScriptDigest(raw), strings.Repeat("c", 64)))
	}
	if g.corrupt == "descriptor" {
		output = []byte(strings.ReplaceAll(string(output), r.ScriptDescriptor.Digest(), strings.Repeat("c", 64)))
	}
	if g.corrupt == "oversized" {
		output = []byte(strings.Repeat(" ", 16<<10+1))
	}
	return commercetool.Result{Output: output, AuditStatus: commercetool.AuditStatusRecorded}, nil
}
func TestFixedScriptRunsOnceAndKnownFailureNeverRepairs(t *testing.T) {
	for _, outcome := range []string{"succeeded", "failed", "not_dispatched", "outcome_unknown"} {
		t.Run(outcome, func(t *testing.T) {
			r, request, model, _, validator, _, _ := fixture(t, proposal("before"))
			gateway := &scriptGateway{outcome: outcome}
			r.config.Script = gateway
			request = skillRequest(t, request, strings.Repeat("a", 64))
			request.SkillVersionRef.PolicyVersion = "skill-script-method-v1"
			request.ContentSelection = reviewproposal.ContentSelection{TargetLanguage: "en", Title: true}
			request.PolicyVersion = reviewproposal.ContentPolicyVersion
			request.ScriptDescriptor = agent.ScriptDescriptor{Profile: "python-title-json-v1", Revision: "profile-1", NetworkRevision: "network-1", ToolVersion: "v1.0.0", ImageDigest: "sha256:" + strings.Repeat("a", 64), ExecdDigest: "sha256:" + strings.Repeat("b", 64)}
			record, err := r.Start(context.Background(), request)
			if err != nil || gateway.calls != 1 || model.calls != 1 || record.State.ScriptReceipt == nil {
				t.Fatal("fixed dispatch lost", err)
			}
			if outcome == "succeeded" && (record.State.Phase != agent.HumanReviewRequired || record.State.Candidate.Changes[0].Value != "supported title" || record.State.ScriptOriginalTitle != "before") {
				t.Fatal("postprocess not bound to review")
			}
			if outcome != "succeeded" && (record.State.Phase != agent.Stopped || validator.calls != 0) {
				t.Fatal("failure fell through to validation/repair")
			}
			if outcome == "failed" && record.State.StopReason != agent.StopTool {
				t.Fatal("known failure relabeled UNKNOWN")
			}
			_, err = r.Start(context.Background(), request)
			if err != nil || gateway.calls != 1 || model.calls != 1 {
				t.Fatal("same run replayed", err)
			}
			changed := request
			changed.ScriptDescriptor.NetworkRevision = "changed-network"
			_, err = r.Start(context.Background(), changed)
			if !errors.Is(err, agent.ErrConflict) || gateway.calls != 1 || model.calls != 1 {
				t.Fatal("changed profile reused original request identity", err)
			}
		})
	}
}

func TestScriptReceiptCannotBorrowAnotherInputOrProfile(t *testing.T) {
	for _, corruption := range []string{"input", "descriptor", "oversized"} {
		t.Run(corruption, func(t *testing.T) {
			r, request, model, _, validator, _, _ := fixture(t, proposal("before"))
			gateway := &scriptGateway{outcome: "succeeded", corrupt: corruption}
			r.config.Script = gateway
			request = skillRequest(t, request, strings.Repeat("a", 64))
			request.SkillVersionRef.PolicyVersion = "skill-script-method-v1"
			request.ContentSelection = reviewproposal.ContentSelection{TargetLanguage: "en", Title: true}
			request.PolicyVersion = reviewproposal.ContentPolicyVersion
			request.ScriptDescriptor = agent.ScriptDescriptor{Profile: "python-title-json-v1", Revision: "1", NetworkRevision: "1", ToolVersion: "v1.0.0", ImageDigest: "sha256:" + strings.Repeat("a", 64), ExecdDigest: "sha256:" + strings.Repeat("b", 64)}
			record, err := r.Start(context.Background(), request)
			if err != nil || record.State.Phase != agent.Stopped || validator.calls != 0 || gateway.calls != 1 || model.calls != 1 || record.State.Candidate.Changes[0].Value != "before" {
				t.Fatal("invalid receipt was consumed", err)
			}
			_, err = r.Start(context.Background(), request)
			if err != nil || gateway.calls != 1 {
				t.Fatal("invalid receipt redispatched", err)
			}
		})
	}
}
