package storecenterapp

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"

	"task-processor/internal/storecenter"
	tiktok "task-processor/internal/storecenter/tiktokshop"
)

type TikTokConnections struct {
	store       storecenter.TikTokConnectionStore
	provider    tiktok.SellerAuthorization
	protection  tiktok.CredentialProtection
	application tiktok.Application
}

func NewTikTokConnections(store storecenter.TikTokConnectionStore, provider tiktok.SellerAuthorization, protection tiktok.CredentialProtection) (*TikTokConnections, error) {
	if store == nil || (provider == nil) != (protection == nil) {
		return nil, storecenter.ErrOfficialConnectionUnavailable
	}
	app := &TikTokConnections{store: store, provider: provider, protection: protection}
	if provider != nil {
		app.application = provider.Application()
	}
	return app, nil
}

func (a *TikTokConnections) Owns(ctx context.Context, org, storeID string) (bool, error) {
	store, err := a.store.Get(ctx, org, storeID)
	if err != nil {
		return false, err
	}
	if store.OrganizationID() != org || store.ID() != storeID {
		return false, storecenter.ErrNotFound
	}
	return store.Platform() == storecenter.PlatformTikTokShop, nil
}

func (a *TikTokConnections) Read(ctx context.Context, org, storeID string) (storecenter.OfficialConnectionView, error) {
	owns, err := a.Owns(ctx, org, storeID)
	if err != nil {
		return storecenter.OfficialConnectionView{}, err
	}
	if !owns {
		return storecenter.OfficialConnectionView{}, storecenter.ErrNotFound
	}
	if a.provider == nil {
		return storecenter.OfficialConnectionView{Platform: storecenter.PlatformTikTokShop, Status: storecenter.ConnectionStatusUnavailable, State: "unconfigured", TikTok: &storecenter.TikTokConnectionDetails{Market: "US", SellerType: "CROSS_BORDER"}}, nil
	}
	view, err := a.store.ReadTikTokConnection(ctx, org, storeID)
	if err == nil && view.AppID != "" && (view.AppID != a.application.AppKey || view.AppRevision != a.application.Revision) {
		view.Status = storecenter.ConnectionStatusUnavailable
	}
	return view, err
}

func (a *TikTokConnections) Applications(ctx context.Context, org, storeID string) ([]storecenter.OfficialApplicationChoice, error) {
	owns, err := a.Owns(ctx, org, storeID)
	if err != nil {
		return nil, err
	}
	if !owns {
		return nil, storecenter.ErrNotFound
	}
	if a.provider == nil {
		return []storecenter.OfficialApplicationChoice{}, nil
	}
	return []storecenter.OfficialApplicationChoice{{AppID: a.application.AppKey, Revision: a.application.Revision, Platform: storecenter.PlatformTikTokShop, AuthorizationMethod: storecenter.OfficialAuthorizationMethod("oauth_callback")}}, nil
}

func (a *TikTokConnections) Begin(ctx context.Context, c storecenter.OfficialConnectionCommand) (storecenter.OfficialConnectionBegin, error) {
	if a.provider == nil {
		return storecenter.OfficialConnectionBegin{}, storecenter.ErrOfficialConnectionUnavailable
	}
	if c.ApplicationID != a.application.AppKey || c.IdentityPartition != "" {
		return storecenter.OfficialConnectionBegin{}, storecenter.ErrNotFound
	}
	var bytes [32]byte
	if _, err := rand.Read(bytes[:]); err != nil {
		return storecenter.OfficialConnectionBegin{}, storecenter.ErrOfficialConnectionUnavailable
	}
	state := base64.RawURLEncoding.EncodeToString(bytes[:])
	hash := sha256.Sum256([]byte(state))
	link, err := a.provider.AuthorizationURL(state)
	if err != nil {
		return storecenter.OfficialConnectionBegin{}, storecenter.ErrOfficialConnectionUnavailable
	}
	attempt, err := a.store.BeginTikTokConnection(ctx, c, a.application, hex.EncodeToString(hash[:]), time.Now().UTC())
	if err != nil {
		return storecenter.OfficialConnectionBegin{}, err
	}
	return storecenter.OfficialConnectionBegin{AttemptID: attempt.AttemptID, AuthorizationURL: link, ExpiresAt: attempt.ExpiresAt, AuthorizationMethod: storecenter.OfficialAuthorizationMethod("oauth_callback")}, nil
}

