package submission

import (
	"encoding/json"
	"time"

	marketplace "task-processor/internal/marketplace/walmart"
	"task-processor/internal/product/collection"
)

type WalmartReadPage struct {
	Offset       int       `json:"offset"`
	Count        int       `json:"count"`
	ResponseHash string    `json:"responseHash"`
	ObservedAt   time.Time `json:"observedAt"`
}

type WalmartReadProgress struct {
	Round            int64             `json:"round"`
	ExpectedHash     string            `json:"expectedHash"`
	StartOffset      int               `json:"startOffset"`
	NextOffset       int               `json:"nextOffset"`
	PreviousWindowID string            `json:"previousWindowId,omitempty"`
	Pages            []WalmartReadPage `json:"pages"`
}

// The whole document has one canonical hash. Native facts and local read
// progress are persisted by the same original Submission observation owner.
type WalmartReadDocument struct {
	marketplace.FeedObservation
	ReadProgress *WalmartReadProgress `json:"readProgress,omitempty"`
}

type WalmartObservedPage struct {
	Offset      int                         `json:"offset"`
	Observation marketplace.FeedObservation `json:"observation"`
}

func DecodeWalmartReadDocument(raw []byte, hash string, receipt OfficialReceipt, generation int64) (WalmartReadDocument, error) {
	var doc WalmartReadDocument
	if len(raw) > MaxExecutionPayloadBytes || json.Unmarshal(raw, &doc) != nil || collection.Digest(doc) != hash || ValidateWalmartReadDocument(doc, receipt, generation) != nil {
		return doc, ErrExecutionEvidenceRequired
	}
	return doc, nil
}

func ValidateWalmartReadDocument(doc WalmartReadDocument, receipt OfficialReceipt, generation int64) error {
	if receipt.Walmart == nil || doc.FeedID != receipt.Walmart.FeedID || doc.ObservedAt.IsZero() || len(doc.Items) > marketplace.MaxSKUs {
		return ErrExecutionEvidenceRequired
	}
	p := doc.ReadProgress
	if p == nil {
		return nil // Current single native observation, with no resumable cursor.
	}
	expected := len(receipt.Walmart.SKUs)
	if p.Round < 1 || p.Round > generation || p.ExpectedHash != collection.Digest(WalmartExpectedSKUs(receipt.Walmart.SKUs)) || p.StartOffset < 0 || p.StartOffset >= expected || p.StartOffset%50 != 0 || p.NextOffset < p.StartOffset || p.NextOffset > expected || len(p.Pages) > 20 || len(p.Pages) != len(doc.ResponseHashes) || (p.StartOffset == 0 && (p.Round != generation || p.PreviousWindowID != "")) || (p.StartOffset > 0 && (!collection.ValidID(p.PreviousWindowID) || !doc.Partial)) {
		return ErrExecutionEvidenceRequired
	}
	if len(p.Pages) == 0 && (doc.FeedStatus != "" || doc.Received != 0 || doc.Succeeded != 0 || doc.Failed != 0 || doc.Processing != 0 || !doc.Partial) {
		return ErrExecutionEvidenceRequired
	}
	offset, count := p.StartOffset, 0
	for n, page := range p.Pages {
		if page.Offset != offset || page.Count < 0 || page.Count > min(50, expected-offset) || !isExecutionDigest(page.ResponseHash) || page.ResponseHash != doc.ResponseHashes[n] || page.ObservedAt.IsZero() || n+1 < len(p.Pages) && page.Count != 50 {
			return ErrExecutionEvidenceRequired
		}
		offset += page.Count
		count += page.Count
	}
	if offset != p.NextOffset || count != len(doc.Items) {
		return ErrExecutionEvidenceRequired
	}
	return nil
}

// StartWalmartRead derives the next position from the original committed head,
// not from the number of displayed items or a caller-selected cursor.
func StartWalmartRead(req WalmartResultRequest, previous *WalmartResultRecord) (WalmartResultRequest, error) {
	if req.Receipt.Walmart == nil || req.Generation < 1 {
		return req, ErrExecutionEvidenceRequired
	}
	p := &WalmartReadProgress{Round: req.Generation, ExpectedHash: collection.Digest(WalmartExpectedSKUs(req.Receipt.Walmart.SKUs)), Pages: []WalmartReadPage{}}
	if previous != nil {
		if previous.ReceiptID != req.Receipt.ID || previous.Generation >= req.Generation || ValidateWalmartReadDocument(WalmartReadDocument{FeedObservation: previous.Observation, ReadProgress: previous.ReadProgress}, req.Receipt, previous.Generation) != nil {
			return req, ErrExecutionEvidenceRequired
		}
		req.BaseObservationID = previous.ID
		old := previous.ReadProgress
		if old != nil && old.NextOffset > 0 && old.NextOffset < len(req.Receipt.Walmart.SKUs) && (len(old.Pages) == 0 || old.Pages[len(old.Pages)-1].Count == 50) {
			p.Round, p.StartOffset, p.NextOffset = old.Round, old.NextOffset, old.NextOffset
			p.PreviousWindowID = old.PreviousWindowID
			if len(old.Pages) > 0 {
				p.PreviousWindowID = previous.ID
			}
		}
	}
	req.ReadProgress = p
	return req, nil
}

