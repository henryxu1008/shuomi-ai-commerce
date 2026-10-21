package routeconfigstore

import (
	"context"
	"fmt"

	"gorm.io/gorm"
	d "task-processor/internal/aicapability/routeconfig"
)

const writerRole = "model_route_config_writer"

func writerConnectionLimit(ctx context.Context, db *gorm.DB) (int, error) {
	if ctx == nil || db == nil || db.Dialector == nil || db.Dialector.Name() != "postgres" {
		return 0, d.ErrUnavailable
	}
	var role struct {
		Limit   int
		Allowed bool
	}
	r := db.WithContext(ctx).Raw(`SELECT rolconnlimit AS limit,
 rolcanlogin AND NOT (rolsuper OR rolcreatedb OR rolcreaterole OR rolbypassrls OR rolreplication) AS allowed
 FROM pg_catalog.pg_roles WHERE rolname=?`, writerRole).Scan(&role)
	if r.Error != nil || r.RowsAffected != 1 || !role.Allowed {
		return 0, d.ErrUnavailable
	}
	return role.Limit, nil
}

// ConfigureWriterConnections is an explicit installer operation, never startup.
// Zero leaves the role unchanged, including an existing four-connection budget.
func ConfigureWriterConnections(ctx context.Context, db *gorm.DB, limit int) error {
	if limit == 0 {
		return nil
	}
	if limit != 2 && limit != 4 {
		return d.ErrInvalid
	}
	current, e := writerConnectionLimit(ctx, db)
	if e != nil {
		return e
	}
	if current > limit {
		return d.ErrInvalid
	}
	return db.WithContext(ctx).Exec(fmt.Sprintf("ALTER ROLE %s CONNECTION LIMIT %d", writerRole, limit)).Error
}

// VerifyWriterConnections inspects the cluster-wide role through an existing
// reader before any writer opens; databases do not partition role capacity.
func VerifyWriterConnections(ctx context.Context, reader *gorm.DB, required int) error {
	if required < 1 || required > 4 {
		return d.ErrInvalid
	}
	limit, e := writerConnectionLimit(ctx, reader)
	if e != nil || (limit != 2 && limit != 4) || limit < required {
		return d.ErrUnavailable
	}
	return nil
}
