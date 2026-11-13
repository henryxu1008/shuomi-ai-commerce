//go:build integration

package aiworkbenchpersistence

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"task-processor/internal/aicapability"
	aistore "task-processor/internal/aicapability/store"
	"task-processor/internal/aiworkbench"
	"task-processor/internal/integration/aiworkbench/einorouter"
)

func TestStoreRouteSameKeyConcurrentAndTypedHistory(t *testing.T) {
	db := workbenchDB(t)
	store, err := New(db)
	require.NoError(t, err)
	ctx := context.Background()
	scope := aiworkbench.Scope{OrganizationID: "org-a", ActorID: "user-a"}
	fixed := &aiworkbench.BusinessScope{Version: aiworkbench.BusinessScopeVersion, Kind: "general"}
	conversation, _, err := store.CreateScoped(ctx, scope, uuid.NewString(), aiworkbench.CreateInput{MemberID: "member-a"}, fixed, nil)
	require.NoError(t, err)
	key := uuid.NewString()
	request := aiworkbench.StoreRouteRequest{Content: "我有几个店铺", MemberID: "member-a", InputHash: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", ModelProfile: []byte(`{"version":"fixture"}`), ModelTimeout: time.Minute}
	wrongMember := request
	wrongMember.MemberID = "member-rejoined"
	_, _, err = store.BeginStoreRoute(ctx, scope, conversation.ID, uuid.NewString(), wrongMember)
	require.ErrorIs(t, err, aiworkbench.ErrNotFound)
	var wg sync.WaitGroup
	errs := make(chan error, 2)
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, _, e := store.BeginStoreRoute(ctx, scope, conversation.ID, key, request)
			errs <- e
		}()
	}
	wg.Wait()
	close(errs)
	for e := range errs {
		require.NoError(t, e)
	}
	receipt, found, err := store.ReplayStoreRoute(ctx, scope, conversation.ID, key, "member-a", request.Content)
	require.NoError(t, err)
	require.True(t, found)
	require.Equal(t, request.Content, receipt.Content)
	require.JSONEq(t, string(request.ModelProfile), string(receipt.ModelProfile))
	_, _, err = store.ReplayStoreRoute(ctx, scope, conversation.ID, key, "member-rejoined", request.Content)
	require.ErrorIs(t, err, aiworkbench.ErrNotFound)
	_, _, err = store.ReplayStoreRoute(ctx, scope, conversation.ID, key, "member-a", "不同的问题")
	require.ErrorIs(t, err, aiworkbench.ErrIdempotencyConflict)
	changed := request
	changed.Content = "不同的问题"
	_, _, err = store.BeginStoreRoute(ctx, scope, conversation.ID, key, changed)
	require.ErrorIs(t, err, aiworkbench.ErrIdempotencyConflict)
	route := aiworkbench.StoreRoute{Kind: "STORE_OVERVIEW", Status: "ALL_NON_DELETED"}
	finished := make(chan error, 2)
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() { defer wg.Done(); _, _, e := store.CompleteStoreRoute(ctx, scope, key, route); finished <- e }()
	}
	wg.Wait()
	close(finished)
	for e := range finished {
		require.NoError(t, e)
	}
	_, replay, err := store.CompleteStoreRoute(ctx, scope, key, route)
	require.NoError(t, err)
	require.True(t, replay)
	messages, _, err := store.ListMessagesPage(ctx, scope, conversation.ID, 0, 50)
	require.NoError(t, err)
	require.Len(t, messages, 2)
	require.Equal(t, "STORE_OVERVIEW_REF", messages[1].Kind)
	require.Equal(t, "ALL_NON_DELETED", messages[1].StoreQueryStatus)
	require.NotContains(t, messages[1].Content, "家")
}

