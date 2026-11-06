package dataacquisition

import (
	"context"
	"task-processor/internal/product/collection"
)

// ReadRepository preserves the existing credential-guarded, single-transaction
// result read without exposing admission, cancellation or execution writes.
type ReadRepository interface {
	ResultReadRepository
	WithResultRead(context.Context, Principal, func(context.Context, ResultReadRepository, CapturedResultReader) error) error
}

type Reader struct {
	repo ReadRepository
	live LiveAccess
}

func NewReader(repo ReadRepository, live LiveAccess) (*Reader, error) {
	if repo == nil || live == nil {
		return nil, ErrUnavailable
	}
	return &Reader{repo: repo, live: live}, nil
}

type jobRead func(context.Context, Principal, string, ResultReadRepository) (Job, error)

func (r *Reader) Read(ctx context.Context, p Principal, id string) (Job, error) {
	return r.readWith(ctx, p, id, r.read)
}

func (r *Reader) readWith(ctx context.Context, p Principal, id string, read jobRead) (Job, error) {
	if p.Scope.Validate() != nil || !collection.ValidID(id) {
		return Job{}, ErrInvalid
	}
	if p.CredentialID != "" {
		var job Job
		err := r.repo.WithResultRead(ctx, p, func(ctx context.Context, repo ResultReadRepository, _ CapturedResultReader) error {
			var err error
			job, err = read(ctx, p, id, repo)
			return err
		})
		if err != nil {
			return Job{}, err
		}
		return job, nil
	}
	return read(ctx, p, id, r.repo)
}

func (r *Reader) read(ctx context.Context, p Principal, id string, repo ResultReadRepository) (Job, error) {
	if err := r.live.CheckRead(ctx, p); err != nil {
		return Job{}, err
	}
	job, err := repo.Read(ctx, p.Scope, id)
	if err != nil {
		return Job{}, err
	}
	if p.CredentialID != "" && job.CredentialID != p.CredentialID {
		return Job{}, ErrNotFound
	}
	return job, nil
}
