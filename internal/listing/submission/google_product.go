package submission

import (
	"context"
	"time"

	"github.com/henryxu1008/shuomi-ai-commerce/internal/product/collection"
	market "github.com/henryxu1008/shuomi-ai-commerce/internal/storecenter/googlemerchant"
)

const GoogleInsertAction = "google_product_insert"

type GooglePlanScope = collection.Scope

type GooglePlan struct {
	ID            string          `json:"id"`
	Owner         GooglePlanScope `json:"-"`
	SourceID      string          `json:"sourceId"`
	StoreID       string          `json:"storeId"`
	Version       uint64          `json:"version,string"`
	ProductHash   string          `json:"-"`
	InventoryHash string          `json:"-"`
	Target        market.Target   `json:"-"`
	Generation    int64           `json:"-"`
	Offers        []market.Offer  `json:"offers"`
	CreatedAt     time.Time       `json:"createdAt"`
}

func ValidGooglePlanID(id string) bool { return collection.ValidID(id) }

func (p GooglePlan) SameConfirmation(other GooglePlan) bool {
	p.CreatedAt = other.CreatedAt
	return p.Owner == other.Owner && p.Target == other.Target && p.Generation == other.Generation &&
		p.ProductHash == other.ProductHash && p.InventoryHash == other.InventoryHash &&
		collection.Digest(p) == collection.Digest(other)
}

func (p GooglePlan) Valid() bool {
	if !collection.ValidID(p.ID) || p.Owner.Validate() != nil || !collection.ValidID(p.SourceID) ||
		!collection.ValidID(p.StoreID) || p.Version == 0 || !isExecutionDigest(p.ProductHash) ||
		!isExecutionDigest(p.InventoryHash) || p.Generation < 1 || len(p.Offers) < 1 || len(p.Offers) > 10 || p.CreatedAt.IsZero() {
		return false
	}
	seen := map[string]bool{}
	for _, offer := range p.Offers {
		if !offer.ValidFor(p.Target) || seen[offer.ID] {
			return false
		}
		seen[offer.ID] = true
	}
	return true
}

type GoogleRecord struct {
	ID        string    `json:"id"`
	CreatedAt time.Time `json:"createdAt"`
}

type GooglePlanRepository interface {
	Save(context.Context, GooglePlan) (GooglePlan, error)
	Read(context.Context, GooglePlanScope, string) (GooglePlan, error)
	List(context.Context, GooglePlanScope, string) ([]GoogleRecord, error)
}
