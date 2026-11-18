package dataacquisitionpersistence

import (
	"context"
	"encoding/json"
	"time"

	"gorm.io/gorm"
	"task-processor/internal/product/collection"
	acq "task-processor/internal/product/dataacquisition"
)

const amazonClaimAction = "zenrows_amazon_claim"
const amazonResultAction = "zenrows_amazon_result"

type amazonFirstPageCommand struct {
	OrganizationID, ActorID, MemberID, InputHash, Action, TargetID string
	ReceiptJSON                                                    []byte
}

const amazonCommandColumns = "organization_id,actor_id,member_id,input_hash,action,target_id::text,receipt_json"

func (c amazonFirstPageCommand) matches(a acq.AmazonFirstPageApproval, action string) bool {
	return c.OrganizationID == a.Scope.OrganizationID && c.ActorID == a.Scope.ActorID &&
		c.MemberID == a.Scope.MemberID && c.TargetID == a.ID && c.InputHash == a.Binding() && c.Action == action
}

func amazonClaim(db *gorm.DB, a acq.AmazonFirstPageApproval) (amazonFirstPageCommand, error) {
	var row amazonFirstPageCommand
	got := db.Raw("SELECT "+amazonCommandColumns+" FROM data_service_commands WHERE organization_id=? AND actor_id=? AND command_key=?", a.Scope.OrganizationID, a.Scope.ActorID, a.ID).Scan(&row)
	if got.Error != nil {
		return row, got.Error
	}
	if got.RowsAffected != 1 {
		return row, acq.ErrNotFound
	}
	if !row.matches(a, amazonClaimAction) {
		return row, acq.ErrConflict
	}
	return row, nil
}

func amazonResultKey(a acq.AmazonFirstPageApproval) string {
	return collection.StableID(a.Scope.OrganizationID, a.Scope.ActorID, "zenrows-amazon-result", a.ID)
}

func (r *Repository) ReadAmazonFirstPage(ctx context.Context, a acq.AmazonFirstPageApproval) (acq.AmazonFirstPageReceipt, error) {
	var out acq.AmazonFirstPageReceipt
	db := r.db.WithContext(ctx)
	if _, err := amazonClaim(db, a); err != nil {
		return out, err
	}
	var row amazonFirstPageCommand
	got := db.Raw("SELECT "+amazonCommandColumns+" FROM data_service_commands WHERE organization_id=? AND actor_id=? AND command_key=?", a.Scope.OrganizationID, a.Scope.ActorID, amazonResultKey(a)).Scan(&row)
	if got.Error != nil {
		return out, got.Error
	}
	if got.RowsAffected == 0 {
		return acq.AmazonFirstPageReceipt{ApprovalID: a.ID, State: acq.AmazonFirstPageUnknown}, nil
	}
	if !row.matches(a, amazonResultAction) || json.Unmarshal(row.ReceiptJSON, &out) != nil || out.Validate() != nil || out.ApprovalID != a.ID {
		return acq.AmazonFirstPageReceipt{}, acq.ErrConflict
	}
	return out, nil
}

