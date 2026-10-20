package model

type AuditReason struct {
	Language string `json:"language"`
	Content  string `json:"content"`
}
type AuditSKC struct {
	Name     string        `json:"skcName"`
	Document string        `json:"documentSn"`
	State    *int          `json:"documentState"`
	Reasons  []AuditReason `json:"failedReason"`
}
type DocumentAudit struct {
	SPU          string     `json:"spuName"`
	Version      string     `json:"version"`
	SKCs         []AuditSKC `json:"skcList"`
	ResponseHash string     `json:"-"`
}
type ShelfSite struct {
	Site  string `json:"siteAbbr"`
	State *int   `json:"shelfStatus"`
}
type SaleSKU struct {
	Code  string `json:"skuCode"`
	State *int   `json:"mallState"`
}
type ShelfSKC struct {
	Name  string      `json:"skcName"`
	Sites []ShelfSite `json:"shelfStatusInfoList"`
	SKUs  []SaleSKU   `json:"skuInfoList"`
}
type ShelfProduct struct {
	SPU          string     `json:"spuName"`
	SKCs         []ShelfSKC `json:"skcInfoList"`
	ResponseHash string     `json:"-"`
}
type WarehouseInventory struct {
	Code   string `json:"warehouseCode"`
	Usable *int64 `json:"usableInventory"`
}
type SKUInventory struct {
	Code       string               `json:"skuCode"`
	Usable     *int64               `json:"totalUsableInventory"`
	Warehouses []WarehouseInventory `json:"warehouseInventoryList"`
}
type GoodsInventory struct {
	SPU  string         `json:"spuName"`
	SKC  string         `json:"skcName"`
	SKUs []SKUInventory `json:"skuList"`
}
type InventorySnapshot struct {
	Type         string           `json:"invType"`
	Goods        []GoodsInventory `json:"goodsInventory"`
	ResponseHash string           `json:"-"`
}
type ShelfChange struct {
	State   int      `json:"shelf_state"`
	Sites   []string `json:"site_list"`
	SKCName string   `json:"skc_name"`
}
type ShelfResult struct {
	SKC          string                `json:"skc"`
	Site         string                `json:"site"`
	ResponseHash string                `json:"responseHash"`
	Response     *ShelfResponseSummary `json:"response,omitempty"`
}

// A bounded diagnostic, never evidence that an UNKNOWN request had no effect.
type ShelfResponseSummary struct {
	Code         string         `json:"code"`
	Message      string         `json:"message"`
	ResponseHash string         `json:"responseHash"`
	Failures     []ShelfFailure `json:"failures"`
}
type ShelfFailure struct {
	Identity string `json:"identity"`
	Code     string `json:"code"`
	Message  string `json:"message"`
}
