package storecenterapp

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/storecenter"
	google "github.com/henryxu1008/shuomi-ai-commerce/internal/storecenter/googlemerchant"
)

type GoogleOAuth interface {
	BeginAuthorization(string) (string, error)
	ExchangeAuthorization(context.Context, string, string) (google.TokenSet, error)
	Refresh(context.Context, google.TokenSet) (google.TokenSet, error)
}

type GoogleAccounts interface {
	ListAccounts(context.Context, string, string) (google.AccountPage, error)
	ListSources(context.Context, string, string, string) (google.SourcePage, error)
	VerifySource(context.Context, string, google.Target) error
	InsertConfirmed(context.Context, string, google.Target, google.Offer, func(context.Context) error) (google.Receipt, error)
	ReadProduct(context.Context, string, google.Target, string) (google.Observation, error)
}

type GoogleConnectionStore interface {
	storecenter.GoogleConnectionRepository
	storecenter.GoogleRefreshRepository
	Get(context.Context, string, string) (*storecenter.Store, error)
}

type GoogleApplicationRegistration struct {
	Application google.Application
	OAuth       GoogleOAuth
	Accounts    GoogleAccounts
	Protection  google.CredentialProtection
}

type GoogleConnections struct {
	repo         GoogleConnectionStore
	registration GoogleApplicationRegistration
}

func NewGoogleConnections(repo GoogleConnectionStore, registration GoogleApplicationRegistration) (*GoogleConnections, error) {
	if repo == nil || registration.Application.ID == "" || registration.Application.Revision == "" ||
		registration.OAuth == nil || registration.Accounts == nil || registration.Protection == nil {
		return nil, storecenter.ErrOfficialConnectionUnavailable
	}
	return &GoogleConnections{repo: repo, registration: registration}, nil
}

func (s *GoogleConnections) Begin(ctx context.Context, org, storeID string, version int64) (storecenter.OfficialConnectionBegin, error) {
	var random [32]byte
	if _, err := rand.Read(random[:]); err != nil {
		return storecenter.OfficialConnectionBegin{}, storecenter.ErrOfficialConnectionUnavailable
	}
	state := base64.RawURLEncoding.EncodeToString(random[:])
	address, err := s.registration.OAuth.BeginAuthorization(state)
	if err != nil {
		return storecenter.OfficialConnectionBegin{}, storecenter.ErrOfficialConnectionUnavailable
	}
	app := s.registration.Application
	command := storecenter.OfficialConnectionCommand{OrganizationID: org, StoreID: storeID, AttemptID: uuid.NewString(),
		ExpectedStoreVersion: version, ApplicationID: app.ID}
	attempt, err := s.repo.BeginGoogleConnection(ctx, command, app, googleStateHash(state), time.Now().UTC())
	if err != nil {
		return storecenter.OfficialConnectionBegin{}, err
	}
	return storecenter.OfficialConnectionBegin{AttemptID: attempt.AttemptID, AuthorizationURL: address, ExpiresAt: attempt.ExpiresAt,
		AuthorizationMethod: storecenter.AuthorizationCallback}, nil
}

func (s *GoogleConnections) Complete(ctx context.Context, org, storeID, attemptID, state, code string) error {
	if len(state) != 43 || code == "" || len(code) > 8192 {
		return storecenter.ErrOfficialAuthorizationRejected
	}
	now := time.Now().UTC()
	attempt, err := s.repo.ClaimGoogleExchange(ctx, org, storeID, attemptID, googleStateHash(state), s.registration.Application, now)
	if err != nil {
		return err
	}
	tokens, err := s.registration.OAuth.ExchangeAuthorization(ctx, state, code)
	if err != nil {
		return storecenter.ErrOfficialAuthorizationRejected
	}
	key, ciphertext, err := s.registration.Protection.Seal(attempt.CredentialBinding(), tokens)
	if err != nil {
		return storecenter.ErrOfficialConnectionUnavailable
	}
	return s.repo.SaveGoogleCandidate(ctx, attempt, key, ciphertext, time.Now().UTC())
}

