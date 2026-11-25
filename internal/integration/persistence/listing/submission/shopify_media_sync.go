package submissionpersistence

import (
	"context"
	"crypto/subtle"
	"database/sql"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/integration/persistence/listing/shopifyguard"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/listing/submission"
	"gorm.io/gorm"
)

const (
	mediaPlanTable = shopifyguard.MediaSyncPlanTable
	mediaStepTable = shopifyguard.MediaSyncStepTable
	mediaRunLease  = 55 * time.Second
	maxMediaCycles = 3
)

// InstallIsolatedShopifyMediaSchema only extends a continuously isolated empty
// execution store. Neither installation nor execution is wired into runtime.
func InstallIsolatedShopifyMediaSchema(ctx context.Context, db *gorm.DB, live *shopifyguard.LiveIsolation) error {
	if err := shopifyguard.VerifyLiveIsolation(db, live); err != nil {
		return err
	}
	return db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec("LOCK TABLE " + attemptTable + " IN SHARE ROW EXCLUSIVE MODE").Error; err != nil {
			return err
		}
		var used bool
		if err := tx.Raw("SELECT EXISTS (SELECT 1 FROM " + attemptTable + " WHERE platform='shopify')").Row().Scan(&used); err != nil {
			return err
		}
		if used {
			return submission.ErrExecutionEvidenceRequired
		}
		return tx.Exec(`CREATE TABLE ` + mediaPlanTable + ` (
			organization_id text NOT NULL, plan_id uuid NOT NULL, shop_key uuid NOT NULL, product_id text NOT NULL,
			generation bigint NOT NULL CHECK(generation>0), source_key text NOT NULL, body_hash text NOT NULL, body_json jsonb NOT NULL,
			expires_at timestamptz NOT NULL,
			token_hash text NOT NULL DEFAULT '', lease_until timestamptz, cycle int NOT NULL DEFAULT 0 CHECK(cycle>=0),
			candidates jsonb NOT NULL DEFAULT '[]'::jsonb, associated boolean NOT NULL DEFAULT false,
			status text NOT NULL DEFAULT 'pending' CHECK(status IN ('pending','uncertain','complete','expired')),
			PRIMARY KEY(organization_id,plan_id));
			CREATE UNIQUE INDEX listing_shopify_media_active ON ` + mediaPlanTable + ` (organization_id,shop_key,product_id) WHERE status IN ('pending','uncertain');
			CREATE TABLE ` + mediaStepTable + ` (
			organization_id text NOT NULL, plan_id uuid NOT NULL, cycle int NOT NULL CHECK(cycle>=0),
			phase text NOT NULL CHECK(phase IN ('prepare_shopify_media','upload_shopify_media','append_shopify_media','associate_shopify_media')),
			position int NOT NULL CHECK(position>=0), attempt_id uuid NOT NULL,
			PRIMARY KEY(organization_id,attempt_id),
			UNIQUE(organization_id,plan_id,cycle,phase,position),
			FOREIGN KEY(organization_id,plan_id) REFERENCES ` + mediaPlanTable + `(organization_id,plan_id));`).Error
	})
}

type isolatedMediaJournal struct{ repository *Repository }

func NewIsolatedShopifyMediaJournal(ctx context.Context, db *gorm.DB, live *shopifyguard.LiveIsolation) (submission.ShopifyMediaJournal, error) {
	if err := shopifyguard.VerifyLiveIsolation(db, live); err != nil {
		return nil, err
	}
	var ready bool
	if err := db.WithContext(ctx).Raw("SELECT to_regclass(?) IS NOT NULL AND to_regclass(?) IS NOT NULL", mediaPlanTable, mediaStepTable).Row().Scan(&ready); err != nil || !ready {
		return nil, submission.ErrExecutionUnavailable
	}
	repository, err := NewOfficialEffectsRepository(shopifyguard.RequireIsolated(db))
	if err != nil {
		return nil, err
	}
	return &isolatedMediaJournal{repository: repository}, nil
}

type retainedMediaRun struct {
	hash      string
	tokenHash string
	lease     *time.Time
	run       submission.ShopifyMediaRun
}

