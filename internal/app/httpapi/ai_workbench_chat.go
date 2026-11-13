package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"task-processor/internal/aiworkbench"
	"task-processor/internal/authidentity"
	"task-processor/internal/storecenter"
)

func (a *aiWorkbenchApplication) listConversations(c *gin.Context, ctx context.Context, scope aiworkbench.Scope) {
	values := c.Request.URL.Query()
	for name, entries := range values {
		if len(entries) != 1 || (name != "after" && name != "limit" && name != "saved" && name != "archived") {
			writeAIWorkbenchError(c, aiworkbench.ErrInvalid)
			return
		}
	}
	if values.Has("after") && len(c.Query("after")) != 32 ||
		(values.Has("saved") && c.Query("saved") != "true") ||
		(values.Has("archived") && c.Query("archived") != "true") || values.Has("saved") && values.Has("archived") {
		writeAIWorkbenchError(c, aiworkbench.ErrInvalid)
		return
	}
	limit, err := workbenchSize(c)
	if err != nil {
		writeAIWorkbenchError(c, err)
		return
	}
	items, next, err := a.store.ListConversations(ctx, scope, c.Query("after"), limit, c.Query("saved") == "true", c.Query("archived") == "true")
	if err != nil {
		writeAIWorkbenchError(c, err)
		return
	}
	for index, item := range items {
		items[index], _ = a.authorizedConversation(ctx, scope, item)
	}
	workbenchReply(c, http.StatusOK, gin.H{"conversations": items, "next": next})
}

func (a *aiWorkbenchApplication) createConversation(c *gin.Context, ctx context.Context, scope aiworkbench.Scope) {
	if c.Request.URL.RawQuery != "" {
		writeAIWorkbenchError(c, aiworkbench.ErrInvalid)
		return
	}
	key, err := workbenchKey(c)
	if err != nil {
		writeAIWorkbenchError(c, err)
		return
	}
	var body struct {
		Favorite          bool                        `json:"favorite"`
		ScopeSelection    *aiworkbench.ScopeSelection `json:"scopeSelection"`
		ScopeConfirmation string                      `json:"scopeConfirmation"`
	}
	if err = workbenchJSON(c, &body); err != nil {
		writeAIWorkbenchError(c, err)
		return
	}
	if a.agent != nil && !a.agent.workbenchKind(aiworkbench.ContentTaskKind) {
		// The existing title-only Workbench has its own registered inputs.
		// Workbench scope applies to the content consumer and metadata-only Chat.
		if body.ScopeSelection != nil || body.ScopeConfirmation != "" {
			writeAIWorkbenchError(c, aiworkbench.ErrInvalid)
			return
		}
		conversation, replay, err := a.store.Create(ctx, scope, key, aiworkbench.CreateInput{Favorite: body.Favorite})
		if err != nil {
			writeAIWorkbenchError(c, err)
			return
		}
		workbenchReply(c, http.StatusOK, gin.H{"conversation": conversation, "replay": replay})
		return
	}
	if body.ScopeSelection == nil || !body.ScopeSelection.Valid() || len(body.ScopeConfirmation) != 64 {
		writeAIWorkbenchError(c, aiworkbench.ErrInvalid)
		return
	}
	i, _ := authidentity.AuthenticatedIdentityFromContext(ctx)
	input := aiworkbench.CreateInput{Favorite: body.Favorite, Selection: body.ScopeSelection, ScopeConfirmation: body.ScopeConfirmation, MemberID: i.EffectiveMemberID}
	if original, found, err := a.store.ReplayCreate(ctx, scope, key, input); found || err != nil {
		if err != nil {
			writeAIWorkbenchError(c, err)
			return
		}
		original, _ = a.authorizedConversation(ctx, scope, original)
		workbenchReply(c, http.StatusOK, gin.H{"conversation": original, "replay": true})
		return
	}
	view, confirmation, preflight := a.scopes.reader.Preview(ctx, scope, *input.Selection)
	if preflight == nil && confirmation != input.ScopeConfirmation {
		preflight = aiworkbench.ErrProposalStale
	}
	conversation, replay, err := a.store.CreateScoped(ctx, scope, key, input, &view.Scope, preflight)
	if err != nil {
		writeAIWorkbenchError(c, err)
		return
	}
	conversation, _ = a.authorizedConversation(ctx, scope, conversation)
	workbenchReply(c, http.StatusOK, gin.H{"conversation": conversation, "replay": replay})
}

