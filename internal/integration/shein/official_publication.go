package shein

import (
	"context"
	"encoding/json"
	"strings"
	"task-processor/internal/marketplace/shein/model"
	"task-processor/internal/storecenter"
)

func (c *OfficialClient) QueryDocumentState(ctx context.Context, k storecenter.OfficialMerchantCredential, spu, version string) (model.DocumentAudit, error) {
	if !boundedParameter(spu, 128) || !boundedParameter(version, 128) {
		return model.DocumentAudit{}, ErrGoodsInvalid
	}
	payload, _ := json.Marshal(map[string]any{"spuList": []any{map[string]string{"spuName": spu, "version": version}}})
	raw, e := c.goodsPost(ctx, "/open-api/goods/query-document-state", k, payload)
	if e != nil {
		return model.DocumentAudit{}, e
	}
	var response struct {
		Code string `json:"code"`
		Info struct {
			Data []model.DocumentAudit `json:"data"`
		} `json:"info"`
	}
	if decodeGoods(raw, &response) != nil || response.Code != "0" || len(response.Info.Data) != 1 {
		return model.DocumentAudit{}, ErrGoodsUnavailable
	}
	v := response.Info.Data[0]
	if v.SPU != spu || v.Version != version || len(v.SKCs) > 40 {
		return model.DocumentAudit{}, ErrGoodsUnavailable
	}
	seen := map[string]bool{}
	for _, s := range v.SKCs {
		if !boundedParameter(s.Name, 128) || seen[s.Name] || len(s.Reasons) > 20 {
			return model.DocumentAudit{}, ErrGoodsUnavailable
		}
		seen[s.Name] = true
		for _, r := range s.Reasons {
			if len(r.Language) > 32 || len(r.Content) > 2048 {
				return model.DocumentAudit{}, ErrGoodsUnavailable
			}
		}
	}
	v.ResponseHash = goodsHash(raw)
	return v, nil
}
func (c *OfficialClient) QueryShelfProduct(ctx context.Context, k storecenter.OfficialMerchantCredential, spu string) (model.ShelfProduct, error) {
	if !boundedParameter(spu, 128) {
		return model.ShelfProduct{}, ErrGoodsInvalid
	}
	payload, _ := json.Marshal(map[string]any{"spuName": spu, "languageList": []string{"en"}})
	raw, e := c.goodsPost(ctx, "/open-api/goods/spu-info", k, payload)
	if e != nil {
		return model.ShelfProduct{}, e
	}
	var response struct {
		Code string              `json:"code"`
		Info *model.ShelfProduct `json:"info"`
	}
	if decodeGoods(raw, &response) != nil || response.Code != "0" || response.Info == nil || response.Info.SPU != spu || len(response.Info.SKCs) > 40 {
		return model.ShelfProduct{}, ErrGoodsUnavailable
	}
	seen := map[string]bool{}
	skus := map[string]bool{}
	for _, s := range response.Info.SKCs {
		if !boundedParameter(s.Name, 128) || seen[s.Name] || len(s.SKUs) > 400 || len(s.Sites) > 200 {
			return model.ShelfProduct{}, ErrGoodsUnavailable
		}
		seen[s.Name] = true
		sites := map[string]bool{}
		for _, site := range s.Sites {
			if sites[site.Site] {
				return model.ShelfProduct{}, ErrGoodsUnavailable
			}
			sites[site.Site] = true
		}
		for _, sku := range s.SKUs {
			if !boundedParameter(sku.Code, 128) || skus[sku.Code] {
				return model.ShelfProduct{}, ErrGoodsUnavailable
			}
			skus[sku.Code] = true
		}
	}
	response.Info.ResponseHash = goodsHash(raw)
	return *response.Info, nil
}
func (c *OfficialClient) QuerySKUInventory(ctx context.Context, k storecenter.OfficialMerchantCredential, skus []string, invType string) (model.InventorySnapshot, error) {
	if len(skus) < 1 || len(skus) > 100 || (invType != "VI" && invType != "PI" && invType != "JI") {
		return model.InventorySnapshot{}, ErrGoodsInvalid
	}
	seen := map[string]bool{}
	for _, id := range skus {
		if !boundedParameter(id, 128) || seen[id] {
			return model.InventorySnapshot{}, ErrGoodsInvalid
		}
		seen[id] = true
	}
	payload, _ := json.Marshal(map[string]any{"skuCodeList": skus, "invType": invType})
	raw, e := c.goodsPost(ctx, "/open-api/stock/stock-query", k, payload)
	if e != nil {
		return model.InventorySnapshot{}, e
	}
	var response struct {
		Code string `json:"code"`
		Info []struct {
			Goods []model.GoodsInventory `json:"goodsInventory"`
		} `json:"info"`
	}
	if decodeGoods(raw, &response) != nil || response.Code != "0" || len(response.Info) > 100 {
		return model.InventorySnapshot{}, ErrGoodsUnavailable
	}
	out := model.InventorySnapshot{Type: invType, ResponseHash: goodsHash(raw)}
	found := map[string]bool{}
	for _, group := range response.Info {
		for _, g := range group.Goods {
			if !boundedParameter(g.SPU, 128) || !boundedParameter(g.SKC, 128) || len(g.SKUs) > 100 {
				return model.InventorySnapshot{}, ErrGoodsUnavailable
			}
			for _, sku := range g.SKUs {
				if !seen[sku.Code] || found[sku.Code] || len(sku.Warehouses) > 200 || sku.Usable != nil && *sku.Usable < 0 {
					return model.InventorySnapshot{}, ErrGoodsUnavailable
				}
				found[sku.Code] = true
				wseen := map[string]bool{}
				for _, w := range sku.Warehouses {
					if !boundedParameter(w.Code, 128) || wseen[w.Code] || w.Usable != nil && *w.Usable < 0 {
						return model.InventorySnapshot{}, ErrGoodsUnavailable
					}
					wseen[w.Code] = true
				}
			}
			out.Goods = append(out.Goods, g)
		}
	}
	return out, nil
}
func (c *OfficialClient) ShelfOn(ctx context.Context, k storecenter.OfficialMerchantCredential, skc string) (model.ShelfResult, error) {
	if !boundedParameter(skc, 128) {
		return model.ShelfResult{}, ErrGoodsInvalid
	}
	payload, _ := json.Marshal(struct {
		Items []model.ShelfChange `json:"skc_site_info_list"`
	}{[]model.ShelfChange{{State: 1, Sites: []string{"shein-us"}, SKCName: skc}}})
	raw, e := c.goodsPost(ctx, "/open-api/goods/modify-skc-shelf", k, payload)
	if e != nil {
		return model.ShelfResult{}, ErrGoodsOutcomeUnknown
	}
	var response struct {
		Code    string `json:"code"`
		Message string `json:"msg"`
		Info    *struct {
			Total    *int `json:"total_count"`
			Success  *int `json:"success_count"`
			Failure  *int `json:"failure_count"`
			Failures []struct {
				Identity string `json:"identity"`
				Code     string `json:"code"`
				Message  string `json:"msg"`
			} `json:"failure_results"`
		} `json:"info"`
	}
	if decodeGoods(raw, &response) != nil {
		return model.ShelfResult{}, ErrGoodsOutcomeUnknown
	}
	if response.Code != "0" || response.Info == nil || response.Info.Total == nil || *response.Info.Total != 1 || response.Info.Success == nil || *response.Info.Success != 1 || response.Info.Failure == nil || *response.Info.Failure != 0 || len(response.Info.Failures) > 0 {
		clean := func(s string, n int) string {
			for _, secret := range []string{k.SecretKey, k.OpenKeyID} {
				if secret != "" {
					s = strings.ReplaceAll(s, secret, "[redacted]")
				}
			}
			s = strings.ReplaceAll(s, "\x00", "")
			r := []rune(s)
			if len(r) > n {
				r = r[:n]
			}
			return string(r)
		}
		summary := &model.ShelfResponseSummary{Code: clean(response.Code, 128), Message: clean(response.Message, 512), ResponseHash: goodsHash(raw), Failures: []model.ShelfFailure{}}
		if response.Info != nil {
			for i, f := range response.Info.Failures {
				if i >= 20 {
					break
				}
				summary.Failures = append(summary.Failures, model.ShelfFailure{Identity: clean(f.Identity, 128), Code: clean(f.Code, 128), Message: clean(f.Message, 512)})
			}
		}
		return model.ShelfResult{SKC: skc, Site: "shein-us", Response: summary}, ErrGoodsOutcomeUnknown
	}
	return model.ShelfResult{SKC: skc, Site: "shein-us", ResponseHash: goodsHash(raw)}, nil
}
