package storecenterapp

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/authidentity"

	wire "github.com/henryxu1008/shuomi-ai-commerce/internal/integration/shopify"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/storecenter"
)

type shopifyTrialLifecycle struct {
	binding storecenter.ShopifyBinding
	freeze  func(context.Context, string, string, int64) error
}

// NewShopifyTrialConnections uses the normal pairing and authorization owner
// but never automatically resumes an installation after a stop notice.
func NewShopifyTrialConnections(store *storecenter.MemberScopedStoreRepository, app ShopifyApplication, binding storecenter.ShopifyBinding,
	freeze func(context.Context, string, string, int64) error) (*ShopifyConnections, error) {
	connections, err := NewShopifyConnections(store, app)
	shop, invalidShop := uuid.Parse(binding.ShopKey)
	record, invalidRecord := uuid.Parse(binding.StoreID)
	if err != nil || !connections.LifecycleNotificationsEnabled() || freeze == nil || binding.AppID != app.AppID ||
		!authidentity.IsBoundedIdentifier(binding.OrganizationID) || binding.ShopID == "" || binding.InstallationID == "" ||
		binding.Generation < 1 || invalidShop != nil || shop == uuid.Nil || shop.String() != binding.ShopKey ||
		invalidRecord != nil || record == uuid.Nil || record.String() != binding.StoreID {
		return nil, storecenter.ErrOfficialConnectionUnavailable
	}
	connections.trialLifecycle = &shopifyTrialLifecycle{binding: binding, freeze: freeze}
	return connections, nil
}

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
	event := storecenter.ShopifyLifecycleNotice{AppID: a.app.AppID, ShopID: notice.ShopID, Topic: topic,
		EventID: eventID, BodyHash: notice.BodyHash}
	if a.trialLifecycle == nil {
		return a.store.ReceiveShopifyLifecycle(ctx, event, time.Now())
	}
	binding := a.trialLifecycle.binding
	if err := a.store.ReceivePairedShopifyTrialLifecycle(ctx, event, time.Now(), binding); err != nil {
		return err
	}
	if err := a.trialLifecycle.freeze(ctx, binding.OrganizationID, binding.ShopKey, binding.Generation); err != nil {
		return storecenter.ErrOfficialConnectionUnavailable
	}
	return nil
}

// ReconcileShopifyLifecycle runs outside webhook delivery. A transient or
// ambiguous platform result retains the exact event's authorization pause.
func (a *ShopifyConnections) ReconcileShopifyLifecycle(ctx context.Context) error {
	if a == nil || !a.store.ShopifyLifecycleInstalled() {
		return storecenter.ErrOfficialConnectionUnavailable
	}
	if a.trialLifecycle != nil {
		return nil
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
