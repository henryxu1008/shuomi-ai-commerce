package listingapp

import (
	"context"
	"errors"
	"time"

	"github.com/henryxu1008/shuomi-ai-commerce/internal/authidentity"
	amazonapi "github.com/henryxu1008/shuomi-ai-commerce/internal/integration/amazon"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/listing/preparation"
	record "github.com/henryxu1008/shuomi-ai-commerce/internal/listing/record/target"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/listing/submission"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/product/asset"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/product/collection"
)

type AmazonListingRepository interface {
	ReadOfficialIntent(context.Context, collection.Scope, string) (submission.OfficialIntent, error)
	ReadAmazonAttempt(context.Context, submission.ExecutionScope, submission.ExecutionTarget) (submission.ExecutionAttempt, error)
	CompleteAmazon(context.Context, submission.ExecutionClaim, submission.OfficialIntent, submission.AmazonReceipt) error
	ReadAmazonReceipt(context.Context, collection.Scope, submission.ExecutionAttempt) (submission.AmazonReceipt, error)
	ReadAmazonObservation(context.Context, submission.ExecutionAttempt) (*submission.AmazonObservation, error)
	SaveAmazonObservation(context.Context, submission.ExecutionAttempt, submission.AmazonObservation) error
}

type AmazonListingService struct {
	Preparer      AmazonTargetPreparer
	Records       record.TargetQuery
	Sources       record.TargetExecutionSourceSelector
	Products      record.EffectiveTargetProductReader
	Assets        record.ApprovedAssetReader
	Authorization collection.ExecutionAuthorizer
	Kernel        *submission.ExecutionKernel
	Repository    AmazonListingRepository
}

type AmazonListingResult struct {
	SourceID    string                        `json:"sourceId"`
	StoreID     string                        `json:"storeId"`
	RecordID    string                        `json:"recordId"`
	AttemptID   string                        `json:"attemptId"`
	SKU         string                        `json:"sku"`
	Submission  string                        `json:"submission"`
	Receipt     *submission.AmazonReceipt     `json:"receipt,omitempty"`
	Observation *submission.AmazonObservation `json:"observation,omitempty"`
	Check       string                        `json:"check,omitempty"`
}

