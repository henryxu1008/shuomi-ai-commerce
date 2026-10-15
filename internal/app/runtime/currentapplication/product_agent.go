package currentapplication

import (
	"errors"
	"time"

	"task-processor/internal/agent"
	"task-processor/internal/agentconfig"
	"task-processor/internal/aicapability"
	governed "task-processor/internal/integration/aicapability/einomodel"
)

// ProductAgentConfig is an explicit local trial option. Credential values stay
// in the existing Organization AI credential owner, never in this option.
type ProductAgentConfig struct {
	Enabled                bool                            `json:"enabled"`
	ContentTitleOnly       bool                            `json:"contentTitleOnly,omitempty"`
	Database               DatabaseConfig                  `json:"database"`
	ReviewDatabase         DatabaseConfig                  `json:"reviewDatabase"`
	AssetDatabase          DatabaseConfig                  `json:"assetDatabase"`
	AllowedOrganizationIDs []string                        `json:"allowedOrganizationIds"`
	TextPolicies           map[string]governed.RoutePolicy `json:"textPolicies"`
	ContentTextPolicies    map[string]governed.RoutePolicy `json:"contentTextPolicies,omitempty"`
	MaterialsTextPolicies  map[string]governed.RoutePolicy `json:"materialsTextPolicies,omitempty"`
	DiagnosisTextPolicies  map[string]governed.RoutePolicy `json:"diagnosisTextPolicies,omitempty"`
	SelectionTextPolicies  map[string]governed.RoutePolicy `json:"selectionTextPolicies,omitempty"`
	Currency               string                          `json:"currency"`
	Steps                  int                             `json:"steps"`
	ModelCalls             int                             `json:"modelCalls"`
	Tokens                 int64                           `json:"tokens"`
	CostMicros             int64                           `json:"costMicros"`
	RuntimeSeconds         int                             `json:"runtimeSeconds"`
}

func (p *ProductAgentConfig) Limits() agent.Limits {
	return agent.Limits{Steps: p.Steps, ModelCalls: p.ModelCalls, Tokens: p.Tokens, CostMicros: p.CostMicros, Currency: p.Currency, Runtime: time.Duration(p.RuntimeSeconds) * time.Second}
}

