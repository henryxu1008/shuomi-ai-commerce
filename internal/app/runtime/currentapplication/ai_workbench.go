package currentapplication

import (
	"errors"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/agent"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/aicapability"
	"net"
	"strconv"

	governed "github.com/henryxu1008/shuomi-ai-commerce/internal/integration/aicapability/einomodel"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/integration/aiworkbench/einorouter"
)

// AIWorkbenchConfig uses the Product Agent's logical database with its own
// least-privilege role. Planning policies never borrow title admission.
type AIWorkbenchConfig struct {
	ChatImageTextPolicies      map[string]governed.RoutePolicy `json:"chatImageTextPolicies,omitempty"`
	ChatImageStorage           *KnowledgeStorageConfig         `json:"chatImageStorage,omitempty"`
	PublicWebSearch            bool                            `json:"publicWebSearch,omitempty"`
	Schedules                  *AIWorkbenchSchedulesConfig     `json:"schedules,omitempty"`
	ChatTurnsEnabled           bool                            `json:"chatTurnsEnabled,omitempty"`
	Enabled                    bool                            `json:"enabled"`
	ContentTasks               bool                            `json:"contentTasks,omitempty"`
	ConversationOnly           bool                            `json:"conversationOnly"`
	AllowedOrganizationIDs     []string                        `json:"allowedOrganizationIds,omitempty"`
	Database                   DatabaseConfig                  `json:"database"`
	PlanningTextPolicies       map[string]governed.RoutePolicy `json:"planningTextPolicies"`
	StoreRouteTextPolicies     map[string]governed.RoutePolicy `json:"storeRouteTextPolicies,omitempty"`
	ChatIntentTextPolicies     map[string]governed.RoutePolicy `json:"chatIntentTextPolicies,omitempty"`
	SkillAuthoringTextPolicies map[string]governed.RoutePolicy `json:"skillAuthoringTextPolicies,omitempty"`
}

