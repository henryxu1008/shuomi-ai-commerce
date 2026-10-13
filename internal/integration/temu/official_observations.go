package temu

import (
	"context"
	"encoding/json"
	"net/url"
	"strconv"
	"task-processor/internal/marketplace/temu/model"
	tm "task-processor/internal/marketplace/temu/observations"
	"task-processor/internal/storecenter"
	o "task-processor/internal/storecenter/observations"
	"time"
)

const supplierGoodsListAPI = "bg.glo.goods.list.get"
const supplierLifecycleAPI = "bg.glo.product.search"
const localGoodsListAPI = "temu.local.goods.list.retrieve"
const localPublishStatusAPI = "bg.local.goods.publish.status.get"

func (c *OfficialClient) QueryObservedTemuProducts(ctx context.Context, credential storecenter.OfficialMerchantCredential, query o.ProductQuery) (o.ProductPage, error) {
	var zero o.ProductPage
	if ctx == nil || !c.validCredential(credential) || query.Page < 1 || len(query.PageToken) > 8192 {
		return zero, o.ErrInvalid
	}
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	information, err := c.QueryStore(ctx, credential)
	if err != nil || information.Verification == nil {
		return zero, o.ErrUnavailable
	}
	facts := information.Verification
	if facts.Mode == storecenter.MerchantLocal {
		return c.observedLocalProducts(ctx, credential, query, facts.Scopes)
	}
	if query.PageToken != "" || facts.Mode != storecenter.MerchantFullyManaged && facts.Mode != storecenter.MerchantSemiManaged || !hasScope(facts.Scopes, supplierGoodsListAPI) {
		return zero, o.ErrForbidden
	}
	fields := map[string]json.RawMessage{"page": json.RawMessage(strconv.Itoa(query.Page)), "pageSize": json.RawMessage("10")}
	var page model.SupplierGoodsPage
	if c.readOfficialResponse(ctx, "https://openapi-b-partner.temu.com", supplierGoodsListAPI, credential.Temu.AccessToken, fields, &page) != nil || page.Total == nil || *page.Total < 0 || page.Data == nil || len(page.Data) > 10 {
		return zero, o.ErrUnavailable
	}
	result := o.ProductPage{Total: int(*page.Total), TemuItems: []tm.Product{}, Notes: []string{}}
	seen := map[string]bool{}
	for _, row := range page.Data {
		p := tm.Product{ID: nativeID(row.SKCID), ProductID: nativeID(row.ProductID), Entity: "supplier_skc", Mode: string(facts.Mode), Title: row.Name, ExternalCode: row.ExternalCode, ImageURL: row.ImageURL, SaleState: "unknown", RawStatus: nativeCode(row.SiteStatus), CreatedAt: row.CreatedAt, SKUs: []tm.SKU{}, Sites: []tm.Site{}}
		for _, sku := range row.SKUs {
			p.SKUs = append(p.SKUs, tm.SKU{ID: nativeID(sku.ID), ExternalCode: sku.ExternalCode, Stock: sku.VirtualStock, StockKind: "virtualStock", Specifications: nativeSpecifications(sku.Specifications)})
		}
		if row.Semi != nil {
			for _, site := range row.Semi.Sites {
				if nativeID(site.ID) == "" {
					return zero, o.ErrUnavailable
				}
				p.Sites = append(p.Sites, tm.Site{ID: nativeID(site.ID), Name: site.Name})
			}
		}
		if !tm.ValidProduct(p) || seen[p.ID] || !validObservedImage(p.ImageURL) {
			return zero, o.ErrUnavailable
		}
		seen[p.ID] = true
		result.TemuItems = append(result.TemuItems, p)
	}
	if hasScope(facts.Scopes, supplierLifecycleAPI) {
		if err = c.observedSupplierLifecycle(ctx, credential, &result); err != nil {
			return zero, err
		}
	} else if len(result.TemuItems) > 0 {
		result.Notes = append(result.Notes, "lifecycle_scope_missing")
	}
	return result, nil
}

