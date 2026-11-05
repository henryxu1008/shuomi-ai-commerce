package knowledge

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	k "task-processor/internal/knowledge"
	"task-processor/internal/ledger/storagecapacity"
)

var _ k.OriginalRepository = (*Repository)(nil)

// ConfigureOriginalGeneration checks the deployment owner's durable activation.
// An active cohort must remain guarded when the new-delete switch is off.
func (r *Repository) ConfigureOriginalGeneration(ctx context.Context, generation string) error {
	var exists bool
	if err := r.db.WithContext(ctx).Raw("SELECT to_regclass('public.knowledge_original_generation') IS NOT NULL").Scan(&exists).Error; err != nil {
		return k.ErrUnavailable
	}
	if !exists {
		if generation != "" {
			return k.ErrUnavailable
		}
		return nil
	}
	var saved string
	if err := r.db.WithContext(ctx).Raw("SELECT COALESCE((SELECT generation::text FROM public.knowledge_original_generation WHERE singleton),'')").Scan(&saved).Error; err != nil {
		return k.ErrUnavailable
	}
	if saved != generation || generation != "" && !k.ValidID(generation) {
		return k.ErrUnavailable
	}
	if generation != "" {
		for _, table := range []string{"knowledge_original_uploads", "knowledge_original_deletions", "knowledge_original_delete_keys"} {
			if err := r.db.WithContext(ctx).Exec("SELECT 1 FROM public." + table + " LIMIT 0").Error; err != nil {
				return k.ErrUnavailable
			}
		}
	}
	r.originalGeneration = generation
	return nil
}

func (r *Repository) markOriginalRevision(tx *gorm.DB, rev *k.Revision) error {
	if r.originalGeneration == "" {
		return nil
	}
	if err := tx.Exec("INSERT INTO public.knowledge_original_uploads(organization_id,revision_id,generation,state) VALUES(?,?,?,'READY')", rev.OrganizationID, rev.ID, r.originalGeneration).Error; err != nil {
		return err
	}
	rev.OriginalGeneration = r.originalGeneration
	return nil
}

func (r *Repository) readOriginalRevision(tx *gorm.DB, rev *k.Revision) error {
	if r.originalGeneration == "" {
		return nil
	}
	var found struct {
		Generation string
		Fenced     bool
	}
	if err := tx.Raw(`SELECT u.generation, EXISTS (SELECT 1 FROM public.knowledge_original_deletions d
		WHERE d.organization_id=u.organization_id AND d.revision_id=u.revision_id) AS fenced
		FROM public.knowledge_original_uploads u WHERE u.organization_id=? AND u.revision_id=?`, rev.OrganizationID, rev.ID).Scan(&found).Error; err != nil {
		return err
	}
	rev.OriginalGeneration, rev.OriginalDeletionFenced = found.Generation, found.Fenced
	return nil
}

func lockOriginalSource(tx *gorm.DB, org, source string, mode string) (k.Base, k.Source, error) {
	baseID, err := sourceBase(tx, org, source)
	if err != nil {
		return k.Base{}, k.Source{}, err
	}
	base, err := lockBase(tx, org, baseID, mode)
	if err != nil {
		return base, k.Source{}, err
	}
	s, err := lockSource(tx, org, source, mode)
	return base, s, err
}