func (s *AmazonListingService) Confirm(ctx context.Context, scope collection.Scope, key, recordID, wireHash string) (AmazonListingResult, error) {
	if !collection.ValidID(key) || !collection.ValidID(recordID) || len(wireHash) != 64 {
		return AmazonListingResult{}, record.ErrInvalid
	}
	if err := s.authorizeAmazonSubmit(ctx, scope); err != nil {
		return AmazonListingResult{}, err
	}
	saved, err := s.Records.ReadTargetRecord(ctx, scope, recordID)
	if err != nil || saved.Amazon == nil || !record.PublicationReady(saved) || saved.Amazon.Draft.WireHash != wireHash {
		return AmazonListingResult{}, record.ErrConflict
	}
	ctx, cancel := context.WithTimeout(ctx, 90*time.Second)
	defer cancel()
	if _, err = s.source(ctx, scope, saved); err != nil {
		return AmazonListingResult{}, err
	}
	storeSubject := amazonSubject(scope)
	if !s.Preparer.Access.Current(ctx, storeSubject, saved.Merchant) {
		return AmazonListingResult{}, record.ErrNotReady
	}
	target := submission.AmazonTarget(saved.Merchant.StoreID, saved.Source.Source.ProductKey)
	execution := submission.ExecutionScope{OrganizationID: scope.OrganizationID}
	if intent, e := s.Repository.ReadOfficialIntent(ctx, scope, key); e == nil {
		if !submission.AmazonIntentMatchesRecord(intent, saved) {
			return AmazonListingResult{}, submission.ErrExecutionIntentConflict
		}
		attempt, e := s.Kernel.ReadIntent(ctx, execution, key)
		if e != nil {
			return AmazonListingResult{}, e
		}
		return s.result(ctx, scope, intent, attempt)
	} else if !errors.Is(e, submission.ErrExecutionNotFound) {
		return AmazonListingResult{}, e
	}
	if prior, e := s.Repository.ReadAmazonAttempt(ctx, execution, target); e == nil {
		if prior.Status != submission.ExecutionFailedDefinitive && prior.Status != submission.ExecutionNotSent {
			return s.readConfirmedAmazonResult(ctx, scope, saved, prior)
		}
		intent, e := s.Repository.ReadOfficialIntent(ctx, scope, prior.IntentKey)
		if e != nil {
			return AmazonListingResult{}, e
		}
		if intent.Amazon == nil {
			return AmazonListingResult{}, record.ErrNotFound
		}
		if prior.Status == submission.ExecutionFailedDefinitive && intent.Amazon.WireHash == wireHash {
			if !submission.AmazonIntentMatchesRecord(intent, saved) {
				return AmazonListingResult{}, submission.ErrExecutionIntentConflict
			}
			return s.result(ctx, scope, intent, prior)
		}
	} else if !errors.Is(e, submission.ErrExecutionNotFound) {
		return AmazonListingResult{}, e
	}
	if !time.Now().Before(saved.Amazon.ValidUntil) {
		return AmazonListingResult{}, record.ErrNotReady
	}
	observedURL, err := s.freshFacts(ctx, scope, saved)
	if err != nil {
		return AmazonListingResult{}, err
	}
	conn, err := s.Preparer.Access.Open(ctx, storeSubject, saved.Merchant.StoreID, &saved.Merchant)
	if err != nil {
		return AmazonListingResult{}, record.ErrNotReady
	}
	draft, err := s.Preparer.amazonPreview(ctx, conn, saved.Source.Source.ProductKey, *saved.Input.Amazon, observedURL)
	if err != nil || draft.WireHash != wireHash {
		return AmazonListingResult{}, record.ErrNotReady
	}
	if unused, err := conn.CheckGTINUnused(ctx, saved.Input.Amazon.GTIN); err != nil || !unused {
		return AmazonListingResult{}, record.ErrNotReady
	}
	if unused, err := conn.CheckSKUUnused(ctx, draft.SKU); err != nil || !unused {
		return AmazonListingResult{}, record.ErrNotReady
	}
	if !s.Preparer.Access.Current(ctx, storeSubject, saved.Merchant) ||
		s.Authorization.AuthorizeExecution(ctx, scope, preparation.PermissionSubmit) != nil {
		return AmazonListingResult{}, record.ErrForbidden
	}
	proof, err := s.source(ctx, scope, saved)
	if err != nil {
		return AmazonListingResult{}, err
	}
	authorization, err := submission.PrepareAmazonSend(ctx, proof, saved, key, conn.OccupancyPartition())
	if err != nil {
		return AmazonListingResult{}, err
	}
	intent, _, err := authorization.Read(ctx)
	if err != nil {
		return AmazonListingResult{}, err
	}
	acquired, err := s.Kernel.Acquire(ctx, submission.AcquireExecutionCommand{
		Scope: execution, IntentKey: key, Target: target, Action: submission.AmazonSubmitAction, Payload: draft.Attributes,
		ClaimOwnerID: "amazon-listing", Lease: submission.ExecutionTimeout, Amazon: authorization,
	})
	if errors.Is(err, submission.ErrExecutionTargetClaimed) || errors.Is(err, submission.ErrExecutionTargetSucceeded) {
		prior, e := s.Repository.ReadAmazonAttempt(ctx, execution, target)
		if e != nil {
			return AmazonListingResult{}, err
		}
		return s.readConfirmedAmazonResult(ctx, scope, saved, prior)
	}
	if err != nil {
		return AmazonListingResult{}, err
	}
	if acquired.Permit == nil {
		return s.readConfirmedAmazonResult(ctx, scope, saved, acquired.Attempt)
	}
	claim := uploadClaim(scope, acquired.Permit)
	if ctx.Err() != nil {
		return s.amazonUnknown(ctx, claim, intent, acquired.Attempt), nil
	}
	sendCtx, cancel := context.WithDeadline(context.WithoutCancel(ctx), acquired.Permit.LeaseExpiresAt)
	defer cancel()
	response, err := s.Preparer.Access.Submit(sendCtx, scope, conn, draft, acquired.AmazonPermit, func(ctx context.Context) error {
		head, err := s.Records.ReadTargetHead(ctx, scope, saved.TargetID)
		if err != nil || !submission.AmazonIntentMatchesRecord(intent, head) || !time.Now().Before(saved.Amazon.ValidUntil) {
			return record.ErrConflict
		}
		if _, err := s.source(ctx, scope, saved); err != nil {
			return err
		}
		return s.authorizeAmazonSubmit(ctx, scope)
	})
	receipt := submission.AmazonReceipt{Accepted: response.Accepted, SubmissionID: response.SubmissionID, Issues: amazonIssues(response.Issues)}
	finishCtx, finish := context.WithTimeout(context.WithoutCancel(ctx), submission.ExecutionTimeout)
	defer finish()
	if err != nil || !receipt.Valid() || !s.Preparer.Access.Current(finishCtx, storeSubject, saved.Merchant) ||
		s.Authorization.AuthorizeExecution(finishCtx, scope, preparation.PermissionRead) != nil ||
		s.Repository.CompleteAmazon(finishCtx, claim, intent, receipt) != nil {
		return s.amazonUnknown(finishCtx, claim, intent, acquired.Attempt), nil
	}
	result := amazonBaseResult(intent, acquired.Attempt)
	result.Receipt = &receipt
	result.Submission = amazonTerminalStatus(receipt)
	return result, nil
}

