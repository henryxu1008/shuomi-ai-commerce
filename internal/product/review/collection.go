package review

import (
	"context"
	reviewproposal "task-processor/internal/product/review/proposal"
)

const MaxPageSize = 100

type PageCursor struct {
	ID string
}

type PageRequest struct {
	Limit  int
	Cursor *PageCursor
}

func (r PageRequest) Validate() error {
	if r.Limit < 1 || r.Limit > MaxPageSize {
		return ErrInvalid
	}
	if r.Cursor != nil && !validID(r.Cursor.ID) {
		return ErrInvalid
	}
	return nil
}

type CollectionItem struct {
	Execution        *ExecutionProvenance
	ContentSelection reviewproposal.ContentSelection
	ID               string
	ProductKey       string
	BaseVersion      uint64
	Revision         uint64
	State            string
}

type Page struct {
	Items      []CollectionItem
	NextCursor *PageCursor
}

func (s *Service) List(ctx context.Context, request PageRequest) (Page, error) {
	ctx, cancel := context.WithTimeout(ctx, Timeout)
	defer cancel()
	scope, err := s.authorize(ctx, false, false)
	if err != nil {
		return Page{}, err
	}
	if err := request.Validate(); err != nil {
		return Page{}, err
	}
	if s.sourceReader == nil {
		return Page{}, ErrUnavailable
	}
	ctx, err = s.sourceReader.AuthorizeRead(ctx)
	if err != nil {
		return Page{}, mapSourceReadError(err)
	}
	page, err := s.store.List(ctx, scope, request)
	if err != nil {
		return Page{}, err
	}
	if request.Cursor != nil {
		anchor, e := s.store.Read(ctx, scope, request.Cursor.ID)
		if e != nil || s.authorizeWorkbench(ctx, anchor.Execution) != nil {
			return Page{}, ErrInvalid
		}
	}
	kept := page.Items[:0]
	for _, item := range page.Items {
		if s.authorizeWorkbench(ctx, item.Execution) != nil {
			continue
		}
		kept = append(kept, item)
	}
	if len(kept) == 0 && len(page.Items) != 0 {
		return Page{}, ErrUnavailable
	}
	if len(kept) != len(page.Items) && page.NextCursor != nil {
		if len(kept) == 0 {
			return Page{}, ErrUnavailable
		}
		page.NextCursor = &PageCursor{ID: kept[len(kept)-1].ID}
	}
	page.Items = kept
	return page, nil
}

// CollectionItemForPersistence projects a validated bounded persistence row.
func CollectionItemForPersistence(record Record) (CollectionItem, error) {
	if err := ValidateStoredRecord(record); err != nil {
		return CollectionItem{}, err
	}
	if record.State != "pending" && record.State != "accepted" {
		return CollectionItem{}, ErrUnavailable
	}
	item := CollectionItem{ID: record.ID, Execution: record.Execution, ProductKey: record.Input.ProductKey, BaseVersion: record.Input.BaseVersion, Revision: record.Revision, State: record.State}
	if record.Content != nil {
		item.ContentSelection = record.Content.Selection
	}
	if err := ValidateCollectionItem(item); err != nil {
		return CollectionItem{}, err
	}
	return item, nil
}

func ValidateCollectionItem(item CollectionItem) error {
	if !item.ContentSelection.Absent() && !item.ContentSelection.Valid() {
		return ErrUnavailable
	}
	if !validID(item.ID) || !ValidKey(item.ProductKey) || item.BaseVersion == 0 || item.BaseVersion > 1<<63-1 || item.Revision == 0 || item.Revision > 1<<63-1 || (item.State != "pending" && item.State != "accepted") {
		return ErrUnavailable
	}
	return nil
}
