package aiworkbench

import (
	"context"
	"encoding/json"
	"time"

	"github.com/henryxu1008/shuomi-ai-commerce/internal/aicapability"
)

// ChatSkillReference is an exact library version, never a model-chosen identity.
type ChatSkillReference struct {
	Version uint64 `json:"version"`
	Digest  string `json:"digest"`
}

type ChatTurnRevision struct {
	Root         string          `json:"root"`
	BaseRevision string          `json:"baseRevision"`
	Snapshot     json.RawMessage `json:"snapshot"`
}

type ChatTurnInput struct {
	Content     string              `json:"content"`
	CreatorOnly bool                `json:"creatorOnly,omitempty"`
	Skill       *ChatSkillReference `json:"skill,omitempty"`
	Revision    *ChatTurnRevision   `json:"revision,omitempty"`
}

// ChatTurnAuthoringBinding is server-owned admission data for a new child.
type ChatTurnAuthoringBinding struct {
	ProfileDigest string                        `json:"profileDigest"`
	BillingMode   aicapability.ModelBillingMode `json:"billingMode"`
}

func (b *ChatTurnAuthoringBinding) Allows(profileDigest string, maximumPoints int64) bool {
	return b != nil && b.ProfileDigest == profileDigest &&
		(b.BillingMode == aicapability.ModelUsageOnly && maximumPoints == 0 ||
			b.BillingMode == aicapability.ModelPlatformPoints && maximumPoints > 0)
}

// ChatTurnEnvelope is frozen once at admission. A replay does not reload the
// available catalog or replace the explicitly selected version.
type ChatTurnEnvelope struct {
	Version      int                       `json:"version"`
	Input        ChatTurnInput             `json:"input"`
	Creator      *ChatSkillReference       `json:"creator,omitempty"`
	Authoring    *ChatTurnAuthoringBinding `json:"authoring,omitempty"`
	GeneralScope bool                      `json:"generalScope"`
}

// ParseDecision applies the frozen allow-list to both live output and receipts.
func (e ChatTurnEnvelope) ParseDecision(raw []byte) (StoreRoute, error) {
	result, err := ParseChatRoute(raw, e.Creator != nil)
	if err != nil {
		return StoreRoute{}, err
	}
	if e.Input.CreatorOnly && result.Kind != "CREATE_TITLE_SKILL" && result.Kind != "CLARIFY" {
		return StoreRoute{}, ErrInvalid
	}
	if !e.GeneralScope && result.Kind == "STORE_OVERVIEW" {
		return StoreRoute{}, ErrInvalid
	}
	return result, nil
}

type ChatTurnCommand struct {
	Route          StoreRouteCommand
	Envelope       ChatTurnEnvelope
	ToolState      string
	ToolKey        string
	AdmissionUntil time.Time
}

type ChatTurnResult struct {
	State     string              `json:"state"`
	Kind      string              `json:"kind,omitempty"`
	Skill     *ChatSkillReference `json:"skill,omitempty"`
	Authoring json.RawMessage     `json:"authoring,omitempty"`
}

// The view is the original author's authorized projection, not a second copy
// stored in a Chat message. HasDraft prevents routing success posing as output.
type ChatTurnSkillResult struct {
	State    string
	HasDraft bool
	View     json.RawMessage
}

type ChatTurnRoutes interface {
	Submit(context.Context, Scope, string, string, string, ChatTurnInput) (ChatTurnCommand, error)
	Read(context.Context, Scope, string, string, string) (ChatTurnCommand, error)
	Recover(context.Context, Scope, string, string, string) (ChatTurnCommand, error)
}

// Bind prepares through the admitted creator and atomically binds READY in the
// Workbench store. Run/Read consume that exact child through the existing owner.
type ChatTurnSkills interface {
	Bind(context.Context, ChatTurnCommand) (ChatTurnCommand, error)
	Run(context.Context, ChatTurnCommand) (ChatTurnSkillResult, error)
	Read(context.Context, ChatTurnCommand) (ChatTurnSkillResult, error)
}

