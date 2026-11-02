package goods

import (
	sheinvalidator "task-processor/internal/marketplace/shein/validator"
	contract "task-processor/internal/marketplace/validator"
	"task-processor/internal/product/asset"
)

// OfficialAssessment contains one native build and its actual input identity.
// It is transient; projection failure is never a new save or publish gate.
type OfficialAssessment struct {
	result  OfficialDraft
	digests [5]string
	err     error
}

func AssessOfficial(draft OfficialDraftInput, rules OfficialRuleSnapshot, inventory asset.ApprovedAssetInventory, images []OfficialImageObservation, proofs []OfficialStockProofObservation) OfficialAssessment {
	a := OfficialAssessment{result: BuildOfficial(draft, rules, inventory, images, proofs)}
	for n, input := range []any{rules, inventory, draft, images, proofs} {
		digest, err := sheinvalidator.OfficialContentDigest(input)
		if err != nil {
			a.err = err
			break
		}
		a.digests[n] = digest
	}
	return a
}
func (a OfficialAssessment) Native() OfficialDraft {
	result := a.result
	if result.Issues != nil {
		result.Issues = append([]OfficialIssue{}, result.Issues...)
	}
	return result
}

// Diagnostic uses that same build. Its native result stays available even if
// checked input encoding, public metadata or report serialization fails.
func (a OfficialAssessment) Diagnostic(request contract.BoundRequest[sheinvalidator.OfficialIdentity]) (contract.DiagnosticResult, error) {
	if a.err != nil {
		return contract.DiagnosticResult{}, a.err
	}
	findings := make([]contract.Check, 0, len(a.result.Issues))
	for _, issue := range a.result.Issues {
		findings = append(findings, contract.Check{Rule: "official_draft.issue", Code: issue.Code, Category: sheinvalidator.OfficialDraftScope, Status: contract.CheckBlocking, Paths: []string{issue.Field}, Message: issue.Message})
	}
	input := sheinvalidator.OfficialEvaluation{Identity: request.Input, RulesDigest: a.digests[0], InventoryDigest: a.digests[1], DraftDigest: a.digests[2], ImageObservationsDigest: a.digests[3], ProofObservationsDigest: a.digests[4], ReadyForUpload: a.result.ReadyForUpload, Findings: findings}
	return sheinvalidator.ProjectOfficialDiagnostic(contract.BoundRequest[sheinvalidator.OfficialEvaluation]{Input: input, Target: request.Target, Action: request.Action, RuleVersion: request.RuleVersion, BindingVersion: request.BindingVersion, ExpectedDigest: request.ExpectedDigest, ReadAt: request.ReadAt, EvaluatedAt: request.EvaluatedAt, Freshness: request.Freshness})
}
