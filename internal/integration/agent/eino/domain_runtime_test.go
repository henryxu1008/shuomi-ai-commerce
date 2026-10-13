package einoruntime

import (
	"context"
	"encoding/json"
	"testing"

	"task-processor/internal/agent"
	"task-processor/internal/product/catalog/tools/canonicalinspect"
)

// The common envelope must not allow a Product candidate to silently discard
// a second domain result. No facts are adopted by this runtime test.
func TestRuntimeRejectsMixedProductDomainProposal(t *testing.T) {
	var action agent.Action
	err := json.Unmarshal([]byte(`{"Kind":"propose","Candidate":{"Changes":[{"Field":"title","Value":"supported title","EvidenceIDs":["source-1"]}]},"DomainProposal":{"Kind":"listing-materials-v1","Payload":{"categoryId":12}}}`), &action)
	if err != nil {
		t.Fatal(err)
	}
	runtime, request, _, _, validator, _, _ := fixture(t, action)
	record, err := runtime.Start(context.Background(), request)
	if err != nil {
		t.Fatal(err)
	}
	if record.State.StopReason != agent.StopInvalidOutput || validator.calls != 0 {
		t.Fatalf("mixed result must be rejected before Product validation: stop=%q calls=%d", record.State.StopReason, validator.calls)
	}
}

type domainValidatorFixture struct {
	calls    int
	request  agent.Request
	history  []agent.Observation
	proposal agent.DomainProposal
}

func (v *domainValidatorFixture) ValidateDomain(_ context.Context, request agent.Request, proposal agent.DomainProposal, history []agent.Observation) (agent.Validation, error) {
	v.calls++
	v.request, v.history, v.proposal = request, clone(history), clone(proposal)
	// The runtime must isolate model/validator-owned buffers and hash the
	// original proposal rather than accepting a validator-supplied hash.
	proposal.Payload[0] = '!'
	if len(history) != 0 && len(history[0].Output) != 0 {
		history[0].Output[0] = '!'
	}
	return agent.Validation{Valid: true, PolicyVersion: request.PolicyVersion, CandidateHash: "untrusted-hash"}, nil
}

func listingDomainFixture(t *testing.T, actions ...agent.Action) (*Runtime, agent.Request, *fakeModel, *domainValidatorFixture, *fakeStore) {
	t.Helper()
	runtime, request, model, _, _, _, store := fixture(t, actions...)
	validator := &domainValidatorFixture{}
	config := runtime.config
	config.Validator = nil
	config.DomainProposalKind = "listing-materials-v1"
	config.DomainValidator = validator
	config.Definition.ID = "listing.materials.agent"
	runtime, err := New(config)
	if err != nil {
		t.Fatal(err)
	}
	request.Binding.ContextKind, request.Binding.ContextID = "listing-target-record", "record-1"
	request.PolicyVersion, request.PromptVersion = "listing-materials-policy-v1", "listing-materials-prompt-v1"
	return runtime, request, model, validator, store
}

func domainProposalFixture() agent.Action {
	return agent.Action{Kind: "propose", DomainProposal: &agent.DomainProposal{Kind: "listing-materials-v1", Payload: json.RawMessage(`{"categoryId":12,"evidenceIds":["source-1"]}`)}}
}

func TestRuntimeDomainProposalUsesOwnValidatorAndOriginalReplay(t *testing.T) {
	action := domainProposalFixture()
	runtime, request, model, validator, store := listingDomainFixture(t,
		agent.Action{Kind: "tool", Tool: canonicalinspect.Definition().Ref}, action)
	expectedHash, err := digest(action.DomainProposal)
	if err != nil {
		t.Fatal(err)
	}
	record, err := runtime.Start(context.Background(), request)
	if err != nil {
		t.Fatal(err)
	}
	if record.State.Phase != agent.HumanReviewRequired || record.State.DomainProposal == nil ||
		string(record.State.DomainProposal.Payload) != string(action.DomainProposal.Payload) || len(record.State.Candidate.Changes) != 0 ||
		validator.calls != 1 || validator.request != request || len(validator.history) < 1 ||
		string(record.State.History[1].Output) != string(validator.history[1].Output) ||
		record.State.Validation == nil || !record.State.Validation.Valid || record.State.Validation.CandidateHash != expectedHash {
		t.Fatalf("domain result must retain exact evidence and its server hash: state=%+v validator=%+v", record.State, validator)
	}
	firstCalls := model.calls
	replayed, err := runtime.Start(context.Background(), request)
	if err != nil || replayed.State.RunID != record.State.RunID || model.calls != firstCalls ||
		replayed.State.DomainProposal == nil || string(replayed.State.DomainProposal.Payload) != string(store.record.State.DomainProposal.Payload) {
		t.Fatalf("exact replay must retain domain proposal without redispatch: calls=%d err=%v", model.calls, err)
	}
}

func TestRuntimeDomainProposalRejectsWrongOwnerAndModelConfidence(t *testing.T) {
	for _, scenario := range []string{"missing", "wrong kind", "mixed product", "product warnings", "product rejections", "confidence", "tool result", "invalid JSON"} {
		t.Run(scenario, func(t *testing.T) {
			action := domainProposalFixture()
			switch scenario {
			case "missing":
				action.DomainProposal = nil
			case "wrong kind":
				action.DomainProposal.Kind = "product-content-v1"
			case "mixed product":
				action.Candidate = proposal("supported title").Candidate
			case "product warnings":
				if err := json.Unmarshal([]byte(`{"Candidate":{"Warnings":[{"Code":"unexpected"}]}}`), &action); err != nil {
					t.Fatal(err)
				}
			case "product rejections":
				if err := json.Unmarshal([]byte(`{"Candidate":{"Rejections":[{"Code":"unexpected"}]}}`), &action); err != nil {
					t.Fatal(err)
				}
			case "confidence":
				action.Confidence = []agent.FieldConfidence{{Field: "categoryId", Known: true, Value: 1}}
			case "tool result":
				action.Kind = "tool"
			case "invalid JSON":
				action.DomainProposal.Payload = json.RawMessage(`{"categoryId":`)
			}
			runtime, request, _, validator, _ := listingDomainFixture(t, action)
			record, err := runtime.Start(context.Background(), request)
			if err != nil {
				t.Fatal(err)
			}
			if record.State.StopReason != agent.StopInvalidOutput || validator.calls != 0 || record.State.DomainProposal != nil {
				t.Fatalf("unadmitted domain result must not reach a validator or durable proposal: reason=%q calls=%d", record.State.StopReason, validator.calls)
			}
		})
	}
}

func TestRuntimeDomainValidatorConfigurationIsExclusive(t *testing.T) {
	runtime, _, _, _, _, _, _ := fixture(t, proposal("supported title"))
	config := runtime.config
	config.DomainProposalKind = "listing-materials-v1"
	config.DomainValidator = &domainValidatorFixture{}
	if _, err := New(config); err == nil {
		t.Fatal("two fact-owner validators must not be admitted")
	}
	config.DomainProposalKind = ""
	if _, err := New(config); err == nil {
		t.Fatal("domain validator needs a fixed consumer kind")
	}
}

func TestRuntimeProductStateOmitsDomainProposal(t *testing.T) {
	runtime, request, _, _, _, _, _ := fixture(t, proposal("supported title"))
	record, err := runtime.Start(context.Background(), request)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := json.Marshal(record.State)
	if err != nil {
		t.Fatal(err)
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(raw, &fields); err != nil {
		t.Fatal(err)
	}
	if _, present := fields["DomainProposal"]; present {
		t.Fatal("existing Product state must not gain a new result field")
	}
}
