package aiworkbench

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestChatRouteOnlyAllowsAnAvailableCreatorAndClosedFields(t *testing.T) {
	raw := []byte(`{"kind":"CREATE_TITLE_SKILL","status":""}`)
	_, err := ParseStoreRoute(raw)
	require.Error(t, err, "the original frozen route format stays closed")
	_, err = ParseChatRoute(raw, false)
	require.Error(t, err)
	decision, err := ParseChatRoute(raw, true)
	require.NoError(t, err)
	require.Equal(t, "CREATE_TITLE_SKILL", decision.Kind)
	for _, invalid := range []string{
		`{"kind":"CREATE_TITLE_SKILL","status":"ACTIVE"}`,
		`{"kind":"CREATE_TITLE_SKILL","status":"","organizationId":"other"}`,
		`{"kind":"CREATE_TITLE_SKILL","kind":"CLARIFY","status":""}`,
		`{"kind":"SAVE_SKILL","status":""}`,
	} {
		_, err = ParseChatRoute([]byte(invalid), true)
		require.Error(t, err)
	}
}

type turnRoutesStub struct{ command ChatTurnCommand }

func (s *turnRoutesStub) Submit(context.Context, Scope, string, string, string, ChatTurnInput) (ChatTurnCommand, error) {
	return s.command, nil
}
func (s *turnRoutesStub) Read(context.Context, Scope, string, string, string) (ChatTurnCommand, error) {
	return s.command, nil
}
func (s *turnRoutesStub) Recover(context.Context, Scope, string, string, string) (ChatTurnCommand, error) {
	return s.command, nil
}

type turnSkillStub struct {
	binds, runs, reads int
	result             ChatTurnSkillResult
}

func (s *turnSkillStub) Bind(_ context.Context, command ChatTurnCommand) (ChatTurnCommand, error) {
	s.binds++
	command.ToolState, command.ToolKey = "BOUND", "original-child"
	return command, nil
}
func (s *turnSkillStub) Run(context.Context, ChatTurnCommand) (ChatTurnSkillResult, error) {
	s.runs++
	return s.result, nil
}
func (s *turnSkillStub) Read(context.Context, ChatTurnCommand) (ChatTurnSkillResult, error) {
	s.reads++
	return s.result, nil
}

func TestChatTurnRoutingCompleteStillRunsTheSingleBoundAuthoring(t *testing.T) {
	routes := &turnRoutesStub{command: ChatTurnCommand{Route: StoreRouteCommand{State: "COMPLETE", Result: "CREATE_TITLE_SKILL"}, ToolState: "PENDING"}}
	skills := &turnSkillStub{result: ChatTurnSkillResult{State: "COMPLETE", HasDraft: true, View: json.RawMessage(`{"key":"original-child","draft":{"name":"title"}}`)}}
	service := ChatTurnService{Routes: routes, Skills: skills}
	result, err := service.Submit(context.Background(), Scope{}, "conversation", "message", "member", ChatTurnInput{Content: "创建标题技能"})
	require.NoError(t, err)
	require.Equal(t, "COMPLETE", result.State)
	require.Equal(t, 1, skills.binds)
	require.Equal(t, 1, skills.runs)
	require.JSONEq(t, string(skills.result.View), string(result.Authoring))
	routes.command.ToolState, routes.command.ToolKey = "BOUND", "original-child"
	_, err = service.Recover(context.Background(), Scope{}, "conversation", "message", "member")
	require.NoError(t, err)
	require.Equal(t, 1, skills.binds, "a bound request never prepares another child")
}

func TestChatTurnReadNeverBindsOrDispatchesAndUncertainRoutesNeverCreate(t *testing.T) {
	routes := &turnRoutesStub{command: ChatTurnCommand{Route: StoreRouteCommand{State: "COMPLETE", Result: "CREATE_TITLE_SKILL"}, ToolState: "PENDING"}}
	skills := &turnSkillStub{}
	service := ChatTurnService{Routes: routes, Skills: skills}
	result, err := service.Read(context.Background(), Scope{}, "conversation", "message", "member")
	require.NoError(t, err)
	require.Equal(t, "PROCESSING", result.State)
	require.Zero(t, skills.binds)
	require.Zero(t, skills.runs)
	for _, state := range []string{"READY_TO_DISPATCH", "PLANNER_UNKNOWN", "ROUTING_RESULT_LOST", "FAILED_BEFORE_DISPATCH", "PLANNER_INVALID_OUTPUT"} {
		routes.command.Route.State = state
		_, err = service.Recover(context.Background(), Scope{}, "conversation", "message", "member")
		require.NoError(t, err)
		require.Zero(t, skills.binds)
		require.Zero(t, skills.runs)
	}
	routes.command.Route.State, routes.command.ToolState = "COMPLETE", "NOT_STARTED"
	result, err = service.Recover(context.Background(), Scope{}, "conversation", "message", "member")
	require.NoError(t, err)
	require.Equal(t, "NOT_STARTED", result.State)
	require.Zero(t, skills.binds)
}

func TestChatTurnDoesNotClaimACompleteDraftWithoutItsProjection(t *testing.T) {
	routes := &turnRoutesStub{command: ChatTurnCommand{Route: StoreRouteCommand{State: "COMPLETE", Result: "CREATE_TITLE_SKILL"}, ToolState: "BOUND", ToolKey: "original-child"}}
	skills := &turnSkillStub{result: ChatTurnSkillResult{State: "COMPLETE"}}
	_, err := (ChatTurnService{Routes: routes, Skills: skills}).Read(context.Background(), Scope{}, "conversation", "message", "member")
	require.ErrorIs(t, err, ErrUnavailable)
	require.Equal(t, 1, skills.reads)
	require.Zero(t, skills.runs)
}
