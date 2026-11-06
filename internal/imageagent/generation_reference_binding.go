package imageagent

import (
	"errors"
	"task-processor/internal/aicapability"
)

// ErrReferenceImageBindingStopped is a definite before-send refusal, never an
// ambiguous transaction outcome. Existing-key reads and finalization remain open.
var ErrReferenceImageBindingStopped = errors.New("reference image binding stopped after observed overage")

func (i GenerationIntent) ReferenceImageBindingDigest() string {
	if !i.Billing.OfficialTariff.IsReferenceImageEstimate() {
		return ""
	}
	return generationHash(struct {
		Tenant, Provider, Model, Protocol, Route, Credential, Configuration string
		Billing                                                             ImageBilling
	}{i.Identity.TenantID, i.Provider, i.Model, i.Protocol, i.RouteReference, i.CredentialReference, i.ConfigurationVersion, i.Billing})
}

// ProjectReferenceImageCost only projects the original committed ledger amount
// for budget accounting/display; it cannot debit, reserve or finalize resources.
func (f GenerationFact) ProjectReferenceImageCost(receipt SlotUsageReceipt) (SlotUsageReceipt, error) {
	if !f.Intent.Billing.OfficialTariff.IsReferenceImageEstimate() {
		return receipt, nil
	}
	if f.Validate() != nil || f.State != GenerationSucceeded || f.Settlement.State != "committed" {
		return receipt, ErrRevisionConflict
	}
	receipt.Actual.CostMicros = aicapability.ReferenceImageCNYMicros(f.Settlement.Points)
	receipt.CostBasis = UsageCostReferenceEstimate
	return receipt, nil
}
