package storecenter

import (
	"testing"
	"time"

	tiktok "task-processor/internal/storecenter/tiktokshop"
)

func TestTikTokConnectionProjectsPendingRefreshWithoutChangingAttempt(t *testing.T) {
	now := time.Now().UTC()
	attempt := tiktokAttemptRow{State: "bound"}
	for _, status := range []tiktok.GrantStatus{tiktok.GrantRefreshDispatched, tiktok.GrantRefreshUnknown} {
		t.Run(string(status), func(t *testing.T) {
			grant := tiktok.GrantSnapshot{ID: "grant", Status: status, Generation: 3, AccessExpiresAt: now.Add(-time.Minute), RefreshExpiresAt: now.Add(time.Hour)}
			view := tiktokConnectionView(officialConnectionRow{}, attempt, grant, tiktokBindingRow{}, now)
			if view.State != string(status) {
				t.Fatalf("canonical refresh state hidden behind original attempt: %q", view.State)
			}
			if attempt.State != "bound" || grant.Status != status {
				t.Fatal("read projection changed canonical facts")
			}
			if view.TikTok.GrantGeneration != 3 {
				t.Fatal("refresh generation lost")
			}
		})
	}
}