func (a *TikTokConnections) Complete(ctx context.Context, c storecenter.CompleteTikTokConnection) (storecenter.OfficialConnectionView, error) {
	if a.provider == nil {
		return storecenter.OfficialConnectionView{}, storecenter.ErrOfficialConnectionUnavailable
	}
	if c.AppKey != a.application.AppKey || c.State == "" || len(c.State) > 256 || len(c.AuthCode) > 2048 {
		return storecenter.OfficialConnectionView{}, storecenter.ErrNotFound
	}
	previous, err := a.store.ReadTikTokAttempt(ctx, c.OrganizationID, c.StoreID, c.AttemptID)
	if err != nil {
		return storecenter.OfficialConnectionView{}, err
	}
	if previous.AppKey != a.application.AppKey || previous.AppRevision != a.application.Revision {
		return storecenter.OfficialConnectionView{}, storecenter.ErrOfficialConnectionUnavailable
	}
	hash := sha256.Sum256([]byte(c.State))
	attempt, dispatch, err := a.store.ClaimTikTokExchange(ctx, c, hex.EncodeToString(hash[:]), time.Now().UTC())
	if err != nil {
		return storecenter.OfficialConnectionView{}, err
	}
	if attempt.AppRevision != a.application.Revision {
		return storecenter.OfficialConnectionView{}, storecenter.ErrOfficialConnectionUnavailable
	}
	if dispatch {
		if c.AuthCode == "" || c.AuthCode == "null" {
			_ = a.store.FailTikTokAttempt(ctx, attempt, false, time.Now().UTC())
			return storecenter.OfficialConnectionView{}, storecenter.ErrOfficialAuthorizationRejected
		}
		tokens, callErr := a.provider.Exchange(ctx, c.AuthCode)
		if callErr != nil {
			unknown := !errors.Is(callErr, tiktok.ErrAuthorizationRejected)
			a.failExchange(ctx, attempt, unknown)
			if unknown {
				return storecenter.OfficialConnectionView{}, storecenter.ErrOfficialExchangeUnknown
			}
			return storecenter.OfficialConnectionView{}, storecenter.ErrOfficialAuthorizationRejected
		}
		key, sealed, sealErr := a.protection.Seal(attempt.CredentialBinding(), tokens)
		if sealErr != nil {
			a.failExchange(ctx, attempt, true)
			return storecenter.OfficialConnectionView{}, storecenter.ErrOfficialExchangeUnknown
		}
		sellerHash := sellerIdentityHash(tokens.OpenID)
		if err = a.store.SaveTikTokCandidate(ctx, attempt, key, sealed, sellerHash, time.Now().UTC()); err != nil {
			return storecenter.OfficialConnectionView{}, storecenter.ErrOfficialExchangeUnknown
		}
	}
	return a.Resume(ctx, c.OrganizationID, c.StoreID, c.AttemptID)
}

func (a *TikTokConnections) failExchange(ctx context.Context, attempt storecenter.TikTokConnectionAttempt, unknown bool) {
	writeCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 2*time.Second)
	defer cancel()
	_ = a.store.FailTikTokAttempt(writeCtx, attempt, unknown, time.Now().UTC())
}

func sellerIdentityHash(openID string) string {
	hash := sha256.Sum256([]byte(openID))
	return hex.EncodeToString(hash[:])
}

func grantCredentialBinding(grant tiktok.GrantSnapshot) tiktok.CredentialBinding {
	return tiktok.CredentialBinding{OrganizationID: grant.OrganizationID, AppKey: grant.AppKey, AppRevision: grant.AppRevision, ID: grant.ID, Purpose: "grant", Generation: grant.Generation}
}

