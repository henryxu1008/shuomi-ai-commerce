package storecenterapp

import (
	"context"
	"errors"
	"time"

	wire "github.com/henryxu1008/shuomi-ai-commerce/internal/integration/shopify"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/storecenter"
)

func (a *ShopifyConnections) StartedInstallationEnabled() bool {
	return a != nil && a.app.StartedInstallation
}

func (a *ShopifyConnections) BeginStartedInstallation(ctx context.Context, attempt, browser, state, query string) (string, error) {
	if !a.StartedInstallationEnabled() || !validShopifyNonce(browser) || !validShopifyNonce(state) {
		return "", storecenter.ErrNotFound
	}
	now := time.Now()
	shop, err := a.app.OAuth.VerifyLaunch(query, now)
	if err != nil {
		return "", storecenter.ErrNotFound
	}
	target, err := a.app.OAuth.AuthorizationURL(shop, state)
	if err != nil {
		return "", storecenter.ErrNotFound
	}
	if err := a.store.BeginShopifyInstall(ctx, attempt, a.app.AppID, shop, shopifyDigestOf(browser), shopifyDigestOf(state), now); err != nil {
		return "", err
	}
	return target, nil
}

func (a *ShopifyConnections) StageStartedInstallation(ctx context.Context, attempt, browser, state, query string) (string, error) {
	if !a.StartedInstallationEnabled() || !validShopifyNonce(browser) || !validShopifyNonce(state) {
		return "", storecenter.ErrNotFound
	}
	now := time.Now()
	callback, err := a.app.OAuth.VerifyCallback(query, state, now)
	if err != nil {
		return "", storecenter.ErrNotFound
	}
	proof, err := a.store.ClaimShopifyInstall(ctx, attempt, a.app.AppID, callback.Shop(), shopifyDigestOf(browser), shopifyDigestOf(state), callback.CodeDigest(), now)
	if err != nil {
		return "", err
	}
	if proof.State == "staged" || proof.State == "selected" {
		return proof.Domain, nil
	}
	token, err := a.app.OAuth.ExchangeOwnerCode(ctx, callback)
	if errors.Is(err, wire.ErrShopOwnerRequired) {
		return "", storecenter.ErrShopifyOwnerRequired
	}
	if err != nil {
		return "", storecenter.ErrOfficialConnectionUnavailable
	}
	installation, err := a.app.Admin.Installation(ctx, callback.Shop(), token.Value, a.app.AppID)
	if err != nil {
		return "", storecenter.ErrOfficialConnectionUnavailable
	}
	identity := shopifyIdentity(installation)
	generation, err := a.store.ShopifyInstallGeneration(ctx, identity)
	if err != nil {
		return "", err
	}
	keyID, ciphertext, err := a.app.Protection.SealInstallProof(installProofAAD(proof, installation, generation), token, time.Now())
	if err != nil {
		return "", storecenter.ErrOfficialConnectionUnavailable
	}
	saved, err := a.store.StageShopifyInstall(ctx, proof, identity, generation, keyID, ciphertext, time.Now())
	return saved.Domain, err
}

func (a *ShopifyConnections) InspectStartedInstallation(ctx context.Context, key storecenter.ShopifyPairingKey, browser string) (storecenter.ShopifyInstallSelection, error) {
	if !a.StartedInstallationEnabled() || !validShopifyNonce(browser) {
		return storecenter.ShopifyInstallSelection{}, storecenter.ErrNotFound
	}
	return a.store.InspectShopifyInstall(ctx, key, a.app.AppID, shopifyDigestOf(browser), time.Now())
}

func (a *ShopifyConnections) ConfirmStartedInstallation(ctx context.Context, command storecenter.OfficialConnectionCommand, browser string) (storecenter.ShopifyBinding, error) {
	if !a.StartedInstallationEnabled() || !validShopifyNonce(browser) {
		return storecenter.ShopifyBinding{}, storecenter.ErrNotFound
	}
	session := shopifyDigestOf(browser)
	proof, err := a.store.SelectShopifyInstall(ctx, command, session, time.Now())
	if err != nil {
		return storecenter.ShopifyBinding{}, err
	}
	if proof.AppID != a.app.AppID {
		return storecenter.ShopifyBinding{}, storecenter.ErrVersionConflict
	}
	if proof.State == "confirmed" {
		return a.store.ConfirmShopifyInstall(ctx, command, session, storecenter.ShopifyInstallation{}, time.Now())
	}
	expected := shopifyWireIdentity(proof.ShopifyInstallation)
	token, err := a.app.Protection.OpenInstallProof(installProofAAD(proof, expected, proof.Generation), proof.KeyID, proof.Ciphertext, time.Now())
	if err != nil {
		return storecenter.ShopifyBinding{}, storecenter.ErrOfficialConnectionUnavailable
	}
	actual, err := a.app.Admin.Installation(ctx, expected.Domain, token.Value, a.app.AppID)
	if err != nil || actual != expected {
		return storecenter.ShopifyBinding{}, storecenter.ErrVersionConflict
	}
	return a.store.ConfirmShopifyInstall(ctx, command, session, shopifyIdentity(actual), time.Now())
}

func (a *ShopifyConnections) ExpireStartedInstallationProofs(ctx context.Context) error {
	return a.store.ExpireShopifyInstallProofs(ctx, time.Now())
}

func installProofAAD(proof storecenter.ShopifyInstallProof, installation wire.Installation, generation int64) wire.InstallProofBinding {
	return wire.InstallProofBinding{AttemptID: proof.AttemptID, SessionDigest: proof.SessionDigest,
		Installation: installation, Generation: generation, ExpiresAt: proof.ExpiresAt}
}
