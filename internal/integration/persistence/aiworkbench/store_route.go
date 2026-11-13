package aiworkbenchpersistence

import (
	"context"
	"encoding/json"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"task-processor/internal/aiworkbench"
)

const routeOperation = "chat_intent_route"
const explicitOverviewOperation = "chat_store_overview"

// ExplicitStoreOverview writes a neutral reference through the Chat owner in
// one transaction. It never invokes or reserves a model.
func (s *Store) ExplicitStoreOverview(ctx context.Context, scope aiworkbench.Scope, conversationID, key, member, status string) (aiworkbench.StoreRouteCommand, bool, error) {
	var result aiworkbench.StoreRouteCommand
	var replay bool
	if s == nil || s.db == nil || ctx == nil || !validScope(scope) || !validKey(conversationID) || !validKey(key) || !validName(member) || (status != "ALL_NON_DELETED" && status != "ACTIVE") {
		return result, false, aiworkbench.ErrInvalid
	}
	fingerprint := digest(struct{ Operation, ConversationID, Status string }{explicitOverviewOperation, conversationID, status})
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := s.lockKey(tx, scope, key); err != nil {
			return err
		}
		var members int64
		if err := tx.Model(&commandRow{}).Where("organization_id=? AND actor_id=? AND conversation_id=? AND operation=? AND member_id=?", scope.OrganizationID, scope.ActorID, conversationID, createOperation, member).Count(&members).Error; err != nil {
			return err
		}
		if members != 1 {
			return aiworkbench.ErrNotFound
		}
		existing, found, err := s.lookupCommand(tx, scope, key)
		if err != nil {
			return err
		}
		if found {
			if existing.Operation != explicitOverviewOperation || existing.ConversationID != conversationID || existing.RequestFingerprint != fingerprint || existing.MemberID == nil || *existing.MemberID != member {
				return aiworkbench.ErrIdempotencyConflict
			}
			result, replay = routeCommand(existing), true
			return nil
		}
		conversation, err := s.lookupConversation(tx, scope, conversationID, true)
		if err != nil {
			return err
		}
		if conversation.Lifecycle != "ACTIVE" {
			return aiworkbench.ErrConversationArchived
		}
		if conversation.BusinessScope == nil || conversation.BusinessScope.Kind != "general" {
			return aiworkbench.ErrInvalid
		}
		var now time.Time
		if err = tx.Raw("SELECT clock_timestamp()").Scan(&now).Error; err != nil {
			return err
		}
		userID, assistantID := uuid.NewString(), uuid.NewString()
		user := messageRow{ID: userID, OrganizationID: scope.OrganizationID, OwnerUserID: scope.ActorID, ConversationID: conversationID, Sequence: conversation.NextSequence, AuthorKind: string(aiworkbench.AuthorUser), Content: "查看店铺概况", CreatedAt: now.UTC()}
		assistant := messageRow{ID: assistantID, OrganizationID: scope.OrganizationID, OwnerUserID: scope.ActorID, ConversationID: conversationID, Sequence: conversation.NextSequence + 1, AuthorKind: string(aiworkbench.AuthorAssistant), Kind: "STORE_OVERVIEW_REF", Content: "店铺概况", StoreQueryStatus: &status, CreatedAt: now.UTC()}
		if err = tx.Create(&user).Error; err != nil {
			return err
		}
		if err = tx.Create(&assistant).Error; err != nil {
			return err
		}
		if err = tx.Model(&conversationRow{}).Where("id=? AND organization_id=? AND owner_user_id=?", conversationID, scope.OrganizationID, scope.ActorID).Updates(map[string]any{"next_sequence": conversation.NextSequence + 2, "updated_at": now}).Error; err != nil {
			return err
		}
		kind := "STORE_OVERVIEW"
		sequence := conversation.NextSequence
		row := commandRow{OrganizationID: scope.OrganizationID, ActorID: scope.ActorID, IdempotencyKey: key, Operation: explicitOverviewOperation, RequestFingerprint: fingerprint, ConversationID: conversationID, State: string(aiworkbench.PlanningComplete), UserMessageID: &userID, SourceSequence: &sequence, AssistantMessageID: &assistantID, MemberID: &member, RouteResult: &kind, CreatedAt: now.UTC(), CommittedAt: &now}
		if err = tx.Create(&row).Error; err != nil {
			return err
		}
		result = routeCommand(row)
		return nil
	})
	if err != nil {
		return aiworkbench.StoreRouteCommand{}, false, err
	}
	return result, replay, nil
}

