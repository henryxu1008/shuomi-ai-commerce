package storecenterapp

import (
	"context"
	"time"

	"github.com/henryxu1008/shuomi-ai-commerce/internal/storecenter"
	tiktok "github.com/henryxu1008/shuomi-ai-commerce/internal/storecenter/tiktokshop"
)

func (a *OfficialConnections) TikTokLifecycleEnabled() bool {
	if a.tiktok == nil {
		return false
	}
	_, verifies := a.tiktok.provider.(tiktok.DeauthorizationVerifier)
	_, persists := a.tiktok.store.(storecenter.TikTokLifecycleStore)
	return verifies && persists
}

func (a *OfficialConnections) ReceiveTikTokLifecycle(ctx context.Context, body []byte, signature string) error {
	if !a.TikTokLifecycleEnabled() {
		return storecenter.ErrOfficialConnectionUnavailable
	}
	event, err := a.tiktok.provider.(tiktok.DeauthorizationVerifier).VerifyDeauthorization(body, signature)
	if err != nil {
		return storecenter.ErrOfficialAuthorizationRejected
	}
	return a.tiktok.store.(storecenter.TikTokLifecycleStore).ApplyTikTokDeauthorization(ctx, event, time.Now().UTC())
}
