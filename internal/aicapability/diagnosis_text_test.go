package aicapability

import (
	"github.com/stretchr/testify/require"
	"testing"
)

func TestDiagnosisDecisionUsesAnIndependentQuoteAndOriginalUsageRules(t *testing.T) {
	in := TextInputIdentity{OrganizationID: "org", ActorID: "actor", MemberID: "member", AgentRunID: "run", Operation: Operation("shop.operations-diagnosis.decision"), System: "fixed comparison rules", Prompt: "exact selected candidate facts", Profile: testTextProfile()}
	quote, err := QuoteText(in)
	require.NoError(t, err)
	require.True(t, SupportsObservedUsageFailure(in.Operation))
	for _, operation := range []Operation{OperationProductAgentDecision, OperationProductContentDecision, OperationListingMaterialsDecision} {
		other := in
		other.Operation = operation
		otherQuote, err := QuoteText(other)
		require.NoError(t, err)
		require.NotEqual(t, quote.InputHash, otherQuote.InputHash)
	}
	in.AgentRunID = ""
	_, err = QuoteText(in)
	require.Error(t, err)
}
