package dataacquisitionpersistence

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"

	"github.com/jackc/pgx/v5/pgconn"

	"task-processor/internal/product/collection"
	"task-processor/internal/product/dataacquisition"
	"task-processor/internal/product/sourcing"
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
	tx, err := r.beginDurableWrite(ctx)
	if err != nil {
		return dataacquisition.SHEINUSTrialAttempt{}, false, err
	}
	defer tx.Rollback()
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
	return r.record(ctx, u, nil)
}

func (r *SHEINUSTrialStore) RecordPage(ctx context.Context, a dataacquisition.SHEINUSTrialApproval, u dataacquisition.SHEINUSTrialUsage, cards []dataacquisition.SHEINUSTrialCard) (dataacquisition.SHEINUSTrialAttempt, error) {
	if u.ApprovalID != a.ID || u.Binding != a.Binding() || u.Credits == nil || *u.Credits < 0 || *u.Credits > dataacquisition.SHEINUSTrialCreditLimit {
		return dataacquisition.SHEINUSTrialAttempt{}, dataacquisition.ErrConflict
	}
	captures := dataacquisition.PrepareSHEINUSTrialCards(a, cards)
	if len(captures) == 0 {
		return r.Record(ctx, u)
	}
	return r.record(ctx, u, func(tx *sql.Tx) error {
		return insertSHEINUSTrialCaptures(ctx, tx, a, captures)
	})
}

func (r *SHEINUSTrialStore) record(ctx context.Context, u dataacquisition.SHEINUSTrialUsage, prepare func(*sql.Tx) error) (dataacquisition.SHEINUSTrialAttempt, error) {
	tx, err := r.beginDurableWrite(ctx)
	if err != nil {
		return dataacquisition.SHEINUSTrialAttempt{}, err
	}
	defer tx.Rollback()
	row := tx.QueryRowContext(ctx, `UPDATE data_acquisition_shein_us_trials SET request_id=$3,credits=COALESCE(credits,$4)
 WHERE approval_id=$1 AND binding=$2 AND (request_id=$3 OR (request_id IS NULL AND NOT $5))
 AND (credits IS NULL OR $4::integer IS NULL OR credits=$4) RETURNING `+sheinUSTrialFields,
		u.ApprovalID, u.Binding, u.RequestID, u.Credits, prepare != nil)
	attempt, err := readSHEINUSTrial(row)
	if errors.Is(err, dataacquisition.ErrNotFound) {
		err = dataacquisition.ErrConflict
	}
	if err != nil {
		return dataacquisition.SHEINUSTrialAttempt{}, err
	}
	if prepare != nil && prepare(tx) != nil {
		return dataacquisition.SHEINUSTrialAttempt{}, dataacquisition.ErrUnknown
	}
	if tx.Commit() != nil {
		return dataacquisition.SHEINUSTrialAttempt{}, dataacquisition.ErrUnknown
	}
	return attempt, nil
}

func insertSHEINUSTrialCaptures(ctx context.Context, tx *sql.Tx, a dataacquisition.SHEINUSTrialApproval, captures []dataacquisition.SHEINUSTrialPrepared) error {
	q := a.PageQuery()
	raw, err := json.Marshal(q)
	if err != nil {
		return err
	}
	scope := a.Scope
	_, err = tx.ExecContext(ctx, `INSERT INTO data_acquisition_jobs
 (organization_id,actor_id,member_id,id,source_kind,trial_binding,command_key,input_hash,query_json,funding,state,quota_reserved,day_window,month_window,created_at,deadline)
 VALUES ($1,$2,$3,$4,'shein_us_data',$5,$4,$6,$7::jsonb,$8,'RUNNING',0,CURRENT_DATE,CURRENT_DATE,now(),now()+interval '30 minutes')`,
		scope.OrganizationID, scope.ActorID, scope.MemberID, a.ID, a.Binding(), collection.Digest(q), string(raw), a.Funding)
	if err != nil {
		return err
	}
	for _, c := range captures {
		_, err = tx.ExecContext(ctx, `INSERT INTO data_acquisition_items
 (organization_id,actor_id,job_id,id,source_kind,native_goods_id,source_position,state,intent_json,evidence_json,evidence_checksum,created_at)
 VALUES ($1,$2,$3,$4,'shein_us_data',$5,$6,'PREPARED_EVIDENCE','{}',$7,$8,now())`,
			scope.OrganizationID, scope.ActorID, c.JobID, c.ItemID, c.NativeGoodsID, c.Position, c.Raw, sourcing.RawSnapshotChecksum(string(c.Raw)))
		if err != nil {
			return err
		}
	}
	return nil
}

func (r *SHEINUSTrialStore) beginDurableWrite(ctx context.Context) (*sql.Tx, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, dataacquisition.ErrUnknown
	}
	if _, err = tx.ExecContext(ctx, "SET LOCAL synchronous_commit = on"); err != nil {
		_ = tx.Rollback()
		return nil, dataacquisition.ErrUnknown
	}
	return tx, nil
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
