package sheinloginmanaged

import (
	"context"
	"errors"
	"strconv"
	"strings"

	"task-processor/internal/listingadmin"
	"task-processor/internal/sheinlogin"
)

func NewStoreSyncClientFactoryWithStoreAPI(storeAPI listingadmin.StoreAPI) func(tenantID int64) sheinlogin.StoreSyncClient {
	return func(tenantID int64) sheinlogin.StoreSyncClient {
		return storeAPI
	}
}

func NewDuplicateStoreLookupWithStoreAPI(storeAPI listingadmin.StoreAPI) func(ctx context.Context, account sheinlogin.Account, actualStoreID string) (*listingadmin.StoreRespDTO, error) {
	return func(ctx context.Context, account sheinlogin.Account, actualStoreID string) (*listingadmin.StoreRespDTO, error) {
		return findDuplicateStore(ctx, storeAPI, account, actualStoreID)
	}
}

func findDuplicateStore(ctx context.Context, storeClient listingadmin.StoreAPI, account sheinlogin.Account, actualStoreID string) (*listingadmin.StoreRespDTO, error) {
	actualStoreID = strings.TrimSpace(actualStoreID)
	parsedStoreID, err := strconv.ParseInt(actualStoreID, 10, 64)
	if account.TenantID <= 0 || account.StoreID <= 0 || err != nil || parsedStoreID <= 0 {
		return nil, errors.New("invalid SHEIN duplicate lookup scope or store id")
	}
	if storeClient == nil {
		return nil, errors.New("SHEIN duplicate lookup unavailable")
	}

	pageNo := 1
	pageSize := 200
	for {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		page, err := storeClient.PageStores(&listingadmin.StorePageReqDTO{
			TenantID: account.TenantID,
			Platform: "SHEIN",
			PageNo:   pageNo,
			PageSize: pageSize,
		})
		if err != nil {
			// StoreAPI errors may contain rows or query parameters outside our scope.
			return nil, errors.New("SHEIN duplicate lookup query failed")
		}
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		offset := int64(pageNo-1) * int64(pageSize)
		if page == nil || page.PageNo != pageNo || page.PageSize != pageSize || page.Total < offset {
			return nil, errors.New("SHEIN duplicate lookup returned an invalid page")
		}
		expectedRows := min(int64(pageSize), page.Total-offset)
		if int64(len(page.List)) != expectedRows {
			return nil, errors.New("SHEIN duplicate lookup returned an incomplete page")
		}
		// Validate the whole returned page before consuming any candidate fields.
		for _, item := range page.List {
			if item == nil || item.TenantID != account.TenantID || item.ID <= 0 {
				return nil, errors.New("SHEIN duplicate lookup returned an invalid row scope")
			}
		}
		for _, item := range page.List {
			if item.ID == account.StoreID {
				continue
			}
			if item.Status != 0 {
				continue
			}
			if strings.EqualFold(strings.TrimSpace(item.Platform), "shein") && strings.TrimSpace(item.StoreID) == actualStoreID {
				return item, nil
			}
		}
		if offset+int64(len(page.List)) >= page.Total {
			break
		}
		pageNo++
	}
	return nil, nil
}
