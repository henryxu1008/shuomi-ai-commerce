package dataacquisitionpersistence

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"gorm.io/gorm"
	"task-processor/internal/ledger/orgresource"
	"task-processor/internal/product/dataacquisition"
)

func (r *Repository) readSnapshot(ctx context.Context, read func(*gorm.DB) error) error {
	if _, bound := r.db.Statement.ConnPool.(*sql.Tx); bound {
		return read(r.db.WithContext(ctx))
	}
	return r.db.WithContext(ctx).Transaction(read, &sql.TxOptions{Isolation: sql.LevelRepeatableRead})
}

// This is a current read-only state, never a no-charge or recovery proof.
func unavailableTerminal(row jobRow, items []itemRow) error {
	fail := func() error {
		return fmt.Errorf("%w: task %s price_snapshot unavailable outside read-only FAILED state", dataacquisition.ErrUnavailable, row.ID)
	}
	if len(row.PriceJSON) != 0 || row.State != "FAILED" || row.QuotaReserved != 0 {
		return fail()
	}
	for _, item := range items {
		if item.State != "FAILED" || item.TerminalEvidence == "" || item.ReservationID != "" || item.ChargeState != "" || item.ClaimToken != "" || item.LeaseUntil != nil || item.EvidenceJSON != nil || item.SourceJSON != nil || item.SavedAt != nil {
			return fail()
		}
		intent, err := item.intent()
		if err != nil || intent.MemberID != row.MemberID || string(intent.Funding) != row.Funding || intent.ResourceType != orgresource.ResourceDataRow || intent.Quantity != 1 || intent.BusinessScope != "amazon-data:"+row.ID+":"+item.ID {
			return fail()
		}
	}
	return nil
}

// VerifyMissingPriceFacts is used by the same owner at startup and while holding
// maintenance locks. No data mutation or inferred price is involved.
func VerifyMissingPriceFacts(ctx context.Context, db *gorm.DB) error {
	return (&Repository{db: db}).readSnapshot(ctx, func(tx *gorm.DB) error { return verifyUnpriced(ctx, tx, false) })
}

// VerifyPrePriceFacts is only for the fixed A schema under maintenance locks.
func VerifyPrePriceFacts(ctx context.Context, db *gorm.DB) error {
	return verifyUnpriced(ctx, db, true)
}

func verifyUnpriced(ctx context.Context, db *gorm.DB, baseline bool) error {
	var rows []jobRow
	columns, where := jobColumns, " AND price_snapshot IS NULL"
	if baseline {
		columns = strings.Replace(columns, ",price_snapshot,", ",NULL::jsonb AS price_snapshot,", 1)
		where = ""
	}
	if err := db.WithContext(ctx).Raw("SELECT " + columns + " FROM data_acquisition_jobs WHERE source_kind='amazon_data'" + where).Scan(&rows).Error; err != nil {
		return err
	}
	for _, row := range rows {
		job, err := row.identity()
		if err != nil {
			return err
		}
		items, err := itemRows(db.WithContext(ctx), job)
		if err != nil {
			return err
		}
		if err = unavailableTerminal(row, items); err != nil {
			return err
		}
	}
	return nil
}
