package aiworkbench

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestIT1ScopeSelectionUsesOwnerFacts(t *testing.T) {
	ctx := context.Background()
	scope := Scope{OrganizationID: "org-a", ActorID: "actor-a"}
	project := ProjectScopeFacts{ID: "project-a", Title: "Project", Goal: "goal", StoreID: "store-a", Revision: 1}
	store := StoreScopeFacts{ID: "store-a", Name: "Shop", Platform: "shein", Region: "US", Version: 1}
	allowed := true
	reader := ScopeReader{
		Project: func(_ context.Context, got Scope, id string) (ProjectScopeFacts, error) {
			if got != scope || id != project.ID {
				return ProjectScopeFacts{}, ErrNotFound
			}
			return project, nil
		},
		Store: func(_ context.Context, got Scope, id string) (StoreScopeFacts, error) {
			if !allowed || got != scope || id != store.ID {
				return StoreScopeFacts{}, ErrNotFound
			}
			return store, nil
		},
	}
	selection := ScopeSelection{Kind: "project", ProjectID: project.ID}
	view, digest, err := reader.Preview(ctx, scope, selection)
	require.NoError(t, err)
	require.Equal(t, store.ID, view.Scope.StoreID)
	require.Equal(t, store.Name, view.StoreName)
	require.Len(t, digest, 64)
	_, _, err = reader.Preview(ctx, scope, ScopeSelection{Kind: "project", ProjectID: project.ID, StoreID: store.ID})
	require.ErrorIs(t, err, ErrInvalid)
	_, _, err = reader.Preview(ctx, Scope{OrganizationID: "org-b", ActorID: "actor-a"}, selection)
	require.ErrorIs(t, err, ErrNotFound)
	allowed = false
	_, _, err = reader.Preview(ctx, scope, selection)
	require.ErrorIs(t, err, ErrNotFound)
	reader.Store = nil
	_, _, err = reader.Preview(ctx, scope, selection)
	require.ErrorIs(t, err, ErrUnavailable)
}

func TestIT2FrozenInputsSeparateDriftFromDisclosedHistory(t *testing.T) {
	ctx := context.Background()
	scope := Scope{OrganizationID: "org", ActorID: "actor"}
	fixed := BusinessScope{Version: BusinessScopeVersion, Kind: "project", ProjectID: "project"}
	project := ProjectScopeFacts{ID: "project", Title: "Original", Goal: "Original goal", Revision: 1}
	reader := ScopeReader{Project: func(context.Context, Scope, string) (ProjectScopeFacts, error) { return project, nil }}
	frozen, err := reader.Freeze(ctx, scope, "conversation", "member", fixed)
	require.NoError(t, err)
	require.NoError(t, frozen.Validate())
	digest := frozen.Digest()
	project.Revision++
	require.NoError(t, reader.Check(ctx, frozen, "member", true))
	project.Goal = "new goal"
	require.ErrorIs(t, reader.Check(ctx, frozen, "member", true), ErrProposalStale)
	require.NoError(t, reader.Check(ctx, frozen, "member", false))
	require.Equal(t, digest, frozen.Digest())
	require.ErrorIs(t, reader.Check(ctx, frozen, "other-member", false), ErrNotFound)
	project.Goal = strings.Repeat("a", 4097)
	_, err = reader.Freeze(ctx, scope, "conversation", "member", fixed)
	require.ErrorIs(t, err, ErrInvalid)
}

func TestIT3SendRequiresCurrentAuthorizedInputAndPreservesOriginal(t *testing.T) {
	ctx := context.Background()
	scope := Scope{OrganizationID: "org", ActorID: "actor"}
	fixed := BusinessScope{Version: BusinessScopeVersion, Kind: "store", StoreID: "shop"}
	var denied bool
	reader := ScopeReader{Store: func(context.Context, Scope, string) (StoreScopeFacts, error) {
		if denied {
			return StoreScopeFacts{}, ErrNotFound
		}
		return StoreScopeFacts{ID: "shop", Name: "Shop", Platform: "shein", Version: 1}, nil
	}}
	frozen, err := reader.Freeze(ctx, scope, "conversation", "member", fixed)
	require.NoError(t, err)
	denied = true
	require.ErrorIs(t, reader.Check(ctx, frozen, "member", false), ErrNotFound)
	require.NotEmpty(t, frozen.Digest())
}

