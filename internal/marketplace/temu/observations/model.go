package observations

import (
	"strconv"
	"strings"
)

// Product preserves native entity, product/SKC identities, original codes and
// inventory meanings. No field here establishes Submission success or consent.
type Product struct {
	ID              string         `json:"id"`
	ProductID       string         `json:"productId"`
	Entity          string         `json:"entity"`
	Mode            string         `json:"mode"`
	Title           string         `json:"title"`
	ExternalCode    string         `json:"externalCode"`
	ImageURL        string         `json:"imageUrl"`
	SaleState       string         `json:"saleState"`
	RawStatus       string         `json:"rawStatus"`
	CreatedAt       *int64         `json:"createdAt,omitempty,string"`
	StatusChangedAt *int64         `json:"statusChangedAt,omitempty,string"`
	SKUCount        *int64         `json:"skuCount,omitempty,string"`
	SKUs            []SKU          `json:"skus"`
	Sites           []Site         `json:"sites"`
	PublishStatus   *PublishStatus `json:"publishStatus,omitempty"`
	SelectionStatus *int64         `json:"selectionStatus,omitempty,string"`
}
type SKU struct {
	ID              string          `json:"id"`
	ExternalCode    string          `json:"externalCode"`
	SerialNumber    string          `json:"serialNumber"`
	Stock           *int64          `json:"stock,omitempty,string"`
	StockKind       string          `json:"stockKind"`
	Specifications  []Specification `json:"specifications"`
	ApplyJITStatus  *int64          `json:"applyJitStatus,omitempty,string"`
	SuggestCloseJIT *bool           `json:"suggestCloseJit,omitempty"`
}
type Specification struct {
	ID         string `json:"id"`
	ParentID   string `json:"parentId"`
	Name       string `json:"name"`
	ParentName string `json:"parentName"`
}
type Site struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}
type PublishStatus struct {
	Status    *int64 `json:"status,string"`
	SubStatus *int64 `json:"subStatus,string"`
}

func ValidIdentifier(id string) bool {
	v, err := strconv.ParseInt(id, 10, 64)
	return err == nil && v > 0 && strconv.FormatInt(v, 10) == id
}
func ValidProduct(p Product) bool {
	if !ValidIdentifier(p.ID) || !ValidIdentifier(p.ProductID) || p.SaleState != "active" && p.SaleState != "off" && p.SaleState != "unknown" || strings.ContainsRune(p.Title, 0) || len(p.SKUs) > 1000 || len(p.Sites) > 100 {
		return false
	}
	if p.Mode == "local" {
		if p.Entity != "local_goods" || p.ProductID != p.ID || len(p.Sites) > 0 {
			return false
		}
	} else if p.Mode != "fully_managed" && p.Mode != "semi_managed" || p.Entity != "supplier_skc" || p.PublishStatus != nil {
		return false
	}
	seen := map[string]bool{}
	for _, sku := range p.SKUs {
		if !ValidIdentifier(sku.ID) || seen[sku.ID] || p.Mode == "local" && sku.StockKind != "stock" || p.Mode != "local" && sku.StockKind != "virtualStock" {
			return false
		}
		seen[sku.ID] = true
	}
	return true
}
