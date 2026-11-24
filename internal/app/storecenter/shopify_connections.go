package storecenterapp

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"net/url"
	"strings"
	"time"

	wire "github.com/henryxu1008/shuomi-ai-commerce/internal/integration/shopify"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/storecenter"
)

type ShopifyApplication struct {
	AppID      string
	InstallURL string
	OAuth      *wire.OAuthClient
	Admin      *wire.AdminClient
	Issuer     *wire.AppTokenIssuer
	Protection *wire.PairingProtection
}

type ShopifyConnections struct {
	store *storecenter.MemberScopedStoreRepository
	app   ShopifyApplication
}

func NewShopifyConnections(store *storecenter.MemberScopedStoreRepository, app ShopifyApplication) (*ShopifyConnections, error) {
	if store == nil || !validShopifyApplication(app) {
		return nil, storecenter.ErrOfficialConnectionUnavailable
	}
	return &ShopifyConnections{store: store, app: app}, nil
}

func validShopifyApplication(app ShopifyApplication) bool {
	target, err := url.Parse(app.InstallURL)
	return err == nil && target.Scheme == "https" && target.User == nil && target.Fragment == "" &&
		(target.Host == "apps.shopify.com" || target.Host == "admin.shopify.com") && target.Path != "" &&
		strings.HasPrefix(app.AppID, "gid://shopify/App/") && app.Admin != nil && app.OAuth != nil && app.Issuer != nil && app.Protection != nil
}

func (a *ShopifyConnections) Begin(ctx context.Context, c storecenter.OfficialConnectionCommand, browser, state string) (storecenter.OfficialConnectionBegin, error) {
	if !validShopifyNonce(browser) || !validShopifyNonce(state) {
		return storecenter.OfficialConnectionBegin{}, storecenter.ErrNotFound
	}
	row, err := a.store.BeginShopifyPairing(ctx, c, a.app.AppID, shopifyDigestOf(browser), shopifyDigestOf(state), time.Now())
	if err != nil {
		return storecenter.OfficialConnectionBegin{}, err
	}
	return storecenter.OfficialConnectionBegin{AttemptID: row.AttemptID, AuthorizationURL: a.app.InstallURL, ExpiresAt: row.ExpiresAt,
		AuthorizationMethod: storecenter.OfficialAuthorizationMethod("shopify_oauth")}, nil
}

func (a *ShopifyConnections) Launch(ctx context.Context, key storecenter.ShopifyPairingKey, browser, state, query string) (string, error) {
	row, err := a.store.ReadShopifyPairing(ctx, key, shopifyDigestOf(browser), time.Now())
	if err != nil {
		return "", err
	}
	if row.State != "created" || row.AppID != a.app.AppID || row.StateDigest != shopifyDigestOf(state) {
		return "", storecenter.ErrVersionConflict
	}
	shop, err := a.app.OAuth.VerifyLaunch(query, time.Now())
	if err != nil {
		return "", storecenter.ErrNotFound
	}
	return a.app.OAuth.AuthorizationURL(shop, state)
}

func (a *ShopifyConnections) Stage(ctx context.Context, key storecenter.ShopifyPairingKey, browser, state, query string) (string, error) {
	proof, err := a.app.OAuth.VerifyCallback(query, state, time.Now())
	if err != nil {
		return "", storecenter.ErrNotFound
	}
	session := shopifyDigestOf(browser)
	row, err := a.store.ClaimShopifyPairing(ctx, key, session, shopifyDigestOf(state), time.Now())
	if err != nil {
		return "", err
	}
	if row.AppID != a.app.AppID {
		return "", storecenter.ErrVersionConflict
	}
	token, err := a.app.OAuth.ExchangeOwnerCode(ctx, proof)
	if err != nil {
		if errors.Is(err, wire.ErrShopOwnerRequired) {
			return "", storecenter.ErrShopifyOwnerRequired
		}
		return "", storecenter.ErrOfficialConnectionUnavailable
	}
	installation, err := a.app.Admin.Installation(ctx, proof.Shop(), token.Value, a.app.AppID)
	if err != nil {
		return "", storecenter.ErrOfficialConnectionUnavailable
	}
	generation, err := a.store.ShopifyPairingGeneration(ctx, key, session, shopifyIdentity(installation), time.Now())
	if err != nil {
		return "", err
	}
	keyID, ciphertext, err := a.app.Protection.Seal(pairingAAD(row, installation, generation), token, time.Now())
	if err != nil {
		return "", storecenter.ErrOfficialConnectionUnavailable
	}
	staged, err := a.store.StageShopifyPairing(ctx, key, session, shopifyIdentity(installation), generation, keyID, ciphertext, time.Now())
	return staged.Domain, err
}

