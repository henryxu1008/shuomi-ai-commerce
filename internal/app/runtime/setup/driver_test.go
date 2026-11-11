package setup

import (
	"encoding/json"
	"github.com/stretchr/testify/require"
	"task-processor/internal/app/setupcontrol"
	"testing"
)

func TestCandidateOnlyChangesSelectedModuleAndPreservesOtherFacts(t *testing.T) {
	original := map[string]any{"schemaVersion": float64(1), "knowledge": map[string]any{"enabled": true, "database": map[string]any{"password": "private", "user": "knowledge_runtime"}}, "controlledLocalDraft": map[string]any{"enabled": true, "allowedOrganizationIds": []any{"A"}}, "sentinel": "preserved"}
	candidate, e := changeModules(original, nil, map[string]bool{"knowledge": false})
	require.NoError(t, e)
	require.Equal(t, "preserved", candidate["sentinel"])
	require.Equal(t, original["controlledLocalDraft"], candidate["controlledLocalDraft"])
	require.Equal(t, false, candidate["knowledge"].(map[string]any)["enabled"])
	require.Equal(t, true, original["knowledge"].(map[string]any)["enabled"])
	_, e = changeModules(original, nil, map[string]bool{"not-an-owner": true})
	require.ErrorIs(t, e, setupcontrol.ErrInvalid)
	_, e = changeModules(original, nil, map[string]bool{"toolMarket": true})
	require.ErrorIs(t, e, setupcontrol.ErrInvalid)
}
func TestNewInstallRejectsRetainedResourcesBeforeDispatch(t *testing.T) {
	require.ErrorIs(t, validateNew(setupcontrol.Request{Mode: "fresh", Project: "original", Profile: "base", ApplicationPort: 35444, IdentityPort: 35443, MailPort: 35425}, true), setupcontrol.ErrInvalid)
	require.ErrorIs(t, validateNew(setupcontrol.Request{Mode: "fresh", Project: "new-project", Profile: "base", ApplicationPort: 35444, IdentityPort: 35444, MailPort: 35425}, false), setupcontrol.ErrInvalid)
	require.NoError(t, validateNew(setupcontrol.Request{Mode: "fresh", Project: "new-project", Profile: "base", ApplicationPort: 45444, IdentityPort: 45443, MailPort: 45425}, false))
}
func TestPrivateConfigurationCannotImportForeignIdentityOrDatabase(t *testing.T) {
	original := map[string]any{"identity": map[string]any{"issuer": "https://localhost:45443"}, "sourceAccountDatabase": map[string]any{"host": "127.0.0.1", "port": float64(5433), "database": "source_accounts", "user": "source_runtime", "password": "private"}}
	provided := map[string]any{}
	require.NoError(t, json.Unmarshal(marshal(original), &provided))
	require.True(t, sameDeployment(original, provided))
	provided["knowledge"] = map[string]any{"database": map[string]any{"host": "foreign", "port": float64(5433), "database": "knowledge", "user": "knowledge_runtime", "password": "private"}}
	require.False(t, sameDeployment(original, provided))
	delete(provided, "knowledge")
	provided["identity"] = map[string]any{"issuer": "https://another-instance"}
	require.False(t, sameDeployment(original, provided))
}
