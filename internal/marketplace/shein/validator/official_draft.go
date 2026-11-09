package validator

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	contract "task-processor/internal/marketplace/validator"
)

const OfficialDraftScope = "shein.official_draft"
const OfficialDraftRuleVersion = "shein.official_draft.v2"
const OfficialDraftBindingVersion = "shein.official-input.go-json.v1"

// OfficialIdentity binds owner-loaded context/Product/merchant facts. The
// caller computes these checked digests; they are never permission or CAS.
// Domain DTOs and all fact acquisition remain outside this pure adapter.
type OfficialIdentity struct {
	ContextDigest          string
	ProductReferenceDigest string
	MerchantDigest         string
}

// OfficialEvaluation is a transient projection of one actual native build.
// The goods owner supplies original findings, not a second rule evaluator.
type OfficialEvaluation struct {
	Identity                                         OfficialIdentity
	RulesDigest, InventoryDigest, DraftDigest        string
	ImageObservationsDigest, ProofObservationsDigest string
	ReadyForUpload                                   bool
	Findings                                         []contract.Check
}

// OfficialContentDigest checks encoding errors instead of silently hashing an
// empty value. It imposes no new ceiling on already bounded owner inputs.
func OfficialContentDigest(input any) (string, error) {
	raw, err := json.Marshal(input)
	if err != nil {
		return "", &contract.Error{Code: contract.EvaluationFailed, Field: "input", Message: "native component encoding failed"}
	}
	sum := sha256.Sum256(raw)
	return "sha256:" + hex.EncodeToString(sum[:]), nil
}

func ProjectOfficialDiagnostic(request contract.BoundRequest[OfficialEvaluation]) (contract.DiagnosticResult, error) {
	if request.Target != (contract.Target{Marketplace: "shein", Site: "shein-us"}) {
		return contract.DiagnosticResult{}, &contract.Error{Code: contract.UnsupportedTarget, Field: "target", Message: "only native SHEIN US is supported"}
	}
	if request.Action != contract.Preview && request.Action != contract.SaveDraft && request.Action != contract.Publish {
		return contract.DiagnosticResult{}, &contract.Error{Code: contract.UnsupportedAction, Field: "action", Message: "explicit native action required"}
	}
	if request.RuleVersion != OfficialDraftRuleVersion || request.BindingVersion != OfficialDraftBindingVersion {
		return contract.DiagnosticResult{}, &contract.Error{Code: contract.UnsupportedVersion, Field: "version", Message: "unsupported native diagnostic version"}
	}
	input := request.Input
	components := []string{input.Identity.ContextDigest, input.Identity.ProductReferenceDigest, input.Identity.MerchantDigest, input.RulesDigest, input.InventoryDigest, input.DraftDigest, input.ImageObservationsDigest, input.ProofObservationsDigest}
	for _, digest := range components {
		if !contract.ValidContentDigest(digest) {
			return contract.DiagnosticResult{}, &contract.Error{Code: contract.InvalidInput, Field: "input", Message: "checked actual native component digests required"}
		}
	}
	// Fixed order is the versioned Go JSON binding, independent of read time.
	subject, err := OfficialContentDigest(struct {
		BindingVersion string          `json:"binding_version"`
		Marketplace    string          `json:"marketplace"`
		Site           string          `json:"site"`
		Action         contract.Action `json:"action"`
		RuleVersion    string          `json:"rule_version"`
		Context        string          `json:"context_digest"`
		Product        string          `json:"product_reference_digest"`
		Merchant       string          `json:"merchant_digest"`
		Rules          string          `json:"rules_digest"`
		Inventory      string          `json:"inventory_digest"`
		Draft          string          `json:"draft_digest"`
		Images         string          `json:"image_observations_digest"`
		Proofs         string          `json:"proof_observations_digest"`
	}{OfficialDraftBindingVersion, request.Target.Marketplace, request.Target.Site, request.Action, OfficialDraftRuleVersion, components[0], components[1], components[2], components[3], components[4], components[5], components[6], components[7]})
	if err != nil {
		return contract.DiagnosticResult{}, err
	}
	bound := contract.BoundInput{Digest: subject, BindingVersion: OfficialDraftBindingVersion, ReadAt: request.ReadAt, EvaluatedAt: request.EvaluatedAt}
	if err := contract.ValidateBoundInput(bound, request.ExpectedDigest); err != nil {
		return contract.DiagnosticResult{}, err
	}
	if request.Freshness.Status != contract.NotEvaluated || request.Freshness.Validate(subject, request.EvaluatedAt) != nil {
		return contract.DiagnosticResult{}, &contract.Error{Code: contract.InvalidInput, Field: "external_freshness", Message: "native diagnostic does not evaluate external freshness"}
	}
	if input.ReadyForUpload != (len(input.Findings) == 0) {
		return contract.DiagnosticResult{}, &contract.Error{Code: contract.EvaluationFailed, Field: "assessment", Message: "inconsistent native build result"}
	}
	status := contract.CheckReady
	if !input.ReadyForUpload {
		status = contract.CheckBlocking
	}
	checks := []contract.Check{{Rule: "official_draft.build", Category: OfficialDraftScope, Status: status, Message: "Aggregate native official draft build result"}}
	for _, finding := range input.Findings {
		if finding.Rule != "official_draft.issue" || finding.Category != OfficialDraftScope || finding.Status != contract.CheckBlocking || finding.Guidance != "" {
			return contract.DiagnosticResult{}, &contract.Error{Code: contract.InvalidInput, Field: "findings", Message: "original native blocking findings required"}
		}
		checks = append(checks, finding)
	}
	offline, err := contract.BuildOfflineChecks(checks)
	if err != nil {
		return contract.DiagnosticResult{}, err
	}
	result := contract.DiagnosticResult{DiagnosticOnly: true, Scope: OfficialDraftScope, Target: request.Target, Action: request.Action, RuleVersion: OfficialDraftRuleVersion, Input: bound, Freshness: request.Freshness.Clone(), NotEvaluated: []string{"live_authorization", "current_head_and_cas", "rule_freshness", "store_publish_permission", "remote_images_and_submission_permit", "remote_business_result"}, OfflineChecks: offline, ActionPolicy: contract.DiagnosticActionPolicy{ReadinessBlockersAllowed: request.Action == contract.SaveDraft}}
	raw, err := json.Marshal(result)
	if err != nil || len(raw) > MaxDiagnosticBytes {
		return contract.DiagnosticResult{}, &contract.Error{Code: contract.EvaluationFailed, Field: "result", Message: "native diagnostic encoding failed or exceeds size limit"}
	}
	return result, nil
}
