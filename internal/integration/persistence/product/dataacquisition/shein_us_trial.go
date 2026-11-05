package dataacquisitionpersistence

import (
	"context"
	"database/sql"
	"errors"

	"github.com/jackc/pgx/v5/pgconn"

	"task-processor/internal/product/dataacquisition"
)

// InstallSHEINUSTrialSchema is an explicit empty Product installation step for
// the isolated trial. It is independent of the Amazon runtime install path.
func InstallSHEINUSTrialSchema(ctx context.Context, db *sql.DB) error {
	if db == nil {
		return dataacquisition.ErrUnavailable
	}
	_, err := db.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS data_acquisition_shein_us_trials (
 approval_id uuid PRIMARY KEY,
 binding char(64) NOT NULL CHECK(binding ~ '^[0-9a-f]{64}$'),
 claimed_at timestamptz NOT NULL,
 request_id varchar(128) UNIQUE CHECK(request_id ~ '^[A-Za-z0-9_-]{1,128}$'),
 credits integer CHECK(credits IS NULL OR (credits >= 0 AND request_id IS NOT NULL))
)`)
	return err
}

type SHEINUSTrialStore struct{ db *sql.DB }

func NewSHEINUSTrialStore(db *sql.DB) (*SHEINUSTrialStore, error) {
	if db == nil {
		return nil, dataacquisition.ErrUnavailable
	}
	return &SHEINUSTrialStore{db}, nil
}

const sheinUSTrialFields = "approval_id,binding,claimed_at,COALESCE(request_id,''),credits"

func (r *SHEINUSTrialStore) Read(ctx context.Context, a dataacquisition.SHEINUSTrialApproval) (dataacquisition.SHEINUSTrialAttempt, error) {
	row := r.db.QueryRowContext(ctx, "SELECT "+sheinUSTrialFields+" FROM data_acquisition_shein_us_trials WHERE approval_id=$1 AND binding=$2", a.ID, a.Binding())
	attempt, err := readSHEINUSTrial(row)
	if errors.Is(err, dataacquisition.ErrUnknown) {
		return attempt, dataacquisition.ErrUnavailable
	}
	return attempt, err
}

func (r *SHEINUSTrialStore) Claim(ctx context.Context, a dataacquisition.SHEINUSTrialApproval) (dataacquisition.SHEINUSTrialAttempt, bool, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return dataacquisition.SHEINUSTrialAttempt{}, false, dataacquisition.ErrUnknown
	}
	defer tx.Rollback()
	if _, err = tx.ExecContext(ctx, "SET LOCAL synchronous_commit = on"); err != nil {
		return dataacquisition.SHEINUSTrialAttempt{}, false, dataacquisition.ErrUnknown
	}
	row := tx.QueryRowContext(ctx, `INSERT INTO data_acquisition_shein_us_trials(approval_id,binding,claimed_at)
 SELECT $1::uuid,$2::char(64),clock_timestamp() WHERE clock_timestamp() < $3::timestamptz
 ON CONFLICT(approval_id) DO NOTHING RETURNING `+sheinUSTrialFields, a.ID, a.Binding(), a.ExpiresAt)
	attempt, err := readSHEINUSTrial(row)
	if err != nil && !errors.Is(err, dataacquisition.ErrNotFound) {
		return attempt, false, err
	}
	if tx.Commit() != nil {
		return dataacquisition.SHEINUSTrialAttempt{}, false, dataacquisition.ErrUnknown
	}
	if err == nil {
		return attempt, true, nil
	}
	attempt, err = r.Read(ctx, a)
	if errors.Is(err, dataacquisition.ErrNotFound) {
		err = dataacquisition.ErrConflict
	}
	return attempt, false, err
}

func (r *SHEINUSTrialStore) Record(ctx context.Context, u dataacquisition.SHEINUSTrialUsage) (dataacquisition.SHEINUSTrialAttempt, error) {
	row := r.db.QueryRowContext(ctx, `UPDATE data_acquisition_shein_us_trials SET request_id=$3,credits=COALESCE(credits,$4)
 WHERE approval_id=$1 AND binding=$2 AND (request_id IS NULL OR request_id=$3)
 AND (credits IS NULL OR $4::integer IS NULL OR credits=$4) RETURNING `+sheinUSTrialFields,
		u.ApprovalID, u.Binding, u.RequestID, u.Credits)
	attempt, err := readSHEINUSTrial(row)
	if errors.Is(err, dataacquisition.ErrNotFound) {
		err = dataacquisition.ErrConflict
	}
	return attempt, err
}

func readSHEINUSTrial(row *sql.Row) (dataacquisition.SHEINUSTrialAttempt, error) {
	var a dataacquisition.SHEINUSTrialAttempt
	err := row.Scan(&a.ApprovalID, &a.Binding, &a.ClaimedAt, &a.RequestID, &a.Credits)
	if errors.Is(err, sql.ErrNoRows) {
		return a, dataacquisition.ErrNotFound
	}
	var pg *pgconn.PgError
	if errors.As(err, &pg) && pg.Code == "23505" {
		return a, dataacquisition.ErrConflict
	}
	if err != nil {
		return a, dataacquisition.ErrUnknown
	}
	return a, nil
}
