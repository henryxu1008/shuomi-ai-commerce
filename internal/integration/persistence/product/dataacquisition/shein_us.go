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
	JobID, ItemID, NativeGoodsID, ItemState, TrialBinding, InputHash, EvidenceChecksum string
	JobState                                                                           string
	Canceled                                                                           bool
	Deadline                                                                           time.Time
	Funding                                                                            orgresource.ResourceFunding

	QueryJSON, EvidenceJSON, SourceJSON []byte
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
 j.trial_binding,i.evidence_checksum,
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

func (r *Repository) SHEINUSTrialPreparedRefs(ctx context.Context, a dataacquisition.SHEINUSTrialApproval) ([]dataacquisition.SHEINUSPreparedRef, error) {
	var rows []struct {
		ID, EvidenceChecksum string
	}
	scope := a.Scope
	err := r.db.WithContext(ctx).Raw(`SELECT i.id,i.evidence_checksum FROM data_acquisition_items i
 JOIN data_acquisition_jobs j ON j.organization_id=i.organization_id AND j.actor_id=i.actor_id AND j.id=i.job_id AND j.source_kind=i.source_kind
 WHERE j.organization_id=? AND j.actor_id=? AND j.member_id=? AND j.id=? AND j.source_kind='shein_us_data' AND j.trial_binding=?
 AND i.state IN ('PREPARED_EVIDENCE','SAVED') ORDER BY i.source_position`,
		scope.OrganizationID, scope.ActorID, scope.MemberID, a.ID, a.Binding()).Scan(&rows).Error
	if err != nil {
		return nil, dataacquisition.ErrUnavailable
	}
	refs := make([]dataacquisition.SHEINUSPreparedRef, 0, len(rows))
	for _, row := range rows {
		refs = append(refs, dataacquisition.SHEINUSPreparedRef{JobID: a.ID, ItemID: row.ID, Checksum: row.EvidenceChecksum})
	}
	return refs, nil
}

// PublishSHEINUS consumes a prepared card for an authorized original page.
func (r *Repository) PublishSHEINUS(ctx context.Context, p dataacquisition.Principal, a dataacquisition.SHEINUSTrialApproval, ref dataacquisition.SHEINUSPreparedRef, publish SHEINUSPublisher) (collection.Source, error) {
	if ctx == nil || p.Scope.Validate() != nil || p.CredentialID != "" || p.CredentialRevision != 0 || !ref.Valid() || publish == nil || a.ID != ref.JobID || a.Scope != p.Scope {
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
		if row.TrialBinding != a.Binding() || row.Funding != a.Funding || row.EvidenceChecksum != ref.Checksum ||
			(!replay && (row.ItemState != "PREPARED_EVIDENCE" || row.JobState != "RUNNING" || row.Canceled || !time.Now().Before(row.Deadline))) {
			return dataacquisition.ErrConflict
		}
		var eligible bool
		err = tx.Raw(`SELECT EXISTS(SELECT 1 FROM data_acquisition_shein_us_trials
 WHERE approval_id=? AND binding=? AND request_id IS NOT NULL AND credits BETWEEN 0 AND ?)`, a.ID, row.TrialBinding, dataacquisition.SHEINUSTrialCreditLimit).Scan(&eligible).Error
		if err != nil {
			return dataacquisition.ErrUnavailable
		}
		if !eligible {
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
		if err := tx.Exec(`UPDATE data_acquisition_jobs j SET state='SUCCEEDED'
 WHERE j.organization_id=? AND j.actor_id=? AND j.id=? AND j.source_kind=? AND j.state='RUNNING'
 AND NOT EXISTS(SELECT 1 FROM data_acquisition_items i WHERE i.organization_id=j.organization_id AND i.actor_id=j.actor_id AND i.job_id=j.id AND i.source_kind=j.source_kind AND i.state<>'SAVED')`,
			scope.OrganizationID, scope.ActorID, ref.JobID, collection.SourceSHEINUSData).Error; err != nil {
			return err
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
