package agentconfigpersistence

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"task-processor/internal/agent"
	"task-processor/internal/agentconfig"
	reviewproposal "task-processor/internal/product/review/proposal"
)

func TestContentParameterHashStaysStableAcrossOwnerExtraction(t *testing.T) {
	parameters := agentconfig.ContentTemplate{Schema: agentconfig.ContentParameterSchema, Selection: reviewproposal.ContentSelection{TargetLanguage: "en-US", Title: true, Description: true}}
	require.True(t, parameters.Valid())
	fingerprint, err := hash(parameters)
	require.NoError(t, err)
	require.Equal(t, "599366d7d1624c6049357babf57483bac89c257337c4b14c31fa90c59d5c9626", fingerprint)
}

func TestContentTemplateRoundTripFreezesItsOwnRevision(t *testing.T) {
	_, store, _ := fixture(t)
	ctx := context.Background()
	scope := agent.Scope{OrganizationID: "content-org", ActorID: "content-actor"}
	enable := command(scope, "enable", 0)
	enable.AgentID, enable.Absent = agentconfig.ContentAgentID, true
	_, err := store.Execute(ctx, enable)
	require.NoError(t, err)
	selection, err := reviewproposal.NewContentSelection("en-US", []string{"title", "description"})
	require.NoError(t, err)
	create := command(scope, "create-template", 0)
	create.AgentID = agentconfig.ContentAgentID
	create.Input = agentconfig.TemplateInput{Name: "English content", TargetPlatform: "product", Content: &agentconfig.ContentTemplate{Schema: agentconfig.ContentParameterSchema, Selection: selection}}
	receipt, err := store.Execute(ctx, create)
	require.NoError(t, err)
	original, err := store.ReadTemplate(ctx, scope, create.AgentID, receipt.TemplateID, 1)
	require.NoError(t, err)
	require.Equal(t, agentconfig.ContentParameterSchema, original.SchemaVersion)
	require.Equal(t, *create.Input.Content, *original.Content)
	require.Nil(t, original.Image)
	changed := *create.Input.Content
	changed.Selection.TargetLanguage = "de"
	update := command(scope, "update-template", 1)
	update.AgentID, update.TemplateID, update.Input = create.AgentID, receipt.TemplateID, create.Input
	update.Input.Content = &changed
	_, err = store.Execute(ctx, update)
	require.NoError(t, err)
	old, err := store.ReadTemplate(ctx, scope, create.AgentID, receipt.TemplateID, 1)
	require.NoError(t, err)
	require.Equal(t, original.TemplateInput, old.TemplateInput)
	require.Equal(t, original.Version, old.Version)
	require.Equal(t, "2", old.Revision, "lifecycle revision advances; parameters of version 1 do not")
	latest, err := store.ReadTemplate(ctx, scope, create.AgentID, receipt.TemplateID, 2)
	require.NoError(t, err)
	require.Equal(t, changed, *latest.Content)
	create.AgentID = "product.title.agent"
	_, err = store.Execute(ctx, create)
	require.ErrorIs(t, err, agentconfig.ErrInvalid)
}