func (a *ShopifyConnections) Confirm(ctx context.Context, key storecenter.ShopifyPairingKey, browser string) (storecenter.ShopifyBinding, error) {
	session := shopifyDigestOf(browser)
	row, err := a.store.ReadShopifyPairing(ctx, key, session, time.Now())
	if err != nil {
		return storecenter.ShopifyBinding{}, err
	}
	if row.AppID != a.app.AppID {
		return storecenter.ShopifyBinding{}, storecenter.ErrVersionConflict
	}
	if row.State == "confirmed" {
		binding, err := a.store.ReadShopifyBinding(ctx, key.OrganizationID, key.StoreID)
		if err != nil {
			return storecenter.ShopifyBinding{}, err
		}
		return a.store.ConfirmShopifyPairing(ctx, key, session, binding.ShopifyInstallation, time.Now())
	}
	if row.State != "staged" {
		return storecenter.ShopifyBinding{}, storecenter.ErrVersionConflict
	}
	expected := shopifyWireIdentity(row.ShopifyInstallation)
	token, err := a.app.Protection.Open(pairingAAD(row, expected, row.Generation), row.KeyID, row.Ciphertext, time.Now())
	if err != nil {
		return storecenter.ShopifyBinding{}, storecenter.ErrOfficialConnectionUnavailable
	}
	actual, err := a.app.Admin.Installation(ctx, expected.Domain, token.Value, a.app.AppID)
	if err != nil || actual != expected {
		return storecenter.ShopifyBinding{}, storecenter.ErrVersionConflict
	}
	return a.store.ConfirmShopifyPairing(ctx, key, session, shopifyIdentity(actual), time.Now())
}

func (a *ShopifyConnections) Read(ctx context.Context, org, storeID string) (storecenter.ShopifyBinding, error) {
	return a.store.ReadShopifyBinding(ctx, org, storeID)
}

func (a *ShopifyConnections) Status(ctx context.Context, org, storeID string) (storecenter.ConnectionStatus, error) {
	_, err := a.store.ReadShopifyBinding(ctx, org, storeID)
	if errors.Is(err, storecenter.ErrNotFound) {
		return storecenter.ConnectionStatusDisconnected, nil
	}
	if err != nil {
		return storecenter.ConnectionStatusUnavailable, err
	}
	return storecenter.ConnectionStatusConnected, nil
}

func pairingAAD(row storecenter.ShopifyPairing, identity wire.Installation, generation int64) wire.PairingBinding {
	return wire.PairingBinding{OrganizationID: row.OrganizationID, ActorID: row.ActorID, MemberID: row.MemberID, StoreID: row.StoreID,
		AttemptID: row.AttemptID, SessionDigest: row.SessionDigest, Installation: identity, Generation: generation, ExpiresAt: row.ExpiresAt}
}

func shopifyIdentity(v wire.Installation) storecenter.ShopifyInstallation {
	return storecenter.ShopifyInstallation{AppID: v.AppID, ShopID: v.ShopID, InstallationID: v.ID, Domain: v.Domain, Currency: v.Currency}
}

func shopifyWireIdentity(v storecenter.ShopifyInstallation) wire.Installation {
	return wire.Installation{AppID: v.AppID, ShopID: v.ShopID, ID: v.InstallationID, Domain: v.Domain, Currency: v.Currency}
}

func validShopifyNonce(value string) bool {
	return len(value) == 43 && strings.IndexFunc(value, func(r rune) bool {
		return !(r >= 'A' && r <= 'Z' || r >= 'a' && r <= 'z' || r >= '0' && r <= '9' || r == '-' || r == '_')
	}) == -1
}

func shopifyDigestOf(value string) string {
	digest := sha256.Sum256([]byte(value))
	return hex.EncodeToString(digest[:])
}
