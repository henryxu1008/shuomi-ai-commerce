package aiworkbenchpersistence

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"task-processor/internal/aiworkbench"
)

func decodeTurn(raw []byte) (aiworkbench.ChatTurnEnvelope, error) {
	var envelope aiworkbench.ChatTurnEnvelope
	if len(raw) == 0 || len(raw) > 98304 || json.Unmarshal(raw, &envelope) != nil || envelope.Version != 1 || envelope.Input.Content == "" || len(envelope.Input.Content) > 8192 {
		return envelope, aiworkbench.ErrInvalid
	}
	if envelope.Input.CreatorOnly && (envelope.Input.Skill != nil || envelope.Input.Revision != nil) {
		return envelope, aiworkbench.ErrInvalid
	}
	validRef := func(ref *aiworkbench.ChatSkillReference) bool {
		return ref == nil || ((ref.Version == 1 || ref.Version == 2) && validDigest(ref.Digest))
	}
	if !validRef(envelope.Creator) || !validRef(envelope.Input.Skill) {
		return envelope, aiworkbench.ErrInvalid
	}
	if revision := envelope.Input.Revision; revision != nil {
		if envelope.Input.Skill == nil || envelope.Input.Skill.Version != 2 || !validKey(revision.Root) || !validKey(revision.BaseRevision) || !json.Valid(revision.Snapshot) || len(revision.Snapshot) > 65536 {
			return envelope, aiworkbench.ErrInvalid
		}
	}
	return envelope, nil
}

func turnFingerprint(conversation string, input aiworkbench.ChatTurnInput) string {
	return digest(struct {
		Operation, Conversation string
		Version                 int
		Input                   aiworkbench.ChatTurnInput
	}{routeOperation, conversation, 1, input})
}

func (s *Store) BeginChatTurn(ctx context.Context, scope aiworkbench.Scope, conversation, key string, envelope aiworkbench.ChatTurnEnvelope, request aiworkbench.StoreRouteRequest) (aiworkbench.ChatTurnCommand, bool, error) {
	raw, err := json.Marshal(envelope)
	if err != nil {
		return aiworkbench.ChatTurnCommand{}, false, aiworkbench.ErrInvalid
	}
	if _, err = decodeTurn(raw); err != nil || request.Content != envelope.Input.Content {
		return aiworkbench.ChatTurnCommand{}, false, aiworkbench.ErrInvalid
	}
	request.TurnInput = raw
	_, replay, err := s.BeginStoreRoute(ctx, scope, conversation, key, request)
	if err != nil {
		return aiworkbench.ChatTurnCommand{}, false, err
	}
	result, err := s.ReadChatTurn(ctx, scope, conversation, key, request.MemberID)
	return result, replay, err
}

func (s *Store) ReadChatTurn(ctx context.Context, scope aiworkbench.Scope, conversation, key, member string) (aiworkbench.ChatTurnCommand, error) {
	if s == nil || s.db == nil || ctx == nil || !validScope(scope) || !validKey(conversation) || !validKey(key) || !validName(member) {
		return aiworkbench.ChatTurnCommand{}, aiworkbench.ErrInvalid
	}
	if err := s.ConversationMember(ctx, scope, conversation, member); err != nil {
		return aiworkbench.ChatTurnCommand{}, err
	}
	row, found, err := s.lookupCommand(s.db.WithContext(ctx), scope, key)
	if err != nil {
		return aiworkbench.ChatTurnCommand{}, err
	}
	if !found || row.Operation != routeOperation || row.ConversationID != conversation || row.MemberID == nil || *row.MemberID != member || row.UserMessageID == nil || len(row.TurnInput) == 0 {
		return aiworkbench.ChatTurnCommand{}, aiworkbench.ErrNotFound
	}
	envelope, err := decodeTurn(row.TurnInput)
	if err != nil {
		return aiworkbench.ChatTurnCommand{}, aiworkbench.ErrUnavailable
	}
	var message messageRow
	if err = s.db.WithContext(ctx).Where("id=? AND organization_id=? AND owner_user_id=? AND conversation_id=?", *row.UserMessageID, scope.OrganizationID, scope.ActorID, conversation).Take(&message).Error; err != nil {
		return aiworkbench.ChatTurnCommand{}, missing(err)
	}
	if message.AuthorKind != "USER" || message.Content != envelope.Input.Content || row.RequestFingerprint != turnFingerprint(conversation, envelope.Input) {
		return aiworkbench.ChatTurnCommand{}, aiworkbench.ErrUnavailable
	}
	command := aiworkbench.ChatTurnCommand{Route: routeCommand(row), Envelope: envelope, AdmissionUntil: row.CreatedAt.Add(5 * time.Minute)}
	command.Route.Content = message.Content
	if row.ToolState != nil {
		command.ToolState = *row.ToolState
	}
	if row.ToolKey != nil {
		command.ToolKey = *row.ToolKey
	}
	return command, nil
}

