package einoruntime

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"task-processor/internal/agent"
	"testing"
)

func skillRequest(t *testing.T, r agent.Request, digest string) agent.Request {
	t.Helper()
	raw, _ := json.Marshal(r)
	var wire map[string]any
	_ = json.Unmarshal(raw, &wire)
	wire["SkillVersionRef"] = map[string]any{"kind": "skill-version-v1", "id": contextTestID, "ownerKind": "USER", "ownerId": "actor", "version": 1, "digest": digest, "policyVersion": "skill-method-v1"}
	raw, _ = json.Marshal(wire)
	if e := json.Unmarshal(raw, &r); e != nil {
		t.Fatal(e)
	}
	return r
}

type skillCaptureModel struct {
	model  *fakeModel
	inputs []string
}

func (m *skillCaptureModel) Quote(ctx context.Context, in agent.ModelInput) (agent.Quote, error) {
	raw, _ := json.Marshal(in)
	m.inputs = append(m.inputs, string(raw))
	return m.model.Quote(ctx, in)
}
func (m *skillCaptureModel) Decide(ctx context.Context, in agent.ModelInput) (agent.ModelResult, error) {
	return m.model.Decide(ctx, in)
}
func TestSkillVersionReachesModelAndBindsTheOriginalRun(t *testing.T) {
	runtime, request, model, _, _, _, _ := fixture(t, proposal("supported title"))
	capture := &skillCaptureModel{model: model}
	runtime.config.Model = capture
	request = skillRequest(t, request, strings.Repeat("a", 64))
	record, e := runtime.Start(context.Background(), request)
	if e != nil {
		t.Fatal(e)
	}
	raw, _ := json.Marshal(record)
	if !strings.Contains(string(raw), "skill-version-v1") || len(capture.inputs) != 1 || !strings.Contains(capture.inputs[0], contextTestID) {
		t.Fatal("skill identity disappeared between request, quote and durable run")
	}
	repeated, e := runtime.Start(context.Background(), request)
	if e != nil || repeated.State.RunID != record.State.RunID || model.calls != 1 {
		t.Fatal("same skill replay dispatched again", e)
	}
	_, e = runtime.Start(context.Background(), skillRequest(t, request, strings.Repeat("b", 64)))
	if !errors.Is(e, agent.ErrConflict) {
		t.Fatal("changed skill reused the original request key", e)
	}
}
func TestPartialSkillRefIsRejectedBeforeClaim(t *testing.T) {
	runtime, request, _, _, _, _, store := fixture(t, proposal("supported title"))
	_, e := runtime.Start(context.Background(), skillRequest(t, request, ""))
	if !errors.Is(e, agent.ErrInvalid) || store.exists {
		t.Fatal("partial selected skill was discarded or admitted", e)
	}
}
