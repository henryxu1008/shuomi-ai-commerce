package automaticfx

import (
	"context"
	"errors"
	"testing"
	"time"

	d "task-processor/internal/aicapability/routeconfig"
)

type fakeOwner struct {
	receipt                d.Revision
	receiptErr, publishErr error
	publishes              int
}

func (f *fakeOwner) Receipt(context.Context, d.Scope, string) (d.Revision, error) {
	return f.receipt, f.receiptErr
}
func (f *fakeOwner) Publish(_ context.Context, actor, key string, in d.PlatformPricesInput, deadline time.Time) (d.Revision, error) {
	f.publishes++
	if f.publishErr != nil {
		return d.Revision{}, f.publishErr
	}
	hash, _ := publicationFingerprint(actor, in, deadline)
	f.receipt = d.Revision{Scope: d.PlatformPricesScope(), ActorID: actor, Key: key, Fingerprint: hash}
	f.receiptErr = nil
	return f.receipt, nil
}

func TestOwnerReceiptReplayAndUnknownNeverRedispatch(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 11, 12, 1, 30, 0, 0, time.UTC)
	in := d.PlatformPricesInput{ExpectedRevision: "fixture-revision", FX: []d.PublishedFXRate{{Currency: "USD", RateDate: "2026-11-12", SourceURL: officialCFETSSource, VerifiedAt: now, EffectiveFrom: now, EffectiveUntil: now.Add(time.Hour)}}}
	f := &fakeOwner{receiptErr: d.ErrNotFound}
	if _, err := PublishFirstAttempt(ctx, f, "2026-11-12", 1, in); err != nil {
		t.Fatal(err)
	}
	if _, err := InspectPublication(ctx, f, "2026-11-12", 1, in); err != nil {
		t.Fatal(err)
	}
	if f.publishes != 1 {
		t.Fatal("inspection dispatched a duplicate")
	}
	f.receipt.Fingerprint = "different"
	if _, err := InspectPublication(ctx, f, "2026-11-12", 1, in); !errors.Is(err, ErrPublicationConflict) {
		t.Fatal("mismatched receipt accepted")
	}
	g := &fakeOwner{receiptErr: d.ErrNotFound, publishErr: d.ErrOutcomeUnknown}
	if _, err := PublishFirstAttempt(ctx, g, "2026-11-12", 1, in); !errors.Is(err, d.ErrOutcomeUnknown) {
		t.Fatalf("got %v", err)
	}
	if _, err := InspectPublication(ctx, g, "2026-11-12", 1, in); !errors.Is(err, d.ErrOutcomeUnknown) {
		t.Fatalf("got %v", err)
	}
	if g.publishes != 1 {
		t.Fatal("unknown was resent")
	}
}