func TestStoreRouteUnavailableAdmissionOwnsKeyAndReplays(t *testing.T) {
	db := workbenchDB(t)
	store, err := New(db)
	require.NoError(t, err)
	ctx := context.Background()
	scope := aiworkbench.Scope{OrganizationID: "org-a", ActorID: "user-a"}
	fixed := &aiworkbench.BusinessScope{Version: aiworkbench.BusinessScopeVersion, Kind: "general"}
	c, _, err := store.CreateScoped(ctx, scope, uuid.NewString(), aiworkbench.CreateInput{MemberID: "member-a"}, fixed, nil)
	require.NoError(t, err)
	key := uuid.NewString()
	unavailable := aiworkbench.StoreRouteRequest{Content: "问题", MemberID: "member-a", Unavailable: true}
	first, replay, err := store.BeginStoreRoute(ctx, scope, c.ID, key, unavailable)
	require.NoError(t, err)
	require.False(t, replay)
	require.Equal(t, string(aiworkbench.PlanningFailedBeforeDispatch), first.State)
	ready := aiworkbench.StoreRouteRequest{Content: "问题", MemberID: "member-a", InputHash: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", ModelProfile: []byte(`{"version":"fixture"}`), ModelTimeout: time.Minute}
	second, replay, err := store.BeginStoreRoute(ctx, scope, c.ID, key, ready)
	require.NoError(t, err)
	require.True(t, replay)
	require.Equal(t, first.InvocationID, second.InvocationID)
	require.Equal(t, string(aiworkbench.PlanningFailedBeforeDispatch), second.State)
	ready.Content = "另一个问题"
	_, _, err = store.BeginStoreRoute(ctx, scope, c.ID, key, ready)
	require.ErrorIs(t, err, aiworkbench.ErrIdempotencyConflict)
	messages, _, err := store.ListMessagesPage(ctx, scope, c.ID, 0, 50)
	require.NoError(t, err)
	require.Len(t, messages, 2)
	require.Contains(t, messages[1].Content, "尚未发送")
}

func TestStoreRouteAvailableAdmissionWinsAgainstUnavailable(t *testing.T) {
	db := workbenchDB(t)
	store, err := New(db)
	require.NoError(t, err)
	ctx := context.Background()
	scope := aiworkbench.Scope{OrganizationID: "org-a", ActorID: "user-a"}
	fixed := &aiworkbench.BusinessScope{Version: aiworkbench.BusinessScopeVersion, Kind: "general"}
	c, _, err := store.CreateScoped(ctx, scope, uuid.NewString(), aiworkbench.CreateInput{MemberID: "member-a"}, fixed, nil)
	require.NoError(t, err)
	key := uuid.NewString()
	ready := aiworkbench.StoreRouteRequest{Content: "问题", MemberID: "member-a", InputHash: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", ModelProfile: []byte(`{"version":"fixture"}`), ModelTimeout: time.Minute}
	original, _, err := store.BeginStoreRoute(ctx, scope, c.ID, key, ready)
	require.NoError(t, err)
	got, replay, err := store.BeginStoreRoute(ctx, scope, c.ID, key, aiworkbench.StoreRouteRequest{Content: "问题", MemberID: "member-a", Unavailable: true})
	require.NoError(t, err)
	require.True(t, replay)
	require.Equal(t, original.InvocationID, got.InvocationID)
	require.Equal(t, string(aiworkbench.PlanningReadyToDispatch), got.State)
	messages, _, err := store.ListMessagesPage(ctx, scope, c.ID, 0, 50)
	require.NoError(t, err)
	require.Len(t, messages, 1)
}

func TestStoreRoutePreparesWithOriginalInvocationOnlyOnce(t *testing.T) {
	db := workbenchDB(t)
	store, err := New(db)
	require.NoError(t, err)
	ctx := context.Background()
	scope := aiworkbench.Scope{OrganizationID: "org-a", ActorID: "user-a"}
	fixed := &aiworkbench.BusinessScope{Version: aiworkbench.BusinessScopeVersion, Kind: "general"}
	conversation, _, err := store.CreateScoped(ctx, scope, uuid.NewString(), aiworkbench.CreateInput{MemberID: "member-a"}, fixed, nil)
	require.NoError(t, err)
	calls := 0
	var invocation string
	request := aiworkbench.StoreRouteRequest{Content: "我的店铺", MemberID: "member-a", Prepare: func(id string) (aiworkbench.StoreRoutePrepared, error) {
		calls++
		invocation = id
		return aiworkbench.StoreRoutePrepared{InputHash: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", ModelProfile: []byte(`{"version":"fixture"}`), ModelTimeout: time.Minute}, nil
	}}
	key := uuid.NewString()
	created, replay, err := store.BeginStoreRoute(ctx, scope, conversation.ID, key, request)
	require.NoError(t, err)
	require.False(t, replay)
	require.NotEmpty(t, invocation)
	require.Equal(t, invocation, created.InvocationID)
	_, replay, err = store.BeginStoreRoute(ctx, scope, conversation.ID, key, request)
	require.NoError(t, err)
	require.True(t, replay)
	require.Equal(t, 1, calls)
}

func TestStoreRouteFrozenQuoteReconstructsWithOriginalInvocation(t *testing.T) {
	db := workbenchDB(t)
	store, err := New(db)
	require.NoError(t, err)
	ctx := context.Background()
	scope := aiworkbench.Scope{OrganizationID: "org-a", ActorID: "user-a"}
	fixed := &aiworkbench.BusinessScope{Version: aiworkbench.BusinessScopeVersion, Kind: "general"}
	conversation, _, err := store.CreateScoped(ctx, scope, uuid.NewString(), aiworkbench.CreateInput{MemberID: "member-a"}, fixed, nil)
	require.NoError(t, err)
	profile := aicapability.ModelProfile{ClientName: "fixture", ProviderID: "fixture", AdapterKind: "openai-compatible", ModelID: "model-a", EndpointIdentityDigest: "endpoint", CredentialVersion: "credential", RoutingPolicyVersion: "route", AdapterPolicyVersion: "adapter", PromptVersion: einorouter.PromptVersion, OutputSchemaVersion: einorouter.SchemaVersion, UsageMappingVersion: "usage", CostPricingVersion: "price", Currency: "CNY", InputMicrosPerMillion: 1, OutputMicrosPerMillion: 1, MaximumPromptTokens: 10000, MaximumCompletionTokens: 500, MaximumInputBytes: 8192, MaximumOutputBytes: 4096, DeadlineBound: time.Minute, OfficialTariff: aicapability.OfficialModelTariff{UnitVersion: aicapability.AIPointMicrounitV1, PriceVersion: "price", FXVersion: "fx", Currency: "CNY", CNYMicrosPerCurrencyUnit: 1000000, Model: "model-a", Channel: "fixture", UsageVersion: "openai-text-v1", Rates: aicapability.OfficialModelRates{InputText: 1, OutputText: 1}}}
	key := uuid.NewString()
	content := "我有几个店铺"
	request := aiworkbench.StoreRouteRequest{Content: content, MemberID: "member-a", Prepare: func(invocation string) (aiworkbench.StoreRoutePrepared, error) {
		prepared, e := einorouter.Prepare(scope, "member-a", invocation, content, profile)
		if e != nil {
			return aiworkbench.StoreRoutePrepared{}, e
		}
		raw, e := json.Marshal(profile)
		return aiworkbench.StoreRoutePrepared{InputHash: prepared.Quote.InputHash, ModelProfile: raw, ModelTimeout: profile.DeadlineBound}, e
	}}
	original, _, err := store.BeginStoreRoute(ctx, scope, conversation.ID, key, request)
	require.NoError(t, err)
	stored, found, err := store.ReplayStoreRoute(ctx, scope, conversation.ID, key, "member-a", content)
	require.NoError(t, err)
	require.True(t, found)
	require.Equal(t, original.InvocationID, stored.InvocationID)
	var frozen aicapability.ModelProfile
	require.NoError(t, json.Unmarshal(stored.ModelProfile, &frozen))
	prepared, err := einorouter.Prepare(scope, stored.MemberID, stored.InvocationID, stored.Content, frozen)
	require.NoError(t, err)
	require.Equal(t, stored.InputHash, prepared.Quote.InputHash)
}

func TestExplicitStoreOverviewUsesNoInvocationAndIsIdempotent(t *testing.T) {
	db := workbenchDB(t)
	store, err := New(db)
	require.NoError(t, err)
	ctx := context.Background()
	scope := aiworkbench.Scope{OrganizationID: "org-a", ActorID: "user-a"}
	fixed := &aiworkbench.BusinessScope{Version: aiworkbench.BusinessScopeVersion, Kind: "general"}
	conversation, _, err := store.CreateScoped(ctx, scope, uuid.NewString(), aiworkbench.CreateInput{MemberID: "member-a"}, fixed, nil)
	require.NoError(t, err)
	key := uuid.NewString()
	first, replay, err := store.ExplicitStoreOverview(ctx, scope, conversation.ID, key, "member-a", "ALL_NON_DELETED")
	require.NoError(t, err)
	require.False(t, replay)
	second, replay, err := store.ExplicitStoreOverview(ctx, scope, conversation.ID, key, "member-a", "ALL_NON_DELETED")
	require.NoError(t, err)
	require.True(t, replay)
	require.Equal(t, first.InvocationID, second.InvocationID)
	require.Empty(t, first.InvocationID)
	_, _, err = store.ExplicitStoreOverview(ctx, scope, conversation.ID, key, "member-a", "ACTIVE")
	require.ErrorIs(t, err, aiworkbench.ErrIdempotencyConflict)
	messages, _, err := store.ListMessagesPage(ctx, scope, conversation.ID, 0, 50)
	require.NoError(t, err)
	require.Len(t, messages, 2)
	require.Equal(t, "STORE_OVERVIEW_REF", messages[1].Kind)
}

func TestStoreRouteFailureProofAndReplay(t *testing.T) {
	db := workbenchDB(t)
	store, err := New(db)
	require.NoError(t, err)
	ctx := context.Background()
	scope := aiworkbench.Scope{OrganizationID: "org-a", ActorID: "user-a"}
	fixed := &aiworkbench.BusinessScope{Version: aiworkbench.BusinessScopeVersion, Kind: "general"}
	conversation, _, err := store.CreateScoped(ctx, scope, uuid.NewString(), aiworkbench.CreateInput{MemberID: "member-a"}, fixed, nil)
	require.NoError(t, err)
	key := uuid.NewString()
	request := aiworkbench.StoreRouteRequest{Content: "问题", MemberID: "member-a", InputHash: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", ModelProfile: []byte(`{"version":"fixture"}`), ModelTimeout: time.Minute}
	command, _, err := store.BeginStoreRoute(ctx, scope, conversation.ID, key, request)
	require.NoError(t, err)
	_, _, err = store.FinalizeStoreRouteFailure(ctx, scope, key, func(context.Context, aiworkbench.StoreRouteCommand) (aiworkbench.PlanningState, error) {
		return "", errors.New("unknown")
	})
	require.Error(t, err)
	proof := func(_ context.Context, got aiworkbench.StoreRouteCommand) (aiworkbench.PlanningState, error) {
		if got.InvocationID != command.InvocationID {
			return "", errors.New("wrong invocation")
		}
		return aiworkbench.PlanningInvalidOutput, nil
	}
	var wg sync.WaitGroup
	results := make(chan error, 2)
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, _, e := store.FinalizeStoreRouteFailure(ctx, scope, key, proof)
			results <- e
		}()
	}
	wg.Wait()
	close(results)
	for e := range results {
		require.NoError(t, e)
	}
	messages, _, err := store.ListMessagesPage(ctx, scope, conversation.ID, 0, 50)
	require.NoError(t, err)
	require.Len(t, messages, 2)
	require.Contains(t, messages[1].Content, "格式无效")
	terminal, replay, err := store.FinalizeStoreRouteFailure(ctx, scope, key, proof)
	require.NoError(t, err)
	require.True(t, replay)
	require.Equal(t, string(aiworkbench.PlanningInvalidOutput), terminal.State)
}

func TestStoreRouteLostRequiresOriginalSettlementAndOnlyOneTerminal(t *testing.T) {
	db := workbenchDB(t)
	store, err := New(db)
	require.NoError(t, err)
	ctx := context.Background()
	scope := aiworkbench.Scope{OrganizationID: "org-a", ActorID: "user-a"}
	fixed := &aiworkbench.BusinessScope{Version: aiworkbench.BusinessScopeVersion, Kind: "general"}
	conversation, _, err := store.CreateScoped(ctx, scope, uuid.NewString(), aiworkbench.CreateInput{MemberID: "member-a"}, fixed, nil)
	require.NoError(t, err)
	key := uuid.NewString()
	request := aiworkbench.StoreRouteRequest{Content: "店铺数量", MemberID: "member-a", InputHash: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", ModelProfile: []byte(`{"version":"fixture"}`), ModelTimeout: time.Minute}
	command, _, err := store.BeginStoreRoute(ctx, scope, conversation.ID, key, request)
	require.NoError(t, err)
	_, _, err = store.MarkStoreRouteLost(ctx, scope, key, func(context.Context, aiworkbench.StoreRouteCommand) error { return nil })
	require.ErrorIs(t, err, aiworkbench.ErrUnavailable, "a settled receipt cannot preempt the still-valid original handler")
	require.NoError(t, db.Exec("UPDATE ai_workbench.commands SET route_commit_deadline=clock_timestamp()-INTERVAL '1 second' WHERE idempotency_key=?", key).Error)
	_, _, err = store.MarkStoreRouteLost(ctx, scope, key, func(context.Context, aiworkbench.StoreRouteCommand) error { return errors.New("settlement unknown") })
	require.Error(t, err)
	proof := func(_ context.Context, got aiworkbench.StoreRouteCommand) error {
		if got.InvocationID != command.InvocationID || got.InputHash != request.InputHash || got.MemberID != request.MemberID {
			return errors.New("mismatched receipt")
		}
		return nil
	}
	var wg sync.WaitGroup
	results := make(chan error, 3)
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() { defer wg.Done(); _, _, e := store.MarkStoreRouteLost(ctx, scope, key, proof); results <- e }()
	}
	wg.Add(1)
	go func() {
		defer wg.Done()
		_, _, e := store.CompleteStoreRoute(ctx, scope, key, aiworkbench.StoreRoute{Kind: "STORE_OVERVIEW", Status: "ACTIVE"})
		if e != nil && !errors.Is(e, aiworkbench.ErrUnavailable) {
			results <- e
			return
		}
		results <- nil
	}()
	wg.Wait()
	close(results)
	for e := range results {
		require.NoError(t, e)
	}
	messages, _, err := store.ListMessagesPage(ctx, scope, conversation.ID, 0, 50)
	require.NoError(t, err)
	require.Len(t, messages, 2)
	require.Contains(t, messages[1].Content, "未在期限内保存")
	terminal, replay, err := store.MarkStoreRouteLost(ctx, scope, key, proof)
	require.NoError(t, err)
	require.True(t, replay)
	require.Equal(t, "ROUTING_RESULT_LOST", terminal.State)
}

func TestStoreRouteExpiredCommitDoesNotAppend(t *testing.T) {
	db := workbenchDB(t)
	store, err := New(db)
	require.NoError(t, err)
	ctx := context.Background()
	scope := aiworkbench.Scope{OrganizationID: "org-a", ActorID: "user-a"}
	fixed := &aiworkbench.BusinessScope{Version: aiworkbench.BusinessScopeVersion, Kind: "general"}
	conversation, _, err := store.CreateScoped(ctx, scope, uuid.NewString(), aiworkbench.CreateInput{MemberID: "member-a"}, fixed, nil)
	require.NoError(t, err)
	key := uuid.NewString()
	request := aiworkbench.StoreRouteRequest{Content: "店铺数量", MemberID: "member-a", InputHash: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", ModelProfile: []byte(`{"version":"fixture"}`), ModelTimeout: time.Minute}
	_, _, err = store.BeginStoreRoute(ctx, scope, conversation.ID, key, request)
	require.NoError(t, err)
	require.NoError(t, db.Exec("UPDATE ai_workbench.commands SET route_commit_deadline = clock_timestamp() - INTERVAL '1 second' WHERE idempotency_key = ?", key).Error)
	_, _, err = store.CompleteStoreRoute(ctx, scope, key, aiworkbench.StoreRoute{Kind: "STORE_OVERVIEW", Status: "ACTIVE"})
	require.ErrorIs(t, err, aiworkbench.ErrUnavailable)
	messages, _, err := store.ListMessagesPage(ctx, scope, conversation.ID, 0, 50)
	require.NoError(t, err)
	require.Len(t, messages, 1)
}

func TestStoreRouteAbsentClaimRecoveryFencesLateDispatch(t *testing.T) {
	db := workbenchDB(t)
	store, err := New(db)
	require.NoError(t, err)
	ctx := context.Background()
	scope := aiworkbench.Scope{OrganizationID: "org-a", ActorID: "user-a"}
	fixed := &aiworkbench.BusinessScope{Version: aiworkbench.BusinessScopeVersion, Kind: "general"}
	c, _, err := store.CreateScoped(ctx, scope, uuid.NewString(), aiworkbench.CreateInput{MemberID: "member-a"}, fixed, nil)
	require.NoError(t, err)
	key := uuid.NewString()
	req := aiworkbench.StoreRouteRequest{Content: "店铺", MemberID: "member-a", InputHash: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", ModelProfile: []byte(`{"version":"fixture"}`), ModelTimeout: time.Minute}
	command, _, err := store.BeginStoreRoute(ctx, scope, c.ID, key, req)
	require.NoError(t, err)
	_, _, err = store.FinalizeStoreRouteAbsent(ctx, scope, key, func(context.Context, string) (bool, error) { return false, nil })
	require.Error(t, err)
	require.NoError(t, db.Exec("UPDATE ai_workbench.commands SET planner_deadline=clock_timestamp()-INTERVAL '1 second' WHERE idempotency_key=?", key).Error)
	_, _, err = store.FinalizeStoreRouteAbsent(ctx, scope, key, func(context.Context, string) (bool, error) { return false, errors.New("primary unavailable") })
	require.Error(t, err)
	_, _, err = store.FinalizeStoreRouteAbsent(ctx, scope, key, func(context.Context, string) (bool, error) { return true, nil })
	require.Error(t, err)
	result, replay, err := store.FinalizeStoreRouteAbsent(ctx, scope, key, func(_ context.Context, id string) (bool, error) {
		require.Equal(t, command.InvocationID, id)
		return false, nil
	})
	require.NoError(t, err)
	require.False(t, replay)
	require.Equal(t, string(aiworkbench.PlanningFailedBeforeDispatch), result.State)
	require.Error(t, store.VerifyStoreRouteDispatch(ctx, command))
	_, replay, err = store.FinalizeStoreRouteAbsent(ctx, scope, key, func(context.Context, string) (bool, error) {
		t.Fatal("terminal must not recheck absence")
		return false, nil
	})
	require.NoError(t, err)
	require.True(t, replay)
}

func TestStoreRouteUncommittedClaimCannotReserveAfterAbsentRecovery(t *testing.T) {
	db := workbenchDB(t)
	store, err := New(db)
	require.NoError(t, err)
	ctx := context.Background()
	scope := aiworkbench.Scope{OrganizationID: "org-a", ActorID: "user-a"}
	fixed := &aiworkbench.BusinessScope{Version: aiworkbench.BusinessScopeVersion, Kind: "general"}
	c, _, err := store.CreateScoped(ctx, scope, uuid.NewString(), aiworkbench.CreateInput{MemberID: "member-a"}, fixed, nil)
	require.NoError(t, err)
	key := uuid.NewString()
	req := aiworkbench.StoreRouteRequest{Content: "店铺", MemberID: "member-a", InputHash: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", ModelProfile: []byte(`{"version":"fixture"}`), ModelTimeout: time.Minute}
	command, _, err := store.BeginStoreRoute(ctx, scope, c.ID, key, req)
	require.NoError(t, err)
	require.NoError(t, db.Exec("CREATE TABLE synthetic_ai_claims (id text PRIMARY KEY)").Error)
	claim := db.Begin()
	require.NoError(t, claim.Exec("INSERT INTO synthetic_ai_claims (id) VALUES (?)", command.InvocationID).Error)
	require.NoError(t, db.Exec("UPDATE ai_workbench.commands SET planner_deadline=clock_timestamp()-INTERVAL '1 second' WHERE idempotency_key=?", key).Error)
	_, _, err = store.FinalizeStoreRouteAbsent(ctx, scope, key, func(ctx context.Context, id string) (bool, error) {
		var n int64
		e := db.WithContext(ctx).Table("synthetic_ai_claims").Where("id=?", id).Count(&n).Error
		return n > 0, e
	})
	require.NoError(t, err)
	require.NoError(t, claim.Commit().Error)
	require.Error(t, store.VerifyStoreRouteDispatch(ctx, command), "a claim committed after recovery must fail the pre-reservation gate")
}

func TestInvocationAbsenceUsesPrimaryAndRejectsRLS(t *testing.T) {
	db := workbenchDB(t)
	require.NoError(t, aistore.AutoMigrateInvocationLedger(db))
	recorder := aistore.NewGormInvocationRecorder(db)
	ctx := context.Background()
	fact := aicapability.InvocationRecord{InvocationID: "global-fixture", TenantID: "another-org", UserID: "actor", MemberID: "member", InputHash: "input", Operation: aicapability.OperationAIWorkbenchStoreRoute, Capability: aicapability.CapabilityAIWorkbenchChatPlanning, Outcome: aicapability.InvocationDispatched, StartedAt: time.Now().UTC(), MaximumPromptTokens: 100, MaximumCompletionTokens: 50, OfficialTariff: aicapability.OfficialModelTariff{UnitVersion: aicapability.AIPointMicrounitV1, PriceVersion: "synthetic", FXVersion: "fixture-fx-identity", Currency: "CNY", CNYMicrosPerCurrencyUnit: 1000000, Model: "fixture-model", Channel: "fixture-channel", UsageVersion: "openai-text-v1", Rates: aicapability.OfficialModelRates{InputText: 1, OutputText: 1}}, ModelID: "fixture-model", Currency: "CNY"}
	_, err := recorder.ClaimInvocation(ctx, fact)
	require.NoError(t, err)
	found, err := recorder.HasInvocationID(ctx, "global-fixture")
	require.NoError(t, err)
	require.True(t, found)
	found, err = recorder.HasInvocationID(ctx, "missing-fixture")
	require.NoError(t, err)
	require.False(t, found)
	require.NoError(t, db.Exec("ALTER TABLE public.ai_invocations ENABLE ROW LEVEL SECURITY").Error)
	_, err = recorder.HasInvocationID(ctx, "missing-fixture")
	require.Error(t, err, "a hidden conflicting claim cannot be interpreted as absent")
}

func TestAbsentRecoveryPoolSaturationFailsClosedAndUnlocks(t *testing.T) {
	db := workbenchDB(t)
	require.NoError(t, aistore.AutoMigrateInvocationLedger(db))
	store, err := New(db)
	require.NoError(t, err)
	recorder := aistore.NewGormInvocationRecorder(db)
	ctx := context.Background()
	scope := aiworkbench.Scope{OrganizationID: "org-a", ActorID: "user-a"}
	fixed := &aiworkbench.BusinessScope{Version: aiworkbench.BusinessScopeVersion, Kind: "general"}
	c, _, err := store.CreateScoped(ctx, scope, uuid.NewString(), aiworkbench.CreateInput{MemberID: "member-a"}, fixed, nil)
	require.NoError(t, err)
	key := uuid.NewString()
	req := aiworkbench.StoreRouteRequest{Content: "店铺", MemberID: "member-a", InputHash: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", ModelProfile: []byte(`{"version":"fixture"}`), ModelTimeout: time.Minute}
	_, _, err = store.BeginStoreRoute(ctx, scope, c.ID, key, req)
	require.NoError(t, err)
	require.NoError(t, db.Exec("UPDATE ai_workbench.commands SET planner_deadline=clock_timestamp()-INTERVAL '1 second' WHERE idempotency_key=?", key).Error)
	pool, err := db.DB()
	require.NoError(t, err)
	pool.SetMaxOpenConns(1)
	defer pool.SetMaxOpenConns(0)
	_, _, err = store.FinalizeStoreRouteAbsent(ctx, scope, key, recorder.HasInvocationID)
	require.Error(t, err)
	var state string
	require.NoError(t, db.Raw("SELECT state FROM ai_workbench.commands WHERE idempotency_key=?", key).Scan(&state).Error)
	require.Equal(t, string(aiworkbench.PlanningReadyToDispatch), state)
}

func TestStoreRouteDispatchGateUsesOriginalDeadlineAndState(t *testing.T) {
	db := workbenchDB(t)
	store, err := New(db)
	require.NoError(t, err)
	ctx := context.Background()
	scope := aiworkbench.Scope{OrganizationID: "org-a", ActorID: "user-a"}
	fixed := &aiworkbench.BusinessScope{Version: aiworkbench.BusinessScopeVersion, Kind: "general"}
	c, _, err := store.CreateScoped(ctx, scope, uuid.NewString(), aiworkbench.CreateInput{MemberID: "member-a"}, fixed, nil)
	require.NoError(t, err)
	key := uuid.NewString()
	request := aiworkbench.StoreRouteRequest{Content: "店铺", MemberID: "member-a", InputHash: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", ModelProfile: []byte(`{"version":"fixture"}`), ModelTimeout: time.Minute}
	command, _, err := store.BeginStoreRoute(ctx, scope, c.ID, key, request)
	require.NoError(t, err)
	require.NoError(t, store.VerifyStoreRouteDispatch(ctx, command))
	wrong := command
	wrong.InputHash = "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
	require.Error(t, store.VerifyStoreRouteDispatch(ctx, wrong))
	require.NoError(t, db.Exec("UPDATE ai_workbench.commands SET planner_deadline=clock_timestamp()-INTERVAL '1 second' WHERE idempotency_key=?", key).Error)
	require.Error(t, store.VerifyStoreRouteDispatch(ctx, command))
}

func TestStoreRouteLockWaitAcrossDeadlineRejectsLateCommit(t *testing.T) {
	db := workbenchDB(t)
	store, err := New(db)
	require.NoError(t, err)
	ctx := context.Background()
	scope := aiworkbench.Scope{OrganizationID: "org-a", ActorID: "user-a"}
	fixed := &aiworkbench.BusinessScope{Version: aiworkbench.BusinessScopeVersion, Kind: "general"}
	conversation, _, err := store.CreateScoped(ctx, scope, uuid.NewString(), aiworkbench.CreateInput{MemberID: "member-a"}, fixed, nil)
	require.NoError(t, err)
	key := uuid.NewString()
	request := aiworkbench.StoreRouteRequest{Content: "店铺数量", MemberID: "member-a", InputHash: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", ModelProfile: []byte(`{"version":"fixture"}`), ModelTimeout: time.Minute}
	_, _, err = store.BeginStoreRoute(ctx, scope, conversation.ID, key, request)
	require.NoError(t, err)
	lock := db.Begin()
	require.NoError(t, lock.Error)
	require.NoError(t, lock.Exec("UPDATE ai_workbench.commands SET route_commit_deadline=clock_timestamp()+INTERVAL '1 second' WHERE idempotency_key=?", key).Error)
	started := make(chan struct{})
	result := make(chan error, 1)
	go func() {
		close(started)
		_, _, e := store.CompleteStoreRoute(ctx, scope, key, aiworkbench.StoreRoute{Kind: "STORE_OVERVIEW", Status: "ACTIVE"})
		result <- e
	}()
	<-started
	time.Sleep(1500 * time.Millisecond)
	require.NoError(t, lock.Commit().Error)
	require.ErrorIs(t, <-result, aiworkbench.ErrUnavailable)
	messages, _, err := store.ListMessagesPage(ctx, scope, conversation.ID, 0, 50)
	require.NoError(t, err)
	require.Len(t, messages, 1)
}

func TestStoreRouteConversationLockWaitAcrossDeadlineRejectsLateCommit(t *testing.T) {
	db := workbenchDB(t)
	store, err := New(db)
	require.NoError(t, err)
	ctx := context.Background()
	scope := aiworkbench.Scope{OrganizationID: "org-a", ActorID: "user-a"}
	fixed := &aiworkbench.BusinessScope{Version: aiworkbench.BusinessScopeVersion, Kind: "general"}
	conversation, _, err := store.CreateScoped(ctx, scope, uuid.NewString(), aiworkbench.CreateInput{MemberID: "member-a"}, fixed, nil)
	require.NoError(t, err)
	key := uuid.NewString()
	request := aiworkbench.StoreRouteRequest{Content: "店铺数量", MemberID: "member-a", InputHash: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", ModelProfile: []byte(`{"version":"fixture"}`), ModelTimeout: time.Minute}
	_, _, err = store.BeginStoreRoute(ctx, scope, conversation.ID, key, request)
	require.NoError(t, err)
	lock := db.Begin()
	require.NoError(t, lock.Error)
	defer lock.Rollback()
	require.NoError(t, lock.Exec("UPDATE ai_workbench.conversations SET title=title WHERE id=?", conversation.ID).Error)
	require.NoError(t, db.Exec("UPDATE ai_workbench.commands SET route_commit_deadline=clock_timestamp()+INTERVAL '2 seconds' WHERE idempotency_key=?", key).Error)
	result := make(chan error, 1)
	go func() {
		_, _, e := store.CompleteStoreRoute(ctx, scope, key, aiworkbench.StoreRoute{Kind: "STORE_OVERVIEW", Status: "ACTIVE"})
		result <- e
	}()
	require.Eventually(t, func() bool {
		var waiting int64
		err := db.Raw("SELECT count(*) FROM pg_stat_activity WHERE datname=current_database() AND wait_event_type='Lock' AND query LIKE '%ai_workbench%conversations%'").Scan(&waiting).Error
		return err == nil && waiting > 0
	}, 1500*time.Millisecond, 25*time.Millisecond, "the T1 writer must be blocked on the conversation row before the deadline")
	time.Sleep(2300 * time.Millisecond)
	require.NoError(t, lock.Commit().Error)
	require.ErrorIs(t, <-result, aiworkbench.ErrUnavailable)
	messages, _, err := store.ListMessagesPage(ctx, scope, conversation.ID, 0, 50)
	require.NoError(t, err)
	require.Len(t, messages, 1)
}
