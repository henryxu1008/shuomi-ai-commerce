package sourcingpersistence

import (
	"context"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/authidentity"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/product/sourcing"
	"gorm.io/gorm"
)

// LockPublication joins exactly the SRC lock namespace used by Publish. Callers
// use READ COMMITTED and read fresh statements after this lock has been granted.
func LockPublication(ctx context.Context, tx *gorm.DB, org, id string) error {
	if tx == nil || !authidentity.IsBoundedIdentifier(org) || !authidentity.IsBoundedIdentifier(id) {
		return sourcing.ErrInvalidSourcePublication
	}
	if _, ok := tx.Statement.ConnPool.(gorm.TxCommitter); !ok {
		return sourcing.ErrSourcePublicationUnavailable
	}
	return advisoryLock(tx.WithContext(ctx), org, id)
}
