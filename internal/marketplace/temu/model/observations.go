package model

// Named read-only protocols. Supplier list entries are SKCs, not products;
// local retrieval uses an opaque pageToken and distinct published status data.
type SupplierGoodsPage struct {
	Data  []SupplierGoodsSummary `json:"data"`
	Total *int64                 `json:"totalCount"`
}
type SupplierGoodsSummary struct {
	ProductID    *int64                `json:"productId"`
	SKCID        *int64                `json:"productSkcId"`
	Name         string                `json:"productName"`
	ExternalCode string                `json:"extCode"`
	ImageURL     string                `json:"mainImageUrl"`
	SiteStatus   *int64                `json:"skcSiteStatus"`
	CreatedAt    *int64                `json:"createdAt"`
	SKUs         []SupplierObservedSKU `json:"productSkuSummaries"`
	Semi         *SupplierObservedSemi `json:"productSemiManaged"`
}
type SupplierObservedSKU struct {
	ID             *int64                  `json:"productSkuId"`
	ExternalCode   string                  `json:"extCode"`
	VirtualStock   *int64                  `json:"virtualStock"`
	Specifications []SupplierSpecification `json:"productSkuSpecList"`
}
type SupplierObservedSemi struct {
	Sites []SupplierObservedSite `json:"bindSites"`
}
type SupplierObservedSite struct {
	ID   *int64 `json:"siteId"`
	Name string `json:"siteName"`
}

type LocalGoodsPage struct {
	Goods      []LocalRetrievedGoods `json:"goodsList"`
	Total      *int64                `json:"total"`
	Pagination *LocalGoodsPagination `json:"pagination"`
}
type LocalGoodsPagination struct {
	NextToken     *string `json:"nextToken"`
	PreviousToken *string `json:"previousToken"`
}
type LocalRetrievedGoods struct {
	ID              string              `json:"goodsId"`
	Name            string              `json:"goodsName"`
	Status          string              `json:"goodsStatus"`
	ExternalCode    string              `json:"outGoodsSn"`
	ImageURL        string              `json:"thumbUrl"`
	CategoryID      string              `json:"catId"`
	CreatedAt       *int64              `json:"goodsCreateTime"`
	StatusChangedAt *int64              `json:"goodsStatusChangeTime"`
	SKUCount        *int64              `json:"variationsCount"`
	SKUs            []LocalRetrievedSKU `json:"skuInfoList"`
}
type LocalRetrievedSKU struct {
	ID             *int64                  `json:"skuId"`
	SerialNumber   string                  `json:"skuSn"`
	Stock          *int64                  `json:"stock"`
	Specifications []SupplierSpecification `json:"specList"`
}
type LocalPublishStatuses struct {
	Goods []LocalPublishStatus `json:"goodsPublishStatusList"`
}
type LocalPublishStatus struct {
	GoodsID   *int64 `json:"goodsId"`
	Status    *int64 `json:"status"`
	SubStatus *int64 `json:"subStatus"`
}

type SupplierLifecyclePage struct {
	Total    *int64                     `json:"total"`
	Products []SupplierLifecycleProduct `json:"dataList"`
}
type SupplierLifecycleProduct struct {
	ID   *int64                 `json:"productId"`
	SKCs []SupplierLifecycleSKC `json:"skcList"`
}
type SupplierLifecycleSKC struct {
	ID              *int64                 `json:"skcId"`
	SelectionStatus *int64                 `json:"selectStatus"`
	SKUs            []SupplierLifecycleSKU `json:"skuList"`
}
type SupplierLifecycleSKU struct {
	ID              *int64 `json:"skuId"`
	ApplyJITStatus  *int64 `json:"applyJitStatus"`
	SuggestCloseJIT *bool  `json:"suggestCloseJit"`
}
