package routeconfigstore

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/henryxu1008/shuomi-ai-commerce/internal/aicapability"
	credential "github.com/henryxu1008/shuomi-ai-commerce/internal/aicapability/channelcredential"
	d "github.com/henryxu1008/shuomi-ai-commerce/internal/aicapability/routeconfig"
	governed "github.com/henryxu1008/shuomi-ai-commerce/internal/integration/aicapability/einomodel"
	qualified "github.com/henryxu1008/shuomi-ai-commerce/internal/integration/aicapability/routeconfig"
	channelstore "github.com/henryxu1008/shuomi-ai-commerce/internal/integration/persistence/channelcredential"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestManagedCompatiblePublicationStopAndReplace(t *testing.T) {
	db, repo := fixture(t)
	ctx := context.Background()
	require.NoError(t, db.Exec(`DO $$ BEGIN
 IF NOT EXISTS(SELECT 1 FROM pg_roles WHERE rolname='product_agent_runtime') THEN CREATE ROLE product_agent_runtime NOLOGIN; END IF;
 IF NOT EXISTS(SELECT 1 FROM pg_roles WHERE rolname='title_credential_writer') THEN CREATE ROLE title_credential_writer NOLOGIN; END IF;
 IF NOT EXISTS(SELECT 1 FROM pg_roles WHERE rolname='model_route_config_writer') THEN CREATE ROLE model_route_config_writer NOLOGIN; END IF;
END $$`).Error)
	require.NoError(t, db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec("SET LOCAL ROLE product_agent_owner").Error; err != nil {
			return err
		}
		return InstallManagedChannels(ctx, tx)
	}))
	require.True(t, ManagedChannelSchemaInstalled(ctx, db))
	require.NoError(t, db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec("SET LOCAL ROLE model_route_config_writer").Error; err != nil {
			return err
		}
		require.True(t, VerifyManagedChannelAccess(ctx, tx, true))
		return nil
	}))
	require.NoError(t, db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec("SET LOCAL ROLE product_agent_runtime").Error; err != nil {
			return err
		}
		require.True(t, VerifyManagedChannelAccess(ctx, tx, false))
		return nil
	}))
	p, err := channelstore.NewProtection("isolated", []byte(strings.Repeat("t", 32)))
	require.NoError(t, err)
	channels, err := channelstore.New(db, p)
	require.NoError(t, err)
	const org, actor = "org-a", "operator"
	const endpoint = "https://fixture.example.test/v1"
	connect := func(key string) credential.RegistrationResult {
		command, err := credential.NewRegistrationCommand(org, actor, key, credential.RegistrationInput{
			Template: credential.CompatibleMiniTemplate, Name: key, BaseURL: endpoint, APIKey: "synthetic-" + key,
		})
		require.NoError(t, err)
		created, err := channels.CreateRegistration(ctx, command)
		require.NoError(t, err)
		return created
	}
	a, b := connect("channel-a"), connect("channel-b")
	shape := aicapability.ModelProfile{ClientName: "anchor", ProviderID: "openai", AdapterKind: string(governed.AdapterOpenAICompatible),
		ModelID: qualified.OpenAIQueryModel, RoutingPolicyVersion: "fixture-route", AdapterPolicyVersion: governed.OpenAIMiniTextAdapterPolicy,
		PromptVersion: "fixture-prompt", OutputSchemaVersion: "fixture-result", UsageMappingVersion: "openai-text-v1", CostPricingVersion: "fixture-cost",
		Currency: "CNY", InputMicrosPerMillion: 6_000_000, OutputMicrosPerMillion: 36_000_000,
		MaximumPromptTokens: 4096, MaximumCompletionTokens: 512, MaximumInputBytes: 65536, MaximumOutputBytes: 16384, DeadlineBound: time.Second}
	scope := d.Scope{OrganizationID: org, Capability: string(aicapability.OperationAIWorkbenchChatPlan)}
	catalog, err := qualified.NewCatalog([]qualified.Definition{{Scope: scope,
		Text:              &governed.RoutePolicy{Profile: shape, AdmittedEndpointIdentityDigest: governed.EndpointIdentityDigest(credential.OfficialOpenAIBaseURL)},
		MaximumCostMicros: 300_000}})
	require.NoError(t, err)
	definitions := &qualified.ManagedDefinitions{Catalog: catalog, Bindings: repo, Registrations: channels, Metadata: channels}
	validate := func(template string) error {
		require.True(t, definitions.Supports(org, template))
		return nil
	}
	bind := func(r credential.RegistrationResult, key string) d.ManagedBinding {
		command := d.ManagedBindingCommand{OrganizationID: org, ActorID: actor, RequestKey: key, ConnectionID: r.Connection.ConnectionID,
			Input: d.ManagedBindingInput{Action: "attach", ExpectedRevision: r.Connection.Revision}}
		binding, err := repo.WriteManagedBinding(ctx, command, validate)
		require.NoError(t, err)
		replayed, err := repo.WriteManagedBinding(ctx, command, validate)
		require.NoError(t, err)
		require.Equal(t, binding, replayed)
		_, err = repo.ManagedBindingReceipt(ctx, "org-b", actor, key)
		require.ErrorIs(t, err, d.ErrNotFound)
		return binding
	}
	ab, bb := bind(a, "attach-a"), bind(b, "attach-b")
	shared := &qualified.SharedCatalog{Catalog: catalog, Managed: definitions, Credentials: repo}
	models := &d.SharedModelService{Catalog: shared, Repository: repo}
	unpriced, err := shared.QualifyModel(ctx, org, d.ModelCategoryText, ab.Reference())
	require.NoError(t, err)
	require.True(t, unpriced.Ready)
	require.Empty(t, unpriced.Price)
	require.Equal(t, aicapability.ModelUsageOnly, unpriced.Text.BillingMode)
	selection := d.SharedModelSelection{SchemaVersion: 1, UnitVersion: aicapability.AIPointMicrounitV1,
		DefaultChannel: credential.CompatibleOpenAIChannel, Text: ab.Reference(),
		Image:      d.ModelBindingReference{ID: "openai-flare", Version: "approved-default-v1"},
		Multimodal: d.ModelBindingReference{ID: "openai-mini", Version: "approved-default-v1"}}
	first, err := models.Publish(ctx, org, actor, "publish-a", d.SharedModelInput{ExpectedRevision: d.NoRevision, Selection: selection})
	require.NoError(t, err)
	reader := qualified.SharedTextReader{Service: models}
	policy, err := reader.ReadPolicy(ctx, governed.RouteKey{OrganizationID: org, Operation: aicapability.OperationAIWorkbenchChatPlan})
	require.NoError(t, err)
	require.Equal(t, a.Connection.ConnectionID, policy.Profile.ConnectionID)
	require.True(t, governed.ValidRouteProfile(policy.Profile, aicapability.OperationAIWorkbenchChatPlan))
	require.Empty(t, policy.Profile.OfficialTariff)
	require.Equal(t, aicapability.ModelUsageOnly, policy.Profile.BillingMode)
	require.Equal(t, aicapability.ReferenceTextTotalsV1, policy.Profile.UsageMappingVersion)
	resolver, err := governed.NewManagedOrganizationRouteResolver(channels, reader)
	require.NoError(t, err)
	route, err := resolver.Resolve(ctx, aicapability.TextInputIdentity{OrganizationID: org, Operation: aicapability.OperationAIWorkbenchChatPlan, Profile: policy.Profile})
	require.NoError(t, err)
	require.Equal(t, endpoint, route.Endpoint)
	require.Equal(t, "synthetic-channel-a", route.APIKey)

	_, err = repo.WriteManagedBinding(ctx, d.ManagedBindingCommand{
		OrganizationID: org, ActorID: actor, RequestKey: "stop", ConnectionID: a.Connection.ConnectionID,
		Input: d.ManagedBindingInput{Action: "disable", BindingID: ab.BindingID, ExpectedRevision: a.Connection.Revision},
	}, nil)
	require.NoError(t, err)
	current, preview, err := models.Current(ctx, org)
	require.NoError(t, err)
	require.Equal(t, first.ID, current.ID)
	require.Len(t, preview.Models, 3)
	require.Equal(t, "MODEL_BINDING_DISABLED", preview.Models[0].Reason)
	_, _, err = models.ReadCategory(ctx, org, d.ModelCategoryText)
	require.ErrorIs(t, err, d.ErrNeedsConfiguration)
	selection.Text = bb.Reference()
	second, err := models.Publish(ctx, org, actor, "publish-b", d.SharedModelInput{ExpectedRevision: first.ID, Selection: selection})
	require.NoError(t, err)
	selection.Text = ab.Reference()
	_, err = models.Publish(ctx, org, actor, "republish-stopped", d.SharedModelInput{ExpectedRevision: second.ID, Selection: selection})
	require.ErrorIs(t, err, d.ErrNeedsConfiguration)
}
