package storecenterapp

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/storecenter"
	google "github.com/henryxu1008/shuomi-ai-commerce/internal/storecenter/googlemerchant"
)

// GoogleCredentialRef binds a revision to the current request. It has no
// credential contents and is separate from the durable publication binding.
type GoogleCredentialRef struct {
	snapshot storecenter.GoogleCredentialSnapshot
}

const googleRefreshSaveBudget = 4 * time.Second

func (s *GoogleConnections) credentialRequest(org, storeID string, binding GooglePublicationBinding) storecenter.GoogleCredentialRequest {
	return storecenter.GoogleCredentialRequest{OrganizationID: org, StoreID: storeID, Target: binding.Target,
		Generation: binding.Generation, Application: s.registration.Application}
}

func (s *GoogleConnections) AcquirePublicationCredential(ctx context.Context, org, storeID string, binding GooglePublicationBinding,
	authorize func(context.Context) error,
) (GoogleCredentialRef, error) {
	request := s.credentialRequest(org, storeID, binding)
	for {
		claimID := uuid.NewString()
		if err := authorize(ctx); err != nil {
			return GoogleCredentialRef{}, err
		}
		claimCtx, cancel := context.WithTimeout(ctx, storecenter.GoogleRefreshClaimWindow)
		state, err := s.repo.GoogleCredential(claimCtx, request, claimID)
		cancel()
		if err != nil {
			// A failed transaction acknowledgement cannot authorize a POST.
			if state.ClaimID == claimID {
				s.releaseUnsent(ctx, request, state, authorize)
			}
			return GoogleCredentialRef{}, googleCredentialError(err)
		}
		switch {
		case state.Usable:
			return s.usableCredentialRef(state)
		case state.Phase == storecenter.GoogleRefreshReauthorize:
			return GoogleCredentialRef{}, storecenter.ErrGoogleReauthorization
		case state.ClaimID == claimID:
			return s.refreshPublicationCredential(ctx, request, state, authorize)
		case state.Phase == storecenter.GoogleRefreshIdle:
			return GoogleCredentialRef{}, storecenter.ErrGoogleRefreshUnavailable
		}
		timer := time.NewTimer(150 * time.Millisecond)
		select {
		case <-ctx.Done():
			timer.Stop()
			return GoogleCredentialRef{}, storecenter.ErrGoogleRefreshUnavailable
		case <-timer.C:
		}
	}
}

