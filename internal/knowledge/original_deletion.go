package knowledge

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"task-processor/internal/ledger/storagecapacity"
)

type OriginalDeleteState string

const (
	OriginalFenced     OriginalDeleteState = "FENCED"
	OriginalDispatched OriginalDeleteState = "DELETE_DISPATCHED"
	OriginalAbsent     OriginalDeleteState = "OBJECT_ABSENCE_PROVEN"
	OriginalReleased   OriginalDeleteState = "RELEASED"
)

type OriginalFile struct {
	SourceID       string                             `json:"sourceId"`
	SourceVersion  int64                              `json:"sourceVersion"`
	RevisionID     string                             `json:"revisionId"`
	RevisionNumber int64                              `json:"revisionNumber"`
	Bytes          int64                              `json:"bytes"`
	CanDelete      bool                               `json:"canDelete"`
	OperationID    string                             `json:"operationId,omitempty"`
	State          OriginalDeleteState                `json:"state,omitempty"`
	Target         storagecapacity.StoredObjectTarget `json:"-"`
}

type OriginalDeletion struct {
	ID, OrganizationID, SourceID, RevisionID, Namespace, ObjectKey, SHA256, ReservationID string
	SourceVersion, RevisionNumber, SizeBytes                                              int64
	State                                                                                 OriginalDeleteState
	PresentDigest, ProofDigest, LeaseOwner                                                string
	LeaseUntil                                                                            *time.Time
}

func (o OriginalDeletion) view() OriginalFile {
	return OriginalFile{SourceID: o.SourceID, SourceVersion: o.SourceVersion, RevisionID: o.RevisionID, RevisionNumber: o.RevisionNumber, Bytes: o.SizeBytes, OperationID: o.ID, State: o.State}
}

func (o OriginalDeletion) object() Object {
	return Object{Key: o.ObjectKey, SHA256: o.SHA256, SizeBytes: o.SizeBytes}
}

func (o OriginalDeletion) target() storagecapacity.StoredObjectTarget {
	return storagecapacity.StoredObjectTarget{OrganizationID: o.OrganizationID, ReservationID: o.ReservationID, Object: storagecapacity.Object{
		Namespace: o.Namespace, Key: o.ObjectKey, SHA256: o.SHA256, SizeBytes: o.SizeBytes, Category: "knowledge_document", ExclusiveBinding: true,
	}}
}

// OriginalRepository is the existing Knowledge owner with V1 upload and delete facts.
type OriginalRepository interface {
	BeginOriginalPut(context.Context, Revision, string) (bool, error)
	ConfirmOriginalPut(context.Context, Revision, string) error
	OriginalFile(context.Context, Scope, string) (OriginalFile, error)
	FenceOriginalDeletion(context.Context, Scope, string, string, string, int64, string) (OriginalDeletion, error)
	ClaimOriginalDeletion(context.Context, Scope, string, string, string) (OriginalDeletion, bool, error)
	AdvanceOriginalDeletion(context.Context, OriginalDeletion, string, OriginalDeleteState, string) error
	FinishOriginalDeletionClaim(context.Context, OriginalDeletion) error
}

type SinglePutStore interface {
	PutImmutableOnce(context.Context, Object) error
}

type OriginalObjectDeleter interface {
	VerifyNamespace(context.Context, string) error
	// InspectOriginal establishes presence or authoritative absence (including versions)
	// only inside the pinned, admitted physical namespace.
	InspectOriginal(context.Context, string, Object) (Inspection, error)
	DeleteOriginal(context.Context, string, Object) error
}

type OriginalDeleteAuthorization func(context.Context) error

type originalDeletionService struct {
	repo     OriginalRepository
	store    OriginalObjectDeleter
	ledger   storagecapacity.StoredReleaser
	allowNew bool
}

func (s *Service) ConfigureOriginalDeletion(objects OriginalObjectDeleter, allowNew bool) error {
	repo, _ := s.repo.(OriginalRepository)
	if !s.StorageCapacityConfigured() || repo == nil || objects == nil {
		return ErrUnavailable
	}
	ledger, ok := s.capacity.Writer.(storagecapacity.StoredReleaser)
	if !ok {
		return ErrUnavailable
	}
	if _, ok = s.objects.(SinglePutStore); !ok {
		return ErrUnavailable
	}
	s.original = &originalDeletionService{repo: repo, store: objects, ledger: ledger, allowNew: allowNew}
	return nil
}

func (s *Service) OriginalDeletionConfigured() bool { return s != nil && s.original != nil }

func (s *Service) OriginalFile(ctx context.Context, scope Scope, source string) (OriginalFile, error) {
	if !validScope(scope) || !ValidID(source) {
		return OriginalFile{}, ErrInvalid
	}
	if s.original == nil {
		return OriginalFile{}, ErrNotFound
	}
	v, err := s.original.repo.OriginalFile(ctx, scope, source)
	if err == nil && v.CanDelete {
		v.Target.Object.Namespace = s.capacity.Namespace
		v.CanDelete = s.original.allowNew && s.original.ledger.CheckStoredReleasable(ctx, v.Target) == nil
	}
	return v, err
}