func (s *Store) ReplayChatTurn(ctx context.Context, scope aiworkbench.Scope, conversation, key, member string, input aiworkbench.ChatTurnInput) (aiworkbench.ChatTurnCommand, bool, error) {
	command, err := s.ReadChatTurn(ctx, scope, conversation, key, member)
	if errors.Is(err, aiworkbench.ErrNotFound) {
		if memberErr := s.ConversationMember(ctx, scope, conversation, member); memberErr != nil {
			return command, false, memberErr
		}
		row, found, lookupErr := s.lookupCommand(s.db.WithContext(ctx), scope, key)
		if lookupErr != nil {
			return command, false, lookupErr
		}
		if found && row.MemberID != nil && *row.MemberID == member {
			return command, false, aiworkbench.ErrIdempotencyConflict
		}
		if !found {
			return command, false, nil
		}
		return command, false, err
	}
	if err != nil {
		return command, false, err
	}
	if command.Route.Fingerprint != turnFingerprint(conversation, input) {
		return command, false, aiworkbench.ErrIdempotencyConflict
	}
	return command, true, nil
}

// BindChatTurnAuthoring is the only writer of the parent-child relation. Both
// READY and BOUND commit together; a read or a concurrent candidate cannot mint
// another root. Model I/O stays outside this Workbench transaction.
func (s *Store) BindChatTurnAuthoring(ctx context.Context, expected aiworkbench.ChatTurnCommand, candidate AuthoringDraft) (aiworkbench.ChatTurnCommand, error) {
	return s.bindChatTurnAuthoring(ctx, expected, &candidate)
}

// ResolveChatTurnBinding is a write-side check used only by an admitted send
// or its explicit recovery. History uses ReadChatTurn and never finalizes it.
func (s *Store) ResolveChatTurnBinding(ctx context.Context, expected aiworkbench.ChatTurnCommand) (aiworkbench.ChatTurnCommand, error) {
	return s.bindChatTurnAuthoring(ctx, expected, nil)
}

func (s *Store) bindChatTurnAuthoring(ctx context.Context, expected aiworkbench.ChatTurnCommand, prepared *AuthoringDraft) (aiworkbench.ChatTurnCommand, error) {
	var result aiworkbench.ChatTurnCommand
	scope, key := expected.Route.Scope, expected.Route.IdempotencyKey
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var row commandRow
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("organization_id=? AND actor_id=? AND idempotency_key=?", scope.OrganizationID, scope.ActorID, key).Take(&row).Error; err != nil {
			return missing(err)
		}
		store := &Store{db: tx}
		current, err := store.ReadChatTurn(ctx, scope, expected.Route.ConversationID, key, expected.Route.MemberID)
		if err != nil {
			return err
		}
		if current.Route.Fingerprint != expected.Route.Fingerprint || current.Route.State != "COMPLETE" || current.Route.Result != "CREATE_TITLE_SKILL" {
			return aiworkbench.ErrIdempotencyConflict
		}
		result = current
		if current.ToolState == "BOUND" || current.ToolState == "NOT_STARTED" {
			return nil
		}
		if current.ToolState != "PENDING" || current.ToolKey != "" {
			return aiworkbench.ErrUnavailable
		}
		var now time.Time
		if err = tx.Raw("SELECT clock_timestamp()").Scan(&now).Error; err != nil {
			return err
		}
		if !now.Before(current.AdmissionUntil) {
			if err = tx.Model(&commandRow{}).Where("organization_id=? AND actor_id=? AND idempotency_key=?", scope.OrganizationID, scope.ActorID, key).Update("tool_state", "NOT_STARTED").Error; err != nil {
				return err
			}
			result.ToolState = "NOT_STARTED"
			return nil
		}
		if prepared == nil {
			return nil
		}
		candidate := *prepared
		ref := current.Envelope.Creator
		if ref == nil || candidate.OrganizationID != scope.OrganizationID || candidate.ActorID != scope.ActorID || candidate.ConversationID != current.Route.ConversationID || candidate.MemberID != current.Route.MemberID || candidate.Requirement != current.Envelope.Input.Content || candidate.SkillVersion != ref.Version || candidate.SkillDigest != ref.Digest || candidate.FormatVersion != int(ref.Version) || !validKey(candidate.Key) || !validDigest(candidate.InputHash) || !validDigest(candidate.ProfileDigest) || !validDigest(candidate.InvocationID) || candidate.MaximumPoints <= 0 || candidate.Deadline == nil {
			return aiworkbench.ErrInvalid
		}
		if revision := current.Envelope.Input.Revision; revision != nil {
			if candidate.RootKey == nil || *candidate.RootKey != revision.Root || candidate.BaseRevision == nil || *candidate.BaseRevision != revision.BaseRevision || candidate.InputSnapshot != string(revision.Snapshot) {
				return aiworkbench.ErrInvalid
			}
		} else if candidate.RootKey != nil || candidate.BaseRevision != nil || candidate.InputSnapshot != "" {
			return aiworkbench.ErrInvalid
		}
		if candidate.ExpiresAt.After(current.AdmissionUntil) {
			candidate.ExpiresAt = current.AdmissionUntil
		}
		if candidate.Deadline.After(candidate.ExpiresAt) {
			deadline := candidate.ExpiresAt
			candidate.Deadline = &deadline
		}
		candidate.CreatedAt = now.UTC()
		child, err := store.StartAuthoring(ctx, scope, candidate)
		if err != nil {
			return err
		}
		if err = tx.Model(&commandRow{}).Where("organization_id=? AND actor_id=? AND idempotency_key=?", scope.OrganizationID, scope.ActorID, key).Updates(map[string]any{"tool_state": "BOUND", "tool_key": child.Key}).Error; err != nil {
			return err
		}
		result.ToolState, result.ToolKey = "BOUND", child.Key
		return nil
	})
	return result, err
}