func (s *GoogleConnections) refreshPublicationCredential(ctx context.Context, request storecenter.GoogleCredentialRequest,
	own storecenter.GoogleCredentialState, authorize func(context.Context) error,
) (GoogleCredentialRef, error) {
	claimCtx, cancel := context.WithDeadline(ctx, own.Deadline)
	defer cancel()
	previous, err := s.openGrant(own.Grant)
	if err != nil || previous.RefreshToken == "" {
		// A decryption error cannot establish that the saved token is invalid.
		if err == nil || errors.Is(err, google.ErrGrantRejected) {
			return s.saveGoogleRefresh(ctx, request, own, storecenter.GoogleRefreshOutcome{Reauthorize: true}, authorize)
		}
		s.releaseUnsent(ctx, request, own, authorize)
		return GoogleCredentialRef{}, storecenter.ErrGoogleRefreshUnavailable
	}
	if err := authorize(claimCtx); err != nil {
		s.releaseUnsent(ctx, request, own, authorize)
		return GoogleCredentialRef{}, err
	}
	// Keep the original local bound as well as the database deadline. A late
	// commit acknowledgement can only shorten the remaining send window.
	permitStarted := time.Now()
	deadline, err := s.repo.AuthorizeGoogleRefresh(claimCtx, request, own)
	if err != nil {
		s.releaseUnsent(ctx, request, own, authorize)
		return GoogleCredentialRef{}, googleCredentialError(err)
	}
	own.Phase, own.Deadline = storecenter.GoogleRefreshPossible, deadline
	sendDeadline := deadline.Add(-storecenter.GoogleRefreshSaveReserve)
	localDeadline := permitStarted.Add(storecenter.GoogleRefreshPermitWindow - storecenter.GoogleRefreshSaveReserve)
	if localDeadline.Before(sendDeadline) {
		sendDeadline = localDeadline
	}
	sendCtx, stop := context.WithDeadline(ctx, sendDeadline)
	if sendCtx.Err() != nil {
		stop()
		s.releaseUnsent(ctx, request, own, authorize)
		return GoogleCredentialRef{}, storecenter.ErrGoogleRefreshUnavailable
	}
	tokens, refreshErr := s.registration.OAuth.Refresh(sendCtx, previous)
	stop()
	var retry google.RefreshRetry
	switch {
	case errors.Is(refreshErr, google.ErrInvalidGrantInput), errors.Is(refreshErr, google.ErrGrantRejected):
		return s.saveGoogleRefresh(ctx, request, own, storecenter.GoogleRefreshOutcome{Reauthorize: true}, authorize)
	case errors.As(refreshErr, &retry):
		return s.saveGoogleRefresh(ctx, request, own, storecenter.GoogleRefreshOutcome{Cooldown: retry.After}, authorize)
	case errors.Is(refreshErr, google.ErrRefreshNotSent):
		s.releaseUnsent(ctx, request, own, authorize)
		return GoogleCredentialRef{}, storecenter.ErrGoogleRefreshUnavailable
	case refreshErr != nil || !time.Now().Add(storecenter.GoogleAccessHeadroom).Before(tokens.ExpiresAt):
		// The endpoint may have rotated the token. Leave the durable possible
		// send in place until a qualified caller observes its expiry.
		return GoogleCredentialRef{}, storecenter.ErrGoogleRefreshUnavailable
	}
	key, ciphertext, err := s.registration.Protection.Seal(own.Snapshot.Binding, tokens)
	if err != nil {
		return GoogleCredentialRef{}, storecenter.ErrGoogleRefreshUnavailable
	}
	return s.saveGoogleRefresh(ctx, request, own, storecenter.GoogleRefreshOutcome{
		KeyID: key, Ciphertext: ciphertext, AccessExpiresAt: tokens.ExpiresAt,
	}, authorize)
}

func (s *GoogleConnections) releaseUnsent(ctx context.Context, request storecenter.GoogleCredentialRequest,
	own storecenter.GoogleCredentialState, authorize func(context.Context) error,
) {
	finish, cancel := context.WithTimeout(context.WithoutCancel(ctx), googleRefreshSaveBudget)
	defer cancel()
	if authorize(finish) != nil {
		return
	}
	current, err := s.repo.GoogleCredential(finish, request, "")
	if err == nil && current.Snapshot == own.Snapshot && current.ClaimID == own.ClaimID &&
		(current.Phase == storecenter.GoogleRefreshClaimed || current.Phase == storecenter.GoogleRefreshPossible) {
		_ = s.repo.FinishGoogleRefresh(finish, request, current, storecenter.GoogleRefreshOutcome{})
	}
}