func (r *Repository) BeginOriginalPut(ctx context.Context, rev k.Revision, reservation string) (first bool, err error) {
	err = r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		base, source, err := lockOriginalSource(tx, rev.OrganizationID, rev.SourceID, "UPDATE")
		if err != nil {
			return err
		}
		if base.State != k.Active || source.State != k.Active {
			return k.ErrInactive
		}
		var current k.Revision
		if err = one(tx, "SELECT * FROM public.knowledge_revisions WHERE organization_id=? AND source_id=? AND id=? FOR UPDATE", &current, rev.OrganizationID, rev.SourceID, rev.ID); err != nil {
			return err
		}
		now, err := databaseNow(tx)
		if err != nil {
			return err
		}
		if current.State != k.Admitted || current.LeaseOwner != rev.LeaseOwner || current.LeaseUntil == nil || !current.LeaseUntil.After(now) {
			return k.ErrLeaseLost
		}
		if err = r.readOriginalRevision(tx, &current); err != nil {
			return err
		}
		if current.OriginalGeneration != r.originalGeneration || current.OriginalGeneration != rev.OriginalGeneration || current.OriginalGeneration == "" || current.OriginalDeletionFenced {
			return k.ErrConflict
		}
		var upload struct{ State, ReservationID string }
		if err = one(tx, "SELECT state,reservation_id FROM public.knowledge_original_uploads WHERE organization_id=? AND revision_id=? FOR UPDATE", &upload, rev.OrganizationID, rev.ID); err != nil {
			return err
		}
		if upload.ReservationID != "" && upload.ReservationID != reservation {
			return k.ErrConflict
		}
		first = upload.State == "READY"
		state := "INELIGIBLE"
		if first {
			state = "IN_FLIGHT"
		}
		return tx.Exec("UPDATE public.knowledge_original_uploads SET state=?,put_token=?,reservation_id=? WHERE organization_id=? AND revision_id=?", state, rev.LeaseOwner, reservation, rev.OrganizationID, rev.ID).Error
	})
	return first, safe(err)
}

func (r *Repository) ConfirmOriginalPut(ctx context.Context, rev k.Revision, reservation string) error {
	// The same send token can confirm only IN_FLIGHT. A second dispatch changes
	// the state under the original write locks before it can send.
	res := r.db.WithContext(ctx).Exec(`UPDATE public.knowledge_original_uploads SET state='CONFIRMED'
		WHERE organization_id=? AND revision_id=? AND generation=? AND state='IN_FLIGHT'
		AND put_token=? AND reservation_id=?`, rev.OrganizationID, rev.ID, rev.OriginalGeneration, rev.LeaseOwner, reservation)
	if res.Error != nil {
		return safe(res.Error)
	}
	if res.RowsAffected != 1 {
		return k.ErrLeaseLost
	}
	return nil
}

func (r *Repository) originalEligible(tx *gorm.DB, base k.Base, s k.Source, revision string, mode string) (rev k.Revision, reservation string, err error) {
	if s.State != k.Disabled || s.LatestRevisionID == "" || revision != s.LatestRevisionID {
		return rev, "", k.ErrConflict
	}
	if err = one(tx, "SELECT * FROM public.knowledge_revisions WHERE organization_id=? AND source_id=? AND id=? FOR "+mode, &rev, s.OrganizationID, s.ID, revision); err != nil {
		return
	}
	if (rev.State != k.Available && rev.State != k.Partial) || rev.LeaseOwner != "" || rev.LeaseUntil != nil {
		return rev, "", k.ErrConflict
	}
	var denied bool
	if err = tx.Raw(`SELECT
		(SELECT count(*) FROM public.knowledge_revisions WHERE organization_id=? AND source_id=?)<>1
		OR EXISTS(SELECT 1 FROM public.knowledge_context_bundle_entries WHERE organization_id=? AND revision_id=?)`, s.OrganizationID, s.ID, s.OrganizationID, revision).Scan(&denied).Error; err != nil {
		return
	}
	now, err := databaseNow(tx)
	if err != nil {
		return rev, "", err
	}
	busy, err := livePermit(tx, s.OrganizationID, base.ID, s.ID, now)
	if err != nil {
		return rev, "", err
	}
	if denied || busy {
		return rev, "", k.ErrConflict
	}
	var proof struct{ ReservationID string }
	err = one(tx, "SELECT reservation_id FROM public.knowledge_original_uploads WHERE organization_id=? AND revision_id=? AND generation=? AND state='CONFIRMED' FOR "+mode, &proof, s.OrganizationID, revision, r.originalGeneration)
	if err != nil {
		return
	}
	if !k.ValidID(proof.ReservationID) {
		return rev, "", k.ErrConflict
	}
	return rev, proof.ReservationID, nil
}

