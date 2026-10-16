package storecenterapp

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"time"

	"github.com/google/uuid"
	marketplace "task-processor/internal/marketplace/walmart"
	"task-processor/internal/storecenter"
	auth "task-processor/internal/storecenter/walmart"
)

type WalmartConnections struct {
	store      storecenter.WalmartConnectionStore
	provider   auth.Authorization
	protection auth.CredentialProtection
}

func newWalmartConnections(store storecenter.OfficialConnectionStore, r *OfficialApplicationRegistry) *WalmartConnections {
	repo, ok := store.(storecenter.WalmartConnectionStore)
	if !ok {
		return nil
	}
	a := &WalmartConnections{store: repo}
	if r != nil {
		a.provider = r.walmartProvider
		a.protection = r.walmartProtection
	}
	return a
}
func (a *WalmartConnections) Owns(ctx context.Context, org, id string) (bool, error) {
	s, e := a.store.Get(ctx, org, id)
	if e != nil {
		return false, e
	}
	return s.OrganizationID() == org && s.ID() == id && s.Platform() == storecenter.PlatformWalmart, nil
}
func (a *WalmartConnections) Read(ctx context.Context, org, id string) (storecenter.OfficialConnectionView, error) {
	if a.provider == nil {
		owns, e := a.Owns(ctx, org, id)
		if e != nil || !owns {
			return storecenter.OfficialConnectionView{}, storecenter.ErrNotFound
		}
		return storecenter.OfficialConnectionView{Platform: storecenter.PlatformWalmart, Status: storecenter.ConnectionStatusUnavailable, State: "unconfigured", AuthorizationMethod: "oauth_callback", Walmart: &storecenter.WalmartConnectionDetails{Market: "US"}}, nil
	}
	v, e := a.store.ReadWalmartConnection(ctx, org, id)
	if e != nil {
		return v, e
	}
	if a.provider == nil {
		v.Status = storecenter.ConnectionStatusUnavailable
		v.State = "unconfigured"
	} else {
		app := a.provider.Application()
		if v.AppID == "" {
			v.Walmart.Environment = app.Environment
		}
		if v.AppID != "" && (v.AppID != app.ClientID || v.AppRevision != app.Revision || v.Walmart == nil || v.Walmart.Environment != app.Environment) {
			v.Status = storecenter.ConnectionStatusUnavailable
		}
	}
	return v, nil
}
func (a *WalmartConnections) Applications(ctx context.Context, org, id string) ([]storecenter.OfficialApplicationChoice, error) {
	if _, e := a.Read(ctx, org, id); e != nil {
		return nil, e
	}
	if a.provider == nil {
		return []storecenter.OfficialApplicationChoice{}, nil
	}
	app := a.provider.Application()
	return []storecenter.OfficialApplicationChoice{{Platform: storecenter.PlatformWalmart, AppID: app.ClientID, Revision: app.Revision, AuthorizationMethod: "oauth_callback"}}, nil
}
func randomWalmartState() (string, error) {
	b := make([]byte, 32)
	if _, e := rand.Read(b); e != nil {
		return "", e
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}
func (a *WalmartConnections) Begin(ctx context.Context, c storecenter.OfficialConnectionCommand) (storecenter.OfficialConnectionBegin, error) {
	var out storecenter.OfficialConnectionBegin
	if a.provider == nil || c.ApplicationID != a.provider.Application().ClientID || c.IdentityPartition != "" {
		return out, storecenter.ErrOfficialConnectionUnavailable
	}
	state, e := randomWalmartState()
	if e != nil {
		return out, e
	}
	nonce, e := randomWalmartState()
	if e != nil {
		return out, e
	}
	app := a.provider.Application()
	link, e := a.provider.AuthorizationURL(state, nonce)
	if e != nil {
		return out, storecenter.ErrOfficialConnectionUnavailable
	}
	attempt, e := a.store.BeginWalmartConnection(ctx, c, app, marketplace.Hash(state), nonce, time.Now().UTC())
	if e != nil {
		return out, e
	}
	return storecenter.OfficialConnectionBegin{AttemptID: attempt.AttemptID, AuthorizationURL: link, ExpiresAt: attempt.ExpiresAt, AuthorizationMethod: "oauth_callback"}, nil
}
func (a *WalmartConnections) Complete(ctx context.Context, c storecenter.CompleteWalmartConnection) (storecenter.OfficialConnectionView, error) {
	if a.provider == nil || c.AppID != a.provider.Application().ClientID {
		return storecenter.OfficialConnectionView{}, storecenter.ErrOfficialConnectionUnavailable
	}
	if c.MerchantConfirmation != "" {
		if c.Code != "" || c.State != "" || c.CallbackType != "" || c.SellerID != "" {
			return storecenter.OfficialConnectionView{}, storecenter.ErrNotFound
		}
		return a.resume(ctx, c.OrganizationID, c.StoreID, c.AttemptID, c.MerchantConfirmation)
	}
	if c.Code == "" || len(c.Code) > 2048 || c.State == "" || len(c.State) > 256 || c.CallbackType != "auth" {
		return storecenter.OfficialConnectionView{}, storecenter.ErrNotFound
	}
	original, e := a.store.ReadWalmartAttempt(ctx, c.OrganizationID, c.StoreID, c.AttemptID)
	if e != nil {
		return storecenter.OfficialConnectionView{}, e
	}
	if !a.matches(original) {
		return storecenter.OfficialConnectionView{}, storecenter.ErrOfficialConnectionUnavailable
	}
	attempt, dispatch, e := a.store.ClaimWalmartExchange(ctx, c, marketplace.Hash(c.State), time.Now().UTC())
	if e != nil {
		return storecenter.OfficialConnectionView{}, e
	}
	if dispatch {
		tokens, callErr := a.provider.Exchange(ctx, c.Code, c.SellerID)
		if callErr != nil {
			writeCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 2*time.Second)
			defer cancel()
			_ = a.store.FailWalmartExchange(writeCtx, attempt, !errors.Is(callErr, marketplace.ErrRejected), time.Now().UTC())
			if errors.Is(callErr, marketplace.ErrRejected) {
				return storecenter.OfficialConnectionView{}, storecenter.ErrOfficialAuthorizationRejected
			}
			return storecenter.OfficialConnectionView{}, storecenter.ErrOfficialExchangeUnknown
		}
		key, sealed, e := a.protection.Seal(attempt.CredentialBinding(), tokens)
		if e != nil {
			return storecenter.OfficialConnectionView{}, storecenter.ErrOfficialExchangeUnknown
		}
		// Commit before retryable profile reads. A lost commit is read back through
		// the same attempt, never by exchanging the code a second time.
		if e = a.store.SaveWalmartCandidate(ctx, attempt, key, sealed, time.Now().UTC()); e != nil {
			return storecenter.OfficialConnectionView{}, storecenter.ErrOfficialExchangeUnknown
		}
	}
	return a.Resume(ctx, c.OrganizationID, c.StoreID, c.AttemptID)
}
func (a *WalmartConnections) matches(v storecenter.WalmartConnectionAttempt) bool {
	p := a.provider.Application()
	return v.AppID == p.ClientID && v.AppRevision == p.Revision && v.Environment == p.Environment && v.CallbackURL == p.CallbackURL
}
func (a *WalmartConnections) Resume(ctx context.Context, org, id, attempt string) (storecenter.OfficialConnectionView, error) {
	return a.resume(ctx, org, id, attempt, "")
}
func (a *WalmartConnections) resume(ctx context.Context, org, id, attempt, confirm string) (storecenter.OfficialConnectionView, error) {
	var out storecenter.OfficialConnectionView
	if a.provider == nil {
		return out, storecenter.ErrOfficialConnectionUnavailable
	}
	v, e := a.store.ReadWalmartAttempt(ctx, org, id, attempt)
	if e != nil {
		return out, e
	}
	if !a.matches(v) {
		return out, storecenter.ErrOfficialConnectionUnavailable
	}
	if v.State == "bound" {
		return a.Read(ctx, org, id)
	}
	if v.State != "credential_received" {
		return out, storecenter.ErrOfficialExchangeUnknown
	}
	tokens, e := a.protection.Open(v.CredentialBinding(), v.KeyID, v.Ciphertext)
	if e != nil || tokens.SellerID != v.SellerID || !time.Now().Before(tokens.AccessExpiresAt) {
		return out, storecenter.ErrOfficialConnectionUnavailable
	}
	detail, e := a.provider.TokenDetail(ctx, tokens)
	if e != nil || !detail.Allows("product_rules", time.Now().UTC()) {
		return out, storecenter.ErrOfficialConnectionUnavailable
	}
	merchant, e := a.provider.Merchant(ctx, tokens)
	if e != nil || merchant.PartnerID != v.SellerID {
		return out, storecenter.ErrOfficialAuthorizationRejected
	}
	if confirm == "" {
		out = storecenter.OfficialConnectionView{Platform: storecenter.PlatformWalmart, AppID: v.AppID, AppRevision: v.AppRevision, AttemptID: v.AttemptID, Version: v.ConnectionVersion, State: "merchant_confirmation_required", Status: storecenter.ConnectionStatusDisconnected, Walmart: &storecenter.WalmartConnectionDetails{Market: "US", Environment: v.Environment, PartnerID: merchant.PartnerID, MerchantName: merchant.Name}}
		return out, nil
	}
	if confirm != merchant.PartnerID {
		return out, storecenter.ErrNotFound
	}
	expiry := tokens.AccessExpiresAt
	if detail.ExpiresAt.Before(expiry) {
		expiry = detail.ExpiresAt
	}
	now := time.Now().UTC()
	g := storecenter.WalmartGrant{OrganizationID: org, StoreID: id, ID: uuid.NewString(), AppID: v.AppID, AppRevision: v.AppRevision, Environment: v.Environment, MerchantIdentityHash: storecenter.WalmartMerchantIdentity(v.Environment, merchant.PartnerID), SellerID: merchant.PartnerID, MerchantName: merchant.Name, Generation: 1, Version: 1, Status: "active", AccessExpiresAt: expiry, RefreshExpiresAt: tokens.RefreshExpiresAt, ScopesHash: marketplace.Hash(detail.Scopes), CreatedAt: now, UpdatedAt: now}
	if v.GrantID != "" {
		g.ID = v.GrantID
		g.Generation = v.GrantGeneration + 1
		g.Version = v.GrantVersion + 1
	}
	g.KeyID, g.Ciphertext, e = a.protection.Seal(g.CredentialBinding(), tokens)
	if e != nil {
		return out, storecenter.ErrOfficialConnectionUnavailable
	}
	return a.store.BindWalmartConnection(ctx, storecenter.WalmartBindingIntent{Attempt: v, Grant: g}, now)
}
func (a *WalmartConnections) Disconnect(ctx context.Context, c storecenter.OfficialConnectionCommand) (storecenter.OfficialConnectionView, error) {
	return a.store.DisconnectWalmartConnection(ctx, c, time.Now().UTC())
}
func (a *WalmartConnections) Status(ctx context.Context, c storecenter.ConnectionStatusInput) (storecenter.ConnectionStatus, error) {
	if a.provider == nil {
		return storecenter.ConnectionStatusUnavailable, nil
	}
	s, e := a.store.Get(ctx, c.OrganizationID, c.StoreID)
	if e != nil || s.Platform() != storecenter.PlatformWalmart || s.ConnectionRef() != c.ConnectionRef {
		return storecenter.ConnectionStatusUnavailable, storecenter.ErrNotFound
	}
	v, e := a.Read(ctx, c.OrganizationID, c.StoreID)
	return v.Status, e
}
func (a *WalmartConnections) Refresh(ctx context.Context, org, id, op string, generation int64) (storecenter.OfficialConnectionView, error) {
	var out storecenter.OfficialConnectionView
	if a.provider == nil {
		return out, storecenter.ErrOfficialConnectionUnavailable
	}
	app := a.provider.Application()
	hash := marketplace.Hash([]any{org, id, op, generation, app.ClientID, app.Revision, app.Environment})
	current, e := a.store.ReadWalmartGrant(ctx, org, id)
	if e != nil {
		return out, e
	}
	if current.AppID != app.ClientID || current.AppRevision != app.Revision || current.Environment != app.Environment {
		return out, storecenter.ErrOfficialConnectionUnavailable
	}
	claim, e := a.store.ClaimWalmartRefresh(ctx, org, id, op, generation, hash, time.Now().UTC())
	if e != nil {
		return out, e
	}
	if !claim.Dispatch {
		return a.Read(ctx, org, id)
	}
	unknown := func() (storecenter.OfficialConnectionView, error) {
		writeCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 2*time.Second)
		defer cancel()
		_ = a.store.UnknownWalmartRefresh(writeCtx, claim, time.Now().UTC())
		return out, storecenter.ErrOfficialExchangeUnknown
	}
	old, e := a.protection.Open(claim.Grant.CredentialBinding(), claim.Grant.KeyID, claim.Grant.Ciphertext)
	if e != nil {
		return unknown()
	}
	tokens, e := a.provider.Refresh(ctx, old)
	if e != nil || tokens.SellerID != old.SellerID {
		return unknown()
	}
	detail, e := a.provider.TokenDetail(ctx, tokens)
	if e != nil || !detail.Allows("product_rules", time.Now().UTC()) {
		return unknown()
	}
	next := claim.Grant
	next.Generation++
	next.Version++
	next.Status = "active"
	next.AccessExpiresAt = tokens.AccessExpiresAt
	if detail.ExpiresAt.Before(next.AccessExpiresAt) {
		next.AccessExpiresAt = detail.ExpiresAt
	}
	next.RefreshExpiresAt = tokens.RefreshExpiresAt
	next.ScopesHash = marketplace.Hash(detail.Scopes)
	next.UpdatedAt = time.Now().UTC()
	next.KeyID, next.Ciphertext, e = a.protection.Seal(next.CredentialBinding(), tokens)
	if e != nil {
		return unknown()
	}
	if e = a.store.CompleteWalmartRefresh(ctx, claim, next, time.Now().UTC()); e != nil {
		return unknown()
	}
	return a.Read(ctx, org, id)
}

func (a *OfficialConnections) RefreshWalmart(ctx context.Context, org, id, operation string, generation int64) (storecenter.OfficialConnectionView, error) {
	if a.walmart == nil {
		return storecenter.OfficialConnectionView{}, storecenter.ErrOfficialConnectionUnavailable
	}
	owns, e := a.walmart.Owns(ctx, org, id)
	if e != nil {
		return storecenter.OfficialConnectionView{}, e
	}
	if !owns {
		return storecenter.OfficialConnectionView{}, storecenter.ErrNotFound
	}
	return a.walmart.Refresh(ctx, org, id, operation, generation)
}
