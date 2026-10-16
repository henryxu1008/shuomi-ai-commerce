package walmart

import "testing"

func TestFeedCountsCannotContradictSKUsAndErrorProvesNoAdmission(t *testing.T) {
	o := FeedObservation{FeedID: "feed", FeedStatus: "PROCESSED", Received: 2, Succeeded: 1, Failed: 1, Items: []ItemResult{{SKU: "a", IngestionStatus: "DATA_ERROR"}, {SKU: "b", IngestionStatus: "DATA_ERROR"}}}
	if o.CompleteFor([]string{"a", "b"}) || o.AllowsRepair([]string{"a", "b"}, []string{"a"}) {
		t.Fatal("contradictory counts granted resend")
	}
	o = FeedObservation{FeedID: "feed", FeedStatus: "ERROR"}
	if !o.AllowsRepair([]string{"a", "b"}, []string{"b"}) {
		t.Fatal("file rejected with no processed items must allow explicit corrected subset")
	}
}
