package storagecapacitystore

import (
	"context"
	"errors"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"task-processor/internal/ledger/storagecapacity"
)

type namespaceRow struct {
	Identity  string `gorm:"primaryKey;size:64"`
	Namespace string `gorm:"size:64"`
	Prefix    string `gorm:"size:128"`
}

func (namespaceRow) TableName() string { return "storage_capacity_namespaces" }

// EnsureNamespace records only a new, empty counted prefix. Existing data is
// never scanned/backfilled or treated as zero. Network checks run outside SQL.
func (r *Repository) EnsureNamespace(ctx context.Context, namespace, prefix string, empty func(context.Context) error) error {
	if !storagecapacity.ValidDigest(namespace) || !storagecapacity.ValidID(prefix, 128) || empty == nil {
		return storagecapacity.ErrInvalid
	}
	identity := storagecapacity.Digest(namespace + "\x00" + prefix)
	var row namespaceRow
	e := r.db.WithContext(ctx).First(&row, "identity = ?", identity).Error
	if e == nil {
		if row.Namespace != namespace || row.Prefix != prefix {
			return storagecapacity.ErrConflict
		}
		return nil
	}
	if !errors.Is(e, gorm.ErrRecordNotFound) {
		return e
	}
	if e = empty(ctx); e != nil {
		return e
	}
	return r.db.WithContext(ctx).Clauses(clause.OnConflict{DoNothing: true}).Create(&namespaceRow{Identity: identity, Namespace: namespace, Prefix: prefix}).Error
}
func (r *Repository) RequireNamespace(ctx context.Context, namespace, prefix string) error {
	var row namespaceRow
	e := r.db.WithContext(ctx).First(&row, "identity = ?", storagecapacity.Digest(namespace+"\x00"+prefix)).Error
	if e != nil || row.Namespace != namespace || row.Prefix != prefix {
		return storagecapacity.ErrUnavailable
	}
	return nil
}
