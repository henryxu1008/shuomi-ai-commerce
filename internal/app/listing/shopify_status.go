package listingapp

import (
	"context"
	"errors"
	"time"

	storeapp "github.com/henryxu1008/shuomi-ai-commerce/internal/app/storecenter"
	record "github.com/henryxu1008/shuomi-ai-commerce/internal/listing/record/target"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/listing/submission"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/storecenter"
)

type ShopifyStatusService struct {
	Store    *storeapp.OfficialProductAccess
	Attempts interface {
		ReadShopifyAttempt(context.Context, submission.ExecutionScope, submission.ExecutionTarget) (submission.ExecutionAttempt, error)
	}
	Kernel interface {
		Expire(context.Context, submission.ExecutionScope, string) (submission.ExecutionAttempt, error)
	}
	Intents     submission.OfficialIntentRepository
	Resolutions submission.OfficialResolutionRepository
	Records     record.TargetRepository
}

func (s *ShopifyStatusService) read(ctx context.Context, view PublicationView) (PublicationView, error) {
	ctx, cancel := context.WithTimeout(ctx, time.Minute)
	defer cancel()
	subject := shopifyProductSubject(view.shopifyScope, storecenter.ProductPurposeStatus)
	if view.Shopify != nil {
		readback, err := s.Store.ReadShopifyDraft(ctx, subject, view.binding, view.Shopify.ID, "")
		if err != nil {
			return view, record.ErrUnavailable
		}
		view.ShopifyStatus = &readback.Product
		view.ShopifyAdminURL = readback.AdminURL
		return view, nil
	}
	scope := submission.ExecutionScope{OrganizationID: view.shopifyScope.OrganizationID}
	attempt, err := s.Attempts.ReadShopifyAttempt(ctx, scope, view.shopifyTarget)
	if errors.Is(err, submission.ErrExecutionNotFound) {
		view.ShopifyAttemptStatus = "not_started"
		return view, nil
	}
	if err != nil {
		return view, err
	}
	view.ShopifyAttemptID, view.ShopifyAttemptStatus = attempt.AttemptID, attempt.Status
	if attempt.Status == submission.ExecutionClaimed && !time.Now().Before(attempt.LeaseExpiresAt) {
		attempt, err = s.Kernel.Expire(ctx, scope, attempt.AttemptID)
		if err != nil {
			return view, err
		}
		view.ShopifyAttemptStatus = attempt.Status
	}
	if attempt.Status != submission.ExecutionOutcomeUnknown {
		return view, nil
	}
	intent, err := s.Intents.ReadOfficialIntent(ctx, view.shopifyScope, attempt.IntentKey)
	if errors.Is(err, submission.ErrExecutionNotFound) {
		return view, nil
	}
	if err != nil || intent.Owner != view.shopifyScope || intent.Target != view.shopifyTarget {
		return view, record.ErrConflict
	}
	saved, err := s.Records.ReadTargetRecord(ctx, intent.Owner, intent.RecordID)
	if err != nil {
		return view, err
	}
	draft, _, err := submission.ShopifyDraftPayload(saved)
	if err != nil {
		return view, err
	}
	readback, err := s.Store.ReadShopifyDraft(ctx, subject, view.binding, "", draft.SourceKey)
	if err != nil {
		return view, nil // The original attempt remains unknown unless the exact source is found.
	}
	proof, err := submission.NewShopifyDraftResolution(intent, saved, attempt, readback.Product)
	if err != nil {
		return view, err
	}
	retained, err := s.Resolutions.ResolveOfficial(ctx, proof)
	if err != nil {
		return view, err
	}
	view.Shopify, view.ShopifyStatus, view.RecordID, view.ReceiptID, view.ObservedAt = retained.Shopify, &readback.Product, retained.RecordID, retained.ID, &retained.ObservedAt
	view.ShopifyAdminURL = readback.AdminURL
	view.ShopifyAttemptStatus = submission.ExecutionSucceeded
	return view, nil
}