func lockedMediaRun(tx *gorm.DB, plan submission.ShopifyMediaPlan) (retainedMediaRun, error) {
	result := retainedMediaRun{run: submission.ShopifyMediaRun{Plan: plan}}
	var candidates []byte
	err := tx.Raw(`SELECT body_hash,token_hash,lease_until,cycle,candidates,associated,status
		FROM `+mediaPlanTable+` WHERE organization_id=? AND plan_id=? FOR UPDATE`, plan.Owner.OrganizationID, plan.ID).
		Row().Scan(&result.hash, &result.tokenHash, &result.lease, &result.run.Cycle, &candidates, &result.run.Associated, &result.run.Status)
	if err != nil {
		return retainedMediaRun{}, err
	}
	if result.hash != plan.Fingerprint() {
		return retainedMediaRun{}, submission.ErrExecutionIntentConflict
	}
	if json.Unmarshal(candidates, &result.run.Candidates) != nil {
		return retainedMediaRun{}, submission.ErrExecutionUnavailable
	}
	return result, nil
}

func (r retainedMediaRun) matches(run submission.ShopifyMediaRun) bool {
	return run.Token != "" && r.run.Cycle == run.Cycle &&
		subtle.ConstantTimeCompare([]byte(r.tokenHash), []byte(submission.ExecutionClaimTokenHash(run.Token))) == 1
}

func (j *isolatedMediaJournal) Begin(ctx context.Context, plan submission.ShopifyMediaPlan) (submission.ShopifyMediaRun, error) {
	if !plan.Valid() {
		return submission.ShopifyMediaRun{}, submission.ErrExecutionInvalid
	}
	raw, err := json.Marshal(plan)
	if err != nil {
		return submission.ShopifyMediaRun{}, submission.ErrExecutionInvalid
	}
	var result submission.ShopifyMediaRun
	err = j.repository.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := shopifyguard.Check(tx, plan.Owner.OrganizationID, plan.ShopKey, plan.Generation, plan.SourceKey); err != nil {
			return err
		}
		if err := expireUnusedMediaRuns(tx, plan); err != nil {
			return err
		}
		insert := tx.Exec("INSERT INTO "+mediaPlanTable+" (organization_id,plan_id,shop_key,product_id,generation,source_key,body_hash,body_json,expires_at) VALUES (?,?,?,?,?,?,?,?::jsonb,?) ON CONFLICT DO NOTHING",
			plan.Owner.OrganizationID, plan.ID, plan.ShopKey, plan.ProductID, plan.Generation, plan.SourceKey, plan.Fingerprint(), string(raw), plan.ExpiresAt)
		if insert.Error != nil {
			return insert.Error
		}
		retained, err := lockedMediaRun(tx, plan)
		if errors.Is(err, sql.ErrNoRows) {
			return submission.ErrExecutionTargetClaimed
		}
		if err != nil {
			return err
		}
		result = retained.run
		now := time.Now()
		if result.Status != submission.ShopifyMediaPending || retained.lease != nil && retained.lease.After(now) {
			return nil
		}
		open, err := openMediaDispatch(tx, plan)
		if err != nil {
			return err
		}
		if open {
			result.Status = submission.ShopifyMediaUncertain
			if err := j.markOpenMediaUnknown(ctx, tx, plan); err != nil {
				return err
			}
			return mediaRunUpdate(tx, plan, "status=?, token_hash='', lease_until=NULL", result.Status)
		}
		if !now.Before(plan.ExpiresAt) && len(result.Candidates) == 0 && !result.Associated {
			return submission.ErrExecutionClaimRejected
		}
		if len(result.Candidates) == 0 && len(plan.Images) > 0 {
			if result.Cycle >= maxMediaCycles {
				result.Status = submission.ShopifyMediaExpired
				return mediaRunUpdate(tx, plan, "status=?, token_hash='', lease_until=NULL", result.Status)
			}
			result.Cycle++
		}
		token, err := uuid.NewRandom()
		if err != nil {
			return err
		}
		result.Token = token.String()
		return mediaRunUpdate(tx, plan, "token_hash=?, lease_until=?, cycle=?", submission.ExecutionClaimTokenHash(result.Token), now.Add(mediaRunLease), result.Cycle)
	})
	if err != nil {
		return submission.ShopifyMediaRun{}, err
	}
	return result, nil
}

