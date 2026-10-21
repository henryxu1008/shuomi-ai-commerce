package currentapplication

import (
	"context"
	"errors"
	"slices"
	"task-processor/internal/aicapability"
	d "task-processor/internal/aicapability/routeconfig"
	governed "task-processor/internal/integration/aicapability/einomodel"
	qualified "task-processor/internal/integration/aicapability/routeconfig"
	store "task-processor/internal/integration/persistence/routeconfig"

	"gorm.io/gorm"
)

// Opt-in per database. Bindings are private, deployment-admitted descriptors;
// they do not select a current route. No credentials or published rates live here.
type ModelRoutePricingConfig struct {
	TextWriter  *DatabaseConfig        `json:"textWriter,omitempty"`
	ImageWriter *DatabaseConfig        `json:"imageWriter,omitempty"`
	Bindings    []qualified.Definition `json:"bindings,omitempty"`
}

func (m *ModelRoutePricingConfig) writerConnections(target DatabaseConfig) int {
	required := 0
	for _, writer := range []*DatabaseConfig{m.TextWriter, m.ImageWriter} {
		if writer != nil && writer.Host == target.Host && writer.Port == target.Port && writer.User == target.User {
			required += writer.MaxConnections
		}
	}
	return required
}

func (c *Config) modelRouteDefinitions() ([]qualified.Definition, error) {
	m := c.ModelRoutePricing
	if m == nil {
		return nil, nil
	}
	invalid := errors.New("model route pricing requires enabled original consumers, same database and bounded dedicated writer")
	if m.TextWriter == nil && m.ImageWriter == nil {
		return nil, invalid
	}
	check := func(writer, original *DatabaseConfig) bool {
		return writer != nil && original != nil && writer.validate("modelRoutePricing.writer") == nil && sameDatabaseTarget(*writer, *original) && writer.User == "model_route_config_writer" && writer.User != original.User && writer.MaxConnections <= 2
	}
	defs := []qualified.Definition{}
	anchors := map[d.Scope]governed.RoutePolicy{}
	if m.TextWriter != nil {
		if c.ProductAgent == nil || !c.ProductAgent.Enabled || !check(m.TextWriter, &c.ProductAgent.Database) {
			return nil, invalid
		}
		p := c.ProductAgent
		add := func(op aicapability.Operation, policies map[string]governed.RoutePolicy) {
			for org, policy := range policies {
				copy := policy
				sc := d.Scope{OrganizationID: org, Capability: string(op)}
				anchors[sc] = policy
				defs = append(defs, qualified.Definition{Scope: sc, Text: &copy, MaximumCostMicros: c.modelRouteCostCeiling(string(op))})
			}
		}
		add(aicapability.OperationProductAgentDecision, p.TextPolicies)
		add(aicapability.OperationProductContentDecision, p.ContentTextPolicies)
		if c.SupplyChain != nil {
			add(aicapability.OperationListingMaterialsDecision, p.MaterialsTextPolicies)
		}
		if c.ProductCollections {
			add(aicapability.OperationProductSelectionDecision, p.SelectionTextPolicies)
		}
		if c.StoreCenter != nil && c.StoreCenter.Enabled && c.StoreCenter.OperationsCockpit {
			add(aicapability.OperationShopDiagnosisDecision, p.DiagnosisTextPolicies)
		}
		if c.AIWorkbench != nil && c.AIWorkbench.Enabled && !c.AIWorkbench.ConversationOnly {
			add(aicapability.OperationAIWorkbenchChatPlan, c.AIWorkbench.PlanningTextPolicies)
		}
		if len(anchors) == 0 {
			return nil, invalid
		}
	}
	imageScopes := map[d.Scope]bool{}
	if m.ImageWriter != nil {
		if c.ImageAgent == nil || c.ImageAgent.WorkerConfigFile == "" || !check(m.ImageWriter, &c.ImageAgent.Database) || !c.ImageAgent.Generation.Configured() {
			return nil, invalid
		}
	}
	for _, def := range m.Bindings {
		if def.Image != nil {
			if m.ImageWriter == nil || !slices.Contains(c.ImageAgent.AllowedOrganizationIDs, def.Scope.OrganizationID) || def.Scope.Capability != d.ImageCapability || def.Image.MaximumPointsPerImage > c.ImageAgent.Generation.PointsPerImage {
				return nil, invalid
			}
			imageScopes[def.Scope] = true
		} else {
			anchor, ok := anchors[def.Scope]
			if !ok || def.Text == nil {
				return nil, invalid
			}
			a, p := anchor.ShapeProfile(), def.Text.ShapeProfile()
			if p.PromptVersion != a.PromptVersion || p.OutputSchemaVersion != a.OutputSchemaVersion || p.Currency != a.Currency || p.MaximumInputBytes > a.MaximumInputBytes || p.MaximumOutputBytes > a.MaximumOutputBytes || p.MaximumPromptTokens > a.MaximumPromptTokens || p.MaximumCompletionTokens > a.MaximumCompletionTokens || p.DeadlineBound > a.DeadlineBound {
				return nil, invalid
			}
			def.MaximumCostMicros = c.modelRouteCostCeiling(def.Scope.Capability)
		}
		defs = append(defs, def)
	}
	if m.ImageWriter != nil {
		for _, org := range c.ImageAgent.AllowedOrganizationIDs {
			if !imageScopes[d.Scope{OrganizationID: org, Capability: d.ImageCapability}] {
				return nil, invalid
			}
		}
	}
	if _, e := qualified.NewCatalog(defs); e != nil {
		return nil, e
	}
	return defs, nil
}

func prepareModelRouteServices(ctx context.Context, reader, writer *gorm.DB, defs []qualified.Definition) (*d.Service, *d.Service, error) {
	if reader == nil || writer == nil || reader == writer {
		return nil, nil, d.ErrUnavailable
	}
	imageOptions := false
	for _, def := range defs {
		imageOptions = imageOptions || def.Scope.Capability == d.ImageCapability
	}
	if e := store.VerifyReader(ctx, reader, imageOptions); e != nil {
		return nil, nil, e
	}
	if e := store.VerifyWriter(ctx, writer, imageOptions); e != nil {
		return nil, nil, e
	}
	catalog, e := qualified.NewCatalog(defs)
	if e != nil {
		return nil, nil, e
	}
	read, e := store.New(reader)
	if e != nil {
		return nil, nil, e
	}
	write, e := store.New(writer)
	if e != nil {
		return nil, nil, e
	}
	return &d.Service{Repository: read, Catalog: catalog}, &d.Service{Repository: write, Catalog: catalog}, nil
}

func (c *Config) modelRouteCostCeiling(capability string) int64 {
	if capability == string(aicapability.OperationProductSelectionDecision) || capability == string(aicapability.OperationShopDiagnosisDecision) {
		return c.ProductAgent.CostMicros / 2
	}
	return c.ProductAgent.CostMicros
}
