package httpapi

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	strict "sigs.k8s.io/json"
	"task-processor/internal/aicapability"
	"task-processor/internal/aiworkbench"
	"task-processor/internal/authidentity"
	"task-processor/internal/authz"
	"task-processor/internal/integration/aiworkbench/einorouter"
	wbstore "task-processor/internal/integration/persistence/aiworkbench"
)

func normalizeChatTurn(input aiworkbench.ChatTurnInput) (aiworkbench.ChatTurnInput, error) {
	if strings.TrimSpace(input.Content) == "" || len(input.Content) > 8192 || !utf8.ValidString(input.Content) {
		return input, aiworkbench.ErrInvalid
	}
	if input.Skill != nil {
		raw, err := hex.DecodeString(input.Skill.Digest)
		if err != nil || len(raw) != 32 || hex.EncodeToString(raw) != input.Skill.Digest || (input.Skill.Version != 1 && input.Skill.Version != 2) {
			return input, aiworkbench.ErrInvalid
		}
	}
	if r := input.Revision; r != nil {
		if input.Skill == nil || input.Skill.Version != 2 || !acquisitionHTTPUUID(r.Root) || !acquisitionHTTPUUID(r.BaseRevision) || len(r.Snapshot) > 65536 {
			return input, aiworkbench.ErrInvalid
		}
		var snapshot titleSkillDraft
		bad, err := strict.UnmarshalStrict(r.Snapshot, &snapshot, strict.DisallowDuplicateFields, strict.DisallowUnknownFields)
		if err != nil || len(bad) != 0 {
			return input, aiworkbench.ErrInvalid
		}
		raw, err := json.Marshal(snapshot)
		if err != nil || len(raw) > 65536 {
			return input, aiworkbench.ErrInvalid
		}
		copy := *r
		copy.Snapshot = raw
		input.Revision = &copy
	}
	return input, nil
}

// One request-scoped adapter extends the existing routing owner. It does not
// create a second router, message log or recovery protocol.
type workbenchChatTurnRoutes struct{ app *aiWorkbenchApplication }
type chatTurnRepository struct {
	*wbstore.Store
	input   aiworkbench.ChatTurnInput
	general bool
}

func (s chatTurnRepository) ReplayStoreRoute(ctx context.Context, scope aiworkbench.Scope, conversation, key, member, _ string) (aiworkbench.StoreRouteCommand, bool, error) {
	c, found, err := s.ReplayChatTurn(ctx, scope, conversation, key, member, s.input)
	return c.Route, found, err
}
func (s chatTurnRepository) BeginStoreRoute(ctx context.Context, scope aiworkbench.Scope, conversation, key string, r aiworkbench.StoreRouteRequest) (aiworkbench.StoreRouteCommand, bool, error) {
	envelope := aiworkbench.ChatTurnEnvelope{Version: 1, Input: s.input, GeneralScope: s.general}
	if len(r.TurnInput) > 0 && json.Unmarshal(r.TurnInput, &envelope) != nil {
		return aiworkbench.StoreRouteCommand{}, false, aiworkbench.ErrInvalid
	}
	c, replay, err := s.BeginChatTurn(ctx, scope, conversation, key, envelope, r)
	return c.Route, replay, err
}

type chatTurnGateway struct {
	*workbenchStoreRouter
	input aiworkbench.ChatTurnInput
}

