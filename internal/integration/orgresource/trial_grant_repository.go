package orgresourceadapter

import (
	"context"
	"encoding/json"
	"errors"
	"strconv"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"task-processor/internal/ledger/orgresource"
)

func (repository *GormRepository) ReplayTrialAIGrant(ctx context.Context, input orgresource.TrialAIGrantExecution) (orgresource.TrialAIGrantResult, bool, error) {
	if ctx == nil {
		return orgresource.TrialAIGrantResult{}, false, orgresource.ErrInvalidInput
	}
	if err := input.Validate(); err != nil {
		return orgresource.TrialAIGrantResult{}, false, err
	}
	var result orgresource.TrialAIGrantResult
	var found bool
	err := repository.runner.runRead(ctx, func(readContext context.Context) error {
		var err error
		result, found, err = replayTrialAIGrantOnce(readContext, repository.db, input)
		return err
	})
	return result, found, err
}

func replayTrialAIGrantOnce(ctx context.Context, db *gorm.DB, input orgresource.TrialAIGrantExecution) (orgresource.TrialAIGrantResult, bool, error) {
	var operation organizationResourceOperationRow
	err := db.WithContext(ctx).Where("organization_id = ? AND operation_id = ?", input.OrganizationID, input.OperationID).Take(&operation).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		var claim organizationResourceSourceClaimRow
		err = db.WithContext(ctx).Where("source_type = ? AND source_identity = ?", orgresource.SourceLocalAITrial, input.SourceIdentity).Take(&claim).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return orgresource.TrialAIGrantResult{}, false, nil
		}
		if err != nil {
			return orgresource.TrialAIGrantResult{}, false, err
		}
		if claim.OrganizationID != input.OrganizationID || claim.ResourceType != string(orgresource.ResourceAIPoint) || claim.RequestFingerprint != input.RequestFingerprint {
			return orgresource.TrialAIGrantResult{}, false, orgresource.ErrIdempotencyKeyConflict
		}
		err = db.WithContext(ctx).Where("organization_id = ? AND operation_id = ?", claim.OrganizationID, claim.OperationID).Take(&operation).Error
	}
	if err != nil {
		return orgresource.TrialAIGrantResult{}, false, err
	}
	if operation.OperationType != orgresource.OperationGrantLocalAITrial || operation.RequestFingerprint != input.RequestFingerprint || operation.ApprovalEvidenceID != input.ApprovalEvidenceID {
		return orgresource.TrialAIGrantResult{}, false, orgresource.ErrIdempotencyKeyConflict
	}
	if operation.State != "succeeded" || operation.ImmutableResult == "" {
		return orgresource.TrialAIGrantResult{}, false, orgresource.ErrConcurrencyRetry
	}
	var snapshot orgresource.TrialAIGrantSnapshot
	if err := json.Unmarshal([]byte(operation.ImmutableResult), &snapshot); err != nil {
		return orgresource.TrialAIGrantResult{}, false, errors.New("trial grant receipt is unavailable")
	}
	if snapshot.OrganizationID != input.OrganizationID || snapshot.OperationID != operation.OperationID || snapshot.ResourceType != orgresource.ResourceAIPoint || snapshot.Quantity != "10000" || snapshot.SourceType != orgresource.SourceLocalAITrial || snapshot.SourceIdentity != input.SourceIdentity || snapshot.ApprovalEvidenceID != input.ApprovalEvidenceID || snapshot.EventID == "" {
		return orgresource.TrialAIGrantResult{}, false, errors.New("trial grant receipt is inconsistent")
	}
	return orgresource.TrialAIGrantResult{Snapshot: snapshot, Replayed: true}, true, nil
}

func (repository *GormRepository) ExecuteTrialAIGrant(ctx context.Context, input orgresource.TrialAIGrantExecution) (orgresource.TrialAIGrantResult, error) {
	if result, found, err := repository.ReplayTrialAIGrant(ctx, input); err != nil || found {
		return result, err
	}
	var committed orgresource.TrialAIGrantResult
	err := repository.runner.run(ctx, func(tx *gorm.DB) error {
		var err error
		committed, err = executeTrialAIGrantTransaction(tx, input)
		return err
	})
	if err == nil && repository.afterCommit != nil {
		err = repository.afterCommit()
	}
	if err == nil {
		return committed, nil
	}
	// A lost commit reply is never permission to choose another grant identity.
	if result, found, readErr := repository.ReplayTrialAIGrant(ctx, input); readErr == nil && found {
		return result, nil
	}
	return orgresource.TrialAIGrantResult{}, err
}

