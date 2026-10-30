package officialpersistence

import (
	"encoding/json"

	"gorm.io/gorm"
	"task-processor/internal/listing/submission"
)

func readWalmartWindow(tx *gorm.DB, receipt submission.OfficialReceipt, id string) (walmartObservationRow, submission.WalmartReadDocument, error) {
	var row walmartObservationRow
	if err := tx.Where("organization_id=? AND receipt_id=? AND id=?", receipt.Owner.OrganizationID, receipt.ID, id).Take(&row).Error; err != nil {
		return row, submission.WalmartReadDocument{}, submission.ErrExecutionEvidenceRequired
	}
	doc, err := submission.DecodeWalmartReadDocument(row.BodyJSON, row.BodyHash, receipt, row.Generation)
	return row, doc, err
}

func (r *OfficialRepository) startWalmartRead(tx *gorm.DB, req submission.WalmartResultRequest, anchor string) (submission.WalmartResultRequest, error) {
	var previous *submission.WalmartResultRecord
	if anchor != "" {
		row, doc, err := readWalmartWindow(tx, req.Receipt, anchor)
		if err != nil {
			return req, err
		}
		previous = &submission.WalmartResultRecord{ID: row.ID, ReceiptID: row.ReceiptID, Generation: row.Generation, Observation: doc.FeedObservation, ReadProgress: doc.ReadProgress}
	}
	return submission.StartWalmartRead(req, previous)
}

func sameWalmartReadStart(want, got submission.WalmartResultRequest) bool {
	a, b := want.ReadProgress, got.ReadProgress
	return a != nil && b != nil && want.BaseObservationID == got.BaseObservationID && a.Round == b.Round && a.ExpectedHash == b.ExpectedHash && a.StartOffset == b.StartOffset && a.PreviousWindowID == b.PreviousWindowID
}

func (r *OfficialRepository) walmartPreviousPages(tx *gorm.DB, receipt submission.OfficialReceipt, row walmartObservationRow, doc submission.WalmartReadDocument) ([]submission.WalmartObservedPage, error) {
	if doc.ReadProgress == nil {
		return nil, nil
	}
	progress, generation := doc.ReadProgress, row.Generation
	id, offset := progress.PreviousWindowID, progress.StartOffset
	pages := []submission.WalmartObservedPage{}
	for windows := 0; id != ""; windows++ {
		if windows >= 20 {
			return nil, submission.ErrExecutionEvidenceRequired
		}
		prior, previous, err := readWalmartWindow(tx, receipt, id)
		p := previous.ReadProgress
		if err != nil || prior.Generation >= generation || p == nil || len(p.Pages) == 0 || p.Round != progress.Round || p.ExpectedHash != progress.ExpectedHash || p.NextOffset != offset || p.StartOffset >= offset {
			return nil, submission.ErrExecutionEvidenceRequired
		}
		pages = append(previous.ObservedPages(), pages...)
		if len(pages) > 20 {
			return nil, submission.ErrExecutionEvidenceRequired
		}
		id, offset, generation = p.PreviousWindowID, p.StartOffset, prior.Generation
	}
	if offset != 0 {
		return nil, submission.ErrExecutionEvidenceRequired
	}
	raw, err := json.Marshal(pages)
	if err != nil || len(raw) > submission.MaxExecutionPayloadBytes {
		return nil, submission.ErrExecutionUnavailable
	}
	return pages, nil
}
