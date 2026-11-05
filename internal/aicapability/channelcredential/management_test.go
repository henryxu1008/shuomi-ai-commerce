package channelcredential

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

type managementFixture struct {
	current  Metadata
	receipt  Metadata
	commands []Command
	err      error
}

func (f *managementFixture) Metadata(_ context.Context, org, id string) (Metadata, error) {
	if f.current.OrganizationID != org || f.current.ConnectionID != id {
		return Metadata{}, ErrNotFound
	}
	return f.current, nil
}
func (f *managementFixture) Receipt(_ context.Context, org, actor, key string) (Metadata, error) {
	if org != "org-a" || actor != "manager" || key != "original" {
		return Metadata{}, ErrNotFound
	}
	return f.receipt, nil
}
func (f *managementFixture) Publish(_ context.Context, c Command) (Metadata, error) {
	f.commands = append(f.commands, c)
	if f.err != nil {
		return Metadata{}, f.err
	}
	return Metadata{OrganizationID: c.OrganizationID, ConnectionID: c.ConnectionID, Provider: c.Provider, Channel: c.Channel, Account: c.Account, Region: c.Region, BaseURL: c.BaseURL, Revision: c.ExpectedRevision + 1, Enabled: c.Enabled, KeyPresent: true}.WithIdentity(), nil
}
func managementTest(t *testing.T) (*Management, *managementFixture) {
	t.Helper()
	f := &managementFixture{}
	m, e := NewManagement(f, f, []ConnectionDefinition{{OrganizationID: "org-a", ConnectionID: "shared", Provider: "openai", Channel: "openai-direct", Region: "global", BaseURL: "https://api.openai.com/v1", Models: []ManagedModel{{ID: "mini", Version: "v1", Model: "mini", Category: "text"}, {ID: "mini", Version: "v1", Model: "mini", Category: "multimodal"}}}})
	require.NoError(t, e)
	return m, f
}
func TestManagementKeepsTrustedConnectionFactsAndSharedModels(t *testing.T) {
	m, f := managementTest(t)
	ctx := context.Background()
	list, e := m.List(ctx, "org-a")
	require.NoError(t, e)
	require.Len(t, list, 1)
	require.Len(t, list[0].Models, 2)
	require.Zero(t, list[0].Revision)
	_, e = m.Publish(ctx, "org-b", "manager", "shared", "original", ManagementInput{Action: "set-key", Account: "account", APIKey: "synthetic-secret"})
	require.ErrorIs(t, e, ErrNotFound)
	require.Empty(t, f.commands)
	result, e := m.Publish(ctx, "org-a", "manager", "shared", "original", ManagementInput{Action: "set-key", Account: "account", APIKey: "synthetic-secret"})
	require.NoError(t, e)
	require.EqualValues(t, 1, result.Connection.Revision)
	require.Equal(t, "https://api.openai.com/v1", f.commands[0].BaseURL)
	require.Equal(t, "synthetic-secret", f.commands[0].Secret)
	raw, e := json.Marshal(result)
	require.NoError(t, e)
	require.NotContains(t, string(raw), "synthetic-secret")
	require.NotContains(t, string(raw), "api.openai.com")
	require.NotContains(t, string(raw), "fingerprint")
}
func TestManagementReceiptNeedsNoSecretOrCurrentDefinitionAndCannotCrossTarget(t *testing.T) {
	m, f := managementTest(t)
	f.receipt = Metadata{OrganizationID: "org-a", ConnectionID: "shared", Revision: 2, Enabled: true, KeyPresent: true, Account: "account", Provider: "openai", Channel: "openai-direct", Region: "global"}
	m.connections = nil // removed definitions must not rewrite or hide the original receipt
	r, e := m.Receipt(context.Background(), "org-a", "manager", "shared", "original")
	require.NoError(t, e)
	require.EqualValues(t, 1, r.ExpectedRevision)
	for _, x := range []struct{ org, actor, target string }{{"org-b", "manager", "shared"}, {"org-a", "other", "shared"}, {"org-a", "manager", "other"}} {
		_, e = m.Receipt(context.Background(), x.org, x.actor, x.target, "original")
		require.ErrorIs(t, e, ErrNotFound)
	}
}
func TestManagementRotationCannotEditIdentityOrTreatUnknownAsSuccess(t *testing.T) {
	m, f := managementTest(t)
	f.current = Metadata{OrganizationID: "org-a", ConnectionID: "shared", Provider: "openai", Channel: "openai-direct", Region: "global", Account: "account", BaseURL: "https://api.openai.com/v1", Revision: 3, Enabled: true, KeyPresent: true}
	_, e := m.Publish(context.Background(), "org-a", "manager", "shared", "rotate", ManagementInput{Action: "set-key", ExpectedRevision: 3, Account: "other", APIKey: "synthetic"})
	require.ErrorIs(t, e, ErrConflict)
	require.Empty(t, f.commands)
	f.err = ErrOutcomeUnknown
	_, e = m.Publish(context.Background(), "org-a", "manager", "shared", "rotate", ManagementInput{Action: "set-key", ExpectedRevision: 3, APIKey: "synthetic"})
	require.ErrorIs(t, e, ErrOutcomeUnknown)
	require.Len(t, f.commands, 1)
	require.Equal(t, "account", f.commands[0].Account)
	_, e = m.Publish(context.Background(), "org-a", "manager", "shared", "disable", ManagementInput{Action: "disable", ExpectedRevision: 3, APIKey: "must-not-send"})
	require.ErrorIs(t, e, ErrInvalid)
	require.Len(t, f.commands, 1)
}
