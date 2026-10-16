package preparation

import (
	"task-processor/internal/product/collection"
	"testing"
)

func TestWalmartUploadApprovalPinsRecordRevisionAndWire(t *testing.T) {
	a := WalmartUploadApproval{SourceID: "00000000-0000-4000-8000-000000000001", RecordID: "00000000-0000-4000-8000-000000000002", Revision: 2, WireHash: collection.Digest("approved feed")}
	if !a.Matches(a.SourceID, a.RecordID, 2, a.WireHash) {
		t.Fatal("exact approval rejected")
	}
	if a.Matches(a.SourceID, a.RecordID, 3, a.WireHash) || a.Matches(a.SourceID, a.RecordID, 2, collection.Digest("changed feed")) {
		t.Fatal("approval followed a mutable head")
	}
}