func (p chatTurnGateway) Authorize(ctx context.Context, scope aiworkbench.Scope, conversation, member string) error {
	a := p.app
	identity, err := a.freshWorkbenchIdentity(ctx, authz.PermissionWorkbenchChatUse)
	if err != nil || identity.TenantID != scope.OrganizationID || identity.UserID != scope.ActorID || identity.EffectiveMemberID != member {
		return aiworkbench.ErrNotFound
	}
	if err = a.store.ConversationMember(ctx, scope, conversation, member); err != nil {
		return err
	}
	row, err := a.store.Get(ctx, scope, conversation)
	if err != nil || row.Archived {
		return aiworkbench.ErrNotFound
	}
	if _, ok := a.authorizedConversation(ctx, scope, row); !ok {
		return aiworkbench.ErrNotFound
	}
	return nil
}
func (p chatTurnGateway) Admit(ctx context.Context, scope aiworkbench.Scope, conversation, member, content string) (aiworkbench.StoreRouteRequest, error) {
	if !p.app.chatTurnsEnabled || !p.app.store.CanChatTurns(ctx) {
		return aiworkbench.StoreRouteRequest{}, aiworkbench.ErrUnavailable
	}
	row, err := p.app.store.Get(ctx, scope, conversation)
	if err != nil {
		return aiworkbench.StoreRouteRequest{}, err
	}
	envelope := aiworkbench.ChatTurnEnvelope{Version: 1, Input: p.input, GeneralScope: row.BusinessScope != nil && row.BusinessScope.Kind == "general"}
	// Freeze only an actually available, authorized library version. Selecting
	// it bypasses intent inference; selecting an invalid version never reroutes.
	if p.app.authoring != nil {
		personal, _, accessErr := p.app.authoringAccess(ctx, scope, conversation, true)
		if accessErr == nil {
			route, routeErr := p.app.authoring.routes.Resolve(personal, aicapability.TextInputIdentity{OrganizationID: scope.OrganizationID, Operation: authoringOperation})
			format := authoringProfileFormat(route.Profile)
			if routeErr == nil && format != 0 {
				version, versionErr := p.app.authoring.skills.Authoring(personal, uint64(format))
				if versionErr == nil {
					envelope.Creator = &aiworkbench.ChatSkillReference{Version: uint64(format), Digest: version.Package.Digest}
				}
			}
		}
	}
	if p.input.Skill != nil && (envelope.Creator == nil || *p.input.Skill != *envelope.Creator) {
		return aiworkbench.StoreRouteRequest{}, aiworkbench.ErrProposalStale
	}
	raw, err := json.Marshal(envelope)
	if err != nil {
		return aiworkbench.StoreRouteRequest{}, aiworkbench.ErrInvalid
	}
	request := aiworkbench.StoreRouteRequest{Content: content, MemberID: member, TurnInput: raw}
	if p.input.Skill != nil {
		return request, nil
	}
	if p.routes == nil {
		return request, aiworkbench.ErrUnavailable
	}
	route, err := p.routes.Resolve(ctx, aicapability.TextInputIdentity{OrganizationID: scope.OrganizationID, Operation: aicapability.OperationAIWorkbenchChatIntent})
	if err != nil {
		return request, aiworkbench.ErrUnavailable
	}
	profile := route.Profile
	if _, err = einorouter.PrepareChat(scope, member, "preflight", envelope, profile); err != nil {
		return request, err
	}
	request.Prepare = func(invocation string) (aiworkbench.StoreRoutePrepared, error) {
		prepared, err := einorouter.PrepareChat(scope, member, invocation, envelope, profile)
		if err != nil {
			return aiworkbench.StoreRoutePrepared{}, err
		}
		wire, err := json.Marshal(profile)
		return aiworkbench.StoreRoutePrepared{InputHash: prepared.Quote.InputHash, ModelProfile: wire, ModelTimeout: profile.DeadlineBound}, err
	}
	return request, nil
}
func (p chatTurnGateway) Decide(ctx context.Context, command aiworkbench.StoreRouteCommand) (aiworkbench.StoreRoute, error) {
	if err := p.Authorize(ctx, command.Scope, command.ConversationID, command.MemberID); err != nil {
		return aiworkbench.StoreRoute{}, err
	}
	var envelope aiworkbench.ChatTurnEnvelope
	var profile aicapability.ModelProfile
	if json.Unmarshal(command.TurnInput, &envelope) != nil || json.Unmarshal(command.ModelProfile, &profile) != nil {
		return aiworkbench.StoreRoute{}, aiworkbench.ErrUnavailable
	}
	prepared, err := einorouter.PrepareChat(command.Scope, command.MemberID, command.InvocationID, envelope, profile)
	if err != nil || prepared.Quote.InputHash != command.InputHash {
		return aiworkbench.StoreRoute{}, aiworkbench.ErrUnavailable
	}
	return (einorouter.Router{Text: gatedStoreRouteText{router: p.workbenchStoreRouter, command: command, authorize: p.Authorize}}).Decide(ctx, prepared)
}
func (p workbenchChatTurnRoutes) Submit(ctx context.Context, scope aiworkbench.Scope, conversation, key, member string, input aiworkbench.ChatTurnInput) (aiworkbench.ChatTurnCommand, error) {
	input, err := normalizeChatTurn(input)
	if err != nil {
		return aiworkbench.ChatTurnCommand{}, err
	}
	if p.app.storeRoute == nil || !p.app.store.CanChatTurns(ctx) {
		return aiworkbench.ChatTurnCommand{}, aiworkbench.ErrUnavailable
	}
	base, ok := p.app.storeRoute.Gateway.(*workbenchStoreRouter)
	if !ok {
		return aiworkbench.ChatTurnCommand{}, aiworkbench.ErrUnavailable
	}
	gateway := chatTurnGateway{workbenchStoreRouter: base, input: input}
	if err = gateway.Authorize(ctx, scope, conversation, member); err != nil {
		return aiworkbench.ChatTurnCommand{}, err
	}
	_, found, err := p.app.store.ReplayChatTurn(ctx, scope, conversation, key, member, input)
	if err != nil {
		return aiworkbench.ChatTurnCommand{}, err
	}
	if !found && !p.app.chatTurnsEnabled {
		return aiworkbench.ChatTurnCommand{}, aiworkbench.ErrUnavailable
	}
	row, err := p.app.store.Get(ctx, scope, conversation)
	if err != nil {
		return aiworkbench.ChatTurnCommand{}, err
	}
	service := aiworkbench.StoreRouteService{Store: chatTurnRepository{Store: p.app.store, input: input, general: row.BusinessScope != nil && row.BusinessScope.Kind == "general"}, Gateway: gateway}
	_, err = service.Route(ctx, scope, conversation, key, member, input.Content)
	if err != nil {
		return aiworkbench.ChatTurnCommand{}, err
	}
	return p.app.store.ReadChatTurn(ctx, scope, conversation, key, member)
}
func (p workbenchChatTurnRoutes) Read(ctx context.Context, scope aiworkbench.Scope, conversation, key, member string) (aiworkbench.ChatTurnCommand, error) {
	return p.app.store.ReadChatTurn(ctx, scope, conversation, key, member)
}
func (p workbenchChatTurnRoutes) Recover(ctx context.Context, scope aiworkbench.Scope, conversation, key, member string) (aiworkbench.ChatTurnCommand, error) {
	original, err := p.Read(ctx, scope, conversation, key, member)
	if err != nil {
		return original, err
	}
	return p.Submit(ctx, scope, conversation, key, member, original.Envelope.Input)
}

