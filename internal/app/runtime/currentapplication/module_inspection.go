package currentapplication

import (
	"context"
	"encoding/json"
	"gorm.io/gorm"
	workbenchstore "task-processor/internal/integration/persistence/aiworkbench"
	projectstore "task-processor/internal/integration/persistence/aiworkbench/projectcenter"
	kernelstore "task-processor/internal/integration/persistence/browserkernel"
	datastore "task-processor/internal/integration/persistence/dataservice"
	ecostore "task-processor/internal/integration/persistence/ecoservices"
	knowledgestore "task-processor/internal/integration/persistence/knowledge"
	notificationstore "task-processor/internal/integration/persistence/notificationcenter"
	acquisitionstore "task-processor/internal/integration/persistence/product/acquisition"
	reportstore "task-processor/internal/integration/persistence/reportcenter"
	toolstore "task-processor/internal/integration/persistence/toolmarket"
	"task-processor/internal/platform/database"
)

// InstallModule is deployment metadata, never an authorization or installed fact.
type InstallModule struct {
	ID     string `json:"id"`
	Label  string `json:"label"`
	UIFlag string `json:"uiFlag,omitempty"`
	Toggle string `json:"toggle"`
}

func InstallationModules() []InstallModule {
	return []InstallModule{
		{"aiWorkbench", "AI工作台", "", "enabled"}, {"projectCenter", "项目中心", "", "presence"},
		{"marketResearch", "市场调研与智能选品", "", "presence"}, {"productCollections", "自有商品与商品资料", "LISTINGKIT_PRODUCT_COLLECTIONS_ENABLED", "boolean"},
		{"dataServices", "数据服务", "LISTINGKIT_DATA_SERVICES_ENABLED", "presence"}, {"acquisitionPlans", "采集计划", "LISTINGKIT_ACQUISITION_PLANS_ENABLED", "boolean"},
		{"supplyCompletionDrafts", "待补全资料", "LISTINGKIT_SUPPLY_COMPLETION_DRAFTS_ENABLED", "boolean"}, {"sourceMaterials", "原素材与图库", "LISTINGKIT_SOURCE_MATERIALS_ENABLED", "presence"},
		{"supplyChain", "正式商品刊登", "LISTINGKIT_SUPPLY_CHAIN_ENABLED", "presence"}, {"controlledLocalDraft", "本地测试草稿", "LISTINGKIT_CONTROLLED_LOCAL_DRAFT_ENABLED", "enabled"},
		{"knowledge", "知识库", "LISTINGKIT_KNOWLEDGE_ENABLED", "enabled"}, {"notificationCenterDatabase", "通知", "LISTINGKIT_NOTIFICATION_CENTER_ENABLED", "presence"},
		{"reportCenter", "报告", "LISTINGKIT_REPORT_CENTER_ENABLED", "presence"}, {"toolMarket", "工具市场", "LISTINGKIT_TOOL_MARKET_ENABLED", "presence"},
		{"ecoservices", "生态服务", "LISTINGKIT_ECOSERVICES_ENABLED", "enabled"}, {"productAgent", "商品智能体", "LISTINGKIT_PRODUCT_AGENT_ENABLED", "enabled"},
		{"browserKernel", "浏览器内核", "", "presence"}, {"fulfillment", "订单履约", "", "presence"}, {"pod", "POD设计", "LISTINGKIT_SDS_POD_ENABLED", "presence"},
	}
}

type ModuleInspection struct {
	InstallModule
	Configured  bool   `json:"configured"`
	Enabled     bool   `json:"enabled"`
	Installed   string `json:"installed"`
	Ready       bool   `json:"ready"`
	Note        string `json:"note"`
	CheckStatus string `json:"checkStatus"`
}
type ReadOnlyModuleOpener func(context.Context, DatabaseConfig) (*gorm.DB, func(), error)

