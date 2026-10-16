package preparation

import (
	"encoding/hex"
	"task-processor/internal/product/collection"
)

type WalmartUploadApproval struct {
	SourceID string `json:"sourceId"`
	RecordID string `json:"recordId"`
	Revision int64  `json:"revision"`
	WireHash string `json:"wireHash"`
}

func (a WalmartUploadApproval) Valid() bool {
	h, e := hex.DecodeString(a.WireHash)
	return e == nil && len(h) == 32 && hex.EncodeToString(h) == a.WireHash && collection.ValidID(a.SourceID) && collection.ValidID(a.RecordID) && a.Revision > 0
}
func (a WalmartUploadApproval) Matches(source, id string, revision int64, hash string) bool {
	return a.Valid() && a.SourceID == source && a.RecordID == id && a.Revision == revision && a.WireHash == hash
}