func routeCommand(row commandRow) aiworkbench.StoreRouteCommand {
	out := aiworkbench.StoreRouteCommand{Scope: aiworkbench.Scope{OrganizationID: row.OrganizationID, ActorID: row.ActorID}, ConversationID: row.ConversationID, IdempotencyKey: row.IdempotencyKey, Fingerprint: row.RequestFingerprint, State: row.State}
	out.ModelProfile = append([]byte(nil), row.PlannerModelProfile...)
	if row.UserMessageID != nil {
		out.UserMessageID = *row.UserMessageID
	}
	if row.MemberID != nil {
		out.MemberID = *row.MemberID
	}
	if row.PlannerInvocationID != nil {
		out.InvocationID = *row.PlannerInvocationID
	}
	if row.PlannerInputHash != nil {
		out.InputHash = *row.PlannerInputHash
	}
	if row.PlannerDeadline != nil {
		out.ModelDeadline = *row.PlannerDeadline
	}
	if row.RouteCommitDeadline != nil {
		out.CommitDeadline = *row.RouteCommitDeadline
	}
	return out
}

// VerifyStoreRouteDispatch is called after claim and again at network
// handoff. Both calls serialize with no-claim recovery on the same row.
func (s *Store) VerifyStoreRouteDispatch(ctx context.Context, expected aiworkbench.StoreRouteCommand) error {
	if s == nil || s.db == nil || ctx == nil || !validScope(expected.Scope) || !validKey(expected.IdempotencyKey) {
		return aiworkbench.ErrInvalid
	}
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var row commandRow
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("organization_id=? AND actor_id=? AND idempotency_key=?", expected.Scope.OrganizationID, expected.Scope.ActorID, expected.IdempotencyKey).Take(&row).Error; err != nil {
			return missing(err)
		}
		current := routeCommand(row)
		if row.Operation != routeOperation || row.State != string(aiworkbench.PlanningReadyToDispatch) || current.Fingerprint != expected.Fingerprint || current.ConversationID != expected.ConversationID || current.InvocationID != expected.InvocationID || current.MemberID != expected.MemberID || current.InputHash != expected.InputHash || !current.ModelDeadline.Equal(expected.ModelDeadline) {
			return aiworkbench.ErrUnavailable
		}
		var now time.Time
		if err := tx.Raw("SELECT clock_timestamp()").Scan(&now).Error; err != nil {
			return err
		}
		if now.After(current.ModelDeadline) {
			return aiworkbench.ErrUnavailable
		}
		return nil
	})
}