func (w *AIWorkbenchConfig) validate(cfg *Config) error {
	if w == nil {
		return nil
	}
	if w.PublicWebSearch && (!w.Enabled || cfg.ModelRoutePricing == nil || cfg.ModelRoutePricing.PlatformMaintainerOrganization == "" || !cfg.channelManagementEnabled()) {
		return errors.New("public web search requires platform channels and channel credential management")
	}
	if len(w.ChatImageTextPolicies) > 64 || len(w.ChatImageTextPolicies) > 0 && w.ChatImageStorage == nil {
		return errors.New("Chat image policies require private storage")
	}
	if w.ChatImageStorage != nil {
		if !w.Enabled || w.ConversationOnly || !w.ContentTasks || !cfg.StorageCapacity || cfg.ModelRoutePricing == nil {
			return errors.New("Chat image storage requires a metered, priced content workbench")
		}
		if err := w.ChatImageStorage.validateCapacityIdentity(); err != nil {
			return errors.New("Chat image storage identity is invalid")
		}
		if err := w.ChatImageStorage.validatePrivateEndpoint("Chat image storage"); err != nil {
			return err
		}
	}
	if w.ChatTurnsEnabled && (!w.Enabled || w.ConversationOnly || cfg.ModelRoutePricing == nil) {
		return errors.New("Chat turns require an enabled priced-model workbench")
	}
	if len(w.ChatIntentTextPolicies) > 64 || len(w.ChatIntentTextPolicies) > 0 && (!w.Enabled || w.ConversationOnly || cfg.ModelRoutePricing == nil) || len(w.SkillAuthoringTextPolicies) > 64 || len(w.SkillAuthoringTextPolicies) > 0 && (!w.Enabled || w.ConversationOnly || cfg.ModelRoutePricing == nil) {
		return errors.New("Chat skill authoring requires an enabled model workbench and current model pricing")
	}
	if err := w.validateSchedules(cfg); err != nil {
		return err
	}
	if cfg.ProductAgent != nil && cfg.ProductAgent.ContentTitleOnly &&
		(len(w.PlanningTextPolicies) != 0 || len(w.StoreRouteTextPolicies) != 0 || len(w.ChatIntentTextPolicies) != 0 || w.Enabled && !w.ConversationOnly) {
		return errors.New("content-title-only permits only storage-only Chat")
	}
	if !w.Enabled {
		return w.Database.validate("aiWorkbench.database")
	}
	if w.ContentTasks && (!cfg.ProductCollections || w.ConversationOnly || cfg.ProductAgent == nil || !cfg.ProductAgent.Enabled || len(cfg.ProductAgent.ContentTextPolicies) == 0) || !w.ContentTasks && cfg.ProductAgent != nil && cfg.ProductAgent.ContentOnly && !w.ConversationOnly {
		return errors.New("content tasks require an explicit content execution consumer")
	}
	if w.ConversationOnly {
		// Storage and history remain available when new daily capture is disabled.
		standaloneStorage := cfg.ProductAgent == nil
		if !standaloneStorage && (cfg.ProductAgent == nil || cfg.ProductAgent.Enabled && !cfg.ProductAgent.ContentTitleOnly && !cfg.ProductAgent.ContentOnly) || len(w.PlanningTextPolicies) != 0 || len(w.StoreRouteTextPolicies) != 0 || len(w.ChatIntentTextPolicies) != 0 ||
			w.Database.validate("aiWorkbench.database") != nil || w.Database.User != "ai_workbench_runtime" ||
			!standaloneStorage && (w.Database.User == cfg.ProductAgent.Database.User || w.Database.Host != cfg.ProductAgent.Database.Host ||
				w.Database.Port != cfg.ProductAgent.Database.Port || w.Database.Database != cfg.ProductAgent.Database.Database) ||
			w.Database.MaxConnections > 8 || len(w.AllowedOrganizationIDs) == 0 || len(w.AllowedOrganizationIDs) > 64 {
			return errors.New("conversation-only Chat requires scoped storage without model execution")
		}
		seen := map[string]bool{}
		for _, id := range w.AllowedOrganizationIDs {
			if !agent.ValidID(id) || seen[id] {
				return errors.New("conversation-only Chat organization allowlist invalid")
			}
			seen[id] = true
		}
		return nil
	}
	if cfg.ProductAgent == nil || !cfg.ProductAgent.Enabled || cfg.CommercialOwnerDatabase == nil ||
		w.Database.validate("aiWorkbench.database") != nil || w.Database.User != "ai_workbench_runtime" ||
		w.Database.User == cfg.ProductAgent.Database.User ||
		w.Database.Host != cfg.ProductAgent.Database.Host || w.Database.Port != cfg.ProductAgent.Database.Port ||
		w.Database.Database != cfg.ProductAgent.Database.Database || w.Database.MaxConnections > 8 ||
		len(w.PlanningTextPolicies) == 0 || len(w.PlanningTextPolicies) > 64 {
		return errors.New("AI Workbench requires its own scoped runtime on Product Agent database")
	}
	allowed := make(map[string]bool, len(cfg.ProductAgent.AllowedOrganizationIDs))
	for _, id := range cfg.ProductAgent.AllowedOrganizationIDs {
		allowed[id] = true
	}
	promptVersion := "ai-workbench-chat-plan-v1"
	if w.ContentTasks {
		promptVersion = "ai-workbench-chat-plan-v3"
		if len(w.AllowedOrganizationIDs) == 0 || len(w.AllowedOrganizationIDs) > 64 {
			return errors.New("content tasks require organization admission")
		}
		seen := map[string]bool{}
		for _, id := range w.AllowedOrganizationIDs {
			if !agent.ValidID(id) || seen[id] || !allowed[id] {
				return errors.New("invalid content task admission")
			}
			seen[id] = true
		}
		allowed = seen
	}
	for org, policy := range w.PlanningTextPolicies {
		profile := policy.ShapeProfile()
		if !allowed[org] || !governed.ValidRouteShape(profile, aicapability.OperationAIWorkbenchChatPlan) || (profile.PromptVersion != promptVersion && !(w.ContentTasks && profile.PromptVersion == "ai-workbench-chat-plan-v4")) ||
			profile.OutputSchemaVersion != "ai-workbench-plan-decision-v1" ||
			profile.Currency != cfg.ProductAgent.Currency ||
			profile.MaximumPromptTokens+profile.MaximumCompletionTokens > cfg.ProductAgent.Tokens ||
			profile.MaximumCompletionTokens > 4096 || profile.MaximumInputBytes > 128<<10 ||
			profile.MaximumOutputBytes > 16<<10 {
			return errors.New("AI Workbench planning policy invalid")
		}
		if _, err := profile.CostFor(profile.MaximumPromptTokens, profile.MaximumCompletionTokens); err != nil {
			return errors.New("AI Workbench planning price invalid")
		}
	}
	if len(w.StoreRouteTextPolicies) > 64 {
		return errors.New("AI Workbench store route policies invalid")
	}
	for org, policy := range w.StoreRouteTextPolicies {
		profile := policy.ShapeProfile()
		if !allowed[org] || !governed.ValidRouteShape(profile, aicapability.OperationAIWorkbenchStoreRoute) || profile.PromptVersion != einorouter.PromptVersion || profile.OutputSchemaVersion != einorouter.SchemaVersion || profile.Currency != cfg.ProductAgent.Currency || profile.MaximumPromptTokens+profile.MaximumCompletionTokens > cfg.ProductAgent.Tokens || profile.MaximumCompletionTokens > 4096 || profile.MaximumInputBytes > 128<<10 || profile.MaximumOutputBytes > 16<<10 {
			return errors.New("AI Workbench store route policy invalid")
		}
		if _, err := profile.CostFor(profile.MaximumPromptTokens, profile.MaximumCompletionTokens); err != nil {
			return errors.New("AI Workbench store route price invalid")
		}
	}
	for org, policy := range w.ChatImageTextPolicies {
		profile := policy.ShapeProfile()
		if !allowed[org] || !governed.ValidRouteShape(profile, aicapability.OperationAIWorkbenchChatImages) ||
			profile.PromptVersion != aicapability.ChatImagePromptVersion || profile.OutputSchemaVersion != aicapability.ChatImageAnswerVersion ||
			profile.Currency != cfg.ProductAgent.Currency || profile.MaximumPromptTokens+profile.MaximumCompletionTokens > cfg.ProductAgent.Tokens ||
			profile.MaximumCompletionTokens > 4096 || profile.MaximumOutputBytes > 16<<10 {
			return errors.New("Chat image policy invalid")
		}
		bound := false
		for _, binding := range cfg.ModelRoutePricing.Bindings {
			if binding.Scope.OrganizationID == org && binding.Scope.Capability == string(aicapability.OperationAIWorkbenchChatImages) &&
				binding.Text != nil && binding.ReferenceVersion != "" && binding.ConnectionID != "" && binding.PriceID != "" && binding.MaximumMicropoints > 0 {
				bound = true
				break
			}
		}
		if !bound {
			return errors.New("Chat images require their current priced model binding")
		}
	}
	for org, policy := range w.ChatIntentTextPolicies {
		profile := policy.ShapeProfile()
		if !allowed[org] || !governed.ValidRouteShape(profile, aicapability.OperationAIWorkbenchChatIntent) || profile.PromptVersion != einorouter.ChatPromptVersion || profile.OutputSchemaVersion != einorouter.ChatSchemaVersion || profile.Currency != cfg.ProductAgent.Currency || profile.MaximumPromptTokens+profile.MaximumCompletionTokens > cfg.ProductAgent.Tokens || profile.MaximumCompletionTokens > 4096 || profile.MaximumInputBytes > 128<<10 || profile.MaximumOutputBytes > 16<<10 {
			return errors.New("Chat intent policy invalid")
		}
		bound := false
		for _, binding := range cfg.ModelRoutePricing.Bindings {
			if binding.Scope.OrganizationID == org && binding.Scope.Capability == string(aicapability.OperationAIWorkbenchChatIntent) && binding.Text != nil && binding.ReferenceVersion != "" && binding.ConnectionID != "" && binding.PriceID != "" {
				bound = true
				break
			}
		}
		if !bound {
			return errors.New("Chat intent requires its current priced model binding")
		}
	}
	for org, policy := range w.SkillAuthoringTextPolicies {
		profile := policy.ShapeProfile()
		versionOne := profile.PromptVersion == "ai-workbench-skill-authoring-v1" && profile.OutputSchemaVersion == "personal-title-skill-draft-v1"
		versionTwo := profile.PromptVersion == "ai-workbench-skill-authoring-v2" && profile.OutputSchemaVersion == "personal-title-skill-draft-v2"
		if !allowed[org] || !governed.ValidRouteShape(profile, aicapability.OperationAIWorkbenchSkillAuthoring) ||
			(!versionOne && !versionTwo) ||
			profile.Currency != cfg.ProductAgent.Currency || profile.MaximumPromptTokens+profile.MaximumCompletionTokens > cfg.ProductAgent.Tokens ||
			profile.MaximumCompletionTokens > 4096 || profile.MaximumInputBytes > 128<<10 || profile.MaximumOutputBytes > 16<<10 {
			return errors.New("Chat skill authoring policy invalid")
		}
		hasBinding := false
		for _, binding := range cfg.ModelRoutePricing.Bindings {
			if binding.Scope.OrganizationID == org && binding.Scope.Capability == string(aicapability.OperationAIWorkbenchSkillAuthoring) &&
				binding.Text != nil && binding.ReferenceVersion != "" && binding.ConnectionID != "" && binding.PriceID != "" {
				hasBinding = true
				break
			}
		}
		if !hasBinding {
			return errors.New("Chat skill authoring requires its current priced model binding")
		}
	}
	return nil
}

