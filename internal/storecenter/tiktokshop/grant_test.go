package tiktokshop

import (
	"errors"
	"strings"
	"testing"
	"time"
)

var grantTime = time.Unix(1791800000, 0).UTC()

const grantOperation = "00000000-0000-4000-8000-000000000003"

func fixtureGrant(t *testing.T) *Grant {
	t.Helper()
	g, err := RestoreGrant(GrantSnapshot{OrganizationID: "org-a", ID: "00000000-0000-4000-8000-000000000001", AppKey: "app-a", AppRevision: "r1", SellerIdentityHash: strings.Repeat("a", 64), Generation: 1, Revision: 1, Status: GrantActive, KeyID: "key-1", Ciphertext: "sealed-generation-1", AccessExpiresAt: grantTime.Add(time.Hour), RefreshExpiresAt: grantTime.Add(24 * time.Hour), CreatedAt: grantTime, UpdatedAt: grantTime})
	if err != nil {
		t.Fatal(err)
	}
	return g
}

func TestReauthorizationFencesAnEarlierRefreshResult(t *testing.T) {
	g := fixtureGrant(t)
	fence, err := g.ClaimRefresh(1, grantOperation, grantTime, grantTime.Add(time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	if g.CanSend(grantTime) {
		t.Fatal("refresh in flight exposes old credential")
	}
	if err := g.Reauthorize(g.Snapshot().Revision, "key-1", "sealed-reauthorization", grantTime.Add(time.Hour), grantTime.Add(24*time.Hour), grantTime.Add(time.Second)); err != nil {
		t.Fatal(err)
	}
	if err := g.CompleteRefresh(fence, "key-1", "sealed-late-refresh", grantTime.Add(time.Hour), grantTime.Add(24*time.Hour), grantTime.Add(2*time.Second)); !errors.Is(err, ErrGrantConflict) {
		t.Fatal("late refresh overwrote reauthorization")
	}
	snapshot := g.Snapshot()
	if snapshot.Generation != 2 || snapshot.Ciphertext != "sealed-reauthorization" || !g.CanSend(grantTime.Add(2*time.Second)) {
		t.Fatal("canonical new generation lost")
	}
}

func TestRefreshUnknownAndRevocationNeverRestoreOldToken(t *testing.T) {
	g := fixtureGrant(t)
	fence, err := g.ClaimRefresh(1, grantOperation, grantTime, grantTime.Add(time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := g.ClaimRefresh(g.Snapshot().Revision, grantOperation, grantTime, grantTime.Add(time.Minute)); !errors.Is(err, ErrGrantConflict) {
		t.Fatal("second dispatcher accepted")
	}
	if err := g.MarkRefreshUnknown(fence, grantTime.Add(time.Second)); err != nil {
		t.Fatal(err)
	}
	if g.CanSend(grantTime.Add(2 * time.Second)) {
		t.Fatal("UNKNOWN used old access token")
	}
	if err := g.CompleteRefresh(fence, "key-1", "late", grantTime.Add(time.Hour), grantTime.Add(24*time.Hour), grantTime.Add(2*time.Second)); err == nil {
		t.Fatal("unknown operation resumed implicitly")
	}
	if err := g.Revoke(g.Snapshot().Revision, grantTime.Add(3*time.Second)); err != nil {
		t.Fatal(err)
	}
	if err := g.Reauthorize(1, "key-1", "old", grantTime.Add(time.Hour), grantTime.Add(24*time.Hour), grantTime.Add(4*time.Second)); err == nil {
		t.Fatal("stale authorization restored revoked grant")
	}
}

func TestExpiredRefreshClaimRequiresExplicitUnknownAndNewAuthorization(t *testing.T) {
	g := fixtureGrant(t)
	fence, err := g.ClaimRefresh(1, grantOperation, grantTime, grantTime.Add(time.Second))
	if err != nil {
		t.Fatal(err)
	}
	if err := g.CompleteRefresh(fence, "key-1", "expired", grantTime.Add(time.Hour), grantTime.Add(24*time.Hour), grantTime.Add(time.Second)); err == nil {
		t.Fatal("expired claim committed")
	}
	if _, err := g.ClaimRefresh(g.Snapshot().Revision, grantOperation, grantTime.Add(time.Second), grantTime.Add(time.Minute)); err == nil {
		t.Fatal("expired dispatch blindly reclaimed")
	}
	if err := g.MarkRefreshUnknown(fence, grantTime.Add(time.Second)); err != nil {
		t.Fatal(err)
	}
	if err := g.Reauthorize(g.Snapshot().Revision, "key-1", "new-consent", grantTime.Add(time.Hour), grantTime.Add(24*time.Hour), grantTime.Add(2*time.Second)); err != nil {
		t.Fatal(err)
	}
	if !g.CanSend(grantTime.Add(2 * time.Second)) {
		t.Fatal("fresh fenced reauthorization unusable")
	}
}
