package storecenter_test

import (
	"testing"

	"task-processor/internal/storecenter"
)

func TestTikTokStoreUsesExplicitUSSellerCenterCode(t *testing.T) {
	input := validCreateStoreInput()
	input.Platform, input.Region, input.ExternalStoreID = "tiktokshop", "US", "USCODE"
	store, err := storecenter.NewStore(input)
	if err != nil {
		t.Fatalf("US TikTok Store create: %v", err)
	}
	if string(store.Platform()) != "tiktokshop" || store.ExternalStoreID() != "USCODE" {
		t.Fatal("TikTok Shop identity changed")
	}
	if _, err := storecenter.RehydrateStore(store.Snapshot()); err != nil {
		t.Fatal(err)
	}
	if _, err := store.EditBasic("New name", "GB", input.ActorSubject, testUpdatedAt); err == nil {
		t.Fatal("unopened TikTok market accepted through edit")
	}
	for _, invalid := range []struct{ region, code string }{{"GB", "USCODE"}, {"US", ""}} {
		input.Region, input.ExternalStoreID = invalid.region, invalid.code
		if _, err := storecenter.NewStore(input); err == nil {
			t.Fatal("unsupported or unidentified TikTok Shop created")
		}
	}
}