func (j *isolatedMediaJournal) Admit(ctx context.Context, run submission.ShopifyMediaRun, phase submission.ShopifyMediaPhase, index int, fingerprint string) (submission.ShopifyMediaDispatch, error) {
	plan := run.Plan
	if !plan.Valid() {
		return submission.ShopifyMediaDispatch{}, submission.ErrExecutionInvalid
	}
	var permit submission.ShopifyMediaDispatch
	err := j.repository.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := shopifyguard.Check(tx, plan.Owner.OrganizationID, plan.ShopKey, plan.Generation, plan.SourceKey); err != nil {
			return err
		}
		retained, err := lockedMediaRun(tx, plan)
		if err != nil {
			return err
		}
		now := time.Now()
		if !retained.matches(run) || retained.lease == nil || !retained.lease.After(now) || !plan.ExpiresAt.After(now) ||
			retained.run.Status != submission.ShopifyMediaPending {
			return submission.ErrExecutionClaimRejected
		}
		if err := validateMediaStep(tx, retained.run, phase, index); err != nil {
			return err
		}
		key := plan.StepIntentKey(phase, run.Cycle, index)
		id, err := uuid.NewV7()
		if err != nil {
			return err
		}
		token, err := uuid.NewRandom()
		if err != nil {
			return err
		}
		reservation, err := submission.NewExecutionReservation(submission.AcquireExecutionCommand{
			Scope: submission.ExecutionScope{OrganizationID: plan.Owner.OrganizationID}, IntentKey: key,
			Target: submission.ExecutionTarget{Platform: "shopify", StoreID: plan.ShopKey, SubjectID: "media-" + key},
			Action: string(phase), Payload: []byte(fingerprint), ClaimOwnerID: plan.Owner.ActorID, Lease: mediaRunLease,
		}, id.String(), token.String(), now)
		if err != nil {
			return err
		}
		attempt := reservation.Attempt
		attempt.FenceEpoch = 1
		row := attemptRowFrom(attempt, reservation.ClaimTokenHash)
		if err := tx.Table(attemptTable).Create(&row).Error; err != nil {
			return err
		}
		fence := targetFenceRow{OrganizationID: attempt.OrganizationID, Platform: attempt.Target.Platform, StoreID: attempt.Target.StoreID,
			SubjectID: attempt.Target.SubjectID, Epoch: attempt.FenceEpoch, CurrentAttemptID: attempt.AttemptID,
			CurrentStatus: string(attempt.Status), UpdatedAt: attempt.UpdatedAt}
		if err := tx.Table(targetTable).Create(&fence).Error; err != nil {
			return err
		}
		if err := tx.Exec("INSERT INTO "+mediaStepTable+" (organization_id,plan_id,cycle,phase,position,attempt_id) VALUES (?,?,?,?,?,?)",
			plan.Owner.OrganizationID, plan.ID, run.Cycle, phase, index, attempt.AttemptID).Error; err != nil {
			return err
		}
		scope := mediaDispatchScope(plan, attempt.AttemptID)
		if err := shopifyguard.AdmitDispatch(tx, scope); err != nil {
			return err
		}
		permit = submission.ShopifyMediaDispatch{AttemptID: attempt.AttemptID, Token: token.String(), Phase: phase, Index: index}
		return nil
	})
	if err != nil {
		return submission.ShopifyMediaDispatch{}, err
	}
	return permit, nil
}

func validateMediaStep(tx *gorm.DB, run submission.ShopifyMediaRun, phase submission.ShopifyMediaPhase, index int) error {
	plan := run.Plan
	open, err := openMediaDispatch(tx, plan)
	if err != nil {
		return err
	}
	if open {
		return submission.ErrExecutionOutcomeUnknown
	}
	if phase == submission.ShopifyMediaAssociate {
		if index != 0 || len(plan.Covers) == 0 || run.Associated || len(plan.Images) > 0 && len(run.Candidates) == 0 {
			return submission.ErrExecutionEvidenceRequired
		}
		return nil
	}
	if len(run.Candidates) != 0 || len(plan.Images) == 0 || run.Cycle < 1 {
		return submission.ErrExecutionEvidenceRequired
	}
	var prepare, upload int
	if err := tx.Raw(`SELECT count(*) FILTER(WHERE phase=?),count(*) FILTER(WHERE phase=?)
		FROM `+mediaStepTable+` WHERE organization_id=? AND plan_id=? AND cycle=?`, submission.ShopifyMediaPrepare,
		submission.ShopifyMediaUpload, plan.Owner.OrganizationID, plan.ID, run.Cycle).Row().Scan(&prepare, &upload); err != nil {
		return err
	}
	switch phase {
	case submission.ShopifyMediaPrepare:
		if index == 0 && prepare == 0 && upload == 0 {
			return nil
		}
	case submission.ShopifyMediaUpload:
		if prepare == 1 && index == upload && index >= 0 && index < len(plan.Images) {
			return nil
		}
	case submission.ShopifyMediaAppend:
		if prepare == 1 && index == 0 && upload == len(plan.Images) {
			return nil
		}
	}
	return submission.ErrExecutionEvidenceRequired
}