func (p *ProductAgentConfig) validate(cfg *Config) error {
	if !p.Enabled {
		return p.Database.validate("productAgent.database")
	}
	if cfg.ProductAcquisitionDatabase == nil || cfg.CommercialOwnerDatabase == nil {
		return errors.New("product agent requires acquisition and current commercial owner")
	}
	databases := map[string]DatabaseConfig{"database": p.Database, "reviewDatabase": p.ReviewDatabase}
	if p.ContentTitleOnly {
		if p.Database.User != "product_agent_runtime" || p.ReviewDatabase.User != "product_review_runtime" {
			return errors.New("content-title-only requires the prepared Run and Review runtime identities")
		}
		if p.AssetDatabase != (DatabaseConfig{}) || len(p.ContentTextPolicies) == 0 || len(p.TextPolicies) != 0 ||
			len(p.MaterialsTextPolicies) != 0 || len(p.SelectionTextPolicies) != 0 || len(p.DiagnosisTextPolicies) != 0 ||
			cfg.AIWorkbench != nil && (len(cfg.AIWorkbench.PlanningTextPolicies) != 0 || cfg.AIWorkbench.Enabled && !cfg.AIWorkbench.ConversationOnly) {
			return errors.New("content-title-only requires only content policies and no Asset or Chat execution")
		}
	} else {
		databases["assetDatabase"] = p.AssetDatabase
	}
	for name, db := range databases {
		if err := db.validate("productAgent." + name); err != nil {
			return err
		}
		if db.MaxConnections > 8 {
			return errors.New("product agent pools must use at most 8 connections")
		}
	}
	product := cfg.ProductAcquisitionDatabase
	if p.ReviewDatabase.Host != product.Host || p.ReviewDatabase.Port != product.Port || p.ReviewDatabase.Database != product.Database {
		return errors.New("product agent review must use the current Product database")
	}
	if !p.Limits().Valid() || p.RuntimeSeconds > 120 || p.RuntimeSeconds < 1 || p.Steps > 16 || p.ModelCalls > 8 {
		return errors.New("product agent limits or title policy invalid")
	}
	if len(p.AllowedOrganizationIDs) == 0 || len(p.AllowedOrganizationIDs) > 64 {
		return errors.New("product agent organization allowlist required")
	}
	seen := map[string]bool{}
	for _, id := range p.AllowedOrganizationIDs {
		if !agent.ValidID(id) || seen[id] {
			return errors.New("product agent organization allowlist invalid")
		}
		seen[id] = true
	}
	if len(p.TextPolicies)+len(p.ContentTextPolicies)+len(p.MaterialsTextPolicies)+len(p.SelectionTextPolicies)+len(p.DiagnosisTextPolicies) == 0 || len(p.TextPolicies) > len(seen) {
		return errors.New("product agent text policies invalid")
	}
	for organizationID, policy := range p.TextPolicies {
		profile := policy.ShapeProfile()
		costMicros, err := profile.MaximumCost()
		if !seen[organizationID] || profile.PromptVersion != "product-title-agent-v1" || profile.OutputSchemaVersion != "product-title-action-v1" ||
			profile.Currency != p.Currency || !governed.ValidRouteProfile(profile, aicapability.OperationProductAgentDecision) || err != nil ||
			profile.MaximumPromptTokens+profile.MaximumCompletionTokens > p.Tokens || costMicros > p.CostMicros {
			return errors.New("product agent organization text policy invalid")
		}
	}
	if len(p.ContentTextPolicies) > len(seen) {
		return errors.New("content policy outside current organization admission")
	}
	for organizationID, policy := range p.ContentTextPolicies {
		profile := policy.ShapeProfile()
		cost, err := profile.MaximumCost()
		if !seen[organizationID] || profile.PromptVersion != agentconfig.ContentPromptVersion || profile.OutputSchemaVersion != agentconfig.ContentActionVersion || profile.Currency != p.Currency || !governed.ValidRouteProfile(profile, aicapability.OperationProductContentDecision) || err != nil || profile.MaximumPromptTokens+profile.MaximumCompletionTokens > p.Tokens || cost > p.CostMicros {
			return errors.New("product content policy invalid")
		}
	}
	if len(p.MaterialsTextPolicies) > len(seen) {
		return errors.New("platform material policy outside current organization admission")
	}
	for org, policy := range p.MaterialsTextPolicies {
		profile := policy.ShapeProfile()
		cost, err := profile.MaximumCost()
		if !seen[org] || profile.PromptVersion != agentconfig.MaterialsPromptVersion || profile.OutputSchemaVersion != agentconfig.MaterialsActionVersion || profile.Currency != p.Currency || !governed.ValidRouteProfile(profile, aicapability.OperationListingMaterialsDecision) || err != nil || profile.MaximumPromptTokens+profile.MaximumCompletionTokens > p.Tokens || cost > p.CostMicros {
			return errors.New("platform materials text policy invalid")
		}
	}
	if len(p.SelectionTextPolicies) > len(seen) {
		return errors.New("selection policy outside current organization admission")
	}
	for org, policy := range p.SelectionTextPolicies {
		if p.Steps < 4 || p.ModelCalls < 2 {
			return errors.New("product selection requires at least four steps and two model calls")
		}
		profile := policy.ShapeProfile()
		cost, err := profile.MaximumCost()
		if !seen[org] || profile.PromptVersion != agentconfig.SelectionPromptVersion || profile.OutputSchemaVersion != agentconfig.SelectionActionVersion || profile.Currency != p.Currency || !governed.ValidRouteProfile(profile, aicapability.OperationProductSelectionDecision) || err != nil || profile.MaximumCompletionTokens > p.Tokens/2 || profile.MaximumPromptTokens > p.Tokens/2-profile.MaximumCompletionTokens || cost > p.CostMicros/2 {
			return errors.New("product selection text policy invalid")
		}
	}
	if len(p.DiagnosisTextPolicies) > len(seen) {
		return errors.New("diagnosis policy outside current organization admission")
	}
	for org, policy := range p.DiagnosisTextPolicies {
		if p.Steps < 4 || p.ModelCalls < 2 {
			return errors.New("shop diagnosis requires at least four steps and two model calls")
		}
		profile := policy.ShapeProfile()
		cost, err := profile.MaximumCost()
		if !seen[org] || profile.PromptVersion != agentconfig.DiagnosisPromptVersion || profile.OutputSchemaVersion != agentconfig.DiagnosisActionVersion || profile.Currency != p.Currency || !governed.ValidRouteProfile(profile, aicapability.OperationShopDiagnosisDecision) || err != nil || profile.MaximumCompletionTokens > p.Tokens/2 || profile.MaximumPromptTokens > p.Tokens/2-profile.MaximumCompletionTokens || cost > p.CostMicros/2 {
			return errors.New("product diagnosis text policy invalid")
		}
	}
	return nil
}