// FinalizeStoreRouteAbsent checks the unfiltered owner primary while holding
// the command lock. The bounded read must never lock or write to that owner.
func (s *Store) FinalizeStoreRouteAbsent(ctx context.Context, scope aiworkbench.Scope, key string, exists func(context.Context, string) (bool, error)) (aiworkbench.StoreRouteCommand, bool, error) {
	var result aiworkbench.StoreRouteCommand
	var replay bool
	if s == nil || s.db == nil || ctx == nil || !validScope(scope) || !validKey(key) || exists == nil {
		return result, false, aiworkbench.ErrInvalid
	}
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var row commandRow
		if e := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("organization_id=? AND actor_id=? AND idempotency_key=?", scope.OrganizationID, scope.ActorID, key).Take(&row).Error; e != nil {
			return missing(e)
		}
		if row.Operation != routeOperation {
			return aiworkbench.ErrIdempotencyConflict
		}
		if row.State != string(aiworkbench.PlanningReadyToDispatch) {
			result, replay = routeCommand(row), true
			return nil
		}
		command := routeCommand(row)
		if command.InvocationID == "" || command.ModelDeadline.IsZero() {
			return aiworkbench.ErrUnavailable
		}
		var now time.Time
		if e := tx.Raw("SELECT clock_timestamp()").Scan(&now).Error; e != nil {
			return e
		}
		if !now.After(command.ModelDeadline) {
			return aiworkbench.ErrUnavailable
		}
		readCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
		defer cancel()
		found, e := exists(readCtx, command.InvocationID)
		if e != nil || found {
			return aiworkbench.ErrUnavailable
		}
		conversation, e := s.lookupConversation(tx, scope, row.ConversationID, true)
		if e != nil {
			return e
		}
		message := messageRow{ID: uuid.NewString(), OrganizationID: scope.OrganizationID, OwnerUserID: scope.ActorID, ConversationID: row.ConversationID, Sequence: conversation.NextSequence, AuthorKind: string(aiworkbench.AuthorAssistant), Content: "识别尚未发送，请检查配置后重试。", CreatedAt: now.UTC()}
		if e = tx.Create(&message).Error; e != nil {
			return e
		}
		if e = tx.Model(&conversationRow{}).Where("id=? AND organization_id=? AND owner_user_id=?", row.ConversationID, scope.OrganizationID, scope.ActorID).Updates(map[string]any{"next_sequence": conversation.NextSequence + 1, "updated_at": now}).Error; e != nil {
			return e
		}
		if e = tx.Model(&commandRow{}).Where("organization_id=? AND actor_id=? AND idempotency_key=?", scope.OrganizationID, scope.ActorID, key).Updates(map[string]any{"state": string(aiworkbench.PlanningFailedBeforeDispatch), "assistant_message_id": message.ID, "committed_at": now}).Error; e != nil {
			return e
		}
		row.State = string(aiworkbench.PlanningFailedBeforeDispatch)
		row.AssistantMessageID = &message.ID
		row.CommittedAt = &now
		result = routeCommand(row)
		return nil
	})
	if err != nil {
		return aiworkbench.StoreRouteCommand{}, false, err
	}
	return result, replay, nil
}

// ReplayStoreRoute never performs model admission or dispatch. It checks the
// original member and exact user request before returning the frozen input.
func (s *Store) ReplayStoreRoute(ctx context.Context, scope aiworkbench.Scope, conversationID, key, member, content string) (aiworkbench.StoreRouteCommand, bool, error) {
	var result aiworkbench.StoreRouteCommand
	if s == nil || s.db == nil || ctx == nil || !validScope(scope) || !validKey(conversationID) || !validKey(key) || !validName(member) || content == "" || len(content) > 8192 || !utf8.ValidString(content) {
		return result, false, aiworkbench.ErrInvalid
	}
	if err := s.ConversationMember(ctx, scope, conversationID, member); err != nil {
		return result, false, err
	}
	row, found, err := s.lookupCommand(s.db.WithContext(ctx), scope, key)
	if err != nil || !found {
		return result, false, err
	}
	fingerprint := digest(struct{ Operation, ConversationID, Content string }{routeOperation, conversationID, content})
	if row.Operation != routeOperation || row.ConversationID != conversationID || row.RequestFingerprint != fingerprint || row.MemberID == nil || *row.MemberID != member || row.UserMessageID == nil {
		return result, false, aiworkbench.ErrIdempotencyConflict
	}
	var message messageRow
	if err = s.db.WithContext(ctx).Where("id = ? AND organization_id = ? AND owner_user_id = ? AND conversation_id = ?", *row.UserMessageID, scope.OrganizationID, scope.ActorID, conversationID).Take(&message).Error; err != nil {
		return result, false, missing(err)
	}
	if message.AuthorKind != string(aiworkbench.AuthorUser) || message.Content != content {
		return result, false, aiworkbench.ErrUnavailable
	}
	result = routeCommand(row)
	result.Content = message.Content
	return result, true, nil
}