func (a *TikTokConnections) Resume(ctx context.Context, org, storeID, attemptID string) (storecenter.OfficialConnectionView, error) {
	if a.provider == nil {
		return storecenter.OfficialConnectionView{}, storecenter.ErrOfficialConnectionUnavailable
	}
	attempt, err := a.store.ReadTikTokAttempt(ctx, org, storeID, attemptID)
	if err != nil {
		return storecenter.OfficialConnectionView{}, err
	}
	if attempt.AppKey != a.application.AppKey || attempt.AppRevision != a.application.Revision {
		return storecenter.OfficialConnectionView{}, storecenter.ErrOfficialConnectionUnavailable
	}
	if attempt.State == "bound" {
		return a.Read(ctx, org, storeID)
	}
	if attempt.State != "credential_received" {
		return storecenter.OfficialConnectionView{}, storecenter.ErrOfficialExchangeUnknown
	}
	tokens, err := a.protection.Open(attempt.CredentialBinding(), attempt.KeyID, attempt.Ciphertext)
	if err != nil || sellerIdentityHash(tokens.OpenID) != attempt.SellerIdentityHash || !time.Now().Before(tokens.AccessExpiresAt) {
		return storecenter.OfficialConnectionView{}, storecenter.ErrOfficialConnectionUnavailable
	}
	current, found, err := a.store.ReadTikTokSellerGrant(ctx, attempt, attempt.SellerIdentityHash)
	if err != nil {
		return storecenter.OfficialConnectionView{}, err
	}
	intent := storecenter.TikTokBindingIntent{Attempt: attempt, ReplaceCredential: true}
	now := time.Now().UTC()
	if !found {
		intent.Grant = tiktok.GrantSnapshot{OrganizationID: org, ID: uuid.NewString(), AppKey: a.application.AppKey, AppRevision: a.application.Revision, SellerIdentityHash: attempt.SellerIdentityHash, Generation: 1, Revision: 1, Status: tiktok.GrantActive, AccessExpiresAt: tokens.AccessExpiresAt, RefreshExpiresAt: tokens.RefreshExpiresAt, CreatedAt: now, UpdatedAt: now}
	} else {
		intent.ExpectedGrantRevision = current.Revision
		if attempt.GrantID == "" {
			// A first authorization of another Shop joins the existing canonical
			// token only after verifying that token's actual authorized Shops.
			grant, e := tiktok.RestoreGrant(current)
			if e != nil || !grant.CanSend(now) || current.AppRevision != a.application.Revision {
				return storecenter.OfficialConnectionView{}, storecenter.ErrOfficialConnectionUnavailable
			}
			tokens, e = a.protection.Open(grantCredentialBinding(current), current.KeyID, current.Ciphertext)
			if e != nil {
				return storecenter.OfficialConnectionView{}, storecenter.ErrOfficialConnectionUnavailable
			}
			intent.Grant = current
			intent.ReplaceCredential = false
		} else {
			if attempt.GrantID != current.ID || attempt.GrantRevision != current.Revision || attempt.GrantGeneration != current.Generation {
				return storecenter.OfficialConnectionView{}, storecenter.ErrVersionConflict
			}
			intent.Grant = current
			intent.Grant.Generation++
			intent.Grant.Revision++
			intent.Grant.Status = tiktok.GrantActive
			intent.Grant.AppRevision = a.application.Revision
			intent.Grant.OperationID = ""
			intent.Grant.ClaimExpiresAt = time.Time{}
			intent.Grant.AccessExpiresAt = tokens.AccessExpiresAt
			intent.Grant.RefreshExpiresAt = tokens.RefreshExpiresAt
			intent.Grant.UpdatedAt = now
		}
	}
	shops, err := a.provider.AuthorizedShops(ctx, tokens.AccessToken)
	if err != nil {
		return storecenter.OfficialConnectionView{}, storecenter.ErrOfficialConnectionUnavailable
	}
	store, err := a.store.Get(ctx, org, storeID)
	if err != nil {
		return storecenter.OfficialConnectionView{}, err
	}
	intent.Shop, err = tiktok.SelectUSCrossBorderShop(shops, store.ExternalStoreID())
	if err != nil {
		return storecenter.OfficialConnectionView{}, storecenter.ErrAlreadyExists
	}
	if intent.ReplaceCredential {
		intent.Grant.KeyID, intent.Grant.Ciphertext, err = a.protection.Seal(grantCredentialBinding(intent.Grant), tokens)
		if err != nil {
			return storecenter.OfficialConnectionView{}, storecenter.ErrOfficialConnectionUnavailable
		}
	}
	return a.store.BindTikTokConnection(ctx, intent, time.Now().UTC())
}

func (a *TikTokConnections) Status(ctx context.Context, input storecenter.ConnectionStatusInput) (storecenter.ConnectionStatus, error) {
	store, err := a.store.Get(ctx, input.OrganizationID, input.StoreID)
	if err != nil {
		return storecenter.ConnectionStatusUnavailable, err
	}
	if store.Platform() != storecenter.PlatformTikTokShop || input.Platform != storecenter.PlatformTikTokShop || store.ConnectionRef() != input.ConnectionRef {
		return storecenter.ConnectionStatusUnavailable, storecenter.ErrNotFound
	}
	view, err := a.Read(ctx, input.OrganizationID, input.StoreID)
	return view.Status, err
}

func (a *TikTokConnections) Disconnect(ctx context.Context, c storecenter.OfficialConnectionCommand) (storecenter.OfficialConnectionView, error) {
	return a.store.DisconnectTikTokConnection(ctx, c, time.Now().UTC())
}

