package httpapi

import (
	"context"
	"task-processor/internal/aicapability"
	governed "task-processor/internal/integration/aicapability/einomodel"
)

// Explicit synthetic shared reader for original isolated consumer regressions.
// Production has no static per-operation fallback.
type syntheticCurrentTextPolicies struct {
	settings *ProductAgentDependencies
	planning map[string]governed.RoutePolicy
}

func (f syntheticCurrentTextPolicies) ReadPolicy(_ context.Context, key governed.RouteKey) (governed.RoutePolicy, error) {
	var policies map[string]governed.RoutePolicy
	switch key.Operation {
	case aicapability.OperationAIWorkbenchChatPlan:
		policies = f.planning
	case aicapability.OperationProductAgentDecision:
		policies = f.settings.TextPolicies
	case aicapability.OperationProductContentDecision:
		policies = f.settings.ContentTextPolicies
	case aicapability.OperationListingMaterialsDecision:
		policies = f.settings.MaterialsTextPolicies
	case aicapability.OperationProductSelectionDecision:
		policies = f.settings.SelectionTextPolicies
	case aicapability.OperationShopDiagnosisDecision:
		policies = f.settings.DiagnosisTextPolicies
	case aicapability.OperationProductImageAnalysis:
		policies = f.settings.ImageAnalysisPolicies
	}
	p := policies[key.OrganizationID]
	p.Profile.ConnectionID = p.Profile.ClientName
	p.Profile.CredentialVersion = p.AdmittedCredentialVersion
	return p, nil
}
