package preparation

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"hash"
	"sort"
	"time"

	"task-processor/internal/product/collection"
)

// UploadPlanInput fixes configuration, not provider facts or authority.
type UploadPlanInput struct {
	PreparationID         string   `json:"preparationId"`
	ExpectedRevision      int64    `json:"expectedRevision"`
	StoreID               string   `json:"storeId"`
	SourceIDs             []string `json:"sourceIds,omitempty"`
	Rule                  string   `json:"rule"`
	PublicationMode       string   `json:"publicationMode"`
	TitleTemplateID       string   `json:"titleTemplateId,omitempty"`
	TitleTemplateRevision string   `json:"titleTemplateRevision,omitempty"`
	TitleQuoteHash        string   `json:"titleQuoteHash,omitempty"`
	TitleModeRevision     string   `json:"titleModeRevision,omitempty"`
}

func (in UploadPlanInput) Operation(action string) OperationInput {
	out := OperationInput{PreparationID: in.PreparationID, ExpectedRevision: in.ExpectedRevision, StoreID: in.StoreID, Action: action}
	if action == OperationOptimize {
		out.TitleTemplateID, out.TitleTemplateRevision = in.TitleTemplateID, in.TitleTemplateRevision
		out.TitleQuoteHash, out.TitleModeRevision = in.TitleQuoteHash, in.TitleModeRevision
	}
	return out
}

func (in UploadPlanInput) Validate() error {
	if in.Rule != "standard" || in.PublicationMode != "platform" && in.PublicationMode != "platform_draft" || len(in.SourceIDs) > 1000 {
		return ErrInvalid
	}
	op := in.Operation(OperationAdapt)
	op.SourceIDs = in.SourceIDs
	if op.Validate() != nil {
		return ErrInvalid
	}
	if in.TitleTemplateID != "" {
		if in.Operation(OperationOptimize).Validate() != nil {
			return ErrInvalid
		}
	} else if in.TitleTemplateRevision != "" || in.TitleQuoteHash != "" || in.TitleModeRevision != "" {
		return ErrInvalid
	}
	seen := map[string]bool{}
	for _, id := range in.SourceIDs {
		if !collection.ValidID(id) || seen[id] {
			return ErrInvalid
		}
		seen[id] = true
	}
	return nil
}

func (in UploadPlanInput) normalized() UploadPlanInput {
	in.SourceIDs = append([]string(nil), in.SourceIDs...)
	sort.Strings(in.SourceIDs)
	return in
}

type UploadPlan struct {
	ID               string          `json:"id"`
	Owner            Scope           `json:"-"`
	Input            UploadPlanInput `json:"input"`
	StoreBindingHash string          `json:"storeBindingHash"`
	Count            int64           `json:"count"`
	Revision         int64           `json:"revision"`
	CreatedAt        time.Time       `json:"createdAt"`
}

type UploadPlanItem struct {
	SourceID            string `json:"sourceId"`
	SourceVersion       uint64 `json:"sourceVersion,string"`
	Revision            int64  `json:"revision"`
	AdaptOperationID    string `json:"adaptOperationId,omitempty"`
	OptimizeOperationID string `json:"optimizeOperationId,omitempty"`
	UploadOperationID   string `json:"uploadOperationId,omitempty"`
}

type UploadPlanReceipt struct {
	Plan     UploadPlan `json:"plan"`
	Replayed bool       `json:"replayed"`
}

// TargetApproval is an exact owner reference. It contains no browser-owned
// readiness or provider payload. Persistence compares it while locking head.
type TargetApproval struct {
	SourceID string `json:"sourceId"`
	RecordID string `json:"recordId"`
	Revision int64  `json:"revision"`
}

func (a TargetApproval) Valid() bool {
	return collection.ValidID(a.SourceID) && collection.ValidID(a.RecordID) && a.Revision > 0
}

// UploadPlanFacts is supplied only by the current owner reader, never HTTP.
// Original operations remain terminal; this projection does not reset them.
type UploadPlanFacts struct {
	HasTarget          bool           `json:"hasTarget"`
	Ready              bool           `json:"ready"`
	Target             TargetApproval `json:"target"`
	PublicationMode    string         `json:"publicationMode"`
	AdaptStatus        string         `json:"adaptStatus,omitempty"`
	OptimizationStatus string         `json:"optimizationStatus,omitempty"`
	ReviewRequired     bool           `json:"reviewRequired"`
	ReviewRejected     bool           `json:"reviewRejected"`
	UploadStatus       string         `json:"uploadStatus,omitempty"`
	UploadConfirmed    bool           `json:"uploadConfirmed"`
}

func NextUploadPlanStep(in UploadPlanInput, item UploadPlanItem, facts UploadPlanFacts) string {
	if item.UploadOperationID != "" {
		if facts.UploadConfirmed {
			return ItemSucceeded
		}
		if ValidItemStatus(facts.UploadStatus) {
			return facts.UploadStatus
		}
		return "unavailable"
	}
	if item.OptimizeOperationID != "" {
		if facts.OptimizationStatus != ItemReview {
			if ValidItemStatus(facts.OptimizationStatus) {
				return facts.OptimizationStatus
			}
			return "unavailable"
		}
		if facts.ReviewRejected {
			return "rejected"
		}
		if facts.ReviewRequired {
			return "review"
		}
	}
	if item.AdaptOperationID != "" && facts.AdaptStatus != ItemSucceeded && facts.AdaptStatus != ItemMissing {
		if ValidItemStatus(facts.AdaptStatus) {
			return facts.AdaptStatus
		}
		return "unavailable"
	}
	if !facts.HasTarget {
		if item.AdaptOperationID != "" {
			return "missing"
		}
		return OperationAdapt
	}
	if !facts.Ready {
		return "missing"
	}
	if in.TitleTemplateID != "" && item.OptimizeOperationID == "" {
		return OperationOptimize
	}
	return OperationUpload
}

type UploadPlanItemView struct {
	Item  UploadPlanItem  `json:"item"`
	Facts UploadPlanFacts `json:"facts"`
	Step  string          `json:"step"`
}

// A preview hash covers the complete ordered selection, including immutable
// target IDs and original operation outcomes, even when only 100 rows display.
type UploadPlanPreviewDigest struct{ h hash.Hash }

func NewUploadPlanPreviewDigest(plan UploadPlan) *UploadPlanPreviewDigest {
	d := &UploadPlanPreviewDigest{h: sha256.New()}
	d.add([]any{plan.ID, plan.Revision, plan.StoreBindingHash, plan.Input})
	return d
}

func (d *UploadPlanPreviewDigest) add(v any) {
	data, _ := json.Marshal(v)
	d.h.Write(data)
	d.h.Write([]byte{'\n'})
}
func (d *UploadPlanPreviewDigest) Add(v UploadPlanItemView) { d.add(v) }
func (d *UploadPlanPreviewDigest) Sum() string              { return hex.EncodeToString(d.h.Sum(nil)) }

func UploadPlanID(scope Scope, key string) string {
	return collection.StableID(scope.OrganizationID, scope.ActorID, "upload-plan", key)
}
