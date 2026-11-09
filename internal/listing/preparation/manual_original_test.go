package preparation

import (
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"task-processor/internal/product/collection"
	"testing"
)

func TestManualOriginalExclusionIsBoundAndDoesNotMutateBaseService(t *testing.T) {
	scope := Scope{OrganizationID: "org", ActorID: "actor", MemberID: "member"}
	item, publication := uuid.NewString(), uuid.NewString()
	base := &Service{}
	configured, err := base.WithManualOriginalExclusion(scope, item, publication)
	require.NoError(t, err)
	require.Nil(t, base.manualOriginal)
	commit := TransferCommit{scope: scope, manualOriginal: configured.manualOriginal}
	require.ErrorIs(t, commit.CheckManualOriginal(collection.Item{ID: item}), ErrForbidden)
	require.ErrorIs(t, commit.CheckManualOriginal(collection.Item{ID: uuid.NewString(), Source: collection.Source{PublicationID: publication}}), ErrForbidden)
	require.NoError(t, commit.CheckManualOriginal(collection.Item{ID: uuid.NewString(), Source: collection.Source{PublicationID: uuid.NewString()}}))
	for _, other := range []Scope{{OrganizationID: "other-org", ActorID: scope.ActorID, MemberID: scope.MemberID}, {OrganizationID: scope.OrganizationID, ActorID: "other-actor", MemberID: scope.MemberID}, {OrganizationID: scope.OrganizationID, ActorID: scope.ActorID, MemberID: "other-member"}} {
		commit.scope = other
		require.NoError(t, commit.CheckManualOriginal(collection.Item{ID: item}))
	}
	for _, invalid := range []struct {
		scope             Scope
		item, publication string
	}{{Scope{}, item, publication}, {scope, "", publication}, {scope, item, "not-publication"}} {
		_, err = base.WithManualOriginalExclusion(invalid.scope, invalid.item, invalid.publication)
		require.Error(t, err)
	}
}
