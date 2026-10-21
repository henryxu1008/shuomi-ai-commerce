package routeconfigstore

import (
	"context"
	"database/sql"
	"errors"
	"net/url"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func TestSharedWriterConnectionsAcrossOwnerDatabases(t *testing.T) {
	db, _ := fixture(t)
	ctx := context.Background()
	const role = "model_route_config_writer"
	require.NoError(t, db.Exec("CREATE ROLE "+role+" LOGIN PASSWORD 'fixture-capacity' CONNECTION LIMIT 2").Error)
	t.Cleanup(func() {
		require.NoError(t, db.Exec("DROP OWNED BY "+role).Error)
		require.NoError(t, db.Exec("DROP ROLE "+role).Error)
	})
	require.NoError(t, db.Exec(GrantSQL(role, true)).Error)
	second := "capacity_" + strings.ReplaceAll(uuid.NewString(), "-", "")
	require.NoError(t, db.Exec("CREATE DATABASE "+second).Error)
	t.Cleanup(func() { require.NoError(t, db.Exec("DROP DATABASE "+second+" WITH (FORCE)").Error) })
	u, e := url.Parse(db.Dialector.(*postgres.Dialector).Config.DSN)
	require.NoError(t, e)
	u.User = url.UserPassword(role, "fixture-capacity")
	text, e := sql.Open("pgx", u.String())
	require.NoError(t, e)
	text.SetMaxOpenConns(2)
	t.Cleanup(func() { require.NoError(t, text.Close()) })
	u.Path = "/" + second
	image, e := sql.Open("pgx", u.String())
	require.NoError(t, e)
	image.SetMaxOpenConns(2)
	t.Cleanup(func() { require.NoError(t, image.Close()) })
	var held []*sql.Conn
	t.Cleanup(func() {
		for _, c := range held {
			_ = c.Close()
		}
	})
	for range 2 {
		c, err := text.Conn(ctx)
		require.NoError(t, err)
		held = append(held, c)
	}
	_, e = image.Conn(ctx)
	var pgErr *pgconn.PgError
	require.True(t, errors.As(e, &pgErr))
	require.Equal(t, "53300", pgErr.Code, "the same role limit is shared across databases")
	require.Error(t, VerifyWriterConnections(ctx, db, 4))
	require.NoError(t, ConfigureWriterConnections(ctx, db, 4))
	require.NoError(t, VerifyWriterConnections(ctx, db, 4))
	for range 2 {
		c, err := image.Conn(ctx)
		require.NoError(t, err)
		held = append(held, c)
	}
	var count int
	require.NoError(t, db.Raw("SELECT count(*) FROM pg_stat_activity WHERE usename=?", role).Scan(&count).Error)
	require.Equal(t, 4, count)
	for _, c := range held {
		require.NoError(t, c.Close())
	}
	held = nil
	writer, e := gorm.Open(postgres.New(postgres.Config{Conn: text}), &gorm.Config{DisableAutomaticPing: true, Logger: logger.Default.LogMode(logger.Silent)})
	require.NoError(t, e)
	require.NoError(t, VerifyWriter(ctx, writer, true), "raising capacity retains exact writer privileges")
	var secret string
	require.Error(t, writer.Raw("SELECT api_key FROM public.ai_client_credentials LIMIT 1").Scan(&secret).Error)
	// The ordinary schema install/omitted capacity argument must not lower 4.
	require.NoError(t, InstallSchema(ctx, db))
	require.NoError(t, ConfigureWriterConnections(ctx, db, 0))
	require.NoError(t, VerifyWriterConnections(ctx, db, 4))
	require.Error(t, ConfigureWriterConnections(ctx, db, 2), "cannot lower an explicitly provisioned finite budget")
	for _, limit := range []int{-1, 1, 3, 5} {
		require.Error(t, ConfigureWriterConnections(ctx, db, limit))
	}
	var retained int
	require.NoError(t, db.Raw("SELECT rolconnlimit FROM pg_roles WHERE rolname=?", role).Scan(&retained).Error)
	require.Equal(t, 4, retained)
}

func TestWriterCapacityInspectionAndInstallerFailClosed(t *testing.T) {
	db, _ := fixture(t)
	ctx := context.Background()
	const role = "model_route_config_writer"
	require.Error(t, VerifyWriterConnections(ctx, db, 2), "missing role")
	require.Error(t, ConfigureWriterConnections(ctx, db, 4), "installer never creates a role")
	require.NoError(t, db.Exec("CREATE ROLE "+role+" LOGIN CONNECTION LIMIT 2").Error)
	t.Cleanup(func() { require.NoError(t, db.Exec("DROP ROLE "+role).Error) })
	require.NoError(t, VerifyWriterConnections(ctx, db, 2))
	require.Error(t, VerifyWriterConnections(ctx, db, 4))
	errStop := errors.New("rollback fixture")
	err := db.Transaction(func(tx *gorm.DB) error {
		require.NoError(t, ConfigureWriterConnections(ctx, tx, 4))
		return errStop
	})
	require.ErrorIs(t, err, errStop)
	require.Error(t, VerifyWriterConnections(ctx, db, 4), "role alteration participates in the install transaction")
	require.NoError(t, db.Exec("ALTER ROLE "+role+" CONNECTION LIMIT -1").Error)
	require.Error(t, VerifyWriterConnections(ctx, db, 2), "unlimited capacity is not admitted")
	require.NoError(t, ConfigureWriterConnections(ctx, db, 4))
	require.NoError(t, VerifyWriterConnections(ctx, db, 4))
	require.NoError(t, db.Exec("ALTER ROLE "+role+" CREATEDB").Error)
	require.Error(t, ConfigureWriterConnections(ctx, db, 4), "privileged role is not admitted")
	require.Error(t, VerifyWriterConnections(ctx, db, 4))
	require.NoError(t, db.Exec("ALTER ROLE "+role+" NOCREATEDB NOLOGIN").Error)
	require.Error(t, ConfigureWriterConnections(ctx, db, 4))
	require.Error(t, VerifyWriterConnections(ctx, db, 4))
	cancelled, cancel := context.WithCancel(ctx)
	cancel()
	require.Error(t, VerifyWriterConnections(cancelled, db, 4))
}
