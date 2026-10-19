package supplychainapp

import (
	"context"
	"crypto/subtle"
	"errors"
	"github.com/google/uuid"
	record "task-processor/internal/listing/record/target"
	"task-processor/internal/listing/submission"
	"task-processor/internal/marketplace/shein/goods"
	"task-processor/internal/marketplace/shein/model"
	"task-processor/internal/product/collection"
	"task-processor/internal/storecenter"
	"time"
)

type SheinShelfConfirmation struct {
	Key            string                `json:"key"`
	SourceID       string                `json:"sourceId"`
	StoreID        string                `json:"storeId"`
	ReceiptID      string                `json:"receiptId"`
	SKC            string                `json:"skc"`
	SPU            string                `json:"spu"`
	Version        string                `json:"version"`
	Site           string                `json:"site"`
	Mode           model.ApplicationMode `json:"mode"`
	Token          string                `json:"token"`
	ExpiresAt      time.Time             `json:"expiresAt"`
	ShelfWay       string                `json:"shelfWay"`
	HopeOnSaleDate string                `json:"hopeOnSaleDate"`
}
type SheinShelfActionResult struct {
	Key         string                      `json:"key"`
	AttemptID   string                      `json:"attemptId"`
	SourceID    string                      `json:"sourceId"`
	StoreID     string                      `json:"storeId"`
	ReceiptID   string                      `json:"receiptId"`
	SKC         string                      `json:"skc"`
	Status      submission.ExecutionStatus  `json:"status"`
	Message     string                      `json:"message"`
	Observation *SheinPublicationStatus     `json:"observation,omitempty"`
	Response    *model.ShelfResponseSummary `json:"response,omitempty"`
}