func (s *AmazonListingService) authorizeAmazonSubmit(ctx context.Context, scope collection.Scope) error {
	for _, permission := range []string{collection.PermissionRead, preparation.PermissionRead, preparation.PermissionManage, preparation.PermissionSubmit} {
		if s.Authorization.AuthorizeExecution(ctx, scope, permission) != nil {
			return record.ErrForbidden
		}
	}
	return nil
}

func (s *AmazonListingService) Read(ctx context.Context, scope collection.Scope, sourceID, storeID string, refresh bool) (AmazonListingResult, error) {
	if !collection.ValidID(sourceID) || !collection.ValidID(storeID) {
		return AmazonListingResult{}, record.ErrInvalid
	}
	ctx, cancel := context.WithTimeout(ctx, 25*time.Second)
	defer cancel()
	if s.Authorization.AuthorizeExecution(ctx, scope, preparation.PermissionRead) != nil {
		return AmazonListingResult{}, record.ErrForbidden
	}
	proof, err := s.Sources.SelectForExecution(ctx, scope, sourceID)
	if err != nil {
		return AmazonListingResult{}, err
	}
	_, source, _, err := proof.Read(ctx)
	if err != nil {
		return AmazonListingResult{}, err
	}
	attempt, err := s.Repository.ReadAmazonAttempt(ctx, submission.ExecutionScope{OrganizationID: scope.OrganizationID}, submission.AmazonTarget(storeID, source.Source.ProductKey))
	if err != nil {
		return AmazonListingResult{}, err
	}
	intent, err := s.Repository.ReadOfficialIntent(ctx, scope, attempt.IntentKey)
	if err != nil || intent.Amazon == nil || intent.Source.ID != sourceID {
		return AmazonListingResult{}, record.ErrNotFound
	}
	storeSubject := amazonSubject(scope)
	binding, err := s.Preparer.Access.Binding(ctx, storeSubject, storeID)
	if err != nil || binding.AmazonBeta != intent.Binding.AmazonBeta {
		return AmazonListingResult{}, record.ErrNotReady
	}
	result, err := s.result(ctx, scope, intent, attempt)
	if err != nil || !refresh || result.Submission == "rejected" {
		return result, err
	}
	conn, err := s.Preparer.Access.Open(ctx, storeSubject, storeID, &binding)
	if err != nil {
		result.Check = "unavailable"
		return result, nil
	}
	stamp := time.Now().UTC().Truncate(time.Microsecond)
	status, err := conn.GetListing(ctx, attempt.Target.SubjectID)
	if !s.Preparer.Access.Current(ctx, storeSubject, conn.Binding()) ||
		s.Authorization.AuthorizeExecution(ctx, scope, preparation.PermissionRead) != nil {
		return AmazonListingResult{}, record.ErrNotReady
	}
	if errors.Is(err, amazonapi.ErrNotFound) {
		result.Observation, result.Check = nil, "not_found"
		return result, nil
	}
	if err != nil {
		result.Check = "unavailable"
		return result, nil
	}
	observation := submission.AmazonObservation{ASIN: status.ASIN, Buyable: status.Buyable, Discoverable: status.Discoverable,
		Issues: amazonIssues(status.Issues), ObservedAt: stamp}
	if !observation.Valid() || s.Repository.SaveAmazonObservation(ctx, attempt, observation) != nil {
		result.Check = "unavailable"
		return result, nil
	}
	result.Observation, err = s.Repository.ReadAmazonObservation(ctx, attempt)
	return result, err
}

func (s *AmazonListingService) source(ctx context.Context, scope collection.Scope, saved record.TargetRecord) (preparation.AuthorizedSource, error) {
	proof, err := s.Sources.SelectForExecution(ctx, scope, saved.Source.ID)
	if err != nil {
		return preparation.AuthorizedSource{}, err
	}
	actualScope, current, _, err := proof.Read(ctx)
	if err != nil || actualScope != scope || current != saved.Source {
		return preparation.AuthorizedSource{}, record.ErrConflict
	}
	return proof, nil
}

func (s *AmazonListingService) freshFacts(ctx context.Context, scope collection.Scope, saved record.TargetRecord) (string, error) {
	proof, err := s.source(ctx, scope, saved)
	if err != nil {
		return "", err
	}
	product, err := s.Products.ReadEffectiveTargetProduct(ctx, proof, saved.EffectiveVersion, "")
	if err != nil || collection.Digest(product.Snapshot) != saved.ProductHash || len(product.Snapshot.Variants) > 1 {
		return "", record.ErrConflict
	}
	inventoryScope := asset.InventoryScope{TenantID: scope.OrganizationID, ProductKey: saved.Source.Source.ProductKey, TargetPlatform: "product", SourceSnapshotVersion: saved.EffectiveVersion}
	inventory, err := s.Assets.GetApprovedInventory(ctx, inventoryScope)
	if err != nil || inventory.Scope != inventoryScope || collection.Digest(inventory) != saved.InventoryHash {
		return "", record.ErrConflict
	}
	observed, err := s.Preparer.amazonImage(ctx, saved.Input.Amazon.MainImageID, inventory)
	if err != nil || observed.ContentHash != saved.Amazon.ImageHash {
		return "", record.ErrConflict
	}
	return observed.SourceURL, nil
}

