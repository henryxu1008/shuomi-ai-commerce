package tiktokshop

import "sort"

// ProductStatus is a current read-only observation, never a mutation receipt.
type ProductStatus struct {
	ProductID   string   `json:"productId"`
	Status      string   `json:"status"`
	AuditStatus string   `json:"auditStatus"`
	Reasons     []string `json:"reasons"`
}

type ProductStatusExpectation struct {
	ProductID       string
	LocalSKUIDs     []string
	GlobalProductID string
	Mappings        []PublishedSKU
}

func DecodeProductStatus(raw []byte, expected ProductStatusExpectation) (ProductStatus, error) {
	if !identifier(expected.ProductID) || len(expected.LocalSKUIDs) < 1 || len(expected.LocalSKUIDs) > 300 {
		return ProductStatus{}, ErrPublicationInvalid
	}
	wanted := map[string]bool{}
	for _, id := range expected.LocalSKUIDs {
		if !identifier(id) || wanted[id] {
			return ProductStatus{}, ErrPublicationInvalid
		}
		wanted[id] = true
	}
	var response struct {
		Code *int `json:"code"`
		Data *struct {
			ID     string `json:"id"`
			Status string `json:"status"`
			SKUs   []struct {
				ID string `json:"id"`
			} `json:"skus"`
			Audit struct {
				Status       string   `json:"status"`
				PreApproved  []string `json:"pre_approved_reasons"`
				NotSubmitted []string `json:"not_submitted_reasons"`
			} `json:"audit"`
			Global struct {
				ID       string `json:"global_product_id"`
				Mappings []struct {
					Local  string `json:"local_sku_id"`
					Global string `json:"global_sku_id"`
				} `json:"sku_mappings"`
			} `json:"global_product_association"`
		} `json:"data"`
	}
	if !decode(raw, &response) || response.Code == nil || *response.Code != 0 || response.Data == nil || response.Data.ID != expected.ProductID || len(response.Data.SKUs) != len(wanted) {
		return ProductStatus{}, ErrPublicationUnknown
	}
	d := response.Data
	switch d.Status {
	case "DRAFT", "PENDING", "FAILED", "ACTIVATE", "SELLER_DEACTIVATED", "PLATFORM_DEACTIVATED", "FREEZE", "DELETED", "SCHEDULED":
	default:
		return ProductStatus{}, ErrPublicationUnknown
	}
	switch d.Audit.Status {
	case "NONE", "AUDITING", "FAILED", "PRE_APPROVED", "APPROVED":
	default:
		return ProductStatus{}, ErrPublicationUnknown
	}
	seen := map[string]bool{}
	for _, sku := range d.SKUs {
		if !wanted[sku.ID] || seen[sku.ID] {
			return ProductStatus{}, ErrPublicationUnknown
		}
		seen[sku.ID] = true
	}
	if expected.GlobalProductID != "" {
		if !identifier(expected.GlobalProductID) || d.Global.ID != expected.GlobalProductID || len(expected.Mappings) != len(wanted) || len(d.Global.Mappings) != len(wanted) {
			return ProductStatus{}, ErrPublicationUnknown
		}
		mapping := map[string]string{}
		globals := map[string]bool{}
		for _, sku := range expected.Mappings {
			if !wanted[sku.LocalSKUID] || !identifier(sku.GlobalSKUID) || mapping[sku.LocalSKUID] != "" || globals[sku.GlobalSKUID] {
				return ProductStatus{}, ErrPublicationInvalid
			}
			mapping[sku.LocalSKUID] = sku.GlobalSKUID
			globals[sku.GlobalSKUID] = true
		}
		seen = map[string]bool{}
		for _, sku := range d.Global.Mappings {
			if mapping[sku.Local] != sku.Global || sku.Global == "" || seen[sku.Local] {
				return ProductStatus{}, ErrPublicationUnknown
			}
			seen[sku.Local] = true
		}
	} else if len(expected.Mappings) != 0 {
		return ProductStatus{}, ErrPublicationInvalid
	}
	reasons := map[string]bool{}
	if d.Audit.Status == "PRE_APPROVED" {
		for _, reason := range d.Audit.PreApproved {
			if reason == "KYC_PENDING" || reason == "RESTRICTED_CATEGORY_PENDING" {
				reasons[reason] = true
			}
		}
	}
	if d.Audit.Status == "NONE" {
		for _, reason := range d.Audit.NotSubmitted {
			if reason == "PAYMENT_ACCOUNT_NOT_LINKED" || reason == "SHOP_INACTIVE" || reason == "W8_NOT_CONFIGURED" {
				reasons[reason] = true
			}
		}
	}
	value := ProductStatus{ProductID: d.ID, Status: d.Status, AuditStatus: d.Audit.Status, Reasons: []string{}}
	for reason := range reasons {
		value.Reasons = append(value.Reasons, reason)
	}
	sort.Strings(value.Reasons)
	return value, nil
}
