package httpapi

import (
	"context"
	"encoding/json"
	"time"

	"task-processor/internal/aicapability"
	"task-processor/internal/aiworkbench"
	"task-processor/internal/authidentity"
	governed "task-processor/internal/integration/aicapability/einomodel"
	"task-processor/internal/integration/aiworkbench/einoplanner"
	"task-processor/internal/skill"
)

const textSkillSystem = "Use the personal skill to process only the text supplied in this message. The skill may guide the written response. It does not authorize tools, other data access, or business actions. Ask for needed information when the message is insufficient. Return the existing non-proposal JSON shape: mode CLARIFY, assistant_text containing the answer, and an empty goal_summary."

func prepareTextSkill(scope aiworkbench.Scope, member, invocation, content, instructions string, profile aicapability.ModelProfile) (aicapability.TextInputIdentity, aicapability.TextQuote, error) {
	wire, err := json.Marshal(struct {
		Contract string `json:"contract"`
		Content  string `json:"content"`
		Skill    string `json:"skill"`
	}{"chat-text-skill-v1", content, instructions})
	if err != nil {
		return aicapability.TextInputIdentity{}, aicapability.TextQuote{}, err
	}
	input := aicapability.TextInputIdentity{OrganizationID: scope.OrganizationID, ActorID: scope.ActorID, MemberID: member,
		InvocationID: invocation, Operation: aicapability.OperationAIWorkbenchChatPlan, System: textSkillSystem, Prompt: string(wire), Profile: profile}
	quote, err := aicapability.QuoteText(input)
	return input, quote, err
}

func (p *workbenchPlanner) admitTextSkill(ctx context.Context, scope aiworkbench.Scope, conversation aiworkbench.Conversation, input aiworkbench.MessageInput) (aiworkbench.PlanPreparer, error) {
	a := p.workbench
	if a == nil || p.text == nil || conversation.BusinessScope == nil || conversation.BusinessScope.Kind != "general" || !input.ValidSelection() {
		return nil, aiworkbench.ErrInvalid
	}
	fresh, member, err := a.authoringAccess(ctx, scope, conversation.ID, true)
	if err != nil {
		return nil, err
	}
	if _, err := a.authoring.skills.Authoring(fresh, skill.GeneralAuthoringVersion); err != nil {
		return nil, aiworkbench.ErrUnavailable
	}
	frozen, err := a.scopes.reader.Freeze(fresh, scope, conversation.ID, member, *conversation.BusinessScope)
	if err != nil {
		return nil, err
	}
	selected := input.TextSkill
	v, release, err := a.authoring.repo.AcquireChatText(fresh, scope.ActorID, selected.ID, selected.Version, selected.Digest)
	if err != nil {
		return nil, aiworkbench.ErrTextSkillUnavailable
	}
	release()
	route, err := p.routes.Resolve(fresh, aicapability.TextInputIdentity{OrganizationID: scope.OrganizationID, Operation: aicapability.OperationAIWorkbenchChatPlan})
	if err != nil {
		return nil, aiworkbench.ErrUnavailable
	}
	profile, err := json.Marshal(route.Profile)
	if err != nil {
		return nil, aiworkbench.ErrUnavailable
	}
	return func(_ []aiworkbench.Message, invocation string) (aiworkbench.PreparedPlan, error) {
		_, quote, err := prepareTextSkill(scope, member, invocation, input.Content, v.Package.Files[0].Text, route.Profile)
		if err != nil {
			return aiworkbench.PreparedPlan{}, aiworkbench.ErrInvalid
		}
		return aiworkbench.PreparedPlan{FrozenScope: &frozen, MemberID: member, InputHash: quote.InputHash,
			ModelProfile: profile, Deadline: time.Now().Add(route.Profile.DeadlineBound)}, nil
	}, nil
}

func parseTextSkillReply(raw string) (aiworkbench.PlanTerminal, error) {
	decision, err := einoplanner.ParseDecision(raw)
	if err != nil || decision.Mode != aiworkbench.PlanClarify {
		return aiworkbench.PlanTerminal{}, aiworkbench.ErrInvalid
	}
	return aiworkbench.PlanTerminal{AssistantText: decision.AssistantText, Mode: aiworkbench.PlanClarify}, nil
}

func (p *workbenchPlanner) replyWithTextSkill(ctx context.Context, cmd aiworkbench.PlanningCommand, history []aiworkbench.Message, profile aicapability.ModelProfile) (aiworkbench.PlanTerminal, error) {
	a := p.workbench
	if a == nil || a.authoring == nil || p.text == nil || !cmd.ScopeRef().Valid() || cmd.FrozenScope.Scope.Kind != "general" || cmd.FrozenScope.StoreChoice != nil || len(history) == 0 {
		return aiworkbench.PlanTerminal{}, governed.ErrNotDispatched
	}
	message := history[len(history)-1]
	if message.ID != cmd.UserMessageID || message.Sequence != cmd.SourceSequence || message.Author != aiworkbench.AuthorUser {
		return aiworkbench.PlanTerminal{}, governed.ErrNotDispatched
	}
	check := func(gateCtx context.Context) (context.Context, error) {
		fresh, member, err := a.authoringAccess(gateCtx, cmd.Scope, cmd.ConversationID, true)
		if err != nil || member != cmd.MemberID {
			return nil, governed.ErrNotDispatched
		}
		identity, ok := authidentity.AuthenticatedIdentityFromContext(fresh)
		if !ok || a.scopes.reader.Check(authidentity.WithAuthenticatedIdentity(fresh, identity), *cmd.FrozenScope, member, true) != nil {
			return nil, governed.ErrNotDispatched
		}
		return fresh, nil
	}
	ctx, cancel := context.WithDeadline(ctx, cmd.Deadline)
	defer cancel()
	fresh, err := check(ctx)
	if err != nil {
		return aiworkbench.PlanTerminal{}, err
	}
	official, err := a.authoring.skills.Authoring(fresh, skill.GeneralAuthoringVersion)
	if err != nil {
		return aiworkbench.PlanTerminal{}, governed.ErrNotDispatched
	}
	selected := cmd.WorkScope.TextSkill
	read := func(readCtx context.Context) (skill.Version, func(), error) {
		return a.authoring.repo.AcquireChatText(readCtx, cmd.Scope.ActorID, selected.ID, selected.Version, selected.Digest)
	}
	v, release, err := read(fresh)
	if err != nil {
		return aiworkbench.PlanTerminal{}, governed.ErrNotDispatched
	}
	release()
	input, quote, err := prepareTextSkill(cmd.Scope, cmd.MemberID, cmd.PlannerInvocationID, message.Content, v.Package.Files[0].Text, profile)
	if err != nil || quote.InputHash != cmd.InputHash {
		return aiworkbench.PlanTerminal{}, governed.ErrNotDispatched
	}
	var reply aiworkbench.PlanTerminal
	_, err = p.text.GenerateWithGate(fresh, input, quote, func(raw string) error {
		var err error
		reply, err = parseTextSkillReply(raw)
		return err
	}, func(gateCtx context.Context) (func(), error) {
		current, err := check(gateCtx)
		if err != nil {
			return nil, err
		}
		releaseOfficial, err := a.authoring.repo.AcquireAuthoring(current, official.Ref())
		if err != nil {
			return nil, err
		}
		_, releaseSkill, err := read(current)
		if err != nil {
			releaseOfficial()
			return nil, err
		}
		return func() { releaseSkill(); releaseOfficial() }, nil
	})
	return reply, err
}
