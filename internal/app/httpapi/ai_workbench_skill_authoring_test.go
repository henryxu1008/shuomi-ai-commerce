package httpapi

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"task-processor/internal/aicapability"
	"task-processor/internal/aiworkbench"
	governed "task-processor/internal/integration/aicapability/einomodel"
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

func TestGeneralAuthoringUsesPrivateChatTextPackage(t *testing.T) {
	raw := `{"name":"handoff-notes","displayName":"交接待办","description":"Organize handoff notes.","instructions":"Return an action list."}`
	draft, err := parseAuthoredSkillDraft(raw, 2, skill.ChatTextTarget)
	require.NoError(t, err)
	pkg, err := draft.skillPackageFor(skill.ChatTextTarget)
	require.NoError(t, err)
	require.True(t, pkg.Supports(skill.ChatTextTarget))
	require.False(t, pkg.Supports(skill.TitleTarget))
	require.Nil(t, pkg.Execution)

	withScript := strings.TrimSuffix(raw, "}") + `,"script":{"description":"Try to run code","source":"print(1)"}}`
	_, err = parseAuthoredSkillDraft(withScript, 2, skill.ChatTextTarget)
	require.ErrorIs(t, err, aiworkbench.ErrInvalid)
}

func TestTextSkillReplyCannotCreateProposalOrReturnTools(t *testing.T) {
	reply, err := textSkillEnvelope.parse(`{"mode":"CLARIFY","assistant_text":"整理好的交接事项","goal_summary":""}`)
	require.NoError(t, err)
	require.Equal(t, "整理好的交接事项", reply.AssistantText)
	require.Nil(t, reply.Proposal)
	for _, raw := range []string{
		`{"mode":"READY","assistant_text":"运行任务","goal_summary":"更新商品"}`,
		`{"mode":"CLARIFY","assistant_text":"执行工具","goal_summary":"","tool_calls":[]}`,
	} {
		_, err = textSkillEnvelope.parse(raw)
		require.ErrorIs(t, err, aiworkbench.ErrInvalid)
	}
}

func TestTextSkillReplyNormalizesMarkdownPresentation(t *testing.T) {
	const table = "| 事项 | 负责人 |\\n| --- | --- |\\n| 校对说明书 | 小周 |"
	fence := strings.Repeat("`", 3)
	object := `{"mode":"CLARIFY","assistant_text":"\n` + table + `\n","goal_summary":""}`
	for _, reply := range []string{
		object,
		"\n " + fence + "json\n" + object + "\n" + fence + "\n",
		fence + "JSON\r\n" + object + "\r\n" + fence,
		fence + "\n" + object + "\n" + fence,
		`{"mode":"CLARIFY","assistant_text":"` + fence + `markdown\n` + table + `\n` + fence + `","goal_summary":""}`,
	} {
		result, err := textSkillEnvelope.parse(reply)
		require.NoError(t, err)
		require.Equal(t, strings.ReplaceAll(table, "\\n", "\n"), result.AssistantText)
		require.Equal(t, aiworkbench.PlanClarify, result.Mode)
		require.Nil(t, result.Proposal)
	}
}

func TestTextSkillReplyPreservesIntentionalCodeBlocks(t *testing.T) {
	fence := strings.Repeat("`", 3)
	for _, text := range []string{
		fence + "python\nprint('example')\n" + fence,
		fence + "markdown\n# Sample source\n" + fence + "\n\nMore text\n\n" + fence + "text\nAnother sample\n" + fence,
	} {
		raw, err := json.Marshal(map[string]string{"mode": "CLARIFY", "assistant_text": text, "goal_summary": ""})
		require.NoError(t, err)
		result, err := textSkillEnvelope.parse(string(raw))
		require.NoError(t, err)
		require.Equal(t, text, result.AssistantText)
	}
}