func (j *isolatedMediaJournal) Observe(ctx context.Context, run submission.ShopifyMediaRun, permit submission.ShopifyMediaDispatch, receipt submission.ShopifyMediaReceipt) error {
	plan := run.Plan
	digest, candidates, rejected, sealed := receipt.Evidence(permit)
	if !sealed {
		return submission.ErrExecutionEvidenceRequired
	}
	var hidden bool
	err := j.repository.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		guardErr := shopifyguard.Check(tx, plan.Owner.OrganizationID, plan.ShopKey, plan.Generation, plan.SourceKey)
		if guardErr != nil && !errors.Is(guardErr, shopifyguard.ErrFrozen) && !errors.Is(guardErr, shopifyguard.ErrRestricted) {
			return guardErr
		}
		retained, err := lockedMediaRun(tx, plan)
		if err != nil {
			return err
		}
		scope := mediaDispatchScope(plan, permit.AttemptID)
		token := submission.ExecutionClaimTokenHash(permit.Token)
		if err := shopifyguard.ObserveMediaDispatchEnd(tx, scope, plan.ID, string(permit.Phase), plan.Owner.ActorID, token); err != nil {
			return err
		}
		if guardErr != nil {
			hidden = true
			return nil
		}
		if !retained.matches(run) {
			hidden = true
			return mediaRunUpdate(tx, plan, "status=?", submission.ShopifyMediaUncertain)
		}
		attempt, savedToken, err := lockAttemptRow(ctx, tx, plan.Owner.OrganizationID, permit.AttemptID)
		if err != nil || attempt.Action != string(permit.Phase) || subtle.ConstantTimeCompare([]byte(savedToken), []byte(token)) != 1 {
			return submission.ErrExecutionClaimRejected
		}
		if _, err := j.repository.lockCurrentFence(ctx, tx, attempt); err != nil {
			return err
		}
		now := time.Now()
		if !now.Before(attempt.LeaseExpiresAt) {
			hidden = true
			return mediaRunUpdate(tx, plan, "status=?", submission.ShopifyMediaUncertain)
		}
		evidence := submission.ExecutionEvidence{Kind: submission.EvidenceProviderResponse, Outcome: submission.ExecutionSucceeded,
			Reference: attempt.AttemptID, Fingerprint: digest, ObservedAt: now}
		if rejected {
			evidence.Outcome = submission.ExecutionFailedDefinitive
			evidence.Reason = "provider rejected the media request"
		}
		finished, err := submission.CompleteClaimedExecution(attempt, evidence, now)
		if err != nil {
			return err
		}
		if err := j.repository.persistAttemptAndFence(ctx, tx, finished); err != nil {
			return err
		}
		if rejected {
			return mediaRunUpdate(tx, plan, "status=?", submission.ShopifyMediaUncertain)
		}
		switch permit.Phase {
		case submission.ShopifyMediaAppend:
			raw, err := json.Marshal(candidates)
			if err != nil {
				return err
			}
			return mediaRunUpdate(tx, plan, "candidates=?::jsonb", string(raw))
		case submission.ShopifyMediaAssociate:
			return mediaRunUpdate(tx, plan, "associated=true")
		}
		return nil
	})
	if err == nil && hidden {
		return submission.ErrExecutionOutcomeUnknown
	}
	return err
}