func (s *Store) BeginStoreRoute(ctx context.Context, scope aiworkbench.Scope, conversationID, key string, request aiworkbench.StoreRouteRequest) (aiworkbench.StoreRouteCommand, bool, error) {
	var result aiworkbench.StoreRouteCommand
	var replay bool
	if s == nil || s.db == nil || ctx == nil || !validScope(scope) || !validKey(conversationID) || !validKey(key) || !validName(request.MemberID) || len(request.Content) == 0 || len(request.Content) > 8192 || !utf8.ValidString(request.Content) {
		return result, false, aiworkbench.ErrInvalid
	}
	fingerprint := digest(struct{ Operation, ConversationID, Content string }{routeOperation, conversationID, request.Content})
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := s.lockKey(tx, scope, key); err != nil {
			return err
		}
		// The same account may leave and later rejoin an organization under a
		// different membership. A scoped conversation belongs to its original
		// membership even when the actor identifier is unchanged.
		var members int64
		if err := tx.Model(&commandRow{}).Where("organization_id = ? AND actor_id = ? AND conversation_id = ? AND operation = ? AND member_id = ?", scope.OrganizationID, scope.ActorID, conversationID, createOperation, request.MemberID).Count(&members).Error; err != nil {
			return err
		}
		if members != 1 {
			return aiworkbench.ErrNotFound
		}
		existing, found, err := s.lookupCommand(tx, scope, key)
		if err != nil {
			return err
		}
		if found {
			if existing.Operation != routeOperation || existing.ConversationID != conversationID || existing.RequestFingerprint != fingerprint || existing.MemberID == nil || *existing.MemberID != request.MemberID {
				return aiworkbench.ErrIdempotencyConflict
			}
			result, replay = routeCommand(existing), true
			return nil
		}
		conversation, err := s.lookupConversation(tx, scope, conversationID, true)
		if err != nil {
			return err
		}
		if conversation.Lifecycle != "ACTIVE" {
			return aiworkbench.ErrConversationArchived
		}
		if conversation.BusinessScope == nil || conversation.BusinessScope.Kind != "general" {
			return aiworkbench.ErrInvalid
		}
		invocation := digest(struct{ OrganizationID, ActorID, ConversationID, Key, Fingerprint string }{scope.OrganizationID, scope.ActorID, conversationID, key, fingerprint})
		if request.Unavailable && (request.Prepare != nil || request.InputHash != "" || len(request.ModelProfile) != 0 || request.ModelTimeout != 0) {
			return aiworkbench.ErrInvalid
		}
		if request.Prepare != nil {
			prepared, e := request.Prepare(invocation)
			if e != nil {
				return e
			}
			request.InputHash, request.ModelProfile, request.ModelTimeout = prepared.InputHash, prepared.ModelProfile, prepared.ModelTimeout
		}
		if !request.Unavailable && (!validDigest(request.InputHash) || !json.Valid(request.ModelProfile) || len(request.ModelProfile) > 8192 || request.ModelTimeout <= 0 || request.ModelTimeout > 5*time.Minute) {
			return aiworkbench.ErrInvalid
		}
		var now time.Time
		if err = tx.Raw("SELECT clock_timestamp()").Scan(&now).Error; err != nil {
			return err
		}
		now = now.UTC().Truncate(time.Microsecond)
		modelDeadline := now.Add(request.ModelTimeout)
		commitDeadline := modelDeadline.Add(30 * time.Second)
		messageID := uuid.NewString()
		message := messageRow{ID: messageID, OrganizationID: scope.OrganizationID, OwnerUserID: scope.ActorID, ConversationID: conversationID, Sequence: conversation.NextSequence, AuthorKind: string(aiworkbench.AuthorUser), Content: request.Content, CreatedAt: now}
		if err = tx.Create(&message).Error; err != nil {
			return err
		}
		sequence := conversation.NextSequence
		conversation.NextSequence++
		var assistantID *string
		if request.Unavailable {
			id := uuid.NewString()
			assistant := messageRow{ID: id, OrganizationID: scope.OrganizationID, OwnerUserID: scope.ActorID, ConversationID: conversationID, Sequence: conversation.NextSequence, AuthorKind: string(aiworkbench.AuthorAssistant), Content: "识别尚未发送，请检查配置后重试。", CreatedAt: now}
			if err = tx.Create(&assistant).Error; err != nil {
				return err
			}
			assistantID = &id
			conversation.NextSequence++
		}
		if conversation.Title == "" && sequence == 1 {
			conversation.Title = initialTitle(request.Content)
		}
		if err = tx.Model(&conversationRow{}).Where("id=? AND organization_id=? AND owner_user_id=?", conversationID, scope.OrganizationID, scope.ActorID).Updates(map[string]any{"title": conversation.Title, "next_sequence": conversation.NextSequence, "updated_at": now}).Error; err != nil {
			return err
		}
		row := commandRow{OrganizationID: scope.OrganizationID, ActorID: scope.ActorID, IdempotencyKey: key, Operation: routeOperation, RequestFingerprint: fingerprint, ConversationID: conversationID, State: string(aiworkbench.PlanningReadyToDispatch), UserMessageID: &messageID, SourceSequence: &sequence, PlannerInvocationID: &invocation, MemberID: &request.MemberID, PlannerInputHash: &request.InputHash, WorkScopeJSON: json.RawMessage(`{}`), PlannerModelProfile: append(json.RawMessage(nil), request.ModelProfile...), PlannerStartedAt: &now, PlannerDeadline: &modelDeadline, RouteCommitDeadline: &commitDeadline, CreatedAt: now}
		if request.Unavailable {
			row.State = string(aiworkbench.PlanningFailedBeforeDispatch)
			row.PlannerInputHash = nil
			row.PlannerModelProfile = nil
			row.PlannerStartedAt = nil
			row.PlannerDeadline = nil
			row.RouteCommitDeadline = nil
			row.AssistantMessageID = assistantID
			row.CommittedAt = &now
		}
		if err = tx.Create(&row).Error; err != nil {
			return err
		}
		result = routeCommand(row)
		return nil
	})
	if err != nil {
		return aiworkbench.StoreRouteCommand{}, false, err
	}
	return result, replay, nil
}

