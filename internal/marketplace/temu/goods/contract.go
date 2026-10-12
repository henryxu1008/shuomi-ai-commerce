package goods

import (
	"encoding/json"
	"time"

	"task-processor/internal/marketplace/temu/model"
	"task-processor/internal/marketplace/temu/rules"
)

// RuleSnapshot contains observed public platform facts. Authorization belongs
// to Store and source selection, never to this snapshot or its category flag.
type RuleSnapshot struct {
	Selection             model.RuleSelection              `json:"selection"`
	Mode                  string                           `json:"mode"`
	LocalTarget           *rules.LocalTarget               `json:"localTarget,omitempty"`
	SemiSites             []rules.SemiSite                 `json:"semiSites,omitempty"`
	SpecificationRevision string                           `json:"specificationRevision"`
	SpecificationSources  []rules.Source                   `json:"specificationSources"`
	LocalCategories       []model.LocalCategory            `json:"localCategories"`
	SupplierCategories    []model.SupplierCategory         `json:"supplierCategories"`
	SelectableLeaf        bool                             `json:"selectableLeaf"`
	SupplierAttributes    *model.SupplierAttributeTemplate `json:"supplierAttributes,omitempty"`
	LocalFieldRules       *model.LocalCategoryRules        `json:"localFieldRules,omitempty"`
	LocalAttributes       *model.LocalAttributeTemplate    `json:"localAttributes,omitempty"`
	LocalVariations       *model.LocalVariationTemplate    `json:"localVariations,omitempty"`
	FreightTemplates      []model.LocalFreightTemplate     `json:"freightTemplates"`
	Trademarks            *model.LocalTrademarkPage        `json:"trademarks,omitempty"`
	Observations          []RuleObservation                `json:"observations"`
	Issues                []string                         `json:"issues"`
	ObservedAt            time.Time                        `json:"observedAt"`
}

type RuleObservation struct {
	EvidenceID  string `json:"evidenceId"`
	API         string `json:"api"`
	InputHash   string `json:"inputHash"`
	DecodedHash string `json:"decodedHash"`
}

// Exactly one mode-specific body is admitted. Editable platform facts remain
// explicit, including missing quantities, prices and units. URLs in the wire
// body are resolved from these approved asset slots by the current owner.
type DraftInput struct {
	Mode         string                  `json:"mode"`
	Selection    model.RuleSelection     `json:"selection"`
	Local        *model.LocalGoodsV2     `json:"local,omitempty"`
	Supplier     *model.SupplierGoods    `json:"supplier,omitempty"`
	PackageUnits rules.LocalPackageUnits `json:"packageUnits"`
	Images       []ImageSlot             `json:"images"`
}

type ImageSlot struct {
	Field   string `json:"field"`
	SKC     int    `json:"skc"`
	SKU     int    `json:"sku"`
	Sort    int    `json:"sort"`
	AssetID string `json:"assetId"`
	Usage   int64  `json:"usage"`
}

type ImageObservation struct {
	AssetID      string   `json:"assetId"`
	SourceURL    string   `json:"sourceUrl"`
	CatID        int64    `json:"catId"`
	Usage        int64    `json:"usage"`
	ContentHash  string   `json:"contentHash"`
	Width        int      `json:"width"`
	Height       int      `json:"height"`
	Bytes        int64    `json:"bytes"`
	MediaType    string   `json:"mediaType"`
	RemoteURLs   []string `json:"remoteUrls,omitempty"`
	ResponseHash string   `json:"responseHash,omitempty"`
}

type Issue struct {
	Field   string `json:"field"`
	Code    string `json:"code"`
	Message string `json:"message"`
}

type Draft struct {
	Mode              string               `json:"mode"`
	Local             *model.LocalGoodsV2  `json:"local,omitempty"`
	Supplier          *model.SupplierGoods `json:"supplier,omitempty"`
	Images            []ImageSlot          `json:"images"`
	Issues            []Issue              `json:"issues"`
	ReadyForUpload    bool                 `json:"readyForUpload"`
	SubmissionPayload json.RawMessage      `json:"submissionPayload,omitempty"`
}