type workbenchChatTurnSkills struct{ app *aiWorkbenchApplication }

func (p workbenchChatTurnSkills) Bind(ctx context.Context, command aiworkbench.ChatTurnCommand) (aiworkbench.ChatTurnCommand, error) {
	scope := command.Route.Scope
	fresh, member, err := p.app.authoringAccess(ctx, scope, command.Route.ConversationID, true)
	if err != nil || member != command.Route.MemberID {
		return command, aiworkbench.ErrNotFound
	}
	command, err = p.app.store.ResolveChatTurnBinding(fresh, command)
	if err != nil || command.ToolState != "PENDING" {
		return command, err
	}
	if command.Envelope.Creator == nil {
		return command, aiworkbench.ErrUnavailable
	}
	key, err := uuid.NewV7()
	if err != nil {
		return command, aiworkbench.ErrUnavailable
	}
	input := authoringPreviewInput{Requirement: command.Envelope.Input.Content, SkillVersion: command.Envelope.Creator.Version, SkillDigest: command.Envelope.Creator.Digest}
	if revision := command.Envelope.Input.Revision; revision != nil {
		input.Root, input.BaseRevision = revision.Root, revision.BaseRevision
		var snapshot titleSkillDraft
		if json.Unmarshal(revision.Snapshot, &snapshot) != nil {
			return command, aiworkbench.ErrInvalid
		}
		input.Snapshot = &snapshot
	}
	candidate, err := newAuthoringRequest(input, command.Route.ConversationID, member, key.String(), scope)
	if err == nil {
		var quote aicapability.TextQuote
		var points int64
		var identity aicapability.TextInputIdentity
		identity, quote, points, _, err = p.app.authoring.prepare(fresh, scope, candidate)
		if err == nil {
			candidate.InputHash, candidate.ProfileDigest, candidate.MaximumPoints = quote.InputHash, quote.ProfileDigest, points
			candidate.ExpiresAt, err = authoringQuoteDeadline(candidate.Key)
			deadline := time.Now().UTC().Add(identity.Profile.DeadlineBound)
			candidate.Deadline = &deadline
		}
	}
	if err != nil {
		original, readErr := p.app.store.ResolveChatTurnBinding(fresh, command)
		if readErr == nil && original.ToolState != "PENDING" {
			return original, nil
		}
		return command, err
	}
	return p.app.store.BindChatTurnAuthoring(fresh, command, candidate)
}
func (p workbenchChatTurnSkills) view(ctx context.Context, command aiworkbench.ChatTurnCommand, run bool) (aiworkbench.ChatTurnSkillResult, error) {
	fresh, member, err := p.app.authoringAccess(ctx, command.Route.Scope, command.Route.ConversationID, run)
	if err != nil || member != command.Route.MemberID {
		return aiworkbench.ChatTurnSkillResult{}, aiworkbench.ErrNotFound
	}
	row, err := p.app.store.AuthoringDraft(fresh, command.Route.Scope, command.Route.ConversationID, command.ToolKey)
	if err != nil || row.MemberID != member {
		return aiworkbench.ChatTurnSkillResult{}, aiworkbench.ErrNotFound
	}
	if run {
		row, err = p.app.generateSkillAuthoring(fresh, command.Route.Scope, row)
		if err != nil {
			return aiworkbench.ChatTurnSkillResult{}, err
		}
	}
	view, err := p.app.authoringView(fresh, command.Route.Scope, row)
	if err != nil {
		return aiworkbench.ChatTurnSkillResult{}, err
	}
	wire, err := json.Marshal(view)
	draft, hasDraft := view["draft"].(*titleSkillDraft)
	return aiworkbench.ChatTurnSkillResult{State: row.State, HasDraft: hasDraft && draft != nil, View: wire}, err
}
func (p workbenchChatTurnSkills) Run(ctx context.Context, c aiworkbench.ChatTurnCommand) (aiworkbench.ChatTurnSkillResult, error) {
	return p.view(ctx, c, true)
}
func (p workbenchChatTurnSkills) Read(ctx context.Context, c aiworkbench.ChatTurnCommand) (aiworkbench.ChatTurnSkillResult, error) {
	return p.view(ctx, c, false)
}