func (s *Store) CompleteStoreRoute(ctx context.Context, scope aiworkbench.Scope, key string, decision aiworkbench.StoreRoute) (aiworkbench.StoreRouteCommand, bool, error) {
	var result aiworkbench.StoreRouteCommand
	var replay bool
	raw, err := json.Marshal(decision)
	if err != nil {
		return result, false, aiworkbench.ErrInvalid
	}
	if _, err = aiworkbench.ParseStoreRoute(raw); err != nil {
		return result, false, err
	}
	if s == nil || s.db == nil || ctx == nil || !validScope(scope) || !validKey(key) {
		return result, false, aiworkbench.ErrInvalid
	}
	terminal := digest(decision)
	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var row commandRow
		if e := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("organization_id=? AND actor_id=? AND idempotency_key=?", scope.OrganizationID, scope.ActorID, key).Take(&row).Error; e != nil {
			return missing(e)
		}
		if row.Operation != routeOperation {
			return aiworkbench.ErrIdempotencyConflict
		}
		if row.State == string(aiworkbench.PlanningComplete) {
			if row.TerminalDigest == nil || *row.TerminalDigest != terminal {
				return aiworkbench.ErrIdempotencyConflict
			}
			result, replay = routeCommand(row), true
			return nil
		}
		conversation, e := s.lookupConversation(tx, scope, row.ConversationID, true)
		if e != nil {
			return e
		}
		// Both row locks can wait. Read the database clock only after acquiring
		// the conversation lock so a late T1 cannot cross the commit fence.
		var now time.Time
		if e := tx.Raw("SELECT clock_timestamp()").Scan(&now).Error; e != nil {
			return e
		}
		if row.State != string(aiworkbench.PlanningReadyToDispatch) || row.RouteCommitDeadline == nil || !aiworkbench.RouteCommitEligible(now, *row.RouteCommitDeadline) {
			return aiworkbench.ErrUnavailable
		}
		message := messageRow{ID: uuid.NewString(), OrganizationID: scope.OrganizationID, OwnerUserID: scope.ActorID, ConversationID: row.ConversationID, Sequence: conversation.NextSequence, AuthorKind: string(aiworkbench.AuthorAssistant), Content: "请补充需求", CreatedAt: now.UTC()}
		if decision.Kind == "STORE_OVERVIEW" {
			message.Kind = "STORE_OVERVIEW_REF"
			message.Content = "店铺概况"
			message.StoreQueryStatus = &decision.Status
		} else if decision.Kind == "PRODUCT_TASK" {
			message.Content = "请选择商品，再继续商品任务。"
		}
		if e = tx.Create(&message).Error; e != nil {
			return e
		}
		if e = tx.Model(&conversationRow{}).Where("id=? AND organization_id=? AND owner_user_id=?", row.ConversationID, scope.OrganizationID, scope.ActorID).Updates(map[string]any{"next_sequence": conversation.NextSequence + 1, "updated_at": now}).Error; e != nil {
			return e
		}
		if e = tx.Model(&commandRow{}).Where("organization_id=? AND actor_id=? AND idempotency_key=?", scope.OrganizationID, scope.ActorID, key).Updates(map[string]any{"state": string(aiworkbench.PlanningComplete), "route_result": decision.Kind, "assistant_message_id": message.ID, "terminal_digest": terminal, "committed_at": now}).Error; e != nil {
			return e
		}
		row.State = string(aiworkbench.PlanningComplete)
		row.AssistantMessageID = &message.ID
		row.RouteResult = &decision.Kind
		row.TerminalDigest = &terminal
		row.CommittedAt = &now
		result = routeCommand(row)
		return nil
	})
	if err != nil {
		return aiworkbench.StoreRouteCommand{}, false, err
	}
	return result, replay, nil
}

