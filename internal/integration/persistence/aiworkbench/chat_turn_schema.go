package aiworkbenchpersistence

import (
	"context"
	"gorm.io/gorm"
	"task-processor/internal/aiworkbench"
)

func chatTurnLayout(db *gorm.DB) (int, error) {
	var count int
	err := db.Raw(`SELECT count(*) FROM pg_attribute a JOIN pg_class c ON c.oid=a.attrelid JOIN pg_namespace n ON n.oid=c.relnamespace WHERE n.nspname='ai_workbench' AND a.attnum>0 AND NOT a.attisdropped AND ((c.relname='commands' AND a.attname IN ('turn_input','tool_state','tool_key')) OR (c.relname='messages' AND a.attname='turn_key'))`).Scan(&count).Error
	if err != nil {
		return 0, err
	}
	if count == 0 {
		return 0, nil
	}
	if count != 4 {
		return 0, aiworkbench.ErrUnavailable
	}
	return 1, nil
}

// Feature readiness never creates or alters a schema. Older current schemas
// keep their existing features while the owner explicitly installs this one.
func (s *Store) CanChatTurns(ctx context.Context) bool {
	if s == nil || s.db == nil || ctx == nil {
		return false
	}
	db := s.db.WithContext(ctx)
	version, err := chatTurnLayout(db)
	if err != nil || version != 1 {
		return false
	}
	var columns int
	if db.Raw(`SELECT count(*) FROM pg_attribute WHERE attnum>0 AND NOT attisdropped AND NOT attnotnull AND ((attrelid='ai_workbench.commands'::regclass AND ((attname='turn_input' AND atttypid='jsonb'::regtype) OR (attname='tool_state' AND atttypid='text'::regtype) OR (attname='tool_key' AND atttypid='uuid'::regtype))) OR (attrelid='ai_workbench.messages'::regclass AND attname='turn_key' AND atttypid='uuid'::regtype))`).Scan(&columns).Error != nil || columns != 4 {
		return false
	}
	return verifyChatTurnConstraints(db)
}