func (c *OfficialClient) observedLocalProducts(ctx context.Context, credential storecenter.OfficialMerchantCredential, query o.ProductQuery, scopes []string) (o.ProductPage, error) {
	var zero o.ProductPage
	origin := localAPIOrigin(c.application.IdentityPartition)
	if origin == "" || !hasScope(scopes, localGoodsListAPI) {
		return zero, o.ErrForbidden
	}
	fields := map[string]json.RawMessage{"pageSize": json.RawMessage("10"), "goodsSearchType": json.RawMessage(`"ALL"`)}
	if query.PageToken != "" {
		if !boundedParameter(query.PageToken, 8192) {
			return zero, o.ErrInvalid
		}
		fields["pageToken"], _ = json.Marshal(query.PageToken)
	}
	var page model.LocalGoodsPage
	if c.readOfficialResponse(ctx, origin, localGoodsListAPI, credential.Temu.AccessToken, fields, &page) != nil || page.Total == nil || *page.Total < 0 || page.Goods == nil || len(page.Goods) > 10 || page.Pagination == nil {
		return zero, o.ErrUnavailable
	}
	result := o.ProductPage{Total: int(*page.Total), TemuItems: []tm.Product{}, TokenPagination: true, NextToken: page.Pagination.NextToken, Notes: []string{}}
	ids := []int64{}
	seen := map[string]bool{}
	for _, row := range page.Goods {
		p := tm.Product{ID: row.ID, ProductID: row.ID, Entity: "local_goods", Mode: "local", Title: row.Name, ExternalCode: row.ExternalCode, ImageURL: row.ImageURL, SaleState: "unknown", RawStatus: row.Status, CreatedAt: row.CreatedAt, StatusChangedAt: row.StatusChangedAt, SKUCount: row.SKUCount, SKUs: []tm.SKU{}, Sites: []tm.Site{}}
		switch row.Status {
		case "Active":
			p.SaleState = "active"
		case "InActive":
			p.SaleState = "off"
		}
		for _, sku := range row.SKUs {
			p.SKUs = append(p.SKUs, tm.SKU{ID: nativeID(sku.ID), SerialNumber: sku.SerialNumber, Stock: sku.Stock, StockKind: "stock", Specifications: nativeSpecifications(sku.Specifications)})
		}
		if !tm.ValidProduct(p) || seen[p.ID] || !validObservedImage(p.ImageURL) {
			return zero, o.ErrUnavailable
		}
		seen[p.ID] = true
		id, _ := strconv.ParseInt(p.ID, 10, 64)
		ids = append(ids, id)
		result.TemuItems = append(result.TemuItems, p)
	}
	if len(ids) > 0 && hasScope(scopes, localPublishStatusAPI) {
		raw, _ := json.Marshal(ids)
		var states model.LocalPublishStatuses
		if c.readOfficialResponse(ctx, origin, localPublishStatusAPI, credential.Temu.AccessToken, map[string]json.RawMessage{"goodsIdList": raw}, &states) != nil || states.Goods == nil || len(states.Goods) > len(ids) {
			return zero, o.ErrUnavailable
		}
		matched := map[string]bool{}
		for _, state := range states.Goods {
			id := nativeID(state.GoodsID)
			if !seen[id] || matched[id] {
				return zero, o.ErrUnavailable
			}
			matched[id] = true
			for i := range result.TemuItems {
				if result.TemuItems[i].ID == id {
					result.TemuItems[i].PublishStatus = &tm.PublishStatus{Status: state.Status, SubStatus: state.SubStatus}
				}
			}
		}
		if len(matched) != len(ids) {
			result.Notes = append(result.Notes, "publish_status_incomplete")
		}
	} else if len(ids) > 0 {
		result.Notes = append(result.Notes, "publish_status_scope_missing")
	}
	return result, nil
}

func (c *OfficialClient) observedSupplierLifecycle(ctx context.Context, credential storecenter.OfficialMerchantCredential, page *o.ProductPage) error {
	ids := []int64{}
	expected := map[string]bool{}
	for _, p := range page.TemuItems {
		for _, sku := range p.SKUs {
			if !expected[sku.ID] {
				id, _ := strconv.ParseInt(sku.ID, 10, 64)
				ids = append(ids, id)
				expected[sku.ID] = true
			}
		}
	}
	if len(ids) == 0 {
		return nil
	}
	if len(ids) > 100 {
		page.Notes = append(page.Notes, "lifecycle_query_limit")
		return nil
	}
	raw, _ := json.Marshal(ids)
	var states model.SupplierLifecyclePage
	fields := map[string]json.RawMessage{"productSkuIdList": raw, "pageNum": json.RawMessage("1"), "pageSize": json.RawMessage("100")}
	if c.readOfficialResponse(ctx, "https://openapi-b-partner.temu.com", supplierLifecycleAPI, credential.Temu.AccessToken, fields, &states) != nil || states.Total == nil || *states.Total < 0 || states.Products == nil || len(states.Products) > 100 {
		return o.ErrUnavailable
	}
	matched := map[string]bool{}
	for _, product := range states.Products {
		for _, group := range product.SKCs {
			index := -1
			for i, p := range page.TemuItems {
				if p.ProductID == nativeID(product.ID) && p.ID == nativeID(group.ID) {
					index = i
					break
				}
			}
			if index < 0 {
				return o.ErrUnavailable
			}
			p := &page.TemuItems[index]
			p.SelectionStatus = group.SelectionStatus
			for _, sku := range group.SKUs {
				id := nativeID(sku.ID)
				if !expected[id] || matched[id] {
					return o.ErrUnavailable
				}
				matched[id] = true
				found := false
				for i := range p.SKUs {
					if p.SKUs[i].ID == id {
						p.SKUs[i].ApplyJITStatus = sku.ApplyJITStatus
						p.SKUs[i].SuggestCloseJIT = sku.SuggestCloseJIT
						found = true
					}
				}
				if !found {
					return o.ErrUnavailable
				}
			}
		}
	}
	if len(matched) != len(expected) || *states.Total != int64(len(states.Products)) {
		page.Notes = append(page.Notes, "lifecycle_incomplete")
	}
	return nil
}

func nativeID(v *int64) string {
	if v == nil || *v < 1 {
		return ""
	}
	return strconv.FormatInt(*v, 10)
}
func nativeCode(v *int64) string {
	if v == nil {
		return ""
	}
	return strconv.FormatInt(*v, 10)
}
func nativeSpecifications(values []model.SupplierSpecification) []tm.Specification {
	out := []tm.Specification{}
	for _, v := range values {
		out = append(out, tm.Specification{ID: nativeID(v.ID), ParentID: nativeID(v.ParentID), Name: v.Name, ParentName: v.ParentName})
	}
	return out
}

func validObservedImage(value string) bool {
	if value == "" {
		return true
	}
	u, err := url.Parse(value)
	return err == nil && len(value) <= 2048 && u.Scheme == "https" && u.Hostname() != "" && u.User == nil && u.Fragment == ""
}
