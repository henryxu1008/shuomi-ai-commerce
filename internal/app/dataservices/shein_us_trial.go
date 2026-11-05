package dataservicesapp

import (
	"context"
	"database/sql"

	"gorm.io/gorm"

	jobstore "task-processor/internal/integration/persistence/product/dataacquisition"
	"task-processor/internal/product/collection"
	"task-processor/internal/product/dataacquisition"
)

// NewSHEINUSTrial assembles the isolated one-page trial. Serving assemblies do
// not register this constructor, a sender, or the trial schema.
func NewSHEINUSTrial(db *sql.DB, authority dataacquisition.SHEINUSTrialAuthority, sender dataacquisition.SHEINUSTrialSender, lookup dataacquisition.SHEINUSTrialLookup) (*dataacquisition.SHEINUSTrial, error) {
	store, err := jobstore.NewSHEINUSTrialStore(db)
	if err != nil {
		return nil, err
	}
	return dataacquisition.NewSHEINUSTrial(store, authority, sender, lookup)
}

// SHEINUSPageTrial is the internal composition for the isolated page trial.
// No serving route or provider credentials are registered here.
type SHEINUSPageTrial struct {
	runner  *dataacquisition.SHEINUSTrial
	repo    *jobstore.Repository
	publish jobstore.SHEINUSPublisher
}

type sheinUSPageAuthority struct {
	dataacquisition.SHEINUSTrialAuthority
	live dataacquisition.LiveAccess
}

func (a sheinUSPageAuthority) AuthorizeSHEINUSTrial(ctx context.Context, p dataacquisition.Principal, id string, action dataacquisition.SHEINUSTrialAction) (dataacquisition.SHEINUSTrialApproval, error) {
	approval, err := a.SHEINUSTrialAuthority.AuthorizeSHEINUSTrial(ctx, p, id, action)
	if err == nil && (action == dataacquisition.SHEINUSTrialSend || action == dataacquisition.SHEINUSTrialPublish) {
		err = a.live.CheckExecution(ctx, p, approval.Funding)
	}
	return approval, err
}

func NewSHEINUSPageTrial(ctx context.Context, db *gorm.DB, live dataacquisition.LiveAccess, authority dataacquisition.SHEINUSTrialAuthority, sender dataacquisition.SHEINUSTrialSender, lookup dataacquisition.SHEINUSTrialLookup) (*SHEINUSPageTrial, error) {
	if db == nil || live == nil || authority == nil {
		return nil, dataacquisition.ErrUnavailable
	}
	pool, err := db.DB()
	if err != nil {
		return nil, dataacquisition.ErrUnavailable
	}
	runner, err := NewSHEINUSTrial(pool, sheinUSPageAuthority{SHEINUSTrialAuthority: authority, live: live}, sender, lookup)
	if err != nil {
		return nil, err
	}
	publisher, err := NewProductPublisher(live)
	if err != nil {
		return nil, err
	}
	repo, err := jobstore.NewRepository(ctx, db, live, publisher, transactionResultReader)
	if err != nil {
		return nil, err
	}
	return &SHEINUSPageTrial{runner: runner, repo: repo, publish: NewSHEINUSPublisher(live)}, nil
}

func (s *SHEINUSPageTrial) Run(ctx context.Context, p dataacquisition.Principal, id string) (dataacquisition.SHEINUSTrialAttempt, []collection.Source, error) {
	attempt, err := s.runner.Send(ctx, p, id)
	if err != nil || attempt.OverLimit() {
		return attempt, nil, err
	}
	approval, err := s.runner.PublicationApproval(ctx, p, id)
	if err != nil {
		return attempt, nil, err
	}
	refs, err := s.repo.SHEINUSTrialPreparedRefs(ctx, approval)
	if err != nil {
		return attempt, nil, err
	}
	sources := make([]collection.Source, 0, len(refs))
	for _, ref := range refs {
		approval, err = s.runner.PublicationApproval(ctx, p, id)
		if err != nil {
			return attempt, nil, err
		}
		source, err := s.repo.PublishSHEINUS(ctx, p, approval, ref, s.publish)
		if err != nil {
			return attempt, nil, err
		}
		sources = append(sources, source)
	}
	if _, err := s.runner.PublicationApproval(ctx, p, id); err != nil {
		return attempt, nil, err
	}
	return attempt, sources, nil
}

func (s *SHEINUSPageTrial) Reconcile(ctx context.Context, p dataacquisition.Principal, id string) (dataacquisition.SHEINUSTrialAttempt, error) {
	return s.runner.Reconcile(ctx, p, id)
}
