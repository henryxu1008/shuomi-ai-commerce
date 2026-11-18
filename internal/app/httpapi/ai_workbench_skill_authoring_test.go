package httpapi

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	workstore "task-processor/internal/integration/persistence/aiworkbench"
	"task-processor/internal/skill"
)

func TestAuthoringQuoteDeadlineIsUniqueToItsServerIssuedKey(t *testing.T) {
	key, err := uuid.NewV7()
	require.NoError(t, err)
	issued, nanos := key.Time().UnixTime()
	expires, err := authoringQuoteDeadline(key.String())
	require.NoError(t, err)
	require.Equal(t, time.Unix(issued, nanos).UTC().Add(authoringQuoteLifetime), expires)
	_, err = authoringQuoteDeadline(uuid.NewString())
	require.Error(t, err)
}

func TestAuthoringQuoteBindsScopeRevisionInputPriceAndExpiry(t *testing.T) {
	signer := &workbenchSkillAuthoring{quoteKey: [32]byte{42}}
	root, revision := "root", "revision"
	row := workstore.AuthoringDraft{OrganizationID: "org", ActorID: "actor", MemberID: "member", ConversationID: "chat", Key: "key",
		SkillVersion: 2, SkillDigest: "locked", InputHash: "input", ProfileDigest: "profile", MaximumPoints: 100,
		Requirement: "modify", InputSnapshot: "draft", RootKey: &root, BaseRevision: &revision, ExpiresAt: time.Now()}
	original := signer.signAuthoringQuote(row)
	require.Len(t, original, 43)
	for _, alter := range []func(*workstore.AuthoringDraft){
		func(r *workstore.AuthoringDraft) { r.MemberID = "other member" },
		func(r *workstore.AuthoringDraft) { r.BaseRevision = nil },
		func(r *workstore.AuthoringDraft) { r.InputSnapshot = "other draft" },
		func(r *workstore.AuthoringDraft) { r.MaximumPoints++ },
		func(r *workstore.AuthoringDraft) { r.ExpiresAt = r.ExpiresAt.Add(time.Second) },
	} {
		tampered := row
		alter(&tampered)
		require.NotEqual(t, original, signer.signAuthoringQuote(tampered))
	}
}

func TestAuthoringPythonDraftBuildsFixedPackageAndPreservesConfirmedBytes(t *testing.T) {
	wire := `{"name":"clean-title","displayName":"空格整理","description":"标题整理","instructions":"保留型号。","script":{"description":"合并重复空格","source":"print('draft code, never execute during authoring')\n"}}`
	draft, err := parseVersionedTitleSkillDraft(wire, 2)
	require.NoError(t, err)
	_, err = parseVersionedTitleSkillDraft(wire, 1)
	require.Error(t, err)
	pkg, err := draft.skillPackage()
	require.NoError(t, err)
	require.Len(t, pkg.Files, 3)
	require.Equal(t, "scripts/main.py", pkg.Execution.Entrypoint)
	require.Equal(t, skill.ScriptProfile, pkg.Execution.Profile)
	files, err := json.Marshal(pkg.Files)
	require.NoError(t, err)
	recovered, err := restoreAuthoringPackage(2, string(files), pkg.Digest)
	require.NoError(t, err)
	require.Equal(t, pkg, recovered)
	_, err = restoreAuthoringPackage(1, string(files), pkg.Digest)
	require.Error(t, err)
	_, err = parseVersionedTitleSkillDraft(`{"name":"clean-title","displayName":"空格整理","description":"标题整理","instructions":"保留型号。","script":{"description":"整理","source":"print('draft')","entrypoint":"/bin/sh"}}`, 2)
	require.Error(t, err)
}

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