// FinalizeStoreRouteFailure accepts only a trusted owner proof of either a
// definite no-dispatch or a settled invalid output. UNKNOWN remains pending.
func (s *Store) FinalizeStoreRouteFailure(ctx context.Context, scope aiworkbench.Scope, key string, verify func(context.Context, aiworkbench.StoreRouteCommand) (aiworkbench.PlanningState, error)) (aiworkbench.StoreRouteCommand, bool, error) {
	var result aiworkbench.StoreRouteCommand
	if s == nil || s.db == nil || ctx == nil || !validScope(scope) || !validKey(key) || verify == nil {
		return result, false, aiworkbench.ErrInvalid
	}
	initial, found, err := s.lookupCommand(s.db.WithContext(ctx), scope, key)
	if err != nil {
		return result, false, err
	}
	if !found || initial.Operation != routeOperation {
		return result, false, aiworkbench.ErrNotFound
	}
	frozen := routeCommand(initial)
	terminal := func(state string) bool {
		return state == string(aiworkbench.PlanningComplete) || state == "ROUTING_RESULT_LOST" || state == string(aiworkbench.PlanningFailedBeforeDispatch) || state == string(aiworkbench.PlanningInvalidOutput)
	}
	if terminal(initial.State) {
		return frozen, true, nil
	}
	state, err := verify(ctx, frozen)
	if err != nil {
		return result, false, err
	}
	if state != aiworkbench.PlanningFailedBeforeDispatch && state != aiworkbench.PlanningInvalidOutput {
		return result, false, aiworkbench.ErrUnavailable
	}
	var replay bool
	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var row commandRow
		if e := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("organization_id=? AND actor_id=? AND idempotency_key=?", scope.OrganizationID, scope.ActorID, key).Take(&row).Error; e != nil {
			return missing(e)
		}
		current := routeCommand(row)
		if row.Operation != routeOperation || current.Fingerprint != frozen.Fingerprint || current.InvocationID != frozen.InvocationID || current.InputHash != frozen.InputHash || current.MemberID != frozen.MemberID || current.ConversationID != frozen.ConversationID {
			return aiworkbench.ErrIdempotencyConflict
		}
		if terminal(row.State) {
			result, replay = current, true
			return nil
		}
		if row.State != string(aiworkbench.PlanningReadyToDispatch) {
			return aiworkbench.ErrUnavailable
		}
		conversation, e := s.lookupConversation(tx, scope, row.ConversationID, true)
		if e != nil {
			return e
		}
		var now time.Time
		if e = tx.Raw("SELECT clock_timestamp()").Scan(&now).Error; e != nil {
			return e
		}
		content := "识别尚未发送，请检查配置后重试。"
		if state == aiworkbench.PlanningInvalidOutput {
			content = "识别结果格式无效，请重新描述需求或查看店铺概况。"
		}
		message := messageRow{ID: uuid.NewString(), OrganizationID: scope.OrganizationID, OwnerUserID: scope.ActorID, ConversationID: row.ConversationID, Sequence: conversation.NextSequence, AuthorKind: string(aiworkbench.AuthorAssistant), Content: content, CreatedAt: now.UTC()}
		if e = tx.Create(&message).Error; e != nil {
			return e
		}
		if e = tx.Model(&conversationRow{}).Where("id=? AND organization_id=? AND owner_user_id=?", row.ConversationID, scope.OrganizationID, scope.ActorID).Updates(map[string]any{"next_sequence": conversation.NextSequence + 1, "updated_at": now}).Error; e != nil {
			return e
		}
		if e = tx.Model(&commandRow{}).Where("organization_id=? AND actor_id=? AND idempotency_key=?", scope.OrganizationID, scope.ActorID, key).Updates(map[string]any{"state": string(state), "assistant_message_id": message.ID, "committed_at": now}).Error; e != nil {
			return e
		}
		row.State = string(state)
		row.AssistantMessageID = &message.ID
		row.CommittedAt = &now
		result = routeCommand(row)
		return nil
	})
	if err != nil {
		return aiworkbench.StoreRouteCommand{}, false, err
	}
	return result, replay, nil
}

