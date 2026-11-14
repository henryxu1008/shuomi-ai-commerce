package aiworkbenchpersistence

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
	"task-processor/internal/aiworkbench"
)

func TestStoreDependenciesBindOriginalCommandsAndRejectMissingOwners(t *testing.T) {
	owner := aiworkbench.Scope{OrganizationID: "org", ActorID: "actor"}
	frozen := aiworkbench.FrozenScope{Owner: owner, ConversationID: "conversation", MemberID: "member", Scope: aiworkbench.BusinessScope{Version: aiworkbench.BusinessScopeVersion, Kind: "general"}, StoreChoice: &aiworkbench.StoreChoiceScope{Kind: "store", Store: &aiworkbench.SelectedStoreScope{StoreID: "store-b", Name: "店铺 B", Platform: "shein", Region: "US", StoreVersion: 1, ConnectionVersion: 1, ObservedAt: "2026-11-14T00:00:00Z"}}}
	require.NoError(t, frozen.Validate())
	body, err := json.Marshal(frozen)
	require.NoError(t, err)
	digest := frozen.Digest()
	member := "member"
	id := "assistant-1"
	row := commandRow{Operation: messageOperation, AssistantMessageID: &id, FrozenScopeJSON: body, ScopeDigest: &digest, OrganizationID: "org", ActorID: "actor", ConversationID: "conversation", MemberID: &member, IdempotencyKey: "11111111-1111-4111-8111-111111111111"}
	messages := []messageRow{{ID: id}}
	refs, err := planningStoreDependencies(messages, []commandRow{row})
	require.NoError(t, err)
	require.Len(t, refs, 1)
	require.Equal(t, digest, refs[0].Digest)
	_, err = planningStoreDependencies(messages, nil)
	require.Error(t, err)
	_, err = planningStoreDependencies(messages, []commandRow{row, row})
	require.Error(t, err)
	wrong := "wrong"
	row.MemberID = &wrong
	_, err = planningStoreDependencies(messages, []commandRow{row})
	require.Error(t, err)
}
