//go:build integration

package aiworkbenchpersistence

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"task-processor/internal/aiworkbench"
)

func TestIT4ScopedPostgresPaginationFiltersBeforeLimitAndChecksAnchors(t *testing.T) {
	db := workbenchDB(t)
	store, err := New(db)
	require.NoError(t, err)
	ctx := context.Background()
	owner := aiworkbench.Scope{OrganizationID: "org-a", ActorID: "actor-a"}
	add := func(selection aiworkbench.ScopeSelection, fixed aiworkbench.BusinessScope, when time.Time) string {
		c, _, err := store.CreateScoped(ctx, owner, uuid.NewString(), aiworkbench.CreateInput{Selection: &selection}, &fixed, nil)
		require.NoError(t, err)
		key := uuid.NewString()
		plan, _, err := store.AppendUser(ctx, owner, c.ID, key, aiworkbench.MessageInput{Content: "Improve title", OperationID: "op-1", TargetPlatform: "shein"},
			func(history []aiworkbench.Message, invocation string) (aiworkbench.PreparedPlan, error) {
				p, err := testPlanPreparer(history, invocation)
				f := &aiworkbench.FrozenScope{Owner: owner, ConversationID: c.ID, MemberID: p.MemberID, Scope: fixed}
				if fixed.StoreID != "" {
					f.StoreVersion, f.ModelInput.StoreName = 1, "Fixture store"
				}
				if fixed.ProjectID != "" {
					f.ProjectRevision, f.ModelInput.ProjectTitle = 1, "Fixture project"
				}
				p.FrozenScope = f
				return p, err
			})
		require.NoError(t, err)
		ref := plan.ScopeRef()
		require.True(t, ref.Valid())
		p := &aiworkbench.ExecutionProposal{Kind: "product.title.optimize", WorkbenchContextRef: ref, GoalSummary: "Improve title",
			OperationID: "op-1", ProductKey: "product-1", CatalogVersion: "1", PublicationID: "publication-1", TargetPlatform: "shein",
			AgentID: "product.title.agent", AgentVersion: "v1.0.0", ObservedAgentRevision: "revision-1", ObservedModeRevision: "0",
			ObservedActivationEpoch: "epoch-1", ExecutionModelProfile: []byte(`{"profile_id":"execution-v1"}`)}
		completed, _, err := store.CompletePlan(ctx, owner, key, aiworkbench.PlanTerminal{AssistantText: "Review", Mode: aiworkbench.PlanReady, GoalSummary: p.GoalSummary, Proposal: p})
		require.NoError(t, err)
		saved, err := store.GetProposal(ctx, owner, completed.ProposalID)
		require.NoError(t, err)
		confirmation := uuid.NewString()
		item, _, err := store.Confirm(ctx, owner, c.ID, saved.ID, confirmation, preparedTaskFixture(t, saved, confirmation))
		require.NoError(t, err)
		require.NoError(t, db.Model(&taskRow{}).Where("id = ?", item.ID).Update("created_at", when).Error)
		return item.ID
	}
	when := time.Now().UTC().Truncate(time.Second)
	general := aiworkbench.BusinessScope{Version: aiworkbench.BusinessScopeVersion, Kind: "general"}
	shop := aiworkbench.BusinessScope{Version: aiworkbench.BusinessScopeVersion, Kind: "store", StoreID: "shop"}
	project := aiworkbench.BusinessScope{Version: aiworkbench.BusinessScopeVersion, Kind: "project", ProjectID: "project", StoreID: "shop"}
	filter := aiworkbench.ScopeSelection{Kind: "store", StoreID: "shop"}
	firstID := add(filter, shop, when)
	secondID := add(aiworkbench.ScopeSelection{Kind: "project", ProjectID: "project"}, project, when)
	add(aiworkbench.ScopeSelection{Kind: "general"}, general, when.Add(time.Second))
	if firstID < secondID {
		firstID, secondID = secondID, firstID
	}
	visible := func(*aiworkbench.BusinessScope, aiworkbench.BusinessTask) bool { return true }
	first, err := store.ListScopedTasks(ctx, owner, filter, "", 1, visible)
	require.NoError(t, err)
	require.Len(t, first.Tasks, 1)
	require.Equal(t, firstID, first.Tasks[0].ID)
	require.NotEmpty(t, first.Next)
	newID := add(filter, shop, when.Add(2*time.Second))
	second, err := store.ListScopedTasks(ctx, owner, filter, first.Next, 1, visible)
	require.NoError(t, err)
	require.Len(t, second.Tasks, 1)
	require.Equal(t, secondID, second.Tasks[0].ID)
	require.Empty(t, second.Next)
	refreshed, err := store.ListScopedTasks(ctx, owner, filter, "", 1, visible)
	require.NoError(t, err)
	require.Equal(t, newID, refreshed.Tasks[0].ID)
	_, err = store.ListScopedTasks(ctx, owner, aiworkbench.ScopeSelection{Kind: "general"}, first.Next, 1, visible)
	require.ErrorIs(t, err, aiworkbench.ErrInvalid)
	hidden := func(_ *aiworkbench.BusinessScope, task aiworkbench.BusinessTask) bool { return task.ID != firstID }
	_, err = store.ListScopedTasks(ctx, owner, filter, first.Next, 1, hidden)
	require.ErrorIs(t, err, aiworkbench.ErrInvalid)
	none := func(*aiworkbench.BusinessScope, aiworkbench.BusinessTask) bool { return false }
	page, err := store.ListScopedTasks(ctx, owner, filter, "", 1, none)
	require.NoError(t, err)
	require.Empty(t, page.Tasks)
	require.Empty(t, page.Next)
	require.True(t, page.Incomplete)
}