func (a *aiWorkbenchApplication) chatTurn(c *gin.Context, ctx context.Context, scope aiworkbench.Scope, action string) {
	permission := authz.PermissionWorkbenchChatRead
	if c.Request.Method == "POST" {
		permission = authz.PermissionWorkbenchChatUse
	}
	fresh, err := a.scopeIdentity(ctx, scope, permission)
	if err != nil {
		writeAIWorkbenchError(c, err)
		return
	}
	identity, _ := authidentity.AuthenticatedIdentityFromContext(fresh)
	conversation := c.Param("conversation_id")
	row, err := a.store.Get(fresh, scope, conversation)
	if err == nil {
		if _, ok := a.authorizedConversation(fresh, scope, row); !ok {
			err = aiworkbench.ErrNotFound
		}
	}
	if err == nil && row.Archived && c.Request.Method == "POST" {
		err = aiworkbench.ErrConversationArchived
	}
	if err == nil && c.Request.URL.RawQuery != "" {
		err = aiworkbench.ErrInvalid
	}
	var result aiworkbench.ChatTurnResult
	service := aiworkbench.ChatTurnService{Routes: workbenchChatTurnRoutes{app: a}, Skills: workbenchChatTurnSkills{app: a}}
	if err == nil {
		key := c.Param("turn_key")
		switch action {
		case "chat-turn-submit":
			var input aiworkbench.ChatTurnInput
			key, err = workbenchKey(c)
			if err == nil {
				err = workbenchJSONLimit(c, &input, 96<<10)
			}
			if err == nil {
				result, err = service.Submit(fresh, scope, conversation, key, identity.EffectiveMemberID, input)
			}
		case "chat-turn-read":
			if !acquisitionHTTPUUID(key) {
				err = aiworkbench.ErrInvalid
			} else {
				result, err = service.Read(fresh, scope, conversation, key, identity.EffectiveMemberID)
			}
		case "chat-turn-recover":
			if !acquisitionHTTPUUID(key) {
				err = aiworkbench.ErrInvalid
			} else {
				err = workbenchEmptyBody(c)
			}
			if err == nil {
				result, err = service.Recover(fresh, scope, conversation, key, identity.EffectiveMemberID)
			}
		default:
			err = aiworkbench.ErrInvalid
		}
	}
	if err != nil {
		if errors.Is(err, aiworkbench.ErrNotFound) {
			writeAIWorkbenchError(c, aiworkbench.ErrNotFound)
		} else {
			writeAIWorkbenchError(c, err)
		}
		return
	}
	workbenchReply(c, 200, result)
}