func (a *aiWorkbenchApplication) getConversation(c *gin.Context, ctx context.Context, scope aiworkbench.Scope) {
	id := c.Param("conversation_id")
	if !acquisitionHTTPUUID(id) {
		writeAIWorkbenchError(c, aiworkbench.ErrInvalid)
		return
	}
	if len(c.Request.URL.Query()) > 2 {
		writeAIWorkbenchError(c, aiworkbench.ErrInvalid)
		return
	}
	limit, err := workbenchSize(c)
	if err != nil {
		writeAIWorkbenchError(c, err)
		return
	}
	var before uint64
	if value := c.Query("before"); value != "" {
		before, err = strconv.ParseUint(value, 10, 63)
		if err != nil || before == 0 || strconv.FormatUint(before, 10) != value {
			writeAIWorkbenchError(c, aiworkbench.ErrInvalid)
			return
		}
	}
	conversation, err := a.store.Get(ctx, scope, id)
	if err != nil {
		writeAIWorkbenchError(c, err)
		return
	}
	if visible, ok := a.authorizedConversation(ctx, scope, conversation); !ok {
		workbenchReply(c, http.StatusOK, gin.H{"conversation": visible, "messages": []any{}, "proposals": []any{}, "before": "", "scopeUnavailable": true})
		return
	}
	messages, next, err := a.store.ListMessagesPage(ctx, scope, id, before, limit)
	if err != nil {
		writeAIWorkbenchError(c, err)
		return
	}
	// Only the latest proposal can match the latest USER turn and be confirmed.
	// Older proposal cards are not rendered by the current Conversation view.
	proposals, err := a.store.ListProposals(ctx, scope, id, 1)
	if err != nil {
		writeAIWorkbenchError(c, err)
		return
	}
	cards := make([]workbenchProposalCard, 0, len(proposals))
	for _, proposal := range proposals {
		card, cardErr := a.proposalReceiptCard(ctx, scope, proposal)
		if cardErr != nil {
			writeAIWorkbenchError(c, cardErr)
			return
		}
		cards = append(cards, card)
	}
	projected := a.projectStoreMessages(ctx, scope, messages)
	for {
		page := gin.H{"conversation": conversation, "messages": projected, "proposals": cards, "before": next}
		wire, marshalErr := json.Marshal(page)
		if marshalErr != nil {
			writeAIWorkbenchError(c, aiworkbench.ErrUnavailable)
			return
		}
		if len(wire) <= workbenchResponseMaxBytes {
			c.Header("Cache-Control", "private, no-store")
			c.Data(http.StatusOK, "application/json; charset=utf-8", wire)
			return
		}
		if len(projected) <= 1 {
			writeAIWorkbenchError(c, aiworkbench.ErrUnavailable)
			return
		}
		// The store returns ascending sequences. Omitted older messages must be
		// reachable on the next page, even if the original 50-row page had no
		// database cursor.
		projected = projected[1:]
		next = strconv.FormatUint(projected[0].Sequence, 10)
	}
}

func (a *aiWorkbenchApplication) postStoreRoute(c *gin.Context, ctx context.Context, scope aiworkbench.Scope) {
	id := c.Param("conversation_id")
	if !acquisitionHTTPUUID(id) || c.Request.URL.RawQuery != "" {
		writeAIWorkbenchError(c, aiworkbench.ErrInvalid)
		return
	}
	key, err := workbenchKey(c)
	if err != nil {
		writeAIWorkbenchError(c, err)
		return
	}
	var body struct {
		Content string `json:"content"`
	}
	if err = workbenchJSON(c, &body); err != nil {
		writeAIWorkbenchError(c, err)
		return
	}
	identity, ok := authidentity.AuthenticatedIdentityFromContext(ctx)
	if !ok || identity.EffectiveMemberID == "" || a.storeRoute == nil {
		writeAIWorkbenchError(c, aiworkbench.ErrUnavailable)
		return
	}
	command, err := a.storeRoute.Route(ctx, scope, id, key, identity.EffectiveMemberID, body.Content)
	if err != nil {
		writeAIWorkbenchError(c, err)
		return
	}
	workbenchReply(c, http.StatusOK, gin.H{"state": command.State})
}

