package currentapplication

import (
	"errors"
	"net"
	"strconv"
	"task-processor/internal/agent"
	"task-processor/internal/aicapability"

	governed "task-processor/internal/integration/aicapability/einomodel"
	"task-processor/internal/integration/aiworkbench/einorouter"
)

// AIWorkbenchConfig uses the Product Agent's logical database with its own
// least-privilege role. Planning policies never borrow title admission.
type AIWorkbenchConfig struct {
	Schedules              *AIWorkbenchSchedulesConfig     `json:"schedules,omitempty"`
	Enabled                bool                            `json:"enabled"`
	ContentTasks           bool                            `json:"contentTasks,omitempty"`
	ConversationOnly       bool                            `json:"conversationOnly"`
	AllowedOrganizationIDs []string                        `json:"allowedOrganizationIds,omitempty"`
	Database               DatabaseConfig                  `json:"database"`
	PlanningTextPolicies   map[string]governed.RoutePolicy `json:"planningTextPolicies"`
	StoreRouteTextPolicies map[string]governed.RoutePolicy `json:"storeRouteTextPolicies,omitempty"`
}

func (w *AIWorkbenchConfig) validate(cfg *Config) error {
	if w == nil {
		return nil
	}
	if err := w.validateSchedules(cfg); err != nil {
		return err
	}
	if cfg.ProductAgent != nil && cfg.ProductAgent.ContentTitleOnly &&
		(len(w.PlanningTextPolicies) != 0 || len(w.StoreRouteTextPolicies) != 0 || w.Enabled && !w.ConversationOnly) {
		return errors.New("content-title-only permits only storage-only Chat")
	}
	if !w.Enabled {
		return w.Database.validate("aiWorkbench.database")
	}
	if w.ContentTasks && (!cfg.ProductCollections || w.ConversationOnly || cfg.ProductAgent == nil || !cfg.ProductAgent.Enabled || len(cfg.ProductAgent.ContentTextPolicies) == 0) || !w.ContentTasks && cfg.ProductAgent != nil && cfg.ProductAgent.ContentOnly && !w.ConversationOnly {
		return errors.New("content tasks require an explicit content execution consumer")
	}
	if w.ConversationOnly {
		reportOnly := w.Schedules != nil && w.Schedules.StoreDailyReports && cfg.ProductAgent == nil
		if !reportOnly && (cfg.ProductAgent == nil || cfg.ProductAgent.Enabled && !cfg.ProductAgent.ContentTitleOnly && !cfg.ProductAgent.ContentOnly) || len(w.PlanningTextPolicies) != 0 || len(w.StoreRouteTextPolicies) != 0 ||
			w.Database.validate("aiWorkbench.database") != nil || w.Database.User != "ai_workbench_runtime" ||
			!reportOnly && (w.Database.User == cfg.ProductAgent.Database.User || w.Database.Host != cfg.ProductAgent.Database.Host ||
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
	if !w.Enabled || (!content && !report) || cfg == nil || cfg.Identity.TenantDirectoryToken == "" || w.Schedules.StoreDailyReports && !report {
		return errors.New("scheduled Workbench requires content execution and exact service IAM")
	}
	host, port, err := net.SplitHostPort(w.Schedules.TemporalAddress)
	value, portErr := strconv.Atoi(port)
	if err != nil || portErr != nil || net.ParseIP(host) == nil || !net.ParseIP(host).IsLoopback() || value < 1 || value > 65535 || !boundedValue(w.Schedules.TemporalNamespace, 128) {
		return errors.New("scheduled Workbench requires explicit bounded loopback Temporal")
	}
	return nil
}
