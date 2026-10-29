package storecenter_test

import (
	"context"
	"errors"
	"github.com/google/uuid"
	"task-processor/internal/storecenter"
	"testing"
	"time"
)

func TestBrowserAuditIsScopedAndCursorCannotMoveAcrossIdentity(t *testing.T) {
	_, a, _, s, store := browserFixture(t)
	ctx := context.Background()
	key := uuid.NewString()
	if _, err := s.Admit(ctx, "org-a", store, key, browserCommand()); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Report(ctx, store, key, storecenter.BrowserReport{SchemaVersion: "store-browser-v1", Kind: "open_result", Outcome: storecenter.AuditOutcomeSucceeded, ReasonCode: "process_started", ObservedAt: time.Now().UTC()}); err != nil {
		t.Fatal(err)
	}
	q := storecenter.BrowserAuditQuery{Limit: 1, From: time.Now().Add(-24 * time.Hour).UTC().Format(time.RFC3339Nano), To: time.Now().Add(time.Hour).UTC().Format(time.RFC3339Nano)}
	page, err := s.Audit(ctx, "org-a", store, q)
	if err != nil || len(page.Items) != 1 || page.NextCursor == nil {
		t.Fatalf("first audit page: %+v %v", page, err)
	}
	firstID := page.Items[0].EventID
	q.Cursor = *page.NextCursor
	next, err := s.Audit(ctx, "org-a", store, q)
	if err != nil || len(next.Items) != 1 || next.Items[0].EventID == firstID {
		t.Fatalf("cursor repeated row: %+v %v", next, err)
	}
	a.subject = "other-subject"
	if _, err = s.Audit(ctx, "org-a", store, q); !errors.Is(err, storecenter.ErrBrowserInvalid) {
		t.Fatalf("cursor moved identity: %v", err)
	}
	a.subject = "subject-create"
	q.Actor = "someone-else"
	if _, err = s.Audit(ctx, "org-a", store, q); !errors.Is(err, storecenter.ErrBrowserDenied) {
		t.Fatalf("member inspected another actor: %v", err)
	}
	q.Actor = ""
	q.Cursor = ""
	q.Action = "open_anything"
	if _, err = s.Audit(ctx, "org-a", store, q); !errors.Is(err, storecenter.ErrBrowserInvalid) {
		t.Fatalf("unbounded action accepted: %v", err)
	}
}