func TestTextSkillReplyStillRejectsInvalidEnvelopes(t *testing.T) {
	fence := strings.Repeat("`", 3)
	valid := `{"mode":"CLARIFY","assistant_text":"待办表","goal_summary":""}`
	for _, reply := range []string{
		fence + "json\n" + valid,
		fence + "json\n" + valid + "\n" + fence + "\n extra",
		fence + "json\n" + valid + "\n" + valid + "\n" + fence,
		fence + "json\n" + `{"mode":"READY","assistant_text":"业务任务","goal_summary":"不允许"}` + "\n" + fence,
		`{"mode":"CLARIFY","assistant_text":"待办表","assistant_text":"覆盖","goal_summary":""}`,
		`{"mode":"CLARIFY","assistant_text":"待办表","goal_summary":"不允许"}`,
		`{"mode":"CLARIFY","assistant_text":"\n \t","goal_summary":""}`,
		`{"mode":"CLARIFY","assistant_text":"` + strings.Repeat("x", 16<<10) + `","goal_summary":""}`,
	} {
		_, err := textSkillEnvelope.parse(reply)
		require.ErrorIs(t, err, aiworkbench.ErrInvalid)
	}
}

func TestTextSkillMarkdownDoesNotNeedModelJSON(t *testing.T) {
	const table = "| 事项 | 负责人 |\n| --- | --- |\n| 校对说明书 | 小周 |"
	fence := strings.Repeat("`", 3)
	for _, answer := range []string{table, "\n" + fence + "markdown\n" + table + "\n" + fence + "\n"} {
		result, err := textSkillMarkdown.parse(answer)
		require.NoError(t, err)
		require.Equal(t, table, result.AssistantText)
		require.Equal(t, aiworkbench.PlanClarify, result.Mode)
		require.Nil(t, result.Proposal)
	}
	const jsonLooking = `{"mode":"READY","goal_summary":"仅作为文字"}`
	result, err := textSkillMarkdown.parse(jsonLooking)
	require.NoError(t, err)
	require.Equal(t, jsonLooking, result.AssistantText)
	require.Equal(t, aiworkbench.PlanClarify, result.Mode)
	require.Nil(t, result.Proposal)
	for _, bad := range []string{" \n", "\xff", strings.Repeat("中", 16<<10)} {
		_, err := textSkillMarkdown.parse(bad)
		require.ErrorIs(t, err, aiworkbench.ErrInvalid)
	}
}

func TestTextSkillRecoveryUsesOnlyFrozenInput(t *testing.T) {
	profile := aicapability.ModelProfile{
		ClientName: "test", ProviderID: "test", AdapterKind: "openai-compatible", ModelID: "test",
		EndpointIdentityDigest: "digest", CredentialVersion: "v1", RoutingPolicyVersion: "v1", AdapterPolicyVersion: "v1",
		PromptVersion: "ai-workbench-chat-plan-v1", OutputSchemaVersion: "ai-workbench-plan-decision-v1", UsageMappingVersion: "v1", CostPricingVersion: "v1",
		OfficialTariff: aicapability.OfficialModelTariff{UnitVersion: aicapability.AIPointMicrounitV1, PriceVersion: "v1", FXVersion: "fixture-fx-identity",
			Currency: "CNY", CNYMicrosPerCurrencyUnit: 1000000, Model: "test", Channel: "fixture-channel", UsageVersion: "openai-text-v1",
			Rates: aicapability.OfficialModelRates{InputText: 10000, OutputText: 10000}},
		Currency: "CNY", InputMicrosPerMillion: 1, OutputMicrosPerMillion: 1, MaximumPromptTokens: 4096, MaximumCompletionTokens: 512,
		MaximumInputBytes: 16 << 10, MaximumOutputBytes: 16 << 10, DeadlineBound: time.Second,
	}
	cmd := aiworkbench.PlanningCommand{Scope: aiworkbench.Scope{OrganizationID: "org", ActorID: "actor"}, MemberID: "member", PlannerInvocationID: "invocation"}
	for _, contract := range []textSkillContract{textSkillMarkdown, textSkillEnvelope} {
		prepared, err := prepareTextSkill(contract, cmd.Scope, cmd.MemberID, cmd.PlannerInvocationID, "交接笔记", "整理待办表格", profile)
		require.NoError(t, err)
		cmd.InputHash = prepared.quote.InputHash
		restored, err := prepareFrozenTextSkill(cmd, "交接笔记", "整理待办表格", profile)
		require.NoError(t, err)
		require.Equal(t, contract.name, restored.contract.name)
		require.Equal(t, prepared.quote, restored.quote)
		require.Equal(t, cmd.PlannerInvocationID, restored.input.InvocationID)
		_, err = prepareFrozenTextSkill(cmd, "替换笔记", "整理待办表格", profile)
		require.ErrorIs(t, err, governed.ErrNotDispatched)
	}
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