func (a *TikTokConnections) Refresh(ctx context.Context, org, storeID, operationID string, generation int64) (storecenter.OfficialConnectionView, error) {
	if a.provider == nil {
		return storecenter.OfficialConnectionView{}, storecenter.ErrOfficialConnectionUnavailable
	}
	current, _, err := a.store.ReadTikTokGrant(ctx, org, storeID)
	if err != nil {
		return storecenter.OfficialConnectionView{}, err
	}
	if current.AppKey != a.application.AppKey || current.AppRevision != a.application.Revision {
		return storecenter.OfficialConnectionView{}, storecenter.ErrOfficialConnectionUnavailable
	}
	input, _ := json.Marshal([]any{org, storeID, operationID, generation, a.application.AppKey, a.application.Revision})
	hash := sha256.Sum256(input)
	claim, err := a.store.ClaimTikTokRefresh(ctx, org, storeID, operationID, generation, hex.EncodeToString(hash[:]), time.Now().UTC())
	if err != nil {
		return storecenter.OfficialConnectionView{}, err
	}
	if !claim.Dispatch {
		return a.Read(ctx, org, storeID)
	}
	unknown := func() (storecenter.OfficialConnectionView, error) {
		writeCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 2*time.Second)
		defer cancel()
		_ = a.store.UnknownTikTokRefresh(writeCtx, claim, time.Now().UTC())
		return storecenter.OfficialConnectionView{}, storecenter.ErrOfficialExchangeUnknown
	}
	if claim.Grant.AppKey != a.application.AppKey || claim.Grant.AppRevision != a.application.Revision {
		return unknown()
	}
	previous, err := a.protection.Open(grantCredentialBinding(claim.Grant), claim.Grant.KeyID, claim.Grant.Ciphertext)
	if err != nil {
		return unknown()
	}
	tokens, err := a.provider.Refresh(ctx, previous.RefreshToken)
	if err != nil || sellerIdentityHash(tokens.OpenID) != claim.Grant.SellerIdentityHash {
		return unknown()
	}
	// Store verifies the exact original CAS fence and completes the operation
	// receipt with this generation in one transaction. No caller may retry it.
	grant, err := tiktok.RestoreGrant(claim.Grant)
	if err != nil {
		return unknown()
	}
	next := claim.Grant
	next.Generation++
	key, cipher, err := a.protection.Seal(grantCredentialBinding(next), tokens)
	if err != nil {
		return unknown()
	}
	now := time.Now().UTC()
	if err = grant.CompleteRefresh(claim.Fence, key, cipher, tokens.AccessExpiresAt, tokens.RefreshExpiresAt, now); err != nil {
		return unknown()
	}
	if err = a.store.CompleteTikTokRefresh(ctx, claim, grant.Snapshot(), now); err != nil {
		return unknown()
	}
	return a.Read(ctx, org, storeID)
}

// Access is server-only and checks current scopes and actual authorized Shop.
// A saved connection projection alone never permits a resource mutation.
func (a *TikTokConnections) Access(ctx context.Context, org, storeID string, requiredScopes ...string) (tiktok.TokenSet, tiktok.Shop, tiktok.GrantSnapshot, error) {
	if a.provider == nil {
		return tiktok.TokenSet{}, tiktok.Shop{}, tiktok.GrantSnapshot{}, storecenter.ErrOfficialConnectionUnavailable
	}
	snapshot, binding, err := a.store.ReadTikTokGrant(ctx, org, storeID)
	if err != nil {
		return tiktok.TokenSet{}, tiktok.Shop{}, tiktok.GrantSnapshot{}, err
	}
	grant, err := tiktok.RestoreGrant(snapshot)
	if err != nil || !grant.CanSend(time.Now().UTC()) || snapshot.AppKey != a.application.AppKey || snapshot.AppRevision != a.application.Revision {
		return tiktok.TokenSet{}, tiktok.Shop{}, snapshot, storecenter.ErrOfficialConnectionUnavailable
	}
	tokens, err := a.protection.Open(grantCredentialBinding(snapshot), snapshot.KeyID, snapshot.Ciphertext)
	if err != nil || sellerIdentityHash(tokens.OpenID) != snapshot.SellerIdentityHash {
		return tiktok.TokenSet{}, tiktok.Shop{}, snapshot, storecenter.ErrOfficialConnectionUnavailable
	}
	for _, required := range requiredScopes {
		found := false
		for _, scope := range tokens.GrantedScopes {
			if required == scope {
				found = true
				break
			}
		}
		if !found {
			return tiktok.TokenSet{}, tiktok.Shop{}, snapshot, storecenter.ErrOfficialConnectionUnavailable
		}
	}
	shops, err := a.provider.AuthorizedShops(ctx, tokens.AccessToken)
	if err != nil {
		return tiktok.TokenSet{}, tiktok.Shop{}, snapshot, storecenter.ErrOfficialConnectionUnavailable
	}
	shop, err := tiktok.SelectUSCrossBorderShop(shops, binding.Shop.Code)
	if err != nil || shop.ID != binding.Shop.ID {
		return tiktok.TokenSet{}, tiktok.Shop{}, snapshot, storecenter.ErrOfficialConnectionUnavailable
	}
	current, currentBinding, err := a.store.ReadTikTokGrant(ctx, org, storeID)
	if err != nil || current != snapshot || currentBinding.Version != binding.Version || currentBinding.Status != "active" {
		return tiktok.TokenSet{}, tiktok.Shop{}, snapshot, storecenter.ErrVersionConflict
	}
	return tokens, shop, snapshot, nil
}
