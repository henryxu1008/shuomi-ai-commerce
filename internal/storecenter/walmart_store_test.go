package storecenter_test

import (
	"task-processor/internal/storecenter"
	"testing"
)

func TestWalmartStoreOnlyUSAndMerchantIdentityNotManuallyRequired(t *testing.T) {
	in := validCreateStoreInput()
	in.Platform = "walmart"
	in.Region = "US"
	in.ExternalStoreID = ""
	s, e := storecenter.NewStore(in)
	if e != nil {
		t.Fatal(e)
	}
	if s.Platform() != storecenter.PlatformWalmart {
		t.Fatal("wrong platform")
	}
	if _, e = storecenter.RehydrateStore(s.Snapshot()); e != nil {
		t.Fatal(e)
	}
	if _, e = s.EditBasic("Store", "CA", in.ActorSubject, testUpdatedAt); e == nil {
		t.Fatal("other market opened")
	}
}