func (a *aiWorkbenchApplication) postExplicitStoreOverview(c *gin.Context, ctx context.Context, scope aiworkbench.Scope) {
	id := c.Param("conversation_id")
	if !acquisitionHTTPUUID(id) || c.Request.URL.RawQuery != "" {
		writeAIWorkbenchError(c, aiworkbench.ErrInvalid)
		return
	}
	key, err := workbenchKey(c)
	if err != nil {
		writeAIWorkbenchError(c, err)
		return
	}
	var body struct {
		Status string `json:"status"`
	}
	if err = workbenchJSON(c, &body); err != nil || body.Status != "ALL_NON_DELETED" && body.Status != "ACTIVE" {
		writeAIWorkbenchError(c, aiworkbench.ErrInvalid)
		return
	}
	identity, ok := authidentity.AuthenticatedIdentityFromContext(ctx)
	if !ok || a.scopes.storeOverview == nil || a.store == nil || identity.EffectiveMemberID == "" {
		writeAIWorkbenchError(c, aiworkbench.ErrUnavailable)
		return
	}
	if err = a.store.ConversationMember(ctx, scope, id, identity.EffectiveMemberID); err != nil {
		writeAIWorkbenchError(c, err)
		return
	}
	if _, err = a.scopes.storeOverview(ctx, scope, storecenter.OverviewScope(body.Status)); err != nil {
		writeAIWorkbenchError(c, aiworkbench.ErrUnavailable)
		return
	}
	command, replay, err := a.store.ExplicitStoreOverview(ctx, scope, id, key, identity.EffectiveMemberID, body.Status)
	if err != nil {
		writeAIWorkbenchError(c, err)
		return
	}
	if _, err = a.scopes.storeOverview(ctx, scope, storecenter.OverviewScope(body.Status)); err != nil {
		writeAIWorkbenchError(c, aiworkbench.ErrUnavailable)
		return
	}
	workbenchReply(c, http.StatusOK, gin.H{"state": command.State, "replay": replay})
}

func (a *aiWorkbenchApplication) proposalReceiptCard(ctx context.Context, scope aiworkbench.Scope, proposal aiworkbench.ExecutionProposal) (workbenchProposalCard, error) {
	if proposal.Scope != scope {
		return workbenchProposalCard{}, aiworkbench.ErrNotFound
	}
	card := a.proposalCard(ctx, proposal)
	if !proposal.WorkbenchContextRef.Absent() {
		f, err := a.authorizedFrozenScope(ctx, scope, proposal.WorkbenchContextRef, false)
		if err != nil {
			return workbenchProposalCard{}, err
		}
		card.BusinessScope = &aiworkbench.ScopeView{Scope: f.Scope, ProjectName: f.ModelInput.ProjectTitle, StoreName: f.ModelInput.StoreName}
	}
	task, found, err := a.store.LookupProposalTask(ctx, scope, proposal.ConversationID, proposal.ID)
	if err != nil || found && task.ProposalDigest != proposal.Digest {
		return workbenchProposalCard{}, aiworkbench.ErrUnavailable
	}
	if found {
		card.ConfirmedTaskID = task.ID
	}
	return card, nil
}

func (a *aiWorkbenchApplication) changeConversation(c *gin.Context, ctx context.Context, scope aiworkbench.Scope) {
	id := c.Param("conversation_id")
	if !acquisitionHTTPUUID(id) || c.Request.URL.RawQuery != "" {
		writeAIWorkbenchError(c, aiworkbench.ErrInvalid)
		return
	}
	version := c.GetHeader("If-Match")
	revision, err := strconv.ParseUint(version, 10, 63)
	if err != nil || revision == 0 || strconv.FormatUint(revision, 10) != version {
		writeAIWorkbenchError(c, aiworkbench.ErrInvalid)
		return
	}
	var body struct {
		Title    *string `json:"title,omitempty"`
		Favorite *bool   `json:"favorite,omitempty"`
		Archived *bool   `json:"archived,omitempty"`
	}
	if err := workbenchJSON(c, &body); err != nil {
		writeAIWorkbenchError(c, err)
		return
	}
	updated, err := a.store.SetMetadata(ctx, scope, id, revision, aiworkbench.MetadataChange{Title: body.Title, Favorite: body.Favorite, Archived: body.Archived})
	if err != nil {
		writeAIWorkbenchError(c, err)
		return
	}
	updated, _ = a.authorizedConversation(ctx, scope, updated)
	workbenchReply(c, http.StatusOK, gin.H{"conversation": updated})
}

