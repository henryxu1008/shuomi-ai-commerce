package setup

import (
	"context"
	"fmt"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/henryxu1008/shuomi-ai-commerce/internal/app/runtime/currentapplication"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/app/setupcontrol"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/platform/privatefile"
)

func TestProjectCenterDisableUsesRuntimeFieldName(t *testing.T) {
	for _, key := range []string{"projectCenter", "ProjectCenter"} {
		t.Run(key, func(t *testing.T) {
			manifest := []byte(fmt.Sprintf(`{
				"schemaVersion":1,
				"listen":{"host":"127.0.0.1","port":18443},
				"identity":{"issuerURL":"http://localhost:18080","authorizationAPIURL":"http://localhost:18080","clientID":"fixture-client","clientSecret":"fixture-secret","projectID":"fixture-project"},
				"sourceAccountDatabase":{"host":"127.0.0.1","port":15432,"user":"source_account_runtime","password":"fixture-source","database":"task_processor","maxConnections":2},
				%q:{"database":{"host":"127.0.0.1","port":15432,"user":"ai_projects_runtime","password":"fixture-project-db","database":"fixture_projects","maxConnections":2}}
			}`, key))
			dir := t.TempDir()
			original, candidate := filepath.Join(dir, "original.json"), filepath.Join(dir, "candidate.json")
			require.NoError(t, privatefile.Write(original, manifest))
			before, err := currentapplication.LoadConfig(original)
			require.NoError(t, err)
			require.NotNil(t, before.ProjectCenter)

			document, err := documentMap(manifest)
			require.NoError(t, err)
			next, err := changeModules(document, nil, map[string]bool{"projectCenter": false})
			require.NoError(t, err)
			require.Equal(t, document["identity"], next["identity"])
			require.Equal(t, document["sourceAccountDatabase"], next["sourceAccountDatabase"])
			require.NoError(t, privatefile.Write(candidate, marshal(next)))
			after, err := currentapplication.LoadConfig(candidate)
			require.NoError(t, err)
			require.False(t, after.ProjectCenter != nil, "entrypoint would still enable the module")

			opened := false
			modules := currentapplication.InspectModules(t.Context(), after, func(context.Context, currentapplication.DatabaseConfig) (*gorm.DB, func(), error) {
				opened = true
				return nil, nil, fmt.Errorf("unexpected module database open")
			})
			require.False(t, opened)
			for _, module := range modules {
				if module.ID == "projectCenter" {
					require.False(t, module.Enabled)
					return
				}
			}
			t.Fatal("project center missing from inspection")
		})
	}
}

func TestModuleCandidateUpdateRecognizesEnabledFieldAndRejectsAmbiguity(t *testing.T) {
	document := map[string]any{"Knowledge": map[string]any{"Enabled": true}}
	next, err := changeModules(document, nil, map[string]bool{"knowledge": false})
	require.NoError(t, err)
	cfg, err := currentapplication.DecodeConfigDocument(marshal(next))
	require.NoError(t, err)
	require.NotNil(t, cfg.Knowledge)
	require.False(t, cfg.Knowledge.Enabled)
	require.Equal(t, map[string]any{"Enabled": false}, next["Knowledge"])
	require.Equal(t, map[string]any{"Enabled": true}, document["Knowledge"])

	provided := map[string]any{"ProjectCenter": map[string]any{}}
	next, err = changeModules(map[string]any{}, provided, map[string]bool{"projectCenter": true})
	require.NoError(t, err)
	cfg, err = currentapplication.DecodeConfigDocument(marshal(next))
	require.NoError(t, err)
	require.NotNil(t, cfg.ProjectCenter)

	for _, ambiguous := range []map[string]any{
		{"knowledge": map[string]any{"enabled": true}, "Knowledge": map[string]any{"enabled": false}},
		{"Knowledge": map[string]any{"enabled": true, "Enabled": false}},
	} {
		_, err := changeModules(ambiguous, nil, map[string]bool{"knowledge": false})
		require.ErrorIs(t, err, setupcontrol.ErrInvalid)
	}
}

func TestModuleStateMustMatchSelectionBeforeAndAfterRestart(t *testing.T) {
	for _, tc := range []struct {
		name      string
		candidate []currentapplication.ModuleInspection
		want      bool
		matches   bool
	}{
		{name: "still enabled", candidate: projectCenterState(true, "verified")},
		{name: "missing", candidate: nil},
		{name: "conflicting states", candidate: append(projectCenterState(false, "missing_configuration"), projectCenterState(true, "verified")...)},
		{name: "disabled", candidate: projectCenterState(false, "missing_configuration"), matches: true},
		{name: "enable but disabled", candidate: projectCenterState(false, "verified"), want: true},
		{name: "enable but unverified", candidate: projectCenterState(true, "unverified"), want: true},
		{name: "enabled and verified", candidate: projectCenterState(true, "verified"), want: true, matches: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.matches, moduleStateMatches(tc.candidate, "projectCenter", tc.want))
		})
	}
}

func projectCenterState(enabled bool, installed string) []currentapplication.ModuleInspection {
	return []currentapplication.ModuleInspection{
		{InstallModule: currentapplication.InstallModule{ID: "projectCenter"}, Enabled: enabled, Installed: installed},
	}
}
