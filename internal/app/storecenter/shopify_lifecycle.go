package storecenterapp

import (
	"context"
	"errors"
	"time"

	wire "github.com/henryxu1008/shuomi-ai-commerce/internal/integration/shopify"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/storecenter"
)

func (a *ShopifyConnections) LifecycleNotificationsEnabled() bool {
	return a != nil && a.store.ShopifyLifecycleInstalled() && a.app.LifecycleNotifications
}

func (a *ShopifyConnections) ReceiveShopifyLifecycle(ctx context.Context, topic, eventID, signature string, body []byte) error {
	if !a.LifecycleNotificationsEnabled() {
		return storecenter.ErrOfficialConnectionUnavailable
	}
	notice, err := a.app.OAuth.VerifyAuthorizationNotice(topic, signature, body)
	if errors.Is(err, wire.ErrNoticeSignature) {
		return storecenter.ErrOfficialAuthorizationRejected
	}
	if err != nil {
		return storecenter.ErrNotFound
	}
	if !notice.RequiresCheck {
		return nil
	}
	return a.store.ReceiveShopifyLifecycle(ctx, storecenter.ShopifyLifecycleNotice{
		AppID: a.app.AppID, ShopID: notice.ShopID, Topic: topic, EventID: eventID, BodyHash: notice.BodyHash,
	}, time.Now())
}

// ReconcileShopifyLifecycle runs outside webhook delivery. A transient or
// ambiguous platform result retains the exact event's authorization pause.
func (a *ShopifyConnections) ReconcileShopifyLifecycle(ctx context.Context) error {
	if a == nil || !a.store.ShopifyLifecycleInstalled() {
		return storecenter.ErrOfficialConnectionUnavailable
	}
	for i := 0; i < 4; i++ {
		claim, err := a.store.ClaimShopifyLifecycle(ctx, a.app.AppID, time.Now())
		if err != nil || claim == nil {
			return err
		}
		probeCtx, cancel := context.WithTimeout(ctx, 4*time.Second)
		decision := a.shopifyLifecycleDecision(probeCtx, *claim)
		cancel()
		if err := a.store.ResolveShopifyLifecycle(ctx, *claim, decision, time.Now()); err != nil {
			return err
		}
	}
	return nil
}

func (a *ShopifyConnections) shopifyLifecycleDecision(ctx context.Context, check storecenter.ShopifyLifecycleCheck) storecenter.ShopifyLifecycleDecision {
	token, err := a.app.Issuer.IssueLifecycleProbe(ctx, check.Domain)
	if err != nil {
		return storecenter.ShopifyLifecycleUncertain
	}
	installation, writeProducts, err := a.app.Admin.ReadInstallationAccess(ctx, check.Domain, token.Value, a.app.AppID)
	if err != nil || installation.ShopID != check.ShopID {
		return storecenter.ShopifyLifecycleUncertain
	}
	if installation.ID != check.InstallationID || !writeProducts {
		return storecenter.ShopifyLifecycleRevoked
	}
	return storecenter.ShopifyLifecycleHealthy
}
