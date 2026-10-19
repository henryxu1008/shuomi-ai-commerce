package goods

import (
	"math"
	"task-processor/internal/marketplace/shein/model"
	"time"
)

type PublicationSKUObservation struct {
	Code                string `json:"code"`
	MallState           *int   `json:"mallState"`
	Usable              *int64 `json:"usable"`
	ReportedUsable      *int64 `json:"reportedUsable"`
	InventoryScopeKnown bool   `json:"inventoryScopeKnown"`
}
type PublicationSKCObservation struct {
	SKC            string                      `json:"skc"`
	AuditState     *int                        `json:"auditState"`
	AuditLabel     string                      `json:"auditLabel"`
	Reasons        []model.AuditReason         `json:"reasons"`
	ShelfState     *int                        `json:"shelfState"`
	ShelfWay       string                      `json:"shelfWay"`
	HopeOnSaleDate string                      `json:"hopeOnSaleDate"`
	ShelfRequire   string                      `json:"shelfRequire"`
	CanShelf       bool                        `json:"canShelf"`
	BlockingReason string                      `json:"blockingReason"`
	SKUs           []PublicationSKUObservation `json:"skus"`
}

func PublishedSKCPlan(payload model.PublishProduct, remote model.PublishedSKC) (model.ProductSKC, bool) {
	for _, local := range payload.SKCs {
		if len(local.SKUs) != len(remote.SKUs) {
			continue
		}
		match := true
		for _, sku := range remote.SKUs {
			found := false
			for _, candidate := range local.SKUs {
				found = found || candidate.SupplierSKU == sku.SupplierSKU
			}
			match = match && found
		}
		if match {
			return local, true
		}
	}
	return model.ProductSKC{}, false
}

// UsableInUS combines only explicit quantities for current US-selling warehouses.
// A provider total or missing warehouse is never a zero-stock proof.
func UsableInUS(sku model.SKUInventory, warehouses []model.Warehouse) (*int64, bool) {
	total := int64(0)
	needed := 0
	for _, w := range warehouses {
		us := false
		for _, country := range w.SaleCountries {
			us = us || country == "US"
		}
		if !us {
			continue
		}
		needed++
		found := false
		for _, stock := range sku.Warehouses {
			if stock.Code == w.Code {
				if found || stock.Usable == nil || *stock.Usable < 0 || *stock.Usable > math.MaxInt64-total {
					return nil, false
				}
				found = true
				total += *stock.Usable
			}
		}
		if !found {
			return nil, false
		}
	}
	if needed == 0 {
		return nil, false
	}
	return &total, true
}
func ObservePublication(published model.PublishResult, payload model.PublishProduct, mode model.ApplicationMode, audit model.DocumentAudit, live model.ShelfProduct, stock model.InventorySnapshot, warehouses []model.Warehouse, now time.Time, skuPage map[string]bool) []PublicationSKCObservation {
	out := []PublicationSKCObservation{}
	for _, remote := range published.SKCs {
		plan, matched := PublishedSKCPlan(payload, remote)
		s := PublicationSKCObservation{SKC: remote.SKCName, ShelfWay: plan.ShelfWay, HopeOnSaleDate: plan.HopeOnSaleDate, ShelfRequire: plan.ShelfRequire, Reasons: []model.AuditReason{}, SKUs: []PublicationSKUObservation{}}
		if audit.SPU == published.SPUName && audit.Version == published.Version && published.Version != "" {
			for _, a := range audit.SKCs {
				if a.Name == remote.SKCName {
					s.AuditState = a.State
					s.Reasons = a.Reasons
				}
			}
		}
		if live.SPU == published.SPUName {
			for _, group := range live.SKCs {
				if group.Name != remote.SKCName {
					continue
				}
				for _, site := range group.Sites {
					if site.Site == "shein-us" {
						s.ShelfState = site.State
					}
				}
			}
		}
		s.AuditLabel = AuditLabel(s.AuditState)
		s.CanShelf, s.BlockingReason = ShelfEligibility(mode, plan, s.AuditState, s.ShelfState, now)
		s.CanShelf = s.CanShelf && matched
		for _, remoteSKU := range remote.SKUs {
			if !skuPage[remoteSKU.SKUCode] {
				continue
			}
			sku := PublicationSKUObservation{Code: remoteSKU.SKUCode}
			if live.SPU == published.SPUName {
				for _, group := range live.SKCs {
					if group.Name == remote.SKCName {
						for _, sale := range group.SKUs {
							if sale.Code == remoteSKU.SKUCode {
								sku.MallState = sale.State
							}
						}
					}
				}
			}
			for _, group := range stock.Goods {
				if group.SPU != published.SPUName || group.SKC != remote.SKCName {
					continue
				}
				for _, inv := range group.SKUs {
					if inv.Code == remoteSKU.SKUCode {
						sku.ReportedUsable = inv.Usable
						// The original fully managed record does not prove whether PI
						// or JI applies. Arrival and sample flags cannot establish it.
						if (mode == model.ModeSelfOperated || mode == model.ModeSemiManaged) && stock.Type == "VI" {
							sku.Usable, sku.InventoryScopeKnown = UsableInUS(inv, warehouses)
						}
					}
				}
			}
			s.SKUs = append(s.SKUs, sku)
		}
		out = append(out, s)
	}
	return out
}
