package httpapi

import (
	"context"
	"encoding/json"
	"github.com/google/uuid"
	"task-processor/internal/aicapability"
	credential "task-processor/internal/aicapability/channelcredential"
	d "task-processor/internal/aicapability/routeconfig"
	governed "task-processor/internal/integration/aicapability/einomodel"
	qualified "task-processor/internal/integration/aicapability/routeconfig"
	"task-processor/internal/integration/aiworkbench/einorouter"
	routestore "task-processor/internal/integration/persistence/routeconfig"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// Explicit synthetic shared reader for original isolated consumer regressions.
// Production has no static per-operation fallback.
type syntheticCurrentTextPolicies struct {
	settings   *ProductAgentDependencies
	planning   map[string]governed.RoutePolicy
	storeRoute map[string]governed.RoutePolicy
}

func (f syntheticCurrentTextPolicies) ReadPolicy(_ context.Context, key governed.RouteKey) (governed.RoutePolicy, error) {
	var policies map[string]governed.RoutePolicy
	switch key.Operation {
	case aicapability.OperationAIWorkbenchChatPlan:
		policies = f.planning
	case aicapability.OperationAIWorkbenchStoreRoute:
		policies = f.storeRoute
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

// Only the saved shared-category projection is synthetic. These two business
// regressions use #1078's real SharedTextReader and the original execution and
// resource owners, without a static serving fallback or real configuration.
type syntheticSelectionSharedCategory struct{ settings *ProductAgentDependencies }

// The same native configuration, price and qualification owners used by the
// application. Only database contents/credentials and the provider are synthetic.
func configuredSelectionSharedReader(t *testing.T, settings *ProductAgentDependencies, db *gorm.DB) qualified.SharedTextReader {
	t.Helper()
	ctx, now := context.Background(), time.Now().UTC()
	connection, err := newHTTPChannelFixture(db).owner.Metadata(ctx, "B", "default")
	require.NoError(t, err)
	require.NoError(t, routestore.InstallSchema(ctx, db))
	repo, err := routestore.New(db)
	require.NoError(t, err)
	prices := &d.PlatformPricesService{Repository: repo}
	_, err = prices.Publish(ctx, "offline-platform", uuid.NewString(), d.PlatformPricesInput{ExpectedRevision: d.NoRevision, Prices: []d.PublishedModelPrice{{ID: "offline-reference", Provider: "deepseek", Model: "deepseek-v4.1-flash", Channel: connection.Channel, Region: connection.Region, ServiceType: "standard-sync", Currency: "CNY", UsageVersion: aicapability.ReferenceTextTotalsV1, Rates: aicapability.OfficialModelRates{InputText: 2_000_000, OutputText: 8_000_000}, SourceURL: "https://api-docs.deepseek.com/", VerifiedAt: now, EffectiveFrom: now.Add(-time.Minute), EffectiveUntil: now.Add(time.Hour)}}})
	require.NoError(t, err)
	defs := []qualified.Definition{}
	for op, policies := range map[aicapability.Operation]map[string]governed.RoutePolicy{aicapability.OperationProductResearchDecision: settings.ResearchTextPolicies, aicapability.OperationProductSelectionDecision: settings.SelectionTextPolicies} {
		if policy, ok := policies["B"]; ok {
			copy := policy
			defs = append(defs, qualified.Definition{ConnectionID: connection.ConnectionID, ConnectionProvider: connection.Provider, Manufacturer: "deepseek", Region: connection.Region, ServiceType: "standard-sync", Channel: connection.Channel, PriceID: "offline-reference", ReferenceVersion: "offline-shared-v1", Scope: d.Scope{OrganizationID: "B", Capability: string(op)}, Text: &copy, MaximumCostMicros: 150000, MaximumMicropoints: 10_000_000})
		}
	}
	catalog, err := qualified.NewCatalog(defs)
	require.NoError(t, err)
	shared := &qualified.SharedCatalog{Catalog: catalog, Credentials: repo, Prices: prices}
	service := &d.SharedModelService{Repository: repo, Catalog: shared}
	selection := d.SharedModelSelection{SchemaVersion: 1, UnitVersion: aicapability.AIPointMicrounitV1, DefaultChannel: connection.Channel, Text: d.ModelBindingReference{ID: "default", Version: "offline-shared-v1"}, Image: d.ModelBindingReference{ID: "openai-flare", Version: "approved-default-v1"}, Multimodal: d.ModelBindingReference{ID: "openai-mini", Version: "approved-default-v1"}}
	_, err = service.Publish(ctx, "B", "admin", uuid.NewString(), d.SharedModelInput{ExpectedRevision: d.NoRevision, Selection: selection})
	require.NoError(t, err)
	q, _, err := service.ReadCategory(ctx, "B", d.ModelCategoryText)
	require.NoError(t, err)
	require.True(t, q.Ready)
	require.True(t, q.Price.IsReferenceEstimate())
	_, _, err = service.ReadCategory(ctx, "B", d.ModelCategoryImage)
	require.ErrorIs(t, err, d.ErrNeedsConfiguration, "saving choices never makes unopened image generation executable")
	return qualified.SharedTextReader{Service: service}
}

// The Chat and content consumers use the same real current selection, channel
// metadata, published fixture tariff and SharedTextReader; the provider is local.
func configuredChatSharedReader(t *testing.T, settings *ProductAgentDependencies, db *gorm.DB, planner governed.RoutePolicy, managedTemplate string, platform bool) qualified.SharedTextReader {
	t.Helper()
	ctx, now := context.Background(), time.Now().UTC()
	connection, err := newHTTPChannelFixture(db).owner.Metadata(ctx, "B", "default")
	require.NoError(t, err)
	require.NoError(t, routestore.InstallSchema(ctx, db))
	repo, err := routestore.New(db)
	require.NoError(t, err)
	prices := &d.PlatformPricesService{Repository: repo}
	owned := managedTemplate != "" && !platform
	if owned {
		prices = nil
	} else if managedTemplate == credential.CompatibleMiniTemplate {
		_, err = prices.Publish(ctx, "offline-platform", uuid.NewString(), d.PlatformPricesInput{ExpectedRevision: d.NoRevision, Prices: []d.PublishedModelPrice{{
			ID: "openai-compatible-mini", Provider: "openai", Model: planner.Profile.ModelID, Channel: credential.CompatibleOpenAIChannel,
			Region: connection.Region, ServiceType: "standard-sync", Currency: "CNY", UsageVersion: aicapability.ReferenceTextTotalsV1,
			Rates: aicapability.OfficialModelRates{InputText: 1_000_000, OutputText: 3_000_000}, SourceURL: "https://developers.openai.com/api/docs/pricing",
			VerifiedAt: now, EffectiveFrom: now.Add(-time.Minute), EffectiveUntil: now.Add(time.Hour),
		}}})
	} else if planner.Profile.AdapterPolicyVersion == governed.OpenAIMiniTextAdapterPolicy {
		_, err = prices.Publish(ctx, "offline-platform", uuid.NewString(), d.PlatformPricesInput{ExpectedRevision: d.NoRevision, Prices: []d.PublishedModelPrice{{ID: "chat-fixture-price", Provider: "openai", Model: planner.Profile.ModelID, Channel: connection.Channel, Region: connection.Region, ServiceType: "standard-sync", Currency: "USD", UsageVersion: "openai-text-v1", Rates: aicapability.OfficialModelRates{InputText: 750000, CachedInputText: 75000, OutputText: 4500000}, SourceURL: "https://developers.openai.com/api/docs/pricing", VerifiedAt: now, EffectiveFrom: now.Add(-time.Minute), EffectiveUntil: now.Add(time.Hour)}}, FX: []d.PublishedFXRate{{Currency: "USD", CNYMicrosPerCurrencyUnit: 6719100, SourceURL: "https://www.chinamoney.com.cn/", RateDate: now.Format("2006-01-02"), VerifiedAt: now, EffectiveFrom: now.Add(-time.Minute), EffectiveUntil: now.Add(time.Hour)}}})
	} else {
		_, err = prices.Publish(ctx, "offline-platform", uuid.NewString(), d.PlatformPricesInput{ExpectedRevision: d.NoRevision, Prices: []d.PublishedModelPrice{{ID: "chat-fixture-price", Provider: "openai", Model: planner.Profile.ModelID, Channel: connection.Channel, Region: connection.Region, ServiceType: "standard-sync", Currency: "CNY", UsageVersion: "openai-text-v1", Rates: aicapability.OfficialModelRates{InputText: 10000, OutputText: 20000}, SourceURL: "https://developers.openai.com/api/docs/pricing", VerifiedAt: now, EffectiveFrom: now.Add(-time.Minute), EffectiveUntil: now.Add(time.Hour)}}})
	}
	require.NoError(t, err)
	var defs []qualified.Definition
	policies := map[aicapability.Operation]governed.RoutePolicy{aicapability.OperationAIWorkbenchChatPlan: planner, aicapability.OperationProductContentDecision: settings.ContentTextPolicies["B"]}
	if owned || platform {
		storeRoute := planner
		storeRoute.Profile.PromptVersion, storeRoute.Profile.OutputSchemaVersion = einorouter.PromptVersion, einorouter.SchemaVersion
		policies[aicapability.OperationAIWorkbenchStoreRoute] = storeRoute
	}
	if owned || platform {
		policies[aicapability.OperationAIWorkbenchChatIntent] = fixtureChatIntentPolicy(planner)
		authoring := planner
		authoring.Profile.PromptVersion, authoring.Profile.OutputSchemaVersion = "ai-workbench-skill-authoring-v2", "personal-title-skill-draft-v2"
		authoring.Profile.MaximumPromptTokens = 16_384
		policies[aicapability.OperationAIWorkbenchSkillAuthoring] = authoring
	}
	for op, policy := range policies {
		copy := policy
		defs = append(defs, qualified.Definition{ConnectionID: connection.ConnectionID, ConnectionProvider: connection.Provider, Manufacturer: "openai", Region: connection.Region, ServiceType: "standard-sync", Channel: connection.Channel, PriceID: "chat-fixture-price", ReferenceVersion: "chat-shared-v1", BoundEvidenceVersion: "fixture-bound", Scope: d.Scope{OrganizationID: "B", Capability: string(op)}, Text: &copy, MaximumCostMicros: settings.Limits.CostMicros})
	}
	if title, ok := settings.TextPolicies["B"]; ok && planner.Profile.AdapterPolicyVersion == governed.OpenAIMiniTextAdapterPolicy {
		defs = append(defs, qualified.Definition{ConnectionID: connection.ConnectionID, ConnectionProvider: connection.Provider, Manufacturer: "openai", Region: connection.Region, ServiceType: "standard-sync", Channel: connection.Channel, PriceID: "chat-fixture-price", ReferenceVersion: "chat-shared-v1", BoundEvidenceVersion: "fixture-bound", Scope: d.Scope{OrganizationID: "B", Capability: string(aicapability.OperationProductAgentDecision)}, Text: &title, MaximumCostMicros: settings.Limits.CostMicros})
	}
	catalog, err := qualified.NewCatalog(defs)
	require.NoError(t, err)
	shared := &qualified.SharedCatalog{Catalog: catalog, Credentials: repo, Prices: prices}
	service := &d.SharedModelService{Repository: repo, Catalog: shared}
	selection := d.SharedModelSelection{SchemaVersion: 1, UnitVersion: aicapability.AIPointMicrounitV1, DefaultChannel: connection.Channel, Text: d.ModelBindingReference{ID: "default", Version: "chat-shared-v1"}, Image: d.ModelBindingReference{ID: "openai-flare", Version: "approved-default-v1"}, Multimodal: d.ModelBindingReference{ID: "openai-mini", Version: "approved-default-v1"}}
	if managedTemplate != "" {
		channels := installManagedChatChannelFixture(t, ctx, db, repo, catalog, managedTemplate)
		shared.Managed = channels.definitions
		selection.Text = channels.binding.Reference()
		if platform {
			pool, err := db.DB()
			require.NoError(t, err)
			pool.SetMaxOpenConns(8)
			installPlatformChatChoice(t, ctx, repo, service, shared, channels.manager)
		}
		selection.DefaultChannel = credential.OfficialOpenAIChannel
		if managedTemplate == credential.CompatibleMiniTemplate {
			selection.DefaultChannel = credential.CompatibleOpenAIChannel
		}
	}
	if !platform {
		_, err = service.Publish(ctx, "B", "admin", uuid.NewString(), d.SharedModelInput{ExpectedRevision: d.NoRevision, Selection: selection})
		require.NoError(t, err)
	}
	q, _, err := service.ReadCategory(ctx, "B", d.ModelCategoryText)
	require.NoError(t, err)
	require.True(t, q.Ready, q.Reason)
	if owned {
		require.Equal(t, aicapability.ModelUsageOnly, q.Text.BillingMode)
		require.Empty(t, q.Price)
		view, err := service.TextChoices(ctx, "B")
		require.NoError(t, err)
		require.NotNil(t, view.Current)
		require.Equal(t, aicapability.ModelUsageOnly, view.Current.BillingMode)
		require.NotEmpty(t, view.Current.Quote)
		require.Nil(t, view.Current.Price)
	}
	if _, ok := settings.TextPolicies["B"]; ok && planner.Profile.AdapterPolicyVersion == governed.OpenAIMiniTextAdapterPolicy {
		title := q.TextProfiles[string(aicapability.OperationProductAgentDecision)]
		require.True(t, governed.ValidRouteProfile(title, aicapability.OperationProductAgentDecision), "the title route remains qualified with the same shared selection")
		require.Equal(t, q.Price, title.OfficialTariff)
	}
	require.Equal(t, q.TextProfiles[string(aicapability.OperationAIWorkbenchChatPlan)].OfficialTariff, q.TextProfiles[string(aicapability.OperationProductContentDecision)].OfficialTariff)
	_, _, err = service.ReadCategory(ctx, "A", d.ModelCategoryText)
	require.Error(t, err)
	reader := qualified.SharedTextReader{Service: service}
	plan, err := reader.ReadPolicy(ctx, governed.RouteKey{OrganizationID: "B", Operation: aicapability.OperationAIWorkbenchChatPlan})
	require.NoError(t, err)
	content, err := reader.ReadPolicy(ctx, governed.RouteKey{OrganizationID: "B", Operation: aicapability.OperationProductContentDecision})
	require.NoError(t, err)
	require.Equal(t, plan.Profile.ConnectionID, content.Profile.ConnectionID)
	require.Equal(t, plan.Profile.RoutingPolicyVersion, content.Profile.RoutingPolicyVersion)
	require.Equal(t, plan.Profile.OfficialTariff, content.Profile.OfficialTariff)
	var contentOnly []qualified.Definition
	for _, def := range defs {
		if def.Scope.Capability == string(aicapability.OperationProductContentDecision) {
			contentOnly = append(contentOnly, def)
		}
	}
	partial, err := qualified.NewCatalog(contentOnly)
	require.NoError(t, err)
	partialReader := qualified.SharedTextReader{Service: &d.SharedModelService{Repository: repo, Catalog: &qualified.SharedCatalog{Catalog: partial, Credentials: repo, Prices: prices}}}
	missing, err := partialReader.ReadPolicy(ctx, governed.RouteKey{OrganizationID: "B", Operation: aicapability.OperationAIWorkbenchChatPlan})
	require.True(t, err != nil || missing.Profile.Validate() != nil, "a content binding alone cannot authorize planning")
	shared.Now = func() time.Time { return now.Add(2 * time.Hour) }
	expired, err := reader.ReadPolicy(ctx, governed.RouteKey{OrganizationID: "B", Operation: aicapability.OperationAIWorkbenchChatPlan})
	if owned {
		require.NoError(t, err)
		require.NoError(t, expired.Profile.Validate(), "owned models are admitted even with no reference price or FX owner")
	} else {
		require.True(t, err != nil || expired.Profile.Validate() != nil, "expired published price cannot authorize Chat")
	}
	shared.Now = nil
	return reader
}

func fixtureChatIntentPolicy(planner governed.RoutePolicy) governed.RoutePolicy {
	planner.Profile.PromptVersion, planner.Profile.OutputSchemaVersion = einorouter.ChatPromptVersion, einorouter.ChatSchemaVersion
	return planner
}

type managedChatChannelFixture struct {
	manager     *credential.Management
	definitions *qualified.ManagedDefinitions
	binding     d.ManagedBinding
}

func installManagedChatChannelFixture(t *testing.T, ctx context.Context, db *gorm.DB, repo *routestore.Store, catalog *qualified.Catalog, template string) managedChatChannelFixture {
	t.Helper()
	require.NoError(t, db.Exec(`DO $$ BEGIN
 IF NOT EXISTS(SELECT 1 FROM pg_roles WHERE rolname='product_agent_runtime') THEN CREATE ROLE product_agent_runtime NOLOGIN; END IF;
 IF NOT EXISTS(SELECT 1 FROM pg_roles WHERE rolname='title_credential_writer') THEN CREATE ROLE title_credential_writer NOLOGIN; END IF;
 IF NOT EXISTS(SELECT 1 FROM pg_roles WHERE rolname='model_route_config_writer') THEN CREATE ROLE model_route_config_writer NOLOGIN; END IF;
END $$`).Error)
	require.NoError(t, db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec("SET LOCAL ROLE product_agent_owner").Error; err != nil {
			return err
		}
		return routestore.InstallManagedChannels(ctx, tx)
	}))
	channels := newHTTPChannelFixture(db).owner
	manager, err := credential.NewManagement(channels, channels, nil)
	require.NoError(t, err)
	manager = manager.WithRegistrations(channels, channels)
	before, err := manager.RegisteredConnections(ctx, "B")
	require.NoError(t, err)
	require.Empty(t, before)
	input := credential.RegistrationInput{Name: "Chat fixture", Template: template, APIKey: "managed-fixture-key"}
	if template == credential.CompatibleMiniTemplate {
		input.BaseURL = "https://fixture.compatible.example.test/v1"
	}
	created, err := manager.CreateRegistration(ctx, "B", "admin", uuid.NewString(), input)
	require.NoError(t, err)
	definitions := &qualified.ManagedDefinitions{Catalog: catalog, Bindings: repo, Registrations: channels, Metadata: channels}
	require.True(t, definitions.Supports("B", template))
	if template == credential.OfficialMiniTemplate {
		require.False(t, definitions.Supports("B", credential.CompatibleMiniTemplate), "the official title pilot scope still requires its approved route")
	}
	cmd := d.ManagedBindingCommand{OrganizationID: "B", ActorID: "admin", ConnectionID: created.Connection.ConnectionID,
		RequestKey: uuid.NewString(), Input: d.ManagedBindingInput{Action: "attach", ExpectedRevision: created.Connection.Revision}}
	binding, err := repo.WriteManagedBinding(ctx, cmd, func(template string) error {
		if !definitions.Supports("B", template) {
			return d.ErrNeedsConfiguration
		}
		return nil
	})
	require.NoError(t, err)
	return managedChatChannelFixture{manager: manager, definitions: definitions, binding: binding}
}

func (f syntheticSelectionSharedCategory) ReadCategory(_ context.Context, org string, category d.ModelCategory) (d.ModelQualification, string, error) {
	if category != d.ModelCategoryText {
		return d.ModelQualification{}, "", d.ErrNeedsConfiguration
	}
	q := d.ModelQualification{OrganizationID: org, Category: category, Ready: true, TextProfiles: map[string]aicapability.ModelProfile{}}
	for op, policies := range map[aicapability.Operation]map[string]governed.RoutePolicy{
		aicapability.OperationProductResearchDecision:  f.settings.ResearchTextPolicies,
		aicapability.OperationProductSelectionDecision: f.settings.SelectionTextPolicies,
	} {
		if policy, ok := policies[org]; ok {
			p := policy.ShapeProfile()
			q.TextProfiles[string(op)] = p
			q.Text, q.Model, q.Price = &p, p.ModelID, p.OfficialTariff
			q.Reference = d.ModelBindingReference{ID: p.ClientName, Version: "synthetic-shared-v1"}
		}
	}
	if q.Text == nil {
		return d.ModelQualification{}, "", d.ErrNotFound
	}
	return q, "route-v1", nil
}

func assertSelectionMicropointAccounting(t *testing.T, db *gorm.DB, operation aicapability.Operation, zero bool) {
	t.Helper()
	var rows []struct {
		Operation, OfficialTariffJSON, BillableUsageJSON                  string
		MaximumPromptTokens, MaximumCompletionTokens, EstimatedCostMicros int64
		UsageKnown                                                        bool
	}
	require.NoError(t, db.Table("ai_invocations").Where("tenant_id = ?", "B").Find(&rows).Error)
	require.Len(t, rows, 2, "recovery and history must not create more invocations")
	var totalActual, totalReserved, totalCost int64
	for _, row := range rows {
		require.Equal(t, string(operation), row.Operation)
		require.True(t, row.UsageKnown)
		require.Positive(t, row.MaximumPromptTokens)
		require.LessOrEqual(t, row.MaximumPromptTokens, int64(32000))
		require.LessOrEqual(t, row.MaximumCompletionTokens, int64(4000))
		var tariff aicapability.OfficialModelTariff
		var usage aicapability.BillableModelUsage
		require.NoError(t, json.Unmarshal([]byte(row.OfficialTariffJSON), &tariff))
		require.NoError(t, json.Unmarshal([]byte(row.BillableUsageJSON), &usage))
		require.Equal(t, aicapability.AIPointMicrounitV1, tariff.UnitVersion)
		require.True(t, tariff.IsReferenceEstimate())
		require.Equal(t, tariff.UsageVersion, usage.MappingVersion)
		require.Contains(t, tariff.PriceVersion, ":", "the original published reference version is frozen")
		reserved, err := tariff.ReserveTextMicropoints(row.MaximumPromptTokens, row.MaximumCompletionTokens)
		require.NoError(t, err)
		actual, err := tariff.SettlementMicropoints(usage)
		require.NoError(t, err)
		totalReserved, totalActual, totalCost = totalReserved+reserved, totalActual+actual, totalCost+row.EstimatedCostMicros
	}
	// The original bounded executor may tighten each serialized prompt below
	// the admitted 32k maximum; do not require an unnecessarily large hold.
	require.Positive(t, totalReserved)
	require.LessOrEqual(t, totalReserved, int64(19200000))
	require.LessOrEqual(t, totalReserved, int64(20000000))
	expectedActual, expectedCost := int64(24000), int64(240)
	if zero {
		expectedActual, expectedCost = 0, 0
	}
	require.EqualValues(t, expectedActual, totalActual, "charge explicit fixture usage, not the reserved upper bound")
	require.EqualValues(t, expectedCost, totalCost)
	require.LessOrEqual(t, totalCost, int64(300000))
	var bucket struct {
		QuantityUnit                  string
		Available, Reserved, Consumed int64
	}
	require.NoError(t, db.Table("saas_organization_resource_buckets").Where("organization_id = ? AND resource_type = ?", "B", "ai_point").First(&bucket).Error)
	require.Equal(t, aicapability.AIPointMicrounitV1, bucket.QuantityUnit)
	require.Equal(t, totalActual, bucket.Consumed)
	require.Zero(t, bucket.Reserved)
	require.EqualValues(t, 10000000-totalActual, bucket.Available)
}

func installPlatformChatChoice(t *testing.T, ctx context.Context, repo *routestore.Store, service *d.SharedModelService, catalog *qualified.SharedCatalog, manager *credential.Management) {
	t.Helper()
	platform := &qualified.PlatformChannels{Reader: repo, Writer: repo, Catalog: catalog, Management: manager}
	catalog.Platform = platform
	create := func(name, key string) string {
		made, err := manager.CreateRegistration(ctx, credential.PlatformOwner, "platform", uuid.NewString(), credential.RegistrationInput{Name: name, Template: credential.OfficialMiniTemplate, APIKey: key})
		require.NoError(t, err)
		return made.Connection.ConnectionID
	}
	otherID := create("Other platform channel", "different-fixture-key")
	selectedID := create("Selected platform channel", "managed-fixture-key")
	book, err := catalog.Prices.Current(ctx)
	require.NoError(t, err)
	general := book.Payload.PlatformPrices.Prices[0]
	general.Version = ""
	otherPrice, selectedPrice := general, general
	otherPrice.ID, otherPrice.PlatformConnectionID = "fixture-price-other", otherID
	selectedPrice.ID, selectedPrice.PlatformConnectionID = "fixture-price-selected", selectedID
	otherPrice.Rates.InputText--
	fx := append([]d.PublishedFXRate(nil), book.Payload.PlatformPrices.FX...)
	for i := range fx {
		fx[i].Version = ""
	}
	_, err = catalog.Prices.Publish(ctx, "platform", uuid.NewString(), d.PlatformPricesInput{ExpectedRevision: book.ID, Prices: []d.PublishedModelPrice{general, otherPrice, selectedPrice}, FX: fx})
	require.NoError(t, err)
	publish := func(connection, price string) (d.PlatformChannel, d.Revision) {
		definition := d.PlatformChannel{ConnectionID: connection, PriceID: price, Organizations: []string{"B"}}
		revision, err := platform.Publish(ctx, "platform", uuid.NewString(), qualified.PlatformPublishInput{ExpectedRevision: d.NoRevision, Channel: definition})
		require.NoError(t, err)
		return definition, revision
	}
	retired, old := publish(otherID, otherPrice.ID)
	selected, _ := publish(selectedID, selectedPrice.ID)
	view, err := service.TextChoices(ctx, "B")
	require.NoError(t, err)
	require.Empty(t, view.Current)
	opts := map[d.ModelBindingReference]d.ModelOption{}
	for _, option := range view.Options {
		opts[option.Reference] = option
	}
	require.Equal(t, "Other platform channel", opts[retired.Reference()].ChannelName)
	require.Equal(t, "Selected platform channel", opts[selected.Reference()].ChannelName)
	require.NotEqual(t, opts[retired.Reference()].Quote, opts[selected.Reference()].Quote)
	require.NotEqual(t, opts[retired.Reference()].Price.Rates.InputText, opts[selected.Reference()].Price.Rates.InputText)
	misbound := retired
	misbound.PriceID = selected.PriceID
	_, err = platform.Publish(ctx, "platform", uuid.NewString(), qualified.PlatformPublishInput{ExpectedRevision: old.ID, Channel: misbound})
	require.ErrorIs(t, err, d.ErrNeedsConfiguration)
	rejected, err := service.TextChoices(ctx, "A")
	require.NoError(t, err)
	require.Empty(t, rejected.Options)
	_, err = service.ChooseText(ctx, "A", "admin", uuid.NewString(), d.TextChoiceInput{ExpectedRevision: d.NoRevision, Reference: retired.Reference(), Quote: opts[retired.Reference()].Quote})
	require.ErrorIs(t, err, d.ErrNeedsConfiguration)
	key := uuid.NewString()
	intent := d.TextChoiceInput{ExpectedRevision: d.NoRevision, Reference: retired.Reference(), Quote: opts[retired.Reference()].Quote}
	type result struct {
		revision d.Revision
		err      error
	}
	completed := make(chan result, 2)
	for i := 0; i < 2; i++ {
		go func() {
			revision, err := service.ChooseText(ctx, "B", "admin", key, intent)
			completed <- result{revision, err}
		}()
	}
	firstAttempt, secondAttempt := <-completed, <-completed
	require.NoError(t, firstAttempt.err)
	require.NoError(t, secondAttempt.err)
	require.Equal(t, firstAttempt.revision.ID, secondAttempt.revision.ID)
	first := firstAttempt.revision
	different := intent
	different.Reference = selected.Reference()
	different.Quote = opts[selected.Reference()].Quote
	_, err = service.ChooseText(ctx, "B", "admin", key, different)
	require.ErrorIs(t, err, d.ErrConflict)
	retired.Organizations = []string{}
	_, err = platform.Publish(ctx, "platform", uuid.NewString(), qualified.PlatformPublishInput{ExpectedRevision: old.ID, Channel: retired})
	require.NoError(t, err)
	after, err := service.TextChoices(ctx, "B")
	require.NoError(t, err)
	require.Equal(t, "Other platform channel", after.Current.ChannelName)
	require.Equal(t, "PLATFORM_ACCESS_WITHDRAWN", after.Current.Reason)
	_, _, err = service.ReadCategory(ctx, "B", d.ModelCategoryText)
	require.ErrorIs(t, err, d.ErrNeedsConfiguration)
	last, err := service.ChooseText(ctx, "B", "admin", uuid.NewString(), d.TextChoiceInput{ExpectedRevision: first.ID, Reference: selected.Reference(), Quote: opts[selected.Reference()].Quote})
	require.NoError(t, err)
	require.Empty(t, last.Payload.SharedModels.Image)
	require.Empty(t, last.Payload.SharedModels.Multimodal)
}