func TestIT4ArchivedProjectRemainsAReadAuthorizedFilter(t *testing.T) {
	ctx := context.Background()
	owner := Scope{OrganizationID: "org", ActorID: "actor"}
	reader := ScopeReader{
		Project: func(_ context.Context, got Scope, id string) (ProjectScopeFacts, error) {
			if got != owner || id != "project" {
				return ProjectScopeFacts{}, ErrNotFound
			}
			return ProjectScopeFacts{ID: id, Title: "Archived project", StoreID: "current-private-shop", Revision: 5, Archived: true}, nil
		},
	}
	selection := ScopeSelection{Kind: "project", ProjectID: "project"}
	view, token, err := reader.FilterOption(ctx, owner, selection)
	require.NoError(t, err)
	require.Equal(t, "project", view.Scope.ProjectID)
	require.Equal(t, "Archived project", view.ProjectName)
	require.Empty(t, view.Scope.StoreID)
	require.Len(t, token, 64)
	_, _, err = reader.Preview(ctx, owner, selection)
	require.ErrorIs(t, err, ErrProposalStale)
	_, _, err = reader.FilterOption(ctx, Scope{OrganizationID: "org", ActorID: "other"}, selection)
	require.ErrorIs(t, err, ErrNotFound)
}

func TestIT4ScopedCursorCannotCrossAccountOrQuery(t *testing.T) {
	scope := Scope{OrganizationID: "org", ActorID: "actor"}
	f := ScopeSelection{Kind: "store", StoreID: "shop"}
	first, last := "c7fa421f-f979-4773-ac0f-1dd998994120", "aa2c59cd-4ec8-41db-b552-a0c50ff5f264"
	cursor, err := EncodeTaskScopeCursor(scope, f, first, last)
	require.NoError(t, err)
	a, b, err := DecodeTaskScopeCursor(scope, f, cursor)
	require.NoError(t, err)
	require.Equal(t, first, a)
	require.Equal(t, last, b)
	for _, changed := range []ScopeSelection{{Kind: "store", StoreID: "other"}, {Kind: "general"}, {Kind: "unrecorded"}} {
		_, _, err = DecodeTaskScopeCursor(scope, changed, cursor)
		require.ErrorIs(t, err, ErrInvalid)
	}
	_, _, err = DecodeTaskScopeCursor(Scope{OrganizationID: "org", ActorID: "other"}, f, cursor)
	require.ErrorIs(t, err, ErrInvalid)
}

func TestIT5FrozenIdentityCannotBeSubstitutedForMissingOwner(t *testing.T) {
	scope := Scope{OrganizationID: "org", ActorID: "actor"}
	fixed := BusinessScope{Version: BusinessScopeVersion, Kind: "project", ProjectID: "project"}
	r := ScopeReader{Project: func(context.Context, Scope, string) (ProjectScopeFacts, error) {
		return ProjectScopeFacts{ID: "project", Title: "Original", Revision: 1}, nil
	}}
	frozen, err := r.Freeze(context.Background(), scope, "conversation", "member", fixed)
	require.NoError(t, err)
	require.ErrorIs(t, (ScopeReader{}).Check(context.Background(), frozen, "member", false), ErrUnavailable)
	r.Project = func(context.Context, Scope, string) (ProjectScopeFacts, error) {
		return ProjectScopeFacts{}, errors.New("owner down")
	}
	require.Error(t, r.Check(context.Background(), frozen, "member", false))
}