func (s *Service) DeleteOriginal(ctx context.Context, scope Scope, source, revision, key string, version int64, authorize OriginalDeleteAuthorization) (OriginalFile, error) {
	if !validScope(scope) || !ValidID(source) || !ValidID(revision) || !ValidID(key) || version < 1 {
		return OriginalFile{}, ErrInvalid
	}
	if s.original == nil || authorize == nil {
		return OriginalFile{}, ErrNotFound
	}
	namespace := ""
	if s.original.allowNew {
		namespace = s.capacity.Namespace
	}
	op, err := s.original.repo.FenceOriginalDeletion(ctx, scope, source, revision, key, version, namespace)
	if err != nil {
		return OriginalFile{}, err
	}
	return s.ContinueOriginalDeletion(ctx, scope, source, op.ID, authorize)
}

func (s *Service) ContinueOriginalDeletion(ctx context.Context, scope Scope, source, operation string, authorize OriginalDeleteAuthorization) (OriginalFile, error) {
	if !validScope(scope) || !ValidID(source) || !ValidID(operation) {
		return OriginalFile{}, ErrInvalid
	}
	if s.original == nil || authorize == nil {
		return OriginalFile{}, ErrNotFound
	}
	d := s.original
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	op, claimed, err := d.repo.ClaimOriginalDeletion(ctx, scope, source, operation, uuid.NewString())
	if err != nil || !claimed {
		return op.view(), err
	}
	defer func() {
		cleanup, stop := context.WithTimeout(context.WithoutCancel(ctx), 2*time.Second)
		defer stop()
		_ = d.repo.FinishOriginalDeletionClaim(cleanup, op)
	}()
	// A storage replacement cannot authorize a new dispatch or a release.
	if op.Namespace != s.capacity.Namespace {
		return op.view(), nil
	}
	if op.State == OriginalFenced || op.State == OriginalDispatched {
		object := op.object()
		seen, readErr := d.store.InspectOriginal(ctx, op.Namespace, object)
		if readErr != nil {
			return op.view(), nil
		}
		if seen.Exists {
			if !matches(object, seen) || d.ledger.CheckStoredReleasable(ctx, op.target()) != nil {
				return op.view(), nil
			}
			if err = authorize(ctx); err != nil {
				return OriginalFile{}, err
			}
			present := Digest([]byte(fmt.Sprintf("knowledge-present-v1\x00%s\x00%s\x00%s\x00%s\x00%d", op.ID, op.Namespace, op.ObjectKey, op.SHA256, op.SizeBytes)))
			if err = d.repo.AdvanceOriginalDeletion(ctx, op, scope.ActorID, OriginalDispatched, present); err != nil {
				return op.view(), err
			}
			op.State, op.PresentDigest = OriginalDispatched, present
			// The subsequent authenticated readback, not the DELETE response, is authoritative.
			_ = d.store.DeleteOriginal(ctx, op.Namespace, object)
			seen, readErr = d.store.InspectOriginal(ctx, op.Namespace, object)
			if readErr != nil || seen.Exists {
				return op.view(), nil
			}
		}
		if op.State != OriginalDispatched || !storagecapacity.ValidDigest(op.PresentDigest) {
			return op.view(), nil
		}
		proof := Digest([]byte("knowledge-absence-v1\x00" + op.PresentDigest + "\x00head-missing-and-no-versions"))
		if err = d.repo.AdvanceOriginalDeletion(ctx, op, scope.ActorID, OriginalAbsent, proof); err != nil {
			return op.view(), err
		}
		op.State, op.ProofDigest = OriginalAbsent, proof
	}
	if op.State != OriginalAbsent || d.store.VerifyNamespace(ctx, op.Namespace) != nil {
		return op.view(), nil
	}
	receipt, err := d.ledger.ReleaseStored(ctx, storagecapacity.StoredDeletion{Target: op.target(), OperationID: op.ID, ProofDigest: op.ProofDigest})
	if err != nil {
		return op.view(), nil
	}
	if receipt.OperationID != op.ID || receipt.Bytes != op.SizeBytes || receipt.ReleasedAt.IsZero() {
		return op.view(), ErrIntegrity
	}
	if err = d.repo.AdvanceOriginalDeletion(ctx, op, scope.ActorID, OriginalReleased, receipt.ReleasedAt.UTC().Format(time.RFC3339Nano)); err != nil {
		return op.view(), err
	}
	op.State = OriginalReleased
	return op.view(), nil
}

func (s *Service) putOriginal(ctx context.Context, revision Revision, object Object, reservation string) (bool, error) {
	if revision.OriginalGeneration == "" {
		return false, s.objects.PutImmutable(ctx, object)
	}
	if s.original == nil {
		return false, ErrUnavailable
	}
	first, err := s.original.repo.BeginOriginalPut(ctx, revision, reservation)
	if err != nil {
		return false, err
	}
	if !first {
		return false, s.objects.PutImmutable(ctx, object)
	}
	store := s.objects.(SinglePutStore)
	return true, store.PutImmutableOnce(ctx, object)
}

func (s *Service) confirmOriginalPut(ctx context.Context, revision Revision, reservation string) error {
	if s.original == nil {
		return ErrUnavailable
	}
	err := s.original.repo.ConfirmOriginalPut(ctx, revision, reservation)
	if errors.Is(err, ErrLeaseLost) {
		return nil
	} // a later send made this version permanently ineligible
	return err
}
