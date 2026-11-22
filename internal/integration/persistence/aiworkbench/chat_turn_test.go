package aiworkbenchpersistence

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/henryxu1008/shuomi-ai-commerce/internal/aiworkbench"
	"github.com/stretchr/testify/require"
)

func TestChatTurnSnapshotRetainsOriginalFingerprintAfterJSONB(t *testing.T) {
	const originalSnapshot = `{"name":"title","displayName":"Title","description":"Title rules","instructions":"Keep model number.","script":{"description":"Format title","source":"print('ok')"}}`
	const storedSnapshot = `{"name": "title", "script": {"source": "print('ok')", "description": "Format title"}, "description": "Title rules", "displayName": "Title", "instructions": "Keep model number."}`
	ref := &aiworkbench.ChatSkillReference{Version: 2, Digest: strings.Repeat("a", 64)}
	revision := aiworkbench.ChatTurnRevision{Root: "11111111-1111-4111-8111-111111111111", BaseRevision: "22222222-2222-4222-8222-222222222222", Snapshot: json.RawMessage(originalSnapshot)}
	input := aiworkbench.ChatTurnInput{Content: "保持型号", Skill: ref, Revision: &revision}
	const conversation = "conversation"
	before := turnFingerprint(conversation, input)
	revision.Snapshot = json.RawMessage(storedSnapshot)
	raw, err := json.Marshal(aiworkbench.ChatTurnEnvelope{Version: 1, Input: input, Creator: ref})
	require.NoError(t, err)
	restored, err := decodeTurn(raw)
	require.NoError(t, err)
	require.Equal(t, before, turnFingerprint(conversation, restored.Input))
	require.Equal(t, originalSnapshot, string(restored.Input.Revision.Snapshot))
}