func originalView(op k.OriginalDeletion) k.OriginalFile {
	return k.OriginalFile{SourceID: op.SourceID, SourceVersion: op.SourceVersion, RevisionID: op.RevisionID, RevisionNumber: op.RevisionNumber, Bytes: op.SizeBytes, OperationID: op.ID, State: op.State}
}

func (r *Repository) OriginalFile(ctx context.Context, scope k.Scope, source string) (view k.OriginalFile, err error) {
	err = r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		base, s, err := lockOriginalSource(tx, scope.OrganizationID, source, "SHARE")
		if err != nil {
			return err
		}
		var op k.OriginalDeletion
		err = one(tx, "SELECT * FROM public.knowledge_original_deletions WHERE organization_id=? AND source_id=?", &op, scope.OrganizationID, source)
		if err == nil {
			view = originalView(op)
			return nil
		}
		if !errors.Is(err, k.ErrNotFound) {
			return err
		}
		rev, reservation, err := r.originalEligible(tx, base, s, s.LatestRevisionID, "SHARE")
		if errors.Is(err, k.ErrConflict) {
			return k.ErrNotFound
		}
		if err != nil {
			return err
		}
		view = k.OriginalFile{SourceID: s.ID, SourceVersion: s.Version, RevisionID: rev.ID, RevisionNumber: rev.Number, Bytes: rev.SizeBytes, CanDelete: true,
			Target: storagecapacity.StoredObjectTarget{OrganizationID: s.OrganizationID, ReservationID: reservation, Object: storagecapacity.Object{
				Key: rev.ObjectKey, SHA256: rev.SHA256, SizeBytes: rev.SizeBytes, Category: "knowledge_document", ExclusiveBinding: true,
			}},
		}
		return nil
	})
	return view, safe(err)
}

func (r *Repository) FenceOriginalDeletion(ctx context.Context, scope k.Scope, source, revision, key string, version int64, namespace string) (op k.OriginalDeletion, err error) {
	err = r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		org := scope.OrganizationID
		if err := tx.Exec("SELECT pg_advisory_xact_lock(hashtextextended(?,0))", org+":original-delete:"+key).Error; err != nil {
			return err
		}
		err := one(tx, `SELECT d.* FROM public.knowledge_original_delete_keys c JOIN public.knowledge_original_deletions d ON d.organization_id=c.organization_id AND d.id=c.operation_id
			WHERE c.organization_id=? AND c.key=?`, &op, org, key)
		if err == nil {
			if op.SourceID != source || op.RevisionID != revision || op.SourceVersion != version {
				return k.ErrConflict
			}
			return nil
		}
		if !errors.Is(err, k.ErrNotFound) {
			return err
		}
		base, s, err := lockOriginalSource(tx, org, source, "UPDATE")
		if err != nil {
			return err
		}
		err = one(tx, "SELECT * FROM public.knowledge_original_deletions WHERE organization_id=? AND source_id=?", &op, org, source)
		if err == nil {
			if op.RevisionID != revision || op.SourceVersion != version {
				return k.ErrConflict
			}
		} else {
			if !errors.Is(err, k.ErrNotFound) {
				return err
			}
			if namespace == "" || s.Version != version {
				return k.ErrConflict
			}
			rev, reservation, err := r.originalEligible(tx, base, s, revision, "UPDATE")
			if err != nil {
				return err
			}
			op = k.OriginalDeletion{ID: uuid.NewString(), OrganizationID: org, SourceID: s.ID, RevisionID: rev.ID, Namespace: namespace, ObjectKey: rev.ObjectKey, SHA256: rev.SHA256,
				ReservationID: reservation, SourceVersion: s.Version, RevisionNumber: rev.Number, SizeBytes: rev.SizeBytes, State: k.OriginalFenced}
			if err = tx.Exec(`INSERT INTO public.knowledge_original_deletions(organization_id,id,source_id,revision_id,namespace,object_key,sha256,reservation_id,
				source_version,revision_number,size_bytes,state,created_by,last_actor,created_at,updated_at) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,clock_timestamp(),clock_timestamp())`,
				org, op.ID, s.ID, rev.ID, op.Namespace, op.ObjectKey, op.SHA256, op.ReservationID, op.SourceVersion, op.RevisionNumber, op.SizeBytes, op.State, scope.ActorID, scope.ActorID).Error; err != nil {
				return err
			}
		}
		return tx.Exec("INSERT INTO public.knowledge_original_delete_keys(organization_id,key,operation_id) VALUES(?,?,?)", org, key, op.ID).Error
	})
	return op, safe(err)
}

