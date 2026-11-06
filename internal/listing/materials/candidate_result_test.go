package materials

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"task-processor/internal/agent"
	"task-processor/internal/commercetool"
	"task-processor/internal/marketplace/shein/goods"
	model "task-processor/internal/marketplace/shein/model"
	"time"
)

func TestCandidateRulesModelResultIsStrictlyReadOnly(t *testing.T) {
	raw := `{"purpose":"category-rules","explanation":{"schemaVersion":"listing-rule-explanation-v1","summary":"请核对该候选类目的基础模板，关联属性条件尚未评价。","references":[{"evidenceId":"` + strings.Repeat("a", 64) + `","callId":"call","ruleDigest":"` + strings.Repeat("b", 64) + `"}],"unknowns":["未评价具体属性组合"]}}`
	a := agent.Action{Kind: "propose", DomainProposal: &agent.DomainProposal{Kind: ProposalKind, Payload: json.RawMessage(raw)}}
	require.NoError(t, (ModelAdmission{}).ValidateAction(a), "a read-only explanation is a domain result, not a writable Target patch")
	a.DomainProposal.Payload = json.RawMessage(strings.Replace(raw, `"purpose":"category-rules"`, `"purpose":"category-rules","categoryId":123`, 1))
	require.Error(t, (ModelAdmission{}).ValidateAction(a), "query result cannot carry writable material fields")
}

type candidateRulesTestReader struct {
	inspectionFixture
	query CandidateRulesDisclosure
}

func (r candidateRulesTestReader) ReadCandidateRules(context.Context, agent.Binding) (CandidateRulesDisclosure, error) {
	return r.query, nil
}

func TestCandidateRulesValidatorBindsActualRecordedObservation(t *testing.T) {
	i := materialInspection(t)
	base := i.Base
	base.ID, base.Input.StoreID = uuid.NewString(), uuid.NewString()
	base.Merchant.StoreID = base.Input.StoreID
	leaf := true
	q, err := NewCandidateRulesDisclosure(base, i.Product, "13", goods.OfficialRuleSnapshot{Categories: []model.Category{{ID: 13, ProductTypeID: 456, Leaf: &leaf}}, Attributes: model.AttributeTemplate{ProductTypeID: 456}}, time.Now().UTC())
	require.NoError(t, err)
	q.Evidence.CallID = "rules-call"
	raw, err := json.Marshal(q)
	require.NoError(t, err)
	result := DomainResult{Purpose: CategoryRulesPurpose, Explanation: &RuleExplanation{SchemaVersion: RuleExplanationSchema, Summary: "仅解释候选基础规则，具体属性条件未评价。", References: []RuleExplanationRef{{q.EvidenceID, q.Evidence.CallID, q.Evidence.RuleDigest}}, Unknowns: []string{}}}
	payload, err := json.Marshal(result)
	require.NoError(t, err)
	candidate := agent.DomainProposal{Kind: ProposalKind, Payload: payload}
	request := agent.Request{Binding: q.Binding, PolicyVersion: PolicyVersion, PromptVersion: PromptVersion}
	history := []agent.Observation{{Tool: RulesToolRef(), CallID: q.Evidence.CallID, AuditStatus: commercetool.AuditStatusRecorded, Output: raw}}
	v := Validator{Reader: candidateRulesTestReader{inspectionFixture: inspectionFixture{i}, query: q}}
	validation, err := v.ValidateDomain(context.Background(), request, candidate, history)
	require.NoError(t, err)
	require.True(t, validation.Valid, "query has its own read-only fixed validator")
	require.NotEmpty(t, validation.Unresolved, "conditional unknowns remain visible despite valid result shape")
	for _, bad := range []agent.Observation{{Tool: RulesToolRef(), CallID: "other-call", AuditStatus: commercetool.AuditStatusRecorded, Output: raw}, {Tool: InspectToolRef(), CallID: q.Evidence.CallID, AuditStatus: commercetool.AuditStatusRecorded, Output: raw}, {Tool: RulesToolRef(), CallID: q.Evidence.CallID, AuditStatus: commercetool.AuditStatusRecordFailed, Output: raw}} {
		validation, err = v.ValidateDomain(context.Background(), request, candidate, []agent.Observation{bad})
		require.NoError(t, err)
		require.False(t, validation.Valid)
	}
	require.NoError(t, (ModelAdmission{}).ValidateBoundAction(q.Binding, agent.Action{Kind: "propose", DomainProposal: &candidate}))
	require.Error(t, (ModelAdmission{}).ValidateBoundAction(i.Binding, agent.Action{Kind: "propose", DomainProposal: &candidate}))
}
