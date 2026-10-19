package listingadmin

import "strings"

func (s listingStore) toStore() Store {
	return Store{
		ID:          s.ID,
		TenantID:    s.TenantID,
		OwnerUserID: strings.TrimSpace(firstNonEmptyStoreValue(s.OwnerUserID, s.CreatedBy, s.Creator)),
		CreatedBy:   strings.TrimSpace(firstNonEmptyStoreValue(s.CreatedBy, s.Creator)),
		UpdatedBy:   strings.TrimSpace(firstNonEmptyStoreValue(s.UpdatedBy, s.Updater)),
		StoreID:     s.StoreID,
		Name:        s.Name,
		Username:    s.Username,
		Password:    s.Password,
		LoginURL:    s.LoginURL,
		ShopType:    s.ShopType,
		Region:      s.Region,
		Platform:    s.Platform,
		Proxy:       s.Proxy,
		Remark:      s.Remark,
		Status:      s.Status,
		ValidFrom:   s.ValidFrom,
		ValidUntil:  s.ValidUntil,
		Expired:     s.Expired,
		CreateTime:  s.CreateTime,
		UpdateTime:  s.UpdateTime,
	}
}

func firstNonEmptyStoreValue(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}