func (s *GoogleConnections) saveGoogleRefresh(ctx context.Context, request storecenter.GoogleCredentialRequest,
	own storecenter.GoogleCredentialState, result storecenter.GoogleRefreshOutcome, authorize func(context.Context) error,
) (GoogleCredentialRef, error) {
	finish, cancel := context.WithTimeout(context.WithoutCancel(ctx), googleRefreshSaveBudget)
	defer cancel()
	if err := authorize(finish); err != nil {
		return GoogleCredentialRef{}, err
	}
	saveErr := s.repo.FinishGoogleRefresh(finish, request, own, result)
	if saveErr != nil {
		// A commit error can conceal a successful write. Read the owner
		// database first, and only retry saving the same in-memory response
		// while this request still owns the original claim.
		if err := authorize(finish); err != nil {
			return GoogleCredentialRef{}, err
		}
		current, err := s.repo.GoogleCredential(finish, request, "")
		if err != nil {
			return GoogleCredentialRef{}, googleCredentialError(err)
		}
		if !current.Snapshot.SameAuthorization(own.Snapshot) {
			return GoogleCredentialRef{}, storecenter.ErrVersionConflict
		}
		if current.Snapshot == own.Snapshot && current.Phase == own.Phase && current.ClaimID == own.ClaimID {
			if err := authorize(finish); err != nil {
				return GoogleCredentialRef{}, err
			}
			_ = s.repo.FinishGoogleRefresh(finish, request, own, result)
		}
	}
	if result.Ciphertext == "" && saveErr == nil {
		if result.Reauthorize {
			return GoogleCredentialRef{}, storecenter.ErrGoogleReauthorization
		}
		return GoogleCredentialRef{}, storecenter.ErrGoogleRefreshUnavailable
	}
	if ctx.Err() != nil {
		return GoogleCredentialRef{}, storecenter.ErrGoogleRefreshUnavailable
	}
	if err := authorize(finish); err != nil {
		return GoogleCredentialRef{}, err
	}
	current, err := s.repo.GoogleCredential(finish, request, "")
	if err != nil {
		return GoogleCredentialRef{}, googleCredentialError(err)
	}
	if !current.Snapshot.SameAuthorization(own.Snapshot) {
		return GoogleCredentialRef{}, storecenter.ErrVersionConflict
	}
	if current.Phase == storecenter.GoogleRefreshReauthorize {
		return GoogleCredentialRef{}, storecenter.ErrGoogleReauthorization
	}
	if !current.Usable || current.Snapshot.Revision <= own.Snapshot.Revision || ctx.Err() != nil {
		return GoogleCredentialRef{}, storecenter.ErrGoogleRefreshUnavailable
	}
	return s.usableCredentialRef(current)
}

func (s *GoogleConnections) usableCredentialRef(state storecenter.GoogleCredentialState) (GoogleCredentialRef, error) {
	if _, err := s.openUsableGrant(state.Grant); err != nil {
		return GoogleCredentialRef{}, err
	}
	return GoogleCredentialRef{snapshot: state.Snapshot}, nil
}

func (s *GoogleConnections) CheckPublicationCredential(ctx context.Context, org, storeID string, binding GooglePublicationBinding,
	ref GoogleCredentialRef, authorize func(context.Context) error,
) error {
	_, err := s.publicationCredential(ctx, s.credentialRequest(org, storeID, binding), ref, authorize)
	return err
}

func (s *GoogleConnections) publicationCredential(ctx context.Context, request storecenter.GoogleCredentialRequest,
	ref GoogleCredentialRef, authorize func(context.Context) error,
) (storecenter.GoogleGrant, error) {
	if err := authorize(ctx); err != nil {
		return storecenter.GoogleGrant{}, err
	}
	state, err := s.repo.GoogleCredential(ctx, request, "")
	if err != nil {
		return storecenter.GoogleGrant{}, googleCredentialError(err)
	}
	if !state.Usable || state.Snapshot != ref.snapshot {
		return storecenter.GoogleGrant{}, storecenter.ErrGoogleRefreshUnavailable
	}
	return state.Grant, nil
}

func (s *GoogleConnections) openGrant(grant storecenter.GoogleGrant) (google.TokenSet, error) {
	return s.registration.Protection.Open(grant.CredentialBinding(), grant.KeyID, grant.Ciphertext)
}

func (s *GoogleConnections) openPublicationCredential(ctx context.Context, request storecenter.GoogleCredentialRequest,
	ref GoogleCredentialRef, authorize func(context.Context) error,
) (google.TokenSet, error) {
	grant, err := s.publicationCredential(ctx, request, ref, authorize)
	if err != nil {
		return google.TokenSet{}, err
	}
	return s.openUsableGrant(grant)
}

func (s *GoogleConnections) openUsableGrant(grant storecenter.GoogleGrant) (google.TokenSet, error) {
	token, err := s.openGrant(grant)
	if err != nil || !time.Now().Add(storecenter.GoogleAccessHeadroom).Before(token.ExpiresAt) {
		return google.TokenSet{}, storecenter.ErrGoogleRefreshUnavailable
	}
	return token, nil
}

func googleCredentialError(err error) error {
	switch {
	case errors.Is(err, storecenter.ErrGoogleReauthorization),
		errors.Is(err, storecenter.ErrVersionConflict),
		errors.Is(err, storecenter.ErrNotFound),
		errors.Is(err, storecenter.ErrOfficialConnectionUnavailable):
		return err
	default:
		return storecenter.ErrGoogleRefreshUnavailable
	}
}