func (r *WalmartResultRequest) ObserveWalmartPage(offset int, page marketplace.FeedObservation) error {
	p := r.ReadProgress
	if r.Receipt.Walmart == nil || p == nil || offset != p.NextOffset || offset >= len(r.Receipt.Walmart.SKUs) || page.FeedID != r.Receipt.Walmart.FeedID || len(page.Items) > min(50, len(r.Receipt.Walmart.SKUs)-offset) || len(page.ResponseHashes) != 1 || !isExecutionDigest(page.ResponseHashes[0]) || page.ObservedAt.IsZero() || len(p.Pages) >= 20 {
		return ErrExecutionEvidenceRequired
	}
	p.Pages = append(p.Pages, WalmartReadPage{Offset: offset, Count: len(page.Items), ResponseHash: page.ResponseHashes[0], ObservedAt: page.ObservedAt})
	p.NextOffset = offset + len(page.Items)
	return nil
}

func WalmartReadDocumentFor(req WalmartResultRequest, observation marketplace.FeedObservation) (WalmartReadDocument, error) {
	doc := WalmartReadDocument{FeedObservation: observation, ReadProgress: req.ReadProgress}
	if doc.ReadProgress != nil && doc.ReadProgress.StartOffset > 0 {
		doc.Partial = true
	}
	if !doc.FeedObservation.CompleteFor(WalmartExpectedSKUs(req.Receipt.Walmart.SKUs)) {
		doc.Partial = true
	}
	if err := ValidateWalmartReadDocument(doc, req.Receipt, req.Generation); err != nil {
		return doc, err
	}
	return doc, nil
}

func WalmartReadFits(req WalmartResultRequest, observation marketplace.FeedObservation) bool {
	doc, err := WalmartReadDocumentFor(req, observation)
	if err != nil {
		return false
	}
	raw, err := json.Marshal(doc)
	return err == nil && len(raw) <= MaxExecutionPayloadBytes
}

// AppendWalmartPage leaves the admitted prefix unchanged when the next whole
// typed window exceeds the original persistence budget.
func (r WalmartResultRequest) AppendWalmartPage(offset int, current, page marketplace.FeedObservation) (WalmartResultRequest, marketplace.FeedObservation, error) {
	if r.ReadProgress == nil {
		return r, current, ErrExecutionEvidenceRequired
	}
	progress := *r.ReadProgress
	progress.Pages = append([]WalmartReadPage(nil), progress.Pages...)
	next := r
	next.ReadProgress = &progress
	if err := next.ObserveWalmartPage(offset, page); err != nil {
		return r, current, err
	}
	window := page
	if len(r.ReadProgress.Pages) > 0 {
		window = current
		window.Items = append(append([]marketplace.ItemResult(nil), current.Items...), page.Items...)
		window.ResponseHashes = append(append([]string(nil), current.ResponseHashes...), page.ResponseHashes...)
		window.ObservedAt = page.ObservedAt
	}
	if !WalmartReadFits(next, window) {
		return r, current, ErrExecutionInvalid
	}
	return next, window, nil
}

func (doc WalmartReadDocument) ObservedPages() []WalmartObservedPage {
	if doc.ReadProgress == nil {
		return nil
	}
	pages := make([]WalmartObservedPage, 0, len(doc.ReadProgress.Pages))
	start := 0
	for _, proof := range doc.ReadProgress.Pages {
		page := doc.FeedObservation
		page.Items = append([]marketplace.ItemResult(nil), doc.Items[start:start+proof.Count]...)
		page.ObservedAt, page.ResponseHashes, page.Partial = proof.ObservedAt, []string{proof.ResponseHash}, true
		pages = append(pages, WalmartObservedPage{Offset: proof.Offset, Observation: page})
		start += proof.Count
	}
	return pages
}