func (s *SheinPublicationService) preflight(ctx context.Context, p loadedSheinPublication, skc string) (SheinSKCStatus, string, error) {
	view, e := s.observe(ctx, p, false, 0)
	if e != nil {
		return SheinSKCStatus{}, "", e
	}
	plan, e := submission.OriginalShelfPlan(p.record, p.receipt, skc)
	if e != nil {
		return SheinSKCStatus{}, "", record.ErrNotReady
	}
	for _, row := range view.SKCs {
		if row.SKC == skc {
			eligible, _ := goods.ShelfEligibility(view.Mode, plan, row.AuditState, row.ShelfState, s.now())
			if !eligible {
				return row, "", record.ErrNotReady
			}
			hash := collection.Digest([]any{p.receipt.ID, collection.Digest(p.record), p.binding, skc, p.receipt.Product.Version, row.AuditState, row.ShelfState, plan.ShelfWay, plan.HopeOnSaleDate, plan.ShelfRequire})
			return row, hash, nil
		}
	}
	return SheinSKCStatus{}, "", record.ErrNotReady
}
func (s *SheinPublicationService) Prepare(ctx context.Context, key string, in SheinShelfInput) (SheinShelfConfirmation, error) {
	var out SheinShelfConfirmation
	if ctx == nil || !collection.ValidID(key) || in.StockOffset != 0 {
		return out, record.ErrInvalid
	}
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	p, e := s.load(ctx, in.SourceID, in.StoreID, in.ReceiptID, true)
	if e != nil {
		return out, e
	}
	row, hash, e := s.preflight(ctx, p, in.SKC)
	if e != nil || !row.CanShelf {
		if e == nil {
			e = record.ErrNotReady
		}
		return out, e
	}
	intent, e := s.d.Intents.ReadOfficialIntent(ctx, p.scope, key)
	if e == nil {
		if !s.matchesIntent(p, intent, in) || intent.Shelf.ObservationHash != hash || !s.now().Before(intent.Shelf.ExpiresAt) {
			return out, record.ErrConflict
		}
	} else {
		if !errors.Is(e, submission.ErrExecutionNotFound) {
			return out, record.ErrUnavailable
		}
		proof, e := submission.NewShelfIntent(ctx, p.source, p.record, p.receipt, p.binding, key, in.SKC, uuid.NewString(), hash, s.now())
		if e != nil {
			return out, e
		}
		intent, e = s.d.Intents.PrepareShelfIntent(ctx, proof)
		if e != nil {
			return out, e
		}
	}
	plan, e := submission.OriginalShelfPlan(p.record, p.receipt, in.SKC)
	if e != nil {
		return out, e
	}
	return SheinShelfConfirmation{Key: intent.Key, SourceID: in.SourceID, StoreID: in.StoreID, ReceiptID: in.ReceiptID, SKC: in.SKC, SPU: p.receipt.Product.SPUName, Version: p.receipt.Product.Version, Site: "shein-us", Mode: model.ApplicationMode(p.binding.ApplicationType), Token: intent.Shelf.ConfirmationToken, ExpiresAt: intent.Shelf.ExpiresAt, ShelfWay: plan.ShelfWay, HopeOnSaleDate: plan.HopeOnSaleDate}, nil
}
func (s *SheinPublicationService) matchesIntent(p loadedSheinPublication, v submission.OfficialIntent, in SheinShelfInput) bool {
	return submission.ValidateOfficialIntent(v) == nil && v.Shelf != nil && v.Owner == p.scope && v.RecordID == p.record.ID && v.RecordHash == collection.Digest(p.record) && v.Binding == p.binding && v.Shelf.OriginalReceiptID == p.receipt.ID && v.Shelf.OriginalReceiptHash == collection.Digest(p.receipt) && v.Shelf.SKC == in.SKC
}
func (s *SheinPublicationService) Read(ctx context.Context, key string, in SheinShelfInput) (SheinShelfActionResult, error) {
	if ctx == nil || !collection.ValidID(key) || in.StockOffset != 0 {
		return SheinShelfActionResult{}, record.ErrInvalid
	}
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	p, e := s.load(ctx, in.SourceID, in.StoreID, in.ReceiptID, false)
	if e != nil {
		return SheinShelfActionResult{}, e
	}
	intent, e := s.d.Intents.ReadOfficialIntent(ctx, p.scope, key)
	if e != nil || !s.matchesIntent(p, intent, in) {
		return SheinShelfActionResult{}, record.ErrForbidden
	}
	attempt, e := s.d.Kernel.ReadIntent(ctx, submission.ExecutionScope{OrganizationID: p.scope.OrganizationID}, key)
	if e != nil {
		return SheinShelfActionResult{}, e
	}
	if attempt.Status == submission.ExecutionClaimed && !s.now().Before(attempt.LeaseExpiresAt) {
		attempt, e = s.d.Kernel.Expire(ctx, submission.ExecutionScope{OrganizationID: p.scope.OrganizationID}, attempt.AttemptID)
	}
	if e != nil || attempt.Target != intent.Target || attempt.PayloadFingerprint != intent.PayloadFingerprint {
		return SheinShelfActionResult{}, record.ErrConflict
	}
	return shelfActionResult(intent, attempt, in), nil
}
func shelfActionResult(intent submission.OfficialIntent, attempt submission.ExecutionAttempt, in SheinShelfInput) SheinShelfActionResult {
	message := "原上架请求结果待核实；请查询平台结果，不会重新发送"
	if attempt.Status == submission.ExecutionSucceeded {
		message = "上架请求成功；当前平台状态与库存另行回读"
	}
	return SheinShelfActionResult{Key: intent.Key, AttemptID: attempt.AttemptID, SourceID: in.SourceID, StoreID: in.StoreID, ReceiptID: in.ReceiptID, SKC: in.SKC, Status: attempt.Status, Message: message}
}
func (s *SheinPublicationService) Confirm(ctx context.Context, key string, in SheinShelfInput) (SheinShelfActionResult, error) {
	var out SheinShelfActionResult
	if ctx == nil || !collection.ValidID(key) || !collection.ValidID(in.Token) || in.StockOffset != 0 {
		return out, record.ErrInvalid
	}
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	p, e := s.load(ctx, in.SourceID, in.StoreID, in.ReceiptID, true)
	if e != nil {
		return out, e
	}
	intent, e := s.d.Intents.ReadOfficialIntent(ctx, p.scope, key)
	if e != nil || !s.matchesIntent(p, intent, in) || subtle.ConstantTimeCompare([]byte(in.Token), []byte(intent.Shelf.ConfirmationToken)) != 1 {
		return out, record.ErrForbidden
	}
	if attempt, e := s.d.Kernel.ReadIntent(ctx, submission.ExecutionScope{OrganizationID: p.scope.OrganizationID}, key); e == nil {
		if attempt.Target != intent.Target || attempt.PayloadFingerprint != intent.PayloadFingerprint {
			return out, record.ErrConflict
		}
		return shelfActionResult(intent, attempt, in), nil
	} else if !errors.Is(e, submission.ErrExecutionNotFound) {
		return out, record.ErrUnavailable
	}
	if !s.now().Before(intent.Shelf.ExpiresAt) {
		return out, record.ErrNotReady
	}
	_, hash, e := s.preflight(ctx, p, in.SKC)
	if e != nil || hash != intent.Shelf.ObservationHash {
		return out, record.ErrNotReady
	}
	acquired, e := s.d.Kernel.Acquire(ctx, submission.AcquireExecutionCommand{Scope: submission.ExecutionScope{OrganizationID: p.scope.OrganizationID}, IntentKey: key, Target: intent.Target, Action: submission.SheinShelfAction, Payload: submission.ShelfPayload(intent), ClaimOwnerID: "supply-shelf", Lease: time.Minute})
	if e != nil {
		return out, e
	}
	if acquired.Permit == nil {
		return shelfActionResult(intent, acquired.Attempt, in), nil
	}
	permit := acquired.Permit
	claim := submission.ExecutionClaim{Scope: submission.ExecutionScope{OrganizationID: p.scope.OrganizationID}, AttemptID: permit.AttemptID, FenceEpoch: permit.FenceEpoch, OwnerID: permit.ClaimOwnerID, Token: permit.ClaimToken}
	unknown := func() (SheinShelfActionResult, error) {
		retain, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
		defer cancel()
		attempt, e := s.d.Kernel.MarkUnknown(retain, claim, submission.UnknownResponseLost)
		if e != nil {
			attempt = acquired.Attempt
			attempt.Status = submission.ExecutionOutcomeUnknown
		}
		return shelfActionResult(intent, attempt, in), nil
	}
	// Fresh original-source and merchant authorization after the committed permit;
	// current observations never create completion or resolve UNKNOWN.
	if _, hash, e = s.preflight(ctx, p, in.SKC); e != nil || hash != intent.Shelf.ObservationHash {
		return unknown()
	}
	if e = s.d.ExecutionAuthorization.AuthorizeExecution(ctx, p.scope, "workbench.listing.submit"); e != nil {
		return unknown()
	}
	m, e := s.merchant(ctx, p, storecenter.ProductPurposeShelfOn)
	if e != nil || !s.now().Before(permit.LeaseExpiresAt) || !s.now().Before(intent.Shelf.ExpiresAt) {
		return unknown()
	}
	call, cancelCall := context.WithTimeout(ctx, 8*time.Second)
	result, e := m.ShelfOn(call, in.SKC)
	cancelCall()
	if e != nil {
		out, e := unknown()
		out.Response = result.Response
		return out, e
	}
	proof, e := submission.NewShelfCompletion(intent, acquired.Attempt, claim, result)
	if e != nil {
		return unknown()
	}
	retain, cancelRetain := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
	defer cancelRetain()
	if _, e = s.d.Receipts.CompleteOfficial(retain, proof); e != nil {
		return unknown()
	}
	acquired.Attempt.Status = submission.ExecutionSucceeded
	out = shelfActionResult(intent, acquired.Attempt, in)
	// One readback after known success; a read error never changes its receipt.
	if observation, e := s.observe(ctx, p, true, 0); e == nil {
		out.Observation = &observation
	}
	return out, nil
}