func (r *Repository) ClaimAmazonFirstPage(ctx context.Context, a acq.AmazonFirstPageApproval) (bool, error) {
	if a.Validate() != nil {
		return false, acq.ErrInvalid
	}
	won := false
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec("SELECT pg_advisory_xact_lock(hashtextextended(?,0))", "zenrows-amazon:"+a.ID).Error; err != nil {
			return err
		}
		var claims []amazonFirstPageCommand
		if err := tx.Raw("SELECT "+amazonCommandColumns+" FROM data_service_commands WHERE command_key=? AND action=? LIMIT 2", a.ID, amazonClaimAction).Scan(&claims).Error; err != nil {
			return err
		}
		if len(claims) != 0 {
			if len(claims) != 1 || !claims[0].matches(a, amazonClaimAction) {
				return acq.ErrConflict
			}
			return nil
		}
		var now time.Time
		if err := tx.Raw("SELECT clock_timestamp()").Scan(&now).Error; err != nil {
			return err
		}
		if !now.Before(a.SendBefore()) {
			return acq.ErrForbidden
		}
		result := tx.Exec(`INSERT INTO data_service_commands
			(organization_id,actor_id,member_id,command_key,input_hash,action,target_id,receipt_json,created_at)
			VALUES(?,?,?,?,?,?,?,'{}',?) ON CONFLICT DO NOTHING`,
			a.Scope.OrganizationID, a.Scope.ActorID, a.Scope.MemberID, a.ID, a.Binding(), amazonClaimAction, a.ID, now)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected != 1 {
			return acq.ErrConflict
		}
		won = true
		return nil
	})
	// A commit acknowledgement failure never conveys permission to send.
	return won && err == nil, err
}

func insertAmazonFirstPageResult(tx *gorm.DB, a acq.AmazonFirstPageApproval, receipt acq.AmazonFirstPageReceipt) error {
	if receipt.Validate() != nil || receipt.ApprovalID != a.ID {
		return acq.ErrInvalid
	}
	if _, err := amazonClaim(tx, a); err != nil {
		return err
	}
	body, err := json.Marshal(receipt)
	if err != nil {
		return err
	}
	return tx.Exec(`INSERT INTO data_service_commands
		(organization_id,actor_id,member_id,command_key,input_hash,action,target_id,receipt_json,created_at)
		VALUES(?,?,?,?,?,?,?,?,clock_timestamp())`,
		a.Scope.OrganizationID, a.Scope.ActorID, a.Scope.MemberID, amazonResultKey(a), a.Binding(), amazonResultAction, a.ID, string(body)).Error
}

func (r *Repository) RecordAmazonFirstPage(ctx context.Context, a acq.AmazonFirstPageApproval, receipt acq.AmazonFirstPageReceipt) error {
	if receipt.State == acq.AmazonFirstPagePrepared {
		return acq.ErrInvalid
	}
	return insertAmazonFirstPageResult(r.db.WithContext(ctx), a, receipt)
}

type preparedAmazonFirstPage struct {
	approval acq.AmazonFirstPageApproval
	receipt  acq.AmazonFirstPageReceipt
	ids      []string
	evidence map[string]acq.Evidence
}

func (r *Repository) PrepareAmazonFirstPage(ctx context.Context, a acq.AmazonFirstPageApproval, submission acq.AmazonFirstPageSubmission) (acq.Job, error) {
	if a.Validate() != nil || submission.Receipt.Validate() != nil || submission.Receipt.ApprovalID != a.ID ||
		submission.Receipt.State != acq.AmazonFirstPageUnknown || submission.Receipt.RequestID == "" ||
		(submission.Receipt.Credits != nil && *submission.Receipt.Credits > acq.AmazonFirstPageRequestCredits) ||
		len(submission.Evidence) < 2 || len(submission.Evidence) > acq.AmazonFirstPageLimit {
		return acq.Job{}, acq.ErrInvalid
	}
	page := &preparedAmazonFirstPage{approval: a, receipt: submission.Receipt, evidence: map[string]acq.Evidence{}}
	for _, fact := range submission.Evidence {
		if _, found := page.evidence[fact.ASIN]; found || fact.Validate() != nil || fact.ParserVersion != acq.AmazonZenRowsSearchVersion {
			return acq.Job{}, acq.ErrInvalid
		}
		page.ids = append(page.ids, fact.ASIN)
		page.evidence[fact.ASIN] = fact
	}
	command := collection.StableID(a.Scope.OrganizationID, a.Scope.ActorID, "zenrows-amazon-job", a.ID)
	return r.admit(ctx, acq.Principal{Scope: a.Scope}, command, a.Query(), a.Funding, a.Price, page)
}