type ChatTurnService struct {
	Routes ChatTurnRoutes
	Skills ChatTurnSkills
}

func (s ChatTurnService) Submit(ctx context.Context, scope Scope, conversation, key, member string, input ChatTurnInput) (ChatTurnResult, error) {
	if s.Routes == nil {
		return ChatTurnResult{}, ErrUnavailable
	}
	command, err := s.Routes.Submit(ctx, scope, conversation, key, member, input)
	if err != nil {
		return ChatTurnResult{}, err
	}
	return s.result(ctx, command, true)
}

func (s ChatTurnService) Read(ctx context.Context, scope Scope, conversation, key, member string) (ChatTurnResult, error) {
	if s.Routes == nil {
		return ChatTurnResult{}, ErrUnavailable
	}
	command, err := s.Routes.Read(ctx, scope, conversation, key, member)
	if err != nil {
		return ChatTurnResult{}, err
	}
	return s.result(ctx, command, false)
}

func (s ChatTurnService) Recover(ctx context.Context, scope Scope, conversation, key, member string) (ChatTurnResult, error) {
	if s.Routes == nil {
		return ChatTurnResult{}, ErrUnavailable
	}
	command, err := s.Routes.Recover(ctx, scope, conversation, key, member)
	if err != nil {
		return ChatTurnResult{}, err
	}
	return s.result(ctx, command, true)
}

func (s ChatTurnService) result(ctx context.Context, command ChatTurnCommand, execute bool) (ChatTurnResult, error) {
	result := ChatTurnResult{Kind: command.Route.Result}
	switch command.Route.State {
	case string(PlanningReadyToDispatch):
		result.State = "PROCESSING"
	case string(PlanningUnknown):
		result.State = "UNKNOWN"
	case "ROUTING_RESULT_LOST", string(PlanningInvalidOutput):
		result.State = "FAILED"
	case string(PlanningFailedBeforeDispatch):
		result.State = "NOT_STARTED"
	case string(PlanningComplete):
		result.State = "COMPLETE"
	default:
		return ChatTurnResult{}, ErrUnavailable
	}
	if command.Route.State != string(PlanningComplete) || command.Route.Result != "CREATE_TITLE_SKILL" {
		return result, nil
	}
	result.Skill = command.Envelope.Creator
	if s.Skills == nil {
		return ChatTurnResult{}, ErrUnavailable
	}
	if command.ToolState == "PENDING" && execute {
		var err error
		command, err = s.Skills.Bind(ctx, command)
		if err != nil {
			return ChatTurnResult{}, err
		}
	}
	switch command.ToolState {
	case "PENDING":
		result.State = "PROCESSING"
		return result, nil
	case "NOT_STARTED":
		result.State = "NOT_STARTED"
		return result, nil
	case "BOUND":
		if command.ToolKey == "" {
			return ChatTurnResult{}, ErrUnavailable
		}
	default:
		return ChatTurnResult{}, ErrUnavailable
	}
	var skill ChatTurnSkillResult
	var err error
	if execute {
		skill, err = s.Skills.Run(ctx, command)
	} else {
		skill, err = s.Skills.Read(ctx, command)
	}
	if err != nil {
		return ChatTurnResult{}, err
	}
	result.Authoring = skill.View
	switch skill.State {
	case "READY", "QUOTED":
		result.State = "PROCESSING"
	case "UNKNOWN":
		result.State = "UNKNOWN"
	case "FAILED", "INVALID":
		result.State = "FAILED"
	case "COMPLETE":
		if !skill.HasDraft || len(skill.View) == 0 {
			return ChatTurnResult{}, ErrUnavailable
		}
		result.State = "COMPLETE"
	default:
		return ChatTurnResult{}, ErrUnavailable
	}
	return result, nil
}
