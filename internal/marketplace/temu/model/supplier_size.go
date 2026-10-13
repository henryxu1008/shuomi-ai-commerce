package model

// E46's normalized fact distinguishes an explicitly empty official mapping
// from an absent, failed or malformed response. CatID is the exact query.
type SupplierSizeRequirement struct {
	CatID    int64              `json:"catId"`
	Observed bool               `json:"observed"`
	Class    *SupplierSizeClass `json:"class,omitempty"`
}
type SupplierSizeClass struct {
	CatID           *int64  `json:"catId"`
	ClassID         *int64  `json:"classId"`
	ParentClassID   *int64  `json:"parentClassId"`
	RelatedClassIDs []int64 `json:"relatedClassIds"`
	ClassType       *int64  `json:"classType"`
}
type SupplierSizePage struct {
	Offset   *int64              `json:"offset"`
	PageSize *int64              `json:"pageSize"`
	Total    *int64              `json:"totalCount"`
	Charts   []SupplierSizeChart `json:"sizeSpecDataList"`
}
type SupplierSizeChart struct {
	ClassID   *int64               `json:"classId"`
	ID        *int64               `json:"businessId"`
	Name      string               `json:"name"`
	Reusable  *bool                `json:"reusable"`
	UpdatedAt *int64               `json:"updatedAt"`
	Content   *SupplierSizeContent `json:"contentDTO"`
}
type SupplierSizeContent struct {
	Records []SupplierSizeRecord `json:"records"`
}
type SupplierSizeRecord struct {
	Values map[string]string `json:"values"`
}

type SupplierSizeSettings struct {
	Code    *int64               `json:"code"`
	Sizes   []string             `json:"sizeList"`
	Mapping *SupplierSizeMapping `json:"mappingContent"`
}
type SupplierSizeMapping struct {
	Records []SupplierSizeRecord `json:"records"`
	Meta    *SupplierSizeMeta    `json:"meta"`
}
type SupplierSizeMeta struct {
	Groups []SupplierSizeGroup `json:"groupList"`
}
type SupplierSizeGroup struct {
	ID          *int64 `json:"id"`
	Name        string `json:"name"`
	Unnecessary *bool  `json:"unnecessary"`
}
type SupplierSizeSettingsObservation struct {
	ClassID  int64                `json:"classId"`
	Settings SupplierSizeSettings `json:"settings"`
}