func executeTrialAIGrantTransaction(tx *gorm.DB, input orgresource.TrialAIGrantExecution) (orgresource.TrialAIGrantResult, error) {
	ctx := tx.Statement.Context
	if result, found, err := replayTrialAIGrantOnce(ctx, tx, input); err != nil || found {
		return result, err
	}
	bucket := organizationResourceBucketRow{OrganizationID: input.OrganizationID, ResourceType: string(orgresource.ResourceAIPoint)}
	if err := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&bucket).Error; err != nil {
		return orgresource.TrialAIGrantResult{}, err
	}
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("organization_id = ? AND resource_type = ?", input.OrganizationID, orgresource.ResourceAIPoint).Take(&bucket).Error; err != nil {
		return orgresource.TrialAIGrantResult{}, err
	}
	if result, found, err := replayTrialAIGrantOnce(ctx, tx, input); err != nil || found {
		return result, err
	}
	now := time.Now().UTC()
	credit, err := applyPositiveCredit(ctx, tx, input.OrganizationID, string(orgresource.ResourceAIPoint), orgresource.TrialAIGrantQuantity, bucket.Available, now)
	if err != nil {
		return orgresource.TrialAIGrantResult{}, err
	}
	snapshot := orgresource.TrialAIGrantSnapshot{OperationID: input.OperationID, OrganizationID: input.OrganizationID, ResourceType: orgresource.ResourceAIPoint, Quantity: "10000", GrossCredit: strconv.FormatInt(credit.gross, 10), DebtRepaid: strconv.FormatInt(credit.debtRepaid, 10), NetCredit: strconv.FormatInt(credit.net, 10), BalanceAfter: strconv.FormatInt(credit.availableAfter, 10), AllocatedAfter: strconv.FormatInt(bucket.Allocated, 10), ReservedAfter: strconv.FormatInt(bucket.Reserved, 10), ConsumedAfter: strconv.FormatInt(bucket.Consumed, 10), SourceType: orgresource.SourceLocalAITrial, SourceIdentity: input.SourceIdentity, ApprovalEvidenceID: input.ApprovalEvidenceID, ActorID: input.ActorID, EventID: uuid.NewString()}
	encoded, err := json.Marshal(snapshot)
	if err != nil {
		return orgresource.TrialAIGrantResult{}, err
	}
	if err := persistTrialAIGrant(tx, input, bucket, snapshot, credit, string(encoded), now); err != nil {
		return orgresource.TrialAIGrantResult{}, err
	}
	return orgresource.TrialAIGrantResult{Snapshot: snapshot}, nil
}

func persistTrialAIGrant(tx *gorm.DB, input orgresource.TrialAIGrantExecution, bucket organizationResourceBucketRow, snapshot orgresource.TrialAIGrantSnapshot, credit positiveCreditAllocation, encoded string, now time.Time) error {
	if err := tx.Create(&organizationResourceOperationRow{OrganizationID: input.OrganizationID, OperationID: input.OperationID, OperationType: orgresource.OperationGrantLocalAITrial, RequestFingerprint: input.RequestFingerprint, State: "succeeded", ImmutableResult: encoded, ApprovalEvidenceID: input.ApprovalEvidenceID, CompletedAt: &now}).Error; err != nil {
		return err
	}
	if err := tx.Create(&organizationResourceSourceClaimRow{SourceType: orgresource.SourceLocalAITrial, SourceIdentity: input.SourceIdentity, ResourceType: string(orgresource.ResourceAIPoint), OrganizationID: input.OrganizationID, OperationID: input.OperationID, RequestFingerprint: input.RequestFingerprint}).Error; err != nil {
		return err
	}
	if err := tx.Model(&organizationResourceBucketRow{}).Where("organization_id = ? AND resource_type = ?", input.OrganizationID, orgresource.ResourceAIPoint).Updates(map[string]any{"available": credit.availableAfter, "updated_at": now}).Error; err != nil {
		return err
	}
	if err := tx.Create(&organizationResourceEventRow{EventID: snapshot.EventID, OrganizationID: input.OrganizationID, OperationID: input.OperationID, ResourceType: string(orgresource.ResourceAIPoint), Quantity: orgresource.TrialAIGrantQuantity, AvailableDelta: credit.net, Reason: orgresource.OperationGrantLocalAITrial, SourceType: orgresource.SourceLocalAITrial, SourceIdentity: input.SourceIdentity, BalanceAfter: credit.availableAfter, AvailableAfter: credit.availableAfter, AllocatedAfter: bucket.Allocated, ReservedAfter: bucket.Reserved, ConsumedAfter: bucket.Consumed, GrossCredit: credit.gross, DebtRepaid: credit.debtRepaid, NetCredit: credit.net, CreatedAt: now}).Error; err != nil {
		return err
	}
	return tx.Create(&organizationResourceAuditLogRow{OrganizationID: input.OrganizationID, OperationID: input.OperationID, Action: orgresource.OperationGrantLocalAITrial, ActorID: input.ActorID, ApprovalEvidenceID: input.ApprovalEvidenceID, Payload: encoded, CreatedAt: now}).Error
}
