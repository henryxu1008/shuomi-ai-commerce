package target

import (
	"task-processor/internal/listing/completiondraft"
	"task-processor/internal/listing/preparation"
	tiktok "task-processor/internal/marketplace/tiktokshop"
	"testing"
)

func TestTikTokCompletionVersionMustMatchExactSavedSourceAndText(t *testing.T) {
	source := preparation.SourceItem{ID: "00000000-0000-4000-8000-000000000001"}
	saved := completiondraft.Record{ID: "00000000-0000-4000-8000-000000000002", Revision: 3, Source: source, Content: completiondraft.Content{Title: "Saved title", Description: "Saved description"}}
	input := TargetInput{SourceID: source.ID, TikTok: &tiktok.DraftInput{Title: saved.Content.Title, Description: saved.Content.Description}, CompletionDraft: &CompletionDraftSelection{RecordID: saved.ID, Revision: 3}}
	if validateCompletionVersion(input, source, saved) != nil {
		t.Fatal("selected saved version rejected")
	}
	for _, change := range []func(*completiondraft.Record){func(v *completiondraft.Record) { v.Source.ID = "another-source" }, func(v *completiondraft.Record) { v.Revision++ }, func(v *completiondraft.Record) { v.Content.Title = "different-title" }, func(v *completiondraft.Record) { v.ID = "another-record" }} {
		v := saved
		change(&v)
		if validateCompletionVersion(input, source, v) == nil {
			t.Fatal("different saved version accepted")
		}
	}
}
