package aiworkbench

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSelectedStoreKeepsBaseScopeAndRequiresCurrentAuthority(t *testing.T) {
	owner := Scope{OrganizationID: "org", ActorID: "actor"}
	base := BusinessScope{Version: BusinessScopeVersion, Kind: "general"}
	reader := ScopeReader{Store: func(_ context.Context, got Scope, id string) (StoreScopeFacts, error) {
		if got != owner || id != "store-b" {
			return StoreScopeFacts{}, ErrNotFound
		}
		return StoreScopeFacts{ID: id, Name: "店铺 B", Platform: "shein", Region: "US", Version: 1}, nil
	}, Connection: func(context.Context, Scope, string) (StoreConnectionFacts, error) {
		return StoreConnectionFacts{Version: 1, ObservedAt: "2026-11-14T00:00:00Z", Connected: true}, nil
	}}
	frozen, err := reader.Freeze(context.Background(), owner, "conversation", "member", base)
	require.NoError(t, err)
	original, err := json.Marshal(frozen)
	require.NoError(t, err)
	require.NotContains(t, string(original), "StoreChoice")
	legacy, err := json.Marshal(struct {
		Owner                         Scope
		ConversationID, MemberID      string
		Scope                         BusinessScope
		ProjectRevision, StoreVersion uint64
		ModelInput                    ScopeModelInput
	}{frozen.Owner, frozen.ConversationID, frozen.MemberID, frozen.Scope, frozen.ProjectRevision, frozen.StoreVersion, frozen.ModelInput})
	require.NoError(t, err)
	require.Equal(t, legacy, original)
	before := frozen.Digest()
	_, confirmation, err := reader.Preview(context.Background(), owner, ScopeSelection{Kind: "store", StoreID: "store-b"})
	require.NoError(t, err)
	frozen.StoreChoice, err = reader.FreezeStoreChoice(context.Background(), owner, &StoreChoiceInput{Kind: "store", StoreID: "store-b", ScopeConfirmation: confirmation, ConnectionVersion: 1})
	require.NoError(t, err)
	require.NoError(t, frozen.Validate())
	require.NotEqual(t, before, frozen.Digest())
	require.Equal(t, base, frozen.Scope)
	require.NoError(t, reader.Check(context.Background(), frozen, "member", false))
	reader.Connection = func(context.Context, Scope, string) (StoreConnectionFacts, error) { return StoreConnectionFacts{}, nil }
	require.ErrorIs(t, reader.Check(context.Background(), frozen, "member", false), ErrUnavailable)
	reader.Store = func(context.Context, Scope, string) (StoreScopeFacts, error) { return StoreScopeFacts{}, ErrNotFound }
	require.ErrorIs(t, reader.Check(context.Background(), frozen, "member", false), ErrNotFound)
}

func TestStoreChoiceNoneIsDistinctFromUnspecified(t *testing.T) {
	base := FrozenScope{Owner: Scope{OrganizationID: "org", ActorID: "actor"}, ConversationID: "conversation", MemberID: "member", Scope: BusinessScope{Version: BusinessScopeVersion, Kind: "general"}}
	require.NoError(t, base.Validate())
	none := base
	none.StoreChoice = &StoreChoiceScope{Kind: "none"}
	require.NoError(t, none.Validate())
	require.NotEqual(t, base.Digest(), none.Digest())
	none.StoreChoice = &StoreChoiceScope{Kind: "store"}
	require.ErrorIs(t, none.Validate(), ErrInvalid)
}

func TestStoreChoiceRequiresAnExactSelection(t *testing.T) {
	require.True(t, (*StoreChoiceInput)(nil).Valid())
	require.True(t, (&StoreChoiceInput{Kind: "none"}).Valid())
	require.False(t, (&StoreChoiceInput{Kind: "none", StoreID: "store-a"}).Valid())
	require.False(t, (&StoreChoiceInput{Kind: "store", StoreID: "store-a"}).Valid())
	require.True(t, (&StoreChoiceInput{Kind: "store", StoreID: "store-a", ScopeConfirmation: strings.Repeat("a", 64), ConnectionVersion: 1}).Valid())
}

func TestEffectiveModelInputProjectsCurrentStoreWithoutChangingBase(t *testing.T) {
	base := FrozenScope{ModelInput: ScopeModelInput{ProjectTitle: "项目", StoreName: "A", Platform: "shein", Region: "US"}}
	require.Equal(t, base.ModelInput, base.EffectiveModelInput())
	base.StoreChoice = &StoreChoiceScope{Kind: "store", Store: &SelectedStoreScope{Name: "B", Platform: "temu", Region: "UK"}}
	require.Equal(t, ScopeModelInput{ProjectTitle: "项目", StoreName: "B", Platform: "temu", Region: "UK"}, base.EffectiveModelInput())
	require.Equal(t, "A", base.ModelInput.StoreName)
	base.StoreChoice = &StoreChoiceScope{Kind: "none"}
	require.Equal(t, ScopeModelInput{ProjectTitle: "项目"}, base.EffectiveModelInput())
}
