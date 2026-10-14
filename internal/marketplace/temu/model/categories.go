package model

// Supplier and local metadata retain their distinct official fields. Neither
// a category response nor a selectable leaf grants publishing permission.
type SupplierCategoryQuery struct {
	ParentCatID int64
	SiteID      int64
}

type LocalCategoryQuery struct {
	ParentCatID int64
	Language    string
}

type SupplierCategory struct {
	CatID       int64  `json:"catId"`
	Name        string `json:"catName"`
	ParentCatID *int64 `json:"parentCatId"`
	CatType     *int64 `json:"catType"`
	Leaf        *bool  `json:"isLeaf"`
	Level       *int64 `json:"catLevel"`
	Hidden      *bool  `json:"isHidden"`
	HiddenType  *int64 `json:"hiddenType"`
}

func (c SupplierCategory) SelectableLeaf() bool {
	return c.CatID > 0 && c.ParentCatID != nil && *c.ParentCatID >= 0 && c.Leaf != nil && *c.Leaf && c.CatType != nil && c.Level != nil && *c.Level > 0 && c.Hidden != nil && !*c.Hidden && c.HiddenType != nil && *c.HiddenType == 0
}

type LocalCategory struct {
	CatID              int64  `json:"catId"`
	Name               string `json:"catName"`
	ParentID           *int64 `json:"parentId"`
	CatType            *int64 `json:"catType"`
	Leaf               *bool  `json:"leaf"`
	Level              *int64 `json:"level"`
	AvailableStatus    *int64 `json:"availableStatus"`
	ExpandCatType      *int64 `json:"expandCatType"`
	SecondHandCategory *bool  `json:"secondHandCategory"`
}

func (c LocalCategory) SelectableLeaf() bool {
	return c.CatID > 0 && c.ParentID != nil && *c.ParentID >= 0 && c.Leaf != nil && *c.Leaf && c.CatType != nil && (*c.CatType == 0 || *c.CatType == 1) && c.Level != nil && *c.Level >= 1 && *c.Level <= 4 && c.AvailableStatus != nil && *c.AvailableStatus == 0
}
