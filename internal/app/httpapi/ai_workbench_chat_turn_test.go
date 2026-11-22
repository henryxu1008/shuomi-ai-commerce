package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/aicapability"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/aiworkbench"
	"github.com/stretchr/testify/require"
)

func TestChatIntentUsageHasItsOwnBusinessPurpose(t *testing.T) {
	fact := aicapability.InvocationRecord{Operation: aicapability.OperationAIWorkbenchChatIntent}
	require.Equal(t, "chat_intent", usageInvocationPurpose(fact))
	fact.BusinessTaskID = "unrelated-task"
	require.Empty(t, usageInvocationPurpose(fact))
}

func TestChatTurnNormalizesExplicitSkillAndEditedSnapshot(t *testing.T) {
	ref := &aiworkbench.ChatSkillReference{Version: 2, Digest: strings.Repeat("a", 64)}
	input := aiworkbench.ChatTurnInput{Content: "保留型号，清理空格", Skill: ref, Revision: &aiworkbench.ChatTurnRevision{
		Root: uuid.NewString(), BaseRevision: uuid.NewString(), Snapshot: json.RawMessage(`{"instructions":"保留型号","name":"title","displayName":"标题","description":"整理"}`)}}
	normalized, err := normalizeChatTurn(input)
	require.NoError(t, err)
	require.JSONEq(t, string(input.Revision.Snapshot), string(normalized.Revision.Snapshot))
	input.Revision.Snapshot = json.RawMessage(`{"instructions":"a","instructions":"b"}`)
	_, err = normalizeChatTurn(input)
	require.Error(t, err)
	input.Revision = nil
	input.Skill.Digest = "not-a-digest"
	_, err = normalizeChatTurn(input)
	require.Error(t, err)
}

func TestChatTurnDoesNotInferManualSkillFromMessage(t *testing.T) {
	input, err := normalizeChatTurn(aiworkbench.ChatTurnInput{Content: "帮我创建标题技能"})
	require.NoError(t, err)
	require.Nil(t, input.Skill)
	_, err = normalizeChatTurn(aiworkbench.ChatTurnInput{Content: "", Skill: &aiworkbench.ChatSkillReference{Version: 2, Digest: strings.Repeat("a", 64)}})
	require.Error(t, err)
}

func TestOversizeChatTurnRevisionReturnsProofBeforeAcceptance(t *testing.T) {
	ref := &aiworkbench.ChatSkillReference{Version: 2, Digest: strings.Repeat("a", 64)}
	draft := aiworkbench.TitleSkillDraft{Name: "title", DisplayName: "Title", Description: "Title rules", Instructions: strings.Repeat("中", 16384), Script: &aiworkbench.TitleSkillScript{Description: "Format title"}}
	raw, err := json.Marshal(draft)
	require.NoError(t, err)
	draft.Script.Source = strings.Repeat("a", 65537-len(raw))
	raw, err = json.Marshal(draft)
	require.NoError(t, err)
	require.Len(t, raw, 65537)
	input := aiworkbench.ChatTurnInput{Content: "Revise", Skill: ref, Revision: &aiworkbench.ChatTurnRevision{Root: uuid.NewString(), BaseRevision: uuid.NewString(), Snapshot: raw}}
	_, err = (workbenchChatTurnRoutes{}).Submit(context.Background(), aiworkbench.Scope{}, "", "", "", input)
	require.ErrorIs(t, err, aiworkbench.ErrInvalid)
	var rejected *aiworkbench.MessageNotAcceptedError
	require.ErrorAs(t, err, &rejected)

	input.Revision.Snapshot = json.RawMessage(`{"unknown":"field"}`)
	_, err = (workbenchChatTurnRoutes{}).Submit(context.Background(), aiworkbench.Scope{}, "", "", "", input)
	require.ErrorIs(t, err, aiworkbench.ErrInvalid)
	require.False(t, errors.As(err, &rejected), "unrelated validation errors keep their existing behavior")
}
