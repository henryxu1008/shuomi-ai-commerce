package target

import (
	"context"

	"task-processor/internal/listing/completiondraft"
	"task-processor/internal/listing/preparation"
	"task-processor/internal/product/collection"
)

type CompletionDraftSelection struct {
	RecordID string `json:"recordId"`
	Revision int64  `json:"revision"`
}
type TargetCompletionVersionReader interface {
	ReadVersion(context.Context, collection.Scope, string) (completiondraft.Record, error)
}

func validateCompletionVersion(input TargetInput, source preparation.SourceItem, saved completiondraft.Record) error {
	ref := input.CompletionDraft
	if input.Walmart != nil {
		if ref == nil || !collection.ValidID(ref.RecordID) || ref.Revision < 1 || saved.ID != ref.RecordID || saved.Revision != ref.Revision || saved.Source != source || input.SourceID != source.ID {
			return ErrConflict
		}
		// Only explicit text adoption is supported. Structured candidate facts
		// remain in CompletionDraft and must be entered/checked separately.
		for _, sku := range input.Walmart.SKUs {
			if sku.Visible["productName"] != saved.Content.Title || sku.Visible["shortDescription"] != saved.Content.Description {
				return ErrConflict
			}
		}
		return nil
	}
	if ref == nil || input.TikTok == nil || !collection.ValidID(ref.RecordID) || ref.Revision < 1 || saved.ID != ref.RecordID || saved.Revision != ref.Revision || saved.Source != source || input.SourceID != source.ID || input.TikTok.Title != saved.Content.Title || input.TikTok.Description != saved.Content.Description {
		return ErrConflict
	}
	return nil
}