func (s *GoogleConnections) candidate(ctx context.Context, org, storeID, attemptID string) (storecenter.GoogleAttempt, google.TokenSet, error) {
	attempt, err := s.repo.ReadGoogleCandidate(ctx, org, storeID, attemptID, time.Now().UTC())
	if err != nil {
		return attempt, google.TokenSet{}, err
	}
	if attempt.AppID != s.registration.Application.ID || attempt.AppRevision != s.registration.Application.Revision {
		return attempt, google.TokenSet{}, storecenter.ErrOfficialAuthorizationRejected
	}
	token, err := s.registration.Protection.Open(attempt.CredentialBinding(), attempt.KeyID, attempt.Ciphertext)
	if err != nil || !time.Now().Before(token.ExpiresAt) {
		return attempt, google.TokenSet{}, storecenter.ErrOfficialCredentialExpired
	}
	return attempt, token, nil
}

func (s *GoogleConnections) Accounts(ctx context.Context, org, storeID, attemptID, page string) (google.AccountPage, error) {
	_, token, err := s.candidate(ctx, org, storeID, attemptID)
	if err != nil {
		return google.AccountPage{}, err
	}
	result, err := s.registration.Accounts.ListAccounts(ctx, token.AccessToken, page)
	return result, googleConnectionError(err)
}

func (s *GoogleConnections) Sources(ctx context.Context, org, storeID, attemptID, account, page string) (google.SourcePage, error) {
	_, token, err := s.candidate(ctx, org, storeID, attemptID)
	if err != nil {
		return google.SourcePage{}, err
	}
	result, err := s.registration.Accounts.ListSources(ctx, token.AccessToken, account, page)
	return result, googleConnectionError(err)
}

func (s *GoogleConnections) Bind(ctx context.Context, org, storeID, attemptID string, selection storecenter.GoogleAccountSelection) (storecenter.GoogleConnectionView, error) {
	if !selection.DedicatedSourceConfirmed {
		return storecenter.GoogleConnectionView{}, storecenter.ErrOfficialAuthorizationRejected
	}
	attempt, token, err := s.candidate(ctx, org, storeID, attemptID)
	if err != nil {
		return storecenter.GoogleConnectionView{}, err
	}
	var random [10]byte
	if _, err := rand.Read(random[:]); err != nil {
		return storecenter.GoogleConnectionView{}, storecenter.ErrOfficialConnectionUnavailable
	}
	target := google.Target{AccountID: selection.AccountID, DataSourceID: selection.DataSourceID,
		ContentLanguage: selection.ContentLanguage, FeedLabel: selection.FeedLabel, OfferPrefix: "sm-" + hex.EncodeToString(random[:]) + "-"}
	prior, err := s.repo.ReadGoogleGrant(ctx, org, storeID, false)
	if err == nil {
		target.OfferPrefix = prior.Target.OfferPrefix
		if prior.Target != target {
			return storecenter.GoogleConnectionView{}, storecenter.ErrVersionConflict
		}
	} else if !errors.Is(err, storecenter.ErrNotFound) {
		return storecenter.GoogleConnectionView{}, err
	}
	if !target.Valid() {
		return storecenter.GoogleConnectionView{}, storecenter.ErrOfficialAuthorizationRejected
	}
	if err := s.registration.Accounts.VerifySource(ctx, token.AccessToken, target); err != nil {
		return storecenter.GoogleConnectionView{}, googleConnectionError(err)
	}
	grant := storecenter.GoogleGrant{OrganizationID: org, StoreID: storeID, ID: attempt.GrantID,
		AppID: attempt.AppID, AppRevision: attempt.AppRevision, Generation: attempt.Generation, Target: target, AccessExpiresAt: token.ExpiresAt}
	grant.KeyID, grant.Ciphertext, err = s.registration.Protection.Seal(grant.CredentialBinding(), token)
	if err != nil {
		return storecenter.GoogleConnectionView{}, storecenter.ErrOfficialConnectionUnavailable
	}
	saved, err := s.repo.BindGoogleConnection(ctx, attempt, grant, time.Now().UTC())
	if err != nil {
		return storecenter.GoogleConnectionView{}, err
	}
	return googleView(saved), nil
}