func amazonBaseResult(intent submission.OfficialIntent, attempt submission.ExecutionAttempt) AmazonListingResult {
	return AmazonListingResult{SourceID: intent.Source.ID, StoreID: intent.Target.StoreID, RecordID: intent.RecordID, AttemptID: attempt.AttemptID, SKU: attempt.Target.SubjectID, Submission: "unknown"}
}

func amazonTerminalStatus(receipt submission.AmazonReceipt) string {
	if receipt.Accepted {
		return "accepted"
	}
	return "rejected"
}

func (s *AmazonListingService) result(ctx context.Context, scope collection.Scope, intent submission.OfficialIntent, attempt submission.ExecutionAttempt) (AmazonListingResult, error) {
	if intent.Amazon == nil || intent.Owner != scope || attempt.OrganizationID != scope.OrganizationID ||
		intent.Target != attempt.Target || intent.Key != attempt.IntentKey || intent.PayloadFingerprint != attempt.PayloadFingerprint {
		return AmazonListingResult{}, record.ErrNotFound
	}
	if s.Authorization.AuthorizeExecution(ctx, scope, preparation.PermissionRead) != nil {
		return AmazonListingResult{}, record.ErrForbidden
	}
	result := amazonBaseResult(intent, attempt)
	switch attempt.Status {
	case submission.ExecutionNotSent:
		result.Submission = "not_sent"
	case submission.ExecutionSucceeded, submission.ExecutionFailedDefinitive:
		receipt, err := s.Repository.ReadAmazonReceipt(ctx, scope, attempt)
		if err != nil {
			return AmazonListingResult{}, err
		}
		result.Receipt = &receipt
		result.Submission = amazonTerminalStatus(receipt)
	}
	observation, err := s.Repository.ReadAmazonObservation(ctx, attempt)
	result.Observation = observation
	return result, err
}

func (s *AmazonListingService) readIntentResult(ctx context.Context, scope collection.Scope, attempt submission.ExecutionAttempt) (AmazonListingResult, error) {
	intent, err := s.Repository.ReadOfficialIntent(ctx, scope, attempt.IntentKey)
	if err != nil {
		return AmazonListingResult{}, err
	}
	return s.result(ctx, scope, intent, attempt)
}

func (s *AmazonListingService) readConfirmedAmazonResult(ctx context.Context, scope collection.Scope, saved record.TargetRecord, attempt submission.ExecutionAttempt) (AmazonListingResult, error) {
	intent, err := s.Repository.ReadOfficialIntent(ctx, scope, attempt.IntentKey)
	if err != nil || !submission.AmazonIntentMatchesRecord(intent, saved) {
		return AmazonListingResult{}, submission.ErrExecutionIntentConflict
	}
	return s.result(ctx, scope, intent, attempt)
}

func (s *AmazonListingService) amazonUnknown(ctx context.Context, claim submission.ExecutionClaim, intent submission.OfficialIntent, attempt submission.ExecutionAttempt) AmazonListingResult {
	finishCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), submission.ExecutionTimeout)
	defer cancel()
	if _, err := s.Kernel.MarkUnknown(finishCtx, claim, submission.UnknownResponseLost); err != nil {
		if current, e := s.Kernel.Get(finishCtx, claim.Scope, claim.AttemptID); e == nil &&
			s.Preparer.Access.Current(finishCtx, amazonSubject(intent.Owner), intent.Binding) {
			if result, readErr := s.readIntentResult(finishCtx, intent.Owner, current); readErr == nil {
				return result
			}
		}
	}
	return amazonBaseResult(intent, attempt)
}

func amazonIssues(issues []amazonapi.Issue) []submission.AmazonIssue {
	var result []submission.AmazonIssue
	for _, issue := range issues {
		if !authidentity.IsBoundedIdentifier(issue.Code) || issue.Severity != "ERROR" && issue.Severity != "WARNING" && issue.Severity != "INFO" {
			continue
		}
		safe := submission.AmazonIssue{Code: issue.Code, Severity: issue.Severity}
		for _, field := range issue.AttributeNames {
			if authidentity.IsBoundedIdentifier(field) {
				safe.Attributes = append(safe.Attributes, field)
				if len(safe.Attributes) == 8 {
					break
				}
			}
		}
		result = append(result, safe)
		if len(result) == 20 {
			break
		}
	}
	return result
}