func (j *isolatedMediaJournal) End(ctx context.Context, run submission.ShopifyMediaRun, status submission.ShopifyMediaStatus) error {
	plan := run.Plan
	return j.repository.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := shopifyguard.Check(tx, plan.Owner.OrganizationID, plan.ShopKey, plan.Generation, plan.SourceKey); err != nil {
			return err
		}
		saved, err := lockedMediaRun(tx, plan)
		if err != nil {
			return err
		}
		if !saved.matches(run) {
			return submission.ErrExecutionClaimRejected
		}
		open, err := openMediaDispatch(tx, plan)
		if err != nil {
			return err
		}
		if open || saved.run.Status == submission.ShopifyMediaUncertain {
			status = submission.ShopifyMediaUncertain
		}
		if open {
			if err := j.markOpenMediaUnknown(ctx, tx, plan); err != nil {
				return err
			}
		}
		return mediaRunUpdate(tx, plan, "status=?, token_hash='', lease_until=NULL", status)
	})
}

func (j *isolatedMediaJournal) markOpenMediaUnknown(ctx context.Context, tx *gorm.DB, plan submission.ShopifyMediaPlan) error {
	var ids []string
	err := tx.Raw(`SELECT s.attempt_id::text FROM `+mediaStepTable+` s
		JOIN public.listing_shopify_dispatches d USING (organization_id,attempt_id)
		WHERE s.organization_id=? AND s.plan_id=? AND NOT d.response_observed`, plan.Owner.OrganizationID, plan.ID).Scan(&ids).Error
	if err != nil {
		return err
	}
	for _, id := range ids {
		attempt, _, err := lockAttemptRow(ctx, tx, plan.Owner.OrganizationID, id)
		if err != nil {
			return err
		}
		unknown, err := submission.TransitionExecutionToUnknown(attempt, submission.UnknownResponseLost, time.Now())
		if err != nil {
			return err
		}
		if err := j.repository.persistAttemptAndFence(ctx, tx, unknown); err != nil {
			return err
		}
	}
	return nil
}

// Release only confirmations that expired before either product mutation was
// admitted. Closed staging responses alone cannot create gallery entries.
func expireUnusedMediaRuns(tx *gorm.DB, plan submission.ShopifyMediaPlan) error {
	return tx.Exec(`UPDATE `+mediaPlanTable+` p SET status='expired', token_hash='', lease_until=NULL
		WHERE p.organization_id=? AND p.shop_key=? AND p.product_id=? AND p.status='pending'
		AND (p.lease_until IS NULL OR p.lease_until<=CURRENT_TIMESTAMP) AND p.expires_at<=CURRENT_TIMESTAMP
		AND NOT EXISTS (SELECT 1 FROM `+mediaStepTable+` s
			JOIN public.listing_shopify_dispatches d USING (organization_id,attempt_id)
			WHERE s.organization_id=p.organization_id AND s.plan_id=p.plan_id
			AND (NOT d.response_observed OR s.phase IN ('append_shopify_media','associate_shopify_media')))`,
		plan.Owner.OrganizationID, plan.ShopKey, plan.ProductID).Error
}

func mediaDispatchScope(plan submission.ShopifyMediaPlan, attemptID string) shopifyguard.Dispatch {
	return shopifyguard.Dispatch{Organization: plan.Owner.OrganizationID, ShopKey: plan.ShopKey, Generation: plan.Generation, AttemptID: attemptID, Epoch: 1}
}

func openMediaDispatch(tx *gorm.DB, plan submission.ShopifyMediaPlan) (bool, error) {
	var open bool
	err := tx.Raw(`SELECT EXISTS(SELECT 1 FROM `+mediaStepTable+` s
		JOIN public.listing_shopify_dispatches d USING (organization_id,attempt_id)
		WHERE s.organization_id=? AND s.plan_id=? AND NOT d.response_observed)`, plan.Owner.OrganizationID, plan.ID).Row().Scan(&open)
	return open, err
}

func mediaRunUpdate(tx *gorm.DB, plan submission.ShopifyMediaPlan, set string, args ...any) error {
	update := tx.Exec("UPDATE "+mediaPlanTable+" SET "+set+" WHERE organization_id=? AND plan_id=?",
		append(args, plan.Owner.OrganizationID, plan.ID)...)
	if update.Error != nil {
		return update.Error
	}
	if update.RowsAffected != 1 {
		return submission.ErrExecutionClaimRejected
	}
	return nil
}
