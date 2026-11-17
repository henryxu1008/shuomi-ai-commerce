package httpapi

import (
	"testing"

	"github.com/stretchr/testify/require"
	"task-processor/internal/skill"
)

func TestAuthoringDraftPreservesEditedTextWithoutAddingExecution(t *testing.T) {
	draft, err := parseTitleSkillDraft(`{"name":"title-shorten","displayName":"精简标题","description":"保留事实","instructions":"保留核实后的型号。"}`)
	require.NoError(t, err)
	draft.Instructions = "按已核实资料保留型号与规格。"
	pkg, err := draft.skillPackage()
	require.NoError(t, err)
	require.Equal(t, skill.TitleTarget, pkg.Target)
	require.Len(t, pkg.Files, 1)
	require.Contains(t, pkg.Files[0].Text, draft.Instructions)
	require.Nil(t, pkg.Execution)
	_, err = parseTitleSkillDraft(`{"name":"title-shorten","displayName":"精简标题","description":"保留事实","instructions":"正文","target":"skill-authoring"}`)
	require.Error(t, err)
	draft.Instructions = "[external](https://example.com)"
	_, err = draft.skillPackage()
	require.Error(t, err)
}
