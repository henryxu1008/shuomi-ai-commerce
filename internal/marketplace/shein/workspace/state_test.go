package workspace

import (
	"testing"
)

func TestFilterManualReviewNotesDropsAutoNotes(t *testing.T) {
	notes := append([]string{AutoReviewNotes[0], "需要补充品牌说明", AutoReviewNotes[1]}, "需要补充品牌说明")

	filtered := FilterManualReviewNotes(notes)

	if len(filtered) != 1 || filtered[0] != "需要补充品牌说明" {
		t.Fatalf("filtered = %#v", filtered)
	}
}