func (s *GoogleConnections) Read(ctx context.Context, org, storeID string) (storecenter.GoogleConnectionView, error) {
	store, err := s.repo.Get(ctx, org, storeID)
	if err != nil {
		return storecenter.GoogleConnectionView{}, err
	}
	if store.Platform() != storecenter.PlatformGoogleMerchant {
		return storecenter.GoogleConnectionView{}, storecenter.ErrNotFound
	}
	grant, err := s.repo.ReadGoogleGrant(ctx, org, storeID, false)
	if errors.Is(err, storecenter.ErrNotFound) {
		return storecenter.GoogleConnectionView{Status: storecenter.ConnectionStatusDisconnected}, nil
	}
	if err != nil {
		return storecenter.GoogleConnectionView{}, err
	}
	view := googleView(grant)
	if view.Status == storecenter.ConnectionStatusConnected && (grant.AppID != s.registration.Application.ID ||
		grant.AppRevision != s.registration.Application.Revision) {
		view.Status = storecenter.ConnectionStatusExpired
	}
	return view, nil
}

func (s *GoogleConnections) Disconnect(ctx context.Context, org, storeID string, version int64) error {
	return s.repo.DisconnectGoogleConnection(ctx, storecenter.OfficialConnectionCommand{
		OrganizationID: org, StoreID: storeID, ExpectedStoreVersion: version}, time.Now().UTC())
}

type GooglePublicationBinding struct {
	Target     google.Target
	Generation int64
}

func (s *GoogleConnections) PublicationBinding(ctx context.Context, org, storeID string) (GooglePublicationBinding, error) {
	grant, err := s.currentGrant(ctx, org, storeID)
	if err != nil {
		return GooglePublicationBinding{}, err
	}
	return GooglePublicationBinding{Target: grant.Target, Generation: grant.Generation}, nil
}

func (s *GoogleConnections) currentGrant(ctx context.Context, org, storeID string) (storecenter.GoogleGrant, error) {
	grant, err := s.repo.ReadGoogleGrant(ctx, org, storeID, true)
	if err != nil {
		return grant, err
	}
	if grant.RefreshUnavailable || grant.AppID != s.registration.Application.ID || grant.AppRevision != s.registration.Application.Revision {
		return storecenter.GoogleGrant{}, storecenter.ErrGoogleReauthorization
	}
	return grant, nil
}

func (s *GoogleConnections) Publish(ctx context.Context, org, storeID string, binding GooglePublicationBinding, ref GoogleCredentialRef,
	offer google.Offer, authorize func(context.Context) error, beforeSend func(context.Context) error,
) (google.Receipt, error) {
	request := s.credentialRequest(org, storeID, binding)
	token, err := s.openPublicationCredential(ctx, request, ref, authorize)
	if err != nil {
		return google.Receipt{}, err
	}
	return s.registration.Accounts.InsertConfirmed(ctx, token.AccessToken, binding.Target, offer, func(ctx context.Context) error {
		if _, err := s.publicationCredential(ctx, request, ref, authorize); err != nil {
			return err
		}
		return beforeSend(ctx)
	})
}

func (s *GoogleConnections) ReadProduct(ctx context.Context, org, storeID string, target google.Target, offerID string,
	ref GoogleCredentialRef, authorize func(context.Context) error,
) (google.Observation, error) {
	token, err := s.openPublicationCredential(ctx, s.credentialRequest(org, storeID, GooglePublicationBinding{Target: target}), ref, authorize)
	if err != nil {
		return google.Observation{}, err
	}
	return s.registration.Accounts.ReadProduct(ctx, token.AccessToken, target, offerID)
}

func googleStateHash(state string) string {
	digest := sha256.Sum256([]byte(state))
	return hex.EncodeToString(digest[:])
}

func googleConnectionError(err error) error {
	switch {
	case err == nil:
		return nil
	case errors.Is(err, google.ErrUnauthorized):
		return storecenter.ErrOfficialAuthorizationRejected
	case errors.Is(err, google.ErrInvalid), errors.Is(err, google.ErrSourceMismatch):
		return storecenter.ErrVersionConflict
	default:
		return storecenter.ErrOfficialConnectionUnavailable
	}
}

func googleView(grant storecenter.GoogleGrant) storecenter.GoogleConnectionView {
	status := storecenter.ConnectionStatusConnected
	if grant.Revoked {
		status = storecenter.ConnectionStatusDisconnected
	} else if grant.RefreshUnavailable {
		status = storecenter.ConnectionStatusExpired
	}
	return storecenter.GoogleConnectionView{Status: status, AccountID: grant.Target.AccountID, DataSourceID: grant.Target.DataSourceID,
		ContentLanguage: grant.Target.ContentLanguage, FeedLabel: grant.Target.FeedLabel}
}