// MarkStoreRouteLost requires a trusted owner-supplied verifier. It must check
// the original invocation AND its committed resource settlement; a provider
// success or an invocation row alone is not settlement proof.
func (s *Store) MarkStoreRouteLost(ctx context.Context, scope aiworkbench.Scope, key string, verify func(context.Context, aiworkbench.StoreRouteCommand) error) (aiworkbench.StoreRouteCommand, bool, error) {
	var result aiworkbench.StoreRouteCommand
	if s == nil || s.db == nil || ctx == nil || !validScope(scope) || !validKey(key) || verify == nil {
		return result, false, aiworkbench.ErrInvalid
	}
	initial, found, err := s.lookupCommand(s.db.WithContext(ctx), scope, key)
	if err != nil {
		return result, false, err
	}
	if !found || initial.Operation != routeOperation {
		return result, false, aiworkbench.ErrNotFound
	}
	if initial.State == string(aiworkbench.PlanningComplete) || initial.State == "ROUTING_RESULT_LOST" {
		return routeCommand(initial), true, nil
	}
	frozen := routeCommand(initial)
	if err = verify(ctx, frozen); err != nil {
		return result, false, err
	}
	var replay bool
	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var row commandRow
		if e := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("organization_id=? AND actor_id=? AND idempotency_key=?", scope.OrganizationID, scope.ActorID, key).Take(&row).Error; e != nil {
			return missing(e)
		}
		current := routeCommand(row)
		if row.Operation != routeOperation || current.Fingerprint != frozen.Fingerprint || current.InvocationID != frozen.InvocationID || current.InputHash != frozen.InputHash || current.MemberID != frozen.MemberID || current.ConversationID != frozen.ConversationID || !current.CommitDeadline.Equal(frozen.CommitDeadline) {
			return aiworkbench.ErrIdempotencyConflict
		}
		if row.State == string(aiworkbench.PlanningComplete) || row.State == "ROUTING_RESULT_LOST" {
			result, replay = current, true
			return nil
		}
		var now time.Time
		if e := tx.Raw("SELECT clock_timestamp()").Scan(&now).Error; e != nil {
			return e
		}
		if row.State != string(aiworkbench.PlanningReadyToDispatch) || !aiworkbench.RouteLostEligible(now, current.CommitDeadline, true) {
			return aiworkbench.ErrUnavailable
		}
		conversation, e := s.lookupConversation(tx, scope, row.ConversationID, true)
		if e != nil {
			return e
		}
		message := messageRow{ID: uuid.NewString(), OrganizationID: scope.OrganizationID, OwnerUserID: scope.ActorID, ConversationID: row.ConversationID, Sequence: conversation.NextSequence, AuthorKind: string(aiworkbench.AuthorAssistant), Content: "识别结果未在期限内保存，请选择查看店铺概况。", CreatedAt: now.UTC()}
		if e = tx.Create(&message).Error; e != nil {
			return e
		}
		if e = tx.Model(&conversationRow{}).Where("id=? AND organization_id=? AND owner_user_id=?", row.ConversationID, scope.OrganizationID, scope.ActorID).Updates(map[string]any{"next_sequence": conversation.NextSequence + 1, "updated_at": now}).Error; e != nil {
			return e
		}
		if e = tx.Model(&commandRow{}).Where("organization_id=? AND actor_id=? AND idempotency_key=?", scope.OrganizationID, scope.ActorID, key).Updates(map[string]any{"state": "ROUTING_RESULT_LOST", "assistant_message_id": message.ID, "committed_at": now}).Error; e != nil {
			return e
		}
		row.State = "ROUTING_RESULT_LOST"
		row.AssistantMessageID = &message.ID
		row.CommittedAt = &now
		result = routeCommand(row)
		return nil
	})
	if err != nil {
		return aiworkbench.StoreRouteCommand{}, false, err
	}
	return result, replay, nil
}
