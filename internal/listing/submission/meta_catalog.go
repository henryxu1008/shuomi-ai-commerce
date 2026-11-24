package submission

import (
	"context"
	"net/url"
	"strings"
	"time"

	"github.com/henryxu1008/shuomi-ai-commerce/internal/product/collection"
	meta "github.com/henryxu1008/shuomi-ai-commerce/internal/storecenter/metacatalog"
)

const MetaCatalogCreateAction = "meta_catalog_create"

type MetaPlanScope = collection.Scope

type MetaPlan struct {
	ID           string       `json:"id"`
	SourceID     string       `json:"sourceId"`
	StoreID      string       `json:"storeId"`
	CatalogID    string       `json:"catalogId"`
	BusinessID   string       `json:"businessId"`
	WhatsAppID   string       `json:"whatsAppId,omitempty"`
	VariantLabel string       `json:"variantLabel"`
	Product      meta.Product `json:"product"`
	CreatedAt    time.Time    `json:"createdAt"`

	Owner         MetaPlanScope `json:"-"`
	Generation    int64         `json:"-"`
	Version       uint64        `json:"-"`
	ProductHash   string        `json:"-"`
	InventoryHash string        `json:"-"`
}

func (p MetaPlan) SameConfirmation(q MetaPlan) bool {
	p.CreatedAt = q.CreatedAt
	return p == q
}

func (p MetaPlan) Valid() bool {
	return collection.ValidID(p.ID) && collection.ValidID(p.SourceID) && collection.ValidID(p.StoreID) &&
		p.Owner.Validate() == nil && p.Generation > 0 && p.Version > 0 && isExecutionDigest(p.ProductHash) && isExecutionDigest(p.InventoryHash) &&
		meta.ValidID(p.CatalogID) && meta.ValidID(p.BusinessID) && (p.WhatsAppID == "" || meta.ValidID(p.WhatsAppID)) && !p.CreatedAt.IsZero() &&
		strings.HasPrefix(p.Product.RetailerID, "sm-") && isExecutionDigest(strings.TrimPrefix(p.Product.RetailerID, "sm-")) &&
		boundedMetaText(p.VariantLabel, 512) && boundedMetaText(p.Product.Title, 200) && boundedMetaText(p.Product.Description, 4096) && boundedMetaText(p.Product.Brand, 100) &&
		validMetaURL(p.Product.ProductURL) && validMetaURL(p.Product.ImageURL) && p.Product.PriceCents > 0 && p.Product.PriceCents <= 1000000000
}

func boundedMetaText(s string, max int) bool { return strings.TrimSpace(s) != "" && len(s) <= max }

func validMetaURL(s string) bool {
	u, err := url.Parse(s)
	return err == nil && len(s) <= 2048 && u.Scheme == "https" && u.Hostname() != "" && u.User == nil && u.Fragment == ""
}

type MetaRecord struct {
	ID        string    `json:"id"`
	CreatedAt time.Time `json:"createdAt"`
}

type MetaPlanRepository interface {
	Save(context.Context, MetaPlan) (MetaPlan, error)
	Read(context.Context, MetaPlanScope, string) (MetaPlan, error)
	List(context.Context, MetaPlanScope, string) ([]MetaRecord, error)
}
