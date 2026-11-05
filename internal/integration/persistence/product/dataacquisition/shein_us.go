package dataacquisitionpersistence

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	collectionstore "task-processor/internal/integration/persistence/product/collection"
	"task-processor/internal/ledger/orgresource"
	"task-processor/internal/product/collection"
	"task-processor/internal/product/dataacquisition"
	"task-processor/internal/product/sourcing"
)

// SHEINUSPublisher joins the Product transaction. Verify is used on replay,
// where the saved publication must already exist.
type SHEINUSPublisher func(context.Context, *gorm.DB, collection.Scope, orgresource.ResourceFunding, sourcing.PublicationCommand, bool) (collection.Source, error)

type sheinUSRow struct {
	JobID, ItemID, NativeGoodsID, ItemState, ReservationID, InputHash string
	JobState                                                          string
	Canceled                                                          bool
	Deadline                                                          time.Time
	Funding                                                           orgresource.ResourceFunding
	ChargeState                                                       orgresource.ReservationState
	QueryJSON, EvidenceJSON, SourceJSON                               []byte
}

func (row sheinUSRow) capture(scope collection.Scope) (dataacquisition.SHEINUSCapture, error) {
	c := dataacquisition.SHEINUSCapture{Scope: scope, JobID: row.JobID, ItemID: row.ItemID, NativeGoodsID: row.NativeGoodsID, Raw: row.EvidenceJSON}
	if json.Unmarshal(row.QueryJSON, &c.Query) != nil || collection.Digest(c.Query) != row.InputHash {
		return dataacquisition.SHEINUSCapture{}, dataacquisition.ErrUnavailable
	}
	if row.ItemState == "SAVED" && (json.Unmarshal(row.SourceJSON, &c.Source) != nil ||
		c.Source.Kind != collection.SourceSHEINUSData || c.Source.PublicationID != row.ItemID ||
		c.Source.OperationID != row.ItemID || c.Source.Version == 0) {
		return dataacquisition.SHEINUSCapture{}, dataacquisition.ErrUnavailable
	}
	return c, nil
}

const sheinUSSelection = `SELECT i.job_id,i.id AS item_id,i.native_goods_id,i.state AS item_state,i.evidence_json,i.source_json,
 COALESCE(i.reservation_id::text,'') AS reservation_id,i.charge_state,
 j.query_json,j.input_hash,j.state AS job_state,j.canceled,j.deadline,j.funding
 FROM data_acquisition_items i JOIN data_acquisition_jobs j
 ON j.organization_id=i.organization_id AND j.actor_id=i.actor_id AND j.id=i.job_id AND j.source_kind=i.source_kind
 WHERE i.organization_id=? AND i.actor_id=? AND j.member_id=? AND i.id=? AND i.source_kind=?`

func readSHEINUSRow(db *gorm.DB, query string, args ...any) (sheinUSRow, error) {
	var row sheinUSRow
	result := db.Raw(query, args...).Scan(&row)
	if result.Error != nil {
		return row, dataacquisition.ErrUnavailable
	}
	if result.RowsAffected != 1 {
		return row, dataacquisition.ErrNotFound
	}
	return row, nil
}

func (r *Repository) ReadSHEINUSCaptured(ctx context.Context, scope collection.Scope, id string) (dataacquisition.SHEINUSCapture, error) {
	if scope.Validate() != nil || !collection.ValidID(id) {
		return dataacquisition.SHEINUSCapture{}, dataacquisition.ErrInvalid
	}
	row, err := readSHEINUSRow(r.db.WithContext(ctx), sheinUSSelection+" AND i.state='SAVED'",
		scope.OrganizationID, scope.ActorID, scope.MemberID, id, collection.SourceSHEINUSData)
	if err != nil {
		return dataacquisition.SHEINUSCapture{}, err
	}
	return row.capture(scope)
}

// PublishSHEINUS consumes an already prepared source item. The offline slice
// has no runtime command or worker capable of preparing one.
func (r *Repository) PublishSHEINUS(ctx context.Context, p dataacquisition.Principal, ref dataacquisition.SHEINUSPreparedRef, publish SHEINUSPublisher) (collection.Source, error) {
	if ctx == nil || p.Scope.Validate() != nil || p.CredentialID != "" || p.CredentialRevision != 0 || !ref.Valid() || publish == nil {
		return collection.Source{}, dataacquisition.ErrInvalid
	}
	var source collection.Source
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		scope := p.Scope
		row, err := readSHEINUSRow(tx, sheinUSSelection+" AND i.job_id=? FOR UPDATE OF j,i",
			scope.OrganizationID, scope.ActorID, scope.MemberID, ref.ItemID, collection.SourceSHEINUSData, ref.JobID)
		if err != nil {
			return err
		}
		if err := r.live.CheckExecution(ctx, p, row.Funding); err != nil {
			return err
		}
		replay := row.ItemState == "SAVED"
		if row.ReservationID != ref.ReservationID || row.ChargeState != orgresource.ReservationReserved ||
			(!replay && (row.ItemState != "PREPARED_EVIDENCE" || row.JobState != "RUNNING" || row.Canceled || !time.Now().Before(row.Deadline))) {
			return dataacquisition.ErrConflict
		}
		capture, err := row.capture(scope)
		if err != nil {
			return err
		}
		command, err := capture.Publication(ref.Checksum)
		if err != nil {
			return err
		}
		published, err := publish(ctx, tx, scope, row.Funding, command, replay)
		if errors.Is(err, sourcing.ErrSourcePublicationConflict) {
			return dataacquisition.ErrConflict
		}
		if err != nil {
			return err
		}
		if published.Kind != collection.SourceSHEINUSData || published.ProductKey != command.ProductKey ||
			published.PublicationID != ref.ItemID || published.OperationID != ref.ItemID || published.Version == 0 {
			return dataacquisition.ErrConflict
		}
		if replay {
			if capture.Source != published {
				return dataacquisition.ErrConflict
			}
			source = published
			return nil
		}
		batch, err := collection.NewPublicationBatch(scope, ref.JobID, collection.SourceSHEINUSData, "SHEIN · US")
		if err != nil {
			return err
		}
		if _, err = collectionstore.AppendDataPublication(ctx, tx, scope, batch, published, time.Now().UTC()); err != nil {
			return err
		}
		raw, err := json.Marshal(published)
		if err != nil {
			return err
		}
		result := tx.Exec(`UPDATE data_acquisition_items
 SET source_json=?,state='SAVED',saved_at=now(),terminal_evidence=?,claim_token=NULL,lease_until=NULL
 WHERE organization_id=? AND actor_id=? AND job_id=? AND id=? AND source_kind=? AND state='PREPARED_EVIDENCE'`,
			string(raw), uuid.NewString(), scope.OrganizationID, scope.ActorID, ref.JobID, ref.ItemID, collection.SourceSHEINUSData)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected != 1 {
			return dataacquisition.ErrConflict
		}
		source = published
		return nil
	})
	if err != nil {
		if source != (collection.Source{}) {
			return collection.Source{}, dataacquisition.ErrUnknown
		}
		return collection.Source{}, err
	}
	return source, nil
}