// InspectModules only opens read-only runtime pools and calls existing verifiers.
// It deliberately never constructs Run, workers, providers or an installer.
func InspectModules(ctx context.Context, cfg *Config, open ReadOnlyModuleOpener) []ModuleInspection {
	if open == nil {
		open = func(ctx context.Context, c DatabaseConfig) (*gorm.DB, func(), error) {
			db, err := database.OpenExistingReadOnlyContext(ctx, &database.Config{Host: c.Host, Port: c.Port, User: c.User, Password: c.Password, Database: c.Database, MaxConnections: 1, MaxIdleConnections: 1})
			if err != nil {
				return nil, func() {}, err
			}
			s, err := db.DB()
			if err != nil {
				return nil, func() {}, err
			}
			return db, func() { _ = s.Close() }, nil
		}
	}
	raw, _ := json.Marshal(cfg)
	var fields map[string]json.RawMessage
	_ = json.Unmarshal(raw, &fields)
	result := []ModuleInspection{}
	for _, definition := range InstallationModules() {
		v := fields[definition.ID]
		m := ModuleInspection{InstallModule: definition, Configured: len(v) > 0 && string(v) != "null", Installed: "unverified", CheckStatus: "unavailable", Note: "安装状态尚未核实；启用后仍需账户权限和外部服务配置"}
		if definition.Toggle == "boolean" {
			m.Configured = cfg.ProductAcquisitionDatabase != nil
			_ = json.Unmarshal(v, &m.Enabled)
		} else if m.Configured {
			m.Enabled = true
			if definition.Toggle == "enabled" {
				var flags struct {
					Enabled bool `json:"enabled"`
				}
				_ = json.Unmarshal(v, &flags)
				m.Enabled = flags.Enabled
			}
		}
		db, verify := inspectionOwner(cfg, definition.ID)
		if !m.Configured {
			m.Installed = "missing_configuration"
			m.Note = "缺少必要配置；请联系维护人员提供本系统的配置文件"
		} else if db != nil && verify != nil {
			pool, closePool, err := open(ctx, *db)
			if err == nil {
				err = verify(ctx, pool)
			}
			if closePool != nil {
				closePool()
			}
			if err == nil {
				m.CheckStatus = "passed"
				m.Installed = "verified"
				m.Ready = m.Enabled
				m.Note = "必需的数据和访问权限已核实；还需在业务系统中配置外部服务、AI使用额度及账户权限"
				if definition.Toggle == "boolean" && !m.Enabled {
					m.Installed = "unverified"
					m.Note = "当前访问权限检查通过；此功能未启用，所需数据是否齐全仍待核实"
				}
			} else {
				m.CheckStatus = "rejected"
				m.Note = "安装检查未通过；缺少必要数据或访问权限，请联系维护人员处理"
			}
		}
		result = append(result, m)
	}
	return result
}
func inspectionOwner(c *Config, id string) (*DatabaseConfig, func(context.Context, *gorm.DB) error) {
	switch id {
	case "aiWorkbench":
		if c.AIWorkbench != nil {
			return &c.AIWorkbench.Database, workbenchstore.VerifySchema
		}
	case "projectCenter":
		if c.ProjectCenter != nil {
			return &c.ProjectCenter.Database, projectstore.VerifySchema
		}
	case "notificationCenterDatabase":
		return c.NotificationCenterDatabase, notificationstore.VerifySchema
	case "knowledge":
		if c.Knowledge != nil {
			return &c.Knowledge.Database, knowledgestore.VerifyRuntime
		}
	case "reportCenter":
		if c.ReportCenter != nil {
			return &c.ReportCenter.Database, reportstore.VerifySchema
		}
	case "toolMarket":
		if c.ToolMarket != nil {
			return &c.ToolMarket.Database, toolstore.VerifySchema
		}
	case "dataServices":
		if c.DataServices != nil {
			return &c.DataServices.Database, datastore.VerifySchema
		}
	case "ecoservices":
		if c.Ecoservices != nil {
			return &c.Ecoservices.Database, ecostore.VerifyRuntime
		}
	case "browserKernel":
		if c.BrowserKernel != nil {
			return &c.BrowserKernel.Database, kernelstore.VerifySchema
		}
	case "productCollections", "acquisitionPlans", "supplyCompletionDrafts", "supplyChain", "controlledLocalDraft":
		return c.ProductAcquisitionDatabase, func(ctx context.Context, d *gorm.DB) error {
			return acquisitionstore.VerifyRuntimePermissions(ctx, d, acquisitionstore.RuntimeCapabilities{Collections: c.ProductCollections, AcquisitionPlans: c.AcquisitionPlans, CompletionDrafts: c.SupplyCompletionDrafts, SupplyChain: c.SupplyChain != nil || c.PrivateDraftTrial != nil || c.ControlledLocalDraft != nil, SupplyMarket: c.SupplyMarket != nil, POD: c.POD != nil, ImageSets: c.ImageAgent != nil && c.ImageAgent.WorkerConfigFile != "", OfflineRecords: c.SheinOfflineRecords != nil})
		}
	}
	return nil, nil
}