func (a *aiWorkbenchApplication) postMessage(c *gin.Context, ctx context.Context, scope aiworkbench.Scope) {
	id := c.Param("conversation_id")
	if !acquisitionHTTPUUID(id) || c.Request.URL.RawQuery != "" {
		writeAIWorkbenchError(c, aiworkbench.ErrInvalid)
		return
	}
	key, err := workbenchKey(c)
	if err != nil {
		writeAIWorkbenchError(c, err)
		return
	}
	var body workbenchMessageBody
	if err := workbenchJSON(c, &body); err != nil {
		writeAIWorkbenchError(c, err)
		return
	}
	command, err := a.service.Message(ctx, scope, id, key, body.input())
	if err != nil {
		writeAIWorkbenchError(c, err)
		return
	}
	if command.FrozenScope != nil {
		if _, scopeErr := a.authorizedFrozenScope(ctx, scope, command.ScopeRef(), false); scopeErr != nil {
			workbenchReply(c, http.StatusAccepted, gin.H{"state": command.State, "scopeUnavailable": true})
			return
		}
	}
	response := gin.H{"state": command.State, "userMessageId": command.UserMessageID,
		"sourceSequence": command.SourceSequence, "assistantMessageId": command.AssistantMessageID,
		"proposalId": command.ProposalID}
	if command.ProposalID != "" {
		proposal, err := a.store.GetProposal(ctx, scope, command.ProposalID)
		if err != nil {
			writeAIWorkbenchError(c, err)
			return
		}
		card, cardErr := a.proposalReceiptCard(ctx, scope, proposal)
		if cardErr != nil {
			writeAIWorkbenchError(c, cardErr)
			return
		}
		response["proposal"] = card
	}
	if accessErr := a.plan.AuthorizeReceipt(ctx, scope); accessErr != nil {
		writeAIWorkbenchError(c, accessErr)
		return
	}
	status := http.StatusOK
	if command.State == aiworkbench.PlanningReadyToDispatch || command.State == aiworkbench.PlanningUnknown {
		status = http.StatusAccepted
	}
	workbenchReply(c, status, response)
}

func (a *aiWorkbenchApplication) confirmProposal(c *gin.Context, ctx context.Context, scope aiworkbench.Scope) {
	conversationID, proposalID := c.Param("conversation_id"), c.Param("proposal_id")
	if !acquisitionHTTPUUID(conversationID) || !acquisitionHTTPUUID(proposalID) || c.Request.URL.RawQuery != "" || workbenchEmptyBody(c) != nil {
		writeAIWorkbenchError(c, aiworkbench.ErrInvalid)
		return
	}
	key, err := workbenchKey(c)
	if err != nil {
		writeAIWorkbenchError(c, err)
		return
	}
	task, replay, err := a.service.Confirm(ctx, scope, conversationID, proposalID, key)
	if err != nil && task.ID == "" {
		if errors.Is(err, aiworkbench.ErrRevisionMismatch) {
			err = aiworkbench.ErrProposalStale
		}
		writeAIWorkbenchError(c, err)
		return
	}
	view, viewErr := a.taskView(ctx, scope, task, false)
	if viewErr != nil {
		writeAIWorkbenchError(c, viewErr)
		return
	}
	if view.ScopeUnavailable {
		writeAIWorkbenchError(c, aiworkbench.ErrNotFound)
		return
	}
	if accessErr := (workbenchExecution{agent: a.agent}).AuthorizeReceipt(ctx, scope); accessErr != nil {
		writeAIWorkbenchError(c, accessErr)
		return
	}
	workbenchReply(c, http.StatusOK, gin.H{"task": view, "replay": replay})
}