func (r *Repository) ClaimOriginalDeletion(ctx context.Context, scope k.Scope, source, operation, lease string) (op k.OriginalDeletion, claimed bool, err error) {
	err = r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := one(tx, "SELECT * FROM public.knowledge_original_deletions WHERE organization_id=? AND source_id=? AND id=? FOR UPDATE", &op, scope.OrganizationID, source, operation); err != nil {
			return err
		}
		now, err := databaseNow(tx)
		if err != nil {
			return err
		}
		if op.State == k.OriginalReleased || op.LeaseUntil != nil && op.LeaseUntil.After(now) {
			return nil
		}
		if op.State == k.OriginalFenced || op.State == k.OriginalDispatched {
			// Source is permanently disabled and the fence was committed at creation.
			// All reference/PUT admissions use the original source lifecycle.
			if op.PresentDigest != "" && op.State == k.OriginalFenced {
				return k.ErrConflict
			}
		}
		until := now.Add(40 * time.Second)
		if err = tx.Exec("UPDATE public.knowledge_original_deletions SET lease_owner=?,lease_until=?,last_actor=?,updated_at=clock_timestamp() WHERE organization_id=? AND id=?", lease, until, scope.ActorID, op.OrganizationID, op.ID).Error; err != nil {
			return err
		}
		op.LeaseOwner, op.LeaseUntil, claimed = lease, &until, true
		return nil
	})
	return op, claimed, safe(err)
}

func (r *Repository) AdvanceOriginalDeletion(ctx context.Context, op k.OriginalDeletion, actor string, next k.OriginalDeleteState, evidence string) error {
	var statement string
	switch {
	case next == k.OriginalDispatched && (op.State == k.OriginalFenced || op.State == k.OriginalDispatched) && storagecapacity.ValidDigest(evidence):
		if op.State == k.OriginalDispatched && op.PresentDigest != evidence {
			return k.ErrConflict
		}
		statement = "present_digest=?"
	case next == k.OriginalAbsent && op.State == k.OriginalDispatched && storagecapacity.ValidDigest(evidence):
		statement = "proof_digest=?"
	case next == k.OriginalReleased && op.State == k.OriginalAbsent:
		if _, err := time.Parse(time.RFC3339Nano, evidence); err != nil {
			return k.ErrInvalid
		}
		statement = "released_at=?::timestamptz"
	default:
		return k.ErrInvalid
	}
	res := r.db.WithContext(ctx).Exec("UPDATE public.knowledge_original_deletions SET "+statement+",state=?,last_actor=?,updated_at=clock_timestamp() WHERE organization_id=? AND id=? AND state=? AND lease_owner=? AND lease_until>clock_timestamp()", evidence, next, actor, op.OrganizationID, op.ID, op.State, op.LeaseOwner)
	if res.Error != nil {
		return safe(res.Error)
	}
	if res.RowsAffected != 1 {
		return k.ErrLeaseLost
	}
	return nil
}

func (r *Repository) FinishOriginalDeletionClaim(ctx context.Context, op k.OriginalDeletion) error {
	return safe(r.db.WithContext(ctx).Exec("UPDATE public.knowledge_original_deletions SET lease_owner='',lease_until=NULL WHERE organization_id=? AND id=? AND lease_owner=?", op.OrganizationID, op.ID, op.LeaseOwner).Error)
}
