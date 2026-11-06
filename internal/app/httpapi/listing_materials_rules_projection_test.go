package httpapi

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"task-processor/internal/agent"
	"task-processor/internal/commercetool"
	"task-processor/internal/listing/materials"
	record "task-processor/internal/listing/record/target"
	"task-processor/internal/marketplace/shein/goods"
	model "task-processor/internal/marketplace/shein/model"
	"task-processor/internal/product/catalog"
	"task-processor/internal/product/collection"
	"task-processor/internal/storecenter"
)

func TestCandidateRulesHistoryUsesActuallyReferencedRead(t *testing.T) {
	product := catalog.PublishedSnapshot{Identity: catalog.SnapshotIdentity{TenantID: "org", ProductKey: "product"}, Version: 1, PublicationID: "publication", Snapshot: catalog.ProductSnapshot{Title: "Original"}}
	base := record.TargetRecord{ID: uuid.NewString(), Revision: 1, EffectiveVersion: 1, ProductHash: collection.Digest(product.Snapshot), Merchant: storecenter.ProductMerchantBinding{OrganizationID: "org", Site: "shein-us"}}
	base.Input.StoreID = uuid.NewString()
	base.Merchant.StoreID = base.Input.StoreID
	base.Source.Source = collection.Source{ProductKey: "product", PublicationID: "publication", Version: 1}
	leaf := true
	q, err := materials.NewCandidateRulesDisclosure(base, product, "123", goods.OfficialRuleSnapshot{Categories: []model.Category{{ID: 123, ProductTypeID: 456, Leaf: &leaf}}, Attributes: model.AttributeTemplate{ProductTypeID: 456}}, time.Now().UTC())
	require.NoError(t, err)
	state := agent.State{Request: agent.Request{Binding: q.Binding, PolicyVersion: materials.PolicyVersion, PromptVersion: materials.PromptVersion}, Phase: agent.HumanReviewRequired}
	for _, call := range []string{"first-read", "second-read"} {
		q.Evidence.CallID = call
		raw, e := json.Marshal(q)
		require.NoError(t, e)
		state.History = append(state.History, agent.Observation{Tool: materials.RulesToolRef(), CallID: call, AuditStatus: commercetool.AuditStatusRecorded, Output: raw})
	}
	explanation := materials.RuleExplanation{SchemaVersion: materials.RuleExplanationSchema, Summary: "Recorded first read", References: []materials.RuleExplanationRef{{EvidenceID: q.EvidenceID, CallID: "first-read", RuleDigest: q.Evidence.RuleDigest}}, Unknowns: []string{}}
	for _, calls := range [][]string{{"first-read"}, {"first-read", "second-read"}} {
		explanation.References = nil
		for _, call := range calls {
			explanation.References = append(explanation.References, materials.RuleExplanationRef{EvidenceID: q.EvidenceID, CallID: call, RuleDigest: q.Evidence.RuleDigest})
		}
		raw, e := json.Marshal(materials.DomainResult{Purpose: materials.CategoryRulesPurpose, Explanation: &explanation})
		require.NoError(t, e)
		state.DomainProposal = &agent.DomainProposal{Kind: materials.ProposalKind, Payload: raw}
		state.Validation = &agent.Validation{Valid: true, PolicyVersion: state.Request.PolicyVersion, CandidateHash: collection.Digest(state.DomainProposal)}
		rules, result, e := recordedCandidateRules(state)
		require.NoError(t, e, "a later successful Tool read must not invalidate references to the earlier recorded read")
		require.Equal(t, "first-read", rules.Evidence.CallID)
		require.Equal(t, explanation.References, result.References)
	}
	state.History = state.History[1:]
	_, _, err = recordedCandidateRules(state)
	require.Error(t, err, "an absent referenced observation cannot be replaced by another successful read")
}