type AIWorkbenchSchedulesConfig struct {
	StoreDailyReports bool   `json:"storeDailyReports,omitempty"`
	TemporalAddress   string `json:"temporalAddress"`
	TemporalNamespace string `json:"temporalNamespace"`
}

func (w *AIWorkbenchConfig) validateSchedules(cfg *Config) error {
	if w == nil || w.Schedules == nil {
		return nil
	}
	content := cfg != nil && w.ContentTasks && !w.ConversationOnly && cfg.ProductAgent != nil && cfg.ProductAgent.Enabled
	report := cfg != nil && w.Schedules.StoreDailyReports && cfg.StoreCenter != nil && cfg.StoreCenter.Enabled && cfg.StoreCenter.OperationsCockpit && cfg.ReportCenter != nil
	storageOnly := cfg != nil && w.ConversationOnly && !w.ContentTasks
	if !w.Enabled || (!content && !report && !storageOnly) || cfg == nil || cfg.Identity.TenantDirectoryToken == "" || w.Schedules.StoreDailyReports && !report {
		return errors.New("scheduled Workbench requires an admitted execution or storage consumer and exact service IAM")
	}
	host, port, err := net.SplitHostPort(w.Schedules.TemporalAddress)
	value, portErr := strconv.Atoi(port)
	if err != nil || portErr != nil || net.ParseIP(host) == nil || !net.ParseIP(host).IsLoopback() || value < 1 || value > 65535 || !boundedValue(w.Schedules.TemporalNamespace, 128) {
		return errors.New("scheduled Workbench requires explicit bounded loopback Temporal")
	}
	return nil
}
