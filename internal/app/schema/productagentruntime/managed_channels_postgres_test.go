package productagentruntime

import (
	"bytes"
	"context"
	"testing"

	credential "github.com/henryxu1008/shuomi-ai-commerce/internal/aicapability/channelcredential"
	channelstore "github.com/henryxu1008/shuomi-ai-commerce/internal/integration/persistence/channelcredential"
	routestore "github.com/henryxu1008/shuomi-ai-commerce/internal/integration/persistence/routeconfig"
	"github.com/stretchr/testify/require"
)

func TestPrepareManagedChannelsPreservesRouteOwnerBoundary(t *testing.T) {
	ctx := context.Background()
	f := newPrepareFixture(t, "run")
	_, err := PrepareRun(ctx, f.db)
	require.NoError(t, err)
	// The retained route tables belong to the original configuration owner.
	require.NoError(t, routestore.InstallSchema(ctx, f.db))
	require.NoError(t, f.db.Exec(`CREATE ROLE model_route_config_writer NOLOGIN;
 GRANT USAGE ON SCHEMA public TO model_route_config_writer;
 GRANT SELECT ON public.ai_route_config_revisions, public.ai_route_config_current
 TO model_route_config_writer, product_agent_runtime;
 GRANT SELECT ON public.ai_channel_credential_bindings TO model_route_config_writer`).Error)
	tables := []string{"public.ai_route_config_current", "public.ai_route_config_revisions", "public.ai_channel_connections"}
	before := retainedSnapshot(t, f.db, tables)
	var oldRead bool
	require.NoError(t, f.db.Raw("SELECT has_table_privilege('product_agent_owner','public.ai_route_config_current','SELECT')").Scan(&oldRead).Error)
	require.False(t, oldRead)

	result, err := PrepareManagedModelChannels(ctx, f.db)
	require.NoError(t, err)
	require.Equal(t, "CREATED", result.Outcome)
	require.Equal(t, before, retainedSnapshot(t, f.db, tables))
	require.True(t, routestore.ManagedChannelSchemaInstalled(ctx, f.db))

	verified, err := PrepareOwnedGRSaiProductImage(ctx, f.db)
	require.NoError(t, err)
	require.Equal(t, "VERIFIED", verified.Outcome)
	protection, err := channelstore.NewProtection("synthetic-key", bytes.Repeat([]byte{1}, 32))
	require.NoError(t, err)
	channels, err := channelstore.New(f.role(t, "title_credential_writer"), protection)
	require.NoError(t, err)
	legacy, err := credential.NewRegistrationCommand("org-one", "actor", "first", credential.RegistrationInput{
		Template: credential.OfficialMiniTemplate, Name: "existing", APIKey: "synthetic-legacy-key",
	})
	require.NoError(t, err)
	existing, err := channels.CreateRegistration(ctx, legacy)
	require.NoError(t, err)
	require.NoError(t, f.db.Exec(`ALTER TABLE public.ai_channel_registrations
DROP CONSTRAINT ai_channel_registrations_template_check,
ADD CONSTRAINT ai_channel_registrations_template_check CHECK (template IN ('openai-mini-v1','openai-compatible-mini-v1'))`).Error)
	owned, err := credential.NewRegistrationCommand("org-one", "actor", "second", credential.RegistrationInput{
		Template: credential.GRSaiImageTemplate, Name: "enterprise image", APIKey: "synthetic-provider-key",
	})
	require.NoError(t, err)
	_, err = channels.CreateRegistration(ctx, owned)
	require.Error(t, err, "the retained check rejects the new template before explicit preparation")
	updated, err := PrepareOwnedGRSaiProductImage(ctx, f.db)
	require.NoError(t, err)
	require.Equal(t, "UPDATED", updated.Outcome)
	preserved, err := channels.Registration(ctx, existing.Registration.OrganizationID, existing.Registration.ConnectionID)
	require.NoError(t, err)
	require.Equal(t, existing.Registration, preserved)
	created, err := channels.CreateRegistration(ctx, owned)
	require.NoError(t, err)
	require.Equal(t, credential.GRSaiImageTemplate, created.Registration.Template)
	verified, err = PrepareOwnedGRSaiProductImage(ctx, f.db)
	require.NoError(t, err)
	require.Equal(t, "VERIFIED", verified.Outcome)
}
